package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"
)

// timeLayout is how the moments travel to the base: readable, ordered and in UTC.
const timeLayout = time.RFC3339Nano

// statusDone es el estado de una tarea cerrada.
const statusDone = "done"

// Task is a task as it is stored.
type Task struct {
	ID          int64
	Title       string
	Description *string
	Status      string
	Priority    int
	Assignee    *string
	ParentID    *int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CloseReason *string
	Notes       *string
}

// CreateTask stores a new task and returns it with the identity the base gave
// it. La tarea y el aviso de su nacimiento se escriben en la misma transacción:
// un aviso sin la tarea de la que habla mentiría a quien lo lee.
func (s *Store) CreateTask(ctx context.Context, task Task) (Task, error) {
	now := time.Now().UTC().Truncate(time.Second)
	task.CreatedAt, task.UpdatedAt = now, now

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, fmt.Errorf("opening the creation transaction: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(ctx, `
		INSERT INTO tasks (title, description, status, priority, assignee, parent_id, created_at, updated_at, close_reason, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.Title,
		task.Description,
		task.Status,
		task.Priority,
		task.Assignee,
		task.ParentID,
		task.CreatedAt.Format(timeLayout),
		task.UpdatedAt.Format(timeLayout),
		task.CloseReason,
		task.Notes,
	)
	if err != nil {
		return Task{}, fmt.Errorf("saving the task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Task{}, fmt.Errorf("reading the identity of the stored task: %w", err)
	}

	if err := recordChange(ctx, tx, s.origin, ResourceTask, id, ActionCreate); err != nil {
		return Task{}, err
	}

	// La subtarea nace con su etiqueta estructural: la misma regla que el
	// cambio de padre, dentro de la misma transacción que la crea.
	if task.ParentID != nil {
		if err := ensureTaskTag(ctx, tx, s.origin); err != nil {
			return Task{}, err
		}

		if err := replaceTaskTags(ctx, tx, id, []string{structuralTaskTag}); err != nil {
			return Task{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Task{}, fmt.Errorf("closing the creation transaction: %w", err)
	}

	task.ID = id

	return task, nil
}

// Tasks returns every task of the project, by the identity the base gave them.
func (s *Store) Tasks(ctx context.Context) ([]Task, error) {
	return s.queryTasks(ctx, `SELECT `+taskColumns+` FROM tasks ORDER BY id`)
}

// Task returns the task with that identity.
func (s *Store) Task(ctx context.Context, id int64) (Task, error) {
	return readTask(ctx, s.db, id)
}

// TaskTitles devuelve el título que lleva ahora cada tarea pedida por su
// identidad interna: una consulta para el lote entero, no una por tarea. La
// tarea que ya no está no aparece en el mapa.
//
//nolint:gosec // la consulta solo añade marcadores, nunca datos
func (s *Store) TaskTitles(ctx context.Context, ids []int64) (map[int64]string, error) {
	titles := make(map[int64]string, len(ids))
	if len(ids) == 0 {
		return titles, nil
	}

	// La consulta solo concatena marcadores, nunca datos: de ahí el nolint.
	query := `SELECT id, title FROM tasks WHERE id IN (` + inPlaceholders(len(ids)) + `)`

	rows, err := s.db.QueryContext(ctx, query, int64Args(ids)...)
	if err != nil {
		return nil, fmt.Errorf("reading the titles of %d tasks: %w", len(ids), err)
	}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			id    int64
			title string
		)

		if err := rows.Scan(&id, &title); err != nil {
			return nil, fmt.Errorf("reading the titles of %d tasks: %w", len(ids), err)
		}

		titles[id] = title
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading the titles of %d tasks: %w", len(ids), err)
	}

	return titles, nil
}

// taskColumns is the column list every read of tasks asks for, in the order
// the scan expects them.
const taskColumns = `id, title, description, status, priority, assignee, parent_id,
	created_at, updated_at, close_reason, notes`

// readTask reads one task from wherever it is asked for.
func readTask(ctx context.Context, from rowQuerier, id int64) (Task, error) {
	task, err := scanTask(func(dest ...any) error {
		return from.QueryRowContext(ctx,
			`SELECT `+taskColumns+` FROM tasks WHERE id = ?`, id).Scan(dest...)
	})
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return Task{}, ErrNotFound
	case err != nil:
		return Task{}, fmt.Errorf("reading task %d: %w", id, err)
	}

	return task, nil
}

// queryTasks reads the tasks a query answers, in the order the query leaves them.
func (s *Store) queryTasks(ctx context.Context, query string, args ...any) ([]Task, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("reading tasks: %w", err)
	}

	defer func() { _ = rows.Close() }()

	tasks := []Task{}

	for rows.Next() {
		task, err := scanTask(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("reading tasks: %w", err)
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading tasks: %w", err)
	}

	return tasks, nil
}

// scanTask reads one row of the task columns and understands its moments, which
// travel as text. ErrNoRows passes through for whoever reads a single row to
// translate it; the rest of the failures already say what went wrong.
func scanTask(scan func(dest ...any) error) (Task, error) {
	var (
		task               Task
		createdAt, updated string
	)

	err := scan(&task.ID, &task.Title, &task.Description, &task.Status, &task.Priority,
		&task.Assignee, &task.ParentID, &createdAt, &updated, &task.CloseReason, &task.Notes)
	if err != nil {
		return Task{}, err
	}

	if task.CreatedAt, err = time.Parse(timeLayout, createdAt); err != nil {
		return Task{}, fmt.Errorf("the creation date of task %d cannot be read: %w", task.ID, err)
	}

	if task.UpdatedAt, err = time.Parse(timeLayout, updated); err != nil {
		return Task{}, fmt.Errorf("the change date of task %d cannot be read: %w", task.ID, err)
	}

	return task, nil
}

// UpdateTask reads the task, hands it to change and writes back what change
// returns. When tags is not nil it also leaves the task with exactly those tag
// names, refusing the whole change when one of them is not in the catalogue.
// Everything happens inside one transaction: two writes at once queue
// behind the base instead of overwriting each other. change must not ask the
// base for anything, because the transaction already holds the write lock.
func (s *Store) UpdateTask(ctx context.Context, id int64, change func(Task) (Task, error), tags []string) (Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, fmt.Errorf("opening the change transaction: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	stored, err := readTask(ctx, tx, id)
	if err != nil {
		return Task{}, err
	}

	changed, err := change(stored)
	if err != nil {
		return Task{}, err
	}

	// La regla del padre se comprueba dentro de la transacción que guarda el
	// bloqueo de escritura: un ciclo que se rechaza no deja nada escrito y
	// nadie puede colar la jerarquía entre la comprobación y la escritura.
	parentChanged := !sameParent(stored.ParentID, changed.ParentID)
	if parentChanged && changed.ParentID != nil {
		if err := checkNoCycle(ctx, tx, id, *changed.ParentID); err != nil {
			return Task{}, err
		}
	}

	// Neither the identity nor the moment of creation belong to whoever changes
	// the task: they are kept from what was read, whatever comes back.
	changed.ID = stored.ID
	changed.CreatedAt = stored.CreatedAt
	changed.UpdatedAt = time.Now().UTC().Truncate(time.Second)

	_, err = tx.ExecContext(ctx, `
		UPDATE tasks
		SET title = ?, description = ?, status = ?, priority = ?, assignee = ?, parent_id = ?,
		    updated_at = ?, close_reason = ?, notes = ?
		WHERE id = ?`,
		changed.Title,
		changed.Description,
		changed.Status,
		changed.Priority,
		changed.Assignee,
		changed.ParentID,
		changed.UpdatedAt.Format(timeLayout),
		changed.CloseReason,
		changed.Notes,
		id,
	)
	if err != nil {
		return Task{}, fmt.Errorf("saving the change of task %d: %w", id, err)
	}

	if err := syncTaskTags(ctx, tx, s.origin, id, tags, parentChanged, changed.ParentID); err != nil {
		return Task{}, err
	}

	if err := recordChange(ctx, tx, s.origin, ResourceTask, id, ActionUpdate); err != nil {
		return Task{}, err
	}

	if err := tx.Commit(); err != nil {
		return Task{}, fmt.Errorf("closing the change transaction of task %d: %w", id, err)
	}

	return changed, nil
}

// syncTaskTags leaves the task with the tags its change asks for, inside the
// transaction that holds the write lock. When the change touched the parent,
// the structural tag follows it — with a parent the task carries it, without
// one it loses it — and the rest of the tags keep what the change asked for
// them, or what the task already had when no list was given. A change that
// names neither the tags nor the parent leaves the tags as they are.
func syncTaskTags(ctx context.Context, within *sql.Tx, origin string, taskID int64, given []string, parentChanged bool, parent *int64) error {
	if given == nil && !parentChanged {
		return nil
	}

	names := given

	if names == nil {
		read, err := readTaskTagNames(ctx, within, taskID)
		if err != nil {
			return err
		}

		names = read
	}

	if !parentChanged {
		return replaceTaskTags(ctx, within, taskID, names)
	}

	if parent != nil {
		if !slices.Contains(names, structuralTaskTag) {
			if err := ensureTaskTag(ctx, within, origin); err != nil {
				return err
			}

			names = append(slices.Clone(names), structuralTaskTag)
		}
	} else {
		names = slices.DeleteFunc(slices.Clone(names), func(name string) bool {
			return name == structuralTaskTag
		})
	}

	return replaceTaskTags(ctx, within, taskID, names)
}

// CloseTask closes the task with the reason given. The refusal and the write
// happen inside one transaction: the check of the subtasks and the write of the
// close read and write under the same lock, so a close that is refused leaves
// nothing written and nobody can slip a subtask in between the check and the
// write. It returns ErrClosed for an already closed task and ErrOpenSubtasks
// for one with subtasks that are not done.
func (s *Store) CloseTask(ctx context.Context, id int64, reason string) (Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, fmt.Errorf("opening the close transaction: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	stored, err := readTask(ctx, tx, id)
	if err != nil {
		return Task{}, err
	}

	if stored.Status == statusDone {
		return Task{}, ErrClosed
	}

	var open int64

	err = tx.QueryRowContext(ctx,
		`SELECT count(*) FROM tasks WHERE parent_id = ? AND status <> ?`, id, statusDone,
	).Scan(&open)
	if err != nil {
		return Task{}, fmt.Errorf("looking at the subtasks of task %d: %w", id, err)
	}

	if open > 0 {
		return Task{}, ErrOpenSubtasks
	}

	now := time.Now().UTC().Truncate(time.Second)

	_, err = tx.ExecContext(ctx, `
		UPDATE tasks
		SET status = ?, close_reason = ?, updated_at = ?
		WHERE id = ?`,
		statusDone,
		reason,
		now.Format(timeLayout),
		id,
	)
	if err != nil {
		return Task{}, fmt.Errorf("saving the close of task %d: %w", id, err)
	}

	if err := recordChange(ctx, tx, s.origin, ResourceTask, id, ActionClose); err != nil {
		return Task{}, err
	}

	if err := tx.Commit(); err != nil {
		return Task{}, fmt.Errorf("closing the close transaction of task %d: %w", id, err)
	}

	stored.Status = statusDone
	stored.CloseReason = &reason
	stored.UpdatedAt = now

	return stored, nil
}

// checkNoCycle refuses a parent change that would make the task its own
// descendant: the new parent cannot be the task itself and cannot hang from
// it. It walks the ancestors of the new parent inside the transaction that
// holds the write lock, so nobody can slip a hierarchy in between the check
// and the write.
func checkNoCycle(ctx context.Context, within *sql.Tx, taskID, newParentID int64) error {
	if newParentID == taskID {
		return ErrCycle
	}

	seen := map[int64]bool{taskID: true}
	current := newParentID

	for current != 0 {
		if seen[current] {
			return ErrCycle
		}

		seen[current] = true

		var parentID sql.NullInt64

		err := within.QueryRowContext(ctx,
			`SELECT parent_id FROM tasks WHERE id = ?`, current).Scan(&parentID)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrNotFound
		case err != nil:
			return fmt.Errorf("walking the ancestors of task %d: %w", current, err)
		}

		if !parentID.Valid {
			return nil
		}

		current = parentID.Int64
	}

	return nil
}

// sameParent says whether two optional parents are the same one.
func sameParent(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	return *a == *b
}
