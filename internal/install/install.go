// Package install puts the product into a project: its folder with the base, the
// settings and the instructions, and the text a project can add outside to point
// at them.
package install

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"tasks-ai/internal/core"
	"tasks-ai/internal/settings"
	"tasks-ai/internal/store"
)

// DirName is the folder the product keeps inside a project.
const DirName = ".task-ai"

// instructionsName is the file with the instructions for people and agents.
const instructionsName = "AGENTS.md"

// pointer is the text a project adds outside to point at the instructions.
const pointer = "Para que las personas y los agentes encuentren las instrucciones del gestor de tareas, " +
	"añade esta línea al archivo de instrucciones de tu proyecto: @" + DirName + "/" + instructionsName

//go:embed agent.md
var instructions []byte

// Result is what an installation left behind.
type Result struct {
	Dir          string `json:"dir"`
	DB           string `json:"db"`
	Instructions string `json:"instructions"`
	Prefix       string `json:"prefix"`
	Pointer      string `json:"pointer"`
}

// Root is the root of the project the folder dir belongs to: the enclosing
// repository when there is one, and the folder itself when there is not.
func Root(dir string) string {
	current := filepath.Clean(dir)

	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current
		}

		parent := filepath.Dir(current)
		if parent == current {
			return filepath.Clean(dir)
		}

		current = parent
	}
}

// Project returns the project the product is installed in, looking in the folder
// dir and in the ones above it.
func Project(dir string) (string, error) {
	current := filepath.Clean(dir)

	for {
		if info, err := os.Stat(filepath.Join(current, DirName)); err == nil && info.IsDir() {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("no hay ningún proyecto con el producto instalado en %s ni en las carpetas que lo contienen", dir)
		}

		current = parent
	}
}

// Run installs the product in the project rooted at root, completing what a
// previous attempt left half done. What is already there is left alone, so the
// settings keep the values the project put in them.
func Run(ctx context.Context, root string) (Result, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Result{}, fmt.Errorf("resolviendo la ruta del proyecto %s: %w", root, err)
	}

	dir := filepath.Join(root, DirName)

	cfg, err := prepare(dir)
	if err != nil {
		return Result{}, err
	}

	dbPath := settings.DatabasePath(dir, cfg)
	instructions := filepath.Join(dir, instructionsName)

	if exists(dbPath) && exists(instructions) {
		return Result{}, fmt.Errorf("el proyecto ya tiene el producto instalado en %s", dir)
	}

	// Las instrucciones van antes que la base: si algo falla a mitad, lo que ya
	// está hecho se puede volver a completar sin borrar nada.
	if err := writeInstructions(dir); err != nil {
		return Result{}, err
	}

	base, err := store.Open(ctx, dbPath)
	if err != nil {
		return Result{}, err
	}

	defer func() { _ = base.Close() }()

	if _, err := core.New(base, cfg.Prefix).CheckPrefix(ctx); err != nil {
		return Result{}, err
	}

	return Result{
		Dir:          dir,
		DB:           dbPath,
		Instructions: instructions,
		Prefix:       cfg.Prefix,
		Pointer:      pointer,
	}, nil
}

// exists says whether there is a file at path.
func exists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}

// prepare creates the product folder and leaves a settings file in it, without
// touching the one the project may have written itself.
func prepare(dir string) (settings.Settings, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // la carpeta del producto usa los permisos propios del proyecto
		return settings.Settings{}, fmt.Errorf("creando la carpeta %s: %w", dir, err)
	}

	if !settings.Exists(dir) {
		if err := settings.Save(dir, settings.Default()); err != nil {
			return settings.Settings{}, err
		}
	}

	return settings.Load(dir)
}

// writeInstructions leaves the instructions in the product folder, without
// treading on the ones that are already there.
func writeInstructions(dir string) error {
	path := filepath.Join(dir, instructionsName)

	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("mirando las instrucciones de %s: %w", dir, err)
	}

	if err := os.WriteFile(path, instructions, 0o644); err != nil { //nolint:gosec // las instrucciones se escriben con los permisos de lectura del proyecto
		return fmt.Errorf("escribiendo %s: %w", path, err)
	}

	return nil
}
