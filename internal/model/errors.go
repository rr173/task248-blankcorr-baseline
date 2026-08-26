// Package model holds the domain entities, state machines and validation
// rules for the geochemistry age-determination blank-correction service.
//
// The service helps geochemistry experimenters decide whether an anomalous
// isotope age is a real sample difference or an artefact of blank selection
// or instrument drift. The core domain objects are measurement runs
// (samples, blanks, standards), the correction relations that bind a sample
// to its matched blank and standard, the computed age results, and the
// published/sealed age versions.
package model

import (
	"errors"
	"fmt"
)

// Common domain errors. They are returned by the model, store and business
// packages so that the HTTP layer can map them to status codes.
var (
	// ErrNotFound is returned when an entity id does not exist.
	ErrNotFound = errors.New("entity not found")
	// ErrInvalid is returned when a field fails domain validation.
	ErrInvalid = errors.New("invalid domain value")
	// ErrConflict is returned when a state transition is not permitted.
	ErrConflict = errors.New("state transition not permitted")
	// ErrSealed is returned when an operation touches a sealed entity.
	ErrSealed = errors.New("entity is sealed and immutable")
	// ErrDuplicate is returned on an idempotency (fingerprint) collision.
	ErrDuplicate = errors.New("duplicate measurement (idempotency key)")
)

// ValidationError wraps a field-level validation failure with context.
type ValidationError struct {
	Field   string
	Value   interface{}
	Reason  string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: field %q value %v: %s", ErrInvalid, e.Field, e.Value, e.Reason)
}

func (e *ValidationError) Unwrap() error { return ErrInvalid }

// NewValidationError builds a ValidationError.
func NewValidationError(field string, value interface{}, reason string) *ValidationError {
	return &ValidationError{Field: field, Value: value, Reason: reason}
}
