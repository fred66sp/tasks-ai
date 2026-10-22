package install_test

import (
	"os"
	"path/filepath"
	"strings"
	"tasks-ai/internal/core"
	"tasks-ai/internal/install"
	"tasks-ai/internal/settings"
	"tasks-ai/internal/store"
	"testing"
)

func TestRoot_staysInTheRepositoryTheFolderBelongsTo(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mkdir(t, filepath.Join(dir, ".git"))

	deep := filepath.Join(dir, "uno", "dos")
	mkdir(t, deep)

	if got := install.Root(deep); got != dir {
		t.Errorf("Root(%q) = %q, se esperaba %q", deep, got, dir)
	}
}

func TestRoot_findsARepositoryWhoseGitEntryIsAFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: outside"), 0o600); err != nil {
		t.Fatalf("preparando el .git como fichero: %v", err)
	}

	deep := filepath.Join(dir, "uno", "dos")
	mkdir(t, deep)

	if got := install.Root(deep); got != dir {
		t.Errorf("Root(%q) = %q, se esperaba %q", deep, got, dir)
	}
}

func TestRoot_fallsBackToTheFolderItWasGiven(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	deep := filepath.Join(dir, "uno")
	mkdir(t, deep)

	if got := install.Root(deep); got != deep {
		t.Errorf("Root(%q) = %q, se esperaba la carpeta dada", deep, got)
	}
}

func TestProject_findsTheInstalledFolderAbove(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mkdir(t, filepath.Join(dir, install.DirName))

	deep := filepath.Join(dir, "uno", "dos")
	mkdir(t, deep)

	got, err := install.Project(deep)
	if err != nil {
		t.Fatalf("Project devolvió %v", err)
	}

	if got != dir {
		t.Errorf("Project(%q) = %q, se esperaba %q", deep, got, dir)
	}
}

func TestProject_saysWhenTheProductIsNotInstalled(t *testing.T) {
	t.Parallel()

	if _, err := install.Project(t.TempDir()); err == nil {
		t.Error("Project aceptó una carpeta sin el producto instalado")
	}
}

func TestRun_leavesItsFolderWithTheBaseTheSettingsAndTheInstructions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	result, err := install.Run(t.Context(), dir, "")
	if err != nil {
		t.Fatalf("Run devolvió %v", err)
	}

	product := filepath.Join(dir, install.DirName)
	if result.Dir != product {
		t.Errorf("la instalación quedó en %q, se esperaba %q", result.Dir, product)
	}

	if result.Prefix != settings.DefaultPrefix || result.Pointer == "" {
		t.Errorf("la instalación devolvió %+v, se esperaba el prefijo por omisión y el texto de fuera", result)
	}

	for _, name := range []string{"tasks.db", "settings.json", "AGENTS.md"} {
		if _, err := os.Stat(filepath.Join(product, name)); err != nil {
			t.Errorf("la instalación no dejó %s: %v", name, err)
		}
	}

	if _, err := os.Stat(result.Instructions); err != nil {
		t.Errorf("las instrucciones que anuncia la instalación no están: %v", err)
	}
}

func TestRun_leavesTheBinaryAndTheExtension(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	result, err := install.Run(t.Context(), dir, "")
	if err != nil {
		t.Fatalf("Run devolvió %v", err)
	}

	product := filepath.Join(dir, install.DirName)

	bin := filepath.Join(product, "bin", "tasks-ai")

	info, err := os.Stat(bin)
	if err != nil {
		t.Fatalf("la instalación no dejó el binario: %v", err)
	}

	if info.Mode()&0o111 == 0 {
		t.Errorf("el binario instalado no es ejecutable: %v", info.Mode())
	}

	if result.Binary != bin {
		t.Errorf("la instalación devolvió el binario %q, se esperaba %q", result.Binary, bin)
	}

	extension := filepath.Join(product, "extensions", "pi")
	if result.Extension != extension {
		t.Errorf("la instalación devolvió la extensión %q, se esperaba %q", result.Extension, extension)
	}

	for _, name := range []string{"index.ts", "board.ts", "cli.ts", "package.json"} {
		if _, err := os.Stat(filepath.Join(extension, name)); err != nil {
			t.Errorf("la instalación no dejó la extensión %s: %v", name, err)
		}
	}
}

