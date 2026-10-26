package store_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"tasks-ai/internal/store"
	"testing"
	"time"
)

func TestOpen_createsTheSchemaAndAnnotatesItsVersion(t *testing.T) {
	t.Parallel()

	version, err := openStore(t).SchemaVersion(t.Context())
	if err != nil {
		t.Fatalf("SchemaVersion devolvió %v", err)
	}

	if version != 7 {
		t.Errorf("una base nueva queda en el esquema %d, se esperaba el 7", version)
	}
}

func TestOpen_doesNotRepeatTheSchemaItAlreadyHas(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "tasks.db")

	first, err := store.Open(t.Context(), path)
	if err != nil {
		t.Fatalf("abriendo la base por primera vez: %v", err)
	}

	created, err := first.CreateTask(t.Context(), newTask("Se conserva"), nil)
	if err != nil {
		t.Fatalf("escribiendo antes de reabrir: %v", err)
	}

	if err := first.Close(); err != nil {
		t.Fatalf("cerrando la base: %v", err)
	}

	again, err := store.Open(t.Context(), path)
	if err != nil {
		t.Fatalf("abriendo la base por segunda vez: %v", err)
	}

	t.Cleanup(func() { _ = again.Close() })

	version, err := again.SchemaVersion(t.Context())
	if err != nil {
		t.Fatalf("SchemaVersion devolvió %v", err)
	}

	if version != 7 {
		t.Errorf("reabrir la base la dejó en el esquema %d, se esperaba el 7", version)
	}

	read, err := again.Task(t.Context(), created.ID)
	if err != nil || read.Title != created.Title {
		t.Errorf("reabrir la base cambió la tarea a %+v (%v), se esperaba %+v", read, err, created)
	}
}

//nolint:gocyclo // migration test checks schema, data, and both indexes together
func TestOpen_migratesIndexesWithoutLosingCommentsOrDependencies(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "tasks.db")

	base, err := store.Open(t.Context(), path)
	if err != nil {
		t.Fatalf("abriendo la base: %v", err)
	}

	task, err := base.CreateTask(t.Context(), newTask("Con comentario"), nil)
	if err != nil {
		t.Fatalf("creando la tarea: %v", err)
	}

	blocker, err := base.CreateTask(t.Context(), newTask("Bloqueante"), nil)
	if err != nil {
		t.Fatalf("creando el bloqueante: %v", err)
	}

	if err := base.AddBlock(t.Context(), task.ID, blocker.ID); err != nil {
		t.Fatalf("añadiendo la dependencia: %v", err)
	}

	if _, err := base.AddComment(t.Context(), task.ID, "conservar"); err != nil {
		t.Fatalf("añadiendo el comentario: %v", err)
	}

	if err := base.Close(); err != nil {
		t.Fatalf("cerrando la base: %v", err)
	}

	// Simula una base de esquema 5 que todavía no conoce los índices nuevos.
	rawBase(t, path, "DROP TABLE adrs")
	rawBase(t, path, "DROP INDEX comments_task_id_idx")
	rawBase(t, path, "DROP INDEX dependencies_blocker_id_idx")
	rawBase(t, path, "UPDATE meta SET value = '5' WHERE key = 'schema_version'")

	again, err := store.Open(t.Context(), path)
	if err != nil {
		t.Fatalf("migrando la base: %v", err)
	}
	defer again.Close() //nolint:errcheck // las aserciones de abajo ya deciden el resultado

	if version, err := again.SchemaVersion(t.Context()); err != nil || version != 7 {
		t.Errorf("la base migrada quedó en el esquema %d (%v), se esperaba el 7", version, err)
	}

	comments, err := again.Comments(t.Context(), task.ID)
	if err != nil || len(comments) != 1 || comments[0].Body != "conservar" {
		t.Errorf("los comentarios tras migrar son %+v (%v), se esperaba conservarlos", comments, err)
	}

	blocked, err := again.HasOpenBlocker(t.Context(), task.ID)
	if err != nil || !blocked {
		t.Errorf("la dependencia tras migrar es %t (%v), se esperaba abierta", blocked, err)
	}

	for _, index := range []string{"comments_task_id_idx", "dependencies_blocker_id_idx"} {
		if !indexExists(t, path, index) {
			t.Errorf("la migración no creó el índice %q", index)
		}
	}
}

func indexExists(t *testing.T, path, wanted string) bool {
	t.Helper()

	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("abriendo la base para mirar sus índices: %v", err)
	}
	defer raw.Close() //nolint:errcheck // la aserción del índice ya decide el resultado

	var found string

	err = raw.QueryRowContext(t.Context(),
		"SELECT name FROM sqlite_master WHERE type = 'index' AND name = ?", wanted,
	).Scan(&found)

	return err == nil && found == wanted
}

func TestOpen_twoStoresOpeningABrandNewBaseAtTheSameTimeBothSucceed(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "tasks.db")

	// Ocho como ocho procesos que arrancan a la vez sobre una base sin crear.
	const openers = 8

	results := make(chan error, openers)
	for range openers {
		go func() {
			base, err := store.Open(t.Context(), path)
			if err != nil {
				results <- err
				return
			}

			results <- base.Close()
		}()
	}

	for range openers {
		if err := <-results; err != nil {
			t.Errorf("abriendo la base nueva a la vez desde dos procesos: %v", err)
		}
	}

	after, err := store.Open(t.Context(), path)
	if err != nil {
		t.Fatalf("abriendo la base tras la carrera: %v", err)
	}

	defer after.Close() //nolint:errcheck // la aserción de abajo ya decide el resultado

	ctx := t.Context()

	version, err := after.SchemaVersion(ctx)
	if err != nil {
		t.Fatalf("SchemaVersion devolvió %v", err)
	}

	if version != 7 {
		t.Errorf("la base quedó en el esquema %d, se esperaba el 7", version)
	}

	if err := after.SetPrefix(ctx, "px"); err != nil {
		t.Errorf("la base quedó inservible tras la carrera: %v", err)
	}
}

