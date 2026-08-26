package store

import (
	"database/sql"
	"fmt"
	"time"

	"task248-blankcorr/internal/model"
)

// CreateBatch inserts a new batch and assigns its ID.
func (s *Store) CreateBatch(b *model.Batch) error {
	if b == nil {
		return fmt.Errorf("%w: nil batch", model.ErrInvalid)
	}
	res, err := s.db.Exec(
		`INSERT INTO batches(name, system_type, lambda, r0, expected_low, expected_high, status, created_at, sealed_at)
		 VALUES(?,?,?,?,?,?,?,?,0)`,
		b.Name, b.SystemType, b.Lambda, b.R0, b.ExpectedLow, b.ExpectedHigh, b.Status, b.CreatedAt.UnixMilli())
	if err != nil {
		return fmt.Errorf("insert batch: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("batch last insert id: %w", err)
	}
	b.ID = id
	return nil
}

// GetBatch loads a batch by id.
func (s *Store) GetBatch(id int64) (*model.Batch, error) {
	row := s.db.QueryRow(
		`SELECT id, name, system_type, lambda, r0, expected_low, expected_high, status, created_at, sealed_at
		 FROM batches WHERE id = ?`, id)
	b := &model.Batch{}
	var createdMs, sealedMs int64
	err := row.Scan(&b.ID, &b.Name, &b.SystemType, &b.Lambda, &b.R0,
		&b.ExpectedLow, &b.ExpectedHigh, &b.Status, &createdMs, &sealedMs)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan batch: %w", err)
	}
	b.CreatedAt = time.UnixMilli(createdMs).UTC()
	b.SealedAt = time.UnixMilli(sealedMs).UTC()
	return b, nil
}

// ListBatches returns all batches ordered by id.
func (s *Store) ListBatches() ([]*model.Batch, error) {
	rows, err := s.db.Query(
		`SELECT id, name, system_type, lambda, r0, expected_low, expected_high, status, created_at, sealed_at
		 FROM batches ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("query batches: %w", err)
	}
	defer rows.Close()
	var out []*model.Batch
	for rows.Next() {
		b := &model.Batch{}
		var createdMs, sealedMs int64
		if err := rows.Scan(&b.ID, &b.Name, &b.SystemType, &b.Lambda, &b.R0,
			&b.ExpectedLow, &b.ExpectedHigh, &b.Status, &createdMs, &sealedMs); err != nil {
			return nil, fmt.Errorf("scan batch row: %w", err)
		}
		b.CreatedAt = time.UnixMilli(createdMs).UTC()
		b.SealedAt = time.UnixMilli(sealedMs).UTC()
		out = append(out, b)
	}
	return out, rows.Err()
}

// UpdateBatchStatus advances the batch status (forward only, enforced by the
// model helper).
func (s *Store) UpdateBatchStatus(id int64, status string) error {
	res, err := s.db.Exec(`UPDATE batches SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return fmt.Errorf("update batch status: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// SetBatchSealed transitions a batch to the immutable sealed state.
func (s *Store) SetBatchSealed(id int64) error {
	now := time.Now().UTC().UnixMilli()
	res, err := s.db.Exec(
		`UPDATE batches SET status = 'sealed', sealed_at = ? WHERE id = ?`, now, id)
	if err != nil {
		return fmt.Errorf("seal batch: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrNotFound
	}
	return nil
}
