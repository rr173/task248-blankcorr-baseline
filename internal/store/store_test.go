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

// TestAgeResultCreatedAtRoundTrip guards against regression of the bug where
// the age result's created_at was scanned but never assigned back to the model,
// so audit-time ordering was lost after a service restart.
func TestAgeResultCreatedAtRoundTrip(t *testing.T) {
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
	m := model.BuildMeasurement(model.MeasurementInput{
		BatchID: b.ID, Kind: model.KindSample, Material: "x",
		MeasuredAt: time.UnixMilli(1000), Ratio: 0.9,
	})
	if err := s.CreateMeasurement(m); err != nil {
		t.Fatal(err)
	}
	rel := &model.CorrectionRelation{
		BatchID: b.ID, SampleID: m.ID, Status: model.CorrConfirmed,
	}
	if err := s.CreateCorrection(rel); err != nil {
		t.Fatal(err)
	}

	// Insert two results with distinct, deterministic created_at timestamps so
	// that the audit ordering is observable after a reopen.
	a1 := &model.AgeResult{
		RelationID: rel.ID, SampleID: m.ID, BatchID: b.ID,
		AgeValue: 1000, CreatedAt: 1_700_000_000_000,
	}
	a2 := &model.AgeResult{
		RelationID: rel.ID, SampleID: m.ID, BatchID: b.ID,
		AgeValue: 2000, CreatedAt: 1_700_000_001_000,
	}
	if err := s.CreateAgeResult(a1); err != nil {
		t.Fatalf("create age result 1: %v", err)
	}
	if err := s.CreateAgeResult(a2); err != nil {
		t.Fatalf("create age result 2: %v", err)
	}
	if a1.CreatedAt == 0 || a2.CreatedAt == 0 {
		t.Fatalf("CreateAgeResult should set CreatedAt, got %d / %d", a1.CreatedAt, a2.CreatedAt)
	}

	// Single-result read must preserve the persisted timestamp.
	got, err := s.GetAgeResult(a2.ID)
	if err != nil {
		t.Fatalf("GetAgeResult: %v", err)
	}
	if got.CreatedAt != a2.CreatedAt {
		t.Fatalf("GetAgeResult CreatedAt = %d, want %d (matching single-result view)",
			got.CreatedAt, a2.CreatedAt)
	}

	// List read must preserve each persisted timestamp so the audit order can be
	// reconstructed after a restart.
	listed, err := s.ListAgeResults(b.ID)
	if err != nil {
		t.Fatalf("ListAgeResults: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("ListAgeResults returned %d results, want 2", len(listed))
	}
	want := map[int64]int64{a1.ID: a1.CreatedAt, a2.ID: a2.CreatedAt}
	for _, r := range listed {
		if r.CreatedAt != want[r.ID] {
			t.Fatalf("ListAgeResults result id=%d CreatedAt = %d, want %d",
				r.ID, r.CreatedAt, want[r.ID])
		}
		// The list and single-read views must agree on the timestamp.
		single, err := s.GetAgeResult(r.ID)
		if err != nil {
			t.Fatalf("GetAgeResult(%d): %v", r.ID, err)
		}
		if single.CreatedAt != r.CreatedAt {
			t.Fatalf("list vs single CreatedAt mismatch for id=%d: list=%d single=%d",
				r.ID, r.CreatedAt, single.CreatedAt)
		}
	}
}
