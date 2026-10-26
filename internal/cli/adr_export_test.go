package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// exportReport es la respuesta de la exportación tal como viaja en JSON.
type exportReport struct {
	Dir     string   `json:"dir"`
	Files   []string `json:"files"`
	Drifted []string `json:"drifted"`
}

// exportTo escribe la exportación en el directorio dado.
func exportTo(t *testing.T, dir string) exportReport {
	t.Helper()

	code, out, errOut := run(t, "adr", "export", "--dir", dir)
	if code != 0 {
		t.Fatalf("adr export salió con %d: %s", code, errOut)
	}

	var report exportReport
	decode(t, out, &report)

	return report
}

// exported escribe la exportación en un directorio temporal y lo devuelve con
// lo que respondió.
func exported(t *testing.T) (string, exportReport) {
	t.Helper()

	dir := t.TempDir()

	return dir, exportTo(t, dir)
}

// readFile lee un archivo de la exportación.
func readFile(t *testing.T, path string) string {
	t.Helper()

	//nolint:gosec // la ruta sale de un directorio temporal de la prueba
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("leyendo %s: %v", path, err)
	}

	return string(raw)
}

//nolint:paralleltest // el test cambia de carpeta con t.Chdir: en paralelo pisaría la carpeta de otra prueba
func TestRun_adrExportWritesAFilePerADRAndTheIndex(t *testing.T) {
	adrProject(t)

	createADR(t, "La primera")
	createADR(t, "La segunda")

	dir, report := exported(t)

	if len(report.Files) != 3 {
		t.Fatalf("la exportación escribió %+v, se esperaban dos archivos y el índice", report.Files)
	}

	first := readFile(t, filepath.Join(dir, "001-la-primera.md"))

	for _, wanted := range []string{"status: proposed", "# ADR-001: La primera", "do not edit by hand"} {
		if !strings.Contains(first, wanted) {
			t.Errorf("el archivo exportado no lleva %q:\n%s", wanted, first)
		}
	}

	index := readFile(t, filepath.Join(dir, "README.md"))
	for _, wanted := range []string{
		"- [ADR-001: La primera](001-la-primera.md) (proposed,",
		"- [ADR-002: La segunda](002-la-segunda.md) (proposed,",
	} {
		if !strings.Contains(index, wanted) {
			t.Errorf("el índice no lleva %q:\n%s", wanted, index)
		}
	}

	// Exportar otra vez no cambia nada, y la comprobación lo dice.
	code, out, errOut := run(t, "adr", "export", "--dir", dir)
	if code != 0 {
		t.Fatalf("repetir la exportación salió con %d: %s", code, errOut)
	}

	decode(t, out, &report)

	if len(report.Drifted) != 0 {
		t.Errorf("repetir la exportación informó de deriva: %+v", report.Drifted)
	}

	if code, _, errOut := run(t, "adr", "export", "--dir", dir, "--check"); code != 0 {
		t.Errorf("adr export --check sobre una exportación al día salió con %d: %s", code, errOut)
	}
}

//nolint:paralleltest // el test cambia de carpeta con t.Chdir: en paralelo pisaría la carpeta de otra prueba
func TestRun_adrExportCheckFindsAndRevertsAHandEdit(t *testing.T) {
	adrProject(t)

	created := createADR(t, "La primera")
	dir, _ := exported(t)

	edited := filepath.Join(dir, "001-la-primera.md")
	if err := os.WriteFile(edited, []byte("# escrito a mano\n"), 0o600); err != nil {
		t.Fatalf("editando el archivo exportado: %v", err)
	}

	code, out, errOut := run(t, "adr", "export", "--dir", dir, "--check")
	if code != 1 {
		t.Fatalf("--check sobre un archivo editado salió con %d, se esperaba 1: %s", code, errOut)
	}

	var report exportReport
	decode(t, out, &report)

	if len(report.Drifted) != 1 || report.Drifted[0] != "001-la-primera.md" {
		t.Errorf("--check informó de %+v, se esperaba el archivo editado", report.Drifted)
	}

	// Exportar de nuevo devuelve el archivo a lo que dice el registro.
	exportTo(t, dir)

	if readFile(t, edited) == "# escrito a mano\n" {
		t.Error("la exportación no revirtió la edición a mano")
	}

	if code, _, errOut := run(t, "adr", "export", "--dir", dir, "--check"); code != 0 {
		t.Errorf("--check tras exportar salió con %d: %s", code, errOut)
	}

	if created.Ref != "ADR-001" {
		t.Errorf("el ADR exportado es %q, se esperaba ADR-001", created.Ref)
	}
}
