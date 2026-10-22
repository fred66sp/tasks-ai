package cli_test

import (
	"slices"
	"strings"
	"tasks-ai/internal/core"
	"testing"
)

// TestRun_updateAssignsTheParent recorre el primer caso del criterio de tk-158
// por la interfaz pública: --parent le da padre a una tarea que no lo tenía y
// no le toca las etiquetas, que las pone quien crea la tarea.
func TestRun_updateAssignsTheParent(t *testing.T) {
	t.Chdir(t.TempDir())
	installProject(t)

	parent := createTask(t, "Padre")
	loose := taggedTask(t, "Suelta", "docs")

	task := updateTask(t, loose.Ref, "--parent", parent.Ref)

	assertParent(t, task, &parent.ID)
	assertTags(t, task, "docs")
}

// TestRun_updateMovesTheParent recorre el segundo caso del criterio: --parent
// mueve al padre que la tarea ya tenía al que se le da, sin tocar etiquetas.
func TestRun_updateMovesTheParent(t *testing.T) {
	t.Chdir(t.TempDir())
	installProject(t)

	first := createTask(t, "Padre uno")
	second := createTask(t, "Padre dos")
	child := childOf(t, "Hija", first.Ref)

	task := updateTask(t, child.Ref, "--parent", second.Ref)

	assertParent(t, task, &second.ID)
	assertTags(t, task)
}

// TestRun_updateRemovesTheParent recorre el tercer caso del criterio: un valor
// vacío quita el padre, deja la tarea en su nivel y no toca las etiquetas.
func TestRun_updateRemovesTheParent(t *testing.T) {
	t.Chdir(t.TempDir())
	installProject(t)

	parent := createTask(t, "Padre")
	other := createTask(t, "Otro padre")
	child := childOf(t, "Hija", parent.Ref)

	task := updateTask(t, child.Ref, "--tags", newTag(t, "docs"))

	if task.ParentID == nil || *task.ParentID != parent.ID {
		t.Fatalf("la hija quedó con el padre %v, se esperaba %d", task.ParentID, parent.ID)
	}

	task = updateTask(t, child.Ref, "--parent", other.Ref)

	assertTags(t, task, "docs")

	task = updateTask(t, child.Ref, "--parent", "")

	assertParent(t, task, nil)
	assertTags(t, task, "docs")
}

// TestRun_updateTreatsTheStructuralTagLikeAnyOther recorre el tercer caso del
// criterio: el producto no tiene etiquetas de primera, así que una estructural
// que falta en el catálogo se rechaza como cualquier nombre desconocido y, con
// ella creada, se asigna como las demás.
func TestRun_updateTreatsTheStructuralTagLikeAnyOther(t *testing.T) {
	t.Chdir(t.TempDir())
	installProject(t)

	task := createTask(t, "La que lleva épica")

	code, out, errOut := run(t, "update", task.Ref, "--tags", "zona")
	if code != 2 {
		t.Fatalf("update --tags zona salió con %d, se esperaba 2: %s", code, errOut)
	}

	if out != "" {
		t.Errorf("un error no debe escribir en la salida: %q", out)
	}

	if message := errorText(t, errOut); !strings.Contains(message, "unknown tag") {
		t.Errorf("el error (%q) no dice que la etiqueta es desconocida", message)
	}

	assertTags(t, showTask(t, task.Ref))

	if code, _, errOut := run(t, "tag", "create", "--name", "zona"); code != 0 {
		t.Fatalf("tag create zona salió con %d: %s", code, errOut)
	}

	assertTags(t, updateTask(t, task.Ref, "--tags", "zona"), "zona")

	// La estructural que deja la instalación se asigna como cualquier otra.
	assertTags(t, updateTask(t, task.Ref, "--tags", "epic"), "epic")
}

// TestRun_updateRefusesAParentThatDoesNotResolve comprueba el primer rechazo del
// contrato: una referencia que no resuelve sale con el código 1 y no escribe
// nada.
func TestRun_updateRefusesAParentThatDoesNotResolve(t *testing.T) {
	t.Chdir(t.TempDir())
	installProject(t)

	parent := createTask(t, "Padre")
	loose := createTask(t, "Sin padre")

	code, out, errOut := run(t, "update", loose.Ref, "--parent", "tk-404")
	if code != 1 {
		t.Fatalf("update --parent tk-404 salió con %d, se esperaba 1: %s", code, errOut)
	}

	if out != "" {
		t.Errorf("un error no debe escribir en la salida: %q", out)
	}

	if message := errorText(t, errOut); !strings.Contains(message, "tk-404") {
		t.Errorf("el error (%q) no dice qué referencia no resolvió", message)
	}

	if task := showTask(t, loose.Ref); task.ParentID != nil {
		t.Errorf("un rechazo escribió el padre %d", *task.ParentID)
	}

	code, _, errOut = run(t, "update", "tk-404", "--parent", parent.Ref)
	if code != 1 {
		t.Errorf("update sobre una tarea que no resuelve salió con %d, se esperaba 1: %s", code, errOut)
	}
}

