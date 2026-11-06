package http_test

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"tasks-ai/internal/core"
	api "tasks-ai/internal/http"
	"tasks-ai/internal/store"
	"testing"
	"time"
)

// newTestServer levanta la API sobre el núcleo dado y devuelve su dirección.
func newTestServer(t *testing.T, apply *core.Core, log *slog.Logger) string {
	t.Helper()

	server := httptest.NewServer(api.New(apply, log, nil, "127.0.0.1").Handler())
	t.Cleanup(server.Close)

	return server.URL
}

// newServer monta la API sobre una base recién abierta y devuelve también la
// base, para que un test la cierre por debajo y vea el fallo interno.
func newServer(t *testing.T) (string, *store.Store) {
	t.Helper()

	base, err := store.OpenAs(t.Context(), filepath.Join(t.TempDir(), "tasks.db"), store.OriginServer)
	if err != nil {
		t.Fatalf("abriendo la base de prueba: %v", err)
	}

	t.Cleanup(func() { _ = base.Close() })

	apply := core.New(base, "tk")
	log := slog.New(slog.DiscardHandler)

	server := newTestServer(t, apply, log)

	return server, base
}

func TestServer_createsAndReadsTasksThroughTheCore(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	created := createsATaskThroughTheAPI(t, server)
	readsItsDetailWithItsEmptyLists(t, server, created)
	listsItAsTheOnlyTaskOfTheProject(t, server, created)
}

func TestServer_listsParentIDAsTheInternalNumberUsedByConsumers(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	status, body := do(t, server, http.MethodPost, "/api/tasks", `{"title":"Padre"}`)
	if status != http.StatusCreated {
		t.Fatalf("crear el padre respondió %d: %s", status, body)
	}

	status, body = do(t, server, http.MethodPost, "/api/tasks", `{"title":"Hija","parent":"tk-1"}`)
	if status != http.StatusCreated {
		t.Fatalf("crear la hija respondió %d: %s", status, body)
	}

	status, body = do(t, server, http.MethodGet, "/api/tasks", "")
	if status != http.StatusOK {
		t.Fatalf("listar padre e hija respondió %d: %s", status, body)
	}

	var tasks []core.Task
	decode(t, body, &tasks)

	if len(tasks) != 2 || tasks[1].ParentID == nil || *tasks[1].ParentID != 1 {
		t.Fatalf("parent_id de la hija es %+v, se esperaba el número interno 1", tasks)
	}
}

// createsATaskThroughTheAPI crea una tarea por la API y comprueba que llega
// firstRef es la referencia visible que el producto da a la primera tarea de un
// proyecto: las pruebas la comparan con lo que el servidor devuelve.
const (
	rejectedEntry = `{"body": "nada"}`
	firstRef      = "tk-1"
	renamedTitle  = "Renombrada"
	createdAction = "creada"
	updatedAction = "actualizada"
	openStatus    = "open"
	docsTag       = "docs"
)

// taskResource es el recurso en plural que la interfaz compara en los avisos de
// cambio de tarea.
const taskResource = "tasks"

// como se pidió: su referencia, su estado y su prioridad por omisión.
func createsATaskThroughTheAPI(t *testing.T, server string) core.Task {
	t.Helper()

	status, body := do(t, server, http.MethodPost, "/api/tasks", `{"title": "Preparar el análisis", "description": "con detalle"}`)
	if status != http.StatusCreated {
		t.Fatalf("crear por la API respondió %d: %s", status, body)
	}

	var created core.Task

	decode(t, body, &created)

	if created.Ref != firstRef || created.Status != openStatus || created.Priority != core.DefaultPriority {
		t.Errorf("la tarea creada es %+v, se esperaba tk-1 abierta con la prioridad por omisión", created)
	}

	return created
}

// readsItsDetailWithItsEmptyLists lee el detalle de la tarea y comprueba que
// trae la guardada y sus listas alrededor, aunque aún estén vacías.
func readsItsDetailWithItsEmptyLists(t *testing.T, server string, created core.Task) {
	t.Helper()

	status, body := do(t, server, http.MethodGet, "/api/tasks/"+created.Ref, "")
	if status != http.StatusOK {
		t.Fatalf("leer el detalle respondió %d: %s", status, body)
	}

	var detail core.Detail

	decode(t, body, &detail)

	if detail.Ref != created.Ref || detail.Children == nil || detail.Blockers == nil || detail.Dependents == nil || detail.Comments == nil {
		t.Errorf("el detalle trae %+v, se esperaban las listas vacías y la tarea guardada", detail)
	}
}

// listsItAsTheOnlyTaskOfTheProject pide la lista del proyecto y comprueba que
// la tarea creada es la única que trae.
func listsItAsTheOnlyTaskOfTheProject(t *testing.T, server string, created core.Task) {
	t.Helper()

	status, body := do(t, server, http.MethodGet, "/api/tasks", "")
	if status != http.StatusOK {
		t.Fatalf("listar respondió %d: %s", status, body)
	}

	var tasks []core.Task

	decode(t, body, &tasks)

	if len(tasks) != 1 || tasks[0].Ref != created.Ref {
		t.Errorf("la lista trae %+v, se esperaba solo %q", tasks, created.Ref)
	}
}

func TestServer_changesATaskTheWayTheCoreChangesIt(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	patchesWhatTheChangeAsks(t, server)
	blockAndUnblockLeaveTheTaskUnblocked(t, server)
	aCommentAddedArrivesInTheDetail(t, server)
}

// patchesWhatTheChangeAsks cambia la prioridad y las etiquetas de una tarea por
// el cambio de la API y comprueba que la respuesta las lleva.
func patchesWhatTheChangeAsks(t *testing.T, server string) {
	t.Helper()

	do(t, server, http.MethodPost, "/api/tasks", `{"title": "Primera"}`)
	do(t, server, http.MethodPost, "/api/tasks", `{"title": "Segunda"}`)
	do(t, server, http.MethodPost, "/api/tags", `{"name": "`+docsTag+`"}`)

	status, body := do(t, server, http.MethodPatch, "/api/tasks/tk-1", `{"priority": 1, "tags": ["`+docsTag+`"]}`)
	if status != http.StatusOK {
		t.Fatalf("cambiar la tarea respondió %d: %s", status, body)
	}

	var changed core.Task

	decode(t, body, &changed)

	if changed.Priority != 1 || len(changed.Tags) != 1 || changed.Tags[0] != docsTag {
		t.Errorf("la tarea quedó %+v, se esperaba prioridad 1 y la etiqueta docs", changed)
	}
}

// blockAndUnblockLeaveTheTaskUnblocked añade un bloqueo y lo retira por sus
// rutas y comprueba que la tarea queda desbloqueada al final.
func blockAndUnblockLeaveTheTaskUnblocked(t *testing.T, server string) {
	t.Helper()

	do(t, server, http.MethodPost, "/api/tasks", `{"title": "Primera"}`)
	do(t, server, http.MethodPost, "/api/tasks", `{"title": "Segunda"}`)

	// El bloqueo y su retirada van por sus rutas y responden con la tarea.
	status, body := do(t, server, http.MethodPost, "/api/tasks/tk-1/blockers", `{"blocker": "tk-2"}`)
	if status != http.StatusOK {
		t.Fatalf("añadir el bloqueo respondió %d: %s", status, body)
	}

	var blocked core.Task

	decode(t, body, &blocked)

	if !blocked.Blocked {
		t.Errorf("la tarea quedó sin estar bloqueada: %+v", blocked)
	}

	status, _ = do(t, server, http.MethodDelete, "/api/tasks/tk-1/blockers/tk-2", "")
	if status != http.StatusOK {
		t.Fatalf("retirar el bloqueo respondió %d", status)
	}

	status, body = do(t, server, http.MethodGet, "/api/tasks/tk-1", "")
	if status != http.StatusOK {
		t.Fatalf("leer el detalle respondió %d", status)
	}

	var detail core.Detail

	decode(t, body, &detail)

	if detail.Blocked {
		t.Errorf("la tarea sigue bloqueada tras retirar el bloqueo: %+v", detail)
	}
}

// aCommentAddedArrivesInTheDetail añade un comentario a una tarea y comprueba
// que el detalle lo trae.
func aCommentAddedArrivesInTheDetail(t *testing.T, server string) {
	t.Helper()

	do(t, server, http.MethodPost, "/api/tasks", `{"title": "Primera"}`)

	// El comentario se añade y el detalle lo trae.
	status, body := do(t, server, http.MethodPost, "/api/tasks/tk-1/comments", `{"body": "un comentario"}`)
	if status != http.StatusOK {
		t.Fatalf("añadir el comentario respondió %d: %s", status, body)
	}

	status, body = do(t, server, http.MethodGet, "/api/tasks/tk-1", "")
	if status != http.StatusOK {
		t.Fatalf("leer el detalle respondió %d: %s", status, body)
	}

	var detail core.Detail

	decode(t, body, &detail)

	if len(detail.Comments) != 1 || detail.Comments[0].Body != "un comentario" {
		t.Errorf("el detalle trae los comentarios %+v, se esperaba el añadido", detail.Comments)
	}
}

