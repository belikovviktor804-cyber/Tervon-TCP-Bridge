package framing

import (
 "bytes"
 "errors"
 "io"
 "testing"
)

func TestStreamRoundTrip(t *testing.T) {
 payload:=bytes.Repeat([]byte{0,255,1,2,3},MaxPayload/2+19)
 var wire bytes.Buffer
 enc:=NewEncoder(&wire)
 n,err:=enc.Write(payload); if err!=nil || n!=len(payload) { t.Fatalf("write %d %v",n,err) }
 if err:=enc.CloseWrite();err!=nil {t.Fatal(err)}
 if _,err:=enc.Write([]byte("x"));!errors.Is(err,io.ErrClosedPipe) {t.Fatalf("write after FIN: %v",err)}
 got,err:=io.ReadAll(NewDecoder(&wire))
 if err!=nil || !bytes.Equal(got,payload) {t.Fatalf("roundtrip %d %v",len(got),err)}
}
func TestStreamRequiresFIN(t *testing.T) {
 var b bytes.Buffer
 if err:=WriteFrame(&b,TypeData,[]byte("abc"));err!=nil {t.Fatal(err)}
 got,err:=io.ReadAll(NewDecoder(&b))
 if string(got)!="abc" || !errors.Is(err,ErrMalformed) {t.Fatalf("got %q err %v",got,err)}
}
func TestStreamFIN(t *testing.T) {
 var b bytes.Buffer
 if err:=WriteFrame(&b,TypeFIN,nil);err!=nil {t.Fatal(err)}
 d:=NewDecoder(&b)
 p:=make([]byte,2)
 if n,err:=d.Read(p);n!=0 || err!=io.EOF {t.Fatalf("fin: %d %v",n,err)}
 if n,err:=d.Read(p);n!=0 || err!=io.EOF {t.Fatalf("repeat fin: %d %v",n,err)}
}
func TestStreamCorrupt(t *testing.T) {
 var b bytes.Buffer
 if err:=WriteFrame(&b,TypeData,[]byte("abc"));err!=nil {t.Fatal(err)}
 raw:=b.Bytes();raw[4]=99
 _,err:=io.ReadAll(NewDecoder(&b))
 if !errors.Is(err,ErrVersion) {t.Fatalf("got %v",err)}
}
func TestStreamSmallReads(t *testing.T) {
 var b bytes.Buffer
 enc:=NewEncoder(&b)
 if _,err:=enc.Write([]byte("abcdef"));err!=nil {t.Fatal(err)}
 if err:=enc.CloseWrite();err!=nil {t.Fatal(err)}
 d:=NewDecoder(&b)
 p:=make([]byte,1)
 for _,want:=range []byte("abcdef") {
  n,err:=d.Read(p);if n!=1 || err!=nil || p[0]!=want {t.Fatalf("n=%d err=%v byte=%v",n,err,p[0])}
 }
 if _,err:=d.Read(p);err!=io.EOF {t.Fatalf("want EOF: %v",err)}
}

func TestStreamRejectTrailingAfterFIN(t *testing.T) {
 for _,tail:=range [][]byte{[]byte("garbage"),{0}} {
  var b bytes.Buffer
  if err:=WriteFrame(&b,TypeFIN,nil);err!=nil {t.Fatal(err)}
  b.Write(tail)
  _,err:=io.ReadAll(NewDecoder(&b))
  if !errors.Is(err,ErrMalformed) {t.Fatalf("tail %x accepted: %v",tail,err)}
 }
}
func TestStreamRejectSecondFIN(t *testing.T) {
 var b bytes.Buffer
 if err:=WriteFrame(&b,TypeFIN,nil);err!=nil {t.Fatal(err)}
 if err:=WriteFrame(&b,TypeFIN,nil);err!=nil {t.Fatal(err)}
 _,err:=io.ReadAll(NewDecoder(&b))
 if !errors.Is(err,ErrMalformed) {t.Fatalf("second FIN accepted: %v",err)}
}
