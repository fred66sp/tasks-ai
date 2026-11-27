package install

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"tasks-ai/internal/core"
	"tasks-ai/internal/settings"
	"testing"
)

// TestUpgrade_migratesALegacyInstallation recorre el criterio de una instalación
// v0.1.0: los datos no se tocan, las extensiones y sus referencias desaparecen,
// las instrucciones se regeneran y la versión se escribe al final.
//
//nolint:gocyclo // recorre la migración completa y debe quedarse lineal
func TestUpgrade_migratesALegacyInstallation(t *testing.T) {
	t.Parallel()

	root, product := legacyProject(t)

	opencode := filepath.Join(root, "opencode.jsonc")
	pi := filepath.Join(root, ".pi", "settings.json")

	writeFileAt(t, opencode, `{
  // la configuración del proyecto
  "$schema": "https://opencode.ai/config.json",
  "plugins": ["./otro/plugin", "./.tasks-ai/extensions/opencode"],
  "model": "x" // comentario final
}
`)
	writeFileAt(t, pi, "{\n  \"packages\": [\"otro/paquete\", \"../.tasks-ai/extensions/pi\"]\n}\n")

	report, err := Upgrade(root)
	if err != nil {
		t.Fatalf("Upgrade devolvió %v", err)
	}

	if report.From != LegacyVersion || report.To != core.Version || report.Version != core.Version {
		t.Errorf("el informe de versiones es %+v, se esperaba %s → %s", report, LegacyVersion, core.Version)
	}

	if len(report.Pending) != 0 {
		t.Errorf("la instalación quedó pendiente: %v", report.Pending)
	}

	if report.Instructions != instructionsUpdated {
		t.Errorf("las instrucciones quedaron en %q, se esperaba %q", report.Instructions, instructionsUpdated)
	}

	if got := readFileAt(t, filepath.Join(product, "tasks.db")); got != "las tareas del proyecto" {
		t.Errorf("la actualización tocó los datos: %q", got)
	}

	if got := readFileAt(t, filepath.Join(product, instructionsName)); got != string(instructionsFor(settings.DefaultPrefix)) {
		t.Error("las instrucciones no son la plantilla nueva con el prefijo del proyecto")
	}

	if got := readFileAt(t, filepath.Join(product, VersionName)); got != core.Version+"\n" {
		t.Errorf("la versión quedó %q, se esperaba %q", got, core.Version)
	}

	if got := readFileAt(t, filepath.Join(product, "bin", "tasks-ai")); got == "binario viejo" {
		t.Error("la actualización no reemplazó el binario")
	}

	for _, name := range []string{"pi", "opencode"} {
		if _, err := os.Stat(filepath.Join(product, "extensions", name)); err == nil {
			t.Errorf("la actualización dejó la extensión %s", name)
		}
	}

	updatedOpencode := readFileAt(t, opencode)
	for _, fragment := range []string{"./otro/plugin", "// la configuración del proyecto", "// comentario final", `"model": "x"`} {
		if !strings.Contains(updatedOpencode, fragment) {
			t.Errorf("la configuración de OpenCode perdió %q: %s", fragment, updatedOpencode)
		}
	}

	if strings.Contains(updatedOpencode, "./.tasks-ai/extensions/opencode") {
		t.Errorf("la configuración de OpenCode conserva la referencia: %s", updatedOpencode)
	}

	var packages struct {
		Packages []string `json:"packages"`
	}

	if err := json.Unmarshal([]byte(readFileAt(t, pi)), &packages); err != nil {
		t.Fatalf("la configuración de Pi no es JSON válido: %v", err)
	}

	if !slices.Equal(packages.Packages, []string{"otro/paquete"}) {
		t.Errorf("los paquetes de Pi quedaron %v, se esperaba el ajeno y sin la extensión", packages.Packages)
	}

	// Repetir la actualización no cambia nada: cada paso compara antes de escribir.
	again, err := Upgrade(root)
	if err != nil {
		t.Fatalf("la segunda actualización devolvió %v", err)
	}

	if again.Version != core.Version || len(again.Pending) != 0 {
		t.Errorf("la segunda actualización quedó %+v, se esperaba completa y sin pendientes", again)
	}

	if again.Instructions != instructionsCurrent {
		t.Errorf("la segunda actualización vio las instrucciones en %q, se esperaba %q", again.Instructions, instructionsCurrent)
	}
}