// TestServer_readsAndEditsTheCommentsOfATask recorre el criterio de tk-105: la
// API expone la lectura y la edición de los comentarios de una tarea, responde
// con sus códigos, no muta ante un rechazo y deja el aviso de la tarea
// actualizada que otras sesiones ven.
//
//nolint:gocyclo // el recorrido completo del criterio debe quedarse lineal
func TestServer_readsAndEditsTheCommentsOfATask(t *testing.T) {
	t.Parallel()

	server, base := newServer(t)

	createsATaskThroughTheAPI(t, server)

	addAComment := func(t *testing.T, body string) {
		t.Helper()

		status, body := do(t, server, http.MethodPost, "/api/tasks/"+firstRef+"/comments", `{"body": "`+body+`"}`)
		if status != http.StatusOK {
			t.Fatalf("añadir el comentario respondió %d: %s", status, body)
		}
	}

	addAComment(t, "primero")
	addAComment(t, "segundo")

	// La lectura responde del primero al último, cada uno con su identidad.
	status, body := do(t, server, http.MethodGet, "/api/tasks/"+firstRef+"/comments", "")
	if status != http.StatusOK {
		t.Fatalf("leer los comentarios respondió %d: %s", status, body)
	}

	var listed []core.Comment
	decode(t, body, &listed)

	if len(listed) != 2 || listed[0].Body != "primero" || listed[1].Body != "segundo" {
		t.Fatalf("la lista es %+v, se esperaba primero y luego segundo", listed)
	}

	if listed[0].ID == 0 || listed[0].ID == listed[1].ID {
		t.Errorf("cada comentario debe traer una identidad propia: %+v", listed)
	}

	before := listed

	// La edición responde con la tarea tal como queda.
	status, body = do(t, server, http.MethodPatch, "/api/tasks/"+firstRef+"/comments/2", `{"body": "segundo, corregido"}`)
	if status != http.StatusOK {
		t.Fatalf("editar el comentario respondió %d: %s", status, body)
	}

	var task core.Task
	decode(t, body, &task)

	if task.Ref != firstRef {
		t.Errorf("la respuesta de la edición es %+v, se esperaba la tarea %q", task, firstRef)
	}

	status, body = do(t, server, http.MethodGet, "/api/tasks/"+firstRef+"/comments", "")
	if status != http.StatusOK {
		t.Fatalf("releer los comentarios respondió %d: %s", status, body)
	}

	decode(t, body, &listed)

	if listed[1].ID != before[1].ID || !listed[1].CreatedAt.Equal(before[1].CreatedAt) {
		t.Errorf("la edición debía conservar identidad y fecha: antes %+v, después %+v", before[1], listed[1])
	}

	if listed[1].Body != "segundo, corregido" {
		t.Errorf("el cuerpo editado es %q, se esperaba %q", listed[1].Body, "segundo, corregido")
	}

	if listed[0] != before[0] {
		t.Errorf("el vecino cambió con la edición: antes %+v, después %+v", before[0], listed[0])
	}

	// El aviso del cambio viaja por el canal: una edición deja la tarea
	// actualizada, igual que cuando el comentario llega.
	events, err := core.New(base, "tk").Events(t.Context(), 0)
	if err != nil {
		t.Fatalf("Events devolvió %v", err)
	}

	last := events[len(events)-1]

	if last.Resource != "task" || last.Ref != firstRef || last.Action != store.ActionUpdate {
		t.Errorf("el último aviso es %+v, se esperaba la tarea %q actualizada", last, firstRef)
	}
}

