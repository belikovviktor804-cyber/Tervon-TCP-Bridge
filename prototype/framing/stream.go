package framing

import (
 "errors"
 "io"
)

// Encoder writes an opaque byte stream as bounded DATA frames.
// CloseWrite emits exactly one FIN. It does not close the underlying writer.
type Encoder struct { w io.Writer; closed bool }
func NewEncoder(w io.Writer) *Encoder { return &Encoder{w:w} }
func (e *Encoder) Write(p []byte) (int,error) {
 if e.closed { return 0, io.ErrClosedPipe }
 total:=0
 for len(p)>0 {
  n:=len(p); if n>MaxPayload { n=MaxPayload }
  if err:=WriteFrame(e.w,TypeData,p[:n]); err!=nil { e.closed=true; return total,err }
  total+=n; p=p[n:]
 }
 return total,nil
}
func (e *Encoder) CloseWrite() error {
 if e.closed { return io.ErrClosedPipe }
 e.closed=true
 return WriteFrame(e.w,TypeFIN,nil)
}

// Decoder exposes DATA payload as an io.Reader. A FIN is required;
// raw EOF without FIN is a truncated stream, never a successful half-close.
type Decoder struct { r io.Reader; pending []byte; finished bool }
func NewDecoder(r io.Reader) *Decoder { return &Decoder{r:r} }
func (d *Decoder) Read(p []byte) (int,error) {
 if len(p)==0 { return 0,nil }
 if len(d.pending)>0 { n:=copy(p,d.pending); d.pending=d.pending[n:]; return n,nil }
 if d.finished { return 0,io.EOF }
 f,err:=ReadFrame(d.r)
 if err!=nil {
  if errors.Is(err,io.EOF) { return 0,ErrMalformed }
  return 0,err
 }
 if f.Type==TypeFIN {
  // FIN must terminate the framed direction. A trailing frame or byte
  // is a protocol violation, not a second independent stream.
  var extra [1]byte
  n,err:=d.r.Read(extra[:])
  if n!=0 || err==nil { return 0,ErrMalformed }
  if !errors.Is(err,io.EOF) { return 0,err }
  d.finished=true
  return 0,io.EOF
 }
 d.pending=f.Payload
 n:=copy(p,d.pending); d.pending=d.pending[n:]
 return n,nil
}