func TestOpen_keepsTheBaseInsideAFolderWhoseNameHasAnInterrogation(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()

	firstPath := filepath.Join(parent, "a?b", "tasks.db")
	twinPath := filepath.Join(parent, "a?c", "tasks.db")

	first, err := store.Open(t.Context(), firstPath)
	if err != nil {
		t.Fatalf("abriendo la base en una ruta con interrogación: %v", err)
	}

	defer first.Close() //nolint:errcheck // la aserción de abajo ya decide el resultado

	if _, err := os.Stat(firstPath); err != nil {
		t.Errorf("la base no está en su carpeta %s: %v", firstPath, err)
	}

	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatalf("leyendo la carpeta padre: %v", err)
	}

	for _, entry := range entries {
		if entry.Name() != "a?b" && entry.Name() != "a?c" {
			t.Errorf("la carpeta padre contiene %q, solo se esperaban a?b y a?c", entry.Name())
		}
	}

	twin, err := store.Open(t.Context(), twinPath)
	if err != nil {
		t.Fatalf("abriendo la base gemela: %v", err)
	}

	defer twin.Close() //nolint:errcheck // la aserción de abajo ya decide el resultado

	if _, err := os.Stat(twinPath); err != nil {
		t.Errorf("la base gemela no está en su carpeta %s: %v", twinPath, err)
	}

	ctx := t.Context()

	if err := first.SetPrefix(ctx, "primera"); err != nil {
		t.Fatalf("anotando el prefijo de la primera base: %v", err)
	}

	recorded, err := twin.RecordedPrefix(ctx)
	if err != nil {
		t.Fatalf("leyendo el prefijo de la base gemela: %v", err)
	}

	if recorded != "" {
		t.Errorf("las dos bases comparten datos: la gemela trae el prefijo %q, se esperaba ninguno", recorded)
	}
}

func TestOpen_createsTheFolderTheBaseLivesIn(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "datos", "tasks.db")

	base, err := store.Open(t.Context(), path)
	if err != nil {
		t.Fatalf("abriendo la base en una subcarpeta aún sin crear: %v", err)
	}

	defer base.Close() //nolint:errcheck // la aserción de abajo ya decide el resultado

	if _, err := os.Stat(path); err != nil {
		t.Errorf("la base no está en la subcarpeta %s: %v", path, err)
	}
}

func TestOpen_saysSoWhenTheFolderOfTheBaseCannotBeCreated(t *testing.T) {
	t.Parallel()

	// Un fichero ocupa el sitio de la carpeta: nadie puede crearla y Open
	// debe rendirse enseguida, sin gastar las pausas de reintento que son
	// solo para el caso en que otro proceso está creando la base.
	folder := filepath.Join(t.TempDir(), "datos")
	if err := os.WriteFile(folder, []byte("aquí hay un fichero, no una carpeta"), 0o600); err != nil {
		t.Fatalf("preparando el fichero que estorba: %v", err)
	}

	started := time.Now()

	_, err := store.Open(t.Context(), filepath.Join(folder, "tasks.db"))
	if err == nil {
		t.Fatal("abriendo la base con una carpeta imposible: no falló")
	}

	if !strings.Contains(err.Error(), "creating the folder of the database") {
		t.Errorf("el error no dice que la carpeta no pudo crearse: %v", err)
	}

	// Las cinco pausas de reintento suman 250 ms: fallar antes demuestra
	// que este caso no las gastó.
	if elapsed := time.Since(started); elapsed >= 250*time.Millisecond {
		t.Errorf("Open tardó %v en fallar: gastó las pausas de reintento", elapsed)
	}
}

func TestOpen_saysSoWhenTheFileIsNotADatabase(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "tasks.db")
	if err := os.WriteFile(path, []byte("esto es un cuaderno, no una base"), 0o600); err != nil {
		t.Fatalf("escribiendo el texto que estorba: %v", err)
	}

	_, err := store.Open(t.Context(), path)
	if err == nil {
		t.Fatal("abriendo un fichero que no es una base: no falló")
	}

	if !strings.Contains(err.Error(), path) {
		t.Errorf("el error no menciona la base %s: %v", path, err)
	}

	if !strings.Contains(err.Error(), "not a database") {
		t.Errorf("el error no dice que el fichero no es una base de datos: %v", err)
	}
}

func TestOpen_saysSoWhenTheRecordedSchemaVersionIsNotANumber(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "tasks.db")

	// Otra herramienta dejó en la base una versión del esquema sin sentido.
	rawBase(t, path, "CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT NOT NULL)")
	rawBase(t, path, "INSERT INTO meta (key, value) VALUES ('schema_version', 'próximamente')")

	_, err := store.Open(t.Context(), path)
	if err == nil {
		t.Fatal("abriendo una base con una versión del esquema sin sentido: no falló")
	}

	if !strings.Contains(err.Error(), `"próximamente"`) || !strings.Contains(err.Error(), "is not a number") {
		t.Errorf("el error no señala la versión guardada: %v", err)
	}
}

func TestCreateTask_givesEachTaskItsOwnIdentity(t *testing.T) {
	t.Parallel()

	base := openStore(t)
	ctx := t.Context()

	first, err := base.CreateTask(ctx, newTask("Primera"), nil)
	if err != nil {
		t.Fatalf("guardando la primera tarea: %v", err)
	}

	second, err := base.CreateTask(ctx, newTask("Segunda"), nil)
	if err != nil {
		t.Fatalf("guardando la segunda tarea: %v", err)
	}

	if first.ID != 1 || second.ID != 2 {
		t.Errorf("las identidades fueron %d y %d, se esperaban 1 y 2", first.ID, second.ID)
	}

	if first.CreatedAt.IsZero() || first.CreatedAt.Location().String() != "UTC" {
		t.Errorf("la tarea nace con la fecha %v, se esperaba una fecha en UTC", first.CreatedAt)
	}
}

func TestTask_returnsWhatWasStored(t *testing.T) {
	t.Parallel()

	description := "con detalle"
	created, read := storedTask(t)

	if read.Title != created.Title || read.Status != created.Status || read.Priority != created.Priority {
		t.Errorf("la tarea leída es %+v, se esperaba %+v", read, created)
	}

	if read.Description == nil || *read.Description != description {
		t.Errorf("la descripción leída es %v, se esperaba %q", read.Description, description)
	}
}

