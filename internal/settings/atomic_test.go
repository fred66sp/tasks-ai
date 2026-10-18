package settings

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAtomically_keepsThePreviousFileWhenReplacementFails(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), FileName)
	previous := []byte("{\"prefix\":\"old\"}\n")

	if err := os.WriteFile(path, previous, 0o600); err != nil {
		t.Fatalf("preparando los ajustes anteriores: %v", err)
	}

	err := writeAtomically(path, []byte("{\"prefix\":\"new\"}\n"), func(string, string) error {
		return errors.New("replacement failed")
	})
	if err == nil {
		t.Fatal("writeAtomically terminó bien cuando la sustitución falló")
	}

	got, err := os.ReadFile(path) //nolint:gosec // la ruta es la carpeta temporal de la prueba
	if err != nil {
		t.Fatalf("leyendo los ajustes anteriores: %v", err)
	}

	if string(got) != string(previous) {
		t.Errorf("la sustitución fallida dejó %q, se esperaba conservar %q", got, previous)
	}
}

func TestWriteAtomically_preservesTheExistingFilePermissions(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), FileName)
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatalf("preparando los ajustes protegidos: %v", err)
	}

	if err := writeAtomically(path, []byte("new"), os.Rename); err != nil {
		t.Fatalf("sustituyendo los ajustes: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("leyendo los permisos de los ajustes: %v", err)
	}

	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("la sustitución cambió los permisos a %o, se esperaban 600", got)
	}
}
