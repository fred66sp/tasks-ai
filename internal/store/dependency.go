package store

import (
	"context"
	"fmt"
)

// AddBlock leaves the task blocked by the blocker: one row per pair. Blocking
// the same pair again changes nothing.
func (s *Store) AddBlock(ctx context.Context, taskID, blockerID int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO dependencies (task_id, blocker_id) VALUES (?, ?)
		ON CONFLICT (task_id, blocker_id) DO NOTHING`, taskID, blockerID)
	if err != nil {
		return fmt.Errorf("bloqueando la tarea %d por la tarea %d: %w", taskID, blockerID, err)
	}

	return nil
}

// RemoveBlock drops the row of the pair. Nothing is there to drop when the pair
// is not, and then it changes nothing either.
func (s *Store) RemoveBlock(ctx context.Context, taskID, blockerID int64) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM dependencies WHERE task_id = ? AND blocker_id = ?`, taskID, blockerID)
	if err != nil {
		return fmt.Errorf("retirando el bloqueo de la tarea %d por la tarea %d: %w", taskID, blockerID, err)
	}

	return nil
}

// HasOpenBlocker tells whether the task has a blocker that is not done: the
// blocked state is computed from the rows and is not stored.
func (s *Store) HasOpenBlocker(ctx context.Context, id int64) (bool, error) {
	var blocked bool

	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM dependencies d
			JOIN tasks t ON t.id = d.blocker_id
			WHERE d.task_id = ? AND t.status <> ?
		)`, id, statusDone).Scan(&blocked)
	if err != nil {
		return false, fmt.Errorf("mirando los bloqueos de la tarea %d: %w", id, err)
	}

	return blocked, nil
}

// Blockers are the tasks that block the one asked for, by their identity.
func (s *Store) Blockers(ctx context.Context, id int64) ([]Task, error) {
	return s.queryTasks(ctx, `
		SELECT `+taskColumns+`
		FROM tasks t
		JOIN dependencies d ON d.blocker_id = t.id
		WHERE d.task_id = ?
		ORDER BY t.id`, id)
}

// Dependents are the tasks the one asked for leaves blocked.
func (s *Store) Dependents(ctx context.Context, id int64) ([]Task, error) {
	return s.queryTasks(ctx, `
		SELECT `+taskColumns+`
		FROM tasks t
		JOIN dependencies d ON d.task_id = t.id
		WHERE d.blocker_id = ?
		ORDER BY t.id`, id)
}

// Children are the tasks born from the one asked for.
func (s *Store) Children(ctx context.Context, id int64) ([]Task, error) {
	return s.queryTasks(ctx,
		`SELECT `+taskColumns+` FROM tasks WHERE parent_id = ? ORDER BY id`, id)
}
