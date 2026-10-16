package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"tasks-ai/internal/store"
	"time"
)

// DefaultPriority is the priority a task is born with.
const DefaultPriority = 2

// statusOpen is the state a task is born in.
const statusOpen = "open"

// Core applies the product's rules over one project.
type Core struct {
	store  *store.Store
	prefix string
}

// New wires the core over a project's data and the prefix its references are
// written with.
func New(project *store.Store, prefix string) *Core {
	return &Core{store: project, prefix: prefix}
}

// Prefix is the prefix this core composes visible references with.
func (c *Core) Prefix() string {
	return c.prefix
}

// Task is a task as it crosses the core's boundary: the internal number, so that
// whoever reads it has a key that does not change, and the visible reference,
// composed with the project's prefix.
type Task struct {
	ID          int64     `json:"id"`
	Ref         string    `json:"ref"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	Status      string    `json:"status"`
	Priority    int       `json:"priority"`
	Assignee    *string   `json:"assignee"`
	ParentID    *int64    `json:"parent_id"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CloseReason *string   `json:"close_reason"`
	Notes       *string   `json:"notes"`
}

// NewTask is what it takes to create a task. An empty text is an absent value.
type NewTask struct {
	Title       string
	Description string
	Assignee    string
	Notes       string
}

// CreateTask stores a new task and returns it as it crosses the boundary.
func (c *Core) CreateTask(ctx context.Context, task NewTask) (Task, error) {
	title := strings.TrimSpace(task.Title)
	if title == "" {
		return Task{}, errors.New("la tarea necesita un título")
	}

	stored, err := c.store.CreateTask(ctx, store.Task{
		Title:       title,
		Description: optional(task.Description),
		Status:      statusOpen,
		Priority:    DefaultPriority,
		Assignee:    optional(task.Assignee),
		Notes:       optional(task.Notes),
	})
	if err != nil {
		return Task{}, fmt.Errorf("guardando la tarea: %w", err)
	}

	return c.task(stored), nil
}

// Task returns the task the reference points at.
func (c *Core) Task(ctx context.Context, ref string) (Task, error) {
	id, err := ParseRef(c.prefix, ref)
	if err != nil {
		return Task{}, err
	}

	stored, err := c.store.Task(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return Task{}, fmt.Errorf("no hay ninguna tarea con la referencia %q", strings.TrimSpace(ref))
		}

		return Task{}, err
	}

	return c.task(stored), nil
}

// CheckPrefix makes the settings prefix and the base agree. The first time it
// records it, and when they differ it returns the warning that the references
// written with the previous prefix no longer resolve.
func (c *Core) CheckPrefix(ctx context.Context) (string, error) {
	recorded, err := c.store.RecordedPrefix(ctx)
	if err != nil {
		return "", err
	}

	switch recorded {
	case "":
		if err := c.store.SetPrefix(ctx, c.prefix); err != nil {
			return "", err
		}

		return "", nil
	case c.prefix:
		return "", nil
	default:
		return fmt.Sprintf(
			"la base se compuso con el prefijo %q y los ajustes dicen %q: las referencias escritas con el anterior ya no resuelven",
			recorded,
			c.prefix,
		), nil
	}
}

// task puts a stored task in the shape that crosses the boundary.
func (c *Core) task(stored store.Task) Task {
	return Task{
		ID:          stored.ID,
		Ref:         Ref(c.prefix, stored.ID),
		Title:       stored.Title,
		Description: stored.Description,
		Status:      stored.Status,
		Priority:    stored.Priority,
		Assignee:    stored.Assignee,
		ParentID:    stored.ParentID,
		Tags:        []string{},
		CreatedAt:   stored.CreatedAt.UTC(),
		UpdatedAt:   stored.UpdatedAt.UTC(),
		CloseReason: stored.CloseReason,
		Notes:       stored.Notes,
	}
}

// optional turns an empty text into an absent value.
func optional(text string) *string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}
