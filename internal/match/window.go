// Package match binds each sample measurement to its nearest eligible blank
// (by time window) and the batch-wide instrument-drift model, producing the
// candidate correction relations that drive age computation.
package match

import (
	"math"
	"time"

	"task248-blankcorr/internal/model"
)

// WithinWindow reports whether the absolute time difference between a and b is
// at most window.
func WithinWindow(a, b time.Time, window time.Duration) bool {
	d := a.Sub(b)
	if d < 0 {
		d = -d
	}
	return d <= window
}

// NearestWithin returns the eligible measurement in candidates whose time is
// closest to target and within window, or nil if none qualifies. Measurements
// that are not eligible for matching (contaminated / excluded / expired) are
// skipped, which is exactly how removing a bad blank changes the match.
func NearestWithin(candidates []*model.Measurement, target time.Time, window time.Duration) *model.Measurement {
	var best *model.Measurement
	bestDelta := int64(math.MaxInt64)
	for _, c := range candidates {
		if !c.IsEligibleForMatch() {
			continue
		}
		delta := target.UnixMilli() - c.MeasuredAt.UnixMilli()
		if delta < 0 {
			delta = -delta
		}
		if window > 0 && delta > window.Milliseconds() {
			continue
		}
		if delta < bestDelta {
			bestDelta = delta
			best = c
		}
	}
	return best
}
