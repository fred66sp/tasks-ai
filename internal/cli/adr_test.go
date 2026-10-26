package cli_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"tasks-ai/internal/core"
	"testing"
)

// adrProject instala el producto en un proyecto temporal y deja el directorio
// de trabajo dentro de él.
func adrProject(t *testing.T) {
	t.Helper()

	t.Chdir(t.TempDir())

	if code, _, errOut := run(t, "init"); code != 0 {
		t.Fatalf("init salió con %d: %s", code, errOut)
	}
}

// createADR crea un ADR desde el CLI y devuelve lo que respondió.
func createADR(t *testing.T, title string) core.ADR {
	t.Helper()

	code, out, errOut := run(t, "adr", "create", "--title", title, "--deciders", "Alfredo")
	if code != 0 {
		t.Fatalf("adr create salió con %d: %s", code, errOut)
	}

	var created core.ADR
	decode(t, out, &created)

	return created
}

//nolint:paralleltest // el test cambia de carpeta con t.Chdir: en paralelo pisaría la carpeta de otra prueba
func TestRun_adrCreatesShowsAndListsTheADR(t *testing.T) {
	adrProject(t)

	created := createADR(t, "Los ADR viven en la base")

	if created.Number != 1 || created.Ref != "ADR-001" || created.Status != "proposed" {
		t.Errorf("el ADR nació como %+v, se esperaba ADR-001 propuesto", created)
	}

	code, out, errOut := run(t, "adr", "show", "ADR-001")
	if code != 0 {
		t.Fatalf("adr show salió con %d: %s", code, errOut)
	}

	var read core.ADR
	decode(t, out, &read)

	if read.Ref != created.Ref || read.Slug != created.Slug {
		t.Errorf("adr show devolvió %+v, se esperaba %+v", read, created)
	}

	code, out, errOut = run(t, "adr", "list")
	if code != 0 {
		t.Fatalf("adr list salió con %d: %s", code, errOut)
	}

	if strings.Contains(out, `"context"`) {
		t.Errorf("el listado trae cuerpos, se esperaban resúmenes: %s", out)
	}

	var listed []core.ADRSummary
	decode(t, out, &listed)

	if len(listed) != 1 || listed[0].Ref != "ADR-001" {
		t.Errorf("el listado es %+v, se esperaba el ADR-001", listed)
	}
}

//nolint:paralleltest // el test cambia de carpeta con t.Chdir: en paralelo pisaría la carpeta de otra prueba
func TestRun_adrEditChangesOnlyWhatItNamesAndRefusesTheFrozenBody(t *testing.T) {
	adrProject(t)

	createADR(t, "Una decisión")

	code, out, errOut := run(t, "adr", "edit", "1", "--decision", "la decisión")
	if code != 0 {
		t.Fatalf("adr edit salió con %d: %s", code, errOut)
	}

	var edited core.ADR
	decode(t, out, &edited)

	if edited.Decision != "la decisión" {
		t.Errorf("la decisión guardada es %q, se esperaba la editada", edited.Decision)
	}

	if code, _, errOut := run(t, "adr", "edit", "1", "--status", "accepted"); code != 0 {
		t.Fatalf("aceptar el ADR salió con %d: %s", code, errOut)
	}

	code, _, errOut = run(t, "adr", "edit", "1", "--decision", "otra decisión")
	if code != 2 {
		t.Errorf("cambiar el cuerpo congelado salió con %d, se esperaba 2: %s", code, errOut)
	}

	if text := errorText(t, errOut); !strings.Contains(text, "frozen") {
		t.Errorf("el rechazo no dice que el cuerpo está congelado: %q", text)
	}

	code, out, errOut = run(t, "adr", "edit", "1", "--note", "se comprobó al implementarlo")
	if code != 0 {
		t.Fatalf("añadir la nota salió con %d: %s", code, errOut)
	}

	decode(t, out, &edited)

	if edited.MoreInfo == nil || !strings.Contains(*edited.MoreInfo, "se comprobó al implementarlo") {
		t.Errorf("la información adicional es %v, se esperaba la nota", edited.MoreInfo)
	}

	if code, _, _ := run(t, "adr", "show", "99"); code != 1 {
		t.Errorf("mostrar un ADR que no está salió con %d, se esperaba 1", code)
	}
}

