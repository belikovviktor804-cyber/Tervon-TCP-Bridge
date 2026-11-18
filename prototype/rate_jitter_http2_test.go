package bridge

import (
    "bytes"
    "crypto/sha256"
    "crypto/tls"
    "errors"
    "io"
    "net"
    "net/http/httptest"
    "testing"
    "sync"
    "time"

    "tervon-tcp-bridge/jitter"
    "tervon-tcp-bridge/padding"
    "tervon-tcp-bridge/telemetry"
)

// m3Pair uses the existing authenticated HTTP/2 and certificate helpers.
// Production services and other protocols are not involved.
func m3Pair(t *testing.T, backend string, jitterOn bool, serverRate bool) (*Client, *telemetry.Registry, func()) {
    t.Helper()
    certificate, pool := cert(t)
    server := &Server{
        Routes: map[string][]Backend{"vpn": {{Name:"test",Address:backend}}},
        AuthorizedServices: map[[32]byte]map[string]bool{
            sha256.Sum256(certificate.Certificate[0]): {"vpn":true},
        },
        Metrics: telemetry.New(),
    }
    old := padding.DefaultPolicy()
    rate := padding.DefaultRatePolicy()
    rate.Window = 100*time.Millisecond
    rate.MaxInterSampleGap=100*time.Millisecond
    j := jitter.DefaultPolicy()
    j.MinDuration=100*time.Millisecond
    j.MaxInterSampleGap=100*time.Millisecond
    j.MinUsefulBytes=1024
    j.MinSamples=2
    server.PaddingPolicy = &old
    if serverRate { server.RatePaddingPolicy=&rate }
    if jitterOn { server.JitterPolicy=&j }
    tlsServer:=httptest.NewUnstartedServer(server)
    tlsServer.EnableHTTP2=true
    tlsServer.TLS=&tls.Config{
        Certificates: []tls.Certificate{certificate},
        ClientAuth: tls.RequireAndVerifyClientCert,
        ClientCAs: pool,
        NextProtos: []string{"h2"},
    }
    tlsServer.StartTLS()
    client:=NewClient(&tls.Config{
        RootCAs:pool,
        ServerName:"example.com",
        Certificates:[]tls.Certificate{certificate},
    },tlsServer.URL+"/v2/stream","vpn")
    client.PaddingPolicy=&old
    client.RatePaddingPolicy=&rate
    if jitterOn { client.JitterPolicy=&j }
    client.Metrics=telemetry.New()
    client.Metrics.SetPaddingConfigured(true)
    return client,server.Metrics,func(){client.HTTP.CloseIdleConnections();tlsServer.Close()}
}

func TestM3RateOneMbpsHTTP2FramingAndLegacyIsolation(t *testing.T) {
    addr,stop:=echo(t)
    defer stop()
    cl,sm,cleanup:=m3Pair(t,addr,false,true)
    defer cleanup()
    local,peer:=net.Pipe()
    defer peer.Close()
    result:=make(chan error,1)
    go func(){result<-cl.Handle(local)}()
    if err:=peer.SetDeadline(time.Now().Add(15*time.Second));err!=nil{t.Fatal(err)}
    block:=bytes.Repeat([]byte{17,23,0,255},1024)
    for i:=0;i<28;i++{
        // Many independent writes, including 128-byte fragments.
        for _,payload:=range [][]byte{block,bytes.Repeat([]byte{42},128)} {
            if _,err:=peer.Write(payload);err!=nil{t.Fatal(err)}
            received:=make([]byte,len(payload))
            if _,err:=io.ReadFull(peer,received);err!=nil{t.Fatal(err)}
            if !bytes.Equal(received,payload){t.Fatal("roundtrip changed payload")}
        }
        time.Sleep(20*time.Millisecond)
    }
    _=peer.Close()
    select {
    case err:=<-result:
        if err!=nil && !errors.Is(err,io.EOF){t.Fatalf("handler error: %v",err)}
    case <-time.After(5*time.Second):
        t.Fatal("HTTP2 rate padding handler stuck")
    }
    clientMetrics:=cl.Metrics.Snapshot()
    serverMetrics:=sm.Snapshot()
    if clientMetrics.PaddingUploadFrames==0 || serverMetrics.PaddingDownloadFrames==0 ||
        clientMetrics.PaddingDownloadFrames==0 || serverMetrics.PaddingUploadFrames==0 {
        t.Fatalf("M3 rate PAD not emitted in negotiated HTTP2 session: upload=%d download=%d",
            clientMetrics.PaddingUploadFrames,serverMetrics.PaddingDownloadFrames)
    }
    if clientMetrics.PaddingUploadWireOverheadBytes > clientMetrics.UploadBytes*5/100 {
        t.Fatal("upload overhead exceeded hard limit")
    }
    if clientMetrics.PaddingUploadState!="OFF" || serverMetrics.PaddingDownloadState!="OFF" {
        t.Fatal("active PAD state leaked after stream close")
    }
}

