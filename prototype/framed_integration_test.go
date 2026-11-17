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
  // Closing a TCP connection with unread data can surface ECONNRESET
  // instead of EOF on Linux. Either outcome proves the peer terminated.
  if e!=nil {
   var op *net.OpError
   if !errors.As(e,&op) || !strings.Contains(strings.ToLower(e.Error()),"reset") {
    t.Fatalf("unexpected backend read error: %v",e)
   }
  }
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

func TestFramedBackendEarlyEOFDoesNotForgeFIN(t *testing.T) {
 backend,err:=net.Listen("tcp","127.0.0.1:0")
 if err!=nil {t.Fatal(err)}
 defer backend.Close()
 backendDone:=make(chan struct{})
 go func(){
  defer close(backendDone)
  conn,e:=backend.Accept()
  if e!=nil {return}
  _=conn.Close() // Backend closes before upload FIN.
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
 req,e:=http.NewRequest(http.MethodPost,cl.URL,strings.NewReader("TBF1")) // Truncated header; no FIN.
 if e!=nil {t.Fatal(e)}
 req.Header.Set("Tervon-Service","vpn")
 req.Header.Set("Tervon-Wire-Version","1")
 req.Header.Set("Tervon-Wire-Capability","tbf1-fin")
 resp,e:=cl.HTTP.Do(req)
 if e!=nil {t.Fatal(e)}
 defer resp.Body.Close()
 if resp.StatusCode!=http.StatusOK {t.Fatalf("status %d",resp.StatusCode)}
 _,e=io.ReadAll(framing.NewDecoder(resp.Body))
 if !errors.Is(e,framing.ErrMalformed) {t.Fatalf("early backend EOF must not produce clean FIN: %v",e)}
 select {
 case <-backendDone:
 case <-time.After(5*time.Second):t.Fatal("backend did not terminate")
 }
}

func TestLegacyV1StreamRemainsByteExact(t *testing.T) {
 addr,stop:=echo(t)
 defer stop()
 cl:=setup(t,[]Backend{{Name:"legacy-echo",Address:addr}})
 if !strings.HasSuffix(cl.URL,"/v1/stream") {t.Fatalf("expected legacy endpoint: %s",cl.URL)}
 a,b:=net.Pipe()
 done:=make(chan error,1)
 go func(){done<-cl.Handle(a)}()
 defer b.Close()
 _=b.SetDeadline(time.Now().Add(8*time.Second))
 payload:=bytes.Repeat([]byte{0,255,10,13,0,1,2,3},8192)
 if _,err:=b.Write(payload);err!=nil {t.Fatal(err)}
 got:=make([]byte,len(payload))
 if _,err:=io.ReadFull(b,got);err!=nil {t.Fatal(err)}
 if !bytes.Equal(got,payload) {t.Fatal("legacy v1 bytes changed")}
 _=b.Close()
 select {
 case err:=<-done:
  if err!=nil {t.Fatalf("legacy v1 handler: %v",err)}
 case <-time.After(4*time.Second):t.Fatal("legacy v1 handler stuck")
 }
}


func TestFramedCancelRequestBodyReaderReleasesServerWorker(t *testing.T) {
	backend, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil { t.Fatal(err) }
	defer backend.Close()
	backendAccepted := make(chan struct{})
	backendDone := make(chan struct{})
	go func() {
		defer close(backendDone)
		c, err := backend.Accept()
		if err != nil { return }
		close(backendAccepted)
		defer c.Close()
		_, _ = io.Copy(io.Discard, c)
	}()

	cert, pool := cert(t)
	server := &Server{
		Routes: map[string][]Backend{"vpn": {{Address: backend.Addr().String()}}},
		AuthorizedServices: map[[32]byte]map[string]bool{
			sha256.Sum256(cert.Certificate[0]): {"vpn": true},
		},
	}
	handlerExited := make(chan struct{})
	hs := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(handlerExited)
		server.ServeHTTP(w, r)
	}))
	hs.EnableHTTP2 = true
	hs.TLS = &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs: pool,
		NextProtos: []string{"h2"},
	}
	hs.StartTLS()
	defer hs.Close()

	cl := NewClient(&tls.Config{
		RootCAs: pool, ServerName: "example.com",
		Certificates: []tls.Certificate{cert},
	}, hs.URL+"/v2/stream", "vpn")
	defer cl.HTTP.CloseIdleConnections()

	// Keep the upload open after a valid DATA frame, emulating a peer that
	// aborts its HTTP/2 stream before sending FIN.
	reader, writer := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cl.URL, reader)
	if err != nil { t.Fatal(err) }
	req.Header.Set("Tervon-Service", "vpn")
	req.Header.Set("Tervon-Wire-Version", "1")
	req.Header.Set("Tervon-Wire-Capability", "tbf1-fin")
	req.ContentLength = -1

	requestDone := make(chan error, 1)
	go func() {
		resp, err := cl.HTTP.Do(req)
		if err == nil {
			_ = resp.Body.Close()
		}
		requestDone <- err
	}()
	select {
	case <-backendAccepted:
	case <-time.After(4*time.Second): t.Fatal("backend was not dialed")
	}
	var frame bytes.Buffer
	if err := framing.WriteFrame(&frame, framing.TypeData, []byte("partial")); err != nil { t.Fatal(err) }
	if _, err := writer.Write(frame.Bytes()); err != nil {
		cancel()
		t.Fatalf("failed to upload DATA before cancellation: %v", err)
	}
	cancel()
	_ = writer.Close()
	select {
	case <-handlerExited:
	case <-time.After(4*time.Second): t.Fatal("server handler did not join upload worker on cancellation")
	}
	select {
	case <-backendDone:
	case <-time.After(4*time.Second): t.Fatal("backend socket remained open after cancellation")
	}
	select {
	case <-requestDone:
	case <-time.After(4*time.Second): t.Fatal("HTTP client request remained blocked after cancellation")
	}
}


