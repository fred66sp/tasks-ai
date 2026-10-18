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
var ErrTagExists = errors.New("a tag with that name already exists")

// ErrUnknownTag dice que una de las etiquetas que se quisieron asignar no está
// en el catálogo.
var ErrUnknownTag = errors.New("unknown tag")

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
// le dio. Un nombre que ya está en el catálogo sale por ErrTagExists. La
// etiqueta y el aviso se escriben en la misma transacción.
func (s *Store) CreateTag(ctx context.Context, tag Tag) (Tag, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Tag{}, fmt.Errorf("opening the tag transaction: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(ctx,
		`INSERT INTO tags (name, color, description) VALUES (?, ?, ?)`,
		tag.Name, tag.Color, tag.Description)
	if err != nil {
		if isConstraint(err) {
			return Tag{}, ErrTagExists
		}

		return Tag{}, fmt.Errorf("saving tag %q: %w", tag.Name, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Tag{}, fmt.Errorf("reading the identity of the stored tag: %w", err)
	}

	if err := recordChange(ctx, tx, s.origin, ResourceTag, id, ActionCreate); err != nil {
		return Tag{}, err
	}

	if err := tx.Commit(); err != nil {
		return Tag{}, fmt.Errorf("closing the tag transaction: %w", err)
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
		return nil, fmt.Errorf("reading the tag catalogue: %w", err)
	}

	defer func() { _ = rows.Close() }()

	catalogue := []TagUse{}

	for rows.Next() {
		var entry TagUse

		if err := rows.Scan(&entry.ID, &entry.Name, &entry.Color, &entry.Description, &entry.Uses); err != nil {
			return nil, fmt.Errorf("reading the tag catalogue: %w", err)
		}

		catalogue = append(catalogue, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading the tag catalogue: %w", err)
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
		return Tag{}, fmt.Errorf("reading tag %q: %w", name, err)
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
		return Tag{}, fmt.Errorf("opening the tag change transaction: %w", err)
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
		return Tag{}, fmt.Errorf("saving the change of tag %q: %w", name, err)
	}

	if err := recordChange(ctx, tx, s.origin, ResourceTag, stored.ID, ActionUpdate); err != nil {
		return Tag{}, err
	}

	if err := tx.Commit(); err != nil {
		return Tag{}, fmt.Errorf("closing the tag change transaction of %q: %w", name, err)
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
		return fmt.Errorf("looking up the name %q of the catalogue: %w", name, err)
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
		return Tag{}, nil, fmt.Errorf("opening the tag deletion transaction: %w", err)
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
			return Tag{}, nil, fmt.Errorf("deleting tag %q: %w", name, err)
		}
	}

	if err := recordChange(ctx, tx, s.origin, ResourceTag, stored.ID, ActionDelete); err != nil {
		return Tag{}, nil, err
	}

	if err := tx.Commit(); err != nil {
		return Tag{}, nil, fmt.Errorf("closing the tag deletion transaction of %q: %w", name, err)
	}

	return stored, users, nil
}

// readTagUsers reads the identities of the tasks that carry the tag.
func readTagUsers(ctx context.Context, within *sql.Tx, tagID int64) ([]int64, error) {
	rows, err := within.QueryContext(ctx,
		`SELECT task_id FROM task_tags WHERE tag_id = ?`, tagID)
	if err != nil {
		return nil, fmt.Errorf("looking up the tasks of tag %d: %w", tagID, err)
	}

	defer func() { _ = rows.Close() }()

	users := []int64{}

	for rows.Next() {
		var taskID int64

		if err := rows.Scan(&taskID); err != nil {
			return nil, fmt.Errorf("looking up the tasks of tag %d: %w", tagID, err)
		}

		users = append(users, taskID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("looking up the tasks of tag %d: %w", tagID, err)
	}

	return users, nil
}

// TagNames devuelve, para cada etiqueta pedida, el nombre que lleva ahora: el
// aviso viaja por el número interno, y quien lo lee compone aquí los nombres de
// todo un lote con una sola consulta, no con una por aviso. La etiqueta que ya no
// está —la que se borró o la que nadie tiene— sale fuera del mapa.
func (s *Store) TagNames(ctx context.Context, ids []int64) (map[int64]string, error) {
	if len(ids) == 0 {
		return map[int64]string{}, nil
	}

	query := `SELECT id, name FROM tags WHERE id IN (` + inPlaceholders(len(ids)) + `)`

	return idNames(ctx, s.db, query, ids)
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
		return nil, fmt.Errorf("reading the tags of task %d: %w", taskID, err)
	}

	defer func() { _ = rows.Close() }()

	names := []string{}

	for rows.Next() {
		var name string

		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("reading the tags of task %d: %w", taskID, err)
		}

		names = append(names, name)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading the tags of task %d: %w", taskID, err)
	}

	return names, nil
}

// TasksTagNames devuelve, para cada tarea de la lista, los nombres de las
// etiquetas que lleva, ordenados por nombre: una consulta para todas las
// tareas, no una por tarea ni una por etiqueta. Una tarea sin etiquetas sale
// con la lista vacía.
//
//nolint:gosec // la consulta solo añade marcadores, nunca datos
func (s *Store) TasksTagNames(ctx context.Context, taskIDs []int64) (map[int64][]string, error) {
	names := make(map[int64][]string, len(taskIDs))
	for _, id := range taskIDs {
		names[id] = []string{}
	}

	if len(taskIDs) == 0 {
		return names, nil
	}

	// La consulta solo concatena marcadores, nunca datos: de ahí el nolint.
	query := `
		SELECT tt.task_id, t.name
		FROM task_tags tt
		JOIN tags t ON t.id = tt.tag_id
		WHERE tt.task_id IN (` + inPlaceholders(len(taskIDs)) + `)
		ORDER BY t.name`

	rows, err := s.db.QueryContext(ctx, query, int64Args(taskIDs)...)
	if err != nil {
		return nil, fmt.Errorf("reading the tags of %d tasks: %w", len(taskIDs), err)
	}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			taskID int64
			name   string
		)

		if err := rows.Scan(&taskID, &name); err != nil {
			return nil, fmt.Errorf("reading the tags of %d tasks: %w", len(taskIDs), err)
		}

		names[taskID] = append(names[taskID], name)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading the tags of %d tasks: %w", len(taskIDs), err)
	}

	return names, nil
}

// replaceTaskTags leaves the task with exactly the tags the list of names
// names, inside the transaction that holds the write lock. A name that is not
// in the catalogue refuses the whole change and nothing is written.
func replaceTaskTags(ctx context.Context, within *sql.Tx, taskID int64, names []string) error {
	if _, err := within.ExecContext(ctx, `DELETE FROM task_tags WHERE task_id = ?`, taskID); err != nil {
		return fmt.Errorf("removing the tags of task %d: %w", taskID, err)
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
			return fmt.Errorf("looking up tag %q: %w", name, err)
		}

		if _, err := within.ExecContext(ctx,
			`INSERT INTO task_tags (task_id, tag_id) VALUES (?, ?)`, taskID, tagID); err != nil {
			return fmt.Errorf("assigning tag %q to task %d: %w", name, taskID, err)
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
