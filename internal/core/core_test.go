package core_test

import (
	"errors"
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

func TestTask_saysWithItsSentinelWhenTheReferenceResolvesToNothing(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")

	_, err := apply.Task(t.Context(), "tk-404")
	if !errors.Is(err, core.ErrNotFound) {
		t.Errorf("Task devolvió %v, se esperaba el centinela de referencia sin resolución", err)
	}

	if errors.Is(err, core.ErrInvalid) {
		t.Error("una referencia que no resuelve no es entrada inválida")
	}
}

func TestCloseTask_saysWithItsSentinelWhenARuleRefusesIt(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")
	ctx := t.Context()

	parent, err := apply.CreateTask(ctx, core.NewTask{Title: "Padre"})
	if err != nil {
		t.Fatalf("creando la tarea padre: %v", err)
	}

	if _, err := apply.CreateTask(ctx, core.NewTask{Title: "Hija", Parent: parent.Ref}); err != nil {
		t.Fatalf("creando la tarea hija: %v", err)
	}

	_, err = apply.CloseTask(ctx, parent.Ref, core.CloseReasonDone)
	if !errors.Is(err, core.ErrConflict) {
		t.Errorf("cerrar con subtareas abiertas devolvió %v, se esperaba el centinela de regla que lo impide", err)
	}

	if err == nil || err.Error() != "no se puede cerrar la tarea \"tk-1\": tiene subtareas abiertas" {
		t.Errorf("el mensaje del cierre rechazado es %q", err)
	}
}

func TestCloseTask_saysWithItsSentinelWhenTheTaskIsAlreadyClosed(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")
	ctx := t.Context()

	task, err := apply.CreateTask(ctx, core.NewTask{Title: "Con motivo"})
	if err != nil {
		t.Fatalf("creando la tarea: %v", err)
	}

	if _, err := apply.CloseTask(ctx, task.Ref, core.CloseReasonDone); err != nil {
		t.Fatalf("cerrando la tarea: %v", err)
	}

	_, err = apply.CloseTask(ctx, task.Ref, core.CloseReasonDone)
	if !errors.Is(err, core.ErrConflict) {
		t.Errorf("cerrar dos veces devolvió %v, se esperaba el centinela de regla que lo impide", err)
	}

	if err == nil || err.Error() != "no se puede cerrar la tarea \"tk-1\": ya está cerrada" {
		t.Errorf("el mensaje del segundo cierre es %q", err)
	}
}

// TestUpdateTask_movesTheStatusUnderTheRule recorre el ciclo del estado por el
// cambio: una tarea pendiente se pone en curso y vuelve a pendiente.
func TestUpdateTask_movesTheStatusUnderTheRule(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")
	ctx := t.Context()

	task, err := apply.CreateTask(ctx, core.NewTask{Title: "Con estados"})
	if err != nil {
		t.Fatalf("creando la tarea: %v", err)
	}

	inProgress := "in_progress"

	progressed, err := apply.UpdateTask(ctx, task.Ref, core.Changes{Status: &inProgress})
	if err != nil {
		t.Fatalf("poniendo la tarea en curso: %v", err)
	}

	if progressed.Status != "in_progress" {
		t.Errorf("la tarea quedó en %q, se esperaba in_progress", progressed.Status)
	}

	open := "open"

	returned, err := apply.UpdateTask(ctx, task.Ref, core.Changes{Status: &open})
	if err != nil {
		t.Fatalf("devolviendo la tarea a pendiente: %v", err)
	}

	if returned.Status != "open" {
		t.Errorf("la tarea quedó en %q, se esperaba open", returned.Status)
	}

	if returned.CloseReason != nil {
		t.Errorf("volver a pendiente no debe dejar motivo de cierre: %+v", returned.CloseReason)
	}
}

// TestUpdateTask_refusesTheStatusChangesTheRulesRefuse recorre los cambios de
// estado que las reglas se niegan a escribir: ponerse en curso solo cabe desde
// pendiente, y cerrar no va por el cambio porque lleva su motivo en su ruta.
func TestUpdateTask_refusesTheStatusChangesTheRulesRefuse(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")
	ctx := t.Context()

	closedRef := func(t *testing.T) string {
		t.Helper()

		task, err := apply.CreateTask(ctx, core.NewTask{Title: "Cerrada"})
		if err != nil {
			t.Fatalf("creando la tarea: %v", err)
		}

		if _, err := apply.CloseTask(ctx, task.Ref, core.CloseReasonDone); err != nil {
			t.Fatalf("cerrando la tarea: %v", err)
		}

		return task.Ref
	}

	pendingRef := func(t *testing.T) string {
		t.Helper()

		task, err := apply.CreateTask(ctx, core.NewTask{Title: "Pendiente"})
		if err != nil {
			t.Fatalf("creando la tarea: %v", err)
		}

		return task.Ref
	}

	tests := []struct {
		name     string
		prepare  func(t *testing.T) string
		status   string
		sentinel error
		message  string
	}{
		{
			name:     "ponerse en curso solo cabe desde pendiente",
			prepare:  closedRef,
			status:   "in_progress",
			sentinel: core.ErrConflict,
			message:  "solo una tarea pendiente se puede poner en curso",
		},
		{
			name:     "cerrar no va por el cambio, va por close con su motivo",
			prepare:  pendingRef,
			status:   "done",
			sentinel: core.ErrInvalid,
			message:  "el estado \"done\" no se pone: cerrar va por close, con el motivo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ref := tt.prepare(t)

			status := tt.status

			_, err := apply.UpdateTask(ctx, ref, core.Changes{Status: &status})
			if !errors.Is(err, tt.sentinel) {
				t.Errorf("el cambio de estado devolvió %v, se esperaba el centinela %v", err, tt.sentinel)
			}

			if err == nil || err.Error() != tt.message {
				t.Errorf("el mensaje del cambio rechazado es %q, se esperaba %q", err, tt.message)
			}
		})
	}
}

