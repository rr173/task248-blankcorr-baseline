package store

import (
	"fmt"
	"time"
)

// CreateExclusion records that a measurement has been excluded from matching
// (for example a contaminated blank), together with the reason.
func (s *Store) CreateExclusion(batchID, measurementID int64, reason string) error {
	_, err := s.db.Exec(
		`INSERT INTO exclusions(batch_id, measurement_id, reason, created_at) VALUES(?,?,?,?)`,
		batchID, measurementID, reason, time.Now().UTC().UnixMilli())
	if err != nil {
		return fmt.Errorf("insert exclusion: %w", err)
	}
	return nil
}

// ListExclusions returns the exclusions recorded for a batch.
func (s *Store) ListExclusions(batchID int64) ([]Exclusion, error) {
	rows, err := s.db.Query(
		`SELECT id, batch_id, measurement_id, reason, created_at FROM exclusions WHERE batch_id = ? ORDER BY id`,
		batchID)
	if err != nil {
		return nil, fmt.Errorf("query exclusions: %w", err)
	}
	defer rows.Close()
	var out []Exclusion
	for rows.Next() {
		e := Exclusion{}
		var createdMs int64
		if err := rows.Scan(&e.ID, &e.BatchID, &e.MeasurementID, &e.Reason, &createdMs); err != nil {
			return nil, fmt.Errorf("scan exclusion: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Exclusion is a record of a removed measurement.
type Exclusion struct {
	ID            int64
	BatchID       int64
	MeasurementID int64
	Reason        string
	CreatedAt     time.Time
}
