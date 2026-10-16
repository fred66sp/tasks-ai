package core_test

import (
	"path/filepath"
	"tasks-ai/internal/core"
	"tasks-ai/internal/store"
	"testing"
)

func TestCreateTask_givesTheTaskItsIdentityAndItsReference(t *testing.T) {
	t.Parallel()

	task := createdTask(t)

	if task.ID != 1 || task.Ref != "px-1" {
		t.Errorf("la tarea nace con identidad %d y referencia %q, se esperaban 1 y %q", task.ID, task.Ref, "px-1")
	}

	if task.Title != "Preparar el análisis" {
		t.Errorf("el título guardado es %q, se esperaba sin los espacios de sobra", task.Title)
	}

	if task.Status != "open" || task.Priority != core.DefaultPriority {
		t.Errorf("la tarea nace como %q y con prioridad %d, se esperaba %q y %d",
			task.Status, task.Priority, "open", core.DefaultPriority)
	}

	if task.Description == nil || *task.Description != "con detalle" {
		t.Errorf("la descripción es %v, se esperaba %q", task.Description, "con detalle")
	}
}

func TestCreateTask_leavesWithoutValueWhatWasNotAskedFor(t *testing.T) {
	t.Parallel()

	task := createdTask(t)

	if task.Assignee != nil || task.Notes != nil || task.CloseReason != nil || task.ParentID != nil {
		t.Errorf("lo que no se pidió debe salir sin valor: %+v", task)
	}

	if task.Tags == nil {
		t.Error("las etiquetas deben salir como lista vacía, no sin valor")
	}
}

func TestCreateTask_needsATitle(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")

	if _, err := apply.CreateTask(t.Context(), core.NewTask{Title: "   "}); err == nil {
		t.Error("CreateTask aceptó una tarea sin título")
	}
}

func TestTask_returnsTheTaskTheReferencePointsAt(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")
	ctx := t.Context()

	created, err := apply.CreateTask(ctx, core.NewTask{Title: "Leer por referencia"})
	if err != nil {
		t.Fatalf("CreateTask devolvió %v", err)
	}

	read, err := apply.Task(ctx, "  TK-1  ")
	if err != nil {
		t.Fatalf("Task devolvió %v", err)
	}

	if read.ID != created.ID || read.Ref != created.Ref || read.Title != created.Title {
		t.Errorf("la tarea leída es %+v, se esperaba %+v", read, created)
	}
}

func TestTask_distinguishesAMissingTaskFromAnotherPrefix(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")
	ctx := t.Context()

	if _, err := apply.Task(ctx, "tk-404"); err == nil {
		t.Error("Task aceptó una referencia que no está en la base")
	}

	if _, err := apply.Task(ctx, "otro-1"); err == nil {
		t.Error("Task aceptó una referencia del prefijo de otro proyecto")
	}
}

func TestCheckPrefix_recordsItTheFirstTimeAndWarnsWhenItChanges(t *testing.T) {
	t.Parallel()

	base := openStore(t)
	ctx := t.Context()

	note, err := core.New(base, "tk").CheckPrefix(ctx)
	if err != nil {
		t.Fatalf("CheckPrefix devolvió %v", err)
	}

	if note != "" {
		t.Errorf("la primera vez no hay nada que avisar, y avisó de %q", note)
	}

	note, err = core.New(base, "tk").CheckPrefix(ctx)
	if err != nil {
		t.Fatalf("CheckPrefix devolvió %v", err)
	}

	if note != "" {
		t.Errorf("con el mismo prefijo no hay nada que avisar, y avisó de %q", note)
	}

	note, err = core.New(base, "px").CheckPrefix(ctx)
	if err != nil {
		t.Fatalf("CheckPrefix devolvió %v", err)
	}

	if note == "" {
		t.Error("cambiar el prefijo debía avisar de que las referencias anteriores ya no resuelven")
	}

	recorded, err := base.RecordedPrefix(ctx)
	if err != nil {
		t.Fatalf("RecordedPrefix devolvió %v", err)
	}

	if recorded != "tk" {
		t.Errorf("el aviso anotó el prefijo %q en la base, debía dejar el que había", recorded)
	}
}

// createdTask crea una tarea con título y descripción en una base nueva con el
// prefijo "px".
func createdTask(t *testing.T) core.Task {
	t.Helper()

	apply := core.New(openStore(t), "px")

	task, err := apply.CreateTask(t.Context(), core.NewTask{
		Title:       "  Preparar el análisis  ",
		Description: "con detalle",
	})
	if err != nil {
		t.Fatalf("CreateTask devolvió %v", err)
	}

	return task
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
