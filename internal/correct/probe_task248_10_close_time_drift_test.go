package correct

import (
	"math"
	"testing"
	"time"

	"task248-blankcorr/internal/model"
)

func TestProbeBug10CloseTimeDriftIsStable(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	standards := []*model.Measurement{
		{Kind: model.KindStandard, Status: model.MeasRaw, MeasuredAt: base, Ratio: 1, RatioUnc: 0.01, CertifiedRatio: 1},
		{Kind: model.KindStandard, Status: model.MeasRaw, MeasuredAt: base.Add(time.Second), Ratio: 0.5, RatioUnc: 0.01, CertifiedRatio: 0.55},
	}
	drift, err := FitDrift(standards)
	if err != nil {
		t.Fatalf("close-time FitDrift error = %v", err)
	}
	factor, unc := drift.At(base.Add(500 * time.Millisecond))
	if math.IsNaN(factor) || math.IsInf(factor, 0) || math.IsNaN(unc) || math.IsInf(unc, 0) {
		t.Fatalf("close-time drift is not finite: factor=%v unc=%v", factor, unc)
	}
	if math.Abs(factor-1.05) > 0.01 {
		t.Fatalf("close-time drift factor = %v, want approximately 1.05", factor)
	}
}
