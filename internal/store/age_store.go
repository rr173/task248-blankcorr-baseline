package store

import (
	"database/sql"
	"fmt"
	"time"

	"task248-blankcorr/internal/model"
)

// CreateAgeResult inserts an age result and assigns its id.
func (s *Store) CreateAgeResult(a *model.AgeResult) error {
	if a == nil {
		return fmt.Errorf("%w: nil age result", model.ErrInvalid)
	}
	createdAt := a.CreatedAt
	if createdAt == 0 {
		createdAt = time.Now().UTC().UnixMilli()
	}
	res, err := s.db.Exec(
		`INSERT INTO age_results(relation_id, sample_id, batch_id, corrected_ratio, corrected_unc,
			age_value, age_unc, age_low, age_high, expected_low, expected_high, anomaly_flag, reason, created_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		a.RelationID, a.SampleID, a.BatchID, a.CorrectedRatio, a.CorrectedUnc,
		a.AgeValue, a.AgeUnc, a.AgeLow, a.AgeHigh, a.ExpectedLow, a.ExpectedHigh,
		boolToInt(a.AnomalyFlag), a.Reason, createdAt)
	if err != nil {
		return fmt.Errorf("insert age result: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("age result last insert id: %w", err)
	}
	a.ID = id
	a.CreatedAt = createdAt
	return nil
}

// PurgeAgeResults removes every age result of a batch. Used before a
// recomputation so that stale results from a previous match are not kept.
func (s *Store) PurgeAgeResults(batchID int64) error {
	if _, err := s.db.Exec(`DELETE FROM age_results WHERE batch_id = ?`, batchID); err != nil {
		return fmt.Errorf("purge age results: %w", err)
	}
	return nil
}

// GetAgeResult loads an age result by id.
func (s *Store) GetAgeResult(id int64) (*model.AgeResult, error) {
	row := s.db.QueryRow(
		`SELECT id, relation_id, sample_id, batch_id, corrected_ratio, corrected_unc,
			age_value, age_unc, age_low, age_high, expected_low, expected_high, anomaly_flag, reason, created_at
		 FROM age_results WHERE id = ?`, id)
	a := &model.AgeResult{}
	var flag int
	var createdMs int64
	err := row.Scan(&a.ID, &a.RelationID, &a.SampleID, &a.BatchID, &a.CorrectedRatio, &a.CorrectedUnc,
		&a.AgeValue, &a.AgeUnc, &a.AgeLow, &a.AgeHigh, &a.ExpectedLow, &a.ExpectedHigh, &flag, &a.Reason, &createdMs)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan age result: %w", err)
	}
	a.AnomalyFlag = flag != 0
	return a, nil
}

// ListAgeResults returns the age results of a batch ordered by sample then id.
func (s *Store) ListAgeResults(batchID int64) ([]*model.AgeResult, error) {
	rows, err := s.db.Query(
		`SELECT id, relation_id, sample_id, batch_id, corrected_ratio, corrected_unc,
			age_value, age_unc, age_low, age_high, expected_low, expected_high, anomaly_flag, reason, created_at
		 FROM age_results WHERE batch_id = ? ORDER BY sample_id, id`, batchID)
	if err != nil {
		return nil, fmt.Errorf("query age results: %w", err)
	}
	defer rows.Close()
	var out []*model.AgeResult
	for rows.Next() {
		a := &model.AgeResult{}
		var flag int
		var createdMs int64
		if err := rows.Scan(&a.ID, &a.RelationID, &a.SampleID, &a.BatchID, &a.CorrectedRatio, &a.CorrectedUnc,
			&a.AgeValue, &a.AgeUnc, &a.AgeLow, &a.AgeHigh, &a.ExpectedLow, &a.ExpectedHigh, &flag, &a.Reason, &createdMs); err != nil {
			return nil, fmt.Errorf("scan age result row: %w", err)
		}
		a.AnomalyFlag = flag != 0
		out = append(out, a)
	}
	return out, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
