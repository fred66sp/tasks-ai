// Package store keeps the project's state in one SQLite file. It holds no rules:
// the core imposes them.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	sqlite "modernc.org/sqlite" // el motor de SQLite, escrito en Go puro
	sqlite3 "modernc.org/sqlite/lib"
)

// ErrNotFound dice que no hay nada guardado con la clave que se pidió.
var ErrNotFound = errors.New("no hay nada guardado con esa clave")

// ErrClosed dice que la tarea que se quiso cerrar ya está cerrada.
var ErrClosed = errors.New("la tarea ya está cerrada")

// ErrOpenSubtasks dice que la tarea que se quiso cerrar tiene subtareas que no
// lo están.
var ErrOpenSubtasks = errors.New("la tarea tiene subtareas abiertas")

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
	{
		version: 2,
		statements: []string{
			`CREATE TABLE dependencies (
				task_id    INTEGER NOT NULL REFERENCES tasks (id),
				blocker_id INTEGER NOT NULL REFERENCES tasks (id),
				PRIMARY KEY (task_id, blocker_id)
			)`,
			`CREATE TABLE comments (
				id         INTEGER PRIMARY KEY AUTOINCREMENT,
				task_id    INTEGER NOT NULL REFERENCES tasks (id),
				body       TEXT    NOT NULL,
				created_at TEXT    NOT NULL
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

// bootstrapAttempts is how many times Open may retry the whole bootstrap when
// the base is brand new: another process may be creating it at the same time.
const bootstrapAttempts = 5

// bootstrapWait is the ceiling for the whole bootstrap. Adding up the wait of
// every attempt would leave a lock nobody is going to release — a long
// transaction, a stuck process — looking like a hung command for half a minute.
const bootstrapWait = 2 * time.Second

// Open opens the database at path, creating it when it is not there, and applies
// the migrations it still owes.
func Open(ctx context.Context, path string) (*Store, error) {
	// Los ajustes aceptan rutas en subcarpetas del proyecto y nadie más crea
	// esa carpeta: sin esto, abrir falla porque SQLite no la crea por su cuenta.
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("creando la carpeta de la base %s: %w", path, err)
	}

	limit := time.Now().Add(bootstrapWait)

	for attempt := 1; ; attempt++ {
		instance, err := openOnce(ctx, path)
		if err == nil {
			return instance, nil
		}

		// Dos procesos pueden crear la base a la vez: el pragma que la pasa a
		// WAL devuelve SQLITE_BUSY sin esperar. Se reintenta solo ese caso, con
		// pausas cortas y contadas, y nunca más allá del techo total: un
		// bloqueo que no se suelta tiene que fallar, no colgar el comando.
		if attempt >= bootstrapAttempts || !isBusy(err) || time.Now().After(limit) {
			return nil, err
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("abriendo la base %s: %w", path, ctx.Err())
		case <-time.After(time.Duration(attempt) * 25 * time.Millisecond):
		}
	}
}

// openOnce connects, pings and migrates once; the caller owns the store it
// returns, and every error leaves nothing open behind.
func openOnce(ctx context.Context, path string) (*Store, error) {
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

// isBusy reports whether the error is SQLite's occupied code, the only case
// Open retries.
func isBusy(err error) bool {
	var sqliteErr *sqlite.Error

	return errors.As(err, &sqliteErr) && sqliteErr.Code() == sqlite3.SQLITE_BUSY
}

// Close lets the database go.
func (s *Store) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("cerrando la base: %w", err)
	}

	return nil
}

// dataSourceName builds the connection string: the transaction that migrates
// takes the write lock up front, the base waits a second for whoever is writing
// before giving up, a write-ahead log lets a reader read while the product
// writes, and the references between tables are honoured.
func dataSourceName(path string) string {
	// SQLite corta el nombre del fichero de una URI en la primera ? o # y lee
	// %HH como escape: sin esto, esos caracteres en la ruta sacan la base de
	// su carpeta. El % se escapa primero para no re-escapar lo ya escapado.
	escaped := strings.ReplaceAll(path, "%", "%25")
	escaped = strings.ReplaceAll(escaped, "?", "%3F")
	escaped = strings.ReplaceAll(escaped, "#", "%23")

	return "file:" + escaped +
		"?_txlock=immediate" +
		"&_pragma=busy_timeout(1000)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(on)"
}

// SchemaVersion is the schema version the database is at.
func (s *Store) SchemaVersion(ctx context.Context) (int, error) {
	return schemaVersion(ctx, s.db)
}

// rowQuerier reads one row: enough for both the live database and a
// transaction, which read the schema version the same way.
type rowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// schemaVersion reads the schema version from wherever a row can be queried,
// 0 when there is no schema yet.
func schemaVersion(ctx context.Context, q rowQuerier) (int, error) {
	var tables int

	err := q.QueryRowContext(ctx,
		`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'meta'`,
	).Scan(&tables)
	if err != nil {
		return 0, fmt.Errorf("mirando el esquema de la base: %w", err)
	}

	if tables == 0 {
		return 0, nil
	}

	var value string

	err = q.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = 'schema_version'`).Scan(&value)
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
// transaction and recording the version it reached. The version is re-read
// inside each transaction: with the write lock already taken, whoever arrives
// second sees the work the first one committed and skips it instead of
// creating the same tables twice.
func (s *Store) migrate(ctx context.Context) error {
	for _, change := range migrations {
		if err := s.apply(ctx, change); err != nil {
			return err
		}
	}

	return nil
}

// apply writes one schema change and its version, all or nothing, unless the
// version is already there when the write lock is taken.
func (s *Store) apply(ctx context.Context, change migration) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("abriendo la transacción del esquema: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	version, err := schemaVersion(ctx, tx)
	if err != nil {
		return fmt.Errorf("aplicando el esquema %d: %w", change.version, err)
	}

	if change.version <= version {
		return nil
	}

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
