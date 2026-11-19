package bridge

import (
 "bytes"
 "crypto/sha256"
 "crypto/tls"
 "errors"
 "fmt"
 "io"
 "net"
 "net/http/httptest"
 "runtime"
 "sort"
 "sync"
 "sync/atomic"
 "testing"
 "time"

 "tervon-tcp-bridge/jitter"
 "tervon-tcp-bridge/padding"
 "tervon-tcp-bridge/telemetry"
)

func m5LoopbackPair(t *testing.T, backend string) (*Client,*telemetry.Registry) {
 t.Helper()
 cert, pool := cert(t)
 pp:=padding.DefaultPolicy()
 rp:=padding.DefaultRatePolicy()
 jp:=jitter.DefaultPolicy()
 serverMetric:=telemetry.New()
 s:=&Server{
  Metrics:serverMetric,
  Routes:map[string][]Backend{"vpn":{{Name:"loopback",Address:backend}}},
  AuthorizedServices:map[[32]byte]map[string]bool{sha256.Sum256(cert.Certificate[0]):{"vpn":true}},
  PaddingPolicy:&pp,RatePaddingPolicy:&rp,JitterPolicy:&jp,
 }
 hs:=httptest.NewUnstartedServer(s)
 hs.EnableHTTP2=true
 hs.TLS=&tls.Config{Certificates:[]tls.Certificate{cert},ClientAuth:tls.RequireAndVerifyClientCert,ClientCAs:pool,NextProtos:[]string{"h2"}}
 hs.StartTLS()
 t.Cleanup(hs.Close)
 cl:=NewClient(&tls.Config{RootCAs:pool,ServerName:"example.com",Certificates:[]tls.Certificate{cert}},hs.URL+"/v2/stream","vpn")
 cl.PaddingPolicy=&pp
 cl.RatePaddingPolicy=&rp
 cl.JitterPolicy=&jp
 cl.Metrics=telemetry.New()
 t.Cleanup(cl.HTTP.CloseIdleConnections)
 return cl, serverMetric
}

func m5RoundTrip(cl *Client, payload []byte, deadline time.Duration) (time.Duration,error) {
 local,peer:=net.Pipe()
 done:=make(chan error,1)
 go func(){done<-cl.Handle(local)}()
 _=peer.SetDeadline(time.Now().Add(deadline))
 started:=time.Now()
 _,err:=peer.Write(payload)
 if err==nil {
  response:=make([]byte,len(payload))
  _,err=io.ReadFull(peer,response)
  if err==nil && !bytes.Equal(payload,response) {err=errors.New("payload mismatch")}
 }
 elapsed:=time.Since(started)
 _=peer.Close()
 select {
 case handleErr:=<-done:
  if err==nil && handleErr!=nil && !errors.Is(handleErr,io.EOF){err=handleErr}
 case <-time.After(3*time.Second):
  if err==nil{err=errors.New("handler did not finish")}
 }
 return elapsed,err
}

func TestM5LoopbackRealTransportFairnessAndDelay(t *testing.T) {
 addr,stop:=echo(t)
 defer stop()
 cl,serverMetric:=m5LoopbackPair(t,addr)
 const streams=4
 const frames=128
 var completed [streams]atomic.Int64
 var wg sync.WaitGroup
 faults:=make(chan error,streams)
 start:=time.Now()
 var before runtime.MemStats
 runtime.ReadMemStats(&before)
 for stream:=0;stream<streams;stream++ {
  wg.Add(1)
  go func(id int) {
   defer wg.Done()
   local,peer:=net.Pipe()
   done:=make(chan error,1)
   go func(){done<-cl.Handle(local)}()
   _=peer.SetDeadline(time.Now().Add(18*time.Second))
   payload:=bytes.Repeat([]byte{byte(id),0x42,0x00,0xff},4096)
   for i:=0;i<frames;i++ {
    if _,err:=peer.Write(payload);err!=nil {faults<-fmt.Errorf("stream %d write frame %d: %w",id,i,err);_ =peer.Close();return}
    got:=make([]byte,len(payload))
    if _,err:=io.ReadFull(peer,got);err!=nil {faults<-fmt.Errorf("stream %d read frame %d: %w",id,i,err);_=peer.Close();return}
    if !bytes.Equal(got,payload) {faults<-fmt.Errorf("stream %d altered bytes at %d",id,i);_=peer.Close();return}
    completed[id].Add(1)
    time.Sleep(25*time.Millisecond)
   }
   _=peer.Close()
   select {
   case err:=<-done:
    if err!=nil && !errors.Is(err,io.EOF){faults<-fmt.Errorf("stream %d handle: %w",id,err)}
   case <-time.After(3*time.Second):faults<-fmt.Errorf("stream %d handler stuck",id)
   }
  }(stream)
 }
 time.Sleep(150*time.Millisecond)
 var probes []time.Duration
 for i:=0;i<24;i++ {
  d,e:=m5RoundTrip(cl,[]byte{byte(i),1,2,3,4,5,6,7},5*time.Second)
  if e!=nil {t.Errorf("short TCP probe %d: %v",i,e);break}
  probes=append(probes,d)
  time.Sleep(45*time.Millisecond)
 }
 wg.Wait()
 close(faults)
 for e:=range faults {t.Error(e)}
 for i:=0;i<streams;i++ {
  if completed[i].Load()!=frames {t.Errorf("stream %d progress %d/%d",i,completed[i].Load(),frames)}
 }
 if len(probes)!=24 {t.Fatalf("short probes completed=%d/24",len(probes))}
 sort.Slice(probes,func(i,j int)bool{return probes[i]<probes[j]})
 p50:=probes[len(probes)/2]
 p95:=probes[(len(probes)*95+99)/100-1]
 max:=probes[len(probes)-1]
 if p95 > time.Second {t.Errorf("loopback short-flow p95=%s exceeds 1s acceptance",p95)}
 var after runtime.MemStats
 runtime.ReadMemStats(&after)
 cm,sm:=cl.Metrics.Snapshot(),serverMetric.Snapshot()
 if cm.JitterUploadFrames==0 && sm.JitterDownloadFrames==0 {
  t.Error("no delayed Jitter frames observed in sustained loopback transport")
 }
 if cm.PaddingUploadFrames==0 && sm.PaddingDownloadFrames==0 {
  t.Error("no PAD frames observed in sustained loopback transport")
 }
 t.Logf("M5_LOOPBACK_RTT_PROBES=24 p50_ms=%.3f p95_ms=%.3f max_ms=%.3f",
  float64(p50)/float64(time.Millisecond),float64(p95)/float64(time.Millisecond),float64(max)/float64(time.Millisecond))
 t.Logf("M5_LOOPBACK_FAIRNESS=4/4 streams each %d frames; all 24 short probes complete; elapsed_ms=%d",frames,time.Since(start).Milliseconds())
 t.Logf("M5_LOOPBACK_ADAPTIVE=client_jitter_frames:%d server_jitter_frames:%d client_pad_frames:%d server_pad_frames:%d",
  cm.JitterUploadFrames,sm.JitterDownloadFrames,cm.PaddingUploadFrames,sm.PaddingDownloadFrames)
 t.Logf("M5_LOOPBACK_HEAP_ALLOC_BEFORE_AFTER=%d,%d bytes (not production RSS)",before.Alloc,after.Alloc)
 t.Log("M5_LIMITATION=LOOPBACK_APPLICATION_RTT_ONLY_NOT_HOMESERVER_FINSERVER_NETWORK_RTT_OR_WIRE_SCHEDULER_BOUND")
}
