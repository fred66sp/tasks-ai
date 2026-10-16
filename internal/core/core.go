package core

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"tasks-ai/internal/store"
	"time"
)

// DefaultPriority is the priority a task is born with.
const DefaultPriority = 2

// The priority moves inside this range; a value outside it is rejected.
const (
	MinPriority = 0
	MaxPriority = 3
)

// ErrInvalid marks an entry that breaks a rule: the command line treats it as
// misuse of the command line, and the API will answer that the entry is invalid.
var ErrInvalid = errors.New("entrada inválida")

// invalidError is an entry that breaks a rule. It carries its own message so it
// reads well, and answers to ErrInvalid so whoever calls it knows what kind of
// error it is without reading the text.
type invalidError struct {
	err error
}

func (e *invalidError) Error() string { return e.err.Error() }

func (e *invalidError) Unwrap() error { return e.err }

func (e *invalidError) Is(target error) bool { return target == ErrInvalid }

// invalid builds the error of an entry that breaks a rule.
func invalid(format string, args ...any) error {
	return &invalidError{err: fmt.Errorf(format, args...)}
}

// statusOpen is the state a task is born in.
const statusOpen = "open"

// statusDone is the state a task is left in when it is closed.
const statusDone = "done"

// The reasons a task can be closed with.
const (
	CloseReasonDone       = "done"
	CloseReasonDuplicate  = "duplicate"
	CloseReasonAbsorbed   = "absorbed"
	CloseReasonStale      = "stale"
	CloseReasonSuperseded = "superseded"
)

// closeReasons lists the values a close reason may take.
var closeReasons = []string{
	CloseReasonDone,
	CloseReasonDuplicate,
	CloseReasonAbsorbed,
	CloseReasonStale,
	CloseReasonSuperseded,
}

// errStillOpen marks a reopen the rules refuse because the task is not closed.
var errStillOpen = errors.New("la tarea no está cerrada")

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

// NewTask is what it takes to create a task. An empty text is an absent value,
// a priority left out is the default one and a parent left out leaves the task
// on its own.
type NewTask struct {
	Title       string
	Description string
	Assignee    string
	Notes       string
	Priority    *int
	Parent      string
}

// CreateTask stores a new task and returns it as it crosses the boundary.
func (c *Core) CreateTask(ctx context.Context, task NewTask) (Task, error) {
	title, err := checkTitle(task.Title)
	if err != nil {
		return Task{}, err
	}

	priority := DefaultPriority
	if task.Priority != nil {
		priority = *task.Priority
	}

	if err := checkPriority(priority); err != nil {
		return Task{}, err
	}

	parentID, err := c.parentID(ctx, task.Parent)
	if err != nil {
		return Task{}, err
	}

	var storedParent *int64
	if parentID != 0 {
		storedParent = &parentID
	}

	stored, err := c.store.CreateTask(ctx, store.Task{
		Title:       title,
		Description: optional(task.Description),
		Status:      statusOpen,
		Priority:    priority,
		Assignee:    optional(task.Assignee),
		Notes:       optional(task.Notes),
		ParentID:    storedParent,
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
			return Task{}, c.missing(ref)
		}

		return Task{}, err
	}

	return c.task(stored), nil
}

// Changes is what an update asks to change. A field left out is not touched, and
// a text given empty clears the field. An update that asks for nothing is
// refused: it would touch the answer without touching the task.
type Changes struct {
	Title       *string
	Description *string
	Notes       *string
	Assignee    *string
	Priority    *int
}

// UpdateTask changes what the update asks for and returns the task as it is left.
func (c *Core) UpdateTask(ctx context.Context, ref string, changes Changes) (Task, error) {
	if changes.empty() {
		return Task{}, invalid("no hay nada que cambiar: indica al menos un campo")
	}

	id, err := ParseRef(c.prefix, ref)
	if err != nil {
		return Task{}, err
	}

	stored, err := c.store.UpdateTask(ctx, id, func(task store.Task) (store.Task, error) {
		return changed(task, changes)
	})
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return Task{}, c.missing(ref)
		}

		return Task{}, fmt.Errorf("cambiando la tarea: %w", err)
	}

	return c.task(stored), nil
}

// changed applies the fields of an update, checking the rules before anything is
// written. It must not ask the base for anything: it runs inside the transaction
// that holds the write lock.
func changed(task store.Task, changes Changes) (store.Task, error) {
	if changes.Title != nil {
		title, err := checkTitle(*changes.Title)
		if err != nil {
			return store.Task{}, err
		}

		task.Title = title
	}

	if changes.Description != nil {
		task.Description = optional(*changes.Description)
	}

	if changes.Notes != nil {
		task.Notes = optional(*changes.Notes)
	}

	if changes.Assignee != nil {
		task.Assignee = optional(*changes.Assignee)
	}

	if changes.Priority != nil {
		if err := checkPriority(*changes.Priority); err != nil {
			return store.Task{}, err
		}

		task.Priority = *changes.Priority
	}

	return task, nil
}

