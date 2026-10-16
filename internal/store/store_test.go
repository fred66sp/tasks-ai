package store_test

import (
	"errors"
	"path/filepath"
	"tasks-ai/internal/store"
	"testing"
)

func TestOpen_createsTheSchemaAndAnnotatesItsVersion(t *testing.T) {
	t.Parallel()

	version, err := openStore(t).SchemaVersion(t.Context())
	if err != nil {
		t.Fatalf("SchemaVersion devolvió %v", err)
	}

	if version != 1 {
		t.Errorf("una base nueva queda en el esquema %d, se esperaba el 1", version)
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

	if version != 1 {
		t.Errorf("reabrir la base la dejó en el esquema %d, se esperaba el 1", version)
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