func TestRun_refusesToInstallOverAnInstallation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	if _, err := install.Run(t.Context(), dir, ""); err != nil {
		t.Fatalf("la primera instalación devolvió %v", err)
	}

	if _, err := install.Run(t.Context(), dir, ""); err == nil {
		t.Error("Run volvió a instalar sobre un proyecto que ya lo tenía")
	}
}

func TestRun_completesAnInstallationThatLostItsInstructions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	if _, err := install.Run(t.Context(), dir, ""); err != nil {
		t.Fatalf("la primera instalación devolvió %v", err)
	}

	instructions := filepath.Join(dir, install.DirName, "AGENTS.md")
	if err := os.Remove(instructions); err != nil {
		t.Fatalf("quitando las instrucciones: %v", err)
	}

	if _, err := install.Run(t.Context(), dir, ""); err != nil {
		t.Fatalf("volver a instalar sobre una instalación incompleta devolvió %v", err)
	}

	if _, err := os.Stat(instructions); err != nil {
		t.Errorf("volver a instalar no devolvió las instrucciones: %v", err)
	}
}

func TestRun_completesAnInstallationOverABaseComposedWithAnotherPrefix(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	product := filepath.Join(dir, install.DirName)

	if _, err := install.Run(t.Context(), dir, "px"); err != nil {
		t.Fatalf("la primera instalación devolvió %v", err)
	}

	// La instalación queda incompleta y la base se queda con el prefijo con el
	// que se compuso.
	for _, name := range []string{settings.FileName, "AGENTS.md"} {
		if err := os.Remove(filepath.Join(product, name)); err != nil {
			t.Fatalf("quitando %s: %v", name, err)
		}
	}

	result, err := install.Run(t.Context(), dir, "")
	if err != nil {
		t.Fatalf("completar la instalación devolvió %v", err)
	}

	if result.Prefix != settings.DefaultPrefix {
		t.Errorf("la instalación quedó con el prefijo %q, se esperaba el de los ajustes", result.Prefix)
	}

	cfg, err := settings.Load(product)
	if err != nil {
		t.Fatalf("Load devolvió %v", err)
	}

	base, err := store.Open(t.Context(), settings.DatabasePath(product, cfg))
	if err != nil {
		t.Fatalf("abriendo la base instalada: %v", err)
	}

	t.Cleanup(func() { _ = base.Close() })

	recorded, err := base.RecordedPrefix(t.Context())
	if err != nil {
		t.Fatalf("RecordedPrefix devolvió %v", err)
	}

	if recorded != "px" {
		t.Errorf("la instalación reescribió el prefijo de la base, quedó en %q y debía seguir en px", recorded)
	}

	// El desajuste sigue siendo real después de completar: eso es lo que init
	// tiene que avisar por la salida de errores.
	note, err := core.New(base, cfg.Prefix).CheckPrefix(t.Context())
	if err != nil {
		t.Fatalf("CheckPrefix devolvió %v", err)
	}

	if note == "" {
		t.Error("completar la instalación se cargó el desajuste de prefijo que había que avisar")
	}
}

