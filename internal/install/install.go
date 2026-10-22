// Package install puts the product into a project: its folder with the base, the
// settings and the instructions, and the text a project can add outside to point
// at them.
package install

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	embedded "tasks-ai"
	"tasks-ai/internal/core"
	"tasks-ai/internal/settings"
	"tasks-ai/internal/store"
)

// DirName is the folder the product keeps inside a project.
const DirName = ".tasks-ai"

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
	Binary       string `json:"binary"`
	Extension    string `json:"extension"`
}

// Root is the root of the project the folder dir belongs to: the enclosing
// repository when there is one, and the folder itself when there is not.
func Root(dir string) string {
	cleaned := filepath.Clean(dir)
	if root, found := walkUp(cleaned, func(current string) bool {
		_, err := os.Stat(filepath.Join(current, ".git"))

		return err == nil
	}); found {
		return root
	}

	return cleaned
}

// Project returns the project the product is installed in, looking in the folder
// dir and in the ones above it.
func Project(dir string) (string, error) {
	if root, found := walkUp(dir, func(current string) bool {
		info, err := os.Stat(filepath.Join(current, DirName))

		return err == nil && info.IsDir()
	}); found {
		return root, nil
	}

	return "", fmt.Errorf("there is no project with the product installed in %s or in the folders that contain it", dir)
}

// walkUp visits dir and each of its parents until a predicate accepts one or the
// filesystem root is reached. Both callers share the same boundary traversal;
// their predicates keep the repository and installation rules separate.
func walkUp(dir string, found func(string) bool) (string, bool) {
	current := filepath.Clean(dir)

	for {
		if found(current) {
			return current, true
		}

		parent := filepath.Dir(current)
		if parent == current {
			return filepath.Clean(dir), false
		}

		current = parent
	}
}

// Run installs the product in the project rooted at root, with the prefix the
// project asks for and completing what a previous attempt left half done. What
// is already there is left alone, so the settings keep the values the project
// put in them: asking for another prefix is refused instead of overwritten.
func Run(ctx context.Context, root, prefix string) (Result, error) {
	return RunWithWarning(ctx, root, prefix, os.Stderr)
}

// RunWithWarning installs the product and writes any prefix warning to warning.
func RunWithWarning(ctx context.Context, root, prefix string, warning io.Writer) (Result, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Result{}, fmt.Errorf("resolving the path of project %s: %w", root, err)
	}

	dir := filepath.Join(root, DirName)

	cfg, err := prepare(dir, prefix)
	if err != nil {
		return Result{}, err
	}

	dbPath := settings.DatabasePath(dir, cfg)
	instructions := filepath.Join(dir, instructionsName)

	if exists(dbPath) && exists(instructions) {
		return Result{}, fmt.Errorf("the project already has the product installed in %s", dir)
	}

	// Las instrucciones van antes que la base: si algo falla a mitad, lo que ya
	// está hecho se puede volver a completar sin borrar nada.
	if err := writeInstructions(dir, cfg.Prefix); err != nil {
		return Result{}, err
	}

	base, err := store.Open(ctx, dbPath)
	if err != nil {
		return Result{}, err
	}

	defer func() { _ = base.Close() }()

	note, err := core.New(base, cfg.Prefix).CheckPrefix(ctx)
	if err != nil {
		return Result{}, err
	}

	// El aviso sale por la salida de errores igual que en los demás comandos: el
	// prefijo distinto no invalida la instalación, que sigue completando.
	if note != "" {
		_, _ = fmt.Fprintf(warning, "warning: %s\n", note)
	}

	binary, err := copyBinary(dir)
	if err != nil {
		return Result{}, err
	}

	extension, err := writeExtension(dir)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Dir:          dir,
		DB:           dbPath,
		Instructions: instructions,
		Prefix:       cfg.Prefix,
		Pointer:      pointer,
		Binary:       binary,
		Extension:    extension,
	}, nil
}

