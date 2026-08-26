package store

import (
	"testing"
	"time"

	"task248-blankcorr/internal/model"
)

func TestProbeBug07ListAgeResultsRestoresCreatedAt(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	b, _ := model.NewBatch("batch", "generic", 1e-4, 1, 0, 0)
	if err := s.CreateBatch(b); err != nil {
		t.Fatal(err)
	}
	m := model.BuildMeasurement(model.MeasurementInput{BatchID: b.ID, Kind: model.KindSample, Material: "x", MeasuredAt: time.UnixMilli(1000), Ratio: 0.9})
	if err := s.CreateMeasurement(m); err != nil {
		t.Fatal(err)
	}
	c := &model.CorrectionRelation{BatchID: b.ID, SampleID: m.ID, Status: model.CorrConfirmed, DriftFactor: 1}
	if err := s.CreateCorrection(c); err != nil {
		t.Fatal(err)
	}
	want := int64(123456789)
	a := &model.AgeResult{RelationID: c.ID, SampleID: m.ID, BatchID: b.ID, CorrectedRatio: 0.9, CreatedAt: want}
	if err := s.CreateAgeResult(a); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListAgeResults(b.ID)
	if err != nil || len(got) != 1 {
		t.Fatalf("ListAgeResults = %#v, err=%v", got, err)
	}
	if got[0].CreatedAt != want {
		t.Fatalf("CreatedAt = %d, want %d", got[0].CreatedAt, want)
	}
}
