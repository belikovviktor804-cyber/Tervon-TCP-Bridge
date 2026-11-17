package framing

import (
    "bytes"
    "errors"
    "io"
    "testing"
    "time"

    "tervon-tcp-bridge/padding"
)

func TestPaddingIsOptInAndPayloadIsByteExact(t *testing.T) {
    p:=padding.DefaultPolicy()
    p.MinDuration=time.Nanosecond
    p.MinUsefulBytes=1024
    p.MinSamples=2
    p.MaxInterSampleGap=time.Second
    var wire bytes.Buffer
    enc,err:=NewPaddingEncoder(&wire,p)
    if err!=nil{t.Fatal(err)}
    t0:=time.Unix(123,0)
    tick:=0
    enc.now=func()time.Time{tick++;return t0.Add(time.Duration(tick)*time.Millisecond)}
    payload:=bytes.Repeat([]byte{0,255,1,3},4096)
    for i:=0;i<10;i++ {
        if n,err:=enc.Write(payload);err!=nil || n!=len(payload) {t.Fatalf("write: %d %v",n,err)}
    }
    if err:=enc.CloseWrite();err!=nil{t.Fatal(err)}
    if enc.PaddingBytes()==0{t.Fatal("stable flow did not generate padding")}
    var got bytes.Buffer
    if _,err:=io.Copy(&got,NewPaddingDecoder(bytes.NewReader(wire.Bytes())));err!=nil{t.Fatal(err)}
    want:=bytes.Repeat(payload,10)
    if !bytes.Equal(got.Bytes(),want){t.Fatal("padding corrupted TCP payload")}
    if _,err:=io.ReadAll(NewDecoder(bytes.NewReader(wire.Bytes())));!errors.Is(err,ErrType){
        t.Fatalf("legacy decoder must reject padding frame: %v",err)
    }
    raw:=bytes.NewReader(wire.Bytes())
    padFrames:=0
    for raw.Len()>0 {
        f,err:=ReadFrameWithPadding(raw)
        if err!=nil{t.Fatal(err)}
        if f.Type==TypePadding {padFrames++}
    }
    if padFrames==0 {t.Fatal("wire contains no padding frames")}
}

func TestPaddingFrameValidation(t *testing.T){
    if err:=WritePaddingFrame(io.Discard,nil);!errors.Is(err,ErrMalformed){t.Fatal(err)}
    if err:=WritePaddingFrame(io.Discard,make([]byte,MaxPayload+1));!errors.Is(err,ErrSize){t.Fatal(err)}
    var b bytes.Buffer
    if err:=WritePaddingFrame(&b,[]byte{1,2,3});err!=nil{t.Fatal(err)}
    if _,err:=ReadFrame(bytes.NewReader(b.Bytes()));!errors.Is(err,ErrType){t.Fatalf("legacy accepted pad %v",err)}
    raw:=b.Bytes()
    if _,err:=ReadFrameWithPadding(bytes.NewReader(raw[:len(raw)-1]));!errors.Is(err,ErrMalformed){t.Fatalf("truncated pad: %v",err)}
    if _,err:=io.ReadAll(NewPaddingDecoder(bytes.NewReader(raw)));!errors.Is(err,ErrMalformed){t.Fatalf("missing FIN: %v",err)}
}

func TestPaddingShortStreamRemainsIdenticalToM1(t *testing.T){
    var standard,padded bytes.Buffer
    p:=padding.DefaultPolicy()
    enc,err:=NewPaddingEncoder(&padded,p);if err!=nil{t.Fatal(err)}
    baseline:=NewEncoder(&standard)
    content:=[]byte("interactive")
    if _,err:=enc.Write(content);err!=nil{t.Fatal(err)}
    if _,err:=baseline.Write(content);err!=nil{t.Fatal(err)}
    if err:=enc.CloseWrite();err!=nil{t.Fatal(err)}
    if err:=baseline.CloseWrite();err!=nil{t.Fatal(err)}
    if !bytes.Equal(padded.Bytes(),standard.Bytes()){t.Fatal("short stream wire changed")}
}