// adrSourceDir escribe un directorio de ADR exportados, con uno sustituido por
// otro que aún no está guardado en la base.
func adrSourceDir(t *testing.T) string {
	t.Helper()

	source := t.TempDir()

	files := map[string]string{
		"001-la-primera.md": adrFile(1, "accepted", "2026-10-20", "La primera", "el contexto", "la decisión"),
		"002-la-segunda.md": adrFile(
			2, "superseded by [ADR-003](003-la-tercera.md)",
			"2026-10-21", "La segunda", "el contexto anterior", "la decisión anterior",
		),
		"003-la-tercera.md": adrFile(3, "accepted", "2026-10-22", "La tercera", "el contexto nuevo", "la decisión nueva"),
		"README.md":         "# Decisiones\n\n- [ADR-001: La primera](001-la-primera.md) (accepted, 2026-10-20)\n",
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(source, name), []byte(content), 0o600); err != nil {
			t.Fatalf("escribiendo %s: %v", name, err)
		}
	}

	return source
}

// importADR runs the import and returns what it answered.
func importADR(t *testing.T, dir string) core.ImportResult {
	t.Helper()

	code, out, errOut := run(t, "adr", "import", "--dir", dir)
	if code != 0 {
		t.Fatalf("adr import salió con %d: %s", code, errOut)
	}

	var result core.ImportResult
	decode(t, out, &result)

	return result
}

//nolint:paralleltest // el test cambia de carpeta con t.Chdir: en paralelo pisaría la carpeta de otra prueba
func TestRun_adrImportReadsTheSectionsAndTheSubstitute(t *testing.T) {
	source := adrSourceDir(t)

	adrProject(t)

	result := importADR(t, source)

	if len(result.Imported) != 3 || len(result.Skipped) != 0 {
		t.Errorf("la importación dejó %+v, se esperaban los tres importados", result)
	}

	code, out, errOut := run(t, "adr", "show", "2")
	if code != 0 {
		t.Fatalf("adr show del importado salió con %d: %s", code, errOut)
	}

	var second core.ADR
	decode(t, out, &second)

	if second.SupersededBy == nil || *second.SupersededBy != "ADR-003" {
		t.Errorf("el sustituto importado es %v, se esperaba ADR-003", second.SupersededBy)
	}

	if second.Context != "el contexto anterior" || second.Decision != "la decisión anterior" {
		t.Errorf("el cuerpo importado es %+v, se esperaba el del archivo", second)
	}
}

//nolint:paralleltest // el test cambia de carpeta con t.Chdir: en paralelo pisaría la carpeta de otra prueba
func TestRun_adrImportRepeatsWithoutDuplicating(t *testing.T) {
	source := adrSourceDir(t)

	adrProject(t)
	importADR(t, source)

	result := importADR(t, source)

	if len(result.Imported) != 0 || len(result.Skipped) != 3 {
		t.Errorf("repetir la importación dejó %+v, se esperaban los tres saltados", result)
	}
}

//nolint:paralleltest // el test cambia de carpeta con t.Chdir: en paralelo pisaría la carpeta de otra prueba
func TestRun_adrHelpNamesItsCommandsInEnglish(t *testing.T) {
	adrProject(t)

	code, out, errOut := run(t, "adr", "--help")
	if code != 0 {
		t.Fatalf("adr --help salió con %d: %s", code, errOut)
	}

	for _, command := range []string{"list", "show", "create", "edit", "import"} {
		if !strings.Contains(out, command) {
			t.Errorf("la ayuda no nombra el comando %q: %s", command, out)
		}
	}

	if !strings.Contains(out, "Reads and writes the ADR") {
		t.Errorf("la ayuda no está en inglés: %s", out)
	}
}

// adrFile builds the markdown of an ADR the way the export writes it.
func adrFile(number int, status, day, title, context, decision string) string {
	return "---\n" +
		"status: " + status + "\n" +
		"date: " + day + "\n" +
		"decision-makers: \"Alfredo González Portero\"\n" +
		"---\n\n" +
		fmt.Sprintf("# ADR-%03d: %s\n\n", number, title) +
		"## Contexto\n\n" + context + "\n\n" +
		"## Decisión\n\n" + decision + "\n\n" +
		"## Consecuencias\n\nlas consecuencias\n"
}
