package store

import (
	"database/sql"
	"fmt"
	"time"

	"task248-blankcorr/internal/model"
)

// CreateCorrection inserts a correction relation and assigns its id.
func (s *Store) CreateCorrection(c *model.CorrectionRelation) error {
	if c == nil {
		return fmt.Errorf("%w: nil correction", model.ErrInvalid)
	}
	res, err := s.db.Exec(
		`INSERT INTO correction_relations(batch_id, sample_id, blank_id, standard_id,
			b_value, b_unc, drift_factor, drift_unc, status, created_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?)`,
		c.BatchID, c.SampleID, c.BlankID, c.StandardID,
		c.BValue, c.BUnc, c.DriftFactor, c.DriftUnc, c.Status, time.Now().UTC().UnixMilli())
	if err != nil {
		return fmt.Errorf("insert correction: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("correction last insert id: %w", err)
	}
	c.ID = id
	c.CreatedAt = time.Now().UTC().UnixMilli()
	return nil
}

// GetCorrection loads a correction relation by id.
func (s *Store) GetCorrection(id int64) (*model.CorrectionRelation, error) {
	row := s.db.QueryRow(
		`SELECT id, batch_id, sample_id, blank_id, standard_id, b_value, b_unc,
		 drift_factor, drift_unc, status, created_at
		 FROM correction_relations WHERE id = ?`, id)
	c := &model.CorrectionRelation{}
	var createdMs int64
	err := row.Scan(&c.ID, &c.BatchID, &c.SampleID, &c.BlankID, &c.StandardID,
		&c.BValue, &c.BUnc, &c.DriftFactor, &c.DriftUnc, &c.Status, &createdMs)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan correction: %w", err)
	}
	c.CreatedAt = createdMs
	return c, nil
}

// ListCorrections returns the correction relations of a batch.
func (s *Store) ListCorrections(batchID int64) ([]*model.CorrectionRelation, error) {
	rows, err := s.db.Query(
		`SELECT id, batch_id, sample_id, blank_id, standard_id, b_value, b_unc,
		 drift_factor, drift_unc, status, created_at
		 FROM correction_relations WHERE batch_id = ? ORDER BY id`, batchID)
	if err != nil {
		return nil, fmt.Errorf("query corrections: %w", err)
	}
	defer rows.Close()
	var out []*model.CorrectionRelation
	for rows.Next() {
		c := &model.CorrectionRelation{}
		var createdMs int64
		if err := rows.Scan(&c.ID, &c.BatchID, &c.SampleID, &c.BlankID, &c.StandardID,
			&c.BValue, &c.BUnc, &c.DriftFactor, &c.DriftUnc, &c.Status, &createdMs); err != nil {
			return nil, fmt.Errorf("scan correction row: %w", err)
		}
		c.CreatedAt = createdMs
		out = append(out, c)
	}
	return out, rows.Err()
}

// PurgeCorrections removes every correction relation of a batch. It is used
// when re-matching (the candidate set is recomputed from scratch). It must
// only be called while the batch is still mutable.
func (s *Store) PurgeCorrections(batchID int64) error {
	if _, err := s.db.Exec(`DELETE FROM correction_relations WHERE batch_id = ?`, batchID); err != nil {
		return fmt.Errorf("purge corrections: %w", err)
	}
	return nil
}

// UpdateCorrectionStatus transitions a correction relation to a new status.
func (s *Store) UpdateCorrectionStatus(id int64, status string) error {
	res, err := s.db.Exec(`UPDATE correction_relations SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return fmt.Errorf("update correction status: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrNotFound
	}
	return nil
}
