package jitter

import (
    "context"
    "errors"
    "testing"
    "time"
)

func TestM3FairnessAcrossIndependentSustainedStreams(t *testing.T) {
    p:=DefaultPolicy()
    p.MinDuration=time.Nanosecond
    p.MinUsefulBytes=1
    p.MinSamples=2
    base:=time.Unix(9000,0)
    const streams=32
    states:=make([]*State,streams)
    spent:=make([]time.Duration,streams)
    for i:=0;i<streams;i++ {states[i],_=New(p)}
    for tick:=0;tick<1000;tick++ {
        now:=base.Add(time.Duration(tick)*time.Millisecond)
        for i:=0;i<streams;i++{
            d,err:=states[i].Observe(now,1024,false)
            if err!=nil {t.Fatal(err)}
            if d<0 || d>p.MaxDelay {t.Fatalf("stream %d exceeded per-frame bound: %v",i,d)}
            spent[i]+=d
        }
    }
    for i:=0;i<streams;i++{
        if !states[i].Active() {t.Fatalf("stream %d starved by other streams",i)}
        if spent[i]>p.MaxDelayPerWindow {t.Fatalf("stream %d exceeded isolated window budget: %v",i,spent[i])}
    }
}

func TestM3BackpressureCancelsOptionalDelay(t *testing.T) {
    p:=DefaultPolicy()
    p.MinDuration=time.Nanosecond
    p.MinUsefulBytes=1
    p.MinSamples=2
    s,_:=New(p)
    now:=time.Unix(9010,0)
    for i:=0;i<10;i++{
        _,err:=s.Observe(now.Add(time.Duration(i)*time.Millisecond),1024,false)
        if err!=nil {t.Fatal(err)}
    }
    if !s.Active(){t.Fatal("expected sustained stream")}
    d,err:=s.Observe(now.Add(11*time.Millisecond),1024,true)
    if d!=0 || err!=nil || s.Active(){t.Fatal("jitter not bypassed under congestion")}
    ctx,cancel:=context.WithCancel(context.Background())
    cancel()
    if !errors.Is(Wait(ctx,time.Hour),context.Canceled){t.Fatal("wait failed to release on cancel")}
}

 
// TestM3RollingThirtySecondPerStreamBounds is a deterministic-clock
// multi-window regression: it validates a policy cap, not a real-network RTT.
func TestM3RollingThirtySecondPerStreamBounds(t *testing.T) {
    policy := DefaultPolicy()
    const streams = 8
    const seconds = 30
    const samplesPerSecond = 10
    base := time.Unix(9600, 0)
    states := make([]*State, streams)
    budgets := make([][]time.Duration, streams)
    for i := range states {
        var err error
        states[i], err = New(policy)
        if err != nil { t.Fatal(err) }
        budgets[i] = make([]time.Duration, seconds)
    }
    for tick := 0; tick < seconds*samplesPerSecond; tick++ {
        now := base.Add(time.Duration(tick)*100*time.Millisecond)
        for stream, state := range states {
            delay, err := state.Observe(now, 16*1024, false)
            if err != nil { t.Fatalf("stream=%d tick=%d: %v", stream, tick, err) }
            if delay < 0 || delay > policy.MaxDelay {
                t.Fatalf("stream=%d tick=%d delay=%v exceeds frame cap=%v",
                    stream, tick, delay, policy.MaxDelay)
            }
            sec := tick/samplesPerSecond
            budgets[stream][sec] += delay
            if budgets[stream][sec] > policy.MaxDelayPerWindow {
                t.Fatalf("stream=%d second=%d delay=%v exceeds window cap=%v",
                    stream, sec, budgets[stream][sec], policy.MaxDelayPerWindow)
            }
            // An interspersed control write must not consume the budget or delay.
            short, err := state.Observe(now.Add(time.Millisecond), 128, false)
            if err != nil || short != 0 {
                t.Fatalf("interactive frame delayed stream=%d tick=%d: %v %v",
                    stream, tick, short, err)
            }
        }
    }
    for i, state := range states {
        if !state.Active() {
            t.Fatalf("stream=%d lost sustained-flow eligibility", i)
        }
    }
}
