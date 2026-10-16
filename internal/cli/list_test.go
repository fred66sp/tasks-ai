package cli_test

import (
	"tasks-ai/internal/core"
	"testing"
)

// statusOpen is how a task waiting for its turn crosses the command line.
const statusOpen = "open"

// El criterio de la tarea: desde la línea de comandos se ven todas las tareas
// con su referencia, su estado y su título, y lo que está bloqueado se
// distingue.
func TestRun_listsEveryTask(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	sow(t)

	code, out, errOut := run(t, "list")
	if code != 0 {
		t.Fatalf("list salió con %d: %s", code, errOut)
	}

	var tasks []core.Task

	decode(t, out, &tasks)

	if len(tasks) != 3 {
		t.Fatalf("la lista trae %d tareas, se esperaban 3: %s", len(tasks), out)
	}

	compare(t, tasks)
}

// sow leaves a project with three tasks: one waiting for another, one closed
// and one free, so the list has to show the three states and the block.
func sow(t *testing.T) {
	t.Helper()

	if code, _, errOut := run(t, "init"); code != 0 {
		t.Fatalf("init salió con %d: %s", code, errOut)
	}

	for _, title := range []string{"Preparar el lanzamiento", "Escribir el anuncio", "Comprobar las copias"} {
		if code, _, errOut := run(t, "create", "--title", title); code != 0 {
			t.Fatalf("create salió con %d: %s", code, errOut)
		}
	}

	if code, _, errOut := run(t, "close", "tk-2", "--reason", "done"); code != 0 {
		t.Fatalf("close salió con %d: %s", code, errOut)
	}

	if code, _, errOut := run(t, "block", "tk-1", "tk-3"); code != 0 {
		t.Fatalf("block salió con %d: %s", code, errOut)
	}
}

// compare checks that the list carries the three tasks with their status, their
// title and what is blocked. The order is not part of the criterion: the tasks
// are compared by their reference.
func compare(t *testing.T, tasks []core.Task) {
	t.Helper()

	found := make(map[string]core.Task, len(tasks))
	for _, task := range tasks {
		found[task.Ref] = task
	}

	want := []struct {
		ref     string
		status  string
		title   string
		blocked bool
	}{
		{"tk-1", statusOpen, "Preparar el lanzamiento", true},
		{"tk-2", "done", "Escribir el anuncio", false},
		{"tk-3", statusOpen, "Comprobar las copias", false},
	}

	for _, entry := range want {
		task, ok := found[entry.ref]
		if !ok {
			t.Errorf("la lista no trae %s", entry.ref)

			continue
		}

		if task.Status != entry.status {
			t.Errorf("%s sale en %q, se esperaba %q", entry.ref, task.Status, entry.status)
		}

		if task.Title != entry.title {
			t.Errorf("%s se titula %q, se esperaba %q", entry.ref, task.Title, entry.title)
		}

		if task.Blocked != entry.blocked {
			t.Errorf("%s sale con bloqueado=%v, se esperaba %v", entry.ref, task.Blocked, entry.blocked)
		}
	}
}
