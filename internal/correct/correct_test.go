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

// TestFitDriftNumericStability guards against the catastrophic-cancellation
// bug that arose when two standards of the same batch landed at very close
// Unix timestamps: with raw ~1.7e12 ms timestamps the normal-equation
// denominator S·Stt − St² subtracted two ~1e25 terms to get a tiny result,
// losing all precision and producing recovery factors far from the observed
// ~1.0 (the repro originally surfaced 0.64 and −134). The centered, scaled
// fit must instead return a factor near the observed recovery and finite
// uncertainty, while still matching the wide-span behaviour.
func TestFitDriftNumericStability(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Two standards one second apart, identical recovery (≈1.0). Any linear
	// fit through two equal points must be ≈1.0 everywhere between them.
	close := []*model.Measurement{
		standard(base, 1.0, 0.01),
		standard(base.Add(1*time.Second), 1.0, 0.01),
	}
	d, err := FitDrift(close)
	if err != nil {
		t.Fatalf("FitDrift (close) failed: %v", err)
	}
	f, u := d.At(base.Add(500 * time.Millisecond))
	if math.IsNaN(f) || math.IsInf(f, 0) || math.Abs(f-1.0) > 1e-6 {
		t.Fatalf("close-in-time factor not ~1.0: got %v (unc %v)", f, u)
	}

	// Same close spacing but a tiny recovery slope; the midpoint recovery must
	// equal the average of the two recoveries (≈1.000001), not a wild value.
	slope := []*model.Measurement{
		&model.Measurement{Kind: model.KindStandard, MeasuredAt: base, Ratio: 0.9990, RatioUnc: 0.01, CertifiedRatio: 1.0, Status: model.MeasRaw},
		&model.Measurement{Kind: model.KindStandard, MeasuredAt: base.Add(1 * time.Second), Ratio: 1.0010, RatioUnc: 0.01, CertifiedRatio: 1.0, Status: model.MeasRaw},
	}
	d2, err := FitDrift(slope)
	if err != nil {
		t.Fatalf("FitDrift (slope) failed: %v", err)
	}
	f2, u2 := d2.At(base.Add(500 * time.Millisecond))
	want2 := 0.5 * (1.0/0.9990 + 1.0/1.0010)
	if math.IsNaN(f2) || math.IsInf(f2, 0) || math.Abs(f2-want2) > 1e-6 {
		t.Fatalf("close-in-time midpoint recovery off: got %v want %v (unc %v)", f2, want2, u2)
	}

	// Wide span still behaves: a 30-day span with recovery rising from 1.0 to
	// 1.1 must give ≈1.05 halfway. This is the same physics the pre-fix code
	// already handled; it must keep working.
	wide := []*model.Measurement{
		&model.Measurement{Kind: model.KindStandard, MeasuredAt: base, Ratio: 1.0, RatioUnc: 0.01, CertifiedRatio: 1.0, Status: model.MeasRaw},
		&model.Measurement{Kind: model.KindStandard, MeasuredAt: base.Add(30 * 24 * time.Hour), Ratio: 1.0 / 1.1, RatioUnc: 0.01, CertifiedRatio: 1.0, Status: model.MeasRaw},
	}
	d3, err := FitDrift(wide)
	if err != nil {
		t.Fatalf("FitDrift (wide) failed: %v", err)
	}
	f3, _ := d3.At(base.Add(15 * 24 * time.Hour))
	if math.Abs(f3-1.05) > 1e-9 {
		t.Fatalf("wide-span midpoint recovery off: got %v want 1.05", f3)
	}
}
