package framing

import (
    "io"
    "testing"
    "time"

    "tervon-tcp-bridge/padding"
)

// BenchmarkShortFlowEncoding compares connection-local CPU and allocations
// for a small interactive payload. An M2-enabled short stream must emit no PAD.
// The result is only codec-level overhead, never an end-to-end latency claim.
func BenchmarkShortFlowEncoding(b *testing.B) {
    payload := make([]byte, 128)
    b.Run("M1_legacy",func(b *testing.B){
        b.ReportAllocs()
        for i:=0;i<b.N;i++{
            enc:=NewEncoder(io.Discard)
            _,_ =enc.Write(payload)
            _=enc.CloseWrite()
        }
    })
    b.Run("M2_enabled_short",func(b *testing.B){
        p:=padding.DefaultPolicy()
        b.ReportAllocs()
        for i:=0;i<b.N;i++{
            enc,err:=NewPaddingEncoder(io.Discard,p)
            if err!=nil{b.Fatal(err)}
            _,_ =enc.Write(payload)
            _=enc.CloseWrite()
            if enc.PaddingBytes()!=0{b.Fatal("short stream emitted PAD")}
        }
    })
}

// BenchmarkSustainedFlowEncoding compares the TBF1 encoder's cost for
// sustained 16 KiB DATA records, including actual PAD generation. Reported
// B/op excludes network buffers and HTTP/2/TLS; it is not a wire benchmark.
func BenchmarkSustainedFlowEncoding(b *testing.B) {
    payload:=make([]byte,16*1024)
    b.Run("M1_legacy",func(b *testing.B){
        enc:=NewEncoder(io.Discard)
        b.SetBytes(int64(len(payload)))
        b.ReportAllocs()
        b.ResetTimer()
        for i:=0;i<b.N;i++{_,_=enc.Write(payload)}
    })
    b.Run("M2_enabled_PAD",func(b *testing.B){
        p:=padding.DefaultPolicy()
        p.MinDuration=time.Nanosecond
        p.MinSamples=2
        p.MinUsefulBytes=1024
        enc,err:=NewPaddingEncoder(io.Discard,p)
        if err!=nil{b.Fatal(err)}
        clock:=time.Unix(1000,0)
        enc.now=func()time.Time{
            clock=clock.Add(time.Millisecond)
            return clock
        }
        b.SetBytes(int64(len(payload)))
        b.ReportAllocs()
        b.ResetTimer()
        for i:=0;i<b.N;i++{_,_=enc.Write(payload)}
        b.StopTimer()
        if b.N>2 && enc.PaddingBytes()==0{b.Fatal("benchmark never emitted PAD")}
    })
}
