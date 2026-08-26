package model

import (
	"fmt"
	"time"
)

// BatchStatus enumerates the lifecycle of a measurement batch (测定批次).
//
//	receiving   接收中   batch created, measurements still being imported
//	pending     待校正   import complete, awaiting correction
//	needs_review 需复核   at least one result is anomalous / flagged
//	published   已发布   an age version has been published
//	sealed      封存     a version is sealed, batch becomes immutable
const (
	BatchReceiving   = "receiving"
	BatchPending     = "pending"
	BatchNeedsReview = "needs_review"
	BatchPublished   = "published"
	BatchSealed      = "sealed"
)

// batchStatusOrder defines the forward progression used by CanAdvanceBatch.
var batchStatusOrder = []string{
	BatchReceiving, BatchPending, BatchNeedsReview, BatchPublished, BatchSealed,
}

// Batch groups a set of measurements that share a decay system and its
// constants. The constants lambda and r0 are required to convert a corrected
// isotope ratio into an age.
type Batch struct {
	ID            int64
	Name          string
	SystemType    string // e.g. "radiocarbon", "u-series", "generic"
	Lambda        float64 // decay constant (1/year)
	R0            float64 // initial isotope ratio (dimensionless)
	ExpectedLow   float64 // optional expected age lower bound
	ExpectedHigh  float64 // optional expected age upper bound
	Status        string
	CreatedAt     time.Time
	SealedAt      time.Time
}

// NewBatch constructs and validates a batch.
func NewBatch(name, systemType string, lambda, r0, expectedLow, expectedHigh float64) (*Batch, error) {
	b := &Batch{
		Name:          name,
		SystemType:    systemType,
		Lambda:        lambda,
		R0:            r0,
		ExpectedLow:   expectedLow,
		ExpectedHigh:  expectedHigh,
		Status:        BatchReceiving,
		CreatedAt:     time.Now().UTC(),
	}
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return b, nil
}

// Validate enforces the invariants of a batch.
func (b *Batch) Validate() error {
	if b.Name == "" {
		return NewValidationError("name", b.Name, "must not be empty")
	}
	if b.SystemType == "" {
		return NewValidationError("system_type", b.SystemType, "must not be empty")
	}
	if b.Lambda <= 0 {
		return NewValidationError("lambda", b.Lambda, "decay constant must be > 0")
	}
	if b.R0 <= 0 {
		return NewValidationError("r0", b.R0, "initial ratio must be > 0")
	}
	if b.ExpectedLow > 0 && b.ExpectedHigh > 0 && b.ExpectedLow >= b.ExpectedHigh {
		return NewValidationError("expected_interval",
			fmt.Sprintf("[%g,%g]", b.ExpectedLow, b.ExpectedHigh),
			"expected_low must be < expected_high")
	}
	return nil
}

// IsSealed reports whether the batch is immutable.
func (b *Batch) IsSealed() bool { return b.Status == BatchSealed }

// CanAdvanceBatch reports whether status can move forward to next (or to the
// given target when target is non-empty) following the defined order.
func CanAdvanceBatch(current, target string) bool {
	if target == "" {
		return true
	}
	ci, ok1 := indexOf(batchStatusOrder, current)
	ti, ok2 := indexOf(batchStatusOrder, target)
	if !ok1 || !ok2 {
		return false
	}
	// forward only, and stay allowed
	return ti >= ci
}

func indexOf(s []string, v string) (int, bool) {
	for i, x := range s {
		if x == v {
			return i, true
		}
	}
	return 0, false
}
