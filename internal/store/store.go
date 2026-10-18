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
var ErrNotFound = errors.New("nothing is stored under that key")

// Las superficies que abren la base y los recursos de los que un cambio puede
// hablar. Viajan dentro de las filas de cambio, así que sus valores son datos,
// no mensajes.
const (
	OriginCLI    = "cli"
	OriginServer = "server"
)

// Los recursos de los que un aviso de cambio habla. Viajan dentro de las
// filas de cambio, así que sus valores son datos, no mensajes.
const (
	ResourceTask = "task"
	ResourceTag  = "tag"
)

// Las acciones que un aviso declara: lo que le pasó al recurso. Viajan dentro
// de las filas de cambio como datos, no como mensajes: cada superficie las
// traduce al vocabulario que muestra.
const (
	ActionCreate = "create"
	ActionUpdate = "update"
	ActionClose  = "close"
	ActionDelete = "delete"
)

// ErrClosed dice que la tarea que se quiso cerrar ya está cerrada.
var ErrClosed = errors.New("the task is already closed")

// ErrOpenSubtasks dice que la tarea que se quiso cerrar tiene subtareas que no
// lo están.
var ErrOpenSubtasks = errors.New("the task has open subtasks")

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
	{
		version: 3,
		statements: []string{
			`CREATE TABLE tags (
				id          INTEGER PRIMARY KEY AUTOINCREMENT,
				name        TEXT    NOT NULL UNIQUE,
				color       TEXT    NOT NULL,
				description TEXT
			)`,
			`CREATE TABLE task_tags (
				task_id INTEGER NOT NULL REFERENCES tasks (id),
				tag_id  INTEGER NOT NULL REFERENCES tags (id),
				PRIMARY KEY (task_id, tag_id)
			)`,
		},
	},
	{
		// Los avisos de cambio: una fila por cambio, escrita en la misma
		// transacción que el cambio. La identidad del recurso viaja por su
		// número interno, nunca por la referencia escrita, de modo que un
		// cambio de prefijo no deja las filas mintiendo.
		version: 4,
		statements: []string{
			`CREATE TABLE changes (
				id          INTEGER PRIMARY KEY AUTOINCREMENT,
				resource    TEXT    NOT NULL,
				resource_id INTEGER NOT NULL,
				origin      TEXT    NOT NULL,
				happened_at TEXT    NOT NULL
			)`,
		},
	},
	{
		// Cada aviso dice lo que pasó, escrito con el cambio. Comparar listados
		// no puede decirlo: varias escrituras de una misma tarea dentro de un
		// vistazo dan un solo par de listas, y un comentario no cambia nada en la
		// tarea que comenta. Las filas de antes se quedan sin acción, que es lo
		// que son: un cambio cuyo modo no se anotó.
		version: 5,
		statements: []string{
			`ALTER TABLE changes ADD COLUMN action TEXT NOT NULL DEFAULT ''`,
		},
	},
	{
		// Las consultas de detalle buscan por estas dos columnas. El índice de la
		// clave primaria de dependencies ya cubre task_id, pero no blocker_id.
		version: 6,
		statements: []string{
			`CREATE INDEX comments_task_id_idx ON comments (task_id)`,
			`CREATE INDEX dependencies_blocker_id_idx ON dependencies (blocker_id)`,
		},
	},
}

// migration is one schema change, applied as a whole.
type migration struct {
	version    int
	statements []string
}

// Store es una base de un proyecto.
type Store struct {
	db     *sql.DB
	origin string
}

// bootstrapAttempts is how many times Open may retry the whole bootstrap when
// the base is brand new: another process may be creating it at the same time.
const bootstrapAttempts = 5

// bootstrapWait is the ceiling for the whole bootstrap. Adding up the wait of
// every attempt would leave a lock nobody is going to release — a long
// transaction, a stuck process — looking like a hung command for half a minute.
const bootstrapWait = 2 * time.Second

// Open abre la base de datos en path, la crea si no está y le aplica las
// migraciones que debe. Habla en nombre de la línea de comandos: es la única
// superficie que abre la base sin decir quién es, porque la instalación corre
// con ella.
func Open(ctx context.Context, path string) (*Store, error) {
	return OpenAs(ctx, path, OriginCLI)
}

// OpenAs abre la base de datos diciendo qué superficie pide los cambios: cada
// aviso lo lleva, de modo que una sesión sabe si un cambio viene de otra.
func OpenAs(ctx context.Context, path, origin string) (*Store, error) {
	// Los ajustes aceptan rutas en subcarpetas del proyecto y nadie más crea
	// esa carpeta: sin esto, abrir falla porque SQLite no la crea por su cuenta.
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("creating the folder of the database %s: %w", path, err)
	}

	limit := time.Now().Add(bootstrapWait)

	for attempt := 1; ; attempt++ {
		instance, err := openOnce(ctx, path, origin)
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
			return nil, fmt.Errorf("opening the database %s: %w", path, ctx.Err())
		case <-time.After(time.Duration(attempt) * 25 * time.Millisecond):
		}
	}
}

