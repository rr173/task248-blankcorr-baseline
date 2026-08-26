package correct

import (
	"fmt"
	"math"
	"time"

	"task248-blankcorr/internal/model"
)

// DriftModel is a weighted linear fit of the standard recovery ratio
// r(t) = a + b·t, where t is the measurement time. Recovery is defined as the
// certified ratio divided by the measured ratio; a drift-free instrument has
// r ≈ 1 everywhere.
//
// To stay numerically stable under ordinary Unix timestamps (millisecond
// magnitudes ~1.7e12) the fit is performed in centered, dimensionless time
// coordinates u = (t - T0)/Tau, where T0 is the weighted mean time and Tau the
// weighted time standard deviation of the training standards. In u-space the
// design matrix is well scaled (u ~ O(1)) regardless of the absolute timestamp
// magnitude or the time span, so the catastrophic cancellation that would
// otherwise occur in denom = S·Stt − St² (two ~1e25 terms whose difference is
// tiny when standards are near in time) is avoided. A and B are the intercept
// (at t = T0, i.e. u = 0) and slope per unit u; At converts the query time back
// to u before evaluating the line.
type DriftModel struct {
	A, B       float64 // intercept at T0 and slope per unit u
	VarA, VarB float64 // variances of A and B (in u-space)
	CovAB      float64 // covariance of A and B
	T0         float64 // weighted mean time (ms since epoch), the centering origin
	Tau        float64 // weighted time std-dev (ms), the scaling factor; 0 for a constant model
	MeanT      float64 // mean time (for reporting) == T0
	NTrained   int
}

// Recovery computes the recovery ratio r = certified/measured and its 1-sigma
// uncertainty from the ratio uncertainty (the certified value is treated as
// exact). It is the dependent variable of the drift fit.
func Recovery(m *model.Measurement) (r, sigR float64, err error) {
	if m.Kind != model.KindStandard {
		return 0, 0, fmt.Errorf("recovery requires a standard, got %q", m.Kind)
	}
	if m.Ratio <= 0 {
		return 0, 0, fmt.Errorf("standard measured ratio must be > 0")
	}
	if m.CertifiedRatio <= 0 {
		return 0, 0, fmt.Errorf("standard certified ratio must be > 0")
	}
	if math.IsNaN(m.Ratio) || math.IsInf(m.Ratio, 0) || math.IsNaN(m.RatioUnc) || math.IsInf(m.RatioUnc, 0) ||
		math.IsNaN(m.CertifiedRatio) || math.IsInf(m.CertifiedRatio, 0) {
		return 0, 0, fmt.Errorf("standard values must be finite")
	}
	r = m.CertifiedRatio / m.Ratio
	// σ_r = r · (σ_ratio / ratio)  (certified exact)
	sigR = r * (m.RatioUnc / m.Ratio)
	return r, sigR, nil
}

