package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	sqlite3 "modernc.org/sqlite/lib"
)

// ErrTagExists dice que ya hay una etiqueta guardada con ese nombre.
var ErrTagExists = errors.New("ya existe una etiqueta con ese nombre")

// ErrUnknownTag dice que una de las etiquetas que se quisieron asignar no está
// en el catálogo.
var ErrUnknownTag = errors.New("etiqueta desconocida")

// Tag es una etiqueta tal como se guarda.
type Tag struct {
	ID          int64
	Name        string
	Color       string
	Description *string
}

// TagUse es una etiqueta del catálogo con el número de tareas que la usan.
type TagUse struct {
	Tag
	Uses int
}

// CreateTag guarda una etiqueta nueva y la devuelve con la identidad que la base
// le dio. Un nombre que ya está en el catálogo sale por ErrTagExists.
func (s *Store) CreateTag(ctx context.Context, tag Tag) (Tag, error) {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO tags (name, color, description) VALUES (?, ?, ?)`,
		tag.Name, tag.Color, tag.Description)
	if err != nil {
		if isConstraint(err) {
			return Tag{}, ErrTagExists
		}

		return Tag{}, fmt.Errorf("guardando la etiqueta %q: %w", tag.Name, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Tag{}, fmt.Errorf("leyendo la identidad de la etiqueta guardada: %w", err)
	}

	tag.ID = id

	return tag, nil
}

// Tags devuelve el catálogo entero con el número de tareas que usa cada
// etiqueta, ordenado por nombre.
func (s *Store) Tags(ctx context.Context) ([]TagUse, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.name, t.color, t.description, count(tt.task_id)
		FROM tags t
		LEFT JOIN task_tags tt ON tt.tag_id = t.id
		GROUP BY t.id
		ORDER BY t.name`)
	if err != nil {
		return nil, fmt.Errorf("leyendo el catálogo de etiquetas: %w", err)
	}

	defer func() { _ = rows.Close() }()

	catalogue := []TagUse{}

	for rows.Next() {
		var entry TagUse

		if err := rows.Scan(&entry.ID, &entry.Name, &entry.Color, &entry.Description, &entry.Uses); err != nil {
			return nil, fmt.Errorf("leyendo el catálogo de etiquetas: %w", err)
		}

		catalogue = append(catalogue, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("leyendo el catálogo de etiquetas: %w", err)
	}

	return catalogue, nil
}

// Tag devuelve la etiqueta con ese nombre.
func (s *Store) Tag(ctx context.Context, name string) (Tag, error) {
	return readTag(ctx, s.db, name)
}

// readTag reads one tag from wherever it is asked for.
func readTag(ctx context.Context, from rowQuerier, name string) (Tag, error) {
	var tag Tag

	err := from.QueryRowContext(ctx,
		`SELECT id, name, color, description FROM tags WHERE name = ?`, name,
	).Scan(&tag.ID, &tag.Name, &tag.Color, &tag.Description)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return Tag{}, ErrNotFound
	case err != nil:
		return Tag{}, fmt.Errorf("leyendo la etiqueta %q: %w", name, err)
	}

	return tag, nil
}

