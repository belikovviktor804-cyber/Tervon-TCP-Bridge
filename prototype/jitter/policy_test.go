package jitter

import (
    "context"
    "errors"
    "testing"
    "time"
)

func TestDefaultPolicyValidation(t *testing.T) {
    if err := DefaultPolicy().Validate(); err != nil { t.Fatal(err) }
    p := DefaultPolicy()
    p.MaxDelay = 40 * time.Millisecond
    if !errors.Is(p.Validate(), ErrPolicy) { t.Fatal("unbounded delay accepted") }
    p = DefaultPolicy()
    p.MaxDelayPerWindow = 100 * time.Millisecond
    if !errors.Is(p.Validate(), ErrPolicy) { t.Fatal("oversized window allowance") }
}

func TestShortAndInteractiveBypass(t *testing.T) {
    s, err := New(DefaultPolicy())
    if err != nil { t.Fatal(err) }
    start := time.Unix(10, 0)
    for i := 0; i < 200; i++ {
        d, err := s.Observe(start.Add(time.Duration(i)*100*time.Millisecond), 128, false)
        if err != nil || d != 0 || s.Active() { t.Fatalf("interactive jitter %v, %v", d, err) }
    }
    for i := 0; i < 4; i++ {
        d, err := s.Observe(start.Add(30*time.Second+time.Duration(i)*300*time.Millisecond), 16*1024, false)
        if err != nil || d != 0 { t.Fatalf("premature jitter: %v, %v", d, err) }
    }
}

func TestSustainedFlowBoundedWindowAndBacklog(t *testing.T) {
    p := DefaultPolicy()
    s, _ := New(p)
    start := time.Unix(30, 0)
    var spent time.Duration
    for i := 0; i < 20; i++ {
        // Sub-second samples activate after 2 seconds.
        d, err := s.Observe(start.Add(time.Duration(i)*200*time.Millisecond), 16*1024, false)
        if err != nil { t.Fatal(err) }
        if d < 0 || d > p.MaxDelay { t.Fatalf("delay outside per-frame cap: %v", d) }
        if i < 10 && d != 0 { t.Fatalf("delay before eligible duration: %v", d) }
        if i >= 10 { spent += d }
        if i == 14 && spent > 2*p.MaxDelayPerWindow {
            t.Fatal("unexpected cumulative delay")
        }
    }
    if !s.Active() { t.Fatal("sustained flow never activated") }
    d, err := s.Observe(start.Add(4100*time.Millisecond), 16*1024, true)
    if err != nil || d != 0 || s.Active() { t.Fatal("backpressure failed to bypass") }
    d, err = s.Observe(start.Add(4200*time.Millisecond), 16*1024, false)
    if err != nil || d != 0 { t.Fatal("failed to reset after congestion") }
}

func TestWindowBudgetNeverExceeded(t *testing.T) {
    p := DefaultPolicy()
    p.MinDuration = time.Nanosecond
    p.MinUsefulBytes = 1
    p.MinSamples = 2
    s, _ := New(p)
    base := time.Unix(100, 0)
    var spent time.Duration
    for i := 0; i < 1000; i++ {
        d, err := s.Observe(base.Add(time.Duration(i)*time.Microsecond), 4096, false)
        if err != nil { t.Fatal(err) }
        spent += d
        if spent > p.MaxDelayPerWindow {
            t.Fatalf("window delay %v exceeds cap %v", spent, p.MaxDelayPerWindow)
        }
    }
}

func TestIdleResetAndIndependentStreams(t *testing.T) {
    p := DefaultPolicy()
    p.MinDuration = time.Nanosecond
    p.MinUsefulBytes = 1
    p.MinSamples = 2
    a, _ := New(p)
    b, _ := New(p)
    base := time.Unix(500, 0)
    _, _ = a.Observe(base, 4096, false)
    _, _ = a.Observe(base.Add(time.Millisecond), 4096, false)
    if !a.Active() || b.Active() { t.Fatal("states coupled") }
    d, err := a.Observe(base.Add(5*time.Second), 4096, false)
    if err != nil || d != 0 || a.Active() { t.Fatal("gap did not reset") }
}

func TestWaitCancellationAndNoExtraTimerForZero(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    cancel()
    if !errors.Is(Wait(ctx, 0), context.Canceled) { t.Fatal("zero wait ignored cancel") }
    if !errors.Is(Wait(ctx, time.Hour), context.Canceled) { t.Fatal("canceled wait blocked") }
    if err := Wait(context.Background(), 0); err != nil { t.Fatal(err) }
    ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Millisecond)
    defer cancel2()
    start := time.Now()
    if !errors.Is(Wait(ctx2, time.Hour), context.DeadlineExceeded) { t.Fatal("deadline not enforced") }
    if time.Since(start) > time.Second { t.Fatal("cancelable wait exceeded bound") }
}
