// Package measure ingests measurement runs and enforces the measurement
// lifecycle (raw -> usable -> contaminated/excluded). Imports are idempotent
// via a fingerprint so the same run is never counted twice.
package measure

import (
	"context"
	"errors"
	"fmt"

	"task248-blankcorr/internal/model"
	"task248-blankcorr/internal/store"
)

// Importer imports measurements and applies measurement status transitions.
type Importer struct {
	s *store.Store
}

// NewImporter constructs an Importer.
func NewImporter(s *store.Store) *Importer { return &Importer{s: s} }

// Import validates and stores a measurement. If an identical run (same
// fingerprint) was already imported, the existing measurement is returned
// together with a true "duplicate" flag so callers can stay idempotent.
func (im *Importer) Import(ctx context.Context, in model.MeasurementInput) (*model.Measurement, bool, error) {
	if err := in.Validate(); err != nil {
		return nil, false, err
	}
	b, err := im.s.GetBatch(in.BatchID)
	if err != nil {
		return nil, false, err
	}
	if b.IsSealed() {
		return nil, false, model.ErrSealed
	}
	m := model.BuildMeasurement(in)
	if err := im.s.CreateMeasurement(m); err != nil {
		if errors.Is(err, model.ErrDuplicate) {
			return m, true, nil
		}
		return nil, false, fmt.Errorf("store measurement: %w", err)
	}
	return m, false, nil
}

// SetStatus transitions a measurement to a new lifecycle status, enforcing the
// state machine (e.g. a contaminated measurement can never return to raw).
// A sealed batch is immutable: its measurement statuses are frozen so a sealed
// age version's inputs cannot drift after the fact.
func (im *Importer) SetStatus(ctx context.Context, id int64, status string) (*model.Measurement, error) {
	m, err := im.s.GetMeasurement(id)
	if err != nil {
		return nil, err
	}
	// reject mutation of a sealed batch's measurements before touching state,
	// so a sealed version's underlying inputs stay exactly as they were
	b, err := im.s.GetBatch(m.BatchID)
	if err != nil {
		return nil, err
	}
	if b.IsSealed() {
		return nil, model.ErrSealed
	}
	if !m.CanTransitionTo(status) {
		return nil, fmt.Errorf("%w: measurement %d %s -> %s", model.ErrConflict, id, m.Status, status)
	}
	if err := im.s.UpdateMeasurementStatus(id, status); err != nil {
		return nil, err
	}
	m.Status = status
	return m, nil
}
