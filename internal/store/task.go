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

// readTask reads one task from wherever it is asked for.
func readTask(ctx context.Context, from rowQuerier, id int64) (Task, error) {
	var (
		task               Task
		createdAt, updated string
	)

	err := from.QueryRowContext(ctx, `
		SELECT id, title, description, status, priority, assignee, parent_id, created_at, updated_at, close_reason, notes
		FROM tasks
		WHERE id = ?`, id,
	).Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&task.Status,
		&task.Priority,
		&task.Assignee,
		&task.ParentID,
		&createdAt,
		&updated,
		&task.CloseReason,
		&task.Notes,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, ErrNotFound
	}

	if err != nil {
		return Task{}, fmt.Errorf("leyendo la tarea %d: %w", id, err)
	}

	if task.CreatedAt, err = time.Parse(timeLayout, createdAt); err != nil {
		return Task{}, fmt.Errorf("la fecha de creación de la tarea %d no se entiende: %w", id, err)
	}

	if task.UpdatedAt, err = time.Parse(timeLayout, updated); err != nil {
		return Task{}, fmt.Errorf("la fecha de cambio de la tarea %d no se entiende: %w", id, err)
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
