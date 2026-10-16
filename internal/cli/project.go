package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"tasks-ai/internal/core"
	"tasks-ai/internal/install"
	"tasks-ai/internal/settings"
	"tasks-ai/internal/store"

	"github.com/spf13/cobra"
)

// project es la base de un proyecto, abierta para el trabajo de turno, con los
// ajustes con los que se abrió.
type project struct {
	core *core.Core
	base *store.Store
	cfg  settings.Settings
}

// openProject finds the project the working folder dir belongs to, reads its
// settings and opens its data saying which surface asks for the changes: la
// línea de comandos abre la suya y el servidor la suya, y así sabe una sesión
// que un cambio viene de otra.
func openProject(ctx context.Context, dir string, warn io.Writer, origin string) (*project, error) {
	root, err := install.Project(dir)
	if err != nil {
		return nil, err
	}

	cfg, err := settings.Load(filepath.Join(root, install.DirName))
	if err != nil {
		return nil, err
	}

	base, err := store.OpenAs(ctx, settings.DatabasePath(filepath.Join(root, install.DirName), cfg), origin)
	if err != nil {
		return nil, err
	}

	apply := core.New(base, cfg.Prefix)

	note, err := apply.CheckPrefix(ctx)
	if err != nil {
		_ = base.Close()

		return nil, err
	}

	if note != "" {
		_, _ = fmt.Fprintf(warn, "aviso: %s\n", note)
	}

	return &project{core: apply, base: base, cfg: cfg}, nil
}

// Close lets the project's data go.
func (p *project) Close() error {
	return p.base.Close()
}

// withProject opens the project of the working folder, hands it to run and
// closes it on the way out. It keeps the boilerplate of opening and closing out
// of every command.
func withProject(command *cobra.Command, run func(*project) error) error {
	dir, err := workingDir()
	if err != nil {
		return err
	}

	opened, err := openProject(command.Context(), dir, command.ErrOrStderr(), store.OriginCLI)
	if err != nil {
		return err
	}

	defer func() { _ = opened.Close() }()

	return run(opened)
}

// workingDir is where the product was called from.
func workingDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("no se pudo leer la carpeta desde la que se llamó: %w", err)
	}

	return dir, nil
}
