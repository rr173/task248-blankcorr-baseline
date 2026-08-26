package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"task248-blankcorr/internal/model"
	"task248-blankcorr/internal/store"
)

// TestSealedBatchRejectsMeasurementMutation reproduces the regression: once an
// age version is sealed, experimenters must not be able to flip a
// measurement status or mark a measurement as excluded, because the sealed
// version's inputs (blank/standard choices, exclusion history) must stay
// frozen. Unsealed batches keep working normally.
func TestSealedBatchRejectsMeasurementMutation(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	svc := New(s, 24*time.Hour)

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	at := func(ms int64) time.Time { return base.Add(time.Duration(ms) * time.Millisecond) }

	b, err := svc.CreateBatch(ctx, "sealed-guard", "generic", 1e-4, 1.0, 900, 1300)
	if err != nil {
		t.Fatal(err)
	}
	// standard, contaminated blank (nearest), clean blank (farther), sample
	imp := func(kind, material string, measuredAt time.Time, ratio, unc float64, cert float64) int64 {
		m, _, err := svc.ImportMeasurement(ctx, model.MeasurementInput{
			BatchID: b.ID, Kind: kind, Material: material, MeasuredAt: measuredAt,
			Ratio: ratio, RatioUnc: unc, CertifiedRatio: cert,
		})
		if err != nil {
			t.Fatalf("import %s: %v", kind, err)
		}
		return m.ID
	}
	imp(model.KindStandard, "SRM-1", at(800), 1.0, 0.01, 1.0)
	badBlank := imp(model.KindBlank, "reagent-A", at(1005), 0.5, 0.01, 0)
	imp(model.KindBlank, "reagent-A", at(1200), 0.01, 0.002, 0)
	imp(model.KindSample, "unknown-X", at(1000), 0.91, 0.01, 0)

	if _, err := svc.Match(ctx, b.ID); err != nil {
		t.Fatalf("match: %v", err)
	}
	results, err := svc.ComputeAges(ctx, b.ID)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	// exclude the contaminated blank and recompute to a clean age
	if _, err := svc.ExcludeMeasurement(ctx, badBlank, "contaminated", true); err != nil {
		t.Fatalf("exclude bad blank pre-seal: %v", err)
	}
	results, err = svc.Recompute(ctx, b.ID)
	if err != nil {
		t.Fatalf("recompute: %v", err)
	}
	// confirm, publish and seal the version
	rels, err := svc.ListCorrections(ctx, b.ID)
	if err != nil {
		t.Fatalf("list relations: %v", err)
	}
	if _, err := svc.SetCorrectionStatus(ctx, rels[0].ID, model.CorrConfirmed); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	ver, err := svc.PublishVersion(ctx, b.ID, "v1", "", []int64{results[0].ID})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if _, err := svc.SealVersion(ctx, ver.ID); err != nil {
		t.Fatalf("seal: %v", err)
	}

	// capture the measurements + exclusions as they were at seal time
	snapMs, err := svc.ListMeasurements(ctx, b.ID, "", nil)
	if err != nil {
		t.Fatalf("snapshot measurements: %v", err)
	}
	snapExcl, err := svc.ListExclusions(ctx, b.ID)
	if err != nil {
		t.Fatalf("snapshot exclusions: %v", err)
	}

	// (1) flipping a measurement status on a sealed batch must be rejected
	for _, target := range []string{model.MeasUsable, model.MeasExcluded, model.MeasContaminated} {
		if _, err := svc.SetMeasurementStatus(ctx, badBlank, target); !errors.Is(err, model.ErrSealed) {
			t.Fatalf("SetMeasurementStatus(%s) on sealed batch: err=%v, want ErrSealed", target, err)
		}
	}

	// (2) excluding a measurement on a sealed batch must be rejected, and
	// must not create a new exclusion record
	if _, err := svc.ExcludeMeasurement(ctx, badBlank, "late change", true); !errors.Is(err, model.ErrSealed) {
		t.Fatalf("ExcludeMeasurement on sealed batch: err=%v, want ErrSealed", err)
	}

	// verify nothing actually changed after the rejected attempts
	afterMs, err := svc.ListMeasurements(ctx, b.ID, "", nil)
	if err != nil {
		t.Fatalf("list measurements: %v", err)
	}
	if len(afterMs) != len(snapMs) {
		t.Fatalf("measurement count changed: %d -> %d", len(snapMs), len(afterMs))
	}
	for i := range snapMs {
		if snapMs[i].Status != afterMs[i].Status {
			t.Fatalf("measurement %d status changed: %s -> %s",
				snapMs[i].ID, snapMs[i].Status, afterMs[i].Status)
		}
	}
	afterExcl, err := svc.ListExclusions(ctx, b.ID)
	if err != nil {
		t.Fatalf("list exclusions: %v", err)
	}
	if len(afterExcl) != len(snapExcl) {
		t.Fatalf("exclusion count changed: %d -> %d", len(snapExcl), len(afterExcl))
	}
}

// TestUnsealedBatchAllowsMeasurementMutation confirms the guard does not
// block the normal review flow on an unsealed batch.
func TestUnsealedBatchAllowsMeasurementMutation(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	svc := New(s, time.Hour)
	b, err := svc.CreateBatch(ctx, "unsealed", "generic", 1e-4, 1.0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	m, _, err := svc.ImportMeasurement(ctx, model.MeasurementInput{
		BatchID: b.ID, Kind: model.KindBlank, Material: "b",
		MeasuredAt: time.UnixMilli(1000), Ratio: 0.5, RatioUnc: 0.01,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetMeasurementStatus(ctx, m.ID, model.MeasUsable); err != nil {
		t.Fatalf("set status on unsealed batch: %v", err)
	}
	if _, err := svc.ExcludeMeasurement(ctx, m.ID, "contaminated", true); err != nil {
		t.Fatalf("exclude on unsealed batch: %v", err)
	}
}
