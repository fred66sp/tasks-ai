package store_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
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

	if version != 4 {
		t.Errorf("una base nueva queda en el esquema %d, se esperaba el 4", version)
	}
}

func TestOpen_doesNotRepeatTheSchemaItAlreadyHas(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "tasks.db")

	first, err := store.Open(t.Context(), path)
	if err != nil {
		t.Fatalf("abriendo la base por primera vez: %v", err)
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

	if version != 4 {
		t.Errorf("reabrir la base la dejó en el esquema %d, se esperaba el 4", version)
	}
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

	if version != 4 {
		t.Errorf("la base quedó en el esquema %d, se esperaba el 4", version)
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

	if !strings.Contains(err.Error(), "creando la carpeta de la base") {
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

	if !strings.Contains(err.Error(), `"próximamente"`) || !strings.Contains(err.Error(), "no es un número") {
		t.Errorf("el error no señala la versión guardada: %v", err)
	}
}

func TestCreateTask_givesEachTaskItsOwnIdentity(t *testing.T) {
	t.Parallel()

	base := openStore(t)
	ctx := t.Context()

	first, err := base.CreateTask(ctx, newTask("Primera"))
	if err != nil {
		t.Fatalf("guardando la primera tarea: %v", err)
	}

	second, err := base.CreateTask(ctx, newTask("Segunda"))
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
		{name: "la fecha de creación", createdAt: "ayer", updatedAt: today, fragment: "la fecha de creación de la tarea 1"},
		{name: "la fecha de cambio", createdAt: today, updatedAt: "el martes pasado", fragment: "la fecha de cambio de la tarea 1"},
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
	})
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

	created, err := base.CreateTask(ctx, newTask("Primera tarea"))
	if err != nil {
		t.Fatalf("guardando la tarea: %v", err)
	}

	changes, err := base.Changes(ctx, 0)
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

	if _, err := base.CreateTask(t.Context(), newTask("Tarea sin rastro")); err != nil {
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

	created, err := base.CreateTask(t.Context(), newTask("Pedida por el servidor"))
	if err != nil {
		t.Fatalf("guardando la tarea: %v", err)
	}

	changes, err := base.Changes(t.Context(), 0)
	if err != nil {
		t.Fatalf("leyendo los cambios: %v", err)
	}

	if len(changes) != 1 || changes[0].Origin != store.OriginServer || changes[0].ResourceID != created.ID {
		t.Errorf("el cambio no dice que lo pidió el servidor: %+v", changes)
	}
}

func TestAddBlock_annotatesTheChangeOnlyWhenSomethingChanges(t *testing.T) {
	t.Parallel()

	base := openStore(t)
	ctx := t.Context()

	blocked, err := base.CreateTask(ctx, newTask("Bloqueada"))
	if err != nil {
		t.Fatalf("guardando la tarea bloqueada: %v", err)
	}

	blocker, err := base.CreateTask(ctx, newTask("Bloqueante"))
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

	changes, err := base.Changes(ctx, 0)
	if err != nil {
		t.Fatalf("leyendo los cambios: %v", err)
	}

	if len(changes) != 3 {
		t.Fatalf("dos creaciones y un bloqueo dejaron %d filas, se esperaban 3", len(changes))
	}

	if changes[2].Resource != store.ResourceTask || changes[2].ResourceID != blocked.ID {
		t.Errorf("el bloqueo habla de %s %d, se esperaba la tarea bloqueada %d", changes[2].Resource, changes[2].ResourceID, blocked.ID)
	}

	// Retirar un bloqueo que está deja aviso; retirar el que ya no está, no.
	if err := base.RemoveBlock(ctx, blocked.ID, blocker.ID); err != nil {
		t.Fatalf("retirando el bloqueo: %v", err)
	}

	if err := base.RemoveBlock(ctx, blocked.ID, blocker.ID); err != nil {
		t.Fatalf("retirando un bloqueo que ya no está: %v", err)
	}

	changes, err = base.Changes(ctx, 0)
	if err != nil {
		t.Fatalf("leyendo los cambios: %v", err)
	}

	if len(changes) != 4 {
		t.Errorf("retirar el bloqueo dos veces dejó %d filas, se esperaban 4", len(changes))
	}
}

func TestChangeOperations_eachLeaveTheirNotice(t *testing.T) {
	t.Parallel()

	base := openStore(t)
	ctx := t.Context()

	created, err := base.CreateTask(ctx, newTask("Con historia"))
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

	changes, err := base.Changes(ctx, 0)
	if err != nil {
		t.Fatalf("leyendo los cambios: %v", err)
	}

	expected := []struct {
		resource string
		id       int64
	}{
		{store.ResourceTask, created.ID},
		{store.ResourceTask, created.ID},
		{store.ResourceTask, created.ID},
		{store.ResourceTask, created.ID},
		{store.ResourceTag, tag.ID},
		{store.ResourceTag, tag.ID},
		{store.ResourceTag, tag.ID},
	}

	if len(changes) != len(expected) {
		t.Fatalf("las operaciones dejaron %d filas, se esperaban %d: %+v", len(changes), len(expected), changes)
	}

	for i, want := range expected {
		if changes[i].Resource != want.resource || changes[i].ResourceID != want.id {
			t.Errorf("la fila %d habla de %s %d, se esperaba %s %d", i+1, changes[i].Resource, changes[i].ResourceID, want.resource, want.id)
		}
	}
}

func TestChanges_readsOnlyWhatHappenedAfter(t *testing.T) {
	t.Parallel()

	base := openStore(t)
	ctx := t.Context()

	first, err := base.CreateTask(ctx, newTask("Primera"))
	if err != nil {
		t.Fatalf("guardando la primera tarea: %v", err)
	}

	if _, err := base.CreateTask(ctx, newTask("Segunda")); err != nil {
		t.Fatalf("guardando la segunda tarea: %v", err)
	}

	after, err := base.Changes(ctx, first.ID)
	if err != nil {
		t.Fatalf("leyendo los cambios: %v", err)
	}

	if len(after) != 1 {
		t.Errorf("leer a partir de %d devolvió %d cambios, se esperaba 1", first.ID, len(after))
	}
}

func TestTagName_returnsTheNameAndSaysWhenTheTagIsGone(t *testing.T) {
	t.Parallel()

	base := openStore(t)
	ctx := t.Context()

	tag, err := base.CreateTag(ctx, store.Tag{Name: "docs", Color: "blue"})
	if err != nil {
		t.Fatalf("creando la etiqueta: %v", err)
	}

	name, err := base.TagName(ctx, tag.ID)
	if err != nil {
		t.Fatalf("leyendo el nombre de la etiqueta: %v", err)
	}

	if name != "docs" {
		t.Errorf("la etiqueta %d se llama %q, se esperaba %q", tag.ID, name, "docs")
	}

	if _, err := base.TagName(ctx, 404); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("TagName devolvió %v, se esperaba ErrNotFound", err)
	}
}

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
