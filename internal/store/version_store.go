package store

import (
	"database/sql"
	"fmt"
	"time"

	"task248-blankcorr/internal/model"
)

// CreateVersion inserts a new age version (draft) and assigns its id.
func (s *Store) CreateVersion(v *model.AgeVersion) error {
	if v == nil {
		return fmt.Errorf("%w: nil version", model.ErrInvalid)
	}
	res, err := s.db.Exec(
		`INSERT INTO age_versions(batch_id, name, status, note, created_at, sealed_at)
		 VALUES(?,?,?,?,?,0)`,
		v.BatchID, v.Name, v.Status, v.Note, v.CreatedAt.UnixMilli())
	if err != nil {
		return fmt.Errorf("insert version: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("version last insert id: %w", err)
	}
	v.ID = id
	return nil
}

// GetVersion loads a version by id.
func (s *Store) GetVersion(id int64) (*model.AgeVersion, error) {
	row := s.db.QueryRow(
		`SELECT id, batch_id, name, status, note, created_at, sealed_at
		 FROM age_versions WHERE id = ?`, id)
	v := &model.AgeVersion{}
	var createdMs, sealedMs int64
	err := row.Scan(&v.ID, &v.BatchID, &v.Name, &v.Status, &v.Note, &createdMs, &sealedMs)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan version: %w", err)
	}
	v.CreatedAt = time.UnixMilli(createdMs).UTC()
	v.SealedAt = time.UnixMilli(sealedMs).UTC()
	return v, nil
}

// ListVersions returns all versions of a batch ordered by id.
func (s *Store) ListVersions(batchID int64) ([]*model.AgeVersion, error) {
	rows, err := s.db.Query(
		`SELECT id, batch_id, name, status, note, created_at, sealed_at
		 FROM age_versions WHERE batch_id = ? ORDER BY id`, batchID)
	if err != nil {
		return nil, fmt.Errorf("query versions: %w", err)
	}
	defer rows.Close()
	var out []*model.AgeVersion
	for rows.Next() {
		v := &model.AgeVersion{}
		var createdMs, sealedMs int64
		if err := rows.Scan(&v.ID, &v.BatchID, &v.Name, &v.Status, &v.Note, &createdMs, &sealedMs); err != nil {
			return nil, fmt.Errorf("scan version row: %w", err)
		}
		v.CreatedAt = time.UnixMilli(createdMs).UTC()
		v.SealedAt = time.UnixMilli(sealedMs).UTC()
		out = append(out, v)
	}
	return out, rows.Err()
}

// UpdateVersionStatus transitions a version to a new status.
func (s *Store) UpdateVersionStatus(id int64, status string) error {
	var sealedClause string
	if status == model.VersionSealed {
		sealedClause = fmt.Sprintf(", sealed_at = %d", time.Now().UTC().UnixMilli())
	}
	res, err := s.db.Exec(
		fmt.Sprintf(`UPDATE age_versions SET status = ?%s WHERE id = ?`, sealedClause), status, id)
	if err != nil {
		return fmt.Errorf("update version status: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// AddVersionEntry links an age result into a version (idempotent).
func (s *Store) AddVersionEntry(versionID, resultID int64) error {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO version_entries(version_id, result_id) VALUES(?,?)`, versionID, resultID)
	if err != nil {
		return fmt.Errorf("insert version entry: %w", err)
	}
	return nil
}

// ListVersionEntries returns the result ids linked to a version.
func (s *Store) ListVersionEntries(versionID int64) ([]int64, error) {
	rows, err := s.db.Query(`SELECT result_id FROM version_entries WHERE version_id = ? ORDER BY result_id`, versionID)
	if err != nil {
		return nil, fmt.Errorf("query version entries: %w", err)
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var rid int64
		if err := rows.Scan(&rid); err != nil {
			return nil, fmt.Errorf("scan version entry: %w", err)
		}
		out = append(out, rid)
	}
	return out, rows.Err()
}
