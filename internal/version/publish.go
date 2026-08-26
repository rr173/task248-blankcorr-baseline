// Package version collects computed age results into named, freezable age
// versions. Publishing shares a version; sealing freezes it (and the owning
// batch) so the chosen blank/standard corrections become immutable.
package version

import (
	"context"
	"fmt"

	"task248-blankcorr/internal/model"
	"task248-blankcorr/internal/store"
)

// Versioner publishes and seals age versions.
type Versioner struct {
	s *store.Store
}

// NewVersioner constructs a Versioner.
func NewVersioner(s *store.Store) *Versioner { return &Versioner{s: s} }

// Publish creates a draft version, links the supplied age results, advances it
// to published and marks the batch as published. It refuses to run on a
// sealed batch.
func (v *Versioner) Publish(ctx context.Context, batchID int64, name, note string, resultIDs []int64) (*model.AgeVersion, error) {
	b, err := v.s.GetBatch(batchID)
	if err != nil {
		return nil, err
	}
	if b.IsSealed() {
		return nil, model.ErrSealed
	}
	seen := make(map[int64]struct{}, len(resultIDs))
	for _, rid := range resultIDs {
		if _, ok := seen[rid]; ok {
			continue
		}
		seen[rid] = struct{}{}
		a, err := v.s.GetAgeResult(rid)
		if err != nil {
			return nil, fmt.Errorf("validate result %d: %w", rid, err)
		}
		if a.BatchID != batchID {
			return nil, fmt.Errorf("%w: result %d belongs to batch %d, not %d", model.ErrConflict, rid, a.BatchID, batchID)
		}
		c, err := v.s.GetCorrection(a.RelationID)
		if err != nil {
			return nil, fmt.Errorf("validate relation for result %d: %w", rid, err)
		}
		if c.Status != model.CorrConfirmed {
			return nil, fmt.Errorf("%w: result %d uses unconfirmed correction %d", model.ErrConflict, rid, c.ID)
		}
	}
	ver := model.NewAgeVersion(batchID, name, note)
	if err := v.s.CreateVersion(ver); err != nil {
		return nil, fmt.Errorf("create version: %w", err)
	}
	for _, rid := range resultIDs {
		if err := v.s.AddVersionEntry(ver.ID, rid); err != nil {
			return nil, fmt.Errorf("link result: %w", err)
		}
	}
	if err := v.s.UpdateVersionStatus(ver.ID, model.VersionPublished); err != nil {
		return nil, fmt.Errorf("publish version: %w", err)
	}
	if err := v.s.UpdateBatchStatus(batchID, model.BatchPublished); err != nil {
		return nil, fmt.Errorf("mark batch published: %w", err)
	}
	return ver, nil
}

// Seal freezes a published version and the owning batch. After sealing, no
// measurement, correction, result or version of that batch may change.
func (v *Versioner) Seal(ctx context.Context, versionID int64) (*model.AgeVersion, error) {
	ver, err := v.s.GetVersion(versionID)
	if err != nil {
		return nil, err
	}
	if !ver.CanTransitionTo(model.VersionSealed) {
		return nil, fmt.Errorf("%w: version %d status %s", model.ErrConflict, versionID, ver.Status)
	}
	if err := v.s.UpdateVersionStatus(versionID, model.VersionSealed); err != nil {
		return nil, fmt.Errorf("seal version: %w", err)
	}
	if err := v.s.SetBatchSealed(ver.BatchID); err != nil {
		return nil, fmt.Errorf("seal batch: %w", err)
	}
	ver.Status = model.VersionSealed
	return ver, nil
}
