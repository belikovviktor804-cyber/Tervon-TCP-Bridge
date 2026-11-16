package bridge

import (
 "bytes"
 "io"
 "net"
 "testing"
 "time"
)

// A backend that closes immediately must not leave the session blocked.
func TestBackendEarlyEOFReturns(t *testing.T) {
 ln, err := net.Listen("tcp", "127.0.0.1:0")
 if err != nil { t.Fatal(err) }
 defer ln.Close()
 go func() { c,e := ln.Accept(); if e == nil { _ = c.Close() } }()
 cl := setup(t, []Backend{{Name:"close",Address:ln.Addr().String()}})
 a,b := net.Pipe()
 defer b.Close()
 done := make(chan error,1)
 go func(){ done <- cl.Handle(a) }()
 select {
 case <-done:
 case <-time.After(4*time.Second): t.Fatal("backend EOF left session blocked")
 }
}

// Real TCP half-close: upload EOF is delivered before backend replies.
func TestTCPHalfClose(t *testing.T) {
 backend, err := net.Listen("tcp","127.0.0.1:0")
 if err != nil {t.Fatal(err)}
 defer backend.Close()
 go func(){ c,e:=backend.Accept(); if e!=nil{return}; defer c.Close(); data,_:=io.ReadAll(c); _,_=c.Write(data) }()
 cl:=setup(t,[]Backend{{Name:"eof-echo",Address:backend.Addr().String()}})
 local,err:=net.Listen("tcp","127.0.0.1:0")
 if err!=nil{t.Fatal(err)}
 defer local.Close()
 done:=make(chan error,1)
 go func(){c,e:=local.Accept();if e!=nil{done<-e;return};done<-cl.Handle(c)}()
 raw,err:=net.Dial("tcp",local.Addr().String())
 if err!=nil{t.Fatal(err)}
 conn:=raw.(*net.TCPConn)
 defer conn.Close()
 _=conn.SetDeadline(time.Now().Add(5*time.Second))
 payload:=[]byte("request-completes-before-response")
 if _,err=conn.Write(payload);err!=nil{t.Fatal(err)}
 if err=conn.CloseWrite();err!=nil{t.Fatal(err)}
 got,err:=io.ReadAll(conn)
 if err!=nil||!bytes.Equal(got,payload){t.Fatalf("half-close: got %q err %v",got,err)}
 select{
 case err:=<-done:if err!=nil{t.Fatal(err)}
 case <-time.After(3*time.Second):t.Fatal("session did not terminate")
 }
}