// exists says whether there is a file at path.
func exists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}

// binName is the name of the binary the install leaves inside the product folder.
const binName = "tasks-ai"

// copyBinary deja el producto en la carpeta del proyecto, copiado del que se está
// ejecutando, para que el proyecto no dependa de dónde se compiló. Devuelve la ruta
// del binario instalado.
func copyBinary(dir string) (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("finding the running binary: %w", err)
	}

	target := filepath.Join(dir, "bin", binName)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil { //nolint:gosec // la carpeta del producto usa los permisos propios del proyecto
		return "", fmt.Errorf("creating %s: %w", filepath.Dir(target), err)
	}

	if err := copyFile(self, target, 0o755); err != nil {
		return "", fmt.Errorf("copying the binary to %s: %w", target, err)
	}

	return target, nil
}

// writeExtension deja la extensión del agente dentro de la carpeta del producto,
// extraída de los ficheros que el binario lleva incrustados, para que el proyecto
// no dependa del repositorio de origen. Devuelve la ruta de la carpeta de la
// extensión instalada.
func writeExtension(dir string) (string, error) {
	target := filepath.Join(dir, "extensions", "pi")

	err := fs.WalkDir(embedded.Pi, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() {
			return nil
		}

		data, err := embedded.Pi.ReadFile(path)
		if err != nil {
			return err
		}

		dest := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil { //nolint:gosec // la carpeta del producto usa los permisos propios del proyecto
			return fmt.Errorf("creating %s: %w", filepath.Dir(dest), err)
		}

		if err := os.WriteFile(dest, data, 0o644); err != nil { //nolint:gosec // los ficheros de la extensión se leen sin ejecutar
			return fmt.Errorf("writing %s: %w", dest, err)
		}

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("writing the extension: %w", err)
	}

	return target, nil
}

// copyFile copia un fichero desde from hasta to, con los permisos dados.
func copyFile(from, to string, mode os.FileMode) error {
	src, err := os.Open(from) //nolint:gosec // el origen lo decide la instalación, no una entrada externa
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	dst, err := os.OpenFile(to, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode) //nolint:gosec // el destino lo decide la instalación, no una entrada externa
	if err != nil {
		return err
	}
	defer func() { _ = dst.Close() }()

	_, err = io.Copy(dst, src)

	return err
}

// prepare creates the product folder and leaves a settings file in it with the
// prefix asked for, without touching the one the project may have written
// itself.
func prepare(dir, prefix string) (settings.Settings, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // la carpeta del producto usa los permisos propios del proyecto
		return settings.Settings{}, fmt.Errorf("creating the folder %s: %w", dir, err)
	}

	if !settings.Exists(dir) {
		cfg := settings.Default()
		if prefix != "" {
			cfg.Prefix = prefix
		}

		if err := settings.Save(dir, cfg); err != nil {
			return settings.Settings{}, err
		}
	}

	cfg, err := settings.Load(dir)
	if err != nil {
		return settings.Settings{}, err
	}

	if prefix != "" && cfg.Prefix != prefix {
		return settings.Settings{}, fmt.Errorf(
			"the project already has its own settings in %s with prefix %q: edit that file to change it",
			filepath.Join(dir, settings.FileName),
			cfg.Prefix,
		)
	}

	return cfg, nil
}

// writeInstructions leaves the instructions in the product folder, with the
// prefix of the project in its examples, without treading on the ones that are
// already there.
func writeInstructions(dir, prefix string) error {
	path := filepath.Join(dir, instructionsName)

	if exists(path) {
		return nil
	}

	text := bytes.ReplaceAll(instructions, []byte(settings.DefaultPrefix+"-"), []byte(prefix+"-"))

	if err := os.WriteFile(path, text, 0o644); err != nil { //nolint:gosec // las instrucciones se escriben con los permisos de lectura del proyecto
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}
