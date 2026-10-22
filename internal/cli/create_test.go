package cli_test

import (
	"strings"
	"tasks-ai/internal/core"
	"testing"
)

// TestRun_createWithAParentLeavesTheTagsAlone recorre el primer caso del
// criterio de tk-158 por la interfaz pública: la tarea que nace con padre no
// recibe ninguna etiqueta y el producto no crea la estructural en el catálogo.
func TestRun_createWithAParentLeavesTheTagsAlone(t *testing.T) {
	t.Chdir(t.TempDir())
	installProject(t)

	parent := createTask(t, "Padre")
	child := childOf(t, "Hija", parent.Ref)

	assertTags(t, child)

	code, out, errOut := run(t, "tags")
	if code != 0 {
		t.Fatalf("tags salió con %d: %s", code, errOut)
	}

	var catalog []core.TagWithUses
	decode(t, out, &catalog)

	if len(catalog) != 0 {
		t.Errorf("el producto creó etiquetas en el catálogo al crear con padre: %v", catalog)
	}
}

// TestRun_createStoresTheTagsItIsGiven recorre el primer caso del criterio de
// tk-170 por la interfaz pública: la tarea nace con las etiquetas que se le dan
// y la lectura las confirma.
func TestRun_createStoresTheTagsItIsGiven(t *testing.T) {
	t.Chdir(t.TempDir())
	installProject(t)

	code, out, errOut := run(t, "create", "--title", "Con etiquetas", "--tags", newTag(t, "docs", "cli"))
	if code != 0 {
		t.Fatalf("create --tags salió con %d: %s", code, errOut)
	}

	var created core.Task

	decode(t, out, &created)

	assertTags(t, created, "docs", "cli")
	assertTags(t, showTask(t, created.Ref), "docs", "cli")
}

// TestRun_createRefusesAnUnknownTag recorre el segundo caso del criterio: un
// nombre fuera del catálogo se rechaza, el rechazo nombra la etiqueta y no deja
// la tarea escrita.
func TestRun_createRefusesAnUnknownTag(t *testing.T) {
	t.Chdir(t.TempDir())
	installProject(t)

	known := newTag(t, "docs")

	code, out, errOut := run(t, "create", "--title", "Con una que no existe", "--tags", known+",inventada")
	if code != 2 {
		t.Fatalf("create con una etiqueta desconocida salió con %d, se esperaba 2: %s", code, errOut)
	}

	if out != "" {
		t.Errorf("un error no debe escribir en la salida: %q", out)
	}

	if message := errorText(t, errOut); !strings.Contains(message, "unknown tag") || !strings.Contains(message, "inventada") {
		t.Errorf("el error (%q) no nombra la etiqueta desconocida", message)
	}

	code, out, errOut = run(t, "list")
	if code != 0 {
		t.Fatalf("list salió con %d: %s", code, errOut)
	}

	var tasks []core.Task

	decode(t, out, &tasks)

	if len(tasks) != 0 {
		t.Errorf("el rechazo dejó tareas escritas: %+v", tasks)
	}
}

// TestRun_createWithAParentKeepsOnlyTheTagsGiven recorre el tercer caso del
// criterio: la forma la escribe la persona, así que crear con padre y etiquetas
// no añade la estructural.
func TestRun_createWithAParentKeepsOnlyTheTagsGiven(t *testing.T) {
	t.Chdir(t.TempDir())
	installProject(t)

	parent := createTask(t, "Padre")

	code, out, errOut := run(t, "create", "--title", "Hija", "--parent", parent.Ref, "--tags", newTag(t, "docs"))
	if code != 0 {
		t.Fatalf("create --parent con etiquetas salió con %d: %s", code, errOut)
	}

	var child core.Task

	decode(t, out, &child)

	assertTags(t, child, "docs")
}

// TestRun_createWithoutTagsStoresNone fija el cuarto caso del criterio: sin
// --tags la tarea nace como nacía, sin ninguna etiqueta.
func TestRun_createWithoutTagsStoresNone(t *testing.T) {
	t.Chdir(t.TempDir())
	installProject(t)

	code, out, errOut := run(t, "create", "--title", "Sin etiquetas")
	if code != 0 {
		t.Fatalf("create sin etiquetas salió con %d: %s", code, errOut)
	}

	var created core.Task

	decode(t, out, &created)

	assertTags(t, created)
}