// TestUpgrade_rewritesTheLegacyInstructionsWithTheProjectPrefix comprueba que la
// comparación byte a byte usa el prefijo del proyecto, no solo tk.
func TestUpgrade_rewritesTheLegacyInstructionsWithTheProjectPrefix(t *testing.T) {
	t.Parallel()

	root, product := legacyProject(t)

	cfg := settings.Default()
	cfg.Prefix = "px"

	if err := settings.Save(product, cfg); err != nil {
		t.Fatalf("guardando los ajustes: %v", err)
	}

	legacy := strings.ReplaceAll(legacyAgentTemplate, settings.DefaultPrefix+"-", "px-")
	writeFileAt(t, filepath.Join(product, instructionsName), legacy)

	report, err := Upgrade(root)
	if err != nil {
		t.Fatalf("Upgrade devolvió %v", err)
	}

	if report.Instructions != instructionsUpdated {
		t.Errorf("las instrucciones quedaron en %q, se esperaba %q", report.Instructions, instructionsUpdated)
	}

	written := readFileAt(t, filepath.Join(product, instructionsName))
	if !strings.Contains(written, "px-1") || strings.Contains(written, "tk-") {
		t.Error("las instrucciones regeneradas no usan el prefijo px")
	}
}

// TestUpgrade_leavesCustomisedInstructionsAlone comprueba que un texto propio no
// se pisa y se avisa como pendiente informativo, sin bloquear la actualización.
func TestUpgrade_leavesCustomisedInstructionsAlone(t *testing.T) {
	t.Parallel()

	root, product := legacyProject(t)

	mine := "# Mis instrucciones\n\nNo las toques.\n"
	writeFileAt(t, filepath.Join(product, instructionsName), mine)

	report, err := Upgrade(root)
	if err != nil {
		t.Fatalf("Upgrade devolvió %v", err)
	}

	if report.Instructions != instructionsPending {
		t.Errorf("las instrucciones quedaron en %q, se esperaba %q", report.Instructions, instructionsPending)
	}

	if len(report.Pending) != 0 {
		t.Errorf("las instrucciones propias bloquearon la actualización: %v", report.Pending)
	}

	if report.Version != core.Version {
		t.Errorf("la versión quedó %q, se esperaba %q", report.Version, core.Version)
	}

	if got := readFileAt(t, filepath.Join(product, instructionsName)); got != mine {
		t.Error("la actualización pisó las instrucciones propias")
	}
}

// TestUpgrade_finishesAnInterruptedUpgrade comprueba que una ejecución cortada a
// mitad se completa en la siguiente, sin VERSION y con una extensión ya retirada.
func TestUpgrade_finishesAnInterruptedUpgrade(t *testing.T) {
	t.Parallel()

	root, product := legacyProject(t)

	if err := os.RemoveAll(filepath.Join(product, "extensions", "pi")); err != nil {
		t.Fatalf("simulando la extensión ya retirada: %v", err)
	}

	writeFileAt(t, filepath.Join(root, ".pi", "settings.json"), "{\"packages\": []}\n")

	if _, err := os.Stat(filepath.Join(product, VersionName)); err == nil {
		t.Fatal("la prueba preparó una instalación con VERSION")
	}

	report, err := Upgrade(root)
	if err != nil {
		t.Fatalf("Upgrade devolvió %v", err)
	}

	if report.Version != core.Version || len(report.Pending) != 0 {
		t.Errorf("la reejecución quedó %+v, se esperaba completa", report)
	}

	if _, err := os.Stat(filepath.Join(product, "extensions", "opencode")); err == nil {
		t.Error("la reejecución no retiró la extensión que quedaba")
	}

	if got := readFileAt(t, filepath.Join(product, VersionName)); got != core.Version+"\n" {
		t.Errorf("la versión quedó %q, se esperaba %q", got, core.Version)
	}

	if got := readFileAt(t, filepath.Join(root, "opencode.jsonc")); got != `{"plugins": []}`+"\n" {
		t.Errorf("el array sin elementos quedó %q, se esperaba uno válido y vacío", got)
	}
}