func TestM3RateCapabilityRejectsUnsupportedServer(t *testing.T) {
    cl,_,cleanup:=m3Pair(t,"127.0.0.1:1",false,false)
    defer cleanup()
    local,peer:=net.Pipe()
    defer peer.Close()
    result:=make(chan error,1)
    go func(){result<-cl.Handle(local)}()
    select {
    case err:=<-result:
        if err==nil || !bytes.Contains([]byte(err.Error()),[]byte("426")){
            t.Fatalf("rate capability not rejected: %v",err)
        }
    case <-time.After(5*time.Second):t.Fatal("capability mismatch hung")
    }
}

func TestM3JitterNegotiationByteExact(t *testing.T){
    addr,stop:=echo(t);defer stop()
    cl,_,cleanup:=m3Pair(t,addr,true,true)
    defer cleanup()
    local,peer:=net.Pipe()
    defer peer.Close()
    result:=make(chan error,1)
    go func(){result<-cl.Handle(local)}()
    _=peer.SetDeadline(time.Now().Add(10*time.Second))
    data:=bytes.Repeat([]byte{3,99,255,0},512)
    for i:=0;i<15;i++ {
        if _,err:=peer.Write(data);err!=nil{t.Fatal(err)}
        got:=make([]byte,len(data))
        if _,err:=io.ReadFull(peer,got);err!=nil{t.Fatal(err)}
        if !bytes.Equal(got,data){t.Fatal("jit1 changed payload")}
        time.Sleep(20*time.Millisecond)
    }
    _=peer.Close()
    select {
    case err:=<-result:
        if err!=nil && !errors.Is(err,io.EOF){t.Fatal(err)}
    case <-time.After(5*time.Second):t.Fatal("jit1 session did not finish")
    }
}

func TestM3PolicyValidationIndependent(t *testing.T){
    p:=padding.DefaultRatePolicy()
    if err:=p.Validate();err!=nil{t.Fatal(err)}
    j:=jitter.DefaultPolicy()
    if err:=j.Validate();err!=nil{t.Fatal(err)}
}

func TestM3JitterCapabilityRejectsUnsupportedServer(t *testing.T) {
    cl,_,cleanup:=m3Pair(t,"127.0.0.1:1",false,true)
    defer cleanup()
    j:=jitter.DefaultPolicy()
    cl.JitterPolicy=&j
    local,peer:=net.Pipe()
    defer peer.Close()
    result:=make(chan error,1)
    go func(){result<-cl.Handle(local)}()
    select {
    case err:=<-result:
        if err==nil || !bytes.Contains([]byte(err.Error()),[]byte("426")){
            t.Fatalf("jitter capability not rejected: %v",err)
        }
    case <-time.After(5*time.Second):t.Fatal("jitter capability mismatch hung")
    }
}

func TestM3ParallelVideoAndInteractiveStreams(t *testing.T) {
    addr,stop:=echo(t)
    defer stop()
    cl,sm,cleanup:=m3Pair(t,addr,true,true)
    defer cleanup()
    const streams=4
    var workers sync.WaitGroup
    failures:=make(chan error,streams)
    for index:=0;index<streams;index++{
        workers.Add(1)
        go func(index int){
            defer workers.Done()
            local,peer:=net.Pipe()
            defer peer.Close()
            completed:=make(chan error,1)
            go func(){completed<-cl.Handle(local)}()
            _=peer.SetDeadline(time.Now().Add(10*time.Second))
            size:=4096
            if index==streams-1 {size=128}
            data:=bytes.Repeat([]byte{0,255,1,42},size/4)
            for i:=0;i<60;i++ {
                if _,err:=peer.Write(data);err!=nil {failures<-err;return}
                read:=make([]byte,len(data))
                if _,err:=io.ReadFull(peer,read);err!=nil{failures<-err;return}
                if !bytes.Equal(read,data){failures<-errors.New("concurrent stream payload mismatch");return}
                time.Sleep(20*time.Millisecond)
            }
            _=peer.Close()
            select{
            case err:=<-completed:
                if err!=nil && !errors.Is(err,io.EOF){failures<-err;return}
            case <-time.After(5*time.Second):
                failures<-errors.New("concurrent HTTP2 stream starvation")
                return
            }
        }(index)
    }
    workers.Wait()
    close(failures)
    for err:=range failures {t.Fatal(err)}
    clientSnap:=cl.Metrics.Snapshot()
    serverSnap:=sm.Snapshot()
    if clientSnap.SessionsActive!=0 || serverSnap.SessionsActive!=0 ||
        clientSnap.SessionsStarted!=streams || clientSnap.SessionsCompleted!=streams {
        t.Fatalf("session leak or accounting mismatch client=%+v server=%+v",clientSnap,serverSnap)
    }
    if clientSnap.PaddingUploadFrames==0 || clientSnap.PaddingDownloadFrames==0 ||
        clientSnap.JitterUploadFrames==0 {
        t.Fatalf("concurrent rate/jitter not exercised PAD-upload=%d PAD-download=%d jitter=%d",
            clientSnap.PaddingUploadFrames,clientSnap.PaddingDownloadFrames,
            clientSnap.JitterUploadFrames)
    }
    if clientSnap.PaddingUploadWireOverheadBytes>clientSnap.UploadBytes*5/100 {
        t.Fatal("concurrent upload PAD overhead >5%")
    }
}