func TestTask_returnsWithoutValueWhatWasNotStored(t *testing.T) {
	t.Parallel()

	created, read := storedTask(t)

	if read.Notes != nil || read.Assignee != nil || read.ParentID != nil || read.CloseReason != nil {
		t.Errorf("lo que se guardó sin valor volvió con valor: %+v", read)
	}

	if !read.CreatedAt.Equal(created.CreatedAt) || !read.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("las fechas cambiaron al guardar y leer: %+v frente a %+v", read, created)
	}
}

func TestTask_saysWhenThereIsNothingUnderThatIdentity(t *testing.T) {
	t.Parallel()

	if _, err := openStore(t).Task(t.Context(), 404); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("Task devolvió %v, se esperaba ErrNotFound", err)
	}
}

func TestTask_saysSoWhenTheStoredDatesCannotBeRead(t *testing.T) {
	t.Parallel()

	today := time.Now().UTC().Format(time.RFC3339Nano)

	for _, caso := range []struct {
		name      string
		createdAt string
		updatedAt string
		fragment  string
	}{
		{name: "la fecha de creación", createdAt: "ayer", updatedAt: today, fragment: "the creation date of task 1"},
		{name: "la fecha de cambio", createdAt: today, updatedAt: "el martes pasado", fragment: "the change date of task 1"},
	} {
		t.Run(caso.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "tasks.db")

			base, err := store.Open(t.Context(), path)
			if err != nil {
				t.Fatalf("abriendo la base de prueba: %v", err)
			}

			if err := base.Close(); err != nil {
				t.Fatalf("cerrando la base: %v", err)
			}

			// Otra herramienta escribió la tarea con una fecha sin sentido.
			rawBase(t, path,
				`INSERT INTO tasks (title, status, created_at, updated_at) VALUES ('Rota', 'open', ?, ?)`,
				caso.createdAt, caso.updatedAt,
			)

			base, err = store.Open(t.Context(), path)
			if err != nil {
				t.Fatalf("reabriendo la base con la fecha rota: %v", err)
			}

			defer base.Close() //nolint:errcheck // la aserción de abajo ya decide el resultado

			if _, err := base.Task(t.Context(), 1); err == nil || !strings.Contains(err.Error(), caso.fragment) {
				t.Errorf("Task devolvió %v, se esperaba un error que mencionara %q", err, caso.fragment)
			}
		})
	}
}

func TestRecordedPrefix_keepsThePrefixItWasGiven(t *testing.T) {
	t.Parallel()

	base := openStore(t)
	ctx := t.Context()

	recorded, err := base.RecordedPrefix(ctx)
	if err != nil {
		t.Fatalf("RecordedPrefix devolvió %v", err)
	}

	if recorded != "" {
		t.Errorf("una base nueva trae el prefijo %q, se esperaba ninguno", recorded)
	}

	if err := base.SetPrefix(ctx, "px"); err != nil {
		t.Fatalf("SetPrefix devolvió %v", err)
	}

	recorded, err = base.RecordedPrefix(ctx)
	if err != nil {
		t.Fatalf("RecordedPrefix devolvió %v", err)
	}

	if recorded != "px" {
		t.Errorf("la base anotó el prefijo %q, se esperaba %q", recorded, "px")
	}
}

// storedTask guarda una tarea con detalle en una base nueva y la lee otra vez.
func storedTask(t *testing.T) (created, read store.Task) {
	t.Helper()

	base := openStore(t)
	description := "con detalle"

	var err error

	created, err = base.CreateTask(t.Context(), store.Task{
		Title:       "Con detalle",
		Description: &description,
		Status:      "open",
		Priority:    3,
	}, nil)
	if err != nil {
		t.Fatalf("guardando la tarea: %v", err)
	}

	read, err = base.Task(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("leyendo la tarea: %v", err)
	}

	return created, read
}

// rawBase ejecuta una instrucción sobre la base directamente con el motor, sin
// pasar por la store: sirve para dejarla en estados que la store no crea pero
// un usuario puede encontrarse.
func rawBase(t *testing.T, path, statement string, args ...any) {
	t.Helper()

	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("abriendo la base por debajo de la store: %v", err)
	}

	defer raw.Close() //nolint:errcheck // las aserciones de abajo ya deciden el resultado

	if _, err := raw.ExecContext(t.Context(), statement, args...); err != nil {
		t.Fatalf("ejecutando %q sobre la base: %v", statement, err)
	}
}

// TestOpen_givesUpWhenTheLockNeverYields: un bloqueo que no se va a resolver solo,
// como el de otro proceso con una transacción larga, no puede dejar al comando
// aparentemente colgado: el arranque se rinde en unos segundos y lo dice.
func TestOpen_givesUpWhenTheLockNeverYields(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "tasks.db")

	// Una base ya creada, para que la carrera al crearla no se mezcle con esto.
	created, err := store.Open(t.Context(), path)
	if err != nil {
		t.Fatalf("abriendo la base de prueba: %v", err)
	}

	if err := created.Close(); err != nil {
		t.Fatalf("cerrando la base de prueba: %v", err)
	}

	// La base debe una migración para que Open necesite la escritura que este
	// test bloquea; una base al día ya no abre transacciones solo para mirar.
	rawBase(t, path, "DROP TABLE adrs")
	rawBase(t, path, "DROP INDEX comments_task_id_idx")
	rawBase(t, path, "DROP INDEX dependencies_blocker_id_idx")
	rawBase(t, path, "UPDATE meta SET value = '5' WHERE key = 'schema_version'")

	// Otra conexión toma la escritura y no la suelta, como un proceso atascado.
	keen, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("abriendo la conexión que bloquea: %v", err)
	}

	defer keen.Close() //nolint:errcheck // las aserciones de abajo ya deciden el resultado

	locker, err := keen.Conn(t.Context())
	if err != nil {
		t.Fatalf("tomando una conexión propia para bloquear: %v", err)
	}

	defer locker.Close() //nolint:errcheck // las aserciones de abajo ya deciden el resultado

	if _, err := locker.ExecContext(t.Context(), "BEGIN IMMEDIATE"); err != nil {
		t.Fatalf("tomando la escritura de la base: %v", err)
	}

	defer func() { _, _ = locker.ExecContext(context.Background(), "ROLLBACK") }()

	started := time.Now()

	_, err = store.Open(t.Context(), path)

	elapsed := time.Since(started)

	if err == nil {
		t.Fatal("abriendo una base bloqueada: no falló")
	}

	if elapsed > 5*time.Second {
		t.Errorf("Open tardó %v en rendirse: un bloqueo que no se resuelve deja al comando colgado", elapsed)
	}
}

