package aif_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"tasks-ai/internal/aif"
	"tasks-ai/internal/core"
	"tasks-ai/internal/store"
	"testing"

	interaction "github.com/fred66sp/agent-interaction-forms"
)

// TestHandle_executesAWriteAndAnswersTheOutcome comprueba la clase executed: la
// escritura llega al núcleo una vez, deja la tarea y el outcome es la tarea.
func TestHandle_executesAWriteAndAnswersTheOutcome(t *testing.T) {
	t.Parallel()

	apply := newCore(t)
	response := aif.New(apply).Handle([]byte(
		`{"aif":"0.2","operation":"tasks_create","input":{"title":"Preparar el análisis","priority":1}}`,
	))

	if response.Result != interaction.ResultExecuted {
		t.Fatalf("result = %q, se esperaba %q: %+v", response.Result, interaction.ResultExecuted, response)
	}

	var task core.Task
	if err := json.Unmarshal(response.Outcome, &task); err != nil {
		t.Fatalf("el outcome no es una tarea: %v (%s)", err, response.Outcome)
	}

	if task.Ref != "tk-1" || task.Title != "Preparar el análisis" || task.Priority != 1 {
		t.Errorf("el outcome es %+v, se esperaba la tarea creada con prioridad 1", task)
	}

	tasks, err := apply.Tasks(t.Context())
	if err != nil {
		t.Fatalf("releyendo las tareas: %v", err)
	}

	if len(tasks) != 1 {
		t.Errorf("la escritura dejó %d tareas, se esperaba 1", len(tasks))
	}
}

// TestHandle_asksForAMissingFieldWithoutWriting comprueba la clase input_needed:
// la biblioteca nombra el miembro que falta antes de que el núcleo escriba nada.
func TestHandle_asksForAMissingFieldWithoutWriting(t *testing.T) {
	t.Parallel()

	apply := newCore(t)
	response := aif.New(apply).Handle([]byte(
		`{"aif":"0.2","operation":"tasks_create","input":{"description":"sin título"}}`,
	))

	if response.Result != interaction.ResultInputNeeded {
		t.Fatalf("result = %q, se esperaba %q: %+v", response.Result, interaction.ResultInputNeeded, response)
	}

	if !slices.Equal(response.Missing, []string{"title"}) {
		t.Errorf("missing = %v, se esperaba solo title", response.Missing)
	}

	if tasks, err := apply.Tasks(t.Context()); err != nil || len(tasks) != 0 {
		t.Errorf("una petición incompleta dejó %d tareas (err %v), se esperaba ninguna", len(tasks), err)
	}
}

// TestHandle_repairsAReferenceThatDoesNotResolve comprueba la clase invalid con
// campo: una referencia inexistente que aporta el consumidor es reparable y no
// toca el dominio.
func TestHandle_repairsAReferenceThatDoesNotResolve(t *testing.T) {
	t.Parallel()

	apply := newCore(t)
	createTask(t, apply, core.NewTask{Title: "La única"})

	response := aif.New(apply).Handle([]byte(
		`{"aif":"0.2","operation":"tasks_show","input":{"ref":"tk-404"}}`,
	))

	if response.Result != interaction.ResultInvalid {
		t.Fatalf("result = %q, se esperaba %q: %+v", response.Result, interaction.ResultInvalid, response)
	}

	problem, ok := response.Invalid["ref"]
	if !ok {
		t.Fatalf("invalid no nombra el campo ref: %+v", response.Invalid)
	}

	if !strings.Contains(problem.Reason, "tk-404") {
		t.Errorf("la razón %q no dice la referencia rechazada", problem.Reason)
	}

	if tasks, err := apply.Tasks(t.Context()); err != nil || len(tasks) != 1 {
		t.Errorf("una lectura rechazada dejó %d tareas (err %v), se esperaba 1", len(tasks), err)
	}
}

