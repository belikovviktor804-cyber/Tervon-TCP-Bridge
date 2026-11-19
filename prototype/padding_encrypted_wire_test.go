package bridge

import (
    "bytes"
    "crypto/sha256"
    "crypto/tls"
    "io"
    "net"
    "net/http/httptest"
    "sync/atomic"
    "testing"
    "time"

    "tervon-tcp-bridge/padding"
    "tervon-tcp-bridge/telemetry"
)

// encryptedTrafficListener measures actual TCP bytes below the TLS listener.
// Counts include TLS records, HTTP/2 framing and TBF1 but not IP/TCP headers.
// This is a local CI regression measurement, never a live bandwidth claim.
type encryptedTrafficListener struct {
    net.Listener
    readBytes atomic.Uint64
    writeBytes atomic.Uint64
}
type encryptedTrafficConn struct {
    net.Conn
    owner *encryptedTrafficListener
}
func (l *encryptedTrafficListener) Accept() (net.Conn,error) {
    c,err:=l.Listener.Accept()
    if err!=nil{return nil,err}
    return &encryptedTrafficConn{Conn:c,owner:l},nil
}
func (c *encryptedTrafficConn) Read(p []byte)(int,error){
    n,err:=c.Conn.Read(p)
    if n>0{c.owner.readBytes.Add(uint64(n))}
    return n,err
}
func (c *encryptedTrafficConn) Write(p []byte)(int,error){
    n,err:=c.Conn.Write(p)
    if n>0{c.owner.writeBytes.Add(uint64(n))}
    return n,err
}

type encryptedWireSample struct {
    incoming uint64
    outgoing uint64
    padUp uint64
    padDown uint64
    useful uint64
}

func measureM2EncryptedTraffic(t *testing.T, enabled bool) encryptedWireSample {
    t.Helper()
    backend,stopBackend:=echo(t)
    defer stopBackend()
    cert,pool:=cert(t)
    p:=padding.DefaultPolicy()
    // Test-only accelerated activation so all traffic is measured within
    // the bounded CI run, without changing production policy.
    p.MinDuration=time.Nanosecond
    p.MinSamples=2
    p.MinUsefulBytes=1024
    metrics:=telemetry.New()
    handler:=&Server{
        Routes:map[string][]Backend{"vpn":{{Name:"echo",Address:backend}}},
        AuthorizedServices:map[[32]byte]map[string]bool{
            sha256.Sum256(cert.Certificate[0]):{"vpn":true},
        },
        Metrics:metrics,
    }
    if enabled {handler.PaddingPolicy=&p}
    hs:=httptest.NewUnstartedServer(handler)
    hs.EnableHTTP2=true
    hs.TLS=&tls.Config{
        Certificates:[]tls.Certificate{cert},
        ClientAuth:tls.RequireAndVerifyClientCert,
        ClientCAs:pool,
        NextProtos:[]string{"h2"},
    }
    counter:=&encryptedTrafficListener{Listener:hs.Listener}
    hs.Listener=counter
    hs.StartTLS()
    defer hs.Close()

    cl:=NewClient(&tls.Config{
        RootCAs:pool,ServerName:"example.com",
        Certificates:[]tls.Certificate{cert},
    },hs.URL+"/v2/stream","vpn")
    defer cl.HTTP.CloseIdleConnections()
    cl.Metrics=telemetry.New()
    if enabled {cl.PaddingPolicy=&p}

    local,peer:=net.Pipe()
    finished:=make(chan error,1)
    go func(){finished<-cl.Handle(local)}()
    defer peer.Close()
    if err:=peer.SetDeadline(time.Now().Add(8*time.Second));err!=nil{t.Fatal(err)}
    payload:=bytes.Repeat([]byte{0,255,1,3},4096)
    const frames=12
    for i:=0;i<frames;i++ {
        if n,err:=peer.Write(payload);err!=nil||n!=len(payload) {
            t.Fatalf("write %d: n=%d err=%v",i,n,err)
        }
        reply:=make([]byte,len(payload))
        if _,err:=io.ReadFull(peer,reply);err!=nil{t.Fatal(err)}
        if !bytes.Equal(reply,payload){t.Fatal("encrypted transport corrupted bytes")}
        time.Sleep(time.Millisecond)
    }
    _=peer.Close()
    select {
    case err:=<-finished:
        if err!=nil && err!=io.EOF{t.Fatalf("bridge handle returned %v",err)}
    case <-time.After(5*time.Second):t.Fatal("bridge handler did not terminate")
    }
    cl.HTTP.CloseIdleConnections()
    return encryptedWireSample{
        incoming:counter.readBytes.Load(),
        outgoing:counter.writeBytes.Load(),
        padUp:cl.Metrics.Snapshot().PaddingUploadWireOverheadBytes,
        padDown:metrics.Snapshot().PaddingDownloadWireOverheadBytes,
        useful:uint64(frames*len(payload)),
    }
}

// TestM2EncryptedHTTP2Accounting measures both directions after serializing
// a fixed workload over mutually authenticated TLS. Baseline and M2 establish
// separate connections, so TLS handshake cost is included in both measurements.
// It does NOT assert a deterministic byte delta: TLS/HTTP2 frame aggregation
// can change depending on the scheduler and runtime.
func TestM2EncryptedHTTP2Accounting(t *testing.T) {
    baseline:=measureM2EncryptedTraffic(t,false)
    padded:=measureM2EncryptedTraffic(t,true)
    if baseline.incoming==0 || baseline.outgoing==0 ||
        padded.incoming==0 || padded.outgoing==0 {t.Fatal("no encrypted bytes measured")}
    if baseline.useful!=padded.useful {t.Fatal("workloads differ")}
    if baseline.padUp!=0||baseline.padDown!=0{t.Fatal("legacy baseline emitted PAD")}
    if padded.padUp==0||padded.padDown==0 {t.Fatal("M2 did not emit real PAD")}
    t.Logf("encrypted TCP bytes below TLS: M1 in=%d out=%d, M2 in=%d out=%d; useful/direction=%d; M2 PAD wire upload=%d download=%d (includes TBF1 header; TCP/IP headers excluded)",
        baseline.incoming,baseline.outgoing,padded.incoming,padded.outgoing,
        baseline.useful,padded.padUp,padded.padDown)
}
