package framing

import (
    "crypto/rand"
    "errors"
    "io"
    "time"

    "tervon-tcp-bridge/padding"
)

// Encoder writes opaque bytes as bounded DATA frames. PAD frames are emitted
// only by a negotiated, enabled encoder and never affect payload or FIN.
type Encoder struct {
    w io.Writer
    closed bool
    pad *padding.State
    now func() time.Time
}

func NewEncoder(w io.Writer) *Encoder { return &Encoder{w:w} }

// NewPaddingEncoder is only for peers that explicitly negotiated pad1.
// Its policy is per direction; each encoder owns its independent state.
func NewPaddingEncoder(w io.Writer, policy padding.Policy) (*Encoder,error) {
    state,err:=padding.New(policy)
    if err!=nil{return nil,err}
    return &Encoder{w:w,pad:state,now:time.Now},nil
}

func (e *Encoder) Write(p []byte) (int,error) {
    if e.closed {return 0,io.ErrClosedPipe}
    total:=0
    for len(p)>0 {
        n:=len(p)
        if n>MaxPayload {n=MaxPayload}
        if err:=WriteFrame(e.w,TypeData,p[:n]);err!=nil{
            e.closed=true
            return total,err
        }
        total+=n
        if e.pad!=nil{
            count:=e.pad.Observe(e.now(),uint64(n))
            if count>0{
                // Only emit padding after DATA, never by delaying a DATA frame.
                pad:=make([]byte,int(count))
                if _,err:=rand.Read(pad);err!=nil{e.closed=true;return total,err}
                if err:=WritePaddingFrame(e.w,pad);err!=nil{e.closed=true;return total,err}
            }
        }
        p=p[n:]
    }
    return total,nil
}
func (e *Encoder) CloseWrite() error {
    if e.closed{return io.ErrClosedPipe}
    e.closed=true
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
            // A PAD is valid only on a negotiated stream. Never forward it.
            continue
        case TypeFIN:
            var extra [1]byte
            n,err:=d.r.Read(extra[:])
            if n!=0 || err==nil{return 0,ErrMalformed}
            if !errors.Is(err,io.EOF){return 0,err}
            d.finished=true
            return 0,io.EOF
        default:
            d.pending=f.Payload
            n:=copy(p,d.pending)
            d.pending=d.pending[n:]
            return n,nil
        }
    }
}
