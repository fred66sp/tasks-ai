package store

import (
	"context"
	"database/sql"
	"fmt"
)

// AddBlock leaves the task blocked by the blocker: one row per pair. Blocking
// the same pair again changes nothing, and no change leaves no notice: solo se
// anota el bloqueo que de verdad se escribió.
func (s *Store) AddBlock(ctx context.Context, taskID, blockerID int64) error {
	return s.changeBlock(ctx, taskID, blockerID, `
		INSERT INTO dependencies (task_id, blocker_id) VALUES (?, ?)
		ON CONFLICT (task_id, blocker_id) DO NOTHING`,
		"bloqueando la tarea %d por la tarea %d")
}

// RemoveBlock drops the row of the pair. Nothing is there to drop when the pair
// is not, and then it changes nothing either: sin cambio no hay aviso.
func (s *Store) RemoveBlock(ctx context.Context, taskID, blockerID int64) error {
	return s.changeBlock(ctx, taskID, blockerID,
		`DELETE FROM dependencies WHERE task_id = ? AND blocker_id = ?`,
		"retirando el bloqueo de la tarea %d por la tarea %d")
}

// changeBlock abre la transacción del bloqueo, ejecuta la instrucción que lo
// añade o lo retira y anota el aviso solo cuando la instrucción escribió algo.
// El texto del fallo lo da quien llama, porque es el que nombra la operación.
func (s *Store) changeBlock(ctx context.Context, taskID, blockerID int64, statement, failure string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("abriendo la transacción del bloqueo: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(ctx, statement, taskID, blockerID)
	if err != nil {
		return fmt.Errorf(failure+": %w", taskID, blockerID, err)
	}

	if err := recordWhenWritten(ctx, tx, s.origin, result, taskID); err != nil {
		return err
	}

	return tx.Commit()
}

// recordWhenWritten deja el aviso del cambio solo cuando la instrucción tocó
// alguna fila: repetir un bloqueo o retirar el que ya no está no cambia nada y
// no deja aviso.
func recordWhenWritten(ctx context.Context, tx *sql.Tx, origin string, result sql.Result, taskID int64) error {
	written, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mirando lo que escribió el bloqueo: %w", err)
	}

	if written == 0 {
		return nil
	}

	return recordChange(ctx, tx, origin, ResourceTask, taskID)
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
