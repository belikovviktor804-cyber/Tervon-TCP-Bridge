package framing

import (
    "bytes"
    "context"
    "io"
    "testing"
    "time"

    "tervon-tcp-bridge/jitter"
    "tervon-tcp-bridge/padding"
)

// This deliberately uses independent encoders for video and short traffic.
// The synthetic clock proves a per-direction scheduling property and does
// not claim measured network RTT or actual 1Mbps delivery on Homeserver.
func TestM3VideoAndShortRequestsDoNotShareJitterBudget(t *testing.T) {
    rate := padding.DefaultRatePolicy()
    jit := jitter.DefaultPolicy()
    var videoWire, interactiveWire bytes.Buffer
    video, err := NewRatePaddingEncoder(&videoWire,rate)
    if err != nil { t.Fatal(err) }
    interactive, err := NewRatePaddingEncoder(&interactiveWire,rate)
    if err != nil { t.Fatal(err) }
    var videoDelays, interactiveDelays int
    if err:=video.EnableJitter(context.Background(),jit,func(d time.Duration){videoDelays++});err!=nil{t.Fatal(err)}
    if err:=interactive.EnableJitter(context.Background(),jit,func(d time.Duration){interactiveDelays++});err!=nil{t.Fatal(err)}
    clock:=time.Unix(50000,0)
    now:=func()time.Time { return clock }
    video.now=now
    interactive.now=now
    videoFrame:=bytes.Repeat([]byte{0x12,0x00,0xff,0x42,0x30},2500) // 12,500 bytes / 100 ms = 1Mbps
    smallFrame:=bytes.Repeat([]byte{0x01},128)
    for i:=0;i<40;i++ {
        if _,err:=video.Write(videoFrame);err!=nil{t.Fatal(err)}
        if _,err:=interactive.Write(smallFrame);err!=nil{t.Fatal(err)}
        clock=clock.Add(100*time.Millisecond)
    }
    if videoDelays==0 {t.Fatal("eligible video stream never jittered")}
    if interactiveDelays!=0 {t.Fatalf("interactive stream delayed %d times",interactiveDelays)}
    if video.PaddingBytes()==0 || interactive.PaddingBytes()!=0 {
        t.Fatalf("PAD crossed stream boundary: video=%d interactive=%d",
            video.PaddingBytes(), interactive.PaddingBytes())
    }
    if err:=video.CloseWrite();err!=nil{t.Fatal(err)}
    if err:=interactive.CloseWrite();err!=nil{t.Fatal(err)}
    got,err:=io.ReadAll(NewPaddingDecoder(&videoWire))
    if err!=nil || !bytes.Equal(got,bytes.Repeat(videoFrame,40)){
        t.Fatalf("video byte integrity: %d err=%v",len(got),err)
    }
    got,err=io.ReadAll(NewPaddingDecoder(&interactiveWire))
    if err!=nil || !bytes.Equal(got,bytes.Repeat(smallFrame,40)){
        t.Fatalf("interactive byte integrity: %d err=%v",len(got),err)
    }
}