// TestServer_refusesTheCommentOperationsThatDoNotApply recorre los rechazos de
// la lectura y la edición: la entrada vacía es 400, el recurso que no está es
// 404, y ninguno deja nada escrito.
func TestServer_refusesTheCommentOperationsThatDoNotApply(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	createsATaskThroughTheAPI(t, server)

	// Una segunda tarea del mismo proyecto: el número de su comentario no es de
	// la tarea que se edita.
	status, _ := do(t, server, http.MethodPost, "/api/tasks", `{"title": "La vecina"}`)
	if status != http.StatusCreated {
		t.Fatalf("crear la vecina respondió %d", status)
	}

	status, _ = do(t, server, http.MethodPost, "/api/tasks/tk-2/comments", `{"body": "de otra tarea"}`)
	if status != http.StatusOK {
		t.Fatalf("añadir el comentario de la vecina respondió %d", status)
	}

	status, _ = do(t, server, http.MethodPost, "/api/tasks/"+firstRef+"/comments", `{"body": "el mío"}`)
	if status != http.StatusOK {
		t.Fatalf("añadir el comentario respondió %d", status)
	}

	before, _, _ := readComments(t, server, firstRef)

	tests := []struct {
		name    string
		method  string
		path    string
		body    string
		status  int
		message string
	}{
		{
			name:    "the comment body is missing on edit",
			method:  http.MethodPatch,
			path:    "/api/tasks/" + firstRef + "/comments/1",
			body:    `{"body": "  "}`,
			status:  http.StatusBadRequest,
			message: "the comment body is missing",
		},
		{
			name:    "a comment that is not of the task is not found",
			method:  http.MethodPatch,
			path:    "/api/tasks/" + firstRef + "/comments/1",
			body:    rejectedEntry,
			status:  http.StatusNotFound,
			message: `there is no comment 1 in task "tk-1"`,
		},
		{
			name:    "an unknown comment is not found",
			method:  http.MethodPatch,
			path:    "/api/tasks/" + firstRef + "/comments/99",
			body:    rejectedEntry,
			status:  http.StatusNotFound,
			message: `there is no comment 99 in task "tk-1"`,
		},
		{
			name:    "an unknown task is not found",
			method:  http.MethodPatch,
			path:    "/api/tasks/tk-404/comments/1",
			body:    rejectedEntry,
			status:  http.StatusNotFound,
			message: `there is no task with reference "tk-404"`,
		},
		{
			name:    "a comment id that is not a number is a bad entry",
			method:  http.MethodPatch,
			path:    "/api/tasks/" + firstRef + "/comments/uno",
			body:    rejectedEntry,
			status:  http.StatusBadRequest,
			message: `the comment id "uno" is not a number`,
		},
		{
			name:    "the comments of an unknown task are not found",
			method:  http.MethodGet,
			path:    "/api/tasks/tk-404/comments",
			body:    "",
			status:  http.StatusNotFound,
			message: `there is no task with reference "tk-404"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			status, body := do(t, server, test.method, test.path, test.body)
			expect(t, status, body, test.status, test.message)
		})
	}

	after, _, _ := readComments(t, server, firstRef)

	if len(after) != len(before) || len(after) == 0 || after[0].Body != before[0].Body {
		t.Errorf("un rechazo debía dejar los comentarios como estaban: antes %+v, después %+v", before, after)
	}
}

// readComments lee los comentarios de la tarea por la API.
func readComments(t *testing.T, server, ref string) ([]core.Comment, int, string) {
	t.Helper()

	status, body := do(t, server, http.MethodGet, "/api/tasks/"+ref+"/comments", "")
	if status != http.StatusOK {
		return nil, status, body
	}

	var comments []core.Comment
	decode(t, body, &comments)

	return comments, status, body
}

func TestServer_servesTheCatalogueOfTags(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	status, body := do(t, server, "POST", "/api/tags", `{"name": "`+docsTag+`"}`)
	if status != http.StatusCreated {
		t.Fatalf("crear la etiqueta respondió %d: %s", status, body)
	}

	var created core.Tag

	decode(t, body, &created)

	if created.Name != docsTag || created.Color != core.DefaultTagColor {
		t.Errorf("la etiqueta creada es %+v, se esperaba docs con el color por omisión", created)
	}

	status, body = do(t, server, "GET", "/api/tags", "")
	if status != http.StatusOK {
		t.Fatalf("leer el catálogo respondió %d: %s", status, body)
	}

	var catalogue []core.TagWithUses

	decode(t, body, &catalogue)

	if len(catalogue) != 1 || catalogue[0].Uses != 0 {
		t.Errorf("el catálogo trae %+v, se esperaba docs sin uso", catalogue)
	}

	status, body = do(t, server, "PATCH", "/api/tags/docs", `{"color": "red"}`)
	if status != http.StatusOK {
		t.Fatalf("cambiar la etiqueta respondió %d: %s", status, body)
	}

	status, body = do(t, server, "DELETE", "/api/tags/docs", "")
	if status != http.StatusOK {
		t.Fatalf("borrar la etiqueta respondió %d: %s", status, body)
	}

	var deleted core.Tag

	decode(t, body, &deleted)

	if deleted.Name != docsTag {
		t.Errorf("la respuesta del borrado trae %+v, se esperaba la etiqueta borrada", deleted)
	}
}

func TestServer_closesAndReopensThroughTheStatusOfThePatch(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	do(t, server, "POST", "/api/tasks", `{"title": "Con ciclo"}`)

	status, body := do(t, server, "PATCH", "/api/tasks/tk-1", `{"status": "done"}`)
	if status != http.StatusBadRequest {
		t.Errorf("poner done por el cambio respondió %d: %s, se esperaba 400", status, body)
	}

	status, body = do(t, server, "PATCH", "/api/tasks/tk-1", `{"status": "in_progress"}`)
	if status != http.StatusOK {
		t.Fatalf("poner en curso respondió %d: %s", status, body)
	}

	status, body = do(t, server, "POST", "/api/tasks/tk-1/close", `{"reason": "done"}`)
	if status != http.StatusOK {
		t.Fatalf("cerrar respondió %d: %s", status, body)
	}

	var closed core.Task

	decode(t, body, &closed)

	if closed.Status != "done" || closed.CloseReason == nil || *closed.CloseReason != "done" {
		t.Errorf("la tarea cerrada es %+v, se esperaba done con su motivo", closed)
	}

	status, body = do(t, server, "PATCH", "/api/tasks/tk-1", `{"status": "open"}`)
	if status != http.StatusOK {
		t.Fatalf("reabrir por el cambio de estado respondió %d: %s", status, body)
	}

	var reopened core.Task

	decode(t, body, &reopened)

	if reopened.Status != openStatus || reopened.CloseReason != nil {
		t.Errorf("la tarea reabierta es %+v, se esperaba open sin motivo", reopened)
	}
}

// TestServer_putsATaskInProgressThroughThePatch recorre el ciclo del estado
// por el cambio de la API: la tarea se pone en curso, vuelve a pendiente y
// cerrar por el cambio sigue rechazándose con el mensaje de siempre.
func TestServer_putsATaskInProgressThroughThePatch(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	do(t, server, "POST", "/api/tasks", `{"title": "En curso"}`)

	status, body := do(t, server, "PATCH", "/api/tasks/tk-1", `{"status": "in_progress"}`)
	if status != http.StatusOK {
		t.Fatalf("poner en curso respondió %d: %s", status, body)
	}

	var progressed core.Task

	decode(t, body, &progressed)

	if progressed.Status != "in_progress" {
		t.Errorf("la tarea quedó en %q, se esperaba in_progress", progressed.Status)
	}

	status, body = do(t, server, "PATCH", "/api/tasks/tk-1", `{"status": "open"}`)
	if status != http.StatusOK {
		t.Fatalf("volver a pendiente respondió %d: %s", status, body)
	}

	var returned core.Task

	decode(t, body, &returned)

	if returned.Status != openStatus {
		t.Errorf("la tarea quedó en %q, se esperaba open", returned.Status)
	}

	status, body = do(t, server, "PATCH", "/api/tasks/tk-1", `{"status": "done"}`)
	expect(t, status, body, http.StatusBadRequest, `status "done" cannot be set: to close use its route with the reason`)
}

// TestServer_changesTheParentOfATaskThroughThePatch recorre el ciclo del padre
// por el cambio de la API: la tarea sin padre recibe uno, se mueve a otro y
// vuelve a su nivel. Las etiquetas no se tocan: las pone quien crea la tarea.
func TestServer_changesTheParentOfATaskThroughThePatch(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	epicA := createdTask(t, server, `{"title": "Épica A"}`)
	epicB := createdTask(t, server, `{"title": "Épica B"}`)
	child := createdTask(t, server, `{"title": "La que se mueve"}`)

	assigned := patchesTheParent(t, server, child.Ref, epicA.Ref)
	if assigned.ParentID == nil || *assigned.ParentID != epicA.ID {
		t.Errorf("la tarea quedó con padre %+v, se esperaba %d", assigned.ParentID, epicA.ID)
	}

	if len(assigned.Tags) != 0 {
		t.Errorf("la tarea con padre debe quedar sin etiquetas, lleva %v", assigned.Tags)
	}

	moved := patchesTheParent(t, server, child.Ref, epicB.Ref)
	if moved.ParentID == nil || *moved.ParentID != epicB.ID {
		t.Errorf("la tarea quedó con padre %+v, se esperaba %d", moved.ParentID, epicB.ID)
	}

	alone := patchesTheParent(t, server, child.Ref, "")
	if alone.ParentID != nil {
		t.Errorf("la tarea quedó con padre %+v, se esperaba sin padre", alone.ParentID)
	}

	if len(alone.Tags) != 0 {
		t.Errorf("la tarea sin padre debe quedar sin etiquetas, lleva %v", alone.Tags)
	}
}

// TestServer_refusesTheParentChangesTheContractRefuses recorre los rechazos del
// contrato —la referencia que no resuelve y los ciclos, consigo misma y con una
// descendiente— y comprueba que ninguno deja nada escrito.
func TestServer_refusesTheParentChangesTheContractRefuses(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	epic := createdTask(t, server, `{"title": "Épica"}`)
	child := createdTask(t, server, `{"title": "Hija", "parent": "tk-1"}`)

	refusals := []struct {
		name    string
		ref     string
		parent  string
		status  int
		message string
	}{
		{
			name:    "el padre no resuelve",
			ref:     child.Ref,
			parent:  "tk-999",
			status:  http.StatusNotFound,
			message: `there is no task with reference "tk-999"`,
		},
		{
			name:    "la tarea del cambio no resuelve",
			ref:     "tk-404",
			parent:  epic.Ref,
			status:  http.StatusNotFound,
			message: `there is no task with reference "tk-404"`,
		},
		{
			name:    "el padre es la propia tarea",
			ref:     epic.Ref,
			parent:  epic.Ref,
			status:  http.StatusBadRequest,
			message: "the task cannot take itself or one of its own descendants as its parent",
		},
		{
			name:    "el padre es una descendiente",
			ref:     epic.Ref,
			parent:  child.Ref,
			status:  http.StatusBadRequest,
			message: "the task cannot take itself or one of its own descendants as its parent",
		},
	}

	for _, refusal := range refusals {
		t.Run(refusal.name, func(t *testing.T) {
			t.Parallel()

			body := fmt.Sprintf(`{"parent": %q}`, refusal.parent)
			status, answer := do(t, server, http.MethodPatch, "/api/tasks/"+refusal.ref, body)
			expect(t, status, answer, refusal.status, refusal.message)
		})
	}

	// Nada de lo rechazado quedó escrito: la épica sigue en su nivel y sin la
	// etiqueta estructural, y la hija sigue colgando de ella.
	after := readTask(t, server, epic.Ref)
	if after.ParentID != nil {
		t.Errorf("un cambio de padre rechazado dejó escrito un padre: %+v", after.ParentID)
	}

	if len(after.Tags) != 0 {
		t.Errorf("un cambio de padre rechazado dejó escrita alguna etiqueta: %v", after.Tags)
	}

	if kept := readTask(t, server, child.Ref); kept.ParentID == nil || *kept.ParentID != epic.ID {
		t.Errorf("la hija quedó con padre %+v, se esperaba %d", kept.ParentID, epic.ID)
	}
}

// TestServer_refusesTheParentTogetherWithTheStatus comprueba que el estado
// sigue yendo solo en el cambio, también cuando lo que lo acompaña es el padre:
// la entrada es inválida y la tarea queda como estaba.
func TestServer_refusesTheParentTogetherWithTheStatus(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	createdTask(t, server, `{"title": "Padre"}`)
	createdTask(t, server, `{"title": "Para poner en curso"}`)

	status, body := do(t, server, http.MethodPatch, "/api/tasks/tk-2", `{"status": "in_progress", "parent": "tk-1"}`)
	expect(t, status, body, http.StatusBadRequest, "the status goes alone in the change")

	after := readTask(t, server, "tk-2")
	if after.ParentID != nil {
		t.Errorf("la tarea quedó con padre %+v, se esperaba sin padre", after.ParentID)
	}

	if after.Status != openStatus {
		t.Errorf("la tarea quedó en %q, se esperaba open", after.Status)
	}
}

// createdTask crea una tarea por la API y devuelve la guardada.
func createdTask(t *testing.T, server, body string) core.Task {
	t.Helper()

	status, answer := do(t, server, http.MethodPost, "/api/tasks", body)
	if status != http.StatusCreated {
		t.Fatalf("crear la tarea respondió %d: %s", status, answer)
	}

	var created core.Task

	decode(t, answer, &created)

	return created
}

// readTask lee el detalle de la tarea y devuelve la tarea tal como está.
func readTask(t *testing.T, server, ref string) core.Task {
	t.Helper()

	status, answer := do(t, server, http.MethodGet, "/api/tasks/"+ref, "")
	if status != http.StatusOK {
		t.Fatalf("leer %s respondió %d: %s", ref, status, answer)
	}

	var detail core.Detail

	decode(t, answer, &detail)

	return detail.Task
}

// patchesTheParent pide el cambio de padre de la tarea y devuelve la tarea tal
// como la respuesta la trae.
func patchesTheParent(t *testing.T, server, ref, parent string) core.Task {
	t.Helper()

	status, answer := do(t, server, http.MethodPatch, "/api/tasks/"+ref, fmt.Sprintf(`{"parent": %q}`, parent))
	if status != http.StatusOK {
		t.Fatalf("cambiar el padre de %s respondió %d: %s", ref, status, answer)
	}

	var changed core.Task

	decode(t, answer, &changed)

	return changed
}

// errorTranslation nombra una negativa de la API con los pasos que la provocan.
type errorTranslation struct {
	name  string
	steps func(t *testing.T, server string)
}

//nolint:funlen // la enumeración completa de las negativas debe quedarse lineal
func errorTranslations() []errorTranslation {
	return []errorTranslation{
		{
			name: "la entrada no es JSON",
			steps: func(t *testing.T, server string) {
				t.Helper()

				status, body := do(t, server, "POST", "/api/tasks", `{`)
				expect(t, status, body, http.StatusBadRequest, "the body of the request is not valid JSON")
			},
		},
		{
			name: "la entrada lleva contenido después del JSON",
			steps: func(t *testing.T, server string) {
				t.Helper()

				status, body := do(t, server, "POST", "/api/tasks", `{"title":"válida"} basura`)
				expect(t, status, body, http.StatusBadRequest, "the body of the request is not valid JSON")

				status, body = do(t, server, "GET", "/api/tasks", "")
				if status != http.StatusOK {
					t.Fatalf("consultar después del JSON rechazado respondió %d: %s", status, body)
				}

				var tasks []core.Task
				decode(t, body, &tasks)

				if len(tasks) != 0 {
					t.Errorf("el JSON posterior creó %d tareas, se esperaban 0", len(tasks))
				}
			},
		},
		{
			name: "la prioridad está fuera del rango y la entrada llega al núcleo",
			steps: func(t *testing.T, server string) {
				t.Helper()

				status, body := do(t, server, "POST", "/api/tasks", `{"title": "Fuera de rango", "priority": 9}`)
				expect(t, status, body, http.StatusBadRequest, "priority 9 is outside the range 0-3")
			},
		},
		{
			name: "falta el título",
			steps: func(t *testing.T, server string) {
				t.Helper()

				status, body := do(t, server, "POST", "/api/tasks", `{"title": "   "}`)
				expect(t, status, body, http.StatusBadRequest, "the task needs a title")
			},
		},
		{
			name: "el cambio no pide nada",
			steps: func(t *testing.T, server string) {
				t.Helper()

				status, body := do(t, server, "PATCH", "/api/tasks/tk-1", `{}`)
				expect(t, status, body, http.StatusBadRequest, "there is nothing to change: name at least one field")
			},
		},
		{
			name: "the close reason is missing",
			steps: func(t *testing.T, server string) {
				t.Helper()

				do(t, server, "POST", "/api/tasks", `{"title": "Sin motivo"}`)
				status, body := do(t, server, "POST", "/api/tasks/tk-1/close", `{}`)
				expect(t, status, body, http.StatusBadRequest, "the close reason is missing")
			},
		},
		{
			name: "falta la bloqueante",
			steps: func(t *testing.T, server string) {
				t.Helper()

				do(t, server, "POST", "/api/tasks", `{"title": "Sin bloqueante"}`)
				status, body := do(t, server, "POST", "/api/tasks/tk-1/blockers", `{}`)
				expect(t, status, body, http.StatusBadRequest, "the reference of the blocker task is missing")
			},
		},
		{
			name: "the comment body is missing",
			steps: func(t *testing.T, server string) {
				t.Helper()

				do(t, server, "POST", "/api/tasks", `{"title": "Sin comentario"}`)
				status, body := do(t, server, "POST", "/api/tasks/tk-1/comments", `{"body": "  "}`)
				expect(t, status, body, http.StatusBadRequest, "the comment body is missing")
			},
		},
		{
			name: "la referencia no corresponde a ninguna tarea",
			steps: func(t *testing.T, server string) {
				t.Helper()

				status, body := do(t, server, http.MethodGet, "/api/tasks/tk-404", "")
				expect(t, status, body, http.StatusNotFound, "there is no task with reference \"tk-404\"")
			},
		},
		{
			name: "la referencia está mal escrita",
			steps: func(t *testing.T, server string) {
				t.Helper()

				status, body := do(t, server, http.MethodGet, "/api/tasks/luego", "")
				expect(t, status, body, http.StatusNotFound, "reference \"luego\" is not valid: tk-<number> is expected")
			},
		},
		{
			name: "la etiqueta no está",
			steps: func(t *testing.T, server string) {
				t.Helper()

				status, body := do(t, server, "DELETE", "/api/tags/nope", "")
				expect(t, status, body, http.StatusNotFound, "there is no tag with the name \"nope\"")
			},
		},
		{
			name: "cerrar con subtareas abiertas lo impide la regla",
			steps: func(t *testing.T, server string) {
				t.Helper()

				do(t, server, "POST", "/api/tasks", `{"title": "Padre"}`)
				do(t, server, "POST", "/api/tasks", `{"title": "Hija", "parent": "tk-1"}`)
				do(t, server, "PATCH", "/api/tasks/tk-1", `{"status": "in_progress"}`)
				status, body := do(t, server, "POST", "/api/tasks/tk-1/close", `{"reason": "done"}`)
				expect(t, status, body, http.StatusConflict, "cannot close task \"tk-1\": it has open subtasks")
			},
		},
		{
			name: "cerrar como hecha una tarea pendiente lo impide la regla",
			steps: func(t *testing.T, server string) {
				t.Helper()

				do(t, server, "POST", "/api/tasks", `{"title": "Sin empezar"}`)
				status, body := do(t, server, "POST", "/api/tasks/tk-1/close", `{"reason": "done"}`)
				expect(t, status, body, http.StatusConflict, "cannot close task \"tk-1\" as done: it is not in progress; start it with --status in_progress")
			},
		},
		{
			name: "cerrar una tarea ya cerrada lo impide la regla",
			steps: func(t *testing.T, server string) {
				t.Helper()

				do(t, server, "POST", "/api/tasks", `{"title": "Dos cierres"}`)
				do(t, server, "PATCH", "/api/tasks/tk-1", `{"status": "in_progress"}`)
				do(t, server, "POST", "/api/tasks/tk-1/close", `{"reason": "done"}`)
				status, body := do(t, server, "POST", "/api/tasks/tk-1/close", `{"reason": "done"}`)
				expect(t, status, body, http.StatusConflict, "cannot close task \"tk-1\": it is already closed")
			},
		},
		{
			name: "el nombre de la etiqueta ya está en el catálogo",
			steps: func(t *testing.T, server string) {
				t.Helper()

				do(t, server, "POST", "/api/tags", `{"name": "`+docsTag+`"}`)
				status, body := do(t, server, "POST", "/api/tags", `{"name": "`+docsTag+`"}`)
				expect(t, status, body, http.StatusConflict, "a tag with the name \"docs\" already exists")
			},
		},
	}
}

func TestServer_translatesTheErrorsOfTheCoreIntoTheirStatusCodes(t *testing.T) {
	t.Parallel()

	for _, test := range errorTranslations() {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			server, _ := newServer(t)
			test.steps(t, server)
		})
	}
}

func TestServer_answersInternalErrorWhenTheBaseFails(t *testing.T) {
	t.Parallel()

	server, base := newServer(t)

	if err := base.Close(); err != nil {
		t.Fatalf("cerrando la base por debajo de la API: %v", err)
	}

	status, body := do(t, server, http.MethodGet, "/api/tasks", "")
	if status != http.StatusInternalServerError {
		t.Errorf("una base cerrada respondió %d: %s, se esperaba 500", status, body)
	}

	if text := errorOf(t, body); text == "" {
		t.Errorf("el fallo interno debe cruzar con su mensaje: %q", body)
	}
}

func TestServer_reportsTheStatusOfTheProduct(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	status, body := do(t, server, http.MethodGet, "/api/status", "")
	if status != http.StatusOK {
		t.Fatalf("el estado respondió %d: %s", status, body)
	}

	var reported core.Status

	decode(t, body, &reported)

	if reported.Version != core.Version || reported.Schema != 7 || reported.Prefix != "tk" {
		t.Errorf("el estado del producto es %+v, se esperaba la versión del producto, el esquema 7 y el prefijo tk", reported)
	}
}

// TestServer_theEventsChannelKeepsOpenAndAnnouncesTheChange recorre lo que la
// interfaz espera del canal: sigue abierto y cada cambio se anuncia con el
// nombre que ella escucha, de modo que la pantalla reaccione sin sondeo.
func TestServer_theEventsChannelKeepsOpenAndAnnouncesTheChange(t *testing.T) {
	t.Parallel()

	server, base := newServer(t)
	apply := core.New(base, "tk")

	events := openEvents(t, server+"/api/events")

	// Otro que no es quien lee cambia los datos: el aviso llega al canal.
	if _, err := apply.CreateTask(t.Context(), core.NewTask{Title: "Avisada"}); err != nil {
		t.Fatalf("creando la tarea por el núcleo: %v", err)
	}

	notice := readRawEvent(t, events)

	if !strings.Contains(notice, "event: "+workspaceEventName) {
		t.Errorf("el aviso no lleva el nombre que la interfaz escucha: %q", notice)
	}

	if !strings.Contains(notice, "id: 1") {
		t.Errorf("el aviso no lleva su identidad delante: %q", notice)
	}
}

// TestServer_aChangeOfATaskAnnouncesTheSameWhateverSurfaceAskedIt recorre la
// regla del aviso: un cambio pedido por la línea de comandos y uno pedido por
// la API producen el mismo aviso —la tarea como recurso y nombrada—, y el
// aviso no dice quién lo pidió: lo sabe quien lo hizo, y la interfaz evita
// repetirse con su propio estado, sin identidad viajando al servidor.
func TestServer_aChangeOfATaskAnnouncesTheSameWhateverSurfaceAskedIt(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "tasks.db")

	served, err := store.OpenAs(t.Context(), path, store.OriginServer)
	if err != nil {
		t.Fatalf("abriendo la base del servidor: %v", err)
	}

	t.Cleanup(func() { _ = served.Close() })

	server := newTestServer(t, core.New(served, "tk"), slog.New(slog.DiscardHandler))

	commandLine, err := store.OpenAs(t.Context(), path, store.OriginCLI)
	if err != nil {
		t.Fatalf("abriendo la base como la línea de comandos: %v", err)
	}

	t.Cleanup(func() { _ = commandLine.Close() })

	outer := core.New(commandLine, "tk")

	events := openEvents(t, server+"/api/events")

	if _, err := outer.CreateTask(t.Context(), core.NewTask{Title: "Del CLI"}); err != nil {
		t.Fatalf("creando la tarea por la línea de comandos: %v", err)
	}

	cliNotice := readRawEvent(t, events)

	if change := readNotice(t, cliNotice); change.Resource != taskResource || len(change.Changes) != 1 || change.Changes[0].ID != firstRef {
		t.Errorf("el aviso del cambio del CLI dice %+v, se esperaba %s como recurso y nombrada", change, firstRef)
	}

	if status, body := do(t, server, http.MethodPost, "/api/tasks", `{"title": "De la API"}`); status != http.StatusCreated {
		t.Fatalf("crear por la API respondió %d: %s", status, body)
	}

	apiNotice := readRawEvent(t, events)

	if change := readNotice(t, apiNotice); change.Resource != taskResource || len(change.Changes) != 1 {
		t.Errorf("el aviso del cambio por la API dice %+v, se esperaba la tarea como recurso y nombrada", change)
	}

	if strings.Contains(cliNotice, `"source"`) || strings.Contains(apiNotice, `"source"`) {
		t.Errorf("el aviso dice quién pidió el cambio, y ya no lo lleva: %q / %q", cliNotice, apiNotice)
	}
}

// TestServer_aChangeAskedOutsideTheServerArrivesAsExternal recorre el aviso de
// un cambio pedido por otra superficie: la línea de comandos abre su base en su
// nombre, y el canal lo anuncia igual que cualquier otro, con la tarea como
// recurso.
func TestServer_aChangeAskedOutsideTheServerArrivesAsExternal(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "tasks.db")

	served, err := store.OpenAs(t.Context(), path, store.OriginServer)
	if err != nil {
		t.Fatalf("abriendo la base del servidor: %v", err)
	}

	t.Cleanup(func() { _ = served.Close() })

	server := newTestServer(t, core.New(served, "tk"), slog.New(slog.DiscardHandler))

	commandLine, err := store.OpenAs(t.Context(), path, store.OriginCLI)
	if err != nil {
		t.Fatalf("abriendo la base como la línea de comandos: %v", err)
	}

	t.Cleanup(func() { _ = commandLine.Close() })

	outer := core.New(commandLine, "tk")

	events := openEvents(t, server+"/api/events")

	if _, err := outer.CreateTask(t.Context(), core.NewTask{Title: "Externa"}); err != nil {
		t.Fatalf("creando la tarea por la línea de comandos: %v", err)
	}

	if change := readNotice(t, readRawEvent(t, events)); change.Resource != "tasks" {
		t.Errorf("el aviso dice %+v, se esperaba la tarea como recurso", change)
	}
}

// TestServer_aChangeAskedThroughTheAPIArrivesAsHTTP recorre el aviso de un
// cambio pedido por una sesión de la API: llega por el canal igual que el que
// pidió la línea de comandos, con la tarea como recurso.
func TestServer_aChangeAskedThroughTheAPIArrivesAsHTTP(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	events := openEvents(t, server+"/api/events")

	if status, body := do(t, server, http.MethodPost, "/api/tasks", `{"title": "De la API"}`); status != http.StatusCreated {
		t.Fatalf("crear por la API respondió %d: %s", status, body)
	}

	if change := readNotice(t, readRawEvent(t, events)); change.Resource != "tasks" {
		t.Errorf("el aviso dice %+v, se esperaba la tarea como recurso", change)
	}
}

// TestServer_aTagChangeAnnouncesItsResourceInThePluralTheInterfaceReads
// recorre el aviso de un cambio de etiqueta: el recurso viaja con el nombre en
// plural que la interfaz compara.
func TestServer_aTagChangeAnnouncesItsResourceInThePluralTheInterfaceReads(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	events := openEvents(t, server+"/api/events")

	if status, body := do(t, server, http.MethodPost, "/api/tags", `{"name": "`+docsTag+`"}`); status != http.StatusCreated {
		t.Fatalf("crear la etiqueta respondió %d: %s", status, body)
	}

	if change := readNotice(t, readRawEvent(t, events)); change.Resource != "tags" || len(change.Changes) != 0 {
		t.Errorf("el aviso dice %+v con %+v, se esperaba tags sin tareas que nombre", change.Resource, change.Changes)
	}
}

// TestServer_theNoticeOfATaskChangeSaysWhichTaskChangedAndHow recorre la regla
// del aviso de la pantalla: cada aviso que nombra una tarea lleva en changes la
// referencia visible, lo que le pasó y su título, para que el aviso diga qué
// tarea cambió y no una excusa genérica.
func TestServer_theNoticeOfATaskChangeSaysWhichTaskChangedAndHow(t *testing.T) {
	t.Parallel()

	server, base := newServer(t)
	apply := core.New(base, "tk")

	events := openEvents(t, server+"/api/events")

	if _, err := apply.CreateTask(t.Context(), core.NewTask{Title: "Avisada"}); err != nil {
		t.Fatalf("creando la tarea por el núcleo: %v", err)
	}

	expectNotice(t, readRawEvent(t, events), taskChange{ID: firstRef, Action: createdAction, Title: "Avisada"})

	if _, err := apply.UpdateTask(t.Context(), firstRef, core.Changes{Title: new(renamedTitle)}); err != nil {
		t.Fatalf("cambiando la tarea por el núcleo: %v", err)
	}

	expectNotice(t, readRawEvent(t, events), taskChange{ID: firstRef, Action: updatedAction, Title: renamedTitle})

	inProgress := "in_progress"
	if _, err := apply.UpdateTask(t.Context(), firstRef, core.Changes{Status: &inProgress}); err != nil {
		t.Fatalf("poniendo la tarea en curso por el núcleo: %v", err)
	}

	expectNotice(t, readRawEvent(t, events), taskChange{ID: firstRef, Action: updatedAction, Title: renamedTitle})

	if _, err := apply.CloseTask(t.Context(), firstRef, "done"); err != nil {
		t.Fatalf("cerrando la tarea por el núcleo: %v", err)
	}

	expectNotice(t, readRawEvent(t, events), taskChange{ID: firstRef, Action: "cerrada", Title: renamedTitle})

	// Un comentario también es un cambio de la tarea: su aviso la nombra con su
	// referencia, lo que le pasó y su título, como cualquier otro.
	if _, err := apply.AddComment(t.Context(), firstRef, "una decisión"); err != nil {
		t.Fatalf("comentando la tarea por el núcleo: %v", err)
	}

	commented := readNotice(t, readRawEvent(t, events))
	if len(commented.Changes) != 1 || commented.Changes[0] != (taskChange{ID: firstRef, Action: updatedAction, Title: renamedTitle}) {
		t.Errorf("el aviso del comentario dice %+v, se esperaba tk-1 actualizada como “Renombrada”", commented.Changes)
	}
}

// expectNotice checks that the raw notice names exactly one change, the wanted
// one.
func expectNotice(t *testing.T, raw string, want taskChange) {
	t.Helper()

	notice := readNotice(t, raw)
	if len(notice.Changes) != 1 || notice.Changes[0] != want {
		t.Errorf("el aviso dice %+v, se esperaba %+v", notice.Changes, want)
	}
}

// TestServer_severalChangesBetweenTwoLooksEachNoticeSaysItsOwnThing recorre las
// escrituras que caben en un solo sondeo: cinco cambios de la misma tarea que
// llegan juntos. Cada aviso lleva su identidad y dice lo que pasó en su propio
// cambio —alta, cambio, cierre, comentario—, no el mismo para los cinco ni
// ninguno. La tarea se crea con un título y se renombra: los cinco avisos
// siguen llevando el título de ahora.
func TestServer_severalChangesBetweenTwoLooksEachNoticeSaysItsOwnThing(t *testing.T) {
	t.Parallel()

	server, base := newServer(t)
	apply := core.New(base, "tk")
	ctx := t.Context()

	if _, err := apply.CreateTask(ctx, core.NewTask{Title: "Con historia"}); err != nil {
		t.Fatalf("creando la tarea: %v", err)
	}

	if _, err := apply.UpdateTask(ctx, firstRef, core.Changes{Title: new(renamedTitle)}); err != nil {
		t.Fatalf("cambiando la tarea: %v", err)
	}

	inProgress := "in_progress"
	if _, err := apply.UpdateTask(ctx, firstRef, core.Changes{Status: &inProgress}); err != nil {
		t.Fatalf("poniendo la tarea en curso: %v", err)
	}

	if _, err := apply.CloseTask(ctx, firstRef, "done"); err != nil {
		t.Fatalf("cerrando la tarea: %v", err)
	}

	if _, err := apply.AddComment(ctx, firstRef, "una decisión"); err != nil {
		t.Fatalf("comentando la tarea: %v", err)
	}

	// Los cinco avisos ya están escritos cuando se abre el canal: los cinco
	// caben en el primer lote que el canal manda.
	next := eventReader(t, openEvents(t, server+"/api/events?after=0"))

	expected := []struct {
		identity string
		change   taskChange
	}{
		// El título que lleva cada aviso es el que la tarea tiene ahora, no el de
		// cuando ocurrió: el producto compone el aviso al leerlo.
		{"id: 1\n", taskChange{ID: firstRef, Action: createdAction, Title: renamedTitle}},
		{"id: 2\n", taskChange{ID: firstRef, Action: updatedAction, Title: renamedTitle}},
		{"id: 3\n", taskChange{ID: firstRef, Action: updatedAction, Title: renamedTitle}},
		{"id: 4\n", taskChange{ID: firstRef, Action: "cerrada", Title: renamedTitle}},
		{"id: 5\n", taskChange{ID: firstRef, Action: updatedAction, Title: renamedTitle}},
	}

	for i, want := range expected {
		raw := next()

		if !strings.Contains(raw, want.identity) {
			t.Errorf("el aviso %d no lleva su identidad delante (%q): %q", i+1, want.identity, raw)
		}

		if notice := readNotice(t, raw); len(notice.Changes) != 1 || notice.Changes[0] != want.change {
			t.Errorf("el aviso %d dice %+v, se esperaba %+v", i+1, notice.Changes, want.change)
		}
	}
}

// TestServer_aNoticeWrittenBeforeItsActionStillSaysWhatChanged recorre la base
// de una versión anterior por la interfaz pública: sus avisos no guardan acción,
// y el canal los describe como un cambio de la tarea, con el título que lleva
// ahora, en vez de dejarlos sin descripción.
func TestServer_aNoticeWrittenBeforeItsActionStillSaysWhatChanged(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "tasks.db")

	first, err := store.OpenAs(t.Context(), path, store.OriginServer)
	if err != nil {
		t.Fatalf("abriendo la base de prueba: %v", err)
	}

	if _, err := core.New(first, "tk").CreateTask(t.Context(), core.NewTask{Title: "De antes"}); err != nil {
		t.Fatalf("creando la tarea: %v", err)
	}

	if err := first.Close(); err != nil {
		t.Fatalf("cerrando la base: %v", err)
	}

	stripNoticeActions(t, path)

	again, err := store.OpenAs(t.Context(), path, store.OriginServer)
	if err != nil {
		t.Fatalf("abriendo la base de antes: %v", err)
	}

	t.Cleanup(func() { _ = again.Close() })

	server := newTestServer(t, core.New(again, "tk"), slog.New(slog.DiscardHandler))

	raw := readRawEvent(t, openEvents(t, server+"/api/events?after=0"))

	if !strings.Contains(raw, "id: 1\n") {
		t.Errorf("el aviso de antes no lleva su identidad delante: %q", raw)
	}

	if notice := readNotice(t, raw); len(notice.Changes) != 1 ||
		notice.Changes[0] != (taskChange{ID: firstRef, Action: updatedAction, Title: "De antes"}) {
		t.Errorf("el aviso de antes dice %+v, se esperaba tk-1 actualizada como “De antes”", notice.Changes)
	}
}

// stripNoticeActions deja la base como estaba antes de que cada aviso guardara
// su acción: quita la columna y vuelve atrás la versión del esquema, que es como
// llega una base de una versión anterior.
func stripNoticeActions(t *testing.T, path string) {
	t.Helper()

	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("abriendo la base por debajo de la store: %v", err)
	}

	defer raw.Close() //nolint:errcheck // la aserción de quien llama ya decide el resultado

	for _, statement := range []string{
		`ALTER TABLE changes DROP COLUMN action`,
		`DROP TABLE adrs`,
		`DROP INDEX comments_task_id_idx`,
		`DROP INDEX dependencies_blocker_id_idx`,
		`UPDATE meta SET value = '4' WHERE key = 'schema_version'`,
	} {
		if _, err := raw.ExecContext(t.Context(), statement); err != nil {
			t.Fatalf("dejando la base sin la acción de los avisos: %v", err)
		}
	}
}

// TestServer_theEventsChannelContinuesWhereTheReaderLeftOff recorre lo que el
// contrato promete a quien vuelve: cada aviso lleva delante su identidad, quien
// continúa desde una no recibe las anteriores, y una identidad que no es un
// número se dice en vez de empezar por el principio en silencio. Quien abre el
// canal sin decir desde dónde no recibe nada del historial: eso lo recorre la
// prueba de la sesión que se abre ahora.
func TestServer_theEventsChannelContinuesWhereTheReaderLeftOff(t *testing.T) {
	t.Parallel()

	server, base := newServer(t)
	apply := core.New(base, "tk")

	for _, title := range []string{"Primera", "Segunda"} {
		if _, err := apply.CreateTask(t.Context(), core.NewTask{Title: title}); err != nil {
			t.Fatalf("creando la tarea %q: %v", title, err)
		}
	}

	second := readRawEvent(t, openEvents(t, server+"/api/events?after=1"))

	if !strings.Contains(second, "id: 2") {
		t.Errorf("quien continúa desde la primera recibió %q, se esperaba la segunda", second)
	}

	if strings.Contains(second, "id: 1") {
		t.Errorf("quien continúa desde la primera volvió a recibir la primera: %q", second)
	}

	// Continuar desde una identidad que no es un número es entrada inválida, y se
	// pide con su propio tope: si el canal no la rechazara, leer su cuerpo se
	// quedaría esperando en vez de fallar.
	client := &http.Client{Timeout: 5 * time.Second}

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server+"/api/events?after=abc", nil)
	if err != nil {
		t.Fatalf("preparando la petición del canal: %v", err)
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("pidiendo el canal desde una identidad inválida: %v", err)
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusBadRequest {
		t.Errorf("continuar desde una identidad que no es un número respondió %d", response.StatusCode)
	}
}

// TestServer_theEventsChannelBringsALongHistoryInBatches recorre la reconexión
// desde un aviso antiguo por la interfaz pública: una historia más larga que el
// lote que el canal lee de una vez llega completa, en orden y sin repetirse, con
// su identidad delante, y las etiquetas que se borraron por el camino siguen
// viajando sin romper el canal ni cortar el historial.
func TestServer_theEventsChannelBringsALongHistoryInBatches(t *testing.T) {
	t.Parallel()

	server, base := newServer(t)
	apply := core.New(base, "tk")

	ctx := t.Context()

	history := 2*core.EventBatch + 1
	tagCreated, tagDeleted := writtenLongHistory(t, apply, history)

	next := eventReader(t, openEvents(t, server+"/api/events?after=0"))

	for id := range history + 2 {
		checkHistoryNotice(t, id+1, next(), tagCreated, tagDeleted)
	}

	// Y el canal sigue vivo después del historial: lo que pasa ahora llega.
	if _, err := apply.CreateTask(ctx, core.NewTask{Title: "La siguiente"}); err != nil {
		t.Fatalf("creando la tarea de después: %v", err)
	}

	live := readNotice(t, next())
	if len(live.Changes) != 1 || live.Changes[0].Title != "La siguiente" {
		t.Errorf("el canal después del historial dice %+v, se esperaba la tarea de después", live)
	}
}

// writtenLongHistory deja en la base una historia más larga que un lote del canal:
// history tareas y, a la mitad, una etiqueta que se crea y se borra en seguida. Los
// dos avisos de la etiqueta caen entre lotes, así que el lote siguiente tiene que
// seguir leyendo después de ellos. Devuelve la identidad de los dos.
func writtenLongHistory(t *testing.T, apply *core.Core, history int) (int, int) {
	t.Helper()

	ctx := t.Context()

	for i := range history {
		if _, err := apply.CreateTask(ctx, core.NewTask{Title: fmt.Sprintf("Tarea %d", i+1)}); err != nil {
			t.Fatalf("creando la tarea %d: %v", i+1, err)
		}

		if i != history/2 {
			continue
		}

		tag, err := apply.CreateTag(ctx, "borrada", "", "")
		if err != nil {
			t.Fatalf("creando la etiqueta: %v", err)
		}

		if _, _, err := apply.DeleteTag(ctx, tag.Name); err != nil {
			t.Fatalf("borrando la etiqueta: %v", err)
		}
	}

	tagCreated := history/2 + 2

	return tagCreated, tagCreated + 1
}

// checkHistoryNotice recorre un aviso del historial como lo lee la interfaz: su
// identidad delante y, cuando habla de una tarea, la tarea nombrada con el título
// que lleva ahora. Los avisos de la etiqueta que se borró viajan sin cambios.
func checkHistoryNotice(t *testing.T, id int, block string, tagCreated, tagDeleted int) {
	t.Helper()

	if want := "id: " + strconv.Itoa(id) + "\n"; !strings.HasPrefix(block, want) {
		t.Fatalf("el aviso %d llegó con %q delante, se esperaba %q", id, firstLine(block), want)
	}

	notice := readNotice(t, block)

	if id == tagCreated || id == tagDeleted {
		if notice.Resource != "tags" || len(notice.Changes) != 0 {
			t.Errorf("el aviso %d de la etiqueta dice %+v, se esperaba el recurso tags sin cambios", id, notice)
		}

		return
	}

	// Las dos escrituras de la etiqueta van en medio del historial: después de ellas
	// cada aviso habla de la tarea dos números más atrás.
	task := id
	if id > tagDeleted {
		task -= 2
	}

	if notice.Resource != taskResource || len(notice.Changes) != 1 {
		t.Fatalf("el aviso %d dice %+v, se esperaba una tarea con su cambio", id, notice)
	}

	want := taskChange{
		ID:     fmt.Sprintf("tk-%d", task),
		Action: createdAction,
		Title:  fmt.Sprintf("Tarea %d", task),
	}

	if notice.Changes[0] != want {
		t.Fatalf("el aviso %d dice %+v, se esperaba %+v", id, notice.Changes[0], want)
	}
}

// firstLine deja delante lo que el canal anuncia primero de un aviso: su
// identidad.
func firstLine(block string) string {
	line, _, _ := strings.Cut(block, "\n")

	return line
}

// TestServer_aSessionOpenedNowDoesNotReceiveTheChangesOfBefore recorre el
// defecto: el aviso cuenta lo que pasa mientras miras, no lo que pasó antes.
// Quien abre el canal recibe el cambio que ocurre después de abrirlo, y no los
// que ya estaban en la base cuando abrió.
func TestServer_aSessionOpenedNowDoesNotReceiveTheChangesOfBefore(t *testing.T) {
	t.Parallel()

	server, base := newServer(t)
	apply := core.New(base, "tk")

	// Dos cambios antes de que nadie mire.
	for _, title := range []string{"De antes", "De antes también"} {
		if _, err := apply.CreateTask(t.Context(), core.NewTask{Title: title}); err != nil {
			t.Fatalf("creando la tarea %q: %v", title, err)
		}
	}

	events := openEvents(t, server+"/api/events")

	if _, err := apply.CreateTask(t.Context(), core.NewTask{Title: "De después"}); err != nil {
		t.Fatalf("creando la tarea de después: %v", err)
	}

	notice := readNotice(t, readRawEvent(t, events))

	if len(notice.Changes) != 1 || notice.Changes[0] != (taskChange{ID: "tk-3", Action: createdAction, Title: "De después"}) {
		t.Errorf("el primer aviso dice %+v, se esperaba tk-3 creada como “De después” y nada de antes", notice.Changes)
	}
}

// readRawEvent devuelve un aviso tal como viaja por el canal, con sus líneas,
// para poder comprobar su forma y no solo lo que lleva dentro.
func readRawEvent(t *testing.T, events io.Reader) string {
	t.Helper()

	return eventReader(t, events)()
}

// eventReader lee el canal aviso a aviso y devuelve la función que trae el
// siguiente: un aviso se ve como un bloque terminado en una línea vacía. Se
// necesita un solo lector por canal —varios avisos seguidos con un lector nuevo
// por aviso perderían lo que el primero ya trajo.
func eventReader(t *testing.T, events io.Reader) func() string {
	t.Helper()

	scanner := bufio.NewScanner(events)

	var block strings.Builder

	return func() string {
		for scanner.Scan() {
			line := scanner.Text()

			block.WriteString(line + "\n")

			if line == "" {
				notice := block.String()
				block.Reset()

				return notice
			}
		}

		if err := scanner.Err(); err != nil {
			t.Fatalf("leyendo el canal de eventos: %v", err)
		}

		t.Fatal("el canal se cerró sin avisar de ningún cambio")

		return ""
	}
}

func expect(t *testing.T, status int, body string, expected int, message string) {
	t.Helper()

	if status != expected {
		t.Errorf("la petición respondió %d, se esperaba %d: %s", status, expected, body)
	}

	if text := errorOf(t, body); text != message {
		t.Errorf("el cuerpo del error dice %q, se esperaba %q", text, message)
	}
}

// TestServer_refusesARequestAddressedToAnotherName comprueba que el servidor no
// contesta a quien llega por un nombre que no es el suyo. Es la petición de una
// página que un DNS rebinding apunta a esta máquina: sin esta comprobación leería
// el tablero entero, porque la API no pide credenciales.
func TestServer_refusesARequestAddressedToAnotherName(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	status, body := doWith(t, server, http.MethodGet, "/api/tasks", "", func(request *http.Request) {
		request.Host = "evil.example:8890"
	})
	if status != http.StatusForbidden {
		t.Fatalf("leer el tablero por otro nombre respondió %d: %s", status, body)
	}
}

// TestServer_refusesAWriteThatComesFromAnotherPage comprueba que una página que
// no es la interfaz no escribe en el tablero. El cuerpo va como texto llano a
// propósito: es lo que un navegador manda a otro origen sin preguntar antes si se
// lo permiten, así que la comprobación no puede ser el tipo del contenido.
func TestServer_refusesAWriteThatComesFromAnotherPage(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	status, body := doWith(t, server, http.MethodPost, "/api/tasks", `{"title":"Desde otra página"}`, func(request *http.Request) {
		request.Header.Set("Origin", "https://evil.example")
		request.Header.Set("Content-Type", "text/plain")
	})
	if status != http.StatusForbidden {
		t.Fatalf("escribir desde otra página respondió %d: %s", status, body)
	}

	status, body = do(t, server, http.MethodGet, "/api/tasks", "")
	if status != http.StatusOK {
		t.Fatalf("listar las tareas respondió %d: %s", status, body)
	}

	var tasks []core.Task
	decode(t, body, &tasks)

	if len(tasks) != 0 {
		t.Fatalf("la escritura rechazada dejó %d tareas en el tablero", len(tasks))
	}
}

// TestServer_acceptsTheWriteOfItsOwnPage comprueba que la interfaz sigue
// escribiendo: el guardián no puede cerrar la puerta de casa.
func TestServer_acceptsTheWriteOfItsOwnPage(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	status, body := doWith(t, server, http.MethodPost, "/api/tasks", `{"title":"Desde la interfaz"}`, func(request *http.Request) {
		request.Header.Set("Origin", server)
	})
	if status != http.StatusCreated {
		t.Fatalf("escribir desde la interfaz respondió %d: %s", status, body)
	}
}

// TestServer_refusesABodyLargerThanTheLimit comprueba que una petición sola no
// puede pedir la memoria del proceso, y que el cuerpo rechazado no deja nada
// escrito.
func TestServer_refusesABodyLargerThanTheLimit(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	huge := `{"title":"` + strings.Repeat("a", 5<<20) + `"}`

	status, body := do(t, server, http.MethodPost, "/api/tasks", huge)
	if status != http.StatusBadRequest || !strings.Contains(body, "larger") {
		t.Fatalf("un cuerpo de cinco megas respondió %d: %s", status, body)
	}

	status, body = do(t, server, http.MethodGet, "/api/tasks", "")
	if status != http.StatusOK {
		t.Fatalf("listar las tareas respondió %d: %s", status, body)
	}

	var tasks []core.Task
	decode(t, body, &tasks)

	if len(tasks) != 0 {
		t.Fatalf("el cuerpo rechazado dejó %d tareas en el tablero", len(tasks))
	}
}

// doWith hace la petición como do y deja prepararla antes: es lo que permite
// mentir sobre el nombre al que se dirige o sobre la página desde la que viene.
func doWith(t *testing.T, server, method, path, body string, prepare func(*http.Request)) (int, string) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), method, server+path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("preparando %s %s: %v", method, path, err)
	}

	if prepare != nil {
		prepare(request)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("pidiendo %s %s: %v", method, path, err)
	}

	defer func() { _ = response.Body.Close() }()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("leyendo la respuesta de %s %s: %v", method, path, err)
	}

	return response.StatusCode, string(raw)
}

func do(t *testing.T, server, method, path, body string) (int, string) {
	t.Helper()

	return doWith(t, server, method, path, body, nil)
}

func openEvents(t *testing.T, url string) io.Reader {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("abriendo el canal de eventos: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("abriendo el canal de eventos: %v", err)
	}

	t.Cleanup(func() { _ = response.Body.Close() })

	if response.StatusCode != http.StatusOK {
		t.Fatalf("el canal de eventos respondió %d", response.StatusCode)
	}

	if !strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
		t.Errorf("el canal anuncia %q, se esperaba text/event-stream", response.Header.Get("Content-Type"))
	}

	return response.Body
}

// workspaceEventName repite aquí el nombre con el que el servidor anuncia cada
// cambio: es el mensaje con nombre que la interfaz escucha por el canal.
const workspaceEventName = "workspace.changed"

// notice is the body of a named notice, in the shape the interface reads.
type notice struct {
	Resource string       `json:"resource"`
	Changes  []taskChange `json:"changes"`
}

// taskChange is one task the change touched, as the toast of the interface
// prints it: su referencia visible, lo que le pasó y su título.
type taskChange struct {
	ID     string `json:"id"`
	Action string `json:"action"`
	Title  string `json:"title"`
}

// readNotice lee la carga del aviso que viaja por el canal, tal como la lee la
// interfaz: el recurso que cambió y, si toca una tarea, la tarea nombrada.
func readNotice(t *testing.T, raw string) notice {
	t.Helper()

	for line := range strings.SplitSeq(raw, "\n") {
		if data, ok := strings.CutPrefix(line, "data: "); ok {
			var change notice

			decode(t, data, &change)

			return change
		}
	}

	t.Fatalf("el aviso no lleva carga que la interfaz pueda leer: %q", raw)

	return notice{}
}

func errorOf(t *testing.T, body string) string {
	t.Helper()

	var parsed struct {
		Error string `json:"error"`
	}

	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return ""
	}

	return parsed.Error
}

func decode(t *testing.T, raw string, target any) {
	t.Helper()

	if err := json.Unmarshal([]byte(raw), target); err != nil {
		t.Fatalf("el JSON devuelto no se entiende (%v): %q", err, raw)
	}
}

// TestServer_listsEveryCommentForTheSearchIndex recorre el criterio de tk-124:
// la API expone todos los comentarios del proyecto con la referencia de la
// tarea que los lleva, y una base sin comentarios responde una lista vacía.
func TestServer_listsEveryCommentForTheSearchIndex(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	status, body := do(t, server, http.MethodGet, "/api/comments", "")
	if status != http.StatusOK {
		t.Fatalf("leer los comentarios de una base vacía respondió %d: %s", status, body)
	}

	var none []core.TaskComment
	decode(t, body, &none)

	if len(none) != 0 {
		t.Errorf("una base sin comentarios respondió %s, se esperaba una lista vacía", body)
	}

	createsATaskThroughTheAPI(t, server)

	if status, body := do(t, server, http.MethodPost, "/api/tasks", `{"title": "La otra"}`); status != http.StatusCreated {
		t.Fatalf("crear la segunda tarea respondió %d: %s", status, body)
	}

	if status, body := do(t, server, http.MethodPost, "/api/tasks/tk-1/comments", `{"body": "de la primera"}`); status != http.StatusOK {
		t.Fatalf("añadir el comentario a tk-1 respondió %d: %s", status, body)
	}

	if status, body := do(t, server, http.MethodPost, "/api/tasks/tk-2/comments", `{"body": "de la segunda"}`); status != http.StatusOK {
		t.Fatalf("añadir el comentario a tk-2 respondió %d: %s", status, body)
	}

	status, body = do(t, server, http.MethodGet, "/api/comments", "")
	if status != http.StatusOK {
		t.Fatalf("leer todos los comentarios respondió %d: %s", status, body)
	}

	var listed []core.TaskComment
	decode(t, body, &listed)

	if len(listed) != 2 {
		t.Fatalf("el índice trae %d comentarios, se esperaban 2: %s", len(listed), body)
	}

	if listed[0].Ref != firstRef || listed[0].Body != "de la primera" {
		t.Errorf("el primer comentario es %+v, se esperaba el de %s", listed[0], firstRef)
	}

	if listed[1].Ref != "tk-2" || listed[1].Body != "de la segunda" {
		t.Errorf("el segundo comentario es %+v, se esperaba el de tk-2", listed[1])
	}
}

// TestServer_createsATaskWithTheTagsItIsGiven recorre el quinto caso del
// criterio de tk-170: la entrada de la creación lleva las etiquetas y la
// lectura las devuelve.
func TestServer_createsATaskWithTheTagsItIsGiven(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	if status, body := do(t, server, http.MethodPost, "/api/tags", `{"name": "`+docsTag+`"}`); status != http.StatusCreated {
		t.Fatalf("crear la etiqueta respondió %d: %s", status, body)
	}

	created := createdTask(t, server, `{"title": "Con etiquetas", "tags": ["`+docsTag+`"]}`)

	if len(created.Tags) != 1 || created.Tags[0] != docsTag {
		t.Fatalf("la tarea creada lleva %v, se esperaba [docs]", created.Tags)
	}

	if read := readTask(t, server, created.Ref); len(read.Tags) != 1 || read.Tags[0] != docsTag {
		t.Errorf("la tarea leída lleva %v, se esperaba [docs]", read.Tags)
	}
}

// TestServer_refusesAnEntryWithAnUnknownTag cierra el quinto caso del criterio
// por el lado del rechazo: la etiqueta que falta en el catálogo se nombra y no
// deja la tarea escrita.
func TestServer_refusesAnEntryWithAnUnknownTag(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	status, body := do(t, server, http.MethodPost, "/api/tasks", `{"title": "Con una que no existe", "tags": ["inventada"]}`)
	if status != http.StatusBadRequest {
		t.Fatalf("crear con una etiqueta desconocida respondió %d, se esperaba 400: %s", status, body)
	}

	if !strings.Contains(body, "unknown tag") {
		t.Errorf("el error (%q) no dice que la etiqueta es desconocida", body)
	}

	status, body = do(t, server, http.MethodGet, "/api/tasks", "")
	if status != http.StatusOK {
		t.Fatalf("listar respondió %d: %s", status, body)
	}

	var tasks []core.Task

	decode(t, body, &tasks)

	if len(tasks) != 0 {
		t.Errorf("el rechazo dejó tareas escritas: %+v", tasks)
	}
}
