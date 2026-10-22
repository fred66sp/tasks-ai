package cli_test

import (
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