func TestCreateTask_annotatesTheChangeItMade(t *testing.T) {
	t.Parallel()

	base := openStore(t)
	ctx := t.Context()

	created, err := base.CreateTask(ctx, newTask("Primera tarea"), nil)
	if err != nil {
		t.Fatalf("guardando la tarea: %v", err)
	}

	changes, err := base.Changes(ctx, 0, wholeHistory)
	if err != nil {
		t.Fatalf("leyendo los cambios: %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("la creación dejó %d filas de cambio, se esperaba 1", len(changes))
	}

	change := changes[0]
	if change.Resource != store.ResourceTask || change.ResourceID != created.ID {
		t.Errorf("el cambio habla de %s %d, se esperaba la tarea %d", change.Resource, change.ResourceID, created.ID)
	}

	if change.Origin != store.OriginCLI {
		t.Errorf("el cambio dice que lo pidió %q, se esperaba %q", change.Origin, store.OriginCLI)
	}

	if change.HappenedAt.IsZero() {
		t.Error("el cambio no dice cuándo ocurrió")
	}
}

func TestCreateTask_namesTheIdentityInTheNoticeNeverTheReference(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "tasks.db")

	base, err := store.OpenAs(t.Context(), path, store.OriginCLI)
	if err != nil {
		t.Fatalf("abriendo la base de prueba: %v", err)
	}

	defer base.Close() //nolint:errcheck // la aserción de abajo ya decide el resultado

	if _, err := base.CreateTask(t.Context(), newTask("Tarea sin rastro"), nil); err != nil {
		t.Fatalf("guardando la tarea: %v", err)
	}

	// La fila se lee tal como está guardada: la identidad viaja por el número
	// interno y la referencia compuesta no ocupa sitio en ningún sitio.
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("abriendo la base por debajo de la store: %v", err)
	}

	defer raw.Close() //nolint:errcheck // las aserciones de abajo ya deciden el resultado

	var resource, resourceID, origin, happenedAt string

	err = raw.QueryRowContext(t.Context(),
		`SELECT resource, CAST(resource_id AS TEXT), origin, happened_at FROM changes`,
	).Scan(&resource, &resourceID, &origin, &happenedAt)
	if err != nil {
		t.Fatalf("leyendo la fila de cambio por debajo de la store: %v", err)
	}

	if resource != store.ResourceTask {
		t.Errorf("la fila habla del recurso %q, se esperaba %q", resource, store.ResourceTask)
	}

	if resourceID != "1" {
		t.Errorf("la fila nombra la identidad %q, se esperaba el número interno 1", resourceID)
	}

	for name, value := range map[string]string{"recurso": resource, "identidad": resourceID, "origen": origin, "momento": happenedAt} {
		if strings.Contains(value, "tk") {
			t.Errorf("la fila escribe %q en %s: el aviso no nombra referencias escritas", value, name)
		}
	}
}

func TestOpenAs_recordsTheSurfaceThatAsksForTheChanges(t *testing.T) {
	t.Parallel()

	base, err := store.OpenAs(t.Context(), filepath.Join(t.TempDir(), "tasks.db"), store.OriginServer)
	if err != nil {
		t.Fatalf("abriendo la base del servidor: %v", err)
	}

	defer base.Close() //nolint:errcheck // la aserción de abajo ya decide el resultado

	created, err := base.CreateTask(t.Context(), newTask("Pedida por el servidor"), nil)
	if err != nil {
		t.Fatalf("guardando la tarea: %v", err)
	}

	changes, err := base.Changes(t.Context(), 0, wholeHistory)
	if err != nil {
		t.Fatalf("leyendo los cambios: %v", err)
	}

	if len(changes) != 1 || changes[0].Origin != store.OriginServer || changes[0].ResourceID != created.ID {
		t.Errorf("el cambio no dice que lo pidió el servidor: %+v", changes)
	}
}

//nolint:gocyclo // the test asserts both idempotent blocker operations and their notices
func TestAddBlock_annotatesTheChangeOnlyWhenSomethingChanges(t *testing.T) {
	t.Parallel()

	base := openStore(t)
	ctx := t.Context()

	blocked, err := base.CreateTask(ctx, newTask("Bloqueada"), nil)
	if err != nil {
		t.Fatalf("guardando la tarea bloqueada: %v", err)
	}

	blocker, err := base.CreateTask(ctx, newTask("Bloqueante"), nil)
	if err != nil {
		t.Fatalf("guardando la tarea bloqueante: %v", err)
	}

	if err := base.AddBlock(ctx, blocked.ID, blocker.ID); err != nil {
		t.Fatalf("bloqueando la tarea: %v", err)
	}

	// Bloquear la misma pareja otra vez no cambia nada, y tampoco deja aviso.
	if err := base.AddBlock(ctx, blocked.ID, blocker.ID); err != nil {
		t.Fatalf("bloqueando la misma pareja otra vez: %v", err)
	}

	changes, err := base.Changes(ctx, 0, wholeHistory)
	if err != nil {
		t.Fatalf("leyendo los cambios: %v", err)
	}

	if len(changes) != 3 {
		t.Fatalf("dos creaciones y un bloqueo dejaron %d filas, se esperaban 3", len(changes))
	}

	if changes[2].Resource != store.ResourceTask || changes[2].ResourceID != blocked.ID {
		t.Errorf("el bloqueo habla de %s %d, se esperaba la tarea bloqueada %d", changes[2].Resource, changes[2].ResourceID, blocked.ID)
	}

	// Un bloqueo cambia la tarea: lo que le pasó es un cambio, no un alta ni un
	// cierre, y el aviso lo dice.
	if changes[2].Action != store.ActionUpdate {
		t.Errorf("el bloqueo dice %q, se esperaba %q", changes[2].Action, store.ActionUpdate)
	}

	// Retirar un bloqueo que está deja aviso; retirar el que ya no está, no.
	if err := base.RemoveBlock(ctx, blocked.ID, blocker.ID); err != nil {
		t.Fatalf("retirando el bloqueo: %v", err)
	}

	if err := base.RemoveBlock(ctx, blocked.ID, blocker.ID); err != nil {
		t.Fatalf("retirando un bloqueo que ya no está: %v", err)
	}

	changes, err = base.Changes(ctx, 0, wholeHistory)
	if err != nil {
		t.Fatalf("leyendo los cambios: %v", err)
	}

	if len(changes) != 4 {
		t.Errorf("retirar el bloqueo dos veces dejó %d filas, se esperaban 4", len(changes))
	}
}

