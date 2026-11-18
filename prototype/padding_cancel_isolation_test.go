package bridge

import (
    "bytes"
    "context"
    "errors"
    "io"
    "net"
    "testing"
    "time"

    "tervon-tcp-bridge/padding"
)

// TestM2CancelOneHTTP2StreamPreservesOtherSessions verifies that an aborted
// PAD-enabled HTTP/2 stream cannot poison the shared transport or leave an
// ACTIVE padding state after the handler exits. Uses only in-process CI
// fixtures, not any host or production listener.
func TestM2CancelOneHTTP2StreamPreservesOtherSessions(t *testing.T) {
    addr,stop:=echo(t)
    defer stop()
    policy:=padding.DefaultPolicy()
    policy.MinDuration=time.Nanosecond
    policy.MinSamples=2
    policy.MinUsefulBytes=1024
    cl,serverStats,closePair:=paddingTestPair(t,true,addr,policy)
    defer closePair()

    ctx,cancel:=context.WithCancel(context.Background())
    local,peer:=net.Pipe()
    canceledDone:=make(chan error,1)
    go func(){canceledDone<-cl.HandleContext(ctx,local)}()
    defer peer.Close()
    _=peer.SetDeadline(time.Now().Add(10*time.Second))

    payload:=bytes.Repeat([]byte{1,255,2,3},2048)
    reply:=make([]byte,len(payload))
    for i:=0;i<5;i++{
        if _,err:=peer.Write(payload);err!=nil {t.Fatalf("before cancellation write: %v",err)}
        if _,err:=io.ReadFull(peer,reply);err!=nil {t.Fatalf("before cancellation read: %v",err)}
        if !bytes.Equal(reply,payload){t.Fatal("pre-cancel payload mismatch")}
    }
    if cl.Metrics.Snapshot().PaddingUploadWireOverheadBytes==0 {
        t.Fatal("canceled stream never generated PAD")
    }

    // A second stream on the same HTTP/2 client should remain healthy while
    // the first is canceled, and must complete its independent FIN path.
    healthyLocal,healthyPeer:=net.Pipe()
    healthyDone:=make(chan error,1)
    go func(){healthyDone<-cl.Handle(healthyLocal)}()
    defer healthyPeer.Close()
    _=healthyPeer.SetDeadline(time.Now().Add(10*time.Second))

    cancel()
    _=peer.Close()
    select {
    case err:=<-canceledDone:
        // Cancellation is expected, not a successful FIN.
        if err==nil {t.Fatal("canceled stream returned clean success")}
    case <-time.After(5*time.Second):
        t.Fatal("canceled HTTP/2 upload did not exit")
    }

    for i:=0;i<6;i++ {
        if _,err:=healthyPeer.Write(payload);err!=nil{t.Fatalf("healthy write: %v",err)}
        if _,err:=io.ReadFull(healthyPeer,reply);err!=nil{t.Fatalf("healthy read: %v",err)}
        if !bytes.Equal(reply,payload){t.Fatal("healthy stream corrupted after sibling canceled")}
    }
    _=healthyPeer.Close()
    select {
    case err:=<-healthyDone:
        if err!=nil && !errors.Is(err,io.EOF){t.Fatalf("healthy stream failed: %v",err)}
    case <-time.After(5*time.Second):
        t.Fatal("healthy HTTP/2 stream stalled after sibling canceled")
    }

    deadline:=time.Now().Add(3*time.Second)
    for {
        localSnap:=cl.Metrics.Snapshot()
        remoteSnap:=serverStats.Snapshot()
        if localSnap.SessionsActive==0 && remoteSnap.SessionsActive==0 &&
            localSnap.PaddingUploadState=="OFF" && remoteSnap.PaddingDownloadState=="OFF" {
            t.Logf("M2 cancel isolation PASS: successful PAD wire upload=%d download=%d; session active client/server=0; PAD states OFF",localSnap.PaddingUploadWireOverheadBytes,remoteSnap.PaddingDownloadWireOverheadBytes)
            return
        }
        if time.Now().After(deadline){
            t.Fatalf("after cancel+healthy close active client=%d server=%d pad-up=%s pad-down=%s",
                localSnap.SessionsActive,remoteSnap.SessionsActive,
                localSnap.PaddingUploadState,remoteSnap.PaddingDownloadState)
        }
        time.Sleep(10*time.Millisecond)
    }
}