func FuzzPaddingDecoder(f *testing.F){
    var seed bytes.Buffer
    _=WriteFrame(&seed,TypeData,[]byte("abc"))
    _=WritePaddingFrame(&seed,[]byte{4,5,6})
    _=WriteFrame(&seed,TypeFIN,nil)
    f.Add(seed.Bytes())
    f.Add([]byte("TBF1"))
    f.Fuzz(func(t *testing.T,data []byte){
        if len(data)>3*MaxPayload+4*HeaderSize{return}
        // Parsing adversarial data must never panic or expose padding.
        _,_ =io.ReadAll(NewPaddingDecoder(bytes.NewReader(data)))
    })
}

func TestPaddingDecoderRejectsExcessAndPlacement(t *testing.T){
    cases:=map[string]func(*bytes.Buffer){
        "pad-before-data":func(b *bytes.Buffer){_ = WritePaddingFrame(b,[]byte{1})},
        "double-pad":func(b *bytes.Buffer){
            _=WriteFrame(b,TypeData,bytes.Repeat([]byte{1},1024))
            _=WritePaddingFrame(b,[]byte{1})
            _=WritePaddingFrame(b,[]byte{2})
        },
        "frame-cap":func(b *bytes.Buffer){
            _=WriteFrame(b,TypeData,bytes.Repeat([]byte{1},16384))
            _=WritePaddingFrame(b,make([]byte,4097))
        },
        "ratio-cap":func(b *bytes.Buffer){
            _=WriteFrame(b,TypeData,bytes.Repeat([]byte{1},100))
            _=WritePaddingFrame(b,make([]byte,26))
        },
    }
    for name,build:=range cases {
        t.Run(name,func(t *testing.T){
            var b bytes.Buffer
            build(&b)
            _=WriteFrame(&b,TypeFIN,nil)
            _,err:=io.ReadAll(NewPaddingDecoder(&b))
            if !errors.Is(err,ErrMalformed){t.Fatalf("invalid padding accepted: %v",err)}
        })
    }
}

func TestPaddingEncoderCallbacksAndFinalOff(t *testing.T){
    p:=padding.DefaultPolicy()
    p.MinDuration=time.Nanosecond
    p.MinSamples=2
    p.MinUsefulBytes=1024
    var wire bytes.Buffer
    enc,err:=NewPaddingEncoder(&wire,p);if err!=nil{t.Fatal(err)}
    var emitted uint64
    changes:=[]bool{}
    enc.SetObservers(func(n uint64){emitted+=n},func(active bool){changes=append(changes,active)})
    t0:=time.Unix(123,0)
    i:=0
    enc.now=func()time.Time{i++;return t0.Add(time.Duration(i)*time.Millisecond)}
    for j:=0;j<10;j++{
        if _,err:=enc.Write(bytes.Repeat([]byte{0,1},2048));err!=nil{t.Fatal(err)}
    }
    if err:=enc.CloseWrite();err!=nil{t.Fatal(err)}
    if emitted==0 || emitted!=enc.PaddingBytes(){t.Fatalf("accounting emitted=%d state=%d",emitted,enc.PaddingBytes())}
    if len(changes)!=2 || !changes[0] || changes[1]{t.Fatalf("active transition leak: %v",changes)}
}

func TestPaddingAbortClearsActivityWithoutFIN(t *testing.T){
    p:=padding.DefaultPolicy()
    p.MinDuration=time.Nanosecond
    p.MinSamples=2
    p.MinUsefulBytes=1024
    var b bytes.Buffer
    e,err:=NewPaddingEncoder(&b,p);if err!=nil{t.Fatal(err)}
    events:=[]bool{}
    e.SetObservers(nil,func(active bool){events=append(events,active)})
    clock:=time.Unix(1234,0)
    e.now=func()time.Time{clock=clock.Add(time.Millisecond);return clock}
    for i:=0;i<5;i++{if _,err:=e.Write(bytes.Repeat([]byte{2},1024));err!=nil{t.Fatal(err)}}
    e.Abort()
    e.Abort()
    if len(events)!=2 || !events[0] || events[1]{t.Fatalf("invalid transitions %v",events)}
    if _,err:=e.Write([]byte("post-abort"));!errors.Is(err,io.ErrClosedPipe){t.Fatalf("accepted write after abort: %v",err)}
    if err:=e.CloseWrite();!errors.Is(err,io.ErrClosedPipe){t.Fatalf("forged FIN: %v",err)}
    if _,err:=io.ReadAll(NewPaddingDecoder(bytes.NewReader(b.Bytes())));!errors.Is(err,ErrMalformed){t.Fatalf("abort produced clean FIN: %v",err)}
}
