package store

import (
	"database/sql"
	"fmt"
	"time"

	"task248-blankcorr/internal/model"
)

// CreateMeasurement inserts a measurement. If an identical fingerprint
// already exists the call is idempotent: the existing row is returned
// together with model.ErrDuplicate so callers can skip the duplicate.
func (s *Store) CreateMeasurement(m *model.Measurement) error {
	if m == nil {
		return fmt.Errorf("%w: nil measurement", model.ErrInvalid)
	}
	// idempotency: skip re-import of the same run
	var existingID int64
	err := s.db.QueryRow(`SELECT id FROM measurements WHERE fingerprint = ?`, m.Fingerprint).Scan(&existingID)
	if err == nil {
		m.ID = existingID
		return model.ErrDuplicate
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("lookup fingerprint: %w", err)
	}
	res, err := s.db.Exec(
		`INSERT INTO measurements(batch_id, kind, material, measured_at, ratio, ratio_unc,
			certified_ratio, secondary_json, fingerprint, status, created_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		m.BatchID, m.Kind, m.Material, m.MeasuredAt.UnixMilli(), m.Ratio, m.RatioUnc,
		m.CertifiedRatio, m.SecondaryJSON, m.Fingerprint, m.Status, time.Now().UTC().UnixMilli())
	if err != nil {
		return fmt.Errorf("insert measurement: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("measurement last insert id: %w", err)
	}
	m.ID = id
	return nil
}

// GetMeasurement loads a measurement by id.
func (s *Store) GetMeasurement(id int64) (*model.Measurement, error) {
	row := s.db.QueryRow(
		`SELECT id, batch_id, kind, material, measured_at, ratio, ratio_unc,
		 certified_ratio, secondary_json, fingerprint, status, created_at
		 FROM measurements WHERE id = ?`, id)
	m, err := scanMeasurement(row)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	return m, err
}

// ListMeasurements returns measurements of a batch, optionally filtered by
// kind and/or a set of statuses. Empty filters mean "no constraint".
func (s *Store) ListMeasurements(batchID int64, kind string, statuses []string) ([]*model.Measurement, error) {
	q := `SELECT id, batch_id, kind, material, measured_at, ratio, ratio_unc,
		 certified_ratio, secondary_json, fingerprint, status, created_at
		 FROM measurements WHERE batch_id = ?`
	args := []interface{}{batchID}
	if kind != "" {
		q += ` AND kind = ?`
		args = append(args, kind)
	}
	if len(statuses) > 0 {
		q += ` AND status IN (`
		for i, st := range statuses {
			if i > 0 {
				q += `,`
			}
			q += `?`
			args = append(args, st)
		}
		q += `)`
	}
	q += ` ORDER BY measured_at, id`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("query measurements: %w", err)
	}
	defer rows.Close()
	var out []*model.Measurement
	for rows.Next() {
		m, err := scanMeasurement(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// UpdateMeasurementStatus changes the lifecycle status of a measurement.
func (s *Store) UpdateMeasurementStatus(id int64, status string) error {
	res, err := s.db.Exec(`UPDATE measurements SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return fmt.Errorf("update measurement status: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrNotFound
	}
	return nil
}

func scanMeasurement(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.Measurement, error) {
	m := &model.Measurement{}
	var measuredMs, createdMs int64
	err := scanner.Scan(&m.ID, &m.BatchID, &m.Kind, &m.Material, &measuredMs, &m.Ratio,
		&m.RatioUnc, &m.CertifiedRatio, &m.SecondaryJSON, &m.Fingerprint, &m.Status, &createdMs)
	if err != nil {
		return nil, fmt.Errorf("scan measurement: %w", err)
	}
	m.MeasuredAt = time.UnixMilli(measuredMs).UTC()
	m.CreatedAt = time.UnixMilli(createdMs).UTC()
	return m, nil
}
