package model

import "fmt"

// CorrectionStatus enumerates the lifecycle of a correction relation
// (校正关系), the binding between a sample and its matched blank/standard.
//
//	candidate 候选  produced by the matcher, not yet reviewed
//	valid     有效  inputs accepted, maths consistent
//	conflict  冲突  drift/blank inputs disagree with expectations
//	confirmed 确认  accepted and used to compute the age result
const (
	CorrCandidate = "candidate"
	CorrValid     = "valid"
	CorrConflict  = "conflict"
	CorrConfirmed = "confirmed"
)

// CorrectionRelation binds a sample measurement to the blank and standard that
// were matched to it by time window, together with the resulting correction
// parameters. It is the unit of "which blank/standard did we use".
type CorrectionRelation struct {
	ID           int64
	BatchID      int64
	SampleID     int64
	BlankID      int64
	StandardID   int64
	BValue       float64 // matched blank ratio
	BUnc         float64 // matched blank uncertainty
	DriftFactor  float64 // recovery ratio r(t) at sample time
	DriftUnc     float64 // uncertainty of drift factor
	Status       string
	CreatedAt    int64
}

// CanTransitionTo enforces the correction-relation state machine.
func (c *CorrectionRelation) CanTransitionTo(next string) bool {
	switch c.Status {
	case CorrCandidate:
		return next == CorrValid || next == CorrConflict || next == CorrConfirmed
	case CorrValid:
		return next == CorrConflict || next == CorrConfirmed
	case CorrConflict:
		return next == CorrValid || next == CorrConfirmed
	case CorrConfirmed:
		// confirmed is terminal until the owning version logic reopens it
		return false
	default:
		return false
	}
}

// Transition validates and applies a status change.
func (c *CorrectionRelation) Transition(next string) error {
	if !c.CanTransitionTo(next) {
		return fmt.Errorf("%w: correction %d %s -> %s", ErrConflict, c.ID, c.Status, next)
	}
	c.Status = next
	return nil
}
