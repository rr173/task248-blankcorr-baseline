package version

import (
	"context"
	"errors"
	"testing"
	"time"

	"task248-blankcorr/internal/model"
	"task248-blankcorr/internal/store"
)

func TestProbeBug03CrossBatchResultCannotPublish(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	batchA, _ := model.NewBatch("a", "generic", 1e-4, 1, 0, 0)
	batchB, _ := model.NewBatch("b", "generic", 1e-4, 1, 0, 0)
	if err := s.CreateBatch(batchA); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateBatch(batchB); err != nil {
		t.Fatal(err)
	}
	m := model.BuildMeasurement(model.MeasurementInput{BatchID: batchB.ID, Kind: model.KindSample, Material: "b-sample", MeasuredAt: time.UnixMilli(1000).UTC(), Ratio: 0.9})
	if err := s.CreateMeasurement(m); err != nil {
		t.Fatal(err)
	}
	c := &model.CorrectionRelation{BatchID: batchB.ID, SampleID: m.ID, Status: model.CorrConfirmed, DriftFactor: 1}
	if err := s.CreateCorrection(c); err != nil {
		t.Fatal(err)
	}
	a := &model.AgeResult{RelationID: c.ID, SampleID: m.ID, BatchID: batchB.ID, CorrectedRatio: 0.9}
	if err := s.CreateAgeResult(a); err != nil {
		t.Fatal(err)
	}
	v := NewVersioner(s)
	if _, err := v.Publish(context.Background(), batchA.ID, "wrong", "", []int64{a.ID}); !errors.Is(err, model.ErrConflict) {
		t.Fatalf("cross-batch publish error = %v, want ErrConflict", err)
	}
	versions, err := s.ListVersions(batchA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 0 {
		t.Fatalf("cross-batch publish left %d versions", len(versions))
	}
}
