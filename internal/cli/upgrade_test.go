package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"tasks-ai/internal/core"
	"tasks-ai/internal/install"
	"tasks-ai/internal/settings"
	"testing"
)

// TestRun_upgradeCompletesALegacyInstallation recorre la actualización por el
// CLI: sale con 0, deja la versión y retira las extensiones y sus referencias.

// shippedExtensionFile is the exact content the v0.1.0 release shipped as the
// tsconfig.json of the two extensions; the hash gate in install retires only
// a file carrying these bytes.
const shippedExtensionFile = `{
	// Comprueba los .ts que Node ejecuta sin compilar: no se emite nada.
	"compilerOptions": {
		"target": "ES2023",
		"lib": ["ES2023"],
		"module": "NodeNext",
		"moduleResolution": "NodeNext",
		"types": ["node"],
		"strict": true,
		"exactOptionalPropertyTypes": true,
		"noImplicitOverride": true,
		"isolatedModules": true,
		"verbatimModuleSyntax": true,
		"erasableSyntaxOnly": true,
		"allowImportingTsExtensions": true,
		"noEmit": true,
		"skipLibCheck": true
	},
	"include": ["**/*.ts"]
}
`

func TestRun_upgradeCompletesALegacyInstallation(t *testing.T) {
	root := legacyCLIProject(t, `{
  // la configuración del proyecto
  "plugins": ["./otro/plugin", "./.tasks-ai/extensions/opencode"]
}
`)

	t.Chdir(root)

	code, out, errOut := run(t, "upgrade")
	if code != 0 {
		t.Fatalf("upgrade salió con %d: %s", code, errOut)
	}

	var report install.UpgradeReport
	decode(t, out, &report)

	if report.Version != core.Version || len(report.Pending) != 0 {
		t.Errorf("el informe quedó %+v, se esperaba completo y sin pendientes", report)
	}

	product := filepath.Join(root, install.DirName)

	//nolint:gosec // la ruta la prepara la prueba
	if got, err := os.ReadFile(filepath.Join(product, install.VersionName)); err != nil || string(got) != core.Version+"\n" {
		t.Errorf("la versión quedó %q (%v), se esperaba %q", got, err, core.Version)
	}

	for _, name := range []string{"pi", "opencode"} {
		if _, err := os.Stat(filepath.Join(product, "extensions", name)); err == nil {
			t.Errorf("upgrade conservó la extensión %s", name)
		}
	}

	if !strings.Contains(out, `"from": "`+install.LegacyVersion+`"`) {
		t.Errorf("el informe no dice de qué versión venía: %s", out)
	}
}

// TestRun_upgradeFailsWhenAReferenceStaysAlive comprueba que una actualización
// incompleta sale con 2, deja la VERSION sin escribir y avisa por stderr.
func TestRun_upgradeFailsWhenAReferenceStaysAlive(t *testing.T) {
	root := legacyCLIProject(t, `{
  "plugins": [
    "./.tasks-ai/extensions/opencode", // que se quede
    "./otro/plugin"
  ]
}
`)

	t.Chdir(root)

	code, out, errOut := run(t, "upgrade")
	if code != 2 {
		t.Fatalf("upgrade con una referencia viva salió con %d, se esperaba 2: %s", code, errOut)
	}

	var report install.UpgradeReport
	decode(t, out, &report)

	if report.Version != "" {
		t.Errorf("la actualización escribió la versión %q", report.Version)
	}

	if len(report.Pending) == 0 {
		t.Error("el informe no lista el pendiente")
	}

	if !strings.Contains(errOut, "warning:") {
		t.Errorf("stderr no avisa del pendiente: %s", errOut)
	}

	product := filepath.Join(root, install.DirName)
	if _, err := os.Stat(filepath.Join(product, install.VersionName)); err == nil {
		t.Error("la actualización incompleta escribió la VERSION")
	}

	if _, err := os.Stat(filepath.Join(product, "extensions", "opencode")); err != nil {
		t.Errorf("la actualización retiró la extensión con una referencia viva: %v", err)
	}
}

