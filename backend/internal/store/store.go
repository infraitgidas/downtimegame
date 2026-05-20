// Package store provides SQLite database access for the Downtime Game.
// Uses modernc.org/sqlite — pure Go, no CGo required.
package store

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Store wraps the SQLite database connection and provides data access methods.
type Store struct {
	DB *sql.DB
}

// New opens (or creates) the SQLite database and runs migrations.
func New(dbPath string) (*Store, error) {
	if dbPath == "" {
		dbPath = "downtime-game.db"
	}

	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}

	// SQLite-specific settings
	db.SetMaxOpenConns(1) // SQLite doesn't support concurrent writes

	s := &Store{DB: db}

	if err := s.migrate(); err != nil {
		return nil, err
	}

	log.Printf("Store: SQLite database ready at %s", dbPath)
	return s, nil
}

// migrate runs schema migrations using embedded SQL.
func (s *Store) migrate() error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, q := range migrations {
		if _, err := tx.Exec(q); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.DB.Close()
}
