package bridge

import (
    "bytes"
    "crypto/sha256"
    "crypto/tls"
    "errors"
    "io"
    "net"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "tervon-tcp-bridge/padding"
)

func paddingTestPair(t *testing.T, enabled bool, backend string) (*Client,func()) {
    t.Helper()
    cert,pool:=cert(t)
    srv:=&Server{
        Routes:map[string][]Backend{"vpn":{{Name:"test",Address:backend}}},
        AuthorizedServices:map[[32]byte]map[string]bool{
            sha256.Sum256(cert.Certificate[0]):{"vpn":true},
        },
    }
    if enabled { p:=padding.DefaultPolicy();srv.PaddingPolicy=&p }
    hs:=httptest.NewUnstartedServer(srv)
    hs.EnableHTTP2=true
    hs.TLS=&tls.Config{
        Certificates:[]tls.Certificate{cert},
        ClientAuth:tls.RequireAndVerifyClientCert,
        ClientCAs:pool,NextProtos:[]string{"h2"},
    }
    hs.StartTLS()
    cl:=NewClient(&tls.Config{
        RootCAs:pool,ServerName:"example.com",
        Certificates:[]tls.Certificate{cert},
    },hs.URL+"/v2/stream","vpn")
    p:=padding.DefaultPolicy()
    cl.PaddingPolicy=&p
    return cl,func(){cl.HTTP.CloseIdleConnections();hs.Close()}
}

func TestM2PaddingNegotiatedRoundTrip(t *testing.T){
    addr,stop:=echo(t)
    defer stop()
    cl,closePair:=paddingTestPair(t,true,addr)
    defer closePair()
    local,peer:=net.Pipe()
    defer peer.Close()
    finished:=make(chan error,1)
    go func(){finished<-cl.Handle(local)}()
    _=peer.SetDeadline(time.Now().Add(10*time.Second))
    payload:=bytes.Repeat([]byte{0,1,255,7},2048)
    for i:=0;i<4;i++{
        if _,err:=peer.Write(payload);err!=nil{t.Fatal(err)}
        got:=make([]byte,len(payload))
        if _,err:=io.ReadFull(peer,got);err!=nil{t.Fatal(err)}
        if !bytes.Equal(got,payload){t.Fatal("roundtrip changed data")}
    }
    _=peer.Close()
    select{
    case err:=<-finished:
        if err!=nil && !errors.Is(err,io.EOF){t.Fatalf("handler: %v",err)}
    case <-time.After(5*time.Second):t.Fatal("padded handler did not finish")
    }
}

func TestM2RejectsPaddingWhenServerDisabled(t *testing.T){
    cl,closePair:=paddingTestPair(t,false,"127.0.0.1:1")
    defer closePair()
    local,peer:=net.Pipe()
    defer peer.Close()
    result:=make(chan error,1)
    go func(){result<-cl.Handle(local)}()
    select {
    case err:=<-result:
        if err==nil || !bytes.Contains([]byte(err.Error()),[]byte("426")){
            t.Fatalf("expected capability rejection 426, got %v",err)
        }
    case <-time.After(5*time.Second):t.Fatal("capability rejection stuck")
    }
}

func TestM2DisabledClientRetainsM1Wire(t *testing.T){
    addr,stop:=echo(t)
    defer stop()
    cl,closePair:=paddingTestPair(t,true,addr)
    defer closePair()
    cl.PaddingPolicy=nil
    local,peer:=net.Pipe()
    defer peer.Close()
    finished:=make(chan error,1)
    go func(){finished<-cl.Handle(local)}()
    _=peer.SetDeadline(time.Now().Add(5*time.Second))
    if _,err:=peer.Write([]byte("legacy"));err!=nil{t.Fatal(err)}
    got:=make([]byte,6)
    if _,err:=io.ReadFull(peer,got);err!=nil{t.Fatal(err)}
    if string(got)!="legacy"{t.Fatalf("legacy data: %q",got)}
    _=peer.Close()
    select {case <-finished:case <-time.After(5*time.Second):t.Fatal("legacy handler stuck")}
}

func TestM2WrongCapabilityRejectedBeforeBackend(t *testing.T){
    cert,pool:=cert(t)
    srv:=&Server{
        Routes:map[string][]Backend{"vpn":{{Address:"127.0.0.1:1"}}},
        AuthorizedServices:map[[32]byte]map[string]bool{
            sha256.Sum256(cert.Certificate[0]):{"vpn":true},
        },
    }
    p:=padding.DefaultPolicy()
    srv.PaddingPolicy=&p
    hs:=httptest.NewUnstartedServer(srv)
    hs.EnableHTTP2=true
    hs.TLS=&tls.Config{Certificates:[]tls.Certificate{cert},ClientAuth:tls.RequireAndVerifyClientCert,ClientCAs:pool,NextProtos:[]string{"h2"}}
    hs.StartTLS()
    defer hs.Close()
    cl:=NewClient(&tls.Config{RootCAs:pool,ServerName:"example.com",Certificates:[]tls.Certificate{cert}},hs.URL+"/v2/stream","vpn")
    defer cl.HTTP.CloseIdleConnections()
    req,_:=http.NewRequest(http.MethodPost,cl.URL,bytes.NewReader(nil))
    req.Header.Set("Tervon-Service","vpn")
    req.Header.Set("Tervon-Wire-Version","1")
    req.Header.Set("Tervon-Wire-Capability","tbf1-fin,pad1,unknown")
    resp,err:=cl.HTTP.Do(req)
    if err!=nil{t.Fatal(err)}
    defer resp.Body.Close()
    if resp.StatusCode!=http.StatusUpgradeRequired{t.Fatalf("status=%d",resp.StatusCode)}
}