// TestUpgrade_refusesADowngrade comprueba que un binario más viejo que la
// instalación no escribe nada.
func TestUpgrade_refusesADowngrade(t *testing.T) {
	t.Parallel()

	root, product := legacyProject(t)

	writeFileAt(t, filepath.Join(product, VersionName), "0.2.1\n")

	legacy := readFileAt(t, filepath.Join(product, instructionsName))
	opencode := readFileAt(t, filepath.Join(root, "opencode.jsonc"))

	_, err := Upgrade(root)
	if !errors.Is(err, ErrUpgradeRefused) {
		t.Fatalf("Upgrade devolvió %v, se esperaba un rechazo", err)
	}

	if !strings.Contains(err.Error(), "0.2.1") || !strings.Contains(err.Error(), core.Version) {
		t.Errorf("el rechazo no nombra las dos versiones: %v", err)
	}

	if got := readFileAt(t, filepath.Join(product, VersionName)); got != "0.2.1\n" {
		t.Errorf("el rechazo reescribió la versión: %q", got)
	}

	if got := readFileAt(t, filepath.Join(product, instructionsName)); got != legacy {
		t.Error("el rechazo reescribió las instrucciones")
	}

	if got := readFileAt(t, filepath.Join(root, "opencode.jsonc")); got != opencode {
		t.Error("el rechazo reescribió la configuración de OpenCode")
	}

	for _, name := range []string{"pi", "opencode"} {
		if _, err := os.Stat(filepath.Join(product, "extensions", name)); err != nil {
			t.Errorf("el rechazo retiró la extensión %s: %v", name, err)
		}
	}
}

// TestUpgrade_keepsTheHostReferenceWithACommentInside comprueba que un comentario
// dentro del array impide editar el fichero, conserva la extensión y deja la
// actualización pendiente.
func TestUpgrade_keepsTheHostReferenceWithACommentInside(t *testing.T) {
	t.Parallel()

	root, product := legacyProject(t)

	opencode := filepath.Join(root, "opencode.jsonc")
	original := `{
  "plugins": [
    "./otro/plugin", // el otro
    "./.tasks-ai/extensions/opencode"
  ]
}
`
	writeFileAt(t, opencode, original)

	report, err := Upgrade(root)
	if err != nil {
		t.Fatalf("Upgrade devolvió %v", err)
	}

	if report.Version != "" {
		t.Errorf("la actualización escribió la versión %q con una referencia viva", report.Version)
	}

	if len(report.Pending) != 1 {
		t.Fatalf("los pendientes son %v, se esperaba la referencia de OpenCode", report.Pending)
	}

	if got := readFileAt(t, opencode); got != original {
		t.Errorf("la actualización reescribió el fichero con un comentario dentro del array: %s", got)
	}

	if _, err := os.Stat(filepath.Join(product, "extensions", "opencode")); err != nil {
		t.Errorf("la actualización retiró la extensión con una referencia viva: %v", err)
	}

	// La de Pi no tenía referencia viva y se retira sin esperar a la otra.
	if _, err := os.Stat(filepath.Join(product, "extensions", "pi")); err == nil {
		t.Error("la actualización conservó la extensión de Pi sin referencia")
	}
}

