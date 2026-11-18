package padding

import (
    "errors"
    "math"
    "math/bits"
    "time"
)

// RatePolicy is the M3 opt-in sustained-throughput detector. It is not used
// by existing M2 configurations, so rolling development cannot change a
// currently running transport.
type RatePolicy struct {
    MinRateBitsPerSecond uint64
    ExitRateBitsPerSecond uint64
    Window time.Duration
    MaxInterSampleGap time.Duration
    MinSamples uint64
    RatioPercent uint64
    MaxPaddingPerFrame uint64
}

var ErrRatePolicy = errors.New("padding: invalid rate policy")

func DefaultRatePolicy() RatePolicy {
    return RatePolicy{
        MinRateBitsPerSecond: 1_000_000,
        ExitRateBitsPerSecond: 700_000,
        Window: time.Second,
        MaxInterSampleGap: time.Second,
        MinSamples: 5,
        RatioPercent: 5,
        MaxPaddingPerFrame: 512,
    }
}

func (p RatePolicy) Validate() error {
    if p.MinRateBitsPerSecond == 0 || p.ExitRateBitsPerSecond == 0 ||
        p.ExitRateBitsPerSecond >= p.MinRateBitsPerSecond ||
        p.Window < 100*time.Millisecond || p.Window > 10*time.Second ||
        p.MaxInterSampleGap <= 0 || p.MaxInterSampleGap > p.Window ||
        p.MinSamples < 2 || p.RatioPercent == 0 || p.RatioPercent > 25 ||
        p.MaxPaddingPerFrame == 0 || p.MaxPaddingPerFrame > 4096 {
        return ErrRatePolicy
    }
    return nil
}

// RateState belongs to a single encoder and a single direction. It never
// retains payloads or IDs. A closed interval must reach the activation rate
// twice consecutively; small TCP writes contribute normally. No background
// timer is created and long idle intervals reset qualification.
type RateState struct {
    p RatePolicy
    last time.Time
    windowStart time.Time
    windowBytes uint64
    windowSamples uint64
    consecutive uint8
    active bool
    useful uint64
    padded uint64
    paddedWire uint64
}

func NewRateState(p RatePolicy) (*RateState, error) {
    if err := p.Validate(); err != nil { return nil, err }
    return &RateState{p:p}, nil
}

func (s *RateState) Active() bool { return s.active }
func (s *RateState) UsefulBytes() uint64 { return s.useful }
func (s *RateState) PaddingBytes() uint64 { return s.padded }
func (s *RateState) PaddingWireBytes() uint64 { return s.paddedWire }

func (s *RateState) resetClassification() {
    s.active = false
    s.windowStart = time.Time{}
    s.windowBytes = 0
    s.windowSamples = 0
    s.consecutive = 0
}

func thresholdBytes(rate uint64, window time.Duration) uint64 {
    // Exact 128-bit multiplication avoids overflow for extreme yet valid
    // policies. Saturation is conservative: never activate on an overflow.
    const bitsPerByteSecond = uint64(8_000_000_000)
    hi, lo := bits.Mul64(rate, uint64(window))
    if hi >= bitsPerByteSecond { return math.MaxUint64 }
    q, _ := bits.Div64(hi, lo, bitsPerByteSecond)
    return q
}

// Observe returns the permissible PAD payload bytes after DATA has been sent.
// It does not write, sleep or inspect application data. A valid high-rate
// stream is classified by completed one-second windows, NOT by TCP framing.
// The first observation after a long gap is never padded.
func (s *RateState) Observe(at time.Time, bytes uint64) uint64 {
    if bytes == 0 || at.IsZero() { return 0 }
    if !s.last.IsZero() &&
        (at.Before(s.last) || at.Sub(s.last) > s.p.MaxInterSampleGap) {
        s.resetClassification()
    }
    s.last = at
    if s.windowStart.IsZero() { s.windowStart = at }
    s.useful = saturatingAdd(s.useful, bytes)

    elapsed := at.Sub(s.windowStart)
    if elapsed >= s.p.Window {
        // Require consecutive completed intervals: prevents a transient
        // burst from switching on Padding. Gaps over one entire window
        // never qualify even if bytes were buffered before the pause.
        if elapsed > 2*s.p.Window {
            s.resetClassification()
            s.windowStart = at
        } else {
            startBytes := thresholdBytes(s.p.MinRateBitsPerSecond, s.p.Window)
            keepBytes := thresholdBytes(s.p.ExitRateBitsPerSecond, s.p.Window)
            if s.active {
                if s.windowBytes < keepBytes { s.active = false; s.consecutive = 0 }
            } else if s.windowBytes >= startBytes && s.windowSamples >= s.p.MinSamples {
                if s.consecutive < 2 { s.consecutive++ }
            } else {
                s.consecutive = 0
            }
            if !s.active && s.consecutive >= 2 { s.active = true }
            s.windowStart = at
        }
        s.windowBytes = 0
        s.windowSamples = 0
    }
    s.windowBytes = saturatingAdd(s.windowBytes, bytes)
    if s.windowSamples < math.MaxUint64 { s.windowSamples++ }

    if !s.active { return 0 }
    budget := fractionOf(s.useful, s.p.RatioPercent, 100)
    if budget <= s.paddedWire || budget-s.paddedWire <= FrameOverheadBytes { return 0 }
    remaining := budget-s.paddedWire-FrameOverheadBytes
    desired := bytes/16
    if desired > s.p.MaxPaddingPerFrame { desired = s.p.MaxPaddingPerFrame }
    if desired > remaining { desired = remaining }
    if desired == 0 { return 0 }
    s.padded = saturatingAdd(s.padded, desired)
    s.paddedWire = saturatingAdd(s.paddedWire, desired+FrameOverheadBytes)
    return desired
}