// FitDrift builds a weighted least-squares drift model from eligible
// standards. Weights are 1/σ_r². At least two standards are required to fit a
// line; with a single standard the model degenerates to a constant r.
func FitDrift(standards []*model.Measurement) (*DriftModel, error) {
	eligible := make([]*model.Measurement, 0, len(standards))
	for _, s := range standards {
		if s.IsEligibleForMatch() {
			eligible = append(eligible, s)
		}
	}
	if len(eligible) == 0 {
		return nil, fmt.Errorf("no eligible standards to fit drift")
	}
	if len(eligible) == 1 {
		s := eligible[0]
		r, sigR, err := Recovery(s)
		if err != nil {
			return nil, err
		}
		return &DriftModel{A: r, B: 0, VarA: sigR * sigR, VarB: 0, CovAB: 0,
			T0: float64(s.MeasuredAt.UnixMilli()), Tau: 0,
			MeanT: float64(s.MeasuredAt.UnixMilli()), NTrained: 1}, nil
	}
	// Recoveries, weights and raw timestamps.
	type pt struct{ t, r, w float64 }
	pts := make([]pt, 0, len(eligible))
	var S, St float64 // weighted sum and weighted time sum (raw ms)
	for _, s := range eligible {
		r, sigR, err := Recovery(s)
		if err != nil {
			return nil, err
		}
		w := 1.0 / (sigR * sigR)
		if sigR == 0 {
			w = 1e24
		}
		t := float64(s.MeasuredAt.UnixMilli())
		pts = append(pts, pt{t: t, r: r, w: w})
		S += w
		St += w * t
	}
	t0 := St / S // weighted mean time (ms)

	// Weighted time variance: Σw(t-t0)² / Σw. Its sqrt is the natural time
	// scale Tau. This is computed in a single centered pass (no subtraction of
	// two ~1e25 raw moments), so it is exact even when standards are near in
	// time and t itself is ~1.7e12.
	var swtt float64
	for _, p := range pts {
		dt := p.t - t0
		swtt += p.w * dt * dt
	}
	tau2 := swtt / S
	if tau2 <= 0 {
		// All timestamps identical: the line is degenerate, fall back to the
		// weighted mean recovery (slope 0). This is the same constant model as
		// the single-standard case and is the only sensible answer.
		var sr float64
		for _, p := range pts {
			sr += p.w * p.r
		}
		meanR := sr / S
		var varA, sumW float64
		for _, p := range pts {
			d := p.r - meanR
			varA += p.w * d * d
			sumW += p.w
		}
		if sumW > 0 {
			varA /= sumW
		}
		return &DriftModel{A: meanR, B: 0, VarA: varA, VarB: 0, CovAB: 0,
			T0: t0, Tau: 0, MeanT: t0, NTrained: len(eligible)}, nil
	}
	tau := math.Sqrt(tau2)

	// Weighted least squares in centered, scaled time u = (t - t0)/tau. In these
	// coordinates u ~ O(1) regardless of timestamp magnitude or time span, so
	// the normal-equation denominator S·Suu2 − Su² no longer subtracts two
	// ~1e25 raw-moment terms — it is well scaled and exact even for near-in-time
	// standards.
	var S2, Su, Suu2, Sr, Sur float64
	for _, p := range pts {
		u := (p.t - t0) / tau
		S2 += p.w
		Su += p.w * u
		Suu2 += p.w * u * u
		Sur += p.w * u * p.r
		Sr += p.w * p.r
	}
	denom := S2*Suu2 - Su*Su
	if denom == 0 {
		return nil, fmt.Errorf("degenerate standard time spread; cannot fit drift")
	}
	// In centered coordinates Σw·u is ~0 by construction, so denom ≈ S·Suu2,
	// but we keep the exact normal-equation form for correctness. The slope B
	// is per unit u; the intercept A is the recovery at t = t0 (u = 0).
	a := (Sr*Suu2 - Su*Sur) / denom
	b := (S2*Sur - Su*Sr) / denom
	varA := Suu2 / denom
	varB := S2 / denom
	covAB := -Su / denom
	return &DriftModel{A: a, B: b, VarA: varA, VarB: varB, CovAB: covAB,
		T0: t0, Tau: tau, MeanT: t0, NTrained: len(eligible)}, nil
}

// At evaluates the fitted recovery ratio at time t and returns the value and
// its prediction uncertainty. The query timestamp is first mapped to the
// centered, scaled coordinate u = (t - T0)/Tau used during fitting, so the
// arithmetic stays in the same well-conditioned space:
//
//	r̂(t) = A + B·u(t)
//	Var(r̂) = VarA + 2·u·CovAB + u²·VarB
//
// Tau == 0 marks a constant (zero-slope) model; u is then 0 and r̂ = A.
func (d *DriftModel) At(t time.Time) (factor, unc float64) {
	tt := float64(t.UnixMilli())
	var u float64
	if d.Tau != 0 {
		u = (tt - d.T0) / d.Tau
	}
	factor = d.A + d.B*u
	varR := d.VarA + 2*u*d.CovAB + u*u*d.VarB
	if varR < 0 {
		varR = 0
	}
	unc = math.Sqrt(varR)
	return factor, unc
}
