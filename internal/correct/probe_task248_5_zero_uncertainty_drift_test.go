package correct

import (
	"math"
	"testing"
	"time"

	"task248-blankcorr/internal/model"
)

func TestProbeBug05ZeroUncertaintyDriftIsFinite(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	standards := []*model.Measurement{
		{Kind: model.KindStandard, Status: model.MeasRaw, MeasuredAt: base, Ratio: 1, CertifiedRatio: 1},
		{Kind: model.KindStandard, Status: model.MeasRaw, MeasuredAt: base.Add(time.Hour), Ratio: 1, CertifiedRatio: 1},
	}
	drift, err := FitDrift(standards)
	if err != nil {
		t.Fatalf("zero-uncertainty FitDrift error = %v", err)
	}
	factor, unc := drift.At(base.Add(30 * time.Minute))
	if math.IsNaN(factor) || math.IsInf(factor, 0) || math.IsNaN(unc) || math.IsInf(unc, 0) {
		t.Fatalf("zero-uncertainty drift is not finite: factor=%v unc=%v", factor, unc)
	}
}