//nolint:gocyclo // the test covers every public operation that records a notice
func TestChangeOperations_eachLeaveTheirNotice(t *testing.T) {
	t.Parallel()

	base := openStore(t)
	ctx := t.Context()

	created, err := base.CreateTask(ctx, newTask("Con historia"), nil)
	if err != nil {
		t.Fatalf("guardando la tarea: %v", err)
	}

	// Cada operación que escribe deja su fila: el cambio, el cierre, el
	// comentario y, en el catálogo, la creación, el renombrado y el borrado.
	if _, err := base.UpdateTask(ctx, created.ID, func(task store.Task) (store.Task, error) {
		task.Title = "Cambiada"

		return task, nil
	}, nil); err != nil {
		t.Fatalf("cambiando la tarea: %v", err)
	}

	if _, err := base.CloseTask(ctx, created.ID, "done"); err != nil {
		t.Fatalf("cerrando la tarea: %v", err)
	}

	if _, err := base.AddComment(ctx, created.ID, "un comentario"); err != nil {
		t.Fatalf("añadiendo el comentario: %v", err)
	}

	tag, err := base.CreateTag(ctx, store.Tag{Name: "docs", Color: "blue"})
	if err != nil {
		t.Fatalf("creando la etiqueta: %v", err)
	}

	renamed, err := base.UpdateTag(ctx, "docs", func(tag store.Tag) (store.Tag, error) {
		tag.Name = "documentacion"

		return tag, nil
	})
	if err != nil {
		t.Fatalf("renombrando la etiqueta: %v", err)
	}

	if _, _, err := base.DeleteTag(ctx, renamed.Name); err != nil {
		t.Fatalf("borrando la etiqueta: %v", err)
	}

	changes, err := base.Changes(ctx, 0, wholeHistory)
	if err != nil {
		t.Fatalf("leyendo los cambios: %v", err)
	}

	expected := []struct {
		resource string
		id       int64
		action   string
	}{
		{store.ResourceTask, created.ID, store.ActionCreate},
		{store.ResourceTask, created.ID, store.ActionUpdate},
		{store.ResourceTask, created.ID, store.ActionClose},
		// El comentario cambia la tarea: su aviso lo dice como un cambio.
		{store.ResourceTask, created.ID, store.ActionUpdate},
		{store.ResourceTag, tag.ID, store.ActionCreate},
		{store.ResourceTag, tag.ID, store.ActionUpdate},
		{store.ResourceTag, tag.ID, store.ActionDelete},
	}

	if len(changes) != len(expected) {
		t.Fatalf("las operaciones dejaron %d filas, se esperaban %d: %+v", len(changes), len(expected), changes)
	}

	for i, want := range expected {
		if changes[i].Resource != want.resource || changes[i].ResourceID != want.id {
			t.Errorf("la fila %d habla de %s %d, se esperaba %s %d", i+1, changes[i].Resource, changes[i].ResourceID, want.resource, want.id)
		}

		if changes[i].Action != want.action {
			t.Errorf("la fila %d dice %q, se esperaba %q", i+1, changes[i].Action, want.action)
		}
	}
}

// TestOpen_addsTheActionToABaseWhoseNoticesDoNotHaveOneYet recorre la migración
// sobre una base ya escrita: las filas de antes siguen ahí con todo lo que
// decían y solo ganan la acción vacía de quien no la anotó, y la base vieja
// sigue escribiendo avisos con su acción a partir de ahí.
//
//nolint:gocyclo // the migration test checks both legacy preservation and new writes
func TestOpen_addsTheActionToABaseWhoseNoticesDoNotHaveOneYet(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "tasks.db")

	base, err := store.Open(t.Context(), path)
	if err != nil {
		t.Fatalf("abriendo la base de prueba: %v", err)
	}

	created, err := base.CreateTask(t.Context(), newTask("De antes"), nil)
	if err != nil {
		t.Fatalf("guardando la tarea: %v", err)
	}

	if err := base.Close(); err != nil {
		t.Fatalf("cerrando la base: %v", err)
	}

	// La base se deja como estaba antes de la columna: sin ella y con el
	// esquema 4 anotado, que es como llega una base de una versión anterior.
	rawBase(t, path, `ALTER TABLE changes DROP COLUMN action`)
	rawBase(t, path, `DROP TABLE adrs`)
	rawBase(t, path, `DROP INDEX comments_task_id_idx`)
	rawBase(t, path, `DROP INDEX dependencies_blocker_id_idx`)
	rawBase(t, path, `UPDATE meta SET value = '4' WHERE key = 'schema_version'`)

	again, err := store.Open(t.Context(), path)
	if err != nil {
		t.Fatalf("abriendo la base de antes: %v", err)
	}

	defer again.Close() //nolint:errcheck // las aserciones de abajo ya deciden el resultado

	ctx := t.Context()

	version, err := again.SchemaVersion(ctx)
	if err != nil {
		t.Fatalf("SchemaVersion devolvió %v", err)
	}

	if version != 7 {
		t.Errorf("abrir la base de antes la dejó en el esquema %d, se esperaba el 7", version)
	}

	changes, err := again.Changes(ctx, 0, wholeHistory)
	if err != nil {
		t.Fatalf("leyendo los cambios: %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("la base de antes conserva %d filas, se esperaba 1", len(changes))
	}

	stored := changes[0]
	if stored.Resource != store.ResourceTask || stored.ResourceID != created.ID || stored.Origin != store.OriginCLI {
		t.Errorf("el aviso de antes se lee %+v, se esperaba la tarea %d pedida por el CLI", stored, created.ID)
	}

	if stored.Action != "" {
		t.Errorf("el aviso de antes dice %q, se esperaba sin acción, que es lo que no se anotó", stored.Action)
	}

	// Y la base migrada sigue sirviendo: el aviso nuevo lleva su acción.
	if _, err := again.AddComment(ctx, created.ID, "un comentario"); err != nil {
		t.Fatalf("añadiendo el comentario: %v", err)
	}

	changes, err = again.Changes(ctx, stored.ID, wholeHistory)
	if err != nil {
		t.Fatalf("leyendo los cambios nuevos: %v", err)
	}

	if len(changes) != 1 || changes[0].Action != store.ActionUpdate {
		t.Errorf("el aviso nuevo dice %+v, se esperaba un cambio de la tarea", changes)
	}
}