// TestHandle_repairsATextThatBreaksTheRules comprueba que una regla de texto
// rechazada por el núcleo vuelve como invalid con su campo y sin efecto.
func TestHandle_repairsATextThatBreaksTheRules(t *testing.T) {
	t.Parallel()

	apply := newCore(t)
	task := createTask(t, apply, core.NewTask{Title: "Con comentario"})

	response := aif.New(apply).Handle([]byte(
		`{"aif":"0.2","operation":"tasks_comment","input":{"ref":"` + task.Ref + `","body":"sin commitear"}}`,
	))

	if response.Result != interaction.ResultInvalid {
		t.Fatalf("result = %q, se esperaba %q: %+v", response.Result, interaction.ResultInvalid, response)
	}

	if _, ok := response.Invalid["body"]; !ok {
		t.Fatalf("invalid no nombra el campo body: %+v", response.Invalid)
	}

	if comments, err := apply.Comments(t.Context(), task.Ref); err != nil || len(comments) != 0 {
		t.Errorf("un comentario rechazado dejó %d (err %v), se esperaba ninguno", len(comments), err)
	}
}

// TestHandle_deniesARuleTheProductRefuses comprueba la clase denied: un rechazo
// de regla del producto llega como denial y no cambia el estado.
func TestHandle_deniesARuleTheProductRefuses(t *testing.T) {
	t.Parallel()

	apply := newCore(t)
	task := createTask(t, apply, core.NewTask{Title: "Sin cerrar"})

	response := aif.New(apply).Handle([]byte(
		`{"aif":"0.2","operation":"tasks_reopen","input":{"ref":"` + task.Ref + `"}}`,
	))

	if response.Result != interaction.ResultDenied {
		t.Fatalf("result = %q, se esperaba %q: %+v", response.Result, interaction.ResultDenied, response)
	}

	if response.Reason == "" {
		t.Error("la denial no trae su razón")
	}

	read, err := apply.Task(t.Context(), task.Ref)
	if err != nil {
		t.Fatalf("releyendo la tarea: %v", err)
	}

	if read.Status != task.Status {
		t.Errorf("la tarea quedó en %q, se esperaba %q", read.Status, task.Status)
	}
}

// TestHandle_failsOnlyOnARealFailure comprueba la clase failed: un fallo de la
// base no se disfraza de entrada reparable ni de rechazo de regla.
func TestHandle_failsOnlyOnARealFailure(t *testing.T) {
	t.Parallel()

	base, err := store.Open(t.Context(), filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatalf("abriendo la base: %v", err)
	}

	apply := core.New(base, "tk")

	if err := base.Close(); err != nil {
		t.Fatalf("cerrando la base: %v", err)
	}

	response := aif.New(apply).Handle([]byte(`{"aif":"0.2","operation":"tasks_board"}`))

	if response.Result != interaction.ResultFailed {
		t.Fatalf("result = %q, se esperaba %q: %+v", response.Result, interaction.ResultFailed, response)
	}

	if response.Message == "" {
		t.Error("el fallo no trae su mensaje")
	}

	if response.Outcome != nil {
		t.Errorf("un fallo no debe traer outcome: %s", response.Outcome)
	}
}

// TestHandle_repairsAnEntryWhoseRuleNamesNoField comprueba que una entrada
// reparable sin campo en el error se ancla al miembro que decide el rechazo.
func TestHandle_repairsAnEntryWhoseRuleNamesNoField(t *testing.T) {
	t.Parallel()

	apply := newCore(t)
	task := createTask(t, apply, core.NewTask{Title: "La única"})

	tests := []struct {
		name    string
		request string
		field   string
	}{
		{
			name:    "an update that asks for nothing",
			request: fmt.Sprintf(`{"aif":"0.2","operation":"tasks_update","input":{"ref":%q}}`, task.Ref),
			field:   "ref",
		},
		{
			name:    "a task that blocks itself",
			request: fmt.Sprintf(`{"aif":"0.2","operation":"tasks_block","input":{"ref":%q,"blocker":%q}}`, task.Ref, task.Ref),
			field:   "blocker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			response := aif.New(apply).Handle([]byte(tt.request))

			if response.Result != interaction.ResultInvalid {
				t.Fatalf("result = %q, se esperaba %q: %+v", response.Result, interaction.ResultInvalid, response)
			}

			if _, ok := response.Invalid[tt.field]; !ok {
				t.Errorf("invalid = %+v, se esperaba el campo %q", response.Invalid, tt.field)
			}
		})
	}
}

