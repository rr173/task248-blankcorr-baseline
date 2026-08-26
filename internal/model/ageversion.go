package model

import "time"

// AgeVersionStatus enumerates the lifecycle of an age version (年代版本).
//
//	draft      草稿   results collected but not yet published
//	published  发布   shared; still mutable until sealed
//	superseded 替代   replaced by a newer version
//	sealed     封存   immutable snapshot (freezes blank/standard choices)
const (
	VersionDraft      = "draft"
	VersionPublished  = "published"
	VersionSuperseded = "superseded"
	VersionSealed     = "sealed"
)

// AgeVersion is a named, freezable collection of age results for a batch.
type AgeVersion struct {
	ID        int64
	BatchID   int64
	Name      string
	Status    string
	Note      string
	CreatedAt time.Time
	SealedAt  time.Time
}

// VersionEntry links an age result into a version.
type VersionEntry struct {
	ID        int64
	VersionID int64
	ResultID  int64
}

// NewAgeVersion builds a draft version.
func NewAgeVersion(batchID int64, name, note string) *AgeVersion {
	return &AgeVersion{
		BatchID:   batchID,
		Name:      name,
		Status:    VersionDraft,
		Note:      note,
		CreatedAt: time.Now().UTC(),
	}
}

// IsMutable reports whether results can still be added or the version can be
// edited. Sealed and superseded versions are immutable.
func (v *AgeVersion) IsMutable() bool {
	return v.Status == VersionDraft || v.Status == VersionPublished
}

// CanTransitionTo enforces the version state machine.
func (v *AgeVersion) CanTransitionTo(next string) bool {
	switch v.Status {
	case VersionDraft:
		return next == VersionPublished || next == VersionSealed
	case VersionPublished:
		return next == VersionSealed || next == VersionSuperseded
	default:
		return false
	}
}

// Transition applies a validated version status change.
func (v *AgeVersion) Transition(next string) error {
	if !v.CanTransitionTo(next) {
		return ErrConflict
	}
	v.Status = next
	if next == VersionSealed {
		v.SealedAt = time.Now().UTC()
	}
	return nil
}
