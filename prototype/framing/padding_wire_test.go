package framing

import (
    "bytes"
    "io"
    "testing"
    "time"

    "tervon-tcp-bridge/padding"
)

// TestM2WireAccounting checks the actual serialized TBF1 frames. The accounting
// is independent from the encoder's counters and includes every frame header.
// It intentionally excludes HTTP/2 and TLS, measured in a later M2 gate.
func TestM2WireAccounting(t *testing.T) {
    cases:=[]struct{
        name string
        sizes []int
        gaps []time.Duration
        padExpected bool
    }{
        {
            name:"interactive",
            sizes:[]int{32,64,128,200,400},
            gaps:[]time.Duration{0,time.Millisecond,2*time.Millisecond,3*time.Millisecond,4*time.Millisecond},
        },
        {
            name:"sustained",
            sizes:[]int{16384,16384,16384,16384,16384,16384,16384,16384},
            gaps:[]time.Duration{0,500*time.Millisecond,time.Second,1500*time.Millisecond,2*time.Second,2500*time.Millisecond,3*time.Second,3500*time.Millisecond},
            padExpected:true,
        },
        {
            name:"burst-then-short",
            sizes:[]int{16384,16384,16384,16384,16384,128,128,128},
            gaps:[]time.Duration{0,500*time.Millisecond,time.Second,1500*time.Millisecond,2*time.Second,10*time.Second,10001*time.Millisecond,10002*time.Millisecond},
            padExpected:true,
        },
    }
    for _,tc:=range cases {
        t.Run(tc.name,func(t *testing.T){
            policy:=padding.DefaultPolicy()
            var wire bytes.Buffer
            enc,err:=NewPaddingEncoder(&wire,policy)
            if err!=nil{t.Fatal(err)}
            start:=time.Unix(1234,0)
            var clock time.Time
            enc.now=func()time.Time{return clock}
            var expected bytes.Buffer
            for i,size:=range tc.sizes{
                clock=start.Add(tc.gaps[i])
                payload:=bytes.Repeat([]byte{0,255,1,3},(size+3)/4)[:size]
                if n,err:=enc.Write(payload);err!=nil || n!=size{
                    t.Fatalf("write size=%d n=%d err=%v",size,n,err)
                }
                expected.Write(payload)
            }
            if err:=enc.CloseWrite();err!=nil{t.Fatal(err)}
            input:=bytes.NewReader(wire.Bytes())
            var useful, padPayload, dataFrames, padFrames, finFrames int
            for input.Len()>0 {
                frame,err:=ReadFrameWithPadding(input)
                if err!=nil{t.Fatal(err)}
                switch frame.Type {
                case TypeData:
                    useful+=len(frame.Payload)
                    dataFrames++
                case TypePadding:
                    padPayload+=len(frame.Payload)
                    padFrames++
                case TypeFIN:
                    finFrames++
                default:
                    t.Fatalf("unexpected frame type %d",frame.Type)
                }
            }
            if useful!=expected.Len() || dataFrames!=len(tc.sizes) || finFrames!=1 {
                t.Fatalf("unexpected DATA/FIN useful=%d data=%d fin=%d",useful,dataFrames,finFrames)
            }
            if (padFrames>0)!=tc.padExpected{
                t.Fatalf("PAD mismatch frames=%d want=%v",padFrames,tc.padExpected)
            }
            padWire:=padPayload+padFrames*HeaderSize
            expectedWire:=useful+(dataFrames+finFrames)*HeaderSize+padWire
            if wire.Len()!=expectedWire{
                t.Fatalf("wire size mismatch got=%d expected=%d",wire.Len(),expectedWire)
            }
            if uint64(padPayload)!=enc.PaddingBytes(){
                t.Fatalf("PAD payload accounting %d vs encoder %d",padPayload,enc.PaddingBytes())
            }
            if padWire*100>useful*int(policy.RatioPercent){
                t.Fatalf("PAD wire overhead exceeded budget: %d/%d",padWire,useful)
            }
            got,err:=io.ReadAll(NewPaddingDecoder(bytes.NewReader(wire.Bytes())))
            if err!=nil || !bytes.Equal(got,expected.Bytes()){
                t.Fatalf("decoded stream changed: bytes=%d err=%v",len(got),err)
            }
            if !tc.padExpected {
                var baseline bytes.Buffer
                legacy:=NewEncoder(&baseline)
                for _,size:=range tc.sizes {
                    payload:=bytes.Repeat([]byte{0,255,1,3},(size+3)/4)[:size]
                    if _,err:=legacy.Write(payload);err!=nil{t.Fatal(err)}
                }
                if err:=legacy.CloseWrite();err!=nil{t.Fatal(err)}
                if !bytes.Equal(baseline.Bytes(),wire.Bytes()){
                    t.Fatal("short traffic changed TBF1 wire data despite no PAD")
                }
            }
        })
    }
}
