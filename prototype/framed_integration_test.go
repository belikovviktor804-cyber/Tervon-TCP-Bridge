package bridge

import (
 "bytes"
 "context"
 "crypto/sha256"
 "crypto/tls"
 "io"
 "net"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
 "time"
)

func TestFramedByteExact(t *testing.T) {
 addr,stop:=echo(t);defer stop()
 cl:=setup(t,[]Backend{{Name:"echo",Address:addr}})
 cl.URL=strings.Replace(cl.URL,"/v1/stream","/v2/stream",1)
 a,b:=net.Pipe()
 done:=make(chan error,1)
 go func(){done<-cl.Handle(a)}()
 payload:=bytes.Repeat([]byte{0,1,255,2,3},20000)
 _=b.SetDeadline(time.Now().Add(10*time.Second))
 if _,err:=b.Write(payload);err!=nil {t.Fatal(err)}
 got:=make([]byte,len(payload))
 if _,err:=io.ReadFull(b,got);err!=nil {t.Fatal(err)}
 if !bytes.Equal(got,payload) {t.Fatal("framed byte mismatch")}
 _=b.Close()
 select {case <-done:case <-time.After(4*time.Second):t.Fatal("framed handler stuck")}
}

func TestFramedRejectVersionBeforeDial(t *testing.T) {
 cert,pool:=cert(t)
 s:=&Server{Routes:map[string][]Backend{"vpn":{{Address:"127.0.0.1:1"}}},AuthorizedServices:map[[32]byte]map[string]bool{sha256.Sum256(cert.Certificate[0]):{"vpn":true}}}
 hs:=httptest.NewUnstartedServer(s)
 hs.EnableHTTP2=true
 hs.TLS=&tls.Config{Certificates:[]tls.Certificate{cert},ClientAuth:tls.RequireAndVerifyClientCert,ClientCAs:pool,NextProtos:[]string{"h2"}}
 hs.StartTLS();defer hs.Close()
 cl:=NewClient(&tls.Config{RootCAs:pool,ServerName:"example.com",Certificates:[]tls.Certificate{cert}},hs.URL+"/v2/stream","vpn")
 defer cl.HTTP.CloseIdleConnections()
 req,err:=http.NewRequest(http.MethodPost,cl.URL,strings.NewReader(""))
 if err!=nil {t.Fatal(err)}
 req.Header.Set("Tervon-Service","vpn")
 req.Header.Set("Tervon-Wire-Version","99")
 req.Header.Set("Tervon-Wire-Capability","tbf1-fin")
 resp,err:=cl.HTTP.Do(req);if err!=nil {t.Fatal(err)}
 defer resp.Body.Close()
 if resp.StatusCode!=http.StatusUpgradeRequired {t.Fatalf("status %d",resp.StatusCode)}
}

func TestFramedCancellationReleasesHandler(t *testing.T) {
 addr,stop:=echo(t)
 defer stop()
 cl:=setup(t,[]Backend{{Name:"echo",Address:addr}})
 cl.URL=strings.Replace(cl.URL,"/v1/stream","/v2/stream",1)
 a,b:=net.Pipe()
 defer b.Close()
 ctx,cancel:=context.WithCancel(context.Background())
 done:=make(chan error,1)
 go func(){done<-cl.HandleContext(ctx,a)}()
 _=b.SetDeadline(time.Now().Add(5*time.Second))
 if _,err:=b.Write([]byte("hello"));err!=nil {cancel();t.Fatal(err)}
 out:=make([]byte,5)
 if _,err:=io.ReadFull(b,out);err!=nil {cancel();t.Fatal(err)}
 if string(out)!="hello" {cancel();t.Fatalf("echo %q",out)}
 cancel()
 select {
 case <-done:
 case <-time.After(4*time.Second):t.Fatal("framed handler stuck after context cancellation")
 }
}

