package core_test

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"tasks-ai/internal/core"
	"tasks-ai/internal/store"
	"testing"
)

const (
	startedStatus = "in_progress"
	renamedTitle  = "Renombrada"
)

// TestAllComments_namesEachCommentWithItsTaskReference comprueba que el índice
// de comentarios del proyecto nombra cada comentario con la referencia visible
// de la tarea que lo lleva, compuesta con el prefijo del proyecto.
func TestAllComments_namesEachCommentWithItsTaskReference(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "px")
	ctx := t.Context()

	none, err := apply.AllComments(ctx)
	if err != nil {
		t.Fatalf("leyendo los comentarios de una base vacía: %v", err)
	}

	if len(none) != 0 {
		t.Errorf("una base sin comentarios respondió %+v, se esperaba ninguno", none)
	}

	first, err := apply.CreateTask(ctx, core.NewTask{Title: "Primera"})
	if err != nil {
		t.Fatalf("creando la primera tarea: %v", err)
	}

	second, err := apply.CreateTask(ctx, core.NewTask{Title: "Segunda"})
	if err != nil {
		t.Fatalf("creando la segunda tarea: %v", err)
	}

	if _, err := apply.AddComment(ctx, first.Ref, "de la primera"); err != nil {
		t.Fatalf("añadiendo el comentario de la primera: %v", err)
	}

	if _, err := apply.AddComment(ctx, second.Ref, "de la segunda"); err != nil {
		t.Fatalf("añadiendo el comentario de la segunda: %v", err)
	}

	all, err := apply.AllComments(ctx)
	if err != nil {
		t.Fatalf("leyendo todos los comentarios: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("el índice trae %d comentarios, se esperaban 2", len(all))
	}

	if all[0].Ref != first.Ref || all[0].Body != "de la primera" {
		t.Errorf("el primer comentario es %+v, se esperaba el de %q", all[0], first.Ref)
	}

	if all[1].Ref != second.Ref || all[1].Body != "de la segunda" {
		t.Errorf("el segundo comentario es %+v, se esperaba el de %q", all[1], second.Ref)
	}
}

