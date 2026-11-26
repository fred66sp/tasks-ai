package aif_test

import (
	"encoding/json"
	"slices"
	"strings"
	"tasks-ai/internal/aif"
	"tasks-ai/internal/core"
	"testing"

	interaction "github.com/fred66sp/agent-interaction-forms"
)

// TestHandle_projectsTheBoard comprueba la proyección de docs/contracts.md §7:
// contadores, en curso, listas para empezar y árbol por el padre.
func TestHandle_projectsTheBoard(t *testing.T) {
	t.Parallel()

	apply := newCore(t)
	parent := createTask(t, apply, core.NewTask{Title: "Épica"})
	started := createTask(t, apply, core.NewTask{Title: "En marcha", Parent: parent.Ref})
	moveTo(t, apply, started.Ref, "in_progress")
	pending := createTask(t, apply, core.NewTask{Title: "Pendiente"})
	blocked := createTask(t, apply, core.NewTask{Title: "Bloqueada", Parent: parent.Ref})
	blocker := createTask(t, apply, core.NewTask{Title: "Bloqueante"})
	done := createTask(t, apply, core.NewTask{Title: "Hecha"})

	blockTask(t, apply, blocked.Ref, blocker.Ref)
	moveTo(t, apply, done.Ref, "in_progress")
	closeTask(t, apply, done.Ref)

	board := boardOf(t, apply)

	if want := (aif.BoardCounts{Open: 4, InProgress: 1, Done: 1, Total: 6}); board.Counts != want {
		t.Errorf("counts = %+v, se esperaba %+v", board.Counts, want)
	}

	if len(board.InProgress) != 1 || board.InProgress[0].Ref != started.Ref {
		t.Errorf("en curso = %+v, se esperaba solo %s", board.InProgress, started.Ref)
	}

	if got := refsOf(board.Ready); len(got) != 3 || !slices.Contains(got, parent.Ref) || !slices.Contains(got, pending.Ref) {
		t.Errorf("listas para empezar = %v, se esperaban %s, %s y %s, no la bloqueada", got, parent.Ref, pending.Ref, blocker.Ref)
	}

	if len(board.Tree) != 4 {
		t.Fatalf("el árbol tiene %d raíces, se esperaban 4: %+v", len(board.Tree), board.Tree)
	}

	if board.Tree[0].Ref != parent.Ref || len(board.Tree[0].Children) != 2 {
		t.Errorf("el árbol no cuelga las dos hijas de %s: %+v", parent.Ref, board.Tree[0])
	}

	if board.InProgress[0].Priority != core.DefaultPriority || board.InProgress[0].ParentID == nil {
		t.Errorf("la tarea en curso no lleva prioridad y padre: %+v", board.InProgress[0])
	}
}

// TestHandle_keepsTheDetailOutOfTheBoard comprueba que de cada tarea viajan
// solo los campos de §7: ni descripción ni notas, y las etiquetas sí.
func TestHandle_keepsTheDetailOutOfTheBoard(t *testing.T) {
	t.Parallel()

	apply := newCore(t)

	if _, err := apply.CreateTag(t.Context(), "tipo/mejora", "", ""); err != nil {
		t.Fatalf("creando la etiqueta: %v", err)
	}

	createTask(t, apply, core.NewTask{
		Title:       "Épica",
		Description: "descripción que no debe viajar",
		Notes:       "notas que tampoco",
		Tags:        []string{"tipo/mejora"},
	})

	raw := outcomeOf(t, apply)

	for _, forbidden := range []string{"descripción que no debe viajar", "notas que tampoco", "description", "notes"} {
		if strings.Contains(raw, forbidden) {
			t.Errorf("el tablero deja viajar %q: %s", forbidden, raw)
		}
	}

	if !strings.Contains(raw, "tipo/mejora") {
		t.Errorf("el tablero no lleva las etiquetas de la tarea: %s", raw)
	}
}

// TestHandle_clipsTheBoardListsAndTree comprueba los recortes de §7: las listas
// a 8 y el árbol a 40, con lo que queda fuera contado.
func TestHandle_clipsTheBoardListsAndTree(t *testing.T) {
	t.Parallel()

	apply := newCore(t)

	for range 10 {
		createTask(t, apply, core.NewTask{Title: "Pendiente"})
	}

	board := boardOf(t, apply)

	if len(board.Ready) != 8 || board.ReadyHidden != 2 {
		t.Errorf("listas para empezar = %d con %d fuera, se esperaban 8 y 2", len(board.Ready), board.ReadyHidden)
	}

	for range 31 {
		createTask(t, apply, core.NewTask{Title: "Del árbol"})
	}

	board = boardOf(t, apply)

	if nodes := countNodes(board.Tree); nodes != 40 {
		t.Errorf("el árbol muestra %d tareas, se esperaban 40", nodes)
	}

	if board.TreeHidden != 1 {
		t.Errorf("el árbol cuenta %d fuera, se esperaba 1", board.TreeHidden)
	}
}

// boardOf asks for the board the way a consumer does.
func boardOf(t *testing.T, apply *core.Core) aif.Board {
	t.Helper()

	response := aif.New(apply).Handle([]byte(`{"aif":"0.2","operation":"tasks_board"}`))
	if response.Result != interaction.ResultExecuted {
		t.Fatalf("tasks_board = %q, se esperaba executed: %+v", response.Result, response)
	}

	var board aif.Board
	if err := json.Unmarshal(response.Outcome, &board); err != nil {
		t.Fatalf("el tablero no se entiende: %v (%s)", err, response.Outcome)
	}

	return board
}

// outcomeOf returns the raw board the profile answers.
func outcomeOf(t *testing.T, apply *core.Core) string {
	t.Helper()

	response := aif.New(apply).Handle([]byte(`{"aif":"0.2","operation":"tasks_board"}`))
	if response.Result != interaction.ResultExecuted {
		t.Fatalf("tasks_board = %q, se esperaba executed: %+v", response.Result, response)
	}

	return string(response.Outcome)
}

// moveTo puts a task in the state the proof needs.
func moveTo(t *testing.T, apply *core.Core, ref, status string) {
	t.Helper()

	if _, err := apply.UpdateTask(t.Context(), ref, core.Changes{Status: &status}); err != nil {
		t.Fatalf("poniendo %s en %s: %v", ref, status, err)
	}
}

// blockTask leaves the task blocked by the other one.
func blockTask(t *testing.T, apply *core.Core, ref, blocker string) {
	t.Helper()

	if _, err := apply.BlockTask(t.Context(), ref, blocker); err != nil {
		t.Fatalf("bloqueando %s por %s: %v", ref, blocker, err)
	}
}

// closeTask closes the task as done.
func closeTask(t *testing.T, apply *core.Core, ref string) {
	t.Helper()

	if _, err := apply.CloseTask(t.Context(), ref, core.CloseReasonDone); err != nil {
		t.Fatalf("cerrando %s: %v", ref, err)
	}
}

// refsOf returns the references of the tasks of one list.
func refsOf(tasks []aif.BoardTask) []string {
	refs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		refs = append(refs, task.Ref)
	}

	return refs
}

// countNodes counts the nodes of the tree, children included.
func countNodes(nodes []aif.BoardNode) int {
	count := 0

	for _, node := range nodes {
		count += 1 + countNodes(node.Children)
	}

	return count
}
