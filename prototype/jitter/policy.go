// Package jitter implements an opt-in, bounded per-direction delay policy.
// It never reads application payloads and has no effect until wired into the
// authenticated Bridge send path. It does not implement traffic encryption.
package jitter

import (
    "context"
    "crypto/rand"
    "errors"
    "math/big"
    "time"
)

var ErrPolicy = errors.New("jitter: invalid policy")

type Policy struct {
    MinDuration time.Duration
    MinUsefulBytes uint64
    MinSamples uint64
    MaxInterSampleGap time.Duration
    MinFrameBytes uint64
    MaxDelay time.Duration
    MaxDelayPerWindow time.Duration
    Window time.Duration
}

// DefaultPolicy is deliberately conservative; the Bridge does not enable
// jitter merely because this package is linked into the executable.
func DefaultPolicy() Policy {
    return Policy{
        MinDuration: 2 * time.Second,
        MinUsefulBytes: 64 * 1024,
        MinSamples: 5,
        MaxInterSampleGap: time.Second,
        MinFrameBytes: 512,
        MaxDelay: 4 * time.Millisecond,
        MaxDelayPerWindow: 20 * time.Millisecond,
        Window: time.Second,
    }
}

func (p Policy) Validate() error {
    if p.MinDuration <= 0 || p.MinUsefulBytes == 0 || p.MinSamples < 2 ||
        p.MaxInterSampleGap <= 0 || p.MinFrameBytes == 0 || p.MinFrameBytes > 65536 ||
        p.MaxDelay <= 0 || p.MaxDelay > 20*time.Millisecond ||
        p.Window <= 0 || p.MaxDelayPerWindow < p.MaxDelay ||
        p.MaxDelayPerWindow > p.Window/4 {
        return ErrPolicy
    }
    return nil
}

// State belongs to exactly one stream direction. Never share it among
// concurrent writers. Independent streams cannot consume one another's
// delay budget; network-level fairness still requires integration tests.
type State struct {
    p Policy
    start, last time.Time
    samples, burstBytes uint64
    windowStart time.Time
    windowDelay time.Duration
    active bool
}

func New(p Policy) (*State, error) {
    if err := p.Validate(); err != nil { return nil, err }
    return &State{p: p}, nil
}

func (s *State) Active() bool { return s.active }

// Observe selects a delay for the next DATA frame. Interactive/small traffic,
// backlog, and idle bursts always bypass jitter. No delay is accumulated
// without a frame and no timer or background goroutine is created.
func (s *State) Observe(now time.Time, n uint64, backlog bool) (time.Duration, error) {
    if now.IsZero() || n < s.p.MinFrameBytes || backlog {
        s.resetBurst()
        return 0, nil
    }
    if !s.last.IsZero() && (now.Before(s.last) || now.Sub(s.last) > s.p.MaxInterSampleGap) {
        s.resetBurst()
    }
    s.last = now
    if s.start.IsZero() { s.start = now }
    s.samples++
    if s.burstBytes > ^uint64(0)-n { s.burstBytes = ^uint64(0) } else { s.burstBytes += n }
    if !s.active && s.samples >= s.p.MinSamples && now.Sub(s.start) >= s.p.MinDuration &&
        s.burstBytes >= s.p.MinUsefulBytes {
        s.active = true
    }
    if !s.active { return 0, nil }
    if s.windowStart.IsZero() || now.Before(s.windowStart) || now.Sub(s.windowStart) >= s.p.Window {
        s.windowStart = now
        s.windowDelay = 0
    }
    remaining := s.p.MaxDelayPerWindow-s.windowDelay
    if remaining <= 0 { return 0, nil }
    max := s.p.MaxDelay
    if remaining < max { max = remaining }
    // Crypto-grade entropy avoids a predictable per-stream timing pattern.
    // Both sides can be tested without transmitting random seeds.
    sample, err := rand.Int(rand.Reader, big.NewInt(int64(max)+1))
    if err != nil { return 0, err }
    delay := time.Duration(sample.Int64())
    s.windowDelay += delay
    return delay, nil
}

func (s *State) resetBurst() {
    s.start = time.Time{}
    s.last = time.Time{}
    s.samples = 0
    s.burstBytes = 0
    s.active = false
}

// Wait never holds a goroutine after the delay or cancellation. A zero delay
// checks cancellation but does not allocate a timer.
func Wait(ctx context.Context, d time.Duration) error {
    if err := ctx.Err(); err != nil { return err }
    if d <= 0 { return nil }
    timer := time.NewTimer(d)
    defer timer.Stop()
    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-timer.C:
        return ctx.Err()
    }
}
