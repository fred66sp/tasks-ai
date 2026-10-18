package core_test

import (
	"errors"
	"path/filepath"
	"slices"
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

	if err == nil || err.Error() != "cannot close task \"tk-1\": it has open subtasks" {
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

	if err == nil || err.Error() != "cannot close task \"tk-1\": it is already closed" {
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
			message:  "only an open task can be put in progress",
		},
		{
			name:     "cerrar no va por el cambio, va por close con su motivo",
			prepare:  pendingRef,
			status:   "done",
			sentinel: core.ErrInvalid,
			message:  "status \"done\" is not set here: closing goes through close, with its reason",
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

	if err == nil || err.Error() != "a tag with the name \"docs\" already exists" {
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

	if status.Schema != 5 {
		t.Errorf("el estado trae el esquema %d, se esperaba el 5", status.Schema)
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
// su origen y lo que le pasó a ese recurso.
//
//nolint:gocyclo // the helper asserts each observable field of both event resources
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

	if first.Action != store.ActionCreate || first.Title != "Con aviso" {
		t.Errorf("el aviso de la tarea dice %q como %q, se esperaba un alta de “Con aviso”", first.Action, first.Title)
	}

	if second.ID != 2 || second.Resource != "tag" || second.Ref != tag.Name || second.Origin != "cli" {
		t.Errorf("el aviso de la etiqueta es %+v, se esperaba la etiqueta %q pedida por el CLI", second, tag.Name)
	}

	// El aviso de una etiqueta no habla de ninguna tarea: no lleva título.
	if second.Action != store.ActionCreate || second.Title != "" {
		t.Errorf("el aviso de la etiqueta dice %q con título %q, se esperaba un alta sin título", second.Action, second.Title)
	}
}

// TestEvents_sayTheTitleTheTaskHasNow recorre el título del aviso: cada aviso
// dice el título que la tarea tiene cuando se lee, no el que tenía cuando se
// escribió, y cada aviso dice lo que pasó en su propio cambio, aunque sean
// varios cambios de la misma tarea.
func TestEvents_sayTheTitleTheTaskHasNow(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "px")
	ctx := t.Context()

	task, err := apply.CreateTask(ctx, core.NewTask{Title: "Con aviso"})
	if err != nil {
		t.Fatalf("creando la tarea: %v", err)
	}

	if _, err := apply.UpdateTask(ctx, task.Ref, core.Changes{Title: new("Renombrada")}); err != nil {
		t.Fatalf("cambiando la tarea: %v", err)
	}

	if _, err := apply.CloseTask(ctx, task.Ref, "done"); err != nil {
		t.Fatalf("cerrando la tarea: %v", err)
	}

	events, err := apply.Events(ctx, 0)
	if err != nil {
		t.Fatalf("Events devolvió %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("tres cambios dejaron %d avisos, se esperaban 3", len(events))
	}

	expected := []struct {
		action string
		title  string
	}{
		{store.ActionCreate, "Renombrada"},
		{store.ActionUpdate, "Renombrada"},
		{store.ActionClose, "Renombrada"},
	}

	for i, want := range expected {
		if events[i].Action != want.action || events[i].Title != want.title {
			t.Errorf("el aviso %d dice %q como %q, se esperaba %q como %q", i+1,
				events[i].Action, events[i].Title, want.action, want.title)
		}
	}
}

// TestEvents_doNotGiveATagTheTitleOfTheTaskThatSharesItsNumber recorre el aviso
// de una etiqueta sobre una base donde tarea y etiqueta llevan el mismo número
// interno: el aviso de la etiqueta no habla de ninguna tarea, así que no tiene
// título que tomar prestado.
func TestEvents_doNotGiveATagTheTitleOfTheTaskThatSharesItsNumber(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "px")
	ctx := t.Context()

	task, err := apply.CreateTask(ctx, core.NewTask{Title: "La tarea"})
	if err != nil {
		t.Fatalf("creando la tarea: %v", err)
	}

	tag, err := apply.CreateTag(ctx, "docs", "", "")
	if err != nil {
		t.Fatalf("creando la etiqueta: %v", err)
	}

	if task.ID != tag.ID {
		t.Fatalf("la prueba necesita que tarea y etiqueta compartan número, son %d y %d", task.ID, tag.ID)
	}

	events, err := apply.Events(ctx, 0)
	if err != nil {
		t.Fatalf("Events devolvió %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("dos cambios dejaron %d avisos, se esperaban 2", len(events))
	}

	if events[0].Resource != store.ResourceTask || events[0].Title != "La tarea" {
		t.Errorf("el aviso de la tarea es %+v, se esperaba la tarea con su título", events[0])
	}

	if events[1].Resource != store.ResourceTag || events[1].Title != "" {
		t.Errorf("el aviso de la etiqueta es %+v, se esperaba una etiqueta sin título", events[1])
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

// TestTasks_resolvesTheBlockedStateAndTheTagsOfEveryTask recorre la regla del
// listado: el estado bloqueado —que se calcula de las dependencias y no se
// guarda— y las etiquetas de cada tarea, resueltos para todas a la vez. Un
// proyecto sin tareas no pregunta nada a la base.
func TestTasks_resolvesTheBlockedStateAndTheTagsOfEveryTask(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")

	if empty := listedTasks(t, apply); len(empty) != 0 {
		t.Errorf("un proyecto sin tareas lista %d, se esperaba ninguna", len(empty))
	}

	newTag(t, apply, "urgente")
	newTag(t, apply, "docs")

	blocked := newTask(t, apply, "Bloqueada")
	blocker := newTask(t, apply, "Bloqueante")
	plain := newTask(t, apply, "Suelta")

	tags := []string{"urgente", "docs"}

	if _, err := apply.UpdateTask(t.Context(), blocked.Ref, core.Changes{Tags: &tags}); err != nil {
		t.Fatalf("poniendo las etiquetas: %v", err)
	}

	if _, err := apply.BlockTask(t.Context(), blocked.Ref, blocker.Ref); err != nil {
		t.Fatalf("bloqueando la tarea: %v", err)
	}

	listed := listedTasks(t, apply)

	if len(listed) != 3 {
		t.Fatalf("el listado devolvió %d tareas, se esperaban 3", len(listed))
	}

	blockedListed := listedTask(t, listed, blocked.Ref)

	if !blockedListed.Blocked {
		t.Errorf("la tarea bloqueada sale sin estarlo: %+v", blockedListed)
	}

	// Las etiquetas salen ordenadas por nombre, no por el orden en que se pidieron.
	if want := []string{"docs", "urgente"}; !slices.Equal(blockedListed.Tags, want) {
		t.Errorf("las etiquetas de %q salen como %v, se esperaban %v", blocked.Ref, blockedListed.Tags, want)
	}

	if listedTask(t, listed, blocker.Ref).Blocked {
		t.Error("la tarea que bloquea sale bloqueada ella misma")
	}

	if plainListed := listedTask(t, listed, plain.Ref); plainListed.Blocked || len(plainListed.Tags) != 0 {
		t.Errorf("la tarea suelta sale con bloqueo %v y etiquetas %v, se esperaba ninguna de las dos cosas",
			plainListed.Blocked, plainListed.Tags)
	}
}

// TestTasks_recomputesTheBlockedStateInsteadOfStoringIt comprueba la otra mitad
// de la regla: el bloqueo se calcula de las dependencias, así que cerrar la
// bloqueante deja de bloquear a la bloqueada sin escribir nada más.
func TestTasks_recomputesTheBlockedStateInsteadOfStoringIt(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")

	blocked := newTask(t, apply, "Bloqueada")
	blocker := newTask(t, apply, "Bloqueante")

	if _, err := apply.BlockTask(t.Context(), blocked.Ref, blocker.Ref); err != nil {
		t.Fatalf("bloqueando la tarea: %v", err)
	}

	if !listedTask(t, listedTasks(t, apply), blocked.Ref).Blocked {
		t.Fatal("la tarea bloqueada no sale bloqueada teniendo su bloqueante abierto")
	}

	if _, err := apply.CloseTask(t.Context(), blocker.Ref, core.CloseReasonDone); err != nil {
		t.Fatalf("cerrando la bloqueante: %v", err)
	}

	if listedTask(t, listedTasks(t, apply), blocked.Ref).Blocked {
		t.Error("cerrar la bloqueante no dejó de bloquear a la bloqueada")
	}
}

// listedTasks devuelve el listado del proyecto.
func listedTasks(t *testing.T, apply *core.Core) []core.Task {
	t.Helper()

	listed, err := apply.Tasks(t.Context())
	if err != nil {
		t.Fatalf("Tasks devolvió %v", err)
	}

	return listed
}

// newTag crea una etiqueta con ese nombre en el núcleo que se le pasa.
func newTag(t *testing.T, apply *core.Core, name string) {
	t.Helper()

	if _, err := apply.CreateTag(t.Context(), name, "", ""); err != nil {
		t.Fatalf("creando la etiqueta %q: %v", name, err)
	}
}

// listedTask devuelve la tarea que el listado trae con esa referencia.
func listedTask(t *testing.T, listed []core.Task, ref string) core.Task {
	t.Helper()

	for _, task := range listed {
		if task.Ref == ref {
			return task
		}
	}

	t.Fatalf("el listado no trae la tarea %q", ref)

	return core.Task{}
}

// newTask crea una tarea con ese título en el núcleo que se le pasa.
func newTask(t *testing.T, apply *core.Core, title string) core.Task {
	t.Helper()

	task, err := apply.CreateTask(t.Context(), core.NewTask{Title: title})
	if err != nil {
		t.Fatalf("creando la tarea %q: %v", title, err)
	}

	return task
}