// empty says whether an update asks for nothing at all.
func (changes Changes) empty() bool {
	return changes.Title == nil &&
		changes.Description == nil &&
		changes.Notes == nil &&
		changes.Assignee == nil &&
		changes.Priority == nil
}

// CloseTask closes the task the reference points at with the reason given: it
// leaves its status in done, records the reason and refreshes the moment of
// change. The reason is required and must be one of the known ones; closing an
// already closed task and closing one with subtasks that are not done are both
// refused, and a refused close writes nothing.
func (c *Core) CloseTask(ctx context.Context, ref, reason string) (Task, error) {
	if err := checkCloseReason(reason); err != nil {
		return Task{}, err
	}

	id, err := ParseRef(c.prefix, ref)
	if err != nil {
		return Task{}, err
	}

	stored, err := c.store.CloseTask(ctx, id, reason)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			return Task{}, c.missing(ref)
		case errors.Is(err, store.ErrClosed):
			return Task{}, fmt.Errorf("no se puede cerrar la tarea %q: ya está cerrada", ref)
		case errors.Is(err, store.ErrOpenSubtasks):
			return Task{}, fmt.Errorf("no se puede cerrar la tarea %q: tiene subtareas abiertas", ref)
		default:
			return Task{}, fmt.Errorf("cerrando la tarea: %w", err)
		}
	}

	return c.task(stored), nil
}

// ReopenTask returns a closed task to open and clears its close reason.
// Reopening a task that is not closed is refused.
func (c *Core) ReopenTask(ctx context.Context, ref string) (Task, error) {
	id, err := ParseRef(c.prefix, ref)
	if err != nil {
		return Task{}, err
	}

	stored, err := c.store.UpdateTask(ctx, id, reopened)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			return Task{}, c.missing(ref)
		case errors.Is(err, errStillOpen):
			return Task{}, fmt.Errorf("no se puede reabrir la tarea %q: no está cerrada", ref)
		default:
			return Task{}, fmt.Errorf("reabriendo la tarea: %w", err)
		}
	}

	return c.task(stored), nil
}

// reopened applies the reopen inside the transaction that holds the write lock.
// It must not ask the base for anything: the task it receives was read there.
func reopened(task store.Task) (store.Task, error) {
	if task.Status != statusDone {
		return store.Task{}, errStillOpen
	}

	task.Status = statusOpen
	task.CloseReason = nil

	return task, nil
}

// parentID resolves the reference of the parent task, 0 when there is none. A
// parent that does not exist is refused the same way as a reference that
// resolves to no task.
func (c *Core) parentID(ctx context.Context, ref string) (int64, error) {
	if strings.TrimSpace(ref) == "" {
		return 0, nil
	}

	id, err := ParseRef(c.prefix, ref)
	if err != nil {
		return 0, err
	}

	if _, err := c.store.Task(ctx, id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return 0, c.missing(ref)
		}

		return 0, fmt.Errorf("buscando la tarea padre: %w", err)
	}

	return id, nil
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

// missing is the error of a reference that points at no task.
func (c *Core) missing(ref string) error {
	return fmt.Errorf("no hay ninguna tarea con la referencia %q", strings.TrimSpace(ref))
}

// checkTitle refuses an empty title and returns it trimmed.
func checkTitle(text string) (string, error) {
	title := strings.TrimSpace(text)
	if title == "" {
		return "", invalid("la tarea necesita un título")
	}

	return title, nil
}

// checkPriority refuses a priority outside its range.
func checkPriority(priority int) error {
	if priority < MinPriority || priority > MaxPriority {
		return invalid("la prioridad %d está fuera del rango %d-%d", priority, MinPriority, MaxPriority)
	}

	return nil
}

// optional turns an empty text into an absent value.
func optional(text string) *string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

// checkCloseReason refuses an absent or unknown close reason.
func checkCloseReason(reason string) error {
	if strings.TrimSpace(reason) == "" {
		return invalid("falta el motivo de cierre: usa --reason con uno de %s", strings.Join(closeReasons, ", "))
	}

	if slices.Contains(closeReasons, reason) {
		return nil
	}

	return invalid("el motivo de cierre %q no es válido: usa uno de %s", reason, strings.Join(closeReasons, ", "))
}
