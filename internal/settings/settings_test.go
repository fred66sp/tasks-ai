package settings_test

import (
	"os"
	"path/filepath"
	"strings"
	"tasks-ai/internal/settings"
	"testing"
)

func TestLoad_usesTheDefaultsWhenThereIsNoFile(t *testing.T) {
	t.Parallel()

	cfg, err := settings.Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load devolvió %v", err)
	}

	if cfg != settings.Default() {
		t.Errorf("los ajustes de un proyecto sin archivo son %+v, se esperaban los de omisión", cfg)
	}
}

func TestLoad_keepsTheDefaultsForWhatTheFileLeavesOut(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	write(t, dir, `{"prefix": "px"}`)

	cfg, err := settings.Load(dir)
	if err != nil {
		t.Fatalf("Load devolvió %v", err)
	}

	if cfg.Prefix != "px" {
		t.Errorf("el prefijo es %q, se esperaba %q", cfg.Prefix, "px")
	}

	if cfg.HTTP.Port != settings.DefaultHTTPPort || cfg.DB != settings.DefaultDB {
		t.Errorf("lo que el archivo no dice debe quedarse en su valor por omisión: %+v", cfg)
	}
}

func TestLoad_refusesSettingsThatWouldBreakTheProduct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		file string
	}{
		{name: "un archivo que no es json", file: `no soy json`},
		{name: "un prefijo con espacios", file: `{"prefix": "dos palabras"}`},
		{name: "un prefijo vacío", file: `{"prefix": ""}`},
		{name: "una base con ruta absoluta", file: `{"db": "/tmp/tasks.db"}`},
		{name: "una base que se sale de la carpeta", file: `{"db": "../tasks.db"}`},
		{name: "una base sin nombre", file: `{"db": ""}`},
		{name: "un puerto fuera del rango", file: `{"http": {"host": "127.0.0.1", "port": 70000}}`},
		{name: "un nivel de registro inventado", file: `{"log": {"level": "verboso"}}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			write(t, dir, test.file)

			if _, err := settings.Load(dir); err == nil {
				t.Errorf("Load aceptó %s", test.file)
			}
		})
	}
}

func TestSave_leavesASettingsFileThatLoadReads(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := settings.Default()
	cfg.Prefix = "px"

	if err := settings.Save(dir, cfg); err != nil {
		t.Fatalf("Save devolvió %v", err)
	}

	read, err := settings.Load(dir)
	if err != nil {
		t.Fatalf("Load devolvió %v", err)
	}

	if read != cfg {
		t.Errorf("lo guardado se lee como %+v, se esperaba %+v", read, cfg)
	}

	raw, err := os.ReadFile(filepath.Join(dir, settings.FileName)) //nolint:gosec // la ruta es la carpeta temporal de la prueba
	if err != nil {
		t.Fatalf("leyendo el archivo escrito: %v", err)
	}

	if !strings.HasSuffix(string(raw), "\n") || !strings.Contains(string(raw), "\n  \"prefix\"") {
		t.Errorf("el archivo escrito no está pensado para leerlo:\n%s", raw)
	}
}

func TestExists_saysWhetherTheFolderHasSettings(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if settings.Exists(dir) {
		t.Error("Exists dijo que hay ajustes donde no los hay")
	}

	if err := settings.Save(dir, settings.Default()); err != nil {
		t.Fatalf("Save devolvió %v", err)
	}

	if !settings.Exists(dir) {
		t.Error("Exists no vio los ajustes recién escritos")
	}
}

func write(t *testing.T, dir, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, settings.FileName), []byte(content), 0o644); err != nil { //nolint:gosec // el archivo de prueba se escribe con los permisos del producto real
		t.Fatalf("escribiendo los ajustes de prueba: %v", err)
	}
}