// TestUpgrade_keepsTheHostReferenceWithABlockCommentInside es el mismo límite
// con un comentario de bloque: el fichero no se toca.
func TestUpgrade_keepsTheHostReferenceWithABlockCommentInside(t *testing.T) {
	t.Parallel()

	root, product := legacyProject(t)

	opencode := filepath.Join(root, "opencode.jsonc")
	original := "{\n  \"plugins\": [/* el otro */ \"./.tasks-ai/extensions/opencode\"]\n}\n"
	writeFileAt(t, opencode, original)

	report, err := Upgrade(root)
	if err != nil {
		t.Fatalf("Upgrade devolvió %v", err)
	}

	if report.Version != "" || len(report.Pending) != 1 {
		t.Errorf("la actualización quedó %+v, se esperaba pendiente de la referencia", report)
	}

	if got := readFileAt(t, opencode); got != original {
		t.Errorf("la actualización reescribió el fichero con un comentario de bloque: %s", got)
	}

	if _, err := os.Stat(filepath.Join(product, "extensions", "opencode")); err != nil {
		t.Errorf("la actualización retiró la extensión con una referencia viva: %v", err)
	}
}

// TestUpgrade_doesNotRemoveASimilarEntry comprueba que una cadena parecida no es
// la entrada: no se retira del fichero.
func TestUpgrade_doesNotRemoveASimilarEntry(t *testing.T) {
	t.Parallel()

	root, product := legacyProject(t)

	opencodeValue := `{"plugins": ["./.tasks-ai/extensions/opencode-old"]}` + "\n"
	piValue := `{"packages": ["../.tasks-ai/extensions/pi-old"]}` + "\n"

	writeFileAt(t, filepath.Join(root, "opencode.jsonc"), opencodeValue)
	writeFileAt(t, filepath.Join(root, ".pi", "settings.json"), piValue)

	report, err := Upgrade(root)
	if err != nil {
		t.Fatalf("Upgrade devolvió %v", err)
	}

	if len(report.Pending) != 0 || report.Version != core.Version {
		t.Errorf("la actualización quedó %+v, se esperaba completa", report)
	}

	if got := readFileAt(t, filepath.Join(root, "opencode.jsonc")); got != opencodeValue {
		t.Errorf("una entrada parecida se retiró: %s", got)
	}

	if got := readFileAt(t, filepath.Join(root, ".pi", "settings.json")); got != piValue {
		t.Errorf("una entrada parecida se retiró de Pi: %s", got)
	}

	// Sin referencia exacta viva, las carpetas de las extensiones se retiran.
	for _, name := range []string{"pi", "opencode"} {
		if _, err := os.Stat(filepath.Join(product, "extensions", name)); err == nil {
			t.Errorf("la actualización conservó la extensión %s sin referencia exacta", name)
		}
	}
}

// TestUpgrade_saysWhenTheProductIsNotInstalled comprueba el rechazo de uso.
func TestUpgrade_saysWhenTheProductIsNotInstalled(t *testing.T) {
	t.Parallel()

	if _, err := Upgrade(t.TempDir()); !errors.Is(err, ErrUpgradeRefused) {
		t.Fatalf("Upgrade devolvió %v, se esperaba un rechazo", err)
	}
}

// TestCompareVersions ordena versiones por sus números.
func TestCompareVersions(t *testing.T) {
	t.Parallel()

	cases := map[string]int{
		"0.1.0|0.2.0":  -1,
		"0.2.0|0.1.0":  1,
		"0.2.0|0.2.0":  0,
		"0.10.0|0.9.0": 1,
		"1.0|1.0.0":    0,
	}

	for pair, want := range cases {
		left, right, _ := strings.Cut(pair, "|")
		if got := compareVersions(left, right); got != want {
			t.Errorf("compareVersions(%q, %q) = %d, se esperaba %d", left, right, got, want)
		}
	}
}

