package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

// CreateTask stores a new task and returns it with the identity the base gave it.
func (s *Store) CreateTask(ctx context.Context, task Task) (Task, error) {
	now := time.Now().UTC().Truncate(time.Second)
	task.CreatedAt, task.UpdatedAt = now, now

	result, err := s.db.ExecContext(ctx, `
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
		return Task{}, fmt.Errorf("guardando la tarea: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Task{}, fmt.Errorf("leyendo la identidad de la tarea guardada: %w", err)
	}

	task.ID = id

	return task, nil
}

// Task returns the task with that identity.
func (s *Store) Task(ctx context.Context, id int64) (Task, error) {
	return readTask(ctx, s.db, id)
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
		return Task{}, fmt.Errorf("leyendo la tarea %d: %w", id, err)
	}

	return task, nil
}

// queryTasks reads the tasks a query answers, in the order the query leaves them.
func (s *Store) queryTasks(ctx context.Context, query string, args ...any) ([]Task, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("leyendo tareas: %w", err)
	}

	defer func() { _ = rows.Close() }()

	tasks := []Task{}

	for rows.Next() {
		task, err := scanTask(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("leyendo tareas: %w", err)
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("leyendo tareas: %w", err)
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
		return Task{}, fmt.Errorf("la fecha de creación de la tarea %d no se entiende: %w", task.ID, err)
	}

	if task.UpdatedAt, err = time.Parse(timeLayout, updated); err != nil {
		return Task{}, fmt.Errorf("la fecha de cambio de la tarea %d no se entiende: %w", task.ID, err)
	}

	return task, nil
}

// UpdateTask reads the task, hands it to change and writes back what change
// returns. Everything happens inside one transaction: two writes at once queue
// behind the base instead of overwriting each other. change must not ask the
// base for anything, because the transaction already holds the write lock.
func (s *Store) UpdateTask(ctx context.Context, id int64, change func(Task) (Task, error)) (Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, fmt.Errorf("abriendo la transacción del cambio: %w", err)
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
		return Task{}, fmt.Errorf("guardando el cambio de la tarea %d: %w", id, err)
	}

	if err := tx.Commit(); err != nil {
		return Task{}, fmt.Errorf("cerrando la transacción del cambio de la tarea %d: %w", id, err)
	}

	return changed, nil
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
		return Task{}, fmt.Errorf("abriendo la transacción del cierre: %w", err)
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
		return Task{}, fmt.Errorf("mirando las subtareas de la tarea %d: %w", id, err)
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
		return Task{}, fmt.Errorf("guardando el cierre de la tarea %d: %w", id, err)
	}

	if err := tx.Commit(); err != nil {
		return Task{}, fmt.Errorf("cerrando la transacción del cierre de la tarea %d: %w", id, err)
	}

	stored.Status = statusDone
	stored.CloseReason = &reason
	stored.UpdatedAt = now

	return stored, nil
}