func TestRun_keepsTheSettingsThatAreAlreadyThere(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	product := filepath.Join(dir, install.DirName)
	mkdir(t, product)

	cfg := settings.Default()
	cfg.Prefix = "px"

	if err := settings.Save(product, cfg); err != nil {
		t.Fatalf("preparando los ajustes del proyecto: %v", err)
	}

	result, err := install.Run(t.Context(), dir, "")
	if err != nil {
		t.Fatalf("Run devolvió %v", err)
	}

	if result.Prefix != "px" {
		t.Errorf("la instalación se quedó con el prefijo %q, se esperaba el que el proyecto había puesto", result.Prefix)
	}

	read, err := settings.Load(product)
	if err != nil {
		t.Fatalf("Load devolvió %v", err)
	}

	if read != cfg {
		t.Errorf("la instalación reescribió los ajustes: %+v frente a %+v", read, cfg)
	}

	base, err := store.Open(t.Context(), settings.DatabasePath(product, read))
	if err != nil {
		t.Fatalf("abriendo la base instalada: %v", err)
	}

	t.Cleanup(func() { _ = base.Close() })

	recorded, err := base.RecordedPrefix(t.Context())
	if err != nil {
		t.Fatalf("RecordedPrefix devolvió %v", err)
	}

	if recorded != "px" {
		t.Errorf("la base se anotó con el prefijo %q, se esperaba el de los ajustes", recorded)
	}
}

func TestRun_installsWithThePrefixTheProjectAsks(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	result, err := install.Run(t.Context(), dir, "tai")
	if err != nil {
		t.Fatalf("Run devolvió %v", err)
	}

	if result.Prefix != "tai" {
		t.Errorf("la instalación devolvió el prefijo %q, se esperaba tai", result.Prefix)
	}

	product := filepath.Join(dir, install.DirName)

	cfg, err := settings.Load(product)
	if err != nil {
		t.Fatalf("Load devolvió %v", err)
	}

	if cfg.Prefix != "tai" {
		t.Errorf("los ajustes guardan el prefijo %q, se esperaba tai", cfg.Prefix)
	}

	written, err := os.ReadFile(result.Instructions)
	if err != nil {
		t.Fatalf("leyendo las instrucciones: %v", err)
	}

	if !strings.Contains(string(written), "tai-1") || strings.Contains(string(written), "tk-") {
		t.Errorf("las instrucciones no componen sus ejemplos con el prefijo del proyecto")
	}

	base, err := store.Open(t.Context(), settings.DatabasePath(product, cfg))
	if err != nil {
		t.Fatalf("abriendo la base instalada: %v", err)
	}

	t.Cleanup(func() { _ = base.Close() })

	recorded, err := base.RecordedPrefix(t.Context())
	if err != nil {
		t.Fatalf("RecordedPrefix devolvió %v", err)
	}

	if recorded != "tai" {
		t.Errorf("la base se anotó con el prefijo %q, se esperaba tai", recorded)
	}

	note, err := core.New(base, cfg.Prefix).CheckPrefix(t.Context())
	if err != nil {
		t.Fatalf("CheckPrefix devolvió %v", err)
	}

	if note != "" {
		t.Errorf("la instalación dejó un aviso para los comandos siguientes: %q", note)
	}
}

func TestRun_refusesAPrefixThatTheProjectSettingsDoNotHave(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	product := filepath.Join(dir, install.DirName)
	mkdir(t, product)

	cfg := settings.Default()
	cfg.Prefix = "px"

	if err := settings.Save(product, cfg); err != nil {
		t.Fatalf("preparando los ajustes del proyecto: %v", err)
	}

	if _, err := install.Run(t.Context(), dir, "tai"); err == nil {
		t.Error("Run aceptó un prefijo distinto del que el proyecto ya tenía en sus ajustes")
	}

	result, err := install.Run(t.Context(), dir, "px")
	if err != nil {
		t.Fatalf("Run rechazó el prefijo que el proyecto ya tenía: %v", err)
	}

	if result.Prefix != "px" {
		t.Errorf("la instalación devolvió el prefijo %q, se esperaba px", result.Prefix)
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o755); err != nil { //nolint:gosec // la carpeta de prueba usa los permisos del producto real
		t.Fatalf("creando %s: %v", path, err)
	}
}