// legacyProject prepares a v0.1.0 installation: data, settings, an old binary,
// both extension folders, the legacy instructions, the host references and no
// VERSION. It returns the project root and the product folder.
func legacyProject(t *testing.T) (root, product string) {
	t.Helper()

	root = t.TempDir()
	product = filepath.Join(root, DirName)

	for _, dir := range []string{"bin", filepath.Join("extensions", "pi"), filepath.Join("extensions", "opencode")} {
		writeDirAt(t, filepath.Join(product, dir))
	}

	if err := settings.Save(product, settings.Default()); err != nil {
		t.Fatalf("preparando los ajustes: %v", err)
	}

	writeFileAt(t, filepath.Join(product, "tasks.db"), "las tareas del proyecto")
	writeFileAt(t, filepath.Join(product, "bin", "tasks-ai"), "binario viejo")
	writeFileAt(t, filepath.Join(product, "extensions", "pi", "tsconfig.json"), shippedExtensionFile)
	writeFileAt(t, filepath.Join(product, "extensions", "opencode", "tsconfig.json"), shippedExtensionFile)
	writeFileAt(t, filepath.Join(product, instructionsName), legacyAgentTemplate)
	writeFileAt(t, filepath.Join(root, "opencode.jsonc"), `{"plugins": ["./.tasks-ai/extensions/opencode"]}`+"\n")

	writeDirAt(t, filepath.Join(root, ".pi"))
	writeFileAt(t, filepath.Join(root, ".pi", "settings.json"), `{"packages": ["../.tasks-ai/extensions/pi"]}`+"\n")

	return root, product
}

func writeDirAt(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o755); err != nil { //nolint:gosec // la carpeta de prueba usa los permisos del producto real
		t.Fatalf("creando %s: %v", path, err)
	}
}

func writeFileAt(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { //nolint:gosec // el fichero de prueba usa los permisos del producto real
		t.Fatalf("escribiendo %s: %v", path, err)
	}
}

func readFileAt(t *testing.T, path string) string {
	t.Helper()

	raw, err := os.ReadFile(path) //nolint:gosec // la ruta la prepara la prueba
	if err != nil {
		t.Fatalf("leyendo %s: %v", path, err)
	}

	return string(raw)
}

// TestLegacyAgentTemplateIsTheV010Instructions comprueba que la constante de
// migración conserva la plantilla tal cual, sin marcador de versión.
func TestLegacyAgentTemplateIsTheV010Instructions(t *testing.T) {
	t.Parallel()

	if !bytes.Contains([]byte(legacyAgentTemplate), []byte("tasks-ai create --title")) {
		t.Error("la plantilla v0.1.0 no es la que llevaba la tabla del CLI")
	}

	if bytes.Contains(instructions, []byte("tasks-ai create --title")) {
		t.Error("la plantilla nueva conserva la tabla de comandos del CLI")
	}
}

// TestUpgrade_leavesACustomContractAlone aplica el criterio del AGENTS.md al
// contrato: si el proyecto lo editó, el producto no lo pisa y termina igual,
// avisando para que una persona lo revise.
func TestUpgrade_leavesACustomContractAlone(t *testing.T) {
	t.Parallel()

	root, product := legacyProject(t)

	const mine = "// el contrato editado por el proyecto\n"
	writeFileAt(t, filepath.Join(product, contractName), mine)

	report, err := Upgrade(root)
	if err != nil {
		t.Fatalf("Upgrade devolvió %v: %v", err, report)
	}

	if report.Contract != instructionsPending {
		t.Errorf("el contrato quedó en %q, se esperaba %q", report.Contract, instructionsPending)
	}

	if len(report.Pending) != 0 {
		t.Errorf("un contrato propio no es pendiente bloqueante: %v", report.Pending)
	}

	if got := readFileAt(t, filepath.Join(product, contractName)); got != mine {
		t.Errorf("la actualización pisó el contrato editado: %q", got)
	}

	if got := readFileAt(t, filepath.Join(product, VersionName)); got != core.Version+"\n" {
		t.Errorf("la instalación no quedó completa: VERSION %q", got)
	}
}

