package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

// MeasurementKind distinguishes the role of a measurement run.
const (
	KindSample   = "sample"   // a real unknown specimen
	KindBlank    = "blank"    // a reagent/process blank (should be ~zero)
	KindStandard = "standard" // a material with a certified ratio
)

// MeasurementStatus enumerates the lifecycle of a measurement run (测次).
//
//	raw        原始    just imported, not yet vetted
//	usable     可用    accepted for correction
//	contaminated 污染  blank/std found compromised; excluded from matching
//	expired     过期    outside the valid time window for its batch
//	excluded    排除    manually removed from consideration
const (
	MeasRaw          = "raw"
	MeasUsable       = "usable"
	MeasContaminated = "contaminated"
	MeasExpired      = "expired"
	MeasExcluded     = "excluded"
)

// eligibleForMatch reports whether a measurement may be used as a correction
// item (blank or standard) when building a correction relation.
func measEligibleForMatch(status string) bool {
	return status == MeasRaw || status == MeasUsable
}

// Measurement is a single isotope-ratio determination.
type Measurement struct {
	ID             int64
	BatchID        int64
	Kind           string
	Material       string
	MeasuredAt     time.Time
	Ratio          float64 // primary isotope ratio
	RatioUnc       float64 // 1-sigma uncertainty of Ratio
	CertifiedRatio float64 // only for standards: the certified/reference ratio
	SecondaryJSON  string  // optional additional ratios (opaque JSON)
	Fingerprint    string  // idempotency key
	Status         string
	CreatedAt      time.Time
}

// MeasurementInput is the user-supplied payload when importing a measurement.
type MeasurementInput struct {
	BatchID        int64
	Kind           string
	Material       string
	MeasuredAt     time.Time
	Ratio          float64
	RatioUnc       float64
	CertifiedRatio float64
	SecondaryJSON  string
}

// Validate checks the input fields.
func (m MeasurementInput) Validate() error {
	if m.BatchID <= 0 {
		return NewValidationError("batch_id", m.BatchID, "must be > 0")
	}
	switch m.Kind {
	case KindSample, KindBlank, KindStandard:
	default:
		return NewValidationError("kind", m.Kind, "must be sample|blank|standard")
	}
	if m.Ratio < 0 {
		return NewValidationError("ratio", m.Ratio, "isotope ratio must be >= 0")
	}
	if m.RatioUnc < 0 {
		return NewValidationError("ratio_unc", m.RatioUnc, "uncertainty must be >= 0")
	}
	if m.Kind == KindStandard && m.CertifiedRatio <= 0 {
		return NewValidationError("certified_ratio", m.CertifiedRatio,
			"standards require a positive certified ratio")
	}
	if m.MeasuredAt.IsZero() {
		return NewValidationError("measured_at", m.MeasuredAt, "must be set")
	}
	return nil
}

// BuildMeasurement converts validated input into a stored Measurement with a
// computed fingerprint. The fingerprint makes re-imports idempotent so the
// same run is never counted twice.
func BuildMeasurement(in MeasurementInput) *Measurement {
	return &Measurement{
		BatchID:        in.BatchID,
		Kind:           in.Kind,
		Material:       in.Material,
		MeasuredAt:     in.MeasuredAt,
		Ratio:          in.Ratio,
		RatioUnc:       in.RatioUnc,
		CertifiedRatio: in.CertifiedRatio,
		SecondaryJSON:  in.SecondaryJSON,
		Fingerprint:    MeasurementFingerprint(in.BatchID, in.Kind, in.Material, in.MeasuredAt, in.Ratio),
		Status:         MeasRaw,
		CreatedAt:      time.Now().UTC(),
	}
}

// MeasurementFingerprint produces a stable idempotency key from the immutable
// attributes of a measurement. Two identical imports yield the same key.
func MeasurementFingerprint(batchID int64, kind, material string, measuredAt time.Time, ratio float64) string {
	h := sha256.New()
	fmt.Fprintf(h, "%d|%s|%s|%d|%s", batchID, kind, material, measuredAt.UnixNano(), strconv.FormatFloat(ratio, 'g', -1, 64))
	return hex.EncodeToString(h.Sum(nil))[:32]
}

// IsEligibleForMatch reports whether this measurement can serve as a blank or
// standard correction item.
func (m *Measurement) IsEligibleForMatch() bool { return measEligibleForMatch(m.Status) }

// IsBlank reports the measurement kind.
func (m *Measurement) IsBlank() bool { return m.Kind == KindBlank }

// IsStandard reports the measurement kind.
func (m *Measurement) IsStandard() bool { return m.Kind == KindStandard }

// IsSample reports the measurement kind.
func (m *Measurement) IsSample() bool { return m.Kind == KindSample }

// CanTransitionTo enforces the measurement status machine. Contaminated /
// expired / excluded are terminal for matching purposes: once flagged the
// measurement leaves the candidate pool and cannot return to raw/usable.
func (m *Measurement) CanTransitionTo(next string) bool {
	switch m.Status {
	case MeasRaw:
		return next == MeasUsable || next == MeasContaminated || next == MeasExpired || next == MeasExcluded
	case MeasUsable:
		return next == MeasContaminated || next == MeasExpired || next == MeasExcluded
	default:
		// contaminated / expired / excluded are terminal
		return false
	}
}
