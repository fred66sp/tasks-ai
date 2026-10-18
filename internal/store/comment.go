package store

import (
	"context"
	"database/sql"
	"errors"
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

// AddComment appends a comment to the task and returns it with the identity the
// base gave it, which is the number EditComment later names it by. El comentario
// y el aviso de su llegada se escriben en la misma transacción.
func (s *Store) AddComment(ctx context.Context, taskID int64, body string) (Comment, error) {
	now := time.Now().UTC().Truncate(time.Second)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Comment{}, fmt.Errorf("opening the comment transaction: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(ctx,
		`INSERT INTO comments (task_id, body, created_at) VALUES (?, ?, ?)`,
		taskID, body, now.Format(timeLayout))
	if err != nil {
		return Comment{}, fmt.Errorf("adding a comment to task %d: %w", taskID, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Comment{}, fmt.Errorf("reading the identity of the added comment: %w", err)
	}

	// Un comentario es un cambio de la tarea que comenta: su aviso lo dice.
	if err := recordChange(ctx, tx, s.origin, ResourceTask, taskID, ActionUpdate); err != nil {
		return Comment{}, err
	}

	if err := tx.Commit(); err != nil {
		return Comment{}, fmt.Errorf("closing the comment transaction: %w", err)
	}

	return Comment{ID: id, TaskID: taskID, Body: body, CreatedAt: now}, nil
}

// EditComment changes the body of the comment of the task that carries the
// number given. The identity, the task it speaks of and the moment it was added
// at are kept from the row that was read: only the body changes. Comment and
// notice are written in the same transaction, so an edit that is refused leaves
// nothing behind. The number is understood inside the task, so a comment that is
// not one of the task's comes back as ErrNotFound.
func (s *Store) EditComment(ctx context.Context, taskID, commentID int64, body string) (Comment, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Comment{}, fmt.Errorf("opening the comment change transaction: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	var (
		stored    Comment
		createdAt string
	)

	err = tx.QueryRowContext(ctx, `
		SELECT id, task_id, body, created_at
		FROM comments
		WHERE id = ? AND task_id = ?`, commentID, taskID,
	).Scan(&stored.ID, &stored.TaskID, &stored.Body, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Comment{}, ErrNotFound
	}

	if err != nil {
		return Comment{}, fmt.Errorf("looking up comment %d of task %d: %w", commentID, taskID, err)
	}

	if stored.CreatedAt, err = time.Parse(timeLayout, createdAt); err != nil {
		return Comment{}, fmt.Errorf("the date of comment %d cannot be read: %w", stored.ID, err)
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE comments SET body = ? WHERE id = ? AND task_id = ?`,
		body, commentID, taskID); err != nil {
		return Comment{}, fmt.Errorf("saving the change of comment %d: %w", commentID, err)
	}

	// Un comentario es un cambio de la tarea que comenta: su aviso lo dice, igual
	// que cuando llega.
	if err := recordChange(ctx, tx, s.origin, ResourceTask, taskID, ActionUpdate); err != nil {
		return Comment{}, err
	}

	if err := tx.Commit(); err != nil {
		return Comment{}, fmt.Errorf("closing the comment change transaction of %d: %w", commentID, err)
	}

	stored.Body = body

	return stored, nil
}

// Comments are the comments of the task, from the first to the last.
func (s *Store) Comments(ctx context.Context, taskID int64) ([]Comment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, task_id, body, created_at
		FROM comments
		WHERE task_id = ?
		ORDER BY created_at, id`, taskID)
	if err != nil {
		return nil, fmt.Errorf("reading the comments of task %d: %w", taskID, err)
	}

	defer func() { _ = rows.Close() }()

	comments := []Comment{}

	for rows.Next() {
		var (
			comment   Comment
			createdAt string
		)

		if err := rows.Scan(&comment.ID, &comment.TaskID, &comment.Body, &createdAt); err != nil {
			return nil, fmt.Errorf("reading the comments of task %d: %w", taskID, err)
		}

		if comment.CreatedAt, err = time.Parse(timeLayout, createdAt); err != nil {
			return nil, fmt.Errorf("the date of comment %d cannot be read: %w", comment.ID, err)
		}

		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading the comments of task %d: %w", taskID, err)
	}

	return comments, nil
}
