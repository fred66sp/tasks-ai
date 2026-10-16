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
)

// project is one project's data, open for the work at hand.
type project struct {
	core *core.Core
	base *store.Store
}

// openProject finds the project the working folder dir belongs to, reads its
// settings and opens its data.
func openProject(ctx context.Context, dir string, warn io.Writer) (*project, error) {
	root, err := install.Project(dir)
	if err != nil {
		return nil, err
	}

	cfg, err := settings.Load(filepath.Join(root, install.DirName))
	if err != nil {
		return nil, err
	}

	base, err := store.Open(ctx, settings.DatabasePath(filepath.Join(root, install.DirName), cfg))
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

	return &project{core: apply, base: base}, nil
}

// Close lets the project's data go.
func (p *project) Close() error {
	return p.base.Close()
}

// workingDir is where the product was called from.
func workingDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("no se pudo leer la carpeta desde la que se llamó: %w", err)
	}

	return dir, nil
}