// TestHandle_reportsAnUnknownOperation comprueba que la operación desconocida
// llega con el catálogo a la vista y sin escribir nada.
func TestHandle_reportsAnUnknownOperation(t *testing.T) {
	t.Parallel()

	apply := newCore(t)
	response := aif.New(apply).Handle([]byte(`{"aif":"0.2","operation":"tasks_nope"}`))

	if response.Result != interaction.ResultUnknownOperation {
		t.Fatalf("result = %q, se esperaba %q", response.Result, interaction.ResultUnknownOperation)
	}

	if !slices.Contains(response.KnownOperations, "tasks_board") || len(response.KnownOperations) != 11 {
		t.Errorf("known_operations = %v, se esperaba el vocabulario completo", response.KnownOperations)
	}

	if tasks, err := apply.Tasks(t.Context()); err != nil || len(tasks) != 0 {
		t.Errorf("una operación desconocida dejó %d tareas (err %v), se esperaba ninguna", len(tasks), err)
	}
}

// TestHandle_rejectsAMalformedRequestOrAnUnsupportedVersion comprueba las dos
// clases estructurales que no llegan al dominio.
func TestHandle_rejectsAMalformedRequestOrAnUnsupportedVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request string
		want    string
	}{
		{name: "an empty request", request: ``, want: interaction.ResultMalformedRequest},
		{name: "a broken object", request: `{"aif":"0.2"`, want: interaction.ResultMalformedRequest},
		{name: "another version", request: `{"aif":"0.1","operation":"tasks_board"}`, want: interaction.ResultUnsupportedVersion},
	}

	apply := newCore(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			response := aif.New(apply).Handle([]byte(tt.request))

			if response.Result != tt.want {
				t.Errorf("result = %q, se esperaba %q", response.Result, tt.want)
			}
		})
	}
}

// TestHandle_discoversTheOperations comprueba el descubrimiento sin operation:
// el vocabulario completo, cada operación con su descripción de una línea.
func TestHandle_discoversTheOperations(t *testing.T) {
	t.Parallel()

	response := aif.New(newCore(t)).Handle([]byte(`{"aif":"0.2"}`))

	if response.Discovery == nil {
		t.Fatalf("la petición sin operation no descubrió el catálogo: %+v", response)
	}

	if response.Discovery.AIF != interaction.Version {
		t.Errorf("aif = %q, se esperaba %q", response.Discovery.AIF, interaction.Version)
	}

	want := []string{
		"tasks_board", "tasks_show", "tasks_create", "tasks_update", "tasks_close",
		"tasks_reopen", "tasks_block", "tasks_unblock", "tasks_comment",
		"tasks_comment_list", "tasks_comment_edit",
	}

	got := make([]string, 0, len(response.Discovery.Operations))
	for _, operation := range response.Discovery.Operations {
		got = append(got, operation.ID)

		if operation.Description == "" || strings.Contains(operation.Description, "\n") {
			t.Errorf("la operación %q no trae una descripción de una línea: %q", operation.ID, operation.Description)
		}
	}

	if !slices.Equal(got, want) {
		t.Errorf("el descubrimiento anuncia %v, se esperaba %v", got, want)
	}
}

// newCore opens a project's core over a temporal base.
func newCore(t *testing.T) *core.Core {
	t.Helper()

	base, err := store.Open(t.Context(), filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatalf("abriendo la base: %v", err)
	}

	t.Cleanup(func() { _ = base.Close() })

	return core.New(base, "tk")
}

// createTask stores a task through the core the adapter uses.
func createTask(t *testing.T, apply *core.Core, task core.NewTask) core.Task {
	t.Helper()

	created, err := apply.CreateTask(t.Context(), task)
	if err != nil {
		t.Fatalf("creando la tarea de la prueba: %v", err)
	}

	return created
}
