package service

import (
	"context"
	"testing"
	"time"

	"task248-blankcorr/internal/model"
	"task248-blankcorr/internal/store"
)

func TestCreateBatchAndImport(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	svc := New(s, time.Hour)
	b, err := svc.CreateBatch(context.Background(), "batch", "generic", 1e-4, 1, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	m, duplicate, err := svc.ImportMeasurement(context.Background(), model.MeasurementInput{BatchID: b.ID, Kind: model.KindSample, Material: "x", MeasuredAt: time.UnixMilli(1000), Ratio: 0.9})
	if err != nil || duplicate || m.ID == 0 || m.Status != model.MeasRaw {
		t.Fatalf("import result = %#v duplicate=%v err=%v", m, duplicate, err)
	}
}

func TestSelfCheckReportsMissingInputs(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	svc := New(s, time.Hour)
	b, err := svc.CreateBatch(context.Background(), "batch", "generic", 1e-4, 1, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	check, err := svc.SelfCheck(context.Background(), b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if check.SampleCount != 0 || len(check.Problems) == 0 {
		t.Fatalf("self-check = %#v, want a missing-input problem", check)
	}
}
