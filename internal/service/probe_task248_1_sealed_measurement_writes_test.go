package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"task248-blankcorr/internal/model"
	"task248-blankcorr/internal/store"
)

func TestProbeBug01SealedMeasurementWrites(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	svc := New(s, time.Hour)
	ctx := context.Background()
	b, err := svc.CreateBatch(ctx, "sealed", "generic", 1e-4, 1, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	m, _, err := svc.ImportMeasurement(ctx, model.MeasurementInput{BatchID: b.ID, Kind: model.KindSample, Material: "x", MeasuredAt: time.UnixMilli(1000), Ratio: 0.9})
	if err != nil {
		t.Fatal(err)
	}
	v := model.NewAgeVersion(b.ID, "v1", "")
	if err := s.CreateVersion(v); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateVersionStatus(v.ID, model.VersionPublished); err != nil {
		t.Fatal(err)
	}
	if err := s.SetBatchSealed(b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetMeasurementStatus(ctx, m.ID, model.MeasUsable); !errors.Is(err, model.ErrSealed) {
		t.Fatalf("status update error = %v, want ErrSealed", err)
	}
	if _, err := svc.ExcludeMeasurement(ctx, m.ID, "late review", false); !errors.Is(err, model.ErrSealed) {
		t.Fatalf("exclusion error = %v, want ErrSealed", err)
	}
	got, err := s.GetMeasurement(m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.MeasRaw {
		t.Fatalf("sealed measurement status = %q, want raw", got.Status)
	}
}
