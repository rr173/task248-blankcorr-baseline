package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"task248-blankcorr/internal/model"
	"task248-blankcorr/internal/store"
)

func TestProbeBug06SelfCheckExcludesContaminatedBlank(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	svc := New(s, time.Hour)
	ctx := context.Background()
	b, err := svc.CreateBatch(ctx, "blank-review", "generic", 1e-4, 1, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	var blankID int64
	for _, in := range []model.MeasurementInput{
		{BatchID: b.ID, Kind: model.KindStandard, Material: "std", MeasuredAt: time.UnixMilli(1000), Ratio: 1, RatioUnc: 0.01, CertifiedRatio: 1},
		{BatchID: b.ID, Kind: model.KindBlank, Material: "blank", MeasuredAt: time.UnixMilli(1000), Ratio: 0.5, RatioUnc: 0.01},
		{BatchID: b.ID, Kind: model.KindSample, Material: "sample", MeasuredAt: time.UnixMilli(1000), Ratio: 0.9, RatioUnc: 0.01},
	} {
		m, _, err := svc.ImportMeasurement(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if in.Kind == model.KindBlank {
			blankID = m.ID
		}
	}
	if _, err := svc.ExcludeMeasurement(ctx, blankID, "contaminated", true); err != nil {
		t.Fatal(err)
	}
	check, err := svc.SelfCheck(ctx, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, problem := range check.Problems {
		if strings.Contains(problem, "no blanks") {
			return
		}
	}
	t.Fatalf("self-check did not report missing active blank: %#v", check)
}