// TestTaskTitles_returnsTheCurrentTitleOfEachTaskAskedFor recorre la lectura que
// compone el aviso: el título de ahora de cada tarea pedida, en una sola
// consulta, con la tarea que ya no está fuera del mapa y ninguna pregunta para
// una lista vacía.
func TestTaskTitles_returnsTheCurrentTitleOfEachTaskAskedFor(t *testing.T) {
	t.Parallel()

	base := openStore(t)
	ctx := t.Context()

	first, err := base.CreateTask(ctx, newTask("Primera"), nil)
	if err != nil {
		t.Fatalf("guardando la primera tarea: %v", err)
	}

	second, err := base.CreateTask(ctx, newTask("Segunda"), nil)
	if err != nil {
		t.Fatalf("guardando la segunda tarea: %v", err)
	}

	if _, err := base.UpdateTask(ctx, second.ID, func(task store.Task) (store.Task, error) {
		task.Title = "Renombrada"

		return task, nil
	}, nil); err != nil {
		t.Fatalf("cambiando la segunda tarea: %v", err)
	}

	titles, err := base.TaskTitles(ctx, []int64{first.ID, second.ID, 404})
	if err != nil {
		t.Fatalf("leyendo los títulos: %v", err)
	}

	if len(titles) != 2 {
		t.Fatalf("los títulos pedidos son %d, se esperaban 2: %+v", len(titles), titles)
	}

	if titles[first.ID] != "Primera" || titles[second.ID] != "Renombrada" {
		t.Errorf("los títulos leídos son %+v, se esperaba la primera y la segunda renombrada", titles)
	}

	none, err := base.TaskTitles(ctx, nil)
	if err != nil {
		t.Fatalf("leyendo los títulos de ninguna tarea: %v", err)
	}

	if len(none) != 0 {
		t.Errorf("una lista vacía pidió %d títulos, se esperaba ninguno", len(none))
	}
}

func TestChanges_readsOnlyWhatHappenedAfter(t *testing.T) {
	t.Parallel()

	base := openStore(t)
	ctx := t.Context()

	first, err := base.CreateTask(ctx, newTask("Primera"), nil)
	if err != nil {
		t.Fatalf("guardando la primera tarea: %v", err)
	}

	if _, err := base.CreateTask(ctx, newTask("Segunda"), nil); err != nil {
		t.Fatalf("guardando la segunda tarea: %v", err)
	}

	after, err := base.Changes(ctx, first.ID, wholeHistory)
	if err != nil {
		t.Fatalf("leyendo los cambios: %v", err)
	}

	if len(after) != 1 {
		t.Errorf("leer a partir de %d devolvió %d cambios, se esperaba 1", first.ID, len(after))
	}
}

// TestChanges_readsBoundedBatchesInTheOrderTheyHappened recorre el lote del
// historial: como mucho los avisos pedidos, en el orden en que ocurrieron, y
// cada lote deja en el último de ellos la identidad desde la que seguir leyendo.
func TestChanges_readsBoundedBatchesInTheOrderTheyHappened(t *testing.T) {
	t.Parallel()

	base := openStore(t)
	ctx := t.Context()

	for _, title := range []string{"Uno", "Dos", "Tres", "Cuatro", "Cinco"} {
		if _, err := base.CreateTask(ctx, newTask(title), nil); err != nil {
			t.Fatalf("guardando la tarea %q: %v", title, err)
		}
	}

	// Cada lote trae como mucho el tope pedido, en el orden en que ocurrieron, y su
	// último aviso es la identidad desde la que seguir leyendo. El último lote sale
	// más corto que el tope y, después de él, la historia ya no da más.
	want := [][]int64{{1, 2}, {3, 4}, {5}, {}}

	after := int64(0)

	for _, ids := range want {
		batch, err := base.Changes(ctx, after, 2)
		if err != nil {
			t.Fatalf("leyendo un lote de dos avisos desde %d: %v", after, err)
		}

		if got := changeIDs(batch); !slices.Equal(got, ids) {
			t.Errorf("el lote leído desde el aviso %d trajo %v, se esperaba %v", after, got, ids)
		}

		if len(batch) > 0 {
			after = batch[len(batch)-1].ID
		}
	}
}

// changeIDs nombra los avisos leídos por su identidad, en el orden en que vienen.
func changeIDs(changes []store.Change) []int64 {
	ids := make([]int64, 0, len(changes))

	for _, change := range changes {
		ids = append(ids, change.ID)
	}

	return ids
}