// TestUpgrade_keepsFilesThatAreNotItsOwn en las extensiones: los ficheros
// originales de la v0.1.0 se retiran por su nombre, lo demás permanece y los
// restos no impiden completar la actualización.
func TestUpgrade_keepsFilesThatAreNotItsOwn(t *testing.T) {
	t.Parallel()

	root, product := legacyProject(t)

	const mine = "el apunte propio del proyecto\n"
	writeFileAt(t, filepath.Join(product, "extensions", "pi", "mi-apunte.txt"), mine)
	writeDirAt(t, filepath.Join(product, "extensions", "opencode", "libro-de-cuentas"))
	writeFileAt(t, filepath.Join(product, "extensions", "opencode", "libro-de-cuentas", "2026.md"), "…")

	report, err := Upgrade(root)
	if err != nil {
		t.Fatalf("Upgrade devolvió %v", err)
	}

	if len(report.Pending) != 0 {
		t.Errorf("los restos propios no son pendiente bloqueante: %v", report.Pending)
	}

	if got := readFileAt(t, filepath.Join(product, "extensions", "pi", "mi-apunte.txt")); got != mine {
		t.Errorf("la actualización borró el fichero propio: %q", got)
	}

	if _, err := os.Stat(filepath.Join(product, "extensions", "opencode", "libro-de-cuentas", "2026.md")); err != nil {
		t.Errorf("la actualización borró contenido propio: %v", err)
	}

	if _, err := os.Stat(filepath.Join(product, "extensions", "pi", "index.ts")); err == nil {
		t.Error("el fichero original de la extensión no se retiró")
	}

	if _, err := os.Stat(filepath.Join(product, "extensions", "opencode", "index.ts")); err == nil {
		t.Error("el fichero original de la extensión no se retiró")
	}

	if got := readFileAt(t, filepath.Join(product, VersionName)); got != core.Version+"\n" {
		t.Errorf("la instalación no quedó completa: VERSION %q", got)
	}
}

// shippedExtensionFile es el contenido exacto con que la v0.1.0 entregó el
// tsconfig.json de las dos extensiones: lo usan los fixtures que simulan una
// instalación intacta, tal como su hash los acredita.
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

// TestUpgrade_keepsEditedExtensionFiles comprueba el hash: un fichero con
// nombre original y contenido distinto es un fichero del proyecto. Se
// conserva, se informa y la actualización queda completa igual.
func TestUpgrade_keepsEditedExtensionFiles(t *testing.T) {
	t.Parallel()

	root, product := legacyProject(t)

	const edited = "// el index.ts que el proyecto modificó\n"
	writeFileAt(t, filepath.Join(product, "extensions", "pi", "index.ts"), edited)

	report, err := Upgrade(root)
	if err != nil {
		t.Fatalf("Upgrade devolvió %v", err)
	}

	if len(report.Pending) != 0 || report.Version != core.Version {
		t.Errorf("la actualización quedó %+v, se esperaba completa", report)
	}

	if got := readFileAt(t, filepath.Join(product, "extensions", "pi", "index.ts")); got != edited {
		t.Errorf("la actualización borró el fichero editado: %q", got)
	}

	// El tsconfig, con su contenido exacto, sí se retiró; el informe nombra
	// lo conservado.
	if _, err := os.Stat(filepath.Join(product, "extensions", "pi", "tsconfig.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("el fichero con el contenido distribuido no se retiró: %v", err)
	}

	found := slices.ContainsFunc(report.Steps, func(step string) bool {
		return strings.Contains(step, "extension pi") && strings.Contains(step, "keeps 1 by the project")
	})
	if !found {
		t.Errorf("el informe omite el fichero editado que quedó: %v", report.Steps)
	}

	if got := readFileAt(t, filepath.Join(product, VersionName)); got != core.Version+"\n" {
		t.Errorf("la instalación no quedó completa: VERSION %q", got)
	}
}