// openOnce connects, pings and migrates once; the caller owns the store it
// returns, and every error leaves nothing open behind.
func openOnce(ctx context.Context, path, origin string) (*Store, error) {
	db, err := sql.Open(driverName, dataSourceName(path))
	if err != nil {
		return nil, fmt.Errorf("opening the database %s: %w", path, err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("opening the database %s: %w", path, err)
	}

	instance := &Store{db: db, origin: origin}
	if err := instance.migrate(ctx); err != nil {
		_ = db.Close()

		return nil, err
	}

	return instance, nil
}

// isCode reports whether the error is one of SQLite's codes, its extended forms
// included: the driver answers with the extended one, whose low byte is the code
// the manual names, and comparing against the plain code would miss it.
func isCode(err error, code int) bool {
	var sqliteErr *sqlite.Error

	return errors.As(err, &sqliteErr) && sqliteErr.Code()&0xFF == code
}

// isBusy reports whether the error is SQLite's occupied code, the only case
// Open retries.
func isBusy(err error) bool {
	return isCode(err, sqlite3.SQLITE_BUSY)
}

// Close lets the database go.
func (s *Store) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("closing the database: %w", err)
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
		return 0, fmt.Errorf("looking at the schema of the database: %w", err)
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
		return 0, fmt.Errorf("reading the schema version: %w", err)
	}

	version, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("the stored schema version (%q) is not a number: %w", value, err)
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
		return "", fmt.Errorf("reading the prefix of the database: %w", err)
	}

	return value, nil
}

// SetPrefix records the prefix the base is composed with.
func (s *Store) SetPrefix(ctx context.Context, prefix string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO meta (key, value) VALUES ('prefix', ?)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`, prefix)
	if err != nil {
		return fmt.Errorf("recording the prefix %q: %w", prefix, err)
	}

	return nil
}

// Change es un aviso de cambio tal como está guardado: el recurso que cambió
// por su clase y su identidad interna, lo que le pasó, quién lo pidió y cuándo
// ocurrió. Action está vacío en las filas escritas antes de que cada aviso
// llevara su acción.
type Change struct {
	ID         int64
	Resource   string
	ResourceID int64
	Origin     string
	Action     string
	HappenedAt time.Time
}

// Changes lee los avisos guardados a partir del id dado, en el orden en que
// ocurrieron: quien lee continúa donde se quedó pasando el último que vio.
func (s *Store) Changes(ctx context.Context, after int64) ([]Change, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, resource, resource_id, origin, action, happened_at FROM changes WHERE id > ? ORDER BY id`, after)
	if err != nil {
		return nil, fmt.Errorf("reading the changes: %w", err)
	}

	defer func() { _ = rows.Close() }()

	changes := []Change{}

	for rows.Next() {
		var (
			change     Change
			happenedAt string
		)

		if err := rows.Scan(&change.ID, &change.Resource, &change.ResourceID, &change.Origin, &change.Action, &happenedAt); err != nil {
			return nil, fmt.Errorf("reading the changes: %w", err)
		}

		if change.HappenedAt, err = time.Parse(timeLayout, happenedAt); err != nil {
			return nil, fmt.Errorf("the date of change %d cannot be read: %w", change.ID, err)
		}

		changes = append(changes, change)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading the changes: %w", err)
	}

	return changes, nil
}

// LastChangeID devuelve la identidad del último cambio anotado, que es donde
// empieza quien abre un canal nuevo: el aviso cuenta lo que pasa mientras mira,
// no el historial que ya tiene delante.
func (s *Store) LastChangeID(ctx context.Context) (int64, error) {
	var last int64

	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0) FROM changes`).Scan(&last); err != nil {
		return 0, fmt.Errorf("reading the latest change: %w", err)
	}

	return last, nil
}

// changeWriter escribe una fila de cambio: la transacción que está haciendo el
// cambio es quien la lleva.
type changeWriter interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// recordChange deja una fila de cambio dentro de la transacción que está
// haciendo el cambio: el aviso y el cambio se escriben juntos, o no se escribe
// nada.
func recordChange(ctx context.Context, within changeWriter, origin, resource string, resourceID int64, action string) error {
	now := time.Now().UTC().Truncate(time.Second)

	if _, err := within.ExecContext(ctx,
		`INSERT INTO changes (resource, resource_id, origin, action, happened_at) VALUES (?, ?, ?, ?, ?)`,
		resource, resourceID, origin, action, now.Format(timeLayout)); err != nil {
		return fmt.Errorf("recording the change of %s %d: %w", resource, resourceID, err)
	}

	return nil
}

// inPlaceholders deja la lista de ? de una cláusula IN para n valores.
func inPlaceholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

// int64Args puts the numbers in the argument list a clause IN asks for.
func int64Args(ids []int64) []any {
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	return args
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
		return fmt.Errorf("opening the schema transaction: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	version, err := schemaVersion(ctx, tx)
	if err != nil {
		return fmt.Errorf("applying schema %d: %w", change.version, err)
	}

	if change.version <= version {
		return nil
	}

	for _, statement := range change.statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("applying schema %d: %w", change.version, err)
		}
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO meta (key, value) VALUES ('schema_version', ?)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`, strconv.Itoa(change.version))
	if err != nil {
		return fmt.Errorf("recording schema %d: %w", change.version, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("closing the schema transaction: %w", err)
	}

	return nil
}