// TestRun_updateRefusesACycleAsParent comprueba el segundo rechazo del contrato:
// tomar como padre la propia tarea o una de sus descendientes es entrada
// inválida —código 2— y no escribe nada.
func TestRun_updateRefusesACycleAsParent(t *testing.T) {
	t.Chdir(t.TempDir())
	installProject(t)

	root := createTask(t, "Raíz")
	child := childOf(t, "Hija", root.Ref)

	cases := []struct {
		name   string
		ref    string
		parent string
	}{
		{name: "itself", ref: root.Ref, parent: root.Ref},
		{name: "one of its descendants", ref: root.Ref, parent: child.Ref},
	}

	//nolint:paralleltest // los casos comparten la carpeta que t.Chdir puso: en paralelo se pisarían
	for _, refused := range cases {
		t.Run(refused.name, func(t *testing.T) {
			code, out, errOut := run(t, "update", refused.ref, "--parent", refused.parent)
			if code != 2 {
				t.Fatalf("update --parent %s salió con %d, se esperaba 2: %s", refused.parent, code, errOut)
			}

			if out != "" {
				t.Errorf("un error no debe escribir en la salida: %q", out)
			}

			if message := errorText(t, errOut); !strings.Contains(message, "descendant") {
				t.Errorf("el mensaje (%q) no explica el ciclo", message)
			}
		})
	}

	if task := showTask(t, root.Ref); task.ParentID != nil {
		t.Errorf("un rechazo escribió el padre %d al padre", *task.ParentID)
	}

	assertParent(t, showTask(t, child.Ref), &root.ID)
}

// TestRun_updateHelpDocumentsTheParentFlag comprueba que la ayuda del cambio
// documenta la bandera nueva.
func TestRun_updateHelpDocumentsTheParentFlag(t *testing.T) {
	t.Chdir(t.TempDir())
	installProject(t)

	code, out, errOut := run(t, "update", "--help")
	if code != 0 {
		t.Fatalf("update --help salió con %d: %s", code, errOut)
	}

	if !strings.Contains(out, "--parent") {
		t.Errorf("la ayuda de update no documenta --parent:\n%s", out)
	}
}

// assertTags compara las etiquetas de una tarea con las que se esperan, ninguna
// cuando no se pasa.
func assertTags(t *testing.T, task core.Task, want ...string) {
	t.Helper()

	if !slices.Equal(slices.Sorted(slices.Values(task.Tags)), slices.Sorted(slices.Values(want))) {
		t.Errorf("la tarea %q lleva %v, se esperaba %v", task.Ref, task.Tags, want)
	}
}

// assertParent compara el padre de una tarea con el que se espera, ninguno
// cuando no se pasa.
func assertParent(t *testing.T, task core.Task, want *int64) {
	t.Helper()

	switch {
	case want == nil && task.ParentID != nil:
		t.Errorf("el padre quedó en %d, se esperaba ninguno", *task.ParentID)
	case want != nil && task.ParentID == nil:
		t.Errorf("la tarea quedó sin padre, se esperaba %d", *want)
	case want != nil && *task.ParentID != *want:
		t.Errorf("el padre quedó en %d, se esperaba %d", *task.ParentID, *want)
	}
}

// installProject instala el producto en la carpeta de trabajo.
func installProject(t *testing.T) {
	t.Helper()

	if code, _, errOut := run(t, "init"); code != 0 {
		t.Fatalf("init salió con %d: %s", code, errOut)
	}
}

func createTask(t *testing.T, title string) core.Task {
	t.Helper()

	code, out, errOut := run(t, "create", "--title", title)
	if code != 0 {
		t.Fatalf("create %q salió con %d: %s", title, code, errOut)
	}

	var task core.Task
	decode(t, out, &task)

	return task
}

// taggedTask crea una tarea sin padre con las etiquetas que se le dan, que
// antes pasan al catálogo.
func taggedTask(t *testing.T, title string, names ...string) core.Task {
	t.Helper()

	task := createTask(t, title)

	return updateTask(t, task.Ref, "--tags", newTag(t, names...))
}

// newTag deja las etiquetas en el catálogo y devuelve la lista que --tags
// lleva, separada por comas.
func newTag(t *testing.T, names ...string) string {
	t.Helper()

	for _, name := range names {
		code, _, errOut := run(t, "tag", "create", "--name", name)
		if code == 0 || strings.Contains(errOut, "already exists") {
			// La instalación deja el vocabulario del contrato: una etiqueta que
			// ya existe vale igual, lo que importa es que esté en el catálogo.
			continue
		}

		t.Fatalf("tag create %q salió con %d: %s", name, code, errOut)
	}

	return strings.Join(names, ",")
}

func childOf(t *testing.T, title, parent string) core.Task {
	t.Helper()

	code, out, errOut := run(t, "create", "--title", title, "--parent", parent)
	if code != 0 {
		t.Fatalf("create %q --parent %s salió con %d: %s", title, parent, code, errOut)
	}

	var task core.Task
	decode(t, out, &task)

	return task
}

func updateTask(t *testing.T, ref string, args ...string) core.Task {
	t.Helper()

	code, out, errOut := run(t, append([]string{"update", ref}, args...)...)
	if code != 0 {
		t.Fatalf("update %s %v salió con %d: %s", ref, args, code, errOut)
	}

	var task core.Task
	decode(t, out, &task)

	return task
}

func showTask(t *testing.T, ref string) core.Task {
	t.Helper()

	code, out, errOut := run(t, "show", ref)
	if code != 0 {
		t.Fatalf("show %s salió con %d: %s", ref, code, errOut)
	}

	var task core.Task
	decode(t, out, &task)

	return task
}
