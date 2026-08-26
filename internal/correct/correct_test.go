package correct

import (
	"math"
	"testing"
	"time"

	"task248-blankcorr/internal/model"
)

func standard(at time.Time, ratio, unc float64) *model.Measurement {
	return &model.Measurement{Kind: model.KindStandard, MeasuredAt: at, Ratio: ratio, RatioUnc: unc, CertifiedRatio: ratio, Status: model.MeasRaw}
}

func TestAgeAndCorrectedRatio(t *testing.T) {
	ratio, unc := CorrectedRatio(0.9, 0.01, 0.1, 0.01, 1.05, 0.02)
	if ratio <= 0 || unc <= 0 {
		t.Fatalf("unexpected corrected ratio: ratio=%v unc=%v", ratio, unc)
	}
	age, sig, err := Age(1e-4, 1, ratio, unc)
	if err != nil || age <= 0 || sig <= 0 {
		t.Fatalf("unexpected age: age=%v sig=%v err=%v", age, sig, err)
	}
}

func TestFitDriftWithSeparatedUnixTimes(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	standards := []*model.Measurement{
		standard(base, 1.0, 0.01),
		&model.Measurement{Kind: model.KindStandard, MeasuredAt: base.Add(24 * time.Hour), Ratio: 0.5, RatioUnc: 0.01, CertifiedRatio: 0.55, Status: model.MeasRaw},
	}
	drift, err := FitDrift(standards)
	if err != nil {
		t.Fatalf("FitDrift failed: %v", err)
	}
	factor, unc := drift.At(base.Add(12 * time.Hour))
	if math.IsNaN(factor) || math.IsInf(factor, 0) || math.IsNaN(unc) || math.IsInf(unc, 0) {
		t.Fatalf("unstable drift output: factor=%v unc=%v", factor, unc)
	}
}