// TestTagNames_returnsTheNameOfEachTagInTheBatch recorre la lectura del lote: los
// nombres de ahora de todas las identidades pedidas, en una sola consulta, con la
// etiqueta borrada y la que nadie tiene fuera del mapa —leerlas no es un error, así
// el aviso de una etiqueta que ya no está se queda sin nombre— y ninguna pregunta
// para una lista vacía.
func TestTagNames_returnsTheNameOfEachTagInTheBatch(t *testing.T) {
	t.Parallel()

	base := openStore(t)
	ctx := t.Context()

	docs, err := base.CreateTag(ctx, store.Tag{Name: "docs", Color: "blue"})
	if err != nil {
		t.Fatalf("creando la etiqueta docs: %v", err)
	}

	feat, err := base.CreateTag(ctx, store.Tag{Name: "feat", Color: "green"})
	if err != nil {
		t.Fatalf("creando la etiqueta feat: %v", err)
	}

	gone, err := base.CreateTag(ctx, store.Tag{Name: "gone", Color: "red"})
	if err != nil {
		t.Fatalf("creando la etiqueta a borrar: %v", err)
	}

	if _, _, err := base.DeleteTag(ctx, gone.Name); err != nil {
		t.Fatalf("borrando la etiqueta: %v", err)
	}

	names, err := base.TagNames(ctx, []int64{docs.ID, feat.ID, gone.ID, 404})
	if err != nil {
		t.Fatalf("leyendo los nombres del lote: %v", err)
	}

	if len(names) != 2 || names[docs.ID] != "docs" || names[feat.ID] != "feat" {
		t.Errorf("el lote devolvió %+v, se esperaban docs y feat", names)
	}

	if _, ok := names[gone.ID]; ok {
		t.Errorf("la etiqueta borrada %d sigue en el mapa %+v, se esperaba fuera", gone.ID, names)
	}

	none, err := base.TagNames(ctx, nil)
	if err != nil {
		t.Fatalf("leyendo los nombres de ninguna etiqueta: %v", err)
	}

	if len(none) != 0 {
		t.Errorf("una lista vacía pidió %d nombres, se esperaba ninguno", len(none))
	}
}

// TestComments_areStoredUnderTheirTaskAndReadFirstToLast recorre el histórico de
// una tarea: cada comentario guarda la tarea que comenta y sale del primero al
// último, aunque todos se escriban dentro del mismo segundo.
//
//nolint:gocyclo // el recorrido completo del criterio debe quedarse lineal
func TestComments_areStoredUnderTheirTaskAndReadFirstToLast(t *testing.T) {
	t.Parallel()

	base := openStore(t)
	ctx := t.Context()

	task, err := base.CreateTask(ctx, newTask("Con histórico"), nil)
	if err != nil {
		t.Fatalf("creando la tarea comentada: %v", err)
	}

	elsewhere, err := base.CreateTask(ctx, newTask("Otra tarea"), nil)
	if err != nil {
		t.Fatalf("creando la otra tarea: %v", err)
	}

	bodies := []string{"el primero", "el segundo", "el tercero"}
	added := make([]store.Comment, 0, len(bodies))

	for _, body := range bodies {
		comment, err := base.AddComment(ctx, task.ID, body)
		if err != nil {
			t.Fatalf("añadiendo el comentario %q: %v", body, err)
		}

		if comment.ID == 0 || comment.TaskID != task.ID || comment.Body != body {
			t.Errorf("el comentario añadido es %+v, se esperaba de la tarea %d con cuerpo %q", comment, task.ID, body)
		}

		added = append(added, comment)
	}

	if _, err := base.AddComment(ctx, elsewhere.ID, "el de otra tarea"); err != nil {
		t.Fatalf("añadiendo el comentario de otra tarea: %v", err)
	}

	read, err := base.Comments(ctx, task.ID)
	if err != nil {
		t.Fatalf("leyendo los comentarios: %v", err)
	}

	if len(read) != len(bodies) {
		t.Fatalf("la tarea tiene %d comentarios, se esperaban %d: %+v", len(read), len(bodies), read)
	}

	for i, want := range added {
		got := read[i]

		if got.ID != want.ID || got.TaskID != task.ID || got.Body != want.Body {
			t.Errorf("el %dº comentario es %+v, se esperaba %+v", i+1, got, want)
		}

		if !got.CreatedAt.Equal(want.CreatedAt) {
			t.Errorf("el %dº comentario se lee en %s, se esperaba %s", i+1, got.CreatedAt, want.CreatedAt)
		}

		if i > 0 && read[i-1].ID >= got.ID {
			t.Errorf("los comentarios salen sin orden: el %dº lleva id %d detrás de %d",
				i+1, got.ID, read[i-1].ID)
		}
	}
}

// TestEditComment_changesOnlyTheBodyOfTheCommentItNames recorre la edición: la
// identidad, la tarea y la fecha de creación se conservan, el cuerpo cambia, los
// demás comentarios siguen como estaban y la tarea queda avisada de su cambio en
// la misma transacción.
//
//nolint:gocyclo // el recorrido completo del criterio debe quedarse lineal
func TestEditComment_changesOnlyTheBodyOfTheCommentItNames(t *testing.T) {
	t.Parallel()

	base := openStore(t)
	ctx := t.Context()

	task, err := base.CreateTask(ctx, newTask("Con histórico"), nil)
	if err != nil {
		t.Fatalf("creando la tarea: %v", err)
	}

	first, err := base.AddComment(ctx, task.ID, "mal escrito")
	if err != nil {
		t.Fatalf("añadiendo el primer comentario: %v", err)
	}

	second, err := base.AddComment(ctx, task.ID, "el segundo")
	if err != nil {
		t.Fatalf("añadiendo el segundo comentario: %v", err)
	}

	before, err := base.LastChangeID(ctx)
	if err != nil {
		t.Fatalf("leyendo el último aviso: %v", err)
	}

	edited, err := base.EditComment(ctx, task.ID, first.ID, "bien escrito")
	if err != nil {
		t.Fatalf("editando el comentario %d: %v", first.ID, err)
	}

	if edited.ID != first.ID || edited.TaskID != task.ID {
		t.Errorf("el comentario editado es %+v, se esperaba que conservara su id %d y su tarea %d",
			edited, first.ID, task.ID)
	}

	if !edited.CreatedAt.Equal(first.CreatedAt) {
		t.Errorf("la edición cambió la fecha a %s, se esperaba la de creación %s", edited.CreatedAt, first.CreatedAt)
	}

	if edited.Body != "bien escrito" {
		t.Errorf("el cuerpo editado es %q, se esperaba %q", edited.Body, "bien escrito")
	}

	read, err := base.Comments(ctx, task.ID)
	if err != nil {
		t.Fatalf("leyendo los comentarios: %v", err)
	}

	if len(read) != 2 || read[0].Body != "bien escrito" {
		t.Fatalf("los comentarios leídos son %+v, se esperaba el primero corregido", read)
	}

	if read[1].ID != second.ID || read[1].Body != second.Body || !read[1].CreatedAt.Equal(second.CreatedAt) {
		t.Errorf("el comentario vecino cambió a %+v, se esperaba %+v", read[1], second)
	}

	notices, err := base.Changes(ctx, before, wholeHistory)
	if err != nil {
		t.Fatalf("leyendo los avisos: %v", err)
	}

	if len(notices) != 1 {
		t.Fatalf("la edición dejó %d avisos, se esperaba 1: %+v", len(notices), notices)
	}

	if notices[0].Resource != store.ResourceTask || notices[0].ResourceID != task.ID ||
		notices[0].Action != store.ActionUpdate {
		t.Errorf("el aviso de la edición es %+v, se esperaba la tarea %d actualizada", notices[0], task.ID)
	}
}

