package match

import (
	"fmt"
	"time"

	"task248-blankcorr/internal/correct"
	"task248-blankcorr/internal/model"
	"task248-blankcorr/internal/store"
)

// Matcher builds correction relations for a batch.
type Matcher struct {
	s *store.Store
}

// NewMatcher constructs a Matcher over the given store.
func NewMatcher(s *store.Store) *Matcher { return &Matcher{s: s} }

// Match rebuilds the candidate correction relations for every eligible sample
// in the batch. Each sample is bound to:
//   - its nearest eligible blank within window (blank subtraction term), and
//   - the batch-wide drift model evaluated at the sample time (recovery term).
//
// Before building the new set, any existing correction relations for the batch
// are purged so that re-matching after excluding a bad blank yields a fresh,
// consistent candidate set. Match refuses to run on a sealed batch.
func (m *Matcher) Match(batchID int64, window time.Duration) ([]*model.CorrectionRelation, error) {
	b, err := m.s.GetBatch(batchID)
	if err != nil {
		return nil, err
	}
	if b.IsSealed() {
		return nil, model.ErrSealed
	}
	samples, err := m.s.ListMeasurements(batchID, model.KindSample, []string{model.MeasRaw, model.MeasUsable})
	if err != nil {
		return nil, fmt.Errorf("list samples: %w", err)
	}
	blanks, err := m.s.ListMeasurements(batchID, model.KindBlank, []string{model.MeasRaw, model.MeasUsable})
	if err != nil {
		return nil, fmt.Errorf("list blanks: %w", err)
	}
	standards, err := m.s.ListMeasurements(batchID, model.KindStandard, []string{model.MeasRaw, model.MeasUsable})
	if err != nil {
		return nil, fmt.Errorf("list standards: %w", err)
	}
	dm, err := correct.FitDrift(standards)
	if err != nil {
		return nil, fmt.Errorf("fit drift: %w", err)
	}
	if err := m.s.PurgeAgeResults(batchID); err != nil {
		return nil, fmt.Errorf("purge old age results: %w", err)
	}
	if err := m.s.PurgeCorrections(batchID); err != nil {
		return nil, fmt.Errorf("purge old corrections: %w", err)
	}

	out := make([]*model.CorrectionRelation, 0, len(samples))
	for _, smp := range samples {
		c := &model.CorrectionRelation{
			BatchID:  batchID,
			SampleID: smp.ID,
			Status:   model.CorrCandidate,
		}
		if blank := NearestWithin(blanks, smp.MeasuredAt, window); blank != nil {
			c.BlankID = blank.ID
			c.BValue = blank.Ratio
			c.BUnc = blank.RatioUnc
		}
		factor, unc := dm.At(smp.MeasuredAt)
		c.DriftFactor = factor
		c.DriftUnc = unc
		if err := m.s.CreateCorrection(c); err != nil {
			return nil, fmt.Errorf("create correction: %w", err)
		}
		out = append(out, c)
	}

	// batch has enough data to move out of the receiving state
	if b.Status == model.BatchReceiving {
		if err := m.s.UpdateBatchStatus(batchID, model.BatchPending); err != nil {
			return nil, fmt.Errorf("advance batch: %w", err)
		}
	}
	return out, nil
}
