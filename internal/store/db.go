// Package store implements SQLite persistence for the blank-correction
// service. All timestamps are stored as integer unix-milliseconds so that
// time-window math and ordering are exact and timezone-independent.
package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Store wraps a *sql.DB and provides all persistence operations. The
// underlying SQLite database is opened with a single connection so that
// concurrent writers serialize cleanly (SQLite is a single-writer engine).
type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the SQLite database at path and runs the
// schema migration. The caller is responsible for calling Close.
func Open(path string) (*Store, error) {
	sqldb, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}
	sqldb.SetMaxOpenConns(1)
	sqldb.SetMaxIdleConns(1)
	if _, err := sqldb.Exec("PRAGMA foreign_keys=ON"); err != nil {
		_ = sqldb.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	s := &Store{db: sqldb}
	if err := s.Migrate(); err != nil {
		_ = sqldb.Close()
		return nil, err
	}
	return s, nil
}

// DB exposes the underlying handle (used by smoke-test restart checks).
func (s *Store) DB() *sql.DB { return s.db }

// Close releases the database handle.
func (s *Store) Close() error { return s.db.Close() }

// Migrate creates all tables if they do not yet exist. It is idempotent.
func (s *Store) Migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS batches (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			system_type TEXT NOT NULL,
			lambda REAL NOT NULL,
			r0 REAL NOT NULL,
			expected_low REAL NOT NULL DEFAULT 0,
			expected_high REAL NOT NULL DEFAULT 0,
			status TEXT NOT NULL CHECK (status IN ('receiving','pending','needs_review','published','sealed')),
			created_at INTEGER NOT NULL,
			sealed_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS measurements (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL REFERENCES batches(id),
			kind TEXT NOT NULL CHECK (kind IN ('sample','blank','standard')),
			material TEXT NOT NULL DEFAULT '',
			measured_at INTEGER NOT NULL,
			ratio REAL NOT NULL,
			ratio_unc REAL NOT NULL,
			certified_ratio REAL NOT NULL DEFAULT 0,
			secondary_json TEXT NOT NULL DEFAULT '',
			fingerprint TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL CHECK (status IN ('raw','usable','contaminated','expired','excluded')),
			created_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_meas_batch ON measurements(batch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_meas_kind ON measurements(kind)`,
		`CREATE TABLE IF NOT EXISTS correction_relations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL REFERENCES batches(id),
			sample_id INTEGER NOT NULL REFERENCES measurements(id),
			blank_id INTEGER NOT NULL DEFAULT 0,
			standard_id INTEGER NOT NULL DEFAULT 0,
			b_value REAL NOT NULL DEFAULT 0,
			b_unc REAL NOT NULL DEFAULT 0,
			drift_factor REAL NOT NULL DEFAULT 0,
			drift_unc REAL NOT NULL DEFAULT 0,
			status TEXT NOT NULL CHECK (status IN ('candidate','valid','conflict','confirmed')),
			created_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_corr_batch ON correction_relations(batch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_corr_sample ON correction_relations(sample_id)`,
		`CREATE TABLE IF NOT EXISTS age_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			relation_id INTEGER NOT NULL REFERENCES correction_relations(id),
			sample_id INTEGER NOT NULL REFERENCES measurements(id),
			batch_id INTEGER NOT NULL REFERENCES batches(id),
			corrected_ratio REAL NOT NULL DEFAULT 0,
			corrected_unc REAL NOT NULL DEFAULT 0,
			age_value REAL NOT NULL DEFAULT 0,
			age_unc REAL NOT NULL DEFAULT 0,
			age_low REAL NOT NULL DEFAULT 0,
			age_high REAL NOT NULL DEFAULT 0,
			expected_low REAL NOT NULL DEFAULT 0,
			expected_high REAL NOT NULL DEFAULT 0,
			anomaly_flag INTEGER NOT NULL DEFAULT 0,
			reason TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_age_batch ON age_results(batch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_age_sample ON age_results(sample_id)`,
		`CREATE TABLE IF NOT EXISTS age_versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL REFERENCES batches(id),
			name TEXT NOT NULL,
			status TEXT NOT NULL CHECK (status IN ('draft','published','superseded','sealed')),
			note TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL,
			sealed_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ver_batch ON age_versions(batch_id)`,
		`CREATE TABLE IF NOT EXISTS version_entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			version_id INTEGER NOT NULL REFERENCES age_versions(id),
			result_id INTEGER NOT NULL REFERENCES age_results(id),
			UNIQUE(version_id, result_id)
		)`,
		`CREATE TABLE IF NOT EXISTS exclusions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL REFERENCES batches(id),
			measurement_id INTEGER NOT NULL REFERENCES measurements(id),
			reason TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL
		)`,
	}
	for _, st := range stmts {
		if _, err := s.db.Exec(st); err != nil {
			return fmt.Errorf("migrate failed: %w", err)
		}
	}
	return nil
}