func TestFramedBackendEarlyEOFWithOpenUploadReleasesHandler(t *testing.T) {
	backend, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil { t.Fatal(err) }
	defer backend.Close()
	backendAccepted := make(chan struct{})
	backendDone := make(chan struct{})
	go func() {
		defer close(backendDone)
		c, err := backend.Accept()
		if err != nil { return }
		close(backendAccepted)
		_ = c.Close() // Deliberately finish before the request sends FIN.
	}()

	cert, pool := cert(t)
	s := &Server{
		Routes: map[string][]Backend{"vpn": {{Address: backend.Addr().String()}}},
		AuthorizedServices: map[[32]byte]map[string]bool{
			sha256.Sum256(cert.Certificate[0]): {"vpn": true},
		},
	}
	handlerExited := make(chan struct{})
	hs := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(handlerExited)
		s.ServeHTTP(w, r)
	}))
	hs.EnableHTTP2 = true
	hs.TLS = &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs: pool,
		NextProtos: []string{"h2"},
	}
	hs.StartTLS()
	defer hs.Close()

	cl := NewClient(&tls.Config{
		RootCAs: pool, ServerName: "example.com",
		Certificates: []tls.Certificate{cert},
	}, hs.URL+"/v2/stream", "vpn")
	defer cl.HTTP.CloseIdleConnections()

	reader, writer := io.Pipe()
	defer writer.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cl.URL, reader)
	if err != nil { t.Fatal(err) }
	req.Header.Set("Tervon-Service", "vpn")
	req.Header.Set("Tervon-Wire-Version", "1")
	req.Header.Set("Tervon-Wire-Capability", "tbf1-fin")
	req.ContentLength = -1
	result := make(chan error, 1)
	go func() {
		resp, err := cl.HTTP.Do(req)
		if err != nil { result <- err; return }
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			result <- fmt.Errorf("status %d", resp.StatusCode)
			return
		}
		_, err = io.ReadAll(framing.NewDecoder(resp.Body))
		if !errors.Is(err, framing.ErrMalformed) {
			result <- fmt.Errorf("early backend EOF without upload FIN: got %v, want ErrMalformed", err)
			return
		}
		result <- nil
	}()
	select {
	case <-backendAccepted:
	case <-time.After(4*time.Second): t.Fatal("backend not dialed")
	}
	select {
	case err := <-result:
		if err != nil { t.Fatal(err) }
	case <-time.After(4*time.Second):
		t.Fatal("early backend EOF blocked framed response")
	}
	select {
	case <-handlerExited:
	case <-time.After(4*time.Second):
		t.Fatal("server handler remained blocked on open HTTP/2 request body")
	}
	select {
	case <-backendDone:
	case <-time.After(4*time.Second): t.Fatal("backend did not close")
	}
}
