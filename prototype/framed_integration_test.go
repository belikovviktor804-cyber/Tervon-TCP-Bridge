package bridge

import (
 "bytes"
 "context"
 "crypto/sha256"
 "crypto/tls"
 "errors"
 "fmt"
 "io"
 "net"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
 "time"
 "tervon-tcp-bridge/framing"
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


func TestFramedHalfCloseResponseAfterUploadEOF(t *testing.T) {
 backend,err:=net.Listen("tcp","127.0.0.1:0")
 if err!=nil {t.Fatal(err)}
 defer backend.Close()
 backendDone:=make(chan error,1)
 go func(){
  conn,e:=backend.Accept()
  if e!=nil {backendDone<-e;return}
  defer conn.Close()
  _=conn.SetDeadline(time.Now().Add(6*time.Second))
  body,e:=io.ReadAll(conn)
  if e!=nil {backendDone<-e;return}
  if !bytes.Equal(body,[]byte("request-after-half-close")) {
   backendDone<-fmt.Errorf("backend received %q",body);return
  }
  _,e=conn.Write([]byte("response-after-half-close"))
  backendDone<-e
 }()
 cl:=setup(t,[]Backend{{Name:"eof-response",Address:backend.Addr().String()}})
 cl.URL=strings.Replace(cl.URL,"/v1/stream","/v2/stream",1)
 local,err:=net.Listen("tcp","127.0.0.1:0")
 if err!=nil {t.Fatal(err)}
 defer local.Close()
 accepted:=make(chan error,1)
 go func(){
  conn,e:=local.Accept()
  if e!=nil {accepted<-e;return}
  accepted<-cl.Handle(conn)
 }()
 peer,err:=net.Dial("tcp",local.Addr().String())
 if err!=nil {t.Fatal(err)}
 defer peer.Close()
 _=peer.SetDeadline(time.Now().Add(8*time.Second))
 if _,err=peer.Write([]byte("request-after-half-close"));err!=nil {t.Fatal(err)}
 if err=peer.(*net.TCPConn).CloseWrite();err!=nil {t.Fatal(err)}
 response,err:=io.ReadAll(peer)
 if err!=nil {t.Fatal(err)}
 if string(response)!="response-after-half-close" {t.Fatalf("response %q",response)}
 select {case err:=<-accepted:if err!=nil {t.Fatal(err)}
 case <-time.After(4*time.Second):t.Fatal("handler did not terminate")}
 select {case err:=<-backendDone:if err!=nil {t.Fatal(err)}
 case <-time.After(4*time.Second):t.Fatal("backend did not terminate")}
}

func TestFramedRejectsTruncatedDownload(t *testing.T) {
 cert,pool:=cert(t)
 handler:=http.HandlerFunc(func(w http.ResponseWriter,r *http.Request) {
  if r.URL.Path!="/v2/stream" {http.NotFound(w,r);return}
  w.Header().Set("Tervon-Wire-Version","1")
  w.Header().Set("Tervon-Wire-Capability","tbf1-fin")
  w.WriteHeader(http.StatusOK)
  _,_=w.Write([]byte("TBF1")) // Incomplete frame header, no FIN.
 })
 hs:=httptest.NewUnstartedServer(handler)
 hs.EnableHTTP2=true
 hs.TLS=&tls.Config{Certificates:[]tls.Certificate{cert},ClientAuth:tls.RequireAndVerifyClientCert,ClientCAs:pool,NextProtos:[]string{"h2"}}
 hs.StartTLS()
 defer hs.Close()
 cl:=NewClient(&tls.Config{RootCAs:pool,ServerName:"example.com",Certificates:[]tls.Certificate{cert}},hs.URL+"/v2/stream","vpn")
 defer cl.HTTP.CloseIdleConnections()
 a,peer:=net.Pipe()
 defer peer.Close()
 result:=make(chan error,1)
 go func(){result<-cl.Handle(a)}()
 select {
 case err:=<-result:
  if !errors.Is(err,framing.ErrMalformed) {t.Fatalf("truncated framed response: got %v, want ErrMalformed",err)}
 case <-time.After(5*time.Second):
  t.Fatal("truncated framed response blocked handler")
 }
}

func TestFramedMalformedUploadClosesBackend(t *testing.T) {
 backend,err:=net.Listen("tcp","127.0.0.1:0")
 if err!=nil {t.Fatal(err)}
 defer backend.Close()
 observed:=make(chan error,1)
 go func(){
  conn,e:=backend.Accept()
  if e!=nil {observed<-e;return}
  defer conn.Close()
  _=conn.SetReadDeadline(time.Now().Add(5*time.Second))
  _,e=io.Copy(io.Discard,conn)
  observed<-e
 }()
 cert,pool:=cert(t)
 s:=&Server{
  Routes:map[string][]Backend{"vpn":{{Address:backend.Addr().String()}}},
  AuthorizedServices:map[[32]byte]map[string]bool{sha256.Sum256(cert.Certificate[0]):{"vpn":true}},
 }
 hs:=httptest.NewUnstartedServer(s)
 hs.EnableHTTP2=true
 hs.TLS=&tls.Config{Certificates:[]tls.Certificate{cert},ClientAuth:tls.RequireAndVerifyClientCert,ClientCAs:pool,NextProtos:[]string{"h2"}}
 hs.StartTLS()
 defer hs.Close()
 cl:=NewClient(&tls.Config{RootCAs:pool,ServerName:"example.com",Certificates:[]tls.Certificate{cert}},hs.URL+"/v2/stream","vpn")
 defer cl.HTTP.CloseIdleConnections()
 req,e:=http.NewRequest(http.MethodPost,cl.URL,strings.NewReader("TBF1"))
 if e!=nil {t.Fatal(e)}
 req.Header.Set("Tervon-Service","vpn")
 req.Header.Set("Tervon-Wire-Version","1")
 req.Header.Set("Tervon-Wire-Capability","tbf1-fin")
 resp,e:=cl.HTTP.Do(req)
 if e!=nil {t.Fatal(e)}
 _=resp.Body.Close()
 select {
 case e:=<-observed:
  if e!=nil {t.Fatalf("backend not closed after malformed upload: %v",e)}
 case <-time.After(6*time.Second):
  t.Fatal("backend did not close after malformed upload")
 }
}

func TestFramedRejectsMissingResponseNegotiation(t *testing.T) {
 cert,pool:=cert(t)
 handler:=http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  w.WriteHeader(http.StatusOK)
 })
 hs:=httptest.NewUnstartedServer(handler)
 hs.EnableHTTP2=true
 hs.TLS=&tls.Config{Certificates:[]tls.Certificate{cert},ClientAuth:tls.RequireAndVerifyClientCert,ClientCAs:pool,NextProtos:[]string{"h2"}}
 hs.StartTLS()
 defer hs.Close()
 cl:=NewClient(&tls.Config{RootCAs:pool,ServerName:"example.com",Certificates:[]tls.Certificate{cert}},hs.URL+"/v2/stream","vpn")
 defer cl.HTTP.CloseIdleConnections()
 a,peer:=net.Pipe()
 defer peer.Close()
 result:=make(chan error,1)
 go func(){result<-cl.Handle(a)}()
 select {
 case err:=<-result:
  if err==nil || !strings.Contains(err.Error(),"framed protocol mismatch") {t.Fatalf("unexpected result: %v",err)}
 case <-time.After(5*time.Second):t.Fatal("negotiation mismatch blocked handler")
 }
}
