package store

import (
	"testing"
	"time"

	"task248-blankcorr/internal/model"
)

func TestBatchRoundTrip(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	b, err := model.NewBatch("batch", "generic", 1e-4, 1, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CreateBatch(b); err != nil {
		t.Fatal(err)
	}
	if got, err := s.GetBatch(b.ID); err != nil || got.Name != b.Name || got.Status != model.BatchReceiving {
		t.Fatalf("GetBatch = %#v, err=%v", got, err)
	}
	m := model.BuildMeasurement(model.MeasurementInput{BatchID: b.ID, Kind: model.KindSample, Material: "x", MeasuredAt: time.UnixMilli(1000), Ratio: 0.9})
	if err := s.CreateMeasurement(m); err != nil {
		t.Fatal(err)
	}
	ms, err := s.ListMeasurements(b.ID, model.KindSample, nil)
	if err != nil || len(ms) != 1 || ms[0].ID != m.ID {
		t.Fatalf("ListMeasurements = %#v, err=%v", ms, err)
	}
}
