package bridge

import (
    "bytes"
    "errors"
    "fmt"
    "io"
    "net"
    "runtime"
    "sync"
    "testing"
    "time"

    "tervon-tcp-bridge/padding"
)

// TestM2ConcurrentPaddingSessions exercises HTTP/2 multiplexing through the
// normal authenticated bridge: independent per-direction policies, shared
// telemetry and simultaneous byte-exact streams. No host services are used.
func TestM2ConcurrentPaddingSessions(t *testing.T) {
    address,stop:=echo(t)
    defer stop()

    policy:=padding.DefaultPolicy()
    policy.MinDuration=time.Nanosecond
    policy.MinSamples=2
    policy.MinUsefulBytes=1024
    cl,serverMetrics,closePair:=paddingTestPair(t,true,address,policy)
    defer closePair()

    const sessions=12
    const exchanges=12
    const payloadSize=8*1024

    beforeGoroutines:=runtime.NumGoroutine()
    var beforeMemory,afterMemory runtime.MemStats
    runtime.ReadMemStats(&beforeMemory)

    var wg sync.WaitGroup
    errs:=make(chan error,sessions)
    for id:=0;id<sessions;id++{
        wg.Add(1)
        go func(id int){
            defer wg.Done()
            local,peer:=net.Pipe()
            done:=make(chan error,1)
            go func(){done<-cl.Handle(local)}()
            defer peer.Close()
            _=peer.SetDeadline(time.Now().Add(15*time.Second))
            payload:=bytes.Repeat([]byte{byte(id),0,255,byte(id ^ 0x5a)},payloadSize/4)
            response:=make([]byte,len(payload))
            for n:=0;n<exchanges;n++{
                if count,err:=peer.Write(payload);err!=nil||count!=len(payload){
                    errs<-fmt.Errorf("session %d write %d: count=%d error=%v",id,n,count,err)
                    _=peer.Close()
                    <-done
                    return
                }
                if _,err:=io.ReadFull(peer,response);err!=nil {
                    errs<-fmt.Errorf("session %d read %d: %w",id,n,err)
                    _=peer.Close()
                    <-done
                    return
                }
                if !bytes.Equal(response,payload){
                    errs<-fmt.Errorf("session %d exchange %d corrupted",id,n)
                    _=peer.Close()
                    <-done
                    return
                }
            }
            _=peer.Close()
            if err:=<-done;err!=nil&&!errors.Is(err,io.EOF){
                errs<-fmt.Errorf("session %d bridge: %w",id,err)
            }
        }(id)
    }
    wg.Wait()
    close(errs)
    for err:=range errs{t.Error(err)}
    if t.Failed(){return}

    client:=cl.Metrics.Snapshot()
    server:=serverMetrics.Snapshot()
    if client.SessionsActive!=0||server.SessionsActive!=0 {
        t.Fatalf("leaked active sessions: client=%d server=%d",client.SessionsActive,server.SessionsActive)
    }
    if client.PaddingUploadState!="OFF"||server.PaddingDownloadState!="OFF"{
        t.Fatalf("leaked PAD state: upload=%s download=%s",client.PaddingUploadState,server.PaddingDownloadState)
    }
    if client.PaddingUploadWireOverheadBytes==0||server.PaddingDownloadWireOverheadBytes==0{
        t.Fatal("concurrent sustained streams did not produce PAD in both directions")
    }
    useful:=uint64(sessions*exchanges*payloadSize)
    if client.UploadBytes!=useful||client.DownloadBytes!=useful||
        server.UploadBytes!=useful||server.DownloadBytes!=useful {
        t.Fatalf("useful bytes mismatch wanted=%d client-up=%d client-down=%d server-up=%d server-down=%d",
            useful,client.UploadBytes,client.DownloadBytes,server.UploadBytes,server.DownloadBytes)
    }
    // Aggregate counters may combine sessions with different activation
    // timing, but each encoder must remain within the default wire budget.
    if client.PaddingUploadWireOverheadBytes>useful*policy.RatioPercent/100 ||
        server.PaddingDownloadWireOverheadBytes>useful*policy.RatioPercent/100 {
        t.Fatal("concurrent PAD wire budget exceeded")
    }
    runtime.ReadMemStats(&afterMemory)
    t.Logf("CI-only concurrency: sessions=%d exchanges=%d payload=%dB useful/direction=%dB PAD-wire-upload=%dB PAD-wire-download=%dB process-total-alloc-delta=%dB process-heap-end=%dB goroutines-before=%d goroutines-after=%d",
        sessions,exchanges,payloadSize,useful,
        client.PaddingUploadWireOverheadBytes,server.PaddingDownloadWireOverheadBytes,
        afterMemory.TotalAlloc-beforeMemory.TotalAlloc,
        afterMemory.HeapAlloc,beforeGoroutines,runtime.NumGoroutine())
}
