package install_test

import (
	"os"
	"path/filepath"
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

	result, err := install.Run(t.Context(), dir)
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

func TestRun_refusesToInstallOverAnInstallation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	if _, err := install.Run(t.Context(), dir); err != nil {
		t.Fatalf("la primera instalación devolvió %v", err)
	}

	if _, err := install.Run(t.Context(), dir); err == nil {
		t.Error("Run volvió a instalar sobre un proyecto que ya lo tenía")
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

	result, err := install.Run(t.Context(), dir)
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

func mkdir(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o755); err != nil { //nolint:gosec // la carpeta de prueba usa los permisos del producto real
		t.Fatalf("creando %s: %v", path, err)
	}
}
