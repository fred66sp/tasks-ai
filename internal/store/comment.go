package store

import (
	"context"
	"fmt"
	"time"
)

// Comment is a comment as it is stored: the task it speaks of, its body and the
// moment it was added at.
type Comment struct {
	ID        int64
	TaskID    int64
	Body      string
	CreatedAt time.Time
}

// AddComment appends a comment to the task. Comments are only added: there is
// no way to change the one that is already there. El comentario y el aviso de
// su llegada se escriben en la misma transacción.
func (s *Store) AddComment(ctx context.Context, taskID int64, body string) (Comment, error) {
	now := time.Now().UTC().Truncate(time.Second)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Comment{}, fmt.Errorf("abriendo la transacción del comentario: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(ctx,
		`INSERT INTO comments (task_id, body, created_at) VALUES (?, ?, ?)`,
		taskID, body, now.Format(timeLayout))
	if err != nil {
		return Comment{}, fmt.Errorf("añadiendo un comentario a la tarea %d: %w", taskID, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Comment{}, fmt.Errorf("leyendo la identidad del comentario añadido: %w", err)
	}

	if err := recordChange(ctx, tx, s.origin, ResourceTask, taskID); err != nil {
		return Comment{}, err
	}

	if err := tx.Commit(); err != nil {
		return Comment{}, fmt.Errorf("cerrando la transacción del comentario: %w", err)
	}

	return Comment{ID: id, TaskID: taskID, Body: body, CreatedAt: now}, nil
}

// Comments are the comments of the task, from the first to the last.
func (s *Store) Comments(ctx context.Context, taskID int64) ([]Comment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, task_id, body, created_at
		FROM comments
		WHERE task_id = ?
		ORDER BY created_at, id`, taskID)
	if err != nil {
		return nil, fmt.Errorf("leyendo los comentarios de la tarea %d: %w", taskID, err)
	}

	defer func() { _ = rows.Close() }()

	comments := []Comment{}

	for rows.Next() {
		var (
			comment   Comment
			createdAt string
		)

		if err := rows.Scan(&comment.ID, &comment.TaskID, &comment.Body, &createdAt); err != nil {
			return nil, fmt.Errorf("leyendo los comentarios de la tarea %d: %w", taskID, err)
		}

		if comment.CreatedAt, err = time.Parse(timeLayout, createdAt); err != nil {
			return nil, fmt.Errorf("la fecha del comentario %d no se entiende: %w", comment.ID, err)
		}

		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("leyendo los comentarios de la tarea %d: %w", taskID, err)
	}

	return comments, nil
}
