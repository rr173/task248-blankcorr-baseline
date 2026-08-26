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
type DriftModel struct {
	A, B       float64 // intercept and slope
	VarA, VarB float64 // variances
	CovAB      float64 // covariance
	MeanT      float64 // mean time (for reporting)
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
		return &DriftModel{A: r, B: 0, VarA: sigR * sigR, VarB: 0, CovAB: 0, NTrained: 1}, nil
	}
	values := make([]struct {
		t, r, sig float64
	}, 0, len(eligible))
	for _, s := range eligible {
		r, sigR, err := Recovery(s)
		if err != nil {
			return nil, err
		}
		values = append(values, struct{ t, r, sig float64 }{float64(s.MeasuredAt.UnixMilli()), r, sigR})
	}
	var S, St, Sr float64
	for _, v := range values {
		w := 1.0 / (v.sig * v.sig)
		if v.sig == 0 {
			w = 1e24
		}
		S += w
		St += w * v.t
		Sr += w * v.r
	}
	if S == 0 || math.IsInf(S, 0) || math.IsNaN(S) {
		return nil, fmt.Errorf("invalid drift weights")
	}
	meanT := St / S
	meanR := Sr / S
	var sxx, sxy float64
	for _, v := range values {
		w := 1.0 / (v.sig * v.sig)
		if v.sig == 0 {
			w = 1e24
		}
		dt := v.t - meanT
		sxx += w * dt * dt
		sxy += w * dt * (v.r - meanR)
	}
	if sxx == 0 || math.IsInf(sxx, 0) || math.IsNaN(sxx) {
		return nil, fmt.Errorf("degenerate standard time spread; cannot fit drift")
	}
	b := sxy / sxx
	a := meanR - b*meanT
	varB := 1 / sxx
	varA := 1/S + meanT*meanT/sxx
	covAB := -meanT / sxx
	return &DriftModel{A: a, B: b, VarA: varA, VarB: varB, CovAB: covAB, MeanT: meanT, NTrained: len(eligible)}, nil
}

// At evaluates the fitted recovery ratio at time t and returns the value and
// its prediction uncertainty:
//
//	Var(r̂) = VarA + 2·t·CovAB + t²·VarB
func (d *DriftModel) At(t time.Time) (factor, unc float64) {
	tt := float64(t.UnixMilli())
	factor = d.A + d.B*tt
	varR := d.VarA + 2*tt*d.CovAB + tt*tt*d.VarB
	if varR < 0 {
		varR = 0
	}
	unc = math.Sqrt(varR)
	return factor, unc
}
