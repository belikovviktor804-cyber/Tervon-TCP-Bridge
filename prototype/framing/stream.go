package framing

import (
    "context"
    "crypto/rand"
    "errors"
    "io"
    "math"
    "time"

    "tervon-tcp-bridge/padding"
    "tervon-tcp-bridge/jitter"
)

// Encoder writes opaque bytes as bounded DATA frames. PAD frames are emitted
// only by a negotiated, enabled encoder and never affect payload or FIN.
type Encoder struct {
    w io.Writer
    closed bool
    pad *padding.State
    now func() time.Time
    onPadding func(uint64)
    onActivity func(bool)
    reportedActive bool
    jitterState *jitter.State
    jitterContext context.Context
    onJitter func(time.Duration)
    priorWriteCongested bool
}

func NewEncoder(w io.Writer) *Encoder { return &Encoder{w:w} }

// NewPaddingEncoder is only for peers that explicitly negotiated pad1.
// Its policy is per direction; each encoder owns its independent state.
func NewPaddingEncoder(w io.Writer, policy padding.Policy) (*Encoder,error) {
    state,err:=padding.New(policy)
    if err!=nil{return nil,err}
    return &Encoder{w:w,pad:state,now:time.Now},nil
}

// SetObservers installs bounded, synchronous aggregate callbacks. The callbacks
// must not block or retain payloads. They report only emitted PAD bytes and
// ACTIVE/OFF transitions, and are invoked by the encoder's single writer.
// EnableJitter is an explicit local opt-in to the bounded scheduler. The
// Bridge must negotiate jit1 with its peer before calling this method.
// It is not enabled by existing M1/M2 constructors or configurations.
func (e *Encoder) EnableJitter(ctx context.Context, policy jitter.Policy, onDelay func(time.Duration)) error {
    if e.closed || ctx == nil { return jitter.ErrPolicy }
    state, err := jitter.New(policy)
    if err != nil { return err }
    e.jitterState = state
    e.jitterContext = ctx
    e.onJitter = onDelay
    if e.now == nil { e.now = time.Now }
    return nil
}

func (e *Encoder) SetObservers(onPadding func(uint64), onActivity func(bool)) {
    e.onPadding=onPadding
    e.onActivity=onActivity
}
func (e *Encoder) updateActive(v bool) {
    if e.reportedActive==v{return}
    e.reportedActive=v
    if e.onActivity!=nil{e.onActivity(v)}
}
func (e *Encoder) fail() {
    e.closed=true
    e.updateActive(false)
}

func (e *Encoder) Write(p []byte) (int,error) {
    if e.closed {return 0,io.ErrClosedPipe}
    total:=0
    for len(p)>0 {
        n:=len(p)
        if n>MaxPayload {n=MaxPayload}
        // Jitter is always before DATA and bounded by the opt-in
        // scheduler. A previously blocked downstream writer disables
        // the next delay to avoid adding latency under backpressure.
        if e.jitterState != nil {
            d, err := e.jitterState.Observe(e.now(), uint64(n), e.priorWriteCongested)
            if err != nil { e.fail(); return total, err }
            if err := jitter.Wait(e.jitterContext, d); err != nil {
                e.fail()
                return total, err
            }
            if d > 0 && e.onJitter != nil { e.onJitter(d) }
        }
        var writeStarted time.Time
        if e.jitterState != nil { writeStarted = time.Now() }
        if err:=WriteFrame(e.w,TypeData,p[:n]);err!=nil{
            e.fail()
            return total,err
        }
        if e.jitterState != nil {
            e.priorWriteCongested = time.Since(writeStarted) > 10*time.Millisecond
        }
        total+=n
        if e.pad!=nil{
            var count uint64
            if !e.pad.ObserveSmallFrame(uint64(n)) {
                count = e.pad.Observe(e.now(),uint64(n))
            }
            e.updateActive(e.pad.Active())
            if count>0{
                // Only emit padding after DATA, never by delaying a DATA frame.
                pad:=make([]byte,int(count))
                if _,err:=rand.Read(pad);err!=nil{e.fail();return total,err}
                if err:=WritePaddingFrame(e.w,pad);err!=nil{e.fail();return total,err}
                if e.onPadding!=nil{e.onPadding(count)}
            }
        }
        p=p[n:]
    }
    return total,nil
}
// Abort marks the encoder terminal without writing FIN. Use after a failed
// stream to release activity accounting; never forge successful half-close.
func (e *Encoder) Abort() {
    e.fail()
}

func (e *Encoder) CloseWrite() error {
    if e.closed{return io.ErrClosedPipe}
    e.closed=true
    e.updateActive(false)
    return WriteFrame(e.w,TypeFIN,nil)
}
func (e *Encoder) PaddingBytes() uint64 {
    if e.pad==nil {return 0}
    return e.pad.PaddingBytes()
}

// Decoder accepts only DATA and FIN unless pad1 was explicitly negotiated.
// PAD frames are discarded before bytes reach the caller's backend.
type Decoder struct {
    r io.Reader
    pending []byte
    finished bool
    allowPadding bool
    useful uint64
    padded uint64
    lastWasData bool
}
func NewDecoder(r io.Reader)*Decoder{return &Decoder{r:r}}
func NewPaddingDecoder(r io.Reader)*Decoder{return &Decoder{r:r,allowPadding:true}}
func (d *Decoder) Read(p []byte)(int,error){
    if len(p)==0{return 0,nil}
    if len(d.pending)>0{
        n:=copy(p,d.pending)
        d.pending=d.pending[n:]
        return n,nil
    }
    if d.finished{return 0,io.EOF}
    for {
        var f Frame
        var err error
        if d.allowPadding {f,err=ReadFrameWithPadding(d.r)} else {f,err=ReadFrame(d.r)}
        if err!=nil{
            if errors.Is(err,io.EOF){return 0,ErrMalformed}
            return 0,err
        }
        switch f.Type {
        case TypePadding:
            // Permit at most one PAD after DATA and a bounded cumulative
            // budget, even for a buggy or malicious authenticated peer.
            if !d.lastWasData || len(f.Payload)>4096 {return 0,ErrMalformed}
            // Include the PAD frame header in the receiving-side budget.
            // This prevents many tiny PAD frames from bypassing the limit.
            wireBytes:=uint64(len(f.Payload))+uint64(HeaderSize)
            if d.padded>math.MaxUint64-wireBytes{return 0,ErrMalformed}
            d.padded+=wireBytes
            // The receiver allows at most 25% of useful bytes as PAD wire
            // overhead; the default encoder imposes a stricter 5% budget.
            if d.padded>d.useful/4{return 0,ErrMalformed}
            d.lastWasData=false
            continue
        case TypeFIN:
            var extra [1]byte
            n,err:=d.r.Read(extra[:])
            if n!=0 || err==nil{return 0,ErrMalformed}
            if !errors.Is(err,io.EOF){return 0,err}
            d.finished=true
            return 0,io.EOF
        default:
            if d.useful>math.MaxUint64-uint64(len(f.Payload)) {
                d.useful=math.MaxUint64
            }else{
                d.useful+=uint64(len(f.Payload))
            }
            d.lastWasData=true
            d.pending=f.Payload
            n:=copy(p,d.pending)
            d.pending=d.pending[n:]
            return n,nil
        }
    }
}