// TestEditComment_refusesWhatIsNotThereWithoutWriting recorre el rechazo: el
// número se entiende dentro de la tarea, así que un comentario de otra tarea o un
// número que nadie lleva sale por ErrNotFound sin tocar el histórico ni dejar
// aviso.
func TestEditComment_refusesWhatIsNotThereWithoutWriting(t *testing.T) {
	t.Parallel()

	base := openStore(t)
	ctx := t.Context()

	task, err := base.CreateTask(ctx, newTask("Con histórico"), nil)
	if err != nil {
		t.Fatalf("creando la tarea: %v", err)
	}

	elsewhere, err := base.CreateTask(ctx, newTask("Otra tarea"), nil)
	if err != nil {
		t.Fatalf("creando la otra tarea: %v", err)
	}

	mine, err := base.AddComment(ctx, task.ID, "el suyo")
	if err != nil {
		t.Fatalf("añadiendo el comentario: %v", err)
	}

	theirs, err := base.AddComment(ctx, elsewhere.ID, "el de otra tarea")
	if err != nil {
		t.Fatalf("añadiendo el comentario de otra tarea: %v", err)
	}

	before, err := base.LastChangeID(ctx)
	if err != nil {
		t.Fatalf("leyendo el último aviso: %v", err)
	}

	tests := []struct {
		name    string
		taskID  int64
		comment int64
	}{
		{name: "the identity of a comment of another task", taskID: task.ID, comment: theirs.ID},
		{name: "an identity nobody carries", taskID: task.ID, comment: theirs.ID + 100},
		{name: "a task that is not stored", taskID: 404, comment: mine.ID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := base.EditComment(ctx, tt.taskID, tt.comment, "cambiado"); !errors.Is(err, store.ErrNotFound) {
				t.Errorf("EditComment devolvió %v, se esperaba ErrNotFound", err)
			}

			// Un rechazo no escribe nada: ni toca el histórico ni deja aviso.
			read, err := base.Comments(ctx, task.ID)
			if err != nil {
				t.Fatalf("leyendo los comentarios: %v", err)
			}

			if len(read) != 1 || read[0].Body != "el suyo" {
				t.Errorf("el rechazo dejó el histórico como %+v, se esperaba intacto", read)
			}

			last, err := base.LastChangeID(ctx)
			if err != nil {
				t.Fatalf("leyendo el último aviso: %v", err)
			}

			if last != before {
				t.Errorf("el rechazo dejó avisos hasta %d, se esperaba %d", last, before)
			}
		})
	}
}

// TestAllComments_listsEveryCommentOfTheProjectOrderedByTask recorre el índice
// que alimenta el buscador: cada comentario del proyecto sale con la tarea que
// lo lleva, de la primera tarea a la última y, dentro de cada una, del primer
// comentario al último. Una base sin comentarios responde sin ninguno.
func TestAllComments_listsEveryCommentOfTheProjectOrderedByTask(t *testing.T) {
	t.Parallel()

	base := openStore(t)
	ctx := t.Context()

	none, err := base.AllComments(ctx)
	if err != nil {
		t.Fatalf("leyendo los comentarios de una base vacía: %v", err)
	}

	if len(none) != 0 {
		t.Errorf("una base sin comentarios respondió %+v, se esperaba ninguno", none)
	}

	first, err := base.CreateTask(ctx, newTask("Con historial"), nil)
	if err != nil {
		t.Fatalf("creando la primera tarea: %v", err)
	}

	second, err := base.CreateTask(ctx, newTask("Otra con historial"), nil)
	if err != nil {
		t.Fatalf("creando la segunda tarea: %v", err)
	}

	firstComment, err := base.AddComment(ctx, first.ID, "primero de la primera")
	if err != nil {
		t.Fatalf("añadiendo el primer comentario: %v", err)
	}

	secondComment, err := base.AddComment(ctx, first.ID, "segundo de la primera")
	if err != nil {
		t.Fatalf("añadiendo el segundo comentario: %v", err)
	}

	thirdComment, err := base.AddComment(ctx, second.ID, "comentario de la otra")
	if err != nil {
		t.Fatalf("añadiendo el comentario de la otra tarea: %v", err)
	}

	all, err := base.AllComments(ctx)
	if err != nil {
		t.Fatalf("leyendo todos los comentarios: %v", err)
	}

	want := []store.Comment{firstComment, secondComment, thirdComment}

	if !slices.Equal(all, want) {
		t.Errorf("el índice es %+v, se esperaba %+v", all, want)
	}
}

// wholeHistory es el límite que las pruebas piden cuando quieren la historia entera
// de una base pequeña: Changes siempre lee como mucho un lote.
const wholeHistory = 100

func openStore(t *testing.T) *store.Store {
	t.Helper()

	base, err := store.Open(t.Context(), filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatalf("abriendo la base de prueba: %v", err)
	}

	t.Cleanup(func() { _ = base.Close() })

	return base
}

func newTask(title string) store.Task {
	return store.Task{Title: title, Status: "open", Priority: 2}
}
