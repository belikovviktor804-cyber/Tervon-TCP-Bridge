package bridge

import (
    "bytes"
    "errors"
    "io"
    "net"
    "runtime"
    "sort"
    "testing"
    "time"

    "tervon-tcp-bridge/padding"
)

type m2LatencySample struct {
    mode string
    payloadSize int
    p50 time.Duration
    p95 time.Duration
    p99 time.Duration
    totalAllocated uint64
    maxHeapAlloc uint64
    padUpload uint64
    padDownload uint64
}

// percentileDuration is a nearest-rank quantile for finite, sorted durations.
// Used for test diagnostics, not production aggregation.
func percentileDuration(samples []time.Duration, percentile int) time.Duration {
    if len(samples)==0{return 0}
    copySamples:=append([]time.Duration(nil), samples...)
    sort.Slice(copySamples,func(i,j int)bool{return copySamples[i]<copySamples[j]})
    rank:=(len(copySamples)*percentile+99)/100
    if rank<1{rank=1}
    if rank>len(copySamples){rank=len(copySamples)}
    return copySamples[rank-1]
}

func measureM2LatencyAndAllocation(t *testing.T, enabled bool, payloadSize int) m2LatencySample {
    t.Helper()
    backend,stop:=echo(t)
    defer stop()
    p:=padding.DefaultPolicy()
    // The detector is accelerated only inside this diagnostic test to
    // exercise active PAD without introducing a two-second artificial wait.
    p.MinDuration=time.Nanosecond
    p.MinSamples=2
    p.MinUsefulBytes=1024
    cl,serverMetrics,closePair:=paddingTestPair(t,true,backend,p)
    defer closePair()
    if !enabled{cl.PaddingPolicy=nil}
    mode:="M1"
    if enabled{mode="M2"}
    local,peer:=net.Pipe()
    finished:=make(chan error,1)
    go func(){finished<-cl.Handle(local)}()
    defer peer.Close()
    if err:=peer.SetDeadline(time.Now().Add(10*time.Second));err!=nil{t.Fatal(err)}
    payload:=bytes.Repeat([]byte{0,1,254,255},(payloadSize+3)/4)[:payloadSize]
    reply:=make([]byte,len(payload))
    const count=48
    elapsed:=make([]time.Duration,0,count)
    var before,after runtime.MemStats
    runtime.ReadMemStats(&before)
    for i:=0;i<count;i++{
        started:=time.Now()
        if _,err:=peer.Write(payload);err!=nil{t.Fatalf("%s write[%d]: %v",mode,i,err)}
        if _,err:=io.ReadFull(peer,reply);err!=nil{t.Fatalf("%s read[%d]: %v",mode,i,err)}
        elapsed=append(elapsed,time.Since(started))
        if !bytes.Equal(reply,payload){t.Fatalf("%s altered response[%d]",mode,i)}
    }
    runtime.ReadMemStats(&after)
    _=peer.Close()
    select {
    case err:=<-finished:
        if err!=nil&&!errors.Is(err,io.EOF){t.Fatalf("%s session close: %v",mode,err)}
    case <-time.After(5*time.Second):
        t.Fatalf("%s handler leaked",mode)
    }
    snapshot:=cl.Metrics.Snapshot()
    remote:=serverMetrics.Snapshot()
    if payloadSize<512 && snapshot.PaddingUploadBytes!=0{
        t.Fatal("interactive upload unexpectedly generated PAD")
    }
    if enabled && payloadSize>=512 && snapshot.PaddingUploadBytes==0{
        t.Fatal("sustained M2 upload did not produce PAD")
    }
    // HeapAlloc is an instantaneous process-wide gauge. TotalAlloc is
    // process-wide, cumulative and not exclusively attributable to Bridge.
    allocDelta:=uint64(0)
    if after.TotalAlloc>=before.TotalAlloc{allocDelta=after.TotalAlloc-before.TotalAlloc}
    result:=m2LatencySample{
        mode:mode,payloadSize:payloadSize,
        p50:percentileDuration(elapsed,50),
        p95:percentileDuration(elapsed,95),
        p99:percentileDuration(elapsed,99),
        totalAllocated:allocDelta,
        maxHeapAlloc:after.HeapAlloc,
        padUpload:snapshot.PaddingUploadWireOverheadBytes,
        padDownload:remote.PaddingDownloadWireOverheadBytes,
    }
    t.Logf("CI-only mTLS H2 latency: mode=%s payload=%dB exchanges=%d p50=%s p95=%s p99=%s process-total-alloc-delta=%dB heap-alloc-end=%dB pad-wire-up=%dB pad-wire-down=%dB",
        result.mode,result.payloadSize,count,result.p50,result.p95,result.p99,
        result.totalAllocated,result.maxHeapAlloc,result.padUpload,result.padDownload)
    return result
}

func TestM2HTTP2LatencyAllocationDiagnostics(t *testing.T) {
    for _,size:=range []int{128,16*1024}{
        t.Run("M1/"+testPayloadName(size),func(t *testing.T){
            measureM2LatencyAndAllocation(t,false,size)
        })
        t.Run("M2/"+testPayloadName(size),func(t *testing.T){
            measureM2LatencyAndAllocation(t,true,size)
        })
    }
}

func testPayloadName(size int)string{
    if size==128{return "interactive_128B"}
    return "sustained_16KiB"
}

func TestM2NearestRankQuantiles(t *testing.T){
    samples:=[]time.Duration{4,3,1,2,5}
    if percentileDuration(samples,50)!=3||
        percentileDuration(samples,95)!=5||
        percentileDuration(samples,99)!=5||
        percentileDuration(nil,50)!=0 {
        t.Fatal("nearest-rank percentile calculation changed")
    }
}
