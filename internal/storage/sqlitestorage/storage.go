// Package sqlitestorage provides a session.StorageBackend backed by SQLite.
package sqlitestorage

import (
	"context"
	"database/sql"
	"time"
)

// Storage is a session.StorageBackend backed by a SQLite database.
type Storage struct {
	db *sql.DB
}

// New opens (or creates) a SQLite database at dbPath and returns a Storage.
func New(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	_, err = db.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS sessions (
		"key"   TEXT PRIMARY KEY,
		value   BLOB NOT NULL,
		expiry  INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil {
		return nil, err
	}

	return &Storage{db: db}, nil
}

// Get returns the value for key, or nil if it is missing or expired.
func (s *Storage) Get(key string) ([]byte, error) {
	var value []byte

	var expiry int64

	err := s.db.QueryRowContext(
		context.Background(),
		`SELECT value, expiry FROM sessions WHERE "key" = ?`, key,
	).Scan(&value, &expiry)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	if expiry != 0 && time.Now().UnixNano() > expiry {
		_ = s.Delete(key) //nolint:errcheck // best-effort cleanup
		return nil, nil
	}

	return value, nil
}

// Set stores value under key with the given expiration duration.
// A zero exp means the entry never expires.
func (s *Storage) Set(key string, val []byte, exp time.Duration) error {
	var expiry int64

	if exp > 0 {
		expiry = time.Now().Add(exp).UnixNano()
	}

	_, err := s.db.ExecContext(
		context.Background(),
		`INSERT INTO sessions ("key", value, expiry) VALUES (?, ?, ?)
		 ON CONFLICT("key") DO UPDATE SET value = excluded.value, expiry = excluded.expiry`,
		key, val, expiry,
	)

	return err
}

// Delete removes the entry for key.
func (s *Storage) Delete(key string) error {
	_, err := s.db.ExecContext(
		context.Background(),
		`DELETE FROM sessions WHERE "key" = ?`, key,
	)

	return err
}
