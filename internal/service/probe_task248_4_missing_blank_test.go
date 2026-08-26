package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"task248-blankcorr/internal/model"
	"task248-blankcorr/internal/store"
)

func TestProbeBug04MissingBlankCannotCorrect(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	svc := New(s, time.Hour)
	ctx := context.Background()
	b, err := svc.CreateBatch(ctx, "no-blank", "generic", 1e-4, 1, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	inputs := []model.MeasurementInput{
		{BatchID: b.ID, Kind: model.KindStandard, Material: "std", MeasuredAt: time.UnixMilli(1000), Ratio: 1, RatioUnc: 0.01, CertifiedRatio: 1},
		{BatchID: b.ID, Kind: model.KindBlank, Material: "far", MeasuredAt: time.UnixMilli(1000 + int64(24*time.Hour/time.Millisecond)), Ratio: 0.01, RatioUnc: 0.001},
		{BatchID: b.ID, Kind: model.KindSample, Material: "sample", MeasuredAt: time.UnixMilli(1000), Ratio: 0.9, RatioUnc: 0.01},
	}
	for _, in := range inputs {
		if _, _, err := svc.ImportMeasurement(ctx, in); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.Match(ctx, b.ID); !errors.Is(err, model.ErrConflict) {
		t.Fatalf("missing-blank match error = %v, want ErrConflict", err)
	}
	rels, err := svc.ListCorrections(ctx, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rels) != 0 {
		t.Fatalf("missing-blank match created %d correction relations", len(rels))
	}
}
