// Package padding implements a bounded, content-blind activity detector and
// padding budget. It does not modify frames or TCP payloads.
package padding

import (
    "errors"
    "math"
    "time"
)

var ErrConfig = errors.New("padding: invalid policy")

// FrameOverheadBytes is the fixed TBF1 header size for a PAD frame.
// A regression test in framing must keep this aligned with HeaderSize.
const FrameOverheadBytes uint64 = 10

// Policy applies independently to each stream direction.
// RatioPercent is a hard cumulative budget (padding / useful bytes).
type Policy struct {
    MinDuration time.Duration
    MinUsefulBytes uint64
    MinSamples uint64
    MaxInterSampleGap time.Duration
    RatioPercent uint64
    MaxPaddingPerFrame uint64
    MinFrameBytes uint64
}

// DefaultPolicy is intentionally conservative and is not enabled by default.
// Values are provisional pending M2 performance measurements.
func DefaultPolicy() Policy {
    return Policy{
        MinDuration: 2*time.Second,
        MinUsefulBytes: 64*1024,
        MinSamples: 5,
        MaxInterSampleGap: time.Second,
        RatioPercent: 5,
        MaxPaddingPerFrame: 512,
        MinFrameBytes: 512,
    }
}

func (p Policy) Validate() error {
    if p.MinDuration <= 0 || p.MinUsefulBytes == 0 || p.MinSamples < 2 ||
        p.MaxInterSampleGap <= 0 || p.RatioPercent == 0 || p.RatioPercent > 25 ||
        p.MaxPaddingPerFrame == 0 || p.MaxPaddingPerFrame > 4096 ||
        p.MinFrameBytes == 0 || p.MinFrameBytes > 65536 {
        return ErrConfig
    }
    return nil
}

// State is a per-direction stream state. It is owned by a single writer;
// callers must synchronize if they share a State across goroutines.
type State struct {
    policy Policy
    started time.Time
    last time.Time
    samples uint64
    useful uint64
    streakUseful uint64
    padded uint64
    paddedWire uint64
    active bool
}

func New(p Policy) (*State, error) {
    if err := p.Validate(); err != nil { return nil, err }
    return &State{policy:p}, nil
}

func (s *State) Active() bool { return s.active }
func (s *State) UsefulBytes() uint64 { return s.useful }
func (s *State) PaddingBytes() uint64 { return s.padded }
func (s *State) PaddingWireBytes() uint64 { return s.paddedWire }

// ObserveSmallFrame is the fast path for a frame too small to justify
// padding. It accounts useful bytes but deliberately resets the sustained
// burst classifier: a stream of interactive small frames must remain OFF.
// It returns false for eligible sizes so callers can use Observe(at, n).
func (s *State) ObserveSmallFrame(n uint64) bool {
    if n == 0 || n >= s.policy.MinFrameBytes { return false }
    s.useful = saturatingAdd(s.useful, n)
    s.active = false
    s.started = time.Time{}
    s.last = time.Time{}
    s.samples = 0
    s.streakUseful = 0
    return true
}

// Observe returns a permitted count of padding bytes, but NEVER writes them.
// No padding is permitted before a stable sequence of substantial samples.
// Long inactivity resets detection; the cumulative overhead budget is not reset.
func (s *State) Observe(at time.Time, usefulFrameBytes uint64) uint64 {
    if usefulFrameBytes == 0 { return 0 }
    if s.ObserveSmallFrame(usefulFrameBytes) { return 0 }
    if at.IsZero() { return 0 }
    if !s.last.IsZero() && (at.Before(s.last) || at.Sub(s.last) > s.policy.MaxInterSampleGap) {
        s.active = false
        s.started = time.Time{}
        s.samples = 0
        s.streakUseful = 0
    }
    s.last = at
    if s.started.IsZero() { s.started = at }
    if s.samples < math.MaxUint64 { s.samples++ }
    s.useful = saturatingAdd(s.useful, usefulFrameBytes)
    s.streakUseful = saturatingAdd(s.streakUseful, usefulFrameBytes)
    // Budget uses a capped ratio and saturation-safe integer arithmetic.
    if !s.active && s.samples >= s.policy.MinSamples &&
        !at.Before(s.started) && at.Sub(s.started) >= s.policy.MinDuration &&
        s.streakUseful >= s.policy.MinUsefulBytes {
        s.active = true
    }
    if !s.active { return 0 }
    budget := fractionOf(s.useful, s.policy.RatioPercent, 100)
    if budget <= s.paddedWire { return 0 }
    remaining := budget - s.paddedWire
    if remaining <= FrameOverheadBytes { return 0 }
    remaining -= FrameOverheadBytes
    desired := usefulFrameBytes / 16 // 6.25% target, bounded by the budget.
    if desired > s.policy.MaxPaddingPerFrame { desired = s.policy.MaxPaddingPerFrame }
    if desired > remaining { desired = remaining }
    if desired == 0 { return 0 }
    s.padded += desired
    s.paddedWire += desired + FrameOverheadBytes // <= budget <= MaxUint64
    return desired
}

func saturatingAdd(a, b uint64) uint64 {
    if math.MaxUint64-a < b { return math.MaxUint64 }
    return a+b
}

func fractionOf(value, numerator, denominator uint64) uint64 {
    q, r := value/denominator, value%denominator
    whole := q*numerator
    // numerator <= 25 and q <= MaxUint64/100; whole cannot overflow.
    return whole + (r*numerator)/denominator
}
