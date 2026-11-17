package padding

import (
    "math"
    "testing"
    "time"
)

func testPolicy() Policy {
    return Policy{
        MinDuration: 2*time.Second, MinUsefulBytes: 64*1024,
        MinSamples: 5, MaxInterSampleGap: time.Second,
        RatioPercent: 5, MaxPaddingPerFrame: 512, MinFrameBytes: 512,
    }
}

func TestNoPaddingForShortOrInteractiveStreams(t *testing.T) {
    s,err:=New(testPolicy()); if err!=nil {t.Fatal(err)}
    now:=time.Unix(100,0)
    for i:=0;i<4;i++ {
        if got:=s.Observe(now.Add(time.Duration(i)*100*time.Millisecond),4096);got!=0 {
            t.Fatalf("short flow padding %d at %d",got,i)
        }
    }
    if s.Active(){t.Fatal("short flow activated")}
    // An idle gap must reset the classifier.
    if got:=s.Observe(now.Add(5*time.Second),1024);got!=0 || s.Active(){
        t.Fatalf("idle gap: padding=%d active=%v",got,s.Active())
    }
    if got:=s.Observe(now.Add(5100*time.Millisecond),64);got!=0{
        t.Fatalf("interactive frame got padding=%d",got)
    }
}

func TestStableFlowActivatesAndBudgetIsBounded(t *testing.T) {
    s,err:=New(testPolicy());if err!=nil {t.Fatal(err)}
    now:=time.Unix(100,0)
    var total uint64
    for i:=0;i<100;i++{
        n:=s.Observe(now.Add(time.Duration(i)*500*time.Millisecond),16*1024)
        if i<4 && n!=0 {t.Fatal("premature activation")}
        if n>testPolicy().MaxPaddingPerFrame {t.Fatal("frame cap exceeded")}
        total+=n
        if total!=s.PaddingBytes(){t.Fatal("accounting mismatch")}
        if total > s.UsefulBytes()*testPolicy().RatioPercent/100 {t.Fatal("payload budget exceeded")}
        if s.PaddingWireBytes() > s.UsefulBytes()*testPolicy().RatioPercent/100 {t.Fatal("wire budget exceeded")}
    }
    if !s.Active() || total==0 {t.Fatalf("stable flow active=%v bytes=%d",s.Active(),total)}
}

func TestDirectionalStateIsIndependent(t *testing.T) {
    up,_:=New(testPolicy());down,_:=New(testPolicy())
    now:=time.Unix(100,0)
    for i:=0;i<10;i++{up.Observe(now.Add(time.Duration(i)*500*time.Millisecond),16*1024)}
    if !up.Active() || down.Active() || down.PaddingBytes()!=0 {t.Fatal("directions coupled")}
}

func TestGapDeactivatesWithoutResettingOverhead(t *testing.T){
    s,_:=New(testPolicy())
    now:=time.Unix(100,0)
    for i:=0;i<10;i++{s.Observe(now.Add(time.Duration(i)*500*time.Millisecond),16*1024)}
    prior:=s.PaddingBytes()
    if prior==0 {t.Fatal("not activated")}
    if n:=s.Observe(now.Add(20*time.Second),16*1024);n!=0 || s.Active() {t.Fatal("idle gap did not disable")}
    if s.PaddingBytes()!=prior {t.Fatal("budget accounting reset")}
}

func TestInvalidPolicyAndOverflow(t *testing.T){
    p:=testPolicy()
    p.RatioPercent=100
    if _,err:=New(p);err!=ErrConfig {t.Fatalf("invalid policy: %v",err)}
    p=testPolicy()
    s,_:=New(p)
    now:=time.Unix(100,0)
    for i:=0;i<8;i++{s.Observe(now.Add(time.Duration(i)*500*time.Millisecond),math.MaxUint64/4)}
    if s.PaddingWireBytes()>fractionOf(s.UsefulBytes(),5,100) {t.Fatal("overflow violated budget")}
    if saturatingAdd(math.MaxUint64,1)!=math.MaxUint64 {t.Fatal("saturating overflow")}
}

func TestNewShortBurstAfterLongFlowStaysOff(t *testing.T){
    p:=testPolicy()
    s,err:=New(p);if err!=nil{t.Fatal(err)}
    now:=time.Unix(1234,0)
    for i:=0;i<10;i++{
        s.Observe(now.Add(time.Duration(i)*500*time.Millisecond),16*1024)
    }
    if !s.Active(){t.Fatal("initial sustained flow not active")}
    // Pause resets the current sustained-flow classification, but never
    // resets aggregate useful bytes or spent overhead budget.
    after:=now.Add(20*time.Second)
    if n:=s.Observe(after,100);n!=0 {t.Fatal("padding immediately after gap")}
    for i:=1;i<20;i++{
        n:=s.Observe(after.Add(time.Duration(i)*200*time.Millisecond),100)
        if n!=0 || s.Active(){t.Fatalf("new short burst activated at sample %d",i)}
    }
}

func TestBudgetIncludesPaddingFrameHeader(t *testing.T){
    p:=testPolicy()
    p.MinDuration=time.Nanosecond
    p.MinSamples=2
    p.MinUsefulBytes=1
    p.RatioPercent=1
    p.MaxPaddingPerFrame=4096
    s,err:=New(p);if err!=nil{t.Fatal(err)}
    start:=time.Unix(100,0)
    var totalPayload uint64
    for i:=0;i<100;i++{
        totalPayload+=s.Observe(start.Add(time.Duration(i)*time.Millisecond),512)
        if s.PaddingWireBytes()>fractionOf(s.UsefulBytes(),p.RatioPercent,100){
            t.Fatalf("wire budget violated: payload=%d wire=%d useful=%d",
                s.PaddingBytes(),s.PaddingWireBytes(),s.UsefulBytes())
        }
    }
    if totalPayload==0{t.Fatal("no PAD emitted")}
    if s.PaddingWireBytes()<=s.PaddingBytes(){t.Fatal("PAD header omitted from budget")}
}
