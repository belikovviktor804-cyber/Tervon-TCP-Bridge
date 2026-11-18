package padding

import (
    "testing"
    "time"
)

func TestRateFourMbpsFragmentedUploadActivates(t *testing.T) {
    p:=DefaultRatePolicy()
    s,err:=NewRateState(p)
    if err!=nil {t.Fatal(err)}
    base:=time.Unix(2000,0)
    var pad uint64
    // 4Mbit/s, 500 kB/s: mix large and 128-byte TCP fragments.
    for i:=0;i<300;i++ {
        at:=base.Add(time.Duration(i)*10*time.Millisecond)
        pad+=s.Observe(at,4872)
        pad+=s.Observe(at.Add(time.Microsecond),128)
        if i < 200 && s.Active() {t.Fatalf("activated before two full seconds at %d",i)}
    }
    if !s.Active() || pad==0 {t.Fatalf("no PAD at 4 Mbps: active=%v PAD=%d",s.Active(),pad)}
    if s.PaddingWireBytes()>s.UsefulBytes()*p.RatioPercent/100 {t.Fatal("budget exceeded")}
}

func TestRateOneMbpsFragmentedVideoActivates(t *testing.T) {
    s,_:=NewRateState(DefaultRatePolicy())
    base:=time.Unix(3000,0)
    var pad uint64
    // 1 Mbps exactly: 125 bytes/ms, with no assumption about TCP packet boundaries.
    for i:=0;i<300;i++{
        at:=base.Add(time.Duration(i)*10*time.Millisecond)
        pad+=s.Observe(at,1122)
        pad+=s.Observe(at.Add(time.Microsecond),128)
    }
    if !s.Active() || pad==0 {t.Fatalf("1 Mbps failed to activate: active=%v PAD=%d",s.Active(),pad)}
}

func TestRateBelowThresholdDoesNotActivate(t *testing.T){
    s,_:=NewRateState(DefaultRatePolicy())
    at:=time.Unix(4000,0)
    for i:=0;i<1000;i++{
        if n:=s.Observe(at.Add(time.Duration(i)*10*time.Millisecond),625); n!=0 {
            t.Fatalf("0.5Mbps emitted PAD at %d: %d",i,n)
        }
    }
    if s.Active(){t.Fatal("slow interactive stream activated")}
}

func TestRateShortBurstDoesNotActivate(t *testing.T){
    s,_:=NewRateState(DefaultRatePolicy())
    at:=time.Unix(5000,0)
    for i:=0;i<50;i++{
        s.Observe(at.Add(time.Duration(i)*10*time.Millisecond),65536)
    }
    if s.Active() || s.PaddingBytes()!=0 {t.Fatal("short burst activated")}
}

func TestRateHysteresisAndIdleReset(t *testing.T){
    p:=DefaultRatePolicy()
    s,_:=NewRateState(p)
    at:=time.Unix(6000,0)
    for i:=0;i<310;i++{
        s.Observe(at.Add(time.Duration(i)*10*time.Millisecond),1250)
    }
    if !s.Active(){t.Fatal("not active at 1Mbps")}
    // 0.8 Mbps between the 1Mbps entry and 0.7Mbps exit rate.
    for i:=310;i<510;i++{
        s.Observe(at.Add(time.Duration(i)*10*time.Millisecond),1000)
    }
    if !s.Active(){t.Fatal("hysteresis unexpectedly disabled")}
    prior:=s.PaddingWireBytes()
    // Go well below exit threshold for at least a full window.
    for i:=510;i<720;i++{
        s.Observe(at.Add(time.Duration(i)*10*time.Millisecond),300)
    }
    if s.Active(){t.Fatal("failed to deactivate below 0.7Mbps")}
    if s.PaddingWireBytes()<prior {t.Fatal("overhead accounting reset")}
    s.Observe(at.Add(20*time.Second),1250)
    if s.Active(){t.Fatal("active after idle gap")}
}

func TestRateDirectionsIndependentAndBudget(t *testing.T){
    p:=DefaultRatePolicy()
    up,_:=NewRateState(p)
    down,_:=NewRateState(p)
    at:=time.Unix(7000,0)
    for i:=0;i<500;i++{
        up.Observe(at.Add(time.Duration(i)*10*time.Millisecond),2500)
        if up.PaddingWireBytes()>fractionOf(up.UsefulBytes(),p.RatioPercent,100){
            t.Fatal("wire budget exceeded")
        }
    }
    if !up.Active() || down.Active() || down.PaddingBytes()!=0 {t.Fatal("streams coupled")}
}

func TestRateRejectBadPolicy(t *testing.T){
    p:=DefaultRatePolicy()
    if err:=p.Validate();err!=nil {t.Fatal(err)}
    p.ExitRateBitsPerSecond=p.MinRateBitsPerSecond
    if _,err:=NewRateState(p);err!=ErrRatePolicy {t.Fatalf("invalid hysteresis allowed %v",err)}
}
