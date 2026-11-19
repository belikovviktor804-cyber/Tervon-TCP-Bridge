package jitter

import (
    "sync"
    "testing"
    "time"
)

// Exercises the exact M4 policy source copied into a disposable workdir.
// Virtual times are chosen to hit eligible, burst, idle and congested paths.
func TestM5SustainedWindowCaps(t *testing.T) {
    p := DefaultPolicy()
    s, err := New(p)
    if err != nil { t.Fatal(err) }
    base := time.Unix(1700000000, 0)
    var windowStart time.Time
    var windowTotal time.Duration
    var maxFrame time.Duration
    var maxWindow time.Duration
    var positive int
    for i := 0; i < 1200; i++ {
        now := base.Add(time.Duration(i) * 20 * time.Millisecond)
        d, e := s.Observe(now, 16*1024, false)
        if e != nil { t.Fatal(e) }
        if d < 0 || d > 4*time.Millisecond { t.Fatalf("frame %d delay %v exceeds 4ms",i,d) }
        if windowStart.IsZero() || now.Sub(windowStart) >= time.Second {
            if windowTotal > maxWindow { maxWindow = windowTotal }
            windowStart = now
            windowTotal = 0
        }
        windowTotal += d
        if windowTotal > 20*time.Millisecond {
            t.Fatalf("window delay %v exceeds 20ms at frame %d", windowTotal, i)
        }
        if d > maxFrame { maxFrame = d }
        if d > 0 { positive++ }
        if i < 100 && d != 0 { t.Fatal("delay before minimum 2s sustained eligibility") }
    }
    if windowTotal > maxWindow { maxWindow = windowTotal }
    if !s.Active() || positive < 100 {
        t.Fatalf("sustained adaptive jitter not exercised: active=%v positive=%d",s.Active(),positive)
    }
    t.Logf("M5_POLICY=PASS eligible_frames=%d max_frame_ns=%d max_window_ns=%d",positive,maxFrame,maxWindow)
}

func TestM5InteractiveBacklogAndIdleBypass(t *testing.T) {
    s,err:=New(DefaultPolicy())
    if err!=nil { t.Fatal(err) }
    base:=time.Unix(1700001000,0)
    for i:=0;i<110;i++ {
        d,e:=s.Observe(base.Add(time.Duration(i)*20*time.Millisecond),16*1024,false)
        if e!=nil {t.Fatal(e)}
        if i<100 && d!=0 {t.Fatal("premature jitter")}
    }
    if !s.Active() {t.Fatal("sustained stream failed to activate")}
    for _,n:=range []uint64{1,128,511} {
        d,e:=s.Observe(base.Add(2200*time.Millisecond),n,false)
        if e!=nil || d!=0 {t.Fatal("interactive traffic delayed")}
    }
    d,e:=s.Observe(base.Add(2220*time.Millisecond),16*1024,true)
    if e!=nil || d!=0 || s.Active() {t.Fatal("backlog failed to disable jitter")}
    d,e=s.Observe(base.Add(2240*time.Millisecond),16*1024,false)
    if e!=nil || d!=0 {t.Fatal("jitter reactivated immediately after congestion")}
    d,e=s.Observe(base.Add(10*time.Second),16*1024,false)
    if e!=nil || d!=0 || s.Active() {t.Fatal("idle reset failed")}
    t.Log("M5_INTERACTIVE_BACKLOG_IDLE=PASS")
}

func TestM5CompetingStreamsHaveIndependentBudgets(t *testing.T) {
    const streams=8
    var wg sync.WaitGroup
    failures:=make(chan string, streams)
    base:=time.Unix(1700002000,0)
    for stream:=0;stream<streams;stream++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            p:=DefaultPolicy()
            s,e:=New(p)
            if e!=nil { failures<-"create";return }
            var last=time.Duration(0)
            var count int
            for i:=0;i<400;i++ {
                now:=base.Add(time.Duration(i)*25*time.Millisecond)
                d,err:=s.Observe(now,8192,false)
                if err!=nil || d<0 || d>p.MaxDelay {failures<-"frame";return}
                if d>0 {count++}
                if now.Sub(base)>=3*time.Second && count==0 {failures<-"starvation";return}
                last=d
            }
            _=last
            if !s.Active() {failures<-"inactive"}
        }()
    }
    wg.Wait()
    close(failures)
    for f:=range failures {t.Errorf("independent jitter stream failed: %s",f)}
    t.Log("M5_INDEPENDENT_STREAM_POLICIES=PASS (does not prove network fairness)")
}
