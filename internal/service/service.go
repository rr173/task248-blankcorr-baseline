// Package service is the orchestration layer. It wires the measurement,
// matching, correction, review and version packages together and exposes the
// high-level operations used by both the HTTP API and the smoke test, so the
// two entry points share exactly the same business logic.
package service

import (
	"context"
	"fmt"
	"time"

	"task248-blankcorr/internal/correct"
	"task248-blankcorr/internal/match"
	"task248-blankcorr/internal/measure"
	"task248-blankcorr/internal/model"
	"task248-blankcorr/internal/review"
	"task248-blankcorr/internal/store"
	"task248-blankcorr/internal/version"
)

// Service exposes the full blank-correction workflow.
type Service struct {
	store     *store.Store
	measure   *measure.Importer
	matcher   *match.Matcher
	reviewer  *review.Reviewer
	versioner *version.Versioner
	window    time.Duration
}

// New constructs a Service with the default matching window.
func New(s *store.Store, window time.Duration) *Service {
	if window <= 0 {
		window = 24 * time.Hour // generous default window
	}
	return &Service{
		store:     s,
		measure:   measure.NewImporter(s),
		matcher:   match.NewMatcher(s),
		reviewer:  review.NewReviewer(s),
		versioner: version.NewVersioner(s),
		window:    window,
	}
}

// Store exposes the underlying store (used by smoke-test restart checks).
func (svc *Service) Store() *store.Store { return svc.store }

// Window returns the configured matching window.
func (svc *Service) Window() time.Duration { return svc.window }

// CreateBatch validates and persists a new measurement batch.
func (svc *Service) CreateBatch(ctx context.Context, name, systemType string, lambda, r0, lo, hi float64) (*model.Batch, error) {
	b, err := model.NewBatch(name, systemType, lambda, r0, lo, hi)
	if err != nil {
		return nil, err
	}
	if err := svc.store.CreateBatch(b); err != nil {
		return nil, fmt.Errorf("create batch: %w", err)
	}
	return b, nil
}

// ImportMeasurement imports a measurement run (idempotent).
func (svc *Service) ImportMeasurement(ctx context.Context, in model.MeasurementInput) (*model.Measurement, bool, error) {
	return svc.measure.Import(ctx, in)
}

// SetMeasurementStatus transitions a measurement lifecycle status.
func (svc *Service) SetMeasurementStatus(ctx context.Context, id int64, status string) (*model.Measurement, error) {
	return svc.setMeasurementStatus(ctx, id, status)
}

func (svc *Service) setMeasurementStatus(ctx context.Context, id int64, status string) (*model.Measurement, error) {
	return svc.measure.SetStatus(ctx, id, status)
}

// Match rebuilds candidate correction relations for a batch.
func (svc *Service) Match(ctx context.Context, batchID int64) ([]*model.CorrectionRelation, error) {
	return svc.matcher.Match(batchID, svc.window)
}

// Recompute re-matches the batch (re-building candidate relations after an
// exclusion) and then recomputes the age results. It is the typical "re-run
// after removing a bad blank" workflow.
func (svc *Service) Recompute(ctx context.Context, batchID int64) ([]*model.AgeResult, error) {
	if _, err := svc.Match(ctx, batchID); err != nil {
		return nil, err
	}
	return svc.ComputeAges(ctx, batchID)
}

// ListCorrections returns a batch's correction relations.
func (svc *Service) ListCorrections(ctx context.Context, batchID int64) ([]*model.CorrectionRelation, error) {
	return svc.store.ListCorrections(batchID)
}

// SetCorrectionStatus transitions a correction relation status.
func (svc *Service) SetCorrectionStatus(ctx context.Context, id int64, status string) (*model.CorrectionRelation, error) {
	c, err := svc.store.GetCorrection(id)
	if err != nil {
		return nil, err
	}
	if err := c.Transition(status); err != nil {
		return nil, err
	}
	if err := svc.store.UpdateCorrectionStatus(id, status); err != nil {
		return nil, err
	}
	return c, nil
}

