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

type failingWriter struct{ err error }
func (w failingWriter) Write([]byte)(int,error){return 0,w.err}

func TestStreamEncoderWriteFailureIsTerminal(t *testing.T) {
 sentinel:=errors.New("write failed")
 enc:=NewEncoder(failingWriter{sentinel})
 if n,err:=enc.Write([]byte("data"));n!=0 || !errors.Is(err,sentinel) {t.Fatalf("write: %d %v",n,err)}
 if _,err:=enc.Write([]byte("retry"));!errors.Is(err,io.ErrClosedPipe) {t.Fatalf("retry after failure: %v",err)}
 if err:=enc.CloseWrite();!errors.Is(err,io.ErrClosedPipe) {t.Fatalf("FIN after failure: %v",err)}
}
func TestStreamEncoderCloseWriteFailureIsTerminal(t *testing.T) {
 sentinel:=errors.New("FIN write failed")
 enc:=NewEncoder(failingWriter{sentinel})
 if err:=enc.CloseWrite();!errors.Is(err,sentinel) {t.Fatalf("close: %v",err)}
 if err:=enc.CloseWrite();!errors.Is(err,io.ErrClosedPipe) {t.Fatalf("second close: %v",err)}
}
func TestStreamRejectTruncatedPayload(t *testing.T) {
 var b bytes.Buffer
 if err:=WriteFrame(&b,TypeData,[]byte("payload"));err!=nil {t.Fatal(err)}
 raw:=b.Bytes()
 for n:=HeaderSize;n<len(raw);n++ {
  _,err:=io.ReadAll(NewDecoder(bytes.NewReader(raw[:n])))
  if !errors.Is(err,ErrMalformed) {t.Fatalf("truncated at %d: %v",n,err)}
 }
}
func FuzzStreamDecoder(f *testing.F) {
 f.Add([]byte{})
 f.Add([]byte("TBF1"))
 var valid bytes.Buffer
 enc:=NewEncoder(&valid)
 _,_=enc.Write([]byte("payload"))
 _=enc.CloseWrite()
 f.Add(valid.Bytes())
 var missingFIN bytes.Buffer
 _=WriteFrame(&missingFIN,TypeData,[]byte("payload"))
 f.Add(missingFIN.Bytes())
 f.Fuzz(func(t *testing.T,data []byte){
  if len(data)>2*MaxPayload+2*HeaderSize+100 {return}
  _,err:=io.ReadAll(NewDecoder(bytes.NewReader(data)))
  if err==nil {
   // Every accepted stream must end with exactly one FIN and no tail.
   raw:=bytes.NewReader(data)
   for {
    frame,e:=ReadFrame(raw)
    if e!=nil {t.Fatalf("decoder accepted invalid frame: %v",e)}
    if frame.Type==TypeFIN {
     if raw.Len()!=0 {t.Fatal("accepted bytes after FIN")}
     break
    }
   }
  }
 })
}

func TestM3DecoderPADObserverCountsOnlyValidatedFrames(t *testing.T) {
    var wire bytes.Buffer
    if err:=WriteFrame(&wire,TypeData,bytes.Repeat([]byte{5},4096));err!=nil{t.Fatal(err)}
    if err:=WritePaddingFrame(&wire,bytes.Repeat([]byte{7},100));err!=nil{t.Fatal(err)}
    if err:=WriteFrame(&wire,TypeFIN,nil);err!=nil{t.Fatal(err)}
    decoder:=NewPaddingDecoder(&wire)
    var called int
    var amount uint64
    decoder.SetPaddingObserver(func(n uint64){called++;amount+=n})
    got,err:=io.ReadAll(decoder)
    if err!=nil || len(got)!=4096 || called!=1 || amount!=100 {
        t.Fatalf("PAD callback mismatch: length=%d callbacks=%d PAD=%d err=%v",len(got),called,amount,err)
    }

    var bad bytes.Buffer
    if err:=WriteFrame(&bad,TypeData,bytes.Repeat([]byte{1},100));err!=nil{t.Fatal(err)}
    if err:=WritePaddingFrame(&bad,bytes.Repeat([]byte{9},100));err!=nil{t.Fatal(err)}
    invalid:=NewPaddingDecoder(&bad)
    var rejectedCalled bool
    invalid.SetPaddingObserver(func(uint64){rejectedCalled=true})
    _,err=io.ReadAll(invalid)
    if !errors.Is(err,ErrMalformed) || rejectedCalled {
        t.Fatalf("malformed PAD counted: err=%v called=%v",err,rejectedCalled)
    }
}
