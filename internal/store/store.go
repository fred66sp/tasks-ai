// Package store keeps the project's state in one SQLite file. It holds no rules:
// the core imposes them.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	_ "modernc.org/sqlite" // el motor de SQLite, escrito en Go puro
)

// ErrNotFound says that nothing is stored under the key that was asked for.
var ErrNotFound = errors.New("no hay nada guardado con esa clave")

// driverName is the engine this package registers with database/sql.
const driverName = "sqlite"

// migrations are the schema changes, in order. Each one is applied once and only
// adds: a local product cannot lose data when it updates.
var migrations = []migration{
	{
		version: 1,
		statements: []string{
			`CREATE TABLE meta (
				key   TEXT PRIMARY KEY,
				value TEXT NOT NULL
			)`,
			`CREATE TABLE tasks (
				id           INTEGER PRIMARY KEY AUTOINCREMENT,
				title        TEXT    NOT NULL,
				description  TEXT,
				status       TEXT    NOT NULL DEFAULT 'open',
				priority     INTEGER NOT NULL DEFAULT 2,
				assignee     TEXT,
				parent_id    INTEGER REFERENCES tasks (id),
				created_at   TEXT    NOT NULL,
				updated_at   TEXT    NOT NULL,
				close_reason TEXT,
				notes        TEXT
			)`,
		},
	},
}

// migration is one schema change, applied as a whole.
type migration struct {
	version    int
	statements []string
}

// Store is one project's data.
type Store struct {
	db *sql.DB
}

// Open opens the database at path, creating it when it is not there, and applies
// the migrations it still owes.
func Open(ctx context.Context, path string) (*Store, error) {
	db, err := sql.Open(driverName, dataSourceName(path))
	if err != nil {
		return nil, fmt.Errorf("abriendo la base %s: %w", path, err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("abriendo la base %s: %w", path, err)
	}

	instance := &Store{db: db}
	if err := instance.migrate(ctx); err != nil {
		_ = db.Close()

		return nil, err
	}

	return instance, nil
}

// Close lets the database go.
func (s *Store) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("cerrando la base: %w", err)
	}

	return nil
}

// dataSourceName builds the connection string: the base waits for whoever is
// writing instead of failing, keeps a write-ahead log so a reader can read while
// the product writes, and honours the references between tables.
func dataSourceName(path string) string {
	return "file:" + path +
		"?_pragma=busy_timeout(5000)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(on)"
}

// SchemaVersion is the schema version the database is at.
func (s *Store) SchemaVersion(ctx context.Context) (int, error) {
	empty, err := s.isEmpty(ctx)
	if err != nil || empty {
		return 0, err
	}

	var value string

	err = s.db.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = 'schema_version'`).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}

	if err != nil {
		return 0, fmt.Errorf("leyendo la versión del esquema: %w", err)
	}

	version, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("la versión del esquema guardada (%q) no es un número: %w", value, err)
	}

	return version, nil
}

// RecordedPrefix is the prefix the base was composed with, empty when the base
// has none yet.
func (s *Store) RecordedPrefix(ctx context.Context) (string, error) {
	var value string

	err := s.db.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = 'prefix'`).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("leyendo el prefijo de la base: %w", err)
	}

	return value, nil
}

// SetPrefix records the prefix the base is composed with.
func (s *Store) SetPrefix(ctx context.Context, prefix string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO meta (key, value) VALUES ('prefix', ?)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`, prefix)
	if err != nil {
		return fmt.Errorf("anotando el prefijo %q: %w", prefix, err)
	}

	return nil
}

// migrate applies the migrations the database still owes, each one in its own
// transaction and recording the version it reached.
func (s *Store) migrate(ctx context.Context) error {
	version, err := s.SchemaVersion(ctx)
	if err != nil {
		return err
	}

	for _, change := range migrations {
		if change.version <= version {
			continue
		}

		if err := s.apply(ctx, change); err != nil {
			return err
		}
	}

	return nil
}

// apply writes one schema change and its version, all or nothing.
func (s *Store) apply(ctx context.Context, change migration) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("abriendo la transacción del esquema: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	for _, statement := range change.statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("aplicando el esquema %d: %w", change.version, err)
		}
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO meta (key, value) VALUES ('schema_version', ?)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`, strconv.Itoa(change.version))
	if err != nil {
		return fmt.Errorf("anotando el esquema %d: %w", change.version, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("cerrando la transacción del esquema: %w", err)
	}

	return nil
}

// isEmpty reports whether the database has no schema yet.
func (s *Store) isEmpty(ctx context.Context) (bool, error) {
	var tables int

	err := s.db.QueryRowContext(ctx,
		`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'meta'`,
	).Scan(&tables)
	if err != nil {
		return false, fmt.Errorf("mirando el esquema de la base: %w", err)
	}

	return tables == 0, nil
}