// TestRun_upgradeRefusesADowngrade comprueba el rechazo de una instalación más
// nueva, con las dos versiones en el mensaje y sin escribir nada.
func TestRun_upgradeRefusesADowngrade(t *testing.T) {
	root := legacyCLIProject(t, `{"plugins": []}`+"\n")

	t.Chdir(root)

	product := filepath.Join(root, install.DirName)
	if err := os.WriteFile(filepath.Join(product, install.VersionName), []byte("9.9.9\n"), 0o600); err != nil {
		t.Fatalf("preparando la versión: %v", err)
	}

	code, out, errOut := run(t, "upgrade")
	if code != 2 {
		t.Fatalf("upgrade de una instalación más nueva salió con %d, se esperaba 2: %s", code, errOut)
	}

	if out != "" {
		t.Errorf("el rechazo escribió en la salida: %q", out)
	}

	message := errorText(t, errOut)
	if !strings.Contains(message, "9.9.9") || !strings.Contains(message, core.Version) {
		t.Errorf("el rechazo no nombra las dos versiones: %q", message)
	}

	//nolint:gosec // la ruta la prepara la prueba
	if got, err := os.ReadFile(filepath.Join(product, install.VersionName)); err != nil || string(got) != "9.9.9\n" {
		t.Errorf("el rechazo reescribió la versión: %q (%v)", got, err)
	}
}

// TestRun_upgradeSaysWhenThereIsNoInstallation comprueba el rechazo de uso.
func TestRun_upgradeSaysWhenThereIsNoInstallation(t *testing.T) {
	t.Chdir(t.TempDir())

	code, _, errOut := run(t, "upgrade")
	if code != 2 {
		t.Fatalf("upgrade sin instalación salió con %d, se esperaba 2: %s", code, errOut)
	}

	if message := errorText(t, errOut); !strings.Contains(message, "no project") {
		t.Errorf("el error no dice que no hay instalación: %q", message)
	}
}

// TestRun_initLeavesNoRetiredFields comprueba que el JSON de init no expone las
// extensiones y sí la versión, y que deja de escribirlas.
func TestRun_initLeavesNoRetiredFields(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	code, out, errOut := run(t, "init")
	if code != 0 {
		t.Fatalf("init salió con %d: %s", code, errOut)
	}

	var fields map[string]json.RawMessage
	decode(t, out, &fields)

	for _, retired := range []string{"extension", "opencode"} {
		if _, ok := fields[retired]; ok {
			t.Errorf("el JSON de init conserva el campo %q: %s", retired, out)
		}
	}

	var version struct {
		Version string `json:"version"`
	}
	decode(t, out, &version)

	if version.Version != core.Version {
		t.Errorf("init devolvió la versión %q, se esperaba %q", version.Version, core.Version)
	}

	product := filepath.Join(dir, install.DirName)

	if _, err := os.Stat(filepath.Join(product, "extensions")); err == nil {
		t.Error("init dejó la carpeta de extensiones retirada")
	}

	//nolint:gosec // la ruta la prepara la prueba
	if got, err := os.ReadFile(filepath.Join(product, install.VersionName)); err != nil || string(got) != core.Version+"\n" {
		t.Errorf("init no dejó la versión: %q (%v)", got, err)
	}
}

// legacyCLIProject prepares a legacy installation with customised instructions,
// so upgrade does not need the v0.1.0 template to run through the command line.
func legacyCLIProject(t *testing.T, opencode string) string {
	t.Helper()

	root := t.TempDir()
	product := filepath.Join(root, install.DirName)

	for _, dir := range []string{"bin", filepath.Join("extensions", "pi"), filepath.Join("extensions", "opencode")} {
		if err := os.MkdirAll(filepath.Join(product, dir), 0o755); err != nil { //nolint:gosec // la prueba usa los permisos del producto real
			t.Fatalf("creando %s: %v", dir, err)
		}
	}

	if err := settings.Save(product, settings.Default()); err != nil {
		t.Fatalf("preparando los ajustes: %v", err)
	}

	writeCLIFile(t, filepath.Join(product, "tasks.db"), "las tareas del proyecto")
	writeCLIFile(t, filepath.Join(product, "bin", "tasks-ai"), "binario viejo")
	writeCLIFile(t, filepath.Join(product, "extensions", "pi", "tsconfig.json"), shippedExtensionFile)
	writeCLIFile(t, filepath.Join(product, "extensions", "opencode", "tsconfig.json"), shippedExtensionFile)
	writeCLIFile(t, filepath.Join(product, "AGENTS.md"), "# Mis instrucciones\n")
	writeCLIFile(t, filepath.Join(root, "opencode.jsonc"), opencode)

	if err := os.MkdirAll(filepath.Join(root, ".pi"), 0o755); err != nil { //nolint:gosec // la prueba usa los permisos del producto real
		t.Fatalf("creando .pi: %v", err)
	}

	writeCLIFile(t, filepath.Join(root, ".pi", "settings.json"), `{"packages": ["../.tasks-ai/extensions/pi"]}`+"\n")

	return root
}

func writeCLIFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { //nolint:gosec // el fichero de prueba usa los permisos del producto real
		t.Fatalf("escribiendo %s: %v", path, err)
	}
}