// ComputeAges recomputes the age results for every non-conflict correction
// relation in the batch, propagates uncertainty and flags anomalies against
// the batch expected interval. Existing age results are purged first.
func (svc *Service) ComputeAges(ctx context.Context, batchID int64) ([]*model.AgeResult, error) {
	b, err := svc.store.GetBatch(batchID)
	if err != nil {
		return nil, err
	}
	if b.IsSealed() {
		return nil, model.ErrSealed
	}
	rels, err := svc.store.ListCorrections(batchID)
	if err != nil {
		return nil, err
	}
	if err := svc.store.PurgeAgeResults(batchID); err != nil {
		return nil, fmt.Errorf("purge age results: %w", err)
	}
	out := make([]*model.AgeResult, 0, len(rels))
	hasAnomaly := false
	for _, c := range rels {
		if c.Status == model.CorrConflict {
			continue
		}
		smp, err := svc.store.GetMeasurement(c.SampleID)
		if err != nil {
			return nil, err
		}
		corr, sigCorr := correct.CorrectedRatio(smp.Ratio, smp.RatioUnc, c.BValue, c.BUnc, c.DriftFactor, c.DriftUnc)
		a := &model.AgeResult{
			RelationID:     c.ID,
			SampleID:       smp.ID,
			BatchID:        batchID,
			CorrectedRatio: corr,
			CorrectedUnc:   sigCorr,
			ExpectedLow:    b.ExpectedLow,
			ExpectedHigh:   b.ExpectedHigh,
		}
		if age, sigAge, err := correct.Age(b.Lambda, b.R0, corr, sigCorr); err == nil {
			a.AgeValue = age
			a.AgeUnc = sigAge
			a.AgeLow = age - 2*sigAge
			a.AgeHigh = age + 2*sigAge
		} else {
			a.Reason = err.Error()
		}
		a.SetAnomaly()
		if err := svc.store.CreateAgeResult(a); err != nil {
			return nil, fmt.Errorf("store age result: %w", err)
		}
		if a.AnomalyFlag {
			hasAnomaly = true
		}
		out = append(out, a)
	}

	// reflect anomaly status on the batch (forward only, never regress a
	// published/sealed batch)
	if !b.IsSealed() && b.Status != model.BatchPublished {
		if hasAnomaly && (b.Status == model.BatchReceiving || b.Status == model.BatchPending) {
			_ = svc.store.UpdateBatchStatus(batchID, model.BatchNeedsReview)
		} else if !hasAnomaly && b.Status == model.BatchNeedsReview {
			_ = svc.store.UpdateBatchStatus(batchID, model.BatchPending)
		}
	}
	return out, nil
}

// ExcludeMeasurement excludes a compromised measurement and records why.
func (svc *Service) ExcludeMeasurement(ctx context.Context, id int64, reason string, contaminated bool) (*model.Measurement, error) {
	return svc.reviewer.ExcludeMeasurement(ctx, id, reason, contaminated)
}

// PublishVersion collects age results into a published version.
func (svc *Service) PublishVersion(ctx context.Context, batchID int64, name, note string, resultIDs []int64) (*model.AgeVersion, error) {
	return svc.versioner.Publish(ctx, batchID, name, note, resultIDs)
}

// SealVersion seals a version and its batch.
func (svc *Service) SealVersion(ctx context.Context, versionID int64) (*model.AgeVersion, error) {
	return svc.versioner.Seal(ctx, versionID)
}

// Read helpers ------------------------------------------------------------

// GetBatch loads a batch.
func (svc *Service) GetBatch(ctx context.Context, id int64) (*model.Batch, error) {
	return svc.store.GetBatch(id)
}

// ListBatches lists all batches.
func (svc *Service) ListBatches(ctx context.Context) ([]*model.Batch, error) {
	return svc.store.ListBatches()
}

// ListMeasurements lists a batch's measurements with optional filters.
func (svc *Service) ListMeasurements(ctx context.Context, batchID int64, kind string, statuses []string) ([]*model.Measurement, error) {
	return svc.store.ListMeasurements(batchID, kind, statuses)
}

// ListAgeResults lists a batch's age results.
func (svc *Service) ListAgeResults(ctx context.Context, batchID int64) ([]*model.AgeResult, error) {
	return svc.store.ListAgeResults(batchID)
}

// GetVersion loads a version.
func (svc *Service) GetVersion(ctx context.Context, id int64) (*model.AgeVersion, error) {
	return svc.store.GetVersion(id)
}

// ListVersions lists a batch's versions.
func (svc *Service) ListVersions(ctx context.Context, batchID int64) ([]*model.AgeVersion, error) {
	return svc.store.ListVersions(batchID)
}

// ListExclusions lists a batch's exclusions.
func (svc *Service) ListExclusions(ctx context.Context, batchID int64) ([]store.Exclusion, error) {
	return svc.store.ListExclusions(batchID)
}
