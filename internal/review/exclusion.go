// Package review manages the human review actions: excluding a compromised
// measurement (typically a contaminated blank) and recomputing. Excluding a
// bad blank and re-matching is the central "is my age real or an artefact?"
// workflow of the service.
package review

import (
	"context"
	"fmt"

	"task248-blankcorr/internal/model"
	"task248-blankcorr/internal/store"
)

// Reviewer applies review decisions.
type Reviewer struct {
	s *store.Store
}

// NewReviewer constructs a Reviewer.
func NewReviewer(s *store.Store) *Reviewer { return &Reviewer{s: s} }

// ExcludeMeasurement marks a measurement as terminal (contaminated or
// excluded) and records the exclusion with a reason. Once excluded, the
// measurement leaves the candidate pool used by the matcher, so re-matching
// automatically falls back to the next best blank/standard. A sealed batch is
// immutable: its measurement statuses and exclusion history are frozen so a
// sealed age version's inputs cannot change after the fact.
func (rv *Reviewer) ExcludeMeasurement(ctx context.Context, id int64, reason string, contaminated bool) (*model.Measurement, error) {
	m, err := rv.s.GetMeasurement(id)
	if err != nil {
		return nil, err
	}
	// reject mutation of a sealed batch's measurements and exclusion history
	// before touching state, so a sealed version's underlying inputs and the
	// exclusions it relied upon stay exactly as they were at seal time
	b, err := rv.s.GetBatch(m.BatchID)
	if err != nil {
		return nil, err
	}
	if b.IsSealed() {
		return nil, model.ErrSealed
	}
	next := model.MeasExcluded
	if contaminated {
		next = model.MeasContaminated
	}
	if !m.CanTransitionTo(next) {
		return nil, fmt.Errorf("%w: measurement %d status %s cannot -> %s", model.ErrConflict, id, m.Status, next)
	}
	if err := rv.s.UpdateMeasurementStatus(id, next); err != nil {
		return nil, err
	}
	if err := rv.s.CreateExclusion(m.BatchID, id, reason); err != nil {
		return nil, err
	}
	m.Status = next
	return m, nil
}