func TestReopenTask_saysWithItsSentinelWhenTheTaskIsNotClosed(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")
	ctx := t.Context()

	task, err := apply.CreateTask(ctx, core.NewTask{Title: "Abierta"})
	if err != nil {
		t.Fatalf("creando la tarea: %v", err)
	}

	_, err = apply.ReopenTask(ctx, task.Ref)
	if !errors.Is(err, core.ErrConflict) {
		t.Errorf("reabrir una tarea abierta devolvió %v, se esperaba el centinela de regla que lo impide", err)
	}
}

func TestCreateTag_saysWithItsSentinelWhenARuleRefusesIt(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")
	ctx := t.Context()

	if _, err := apply.CreateTag(ctx, "docs", "", ""); err != nil {
		t.Fatalf("creando la etiqueta: %v", err)
	}

	_, err := apply.CreateTag(ctx, "docs", "", "")
	if !errors.Is(err, core.ErrConflict) {
		t.Errorf("crear una etiqueta repetida devolvió %v, se esperaba el centinela de regla que lo impide", err)
	}

	if err == nil || err.Error() != "ya existe una etiqueta con el nombre \"docs\"" {
		t.Errorf("el mensaje de la etiqueta repetida es %q", err)
	}
}

func TestEditTag_saysWithItsSentinelWhenTheTagIsMissing(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")

	_, err := apply.EditTag(t.Context(), "docs", core.TagChanges{Color: &[]string{"red"}[0]})
	if !errors.Is(err, core.ErrNotFound) {
		t.Errorf("cambiar una etiqueta que no está devolvió %v, se esperaba el centinela de referencia sin resolución", err)
	}
}

func TestStatus_reportsTheVersionTheSchemaAndThePrefix(t *testing.T) {
	t.Parallel()

	status, err := core.New(openStore(t), "px").Status(t.Context())
	if err != nil {
		t.Fatalf("Status devolvió %v", err)
	}

	if status.Version != core.Version {
		t.Errorf("el estado trae la versión %q, se esperaba %q", status.Version, core.Version)
	}

	if status.Schema != 4 {
		t.Errorf("el estado trae el esquema %d, se esperaba el 4", status.Schema)
	}

	if status.Prefix != "px" {
		t.Errorf("el estado trae el prefijo %q, se esperaba %q", status.Prefix, "px")
	}
}

func TestEvents_speakOfTheChangeWithItsReferenceComposed(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "px")

	task, tag := createTaskAndTag(t, apply)
	speakOfTheTaskAndTheTag(t, apply, task, tag)
	readingAfterTheLastNoticeSeenReturnsOnlyTheLaterOnes(t, apply)
}

// createTaskAndTag deja un cambio de cada recurso en la base y devuelve lo
// creado, para comparar con ello los avisos que la base ha guardado.
func createTaskAndTag(t *testing.T, apply *core.Core) (core.Task, core.Tag) {
	t.Helper()

	task, err := apply.CreateTask(t.Context(), core.NewTask{Title: "Con aviso"})
	if err != nil {
		t.Fatalf("creando la tarea: %v", err)
	}

	tag, err := apply.CreateTag(t.Context(), "docs", "", "")
	if err != nil {
		t.Fatalf("creando la etiqueta: %v", err)
	}

	return task, tag
}

// speakOfTheTaskAndTheTag comprueba que el canal guarda un aviso por cambio y
// que cada uno habla del suyo: su identidad, su recurso, su referencia compuesta
// y su origen.
func speakOfTheTaskAndTheTag(t *testing.T, apply *core.Core, task core.Task, tag core.Tag) {
	t.Helper()

	events, err := apply.Events(t.Context(), 0)
	if err != nil {
		t.Fatalf("Events devolvió %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("dos cambios dejaron %d avisos, se esperaban 2", len(events))
	}

	first, second := events[0], events[1]

	if first.ID != 1 || first.Resource != "task" || first.Ref != task.Ref || first.Origin != "cli" {
		t.Errorf("el aviso de la tarea es %+v, se esperaba la tarea %q pedida por el CLI", first, task.Ref)
	}

	if second.ID != 2 || second.Resource != "tag" || second.Ref != tag.Name || second.Origin != "cli" {
		t.Errorf("el aviso de la etiqueta es %+v, se esperaba la etiqueta %q pedida por el CLI", second, tag.Name)
	}
}

// readingAfterTheLastNoticeSeenReturnsOnlyTheLaterOnes comprueba que leer a
// partir de un aviso deja fuera los anteriores: quien lee continúa donde se
// quedó.
func readingAfterTheLastNoticeSeenReturnsOnlyTheLaterOnes(t *testing.T, apply *core.Core) {
	t.Helper()

	// Leer a partir del aviso 1: el primero ya se vio.
	remaining, err := apply.Events(t.Context(), 1)
	if err != nil {
		t.Fatalf("Events devolvió %v", err)
	}

	if len(remaining) != 1 || remaining[0].ID != 2 {
		t.Errorf("leer a partir del aviso 1 devolvió %+v, se esperaba solo el aviso 2", remaining)
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