// UpdateTag reads the tag, hands it to change and writes back what change
// returns, all inside one transaction. change must not ask the base for
// anything, because the transaction already holds the write lock. A rename onto
// a name another tag already carries leaves everything as it was and comes back
// as ErrTagExists, and a name nobody carries comes back as ErrNotFound.
func (s *Store) UpdateTag(ctx context.Context, name string, change func(Tag) (Tag, error)) (Tag, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Tag{}, fmt.Errorf("abriendo la transacción del cambio de etiqueta: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	stored, err := readTag(ctx, tx, name)
	if err != nil {
		return Tag{}, err
	}

	changed, err := change(stored)
	if err != nil {
		return Tag{}, err
	}

	if changed.Name != stored.Name {
		if err := checkTagName(ctx, tx, changed.Name, stored.ID); err != nil {
			return Tag{}, err
		}
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE tags SET name = ?, color = ?, description = ? WHERE id = ?`,
		changed.Name, changed.Color, changed.Description, stored.ID)
	if err != nil {
		return Tag{}, fmt.Errorf("guardando el cambio de la etiqueta %q: %w", name, err)
	}

	if err := tx.Commit(); err != nil {
		return Tag{}, fmt.Errorf("cerrando la transacción del cambio de la etiqueta %q: %w", name, err)
	}

	return changed, nil
}

// checkTagName refuses a name another tag already carries. It runs inside the
// transaction that holds the write lock, so nobody can slip the same name in
// between the check and the write.
func checkTagName(ctx context.Context, within *sql.Tx, name string, own int64) error {
	var found int64

	err := within.QueryRowContext(ctx,
		`SELECT id FROM tags WHERE name = ?`, name).Scan(&found)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil
	case err != nil:
		return fmt.Errorf("mirando el nombre %q del catálogo: %w", name, err)
	}

	if found != own {
		return ErrTagExists
	}

	return nil
}

// DeleteTag deletes the tag with that name together with its assignments, all
// inside one transaction, and returns the deleted tag and the tasks that were
// using it. A name nobody carries comes back as ErrNotFound and nothing is
// written.
func (s *Store) DeleteTag(ctx context.Context, name string) (Tag, []int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Tag{}, nil, fmt.Errorf("abriendo la transacción del borrado de etiqueta: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	stored, err := readTag(ctx, tx, name)
	if err != nil {
		return Tag{}, nil, err
	}

	users, err := readTagUsers(ctx, tx, stored.ID)
	if err != nil {
		return Tag{}, nil, err
	}

	for _, statement := range []string{
		`DELETE FROM task_tags WHERE tag_id = ?`,
		`DELETE FROM tags WHERE id = ?`,
	} {
		if _, err := tx.ExecContext(ctx, statement, stored.ID); err != nil {
			return Tag{}, nil, fmt.Errorf("borrando la etiqueta %q: %w", name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return Tag{}, nil, fmt.Errorf("cerrando la transacción del borrado de la etiqueta %q: %w", name, err)
	}

	return stored, users, nil
}

// readTagUsers reads the identities of the tasks that carry the tag.
func readTagUsers(ctx context.Context, within *sql.Tx, tagID int64) ([]int64, error) {
	rows, err := within.QueryContext(ctx,
		`SELECT task_id FROM task_tags WHERE tag_id = ?`, tagID)
	if err != nil {
		return nil, fmt.Errorf("buscando las tareas de la etiqueta %d: %w", tagID, err)
	}

	defer func() { _ = rows.Close() }()

	users := []int64{}

	for rows.Next() {
		var taskID int64

		if err := rows.Scan(&taskID); err != nil {
			return nil, fmt.Errorf("buscando las tareas de la etiqueta %d: %w", tagID, err)
		}

		users = append(users, taskID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("buscando las tareas de la etiqueta %d: %w", tagID, err)
	}

	return users, nil
}

// TaskTagNames returns the names of the tags a task carries, ordered by name.
func (s *Store) TaskTagNames(ctx context.Context, taskID int64) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.name
		FROM task_tags tt
		JOIN tags t ON t.id = tt.tag_id
		WHERE tt.task_id = ?
		ORDER BY t.name`, taskID)
	if err != nil {
		return nil, fmt.Errorf("leyendo las etiquetas de la tarea %d: %w", taskID, err)
	}

	defer func() { _ = rows.Close() }()

	names := []string{}

	for rows.Next() {
		var name string

		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("leyendo las etiquetas de la tarea %d: %w", taskID, err)
		}

		names = append(names, name)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("leyendo las etiquetas de la tarea %d: %w", taskID, err)
	}

	return names, nil
}

// replaceTaskTags leaves the task with exactly the tags the list of names
// names, inside the transaction that holds the write lock. A name that is not
// in the catalogue refuses the whole change and nothing is written.
func replaceTaskTags(ctx context.Context, within *sql.Tx, taskID int64, names []string) error {
	if _, err := within.ExecContext(ctx, `DELETE FROM task_tags WHERE task_id = ?`, taskID); err != nil {
		return fmt.Errorf("quitando las etiquetas de la tarea %d: %w", taskID, err)
	}

	assigned := []string{}

	for _, name := range names {
		if slices.Contains(assigned, name) {
			continue
		}

		var tagID int64

		err := within.QueryRowContext(ctx,
			`SELECT id FROM tags WHERE name = ?`, name).Scan(&tagID)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: %q", ErrUnknownTag, name)
		}

		if err != nil {
			return fmt.Errorf("buscando la etiqueta %q: %w", name, err)
		}

		if _, err := within.ExecContext(ctx,
			`INSERT INTO task_tags (task_id, tag_id) VALUES (?, ?)`, taskID, tagID); err != nil {
			return fmt.Errorf("asignando la etiqueta %q a la tarea %d: %w", name, taskID, err)
		}

		assigned = append(assigned, name)
	}

	return nil
}

// isConstraint reports whether the error is one of SQLite's constraint
// violations, the only case CreateTag translates into ErrTagExists.
func isConstraint(err error) bool {
	return isCode(err, sqlite3.SQLITE_CONSTRAINT)
}