func TestCreateTask_givesTheTaskItsIdentityAndItsReference(t *testing.T) {
	t.Parallel()

	task := createdTask(t)

	if task.ID != 1 || task.Ref != "px-1" {
		t.Errorf("la tarea nace con identidad %d y referencia %q, se esperaban 1 y %q", task.ID, task.Ref, "px-1")
	}

	if task.Title != "Preparar el análisis" {
		t.Errorf("el título guardado es %q, se esperaba sin los espacios de sobra", task.Title)
	}

	if task.Status != openStatus || task.Priority != core.DefaultPriority {
		t.Errorf("la tarea nace como %q y con prioridad %d, se esperaba %q y %d",
			task.Status, task.Priority, openStatus, core.DefaultPriority)
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

	start(t, apply, parent.Ref)

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

	start(t, apply, task.Ref)

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

// TestCloseTask_refusesDoneWhenTheWorkWasNotStarted deja escrito que cerrar como
// hecha es lo último: una tarea que sigue pendiente no se cierra con done, y el
// rechazo no deja nada escrito.
func TestCloseTask_refusesDoneWhenTheWorkWasNotStarted(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")
	ctx := t.Context()

	task, err := apply.CreateTask(ctx, core.NewTask{Title: "Sin empezar"})
	if err != nil {
		t.Fatalf("creando la tarea: %v", err)
	}

	_, err = apply.CloseTask(ctx, task.Ref, core.CloseReasonDone)
	if !errors.Is(err, core.ErrConflict) {
		t.Errorf("cerrar como hecha una tarea pendiente devolvió %v, se esperaba el centinela de regla que lo impide", err)
	}

	if err == nil || err.Error() != "cannot close task \"tk-1\" as done: it is not in progress; start it with --status in_progress" {
		t.Errorf("el mensaje del cierre rechazado es %q", err)
	}

	kept, err := apply.Task(ctx, task.Ref)
	if err != nil {
		t.Fatalf("leyendo la tarea rechazada: %v", err)
	}

	if kept.Status != openStatus || kept.CloseReason != nil {
		t.Errorf("el rechazo dejó la tarea en %q con motivo %v, se esperaba pendiente y sin motivo", kept.Status, kept.CloseReason)
	}
}

// TestCloseTask_closesAPendingTaskWithAReasonOtherThanDone deja escrito que una
// tarea puede morir sin haberse empezado: duplicate no exige pasar por curso.
func TestCloseTask_closesAPendingTaskWithAReasonOtherThanDone(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")
	ctx := t.Context()

	task, err := apply.CreateTask(ctx, core.NewTask{Title: "Duplicada"})
	if err != nil {
		t.Fatalf("creando la tarea: %v", err)
	}

	closed, err := apply.CloseTask(ctx, task.Ref, core.CloseReasonDuplicate)
	if err != nil {
		t.Fatalf("cerrando como duplicada: %v", err)
	}

	if closed.Status != "done" || closed.CloseReason == nil || *closed.CloseReason != core.CloseReasonDuplicate {
		t.Errorf("la tarea cerrada es %+v, se esperaba done con motivo duplicate", closed)
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

	inProgress := startedStatus

	progressed, err := apply.UpdateTask(ctx, task.Ref, core.Changes{Status: &inProgress})
	if err != nil {
		t.Fatalf("poniendo la tarea en curso: %v", err)
	}

	if progressed.Status != startedStatus {
		t.Errorf("la tarea quedó en %q, se esperaba in_progress", progressed.Status)
	}

	open := openStatus

	returned, err := apply.UpdateTask(ctx, task.Ref, core.Changes{Status: &open})
	if err != nil {
		t.Fatalf("devolviendo la tarea a pendiente: %v", err)
	}

	if returned.Status != openStatus {
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

		start(t, apply, task.Ref)

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
			status:   startedStatus,
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

	if status.Schema != 7 {
		t.Errorf("el estado trae el esquema %d, se esperaba el 7", status.Schema)
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

	if _, err := apply.UpdateTask(ctx, task.Ref, core.Changes{Title: new(renamedTitle)}); err != nil {
		t.Fatalf("cambiando la tarea: %v", err)
	}

	start(t, apply, task.Ref)

	if _, err := apply.CloseTask(ctx, task.Ref, "done"); err != nil {
		t.Fatalf("cerrando la tarea: %v", err)
	}

	events, err := apply.Events(ctx, 0)
	if err != nil {
		t.Fatalf("Events devolvió %v", err)
	}

	if len(events) != 4 {
		t.Fatalf("cuatro cambios dejaron %d avisos, se esperaban 4", len(events))
	}

	expected := []struct {
		action string
		title  string
	}{
		{store.ActionCreate, renamedTitle},
		{store.ActionUpdate, renamedTitle},
		{store.ActionUpdate, renamedTitle},
		{store.ActionClose, renamedTitle},
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

// TestEvents_readsALongHistoryInMoreThanOneBatch recorre la reconexión desde un
// aviso antiguo: una historia más larga que un lote no puede venir entera de una
// vez —un solo lote con todo sería cargarla toda en memoria—, y los lotes leídos
// uno detrás de otro salen completos, en orden, con su referencia y el título que
// la tarea lleva ahora.
func TestEvents_readsALongHistoryInMoreThanOneBatch(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "px")

	const history = 250

	for i := range history {
		if _, err := apply.CreateTask(t.Context(), core.NewTask{Title: fmt.Sprintf("Tarea %d", i+1)}); err != nil {
			t.Fatalf("creando la tarea %d: %v", i+1, err)
		}
	}

	seen, sizes := readHistoryBatches(t, apply, history)

	// La historia cabe en lotes llenos del tope y, al final, uno más corto: ahí se
	// ve el borde del lote, que es donde una reconexión puede dejar avisos atrás.
	want := slices.Repeat([]int{core.EventBatch}, history/core.EventBatch)
	if tail := history % core.EventBatch; tail > 0 {
		want = append(want, tail)
	}

	if !slices.Equal(sizes, want) {
		t.Errorf("una historia de %d avisos llegó en lotes %v, se esperaban %v", history, sizes, want)
	}

	ordered := make([]int64, history)
	for i := range ordered {
		ordered[i] = int64(i + 1)
	}

	if !slices.Equal(seen, ordered) {
		t.Errorf("los avisos leídos por lotes son %d %v, se esperaban los %d primeros en orden",
			len(seen), seen, history)
	}
}

// readHistoryBatches lee la historia entera por lotes, como la recorre quien se
// reconecta: sigue desde el último aviso del lote hasta que un lote sale vacío. Cada
// aviso tiene que salir con la referencia que le toca y el título que su tarea lleva
// ahora, sea el lote que sea. Devuelve los ids leídos y el tamaño de cada lote.
func readHistoryBatches(t *testing.T, apply *core.Core, history int) ([]int64, []int) {
	t.Helper()

	ctx := t.Context()

	var (
		after int64
		seen  []int64
		sizes []int
	)

	for range history + 1 {
		batch, err := apply.Events(ctx, after)
		if err != nil {
			t.Fatalf("Events devolvió %v", err)
		}

		if len(batch) == 0 {
			break
		}

		sizes = append(sizes, len(batch))

		for _, event := range batch {
			want := fmt.Sprintf("Tarea %d", event.ID)
			if event.Ref != fmt.Sprintf("px-%d", event.ID) || event.Title != want {
				t.Fatalf("el aviso %d dice %q con título %q, se esperaba %q", event.ID, event.Ref, event.Title, want)
			}

			seen = append(seen, event.ID)
		}

		after = batch[len(batch)-1].ID
	}

	return seen, sizes
}

// TestEvents_composeTheNameEachTagCarriesNowAndLeaveTheDeletedOneWithoutName
// recorre el lote de etiquetas: la que sigue en el catálogo sale con el nombre que
// lleva ahora, y la que se borró se queda sin nombre —como hasta ahora— en vez de
// romper la lectura de los avisos que la nombran.
func TestEvents_composeTheNameEachTagCarriesNowAndLeaveTheDeletedOneWithoutName(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "px")
	ctx := t.Context()

	kept, err := apply.CreateTag(ctx, "docs", "", "")
	if err != nil {
		t.Fatalf("creando la etiqueta que queda: %v", err)
	}

	gone, err := apply.CreateTag(ctx, "gone", "", "")
	if err != nil {
		t.Fatalf("creando la etiqueta a borrar: %v", err)
	}

	if _, err := apply.EditTag(ctx, kept.Name, core.TagChanges{Name: new("documentacion")}); err != nil {
		t.Fatalf("renombrando la etiqueta: %v", err)
	}

	if _, _, err := apply.DeleteTag(ctx, gone.Name); err != nil {
		t.Fatalf("borrando la etiqueta: %v", err)
	}

	events, err := apply.Events(ctx, 0)
	if err != nil {
		t.Fatalf("Events devolvió %v: el aviso de una etiqueta borrada no puede romper la lectura", err)
	}

	// Los avisos de la etiqueta borrada salen en su puesto, sin nombre: el que ya
	// no está no tiene cómo nombrarse.
	want := []string{"documentacion", "", "documentacion", ""}

	if len(events) != len(want) {
		t.Fatalf("cuatro cambios dejaron %d avisos, se esperaban %d", len(events), len(want))
	}

	for i, event := range events {
		if event.Resource != store.ResourceTag || event.Ref != want[i] || event.Title != "" {
			t.Errorf("el aviso %d es %+v, se esperaba la etiqueta %q sin título", i+1, event, want[i])
		}
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

	start(t, apply, blocker.Ref)

	if _, err := apply.CloseTask(t.Context(), blocker.Ref, core.CloseReasonDone); err != nil {
		t.Fatalf("cerrando la bloqueante: %v", err)
	}

	if listedTask(t, listedTasks(t, apply), blocked.Ref).Blocked {
		t.Error("cerrar la bloqueante no dejó de bloquear a la bloqueada")
	}
}

// TestComments_areCreatedListedAndEditedUnderTheirIdentity recorre el histórico
// por el núcleo: se añaden varios, se leen del primero al último con su
// identidad y su fecha, y se edita uno conservando las dos, con la tarea tal como
// queda y su aviso actualizado.
//
//nolint:gocyclo // el recorrido completo del criterio debe quedarse lineal
func TestComments_areCreatedListedAndEditedUnderTheirIdentity(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")
	ctx := t.Context()

	task := newTask(t, apply, "Con histórico")

	for _, body := range []string{"  el primero  ", "el segundo"} {
		changed, err := apply.AddComment(ctx, task.Ref, body)
		if err != nil {
			t.Fatalf("añadiendo el comentario %q: %v", body, err)
		}

		if changed.Ref != task.Ref {
			t.Errorf("añadir un comentario devolvió la tarea %+v, se esperaba %q", changed, task.Ref)
		}
	}

	listed := listedComments(t, apply, task.Ref)

	if len(listed) != 2 || listed[0].Body != "el primero" || listed[1].Body != "el segundo" {
		t.Fatalf("el histórico lista %+v, se esperaban los dos cuerpos sin los espacios de sobra", listed)
	}

	if listed[0].ID == 0 || listed[1].ID <= listed[0].ID {
		t.Errorf("los comentarios salen identificados como %+v, se esperaba su número dentro de la tarea", listed)
	}

	if listed[0].CreatedAt.IsZero() || listed[1].CreatedAt.IsZero() {
		t.Errorf("los comentarios salen sin fecha de creación: %+v", listed)
	}

	created, err := apply.EditComment(ctx, task.Ref, listed[0].ID, "corregido")
	if err != nil {
		t.Fatalf("editando el comentario %d: %v", listed[0].ID, err)
	}

	if created.Ref != task.Ref {
		t.Errorf("editar un comentario devolvió la tarea %+v, se esperaba %q", created, task.Ref)
	}

	after := listedComments(t, apply, task.Ref)

	if len(after) != 2 || after[0].ID != listed[0].ID || after[0].Body != "corregido" {
		t.Fatalf("tras editar, el histórico es %+v, se esperaba el comentario %d corregido", after, listed[0].ID)
	}

	if !after[0].CreatedAt.Equal(listed[0].CreatedAt) {
		t.Errorf("la edición cambió la fecha a %s, se esperaba %s", after[0].CreatedAt, listed[0].CreatedAt)
	}

	if after[1].ID != listed[1].ID || after[1].Body != listed[1].Body ||
		!after[1].CreatedAt.Equal(listed[1].CreatedAt) {
		t.Errorf("el comentario vecino cambió a %+v, se esperaba %+v", after[1], listed[1])
	}

	detail, err := apply.TaskDetail(ctx, task.Ref)
	if err != nil {
		t.Fatalf("TaskDetail devolvió %v", err)
	}

	if len(detail.Comments) != 2 || detail.Comments[0].ID != listed[0].ID ||
		detail.Comments[0].Body != "corregido" {
		t.Errorf("el detalle trae los comentarios %+v, se esperaban con su identidad", detail.Comments)
	}

	events, err := apply.Events(ctx, 0)
	if err != nil {
		t.Fatalf("Events devolvió %v", err)
	}

	// Un aviso por cambio: el alta de la tarea, los dos comentarios y la edición.
	if len(events) != 4 {
		t.Fatalf("cuatro cambios dejaron %d avisos: %+v", len(events), events)
	}

	last := events[len(events)-1]

	if last.Resource != store.ResourceTask || last.Ref != task.Ref || last.Action != store.ActionUpdate {
		t.Errorf("el aviso de la edición es %+v, se esperaba la tarea %q actualizada", last, task.Ref)
	}
}

// TestComments_shareTheRuleOfTheBodyAndRefuseWhatIsNotTheirs recorre la regla
// compartida por todas las superficies: el cuerpo vacío se rechaza igual al añadir
// que al editar, y una identidad que no es de la tarea o una referencia que no
// resuelve se rechazan sin tocar el histórico.
func TestComments_shareTheRuleOfTheBodyAndRefuseWhatIsNotTheirs(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")
	ctx := t.Context()

	task := newTask(t, apply, "Con histórico")
	elsewhere := newTask(t, apply, "Otra tarea")

	if _, err := apply.AddComment(ctx, task.Ref, "el suyo"); err != nil {
		t.Fatalf("añadiendo el comentario: %v", err)
	}

	if _, err := apply.AddComment(ctx, elsewhere.Ref, "el de otra tarea"); err != nil {
		t.Fatalf("añadiendo el comentario de otra tarea: %v", err)
	}

	mine := listedComments(t, apply, task.Ref)[0]
	theirs := listedComments(t, apply, elsewhere.Ref)[0]

	_, added := apply.AddComment(ctx, task.Ref, "   ")
	if !errors.Is(added, core.ErrInvalid) {
		t.Errorf("AddComment aceptó un cuerpo vacío y devolvió %v, se esperaba ErrInvalid", added)
	}

	_, edited := apply.EditComment(ctx, task.Ref, mine.ID, "   ")
	if !errors.Is(edited, core.ErrInvalid) {
		t.Errorf("EditComment aceptó un cuerpo vacío y devolvió %v, se esperaba ErrInvalid", edited)
	}

	// La regla es una sola: el mismo cuerpo vacío sale rechazado con la misma
	// palabra, venga de la superficie que venga.
	if added.Error() != edited.Error() {
		t.Errorf("el cuerpo vacío se rechaza con %q al añadir y %q al editar, se esperaba el mismo mensaje",
			added, edited)
	}

	tests := []struct {
		name string
		ref  string
		id   int64
	}{
		{name: "a comment of another task", ref: task.Ref, id: theirs.ID},
		{name: "an identity nobody carries", ref: task.Ref, id: theirs.ID + 100},
		{name: "a reference that resolves to nothing", ref: "tk-404", id: mine.ID},
	}

	if _, err := apply.Comments(ctx, "tk-404"); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("Comments aceptó una referencia que no resuelve y devolvió %v, se esperaba ErrNotFound", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := apply.EditComment(ctx, tt.ref, tt.id, "cambiado"); !errors.Is(err, core.ErrNotFound) {
				t.Errorf("EditComment devolvió %v, se esperaba ErrNotFound", err)
			}

			// Un rechazo no escribe nada: el histórico de la tarea comentada sigue
			// igual que estaba.
			after := listedComments(t, apply, task.Ref)
			if len(after) != 1 || after[0].ID != mine.ID || after[0].Body != "el suyo" {
				t.Errorf("el rechazo dejó el histórico como %+v, se esperaba intacto", after)
			}
		})
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

// start pone la tarea en curso: el paso que pide el cierre como hecha.
func start(t *testing.T, apply *core.Core, ref string) {
	t.Helper()

	inProgress := startedStatus

	if _, err := apply.UpdateTask(t.Context(), ref, core.Changes{Status: &inProgress}); err != nil {
		t.Fatalf("poniendo en curso %s: %v", ref, err)
	}
}

// listedComments devuelve el histórico de la tarea, del primero al último.
func listedComments(t *testing.T, apply *core.Core, ref string) []core.Comment {
	t.Helper()

	comments, err := apply.Comments(t.Context(), ref)
	if err != nil {
		t.Fatalf("Comments devolvió %v", err)
	}

	return comments
}

// changeParent applies the parent change the reference names and returns the
// task as it is left.
func changeParent(t *testing.T, apply *core.Core, ref, parent string) core.Task {
	t.Helper()

	changed, err := apply.UpdateTask(t.Context(), ref, core.Changes{Parent: &parent})
	if err != nil {
		t.Fatalf("cambiando el padre de %q a %q: %v", ref, parent, err)
	}

	return changed
}

// assertTags comprueba que la tarea lleva exactamente esas etiquetas.
func assertTags(t *testing.T, task core.Task, want ...string) {
	t.Helper()

	if !slices.Equal(slices.Sorted(slices.Values(task.Tags)), slices.Sorted(slices.Values(want))) {
		t.Errorf("la tarea %q lleva %v, se esperaba %v", task.Ref, task.Tags, want)
	}
}

// TestUpdateTask_assignsMovesAndRemovesTheParent recorre el ciclo completo del
// cambio de padre: una tarea sin padre recibe uno, se mueve a otro y vuelve a
// su nivel. Las etiquetas no se tocan: las pone quien crea la tarea.
func TestUpdateTask_assignsMovesAndRemovesTheParent(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")
	ctx := t.Context()

	if _, err := apply.CreateTag(ctx, "docs", "", ""); err != nil {
		t.Fatalf("creando la etiqueta docs: %v", err)
	}

	epicA := newTask(t, apply, "Épica A")
	epicB := newTask(t, apply, "Épica B")
	child := newTask(t, apply, "La que se mueve")
	other := newTask(t, apply, "Otra épica")

	docs := []string{"docs"}

	if _, err := apply.UpdateTask(ctx, child.Ref, core.Changes{Tags: &docs}); err != nil {
		t.Fatalf("poniendo la etiqueta docs: %v", err)
	}

	moved := changeParent(t, apply, child.Ref, epicA.Ref)

	if moved.ParentID == nil || *moved.ParentID != epicA.ID {
		t.Errorf("la tarea quedó con padre %+v, se esperaba %d", moved.ParentID, epicA.ID)
	}

	assertTags(t, moved, "docs")

	moved = changeParent(t, apply, child.Ref, epicB.Ref)

	if moved.ParentID == nil || *moved.ParentID != epicB.ID {
		t.Errorf("la tarea quedó con padre %+v, se esperaba %d", moved.ParentID, epicB.ID)
	}

	assertTags(t, moved, "docs")

	alone := changeParent(t, apply, child.Ref, "")

	if alone.ParentID != nil {
		t.Errorf("la tarea quedó con padre %+v, se esperaba sin padre", alone.ParentID)
	}

	assertTags(t, alone, "docs")

	// El árbol no cambia por el camino: la tarea que nunca fue padre sigue igual.
	intact, err := apply.Task(ctx, other.Ref)
	if err != nil {
		t.Fatalf("leyendo la tarea ajena: %v", err)
	}

	if intact.ParentID != nil {
		t.Errorf("el cambio de padre tocó una tarea ajena: %+v", intact.ParentID)
	}

	assertTags(t, intact)
}

// TestUpdateTask_leavesTheCatalogueAlone comprueba que el cambio de padre no
// escribe en el catálogo: el producto no pone etiquetas estructurales, así que
// un catálogo vacío se queda vacío por muchos padres que se asignen.
func TestUpdateTask_leavesTheCatalogueAlone(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")
	ctx := t.Context()

	epic := newTask(t, apply, "Épica")
	child := newTask(t, apply, "Hija")

	assign := epic.Ref

	moved, err := apply.UpdateTask(ctx, child.Ref, core.Changes{Parent: &assign})
	if err != nil {
		t.Fatalf("asignando el padre sobre un catálogo vacío: %v", err)
	}

	assertTags(t, moved)

	if tags := catalogTags(t, apply); len(tags) != 0 {
		t.Errorf("el cambio de padre creó etiquetas en el catálogo: %v", tags)
	}
}

// TestUpdateTask_refusesTheParentChangesTheRulesRefuse recorre los cambios de
// padre que las reglas se niegan a escribir: una referencia que no resuelve y
// los ciclos —consigo misma y con una de sus descendientes—. Nada de ellos
// deja nada escrito.
func TestUpdateTask_refusesTheParentChangesTheRulesRefuse(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")
	ctx := t.Context()

	epic := newTask(t, apply, "Épica")
	child := newTask(t, apply, "Hija")

	assign := epic.Ref

	if _, err := apply.UpdateTask(ctx, child.Ref, core.Changes{Parent: &assign}); err != nil {
		t.Fatalf("asignando el padre %q: %v", epic.Ref, err)
	}

	refusals := []struct {
		name   string
		ref    string
		parent string
		want   error
	}{
		{"padre que no resuelve", child.Ref, "tk-999", core.ErrNotFound},
		{"padre en sí misma", epic.Ref, epic.Ref, core.ErrInvalid},
		{"padre en su propia descendiente", epic.Ref, child.Ref, core.ErrInvalid},
	}

	for _, refusal := range refusals {
		_, err := apply.UpdateTask(ctx, refusal.ref, core.Changes{Parent: &refusal.parent})
		if !errors.Is(err, refusal.want) {
			t.Errorf("%s devolvió %v, se esperaba el centinela %v", refusal.name, err, refusal.want)
		}
	}

	after, err := apply.Task(ctx, epic.Ref)
	if err != nil {
		t.Fatalf("leyendo la épica tras los rechazos: %v", err)
	}

	if after.ParentID != nil {
		t.Errorf("un cambio de padre rechazado dejó escrito un padre: %+v", after.ParentID)
	}

	assertTags(t, after)
}

// catalogTags devuelve el catálogo de etiquetas tal como lo responde el núcleo.
func catalogTags(t *testing.T, apply *core.Core) []core.TagWithUses {
	t.Helper()

	tags, err := apply.Tags(t.Context())
	if err != nil {
		t.Fatalf("Tags devolvió %v", err)
	}

	return tags
}

// TestUpdateTask_allowsAClosedParent deja escrito con una prueba que el padre
// puede estar cerrado: una tarea cerrada admite subtareas nuevas, igual que
// una épica cerrada conserva las que ya tenía. Colgar de ella no la reabre ni
// cambia su estado.
func TestUpdateTask_allowsAClosedParent(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")
	ctx := t.Context()

	closed := newTask(t, apply, "Épica cerrada")

	start(t, apply, closed.Ref)

	if _, err := apply.CloseTask(ctx, closed.Ref, core.CloseReasonDone); err != nil {
		t.Fatalf("cerrando la épica: %v", err)
	}

	child := newTask(t, apply, "Hija tardía")

	assign := closed.Ref

	moved, err := apply.UpdateTask(ctx, child.Ref, core.Changes{Parent: &assign})
	if err != nil {
		t.Fatalf("colgando de una tarea cerrada: %v", err)
	}

	if moved.ParentID == nil || *moved.ParentID != closed.ID {
		t.Errorf("la tarea quedó con padre %+v, se esperaba %d", moved.ParentID, closed.ID)
	}

	assertTags(t, moved)

	parent, err := apply.Task(ctx, closed.Ref)
	if err != nil {
		t.Fatalf("leyendo el padre cerrado: %v", err)
	}

	if parent.Status != "done" {
		t.Errorf("colgar una subtarea cambió el estado del padre: %q", parent.Status)
	}
}

// TestCreateTask_withAParentCarriesNoTags deja escrito que la tarea que nace
// con padre no recibe etiqueta ninguna: la estructural la pone quien crea la
// tarea, igual que la que lo gana después.
func TestCreateTask_withAParentCarriesNoTags(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")
	ctx := t.Context()

	epic := newTask(t, apply, "Épica")

	child, err := apply.CreateTask(ctx, core.NewTask{Title: "Hija", Parent: epic.Ref})
	if err != nil {
		t.Fatalf("creando la subtarea: %v", err)
	}

	created := listedTask(t, listedTasks(t, apply), child.Ref)

	assertTags(t, created)

	if tags := catalogTags(t, apply); len(tags) != 0 {
		t.Errorf("la creación con padre creó etiquetas en el catálogo: %v", tags)
	}
}
