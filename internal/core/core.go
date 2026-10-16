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

// ErrNotFound marks a reference that resolves to nothing stored: the API
// answers it with 404, and the command line with its generic error code.
var ErrNotFound = errors.New("no hay ninguna tarea ni etiqueta con esa referencia")

// ErrConflict marks an operation a rule of the product refuses: the API answers
// it with 409, and the command line with its generic error code.
var ErrConflict = errors.New("la regla del producto lo impide")

// Version is the version of the whole product, the same for both surfaces:
// they travel together in one binary and the API answers with it.
const Version = "0.1.0"

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

// notFoundError is a reference that resolves to nothing. It carries its own
// message and answers to ErrNotFound, igual que invalidError responde al suyo.
type notFoundError struct {
	err error
}

func (e *notFoundError) Error() string { return e.err.Error() }

func (e *notFoundError) Unwrap() error { return e.err }

func (e *notFoundError) Is(target error) bool { return target == ErrNotFound }

// notFound builds the error of a reference that resolves to nothing.
func notFound(format string, args ...any) error {
	return &notFoundError{err: fmt.Errorf(format, args...)}
}

// forbiddenError is an operation a rule of the product refuses. It carries its
// own message and answers to ErrConflict.
type forbiddenError struct {
	err error
}

func (e *forbiddenError) Error() string { return e.err.Error() }

func (e *forbiddenError) Unwrap() error { return e.err }

func (e *forbiddenError) Is(target error) bool { return target == ErrConflict }

// forbidden builds the error of an operation a rule refuses.
func forbidden(format string, args ...any) error {
	return &forbiddenError{err: fmt.Errorf(format, args...)}
}

// statusOpen is the state a task is born in.
const statusOpen = "open"

// statusDone is the state a task is left in when it is closed.
const statusDone = "done"

// statusInProgress is the state of a task whose work has started: solo se
// alcanza desde pendiente, y volver de él a pendiente es un cambio de estado
// más.
const statusInProgress = "in_progress"

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

// Status is the state of the product at a glance: its version, the version of
// the schema its base is at and the prefix its references are written with.
type Status struct {
	Version string `json:"version"`
	Schema  int    `json:"schema"`
	Prefix  string `json:"prefix"`
}

// Status reads the state of the product: the API answers it as it is.
func (c *Core) Status(ctx context.Context) (Status, error) {
	schema, err := c.store.SchemaVersion(ctx)
	if err != nil {
		return Status{}, err
	}

	return Status{Version: Version, Schema: schema, Prefix: c.prefix}, nil
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
	Blocked     bool      `json:"blocked"`
	Priority    int       `json:"priority"`
	Assignee    *string   `json:"assignee"`
	ParentID    *int64    `json:"parent_id"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CloseReason *string   `json:"close_reason"`
	Notes       *string   `json:"notes"`
}

// Comment is a comment as it crosses the boundary: its body and the moment it
// was added at. Comments are only added; nothing changes one afterwards.
type Comment struct {
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// Detail is the task with everything around it: its children, its blockers, its
// dependents and its comments. The lists carry the shape the commands already
// answer with, so a list entry carries no detail of its own.
type Detail struct {
	Task
	Children   []Task    `json:"children"`
	Blockers   []Task    `json:"blockers"`
	Dependents []Task    `json:"dependents"`
	Comments   []Comment `json:"comments"`
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

	return c.task(ctx, stored)
}

// Tasks lists the tasks of the project, by the identity the base gave them. La
// línea de comandos no lista: el listado es de la API, y lee el mismo núcleo.
func (c *Core) Tasks(ctx context.Context) ([]Task, error) {
	stored, err := c.store.Tasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("leyendo las tareas: %w", err)
	}

	return c.tasks(ctx, stored)
}

// Event is a change of the project as it crosses the boundary: el aviso viajó
// por el número interno, y aquí se compone la referencia que se muestra ahora.
type Event struct {
	ID       int64     `json:"id"`
	Resource string    `json:"resource"`
	Ref      string    `json:"ref"`
	Origin   string    `json:"origin"`
	At       time.Time `json:"at"`
}

// Events reads the changes that happened after the notice id given, with the
// reference of each changed resource composed the way it is shown now: quien
// lee continúa donde se quedó pasando el último aviso que vio.
func (c *Core) Events(ctx context.Context, after int64) ([]Event, error) {
	changes, err := c.store.Changes(ctx, after)
	if err != nil {
		return nil, fmt.Errorf("leyendo los avisos de cambio: %w", err)
	}

	events := make([]Event, 0, len(changes))

	for _, change := range changes {
		event := Event{
			ID:       change.ID,
			Resource: change.Resource,
			Origin:   change.Origin,
			At:       change.HappenedAt.UTC(),
		}

		if event.Ref, err = c.referenceOf(ctx, change); err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	return events, nil
}

// referenceOf composes the reference a notice speaks of: la tarea se nombra con
// el prefijo de ahora, y la etiqueta con el nombre que lleva ahora.
func (c *Core) referenceOf(ctx context.Context, change store.Change) (string, error) {
	switch change.Resource {
	case store.ResourceTask:
		return Ref(c.prefix, change.ResourceID), nil
	case store.ResourceTag:
		name, err := c.store.TagName(ctx, change.ResourceID)
		if errors.Is(err, store.ErrNotFound) {
			// La etiqueta se borró: su aviso no tiene nombre que mostrar.
			return "", nil
		}

		if err != nil {
			return "", fmt.Errorf("componiendo el aviso de la etiqueta %d: %w", change.ResourceID, err)
		}

		return name, nil
	default:
		return "", fmt.Errorf("el aviso %d habla del recurso %q, que no se entiende", change.ID, change.Resource)
	}
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

	return c.task(ctx, stored)
}

// Changes is what an update asks to change. A field left out is not touched, and
// a text given empty clears the field; the tags travel as the whole list of
// names, and an empty list leaves the task without any. An update that asks for
// nothing is refused: it would touch the answer without touching the task.
type Changes struct {
	Title       *string
	Description *string
	Notes       *string
	Assignee    *string
	Priority    *int
	Tags        *[]string
	Status      *string
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

	var tags []string
	if changes.Tags != nil {
		tags = *changes.Tags
	}

	stored, err := c.store.UpdateTask(ctx, id, func(task store.Task) (store.Task, error) {
		return changed(task, changes)
	}, tags)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			return Task{}, c.missing(ref)
		case errors.Is(err, store.ErrUnknownTag):
			return Task{}, invalid("%v", err)
		case errors.Is(err, ErrInvalid), errors.Is(err, ErrConflict):
			// La regla que el cambio ha pisado ya trae su mensaje: sale tal
			// cual, sin la envoltura de la operación.
			return Task{}, err
		default:
			return Task{}, fmt.Errorf("cambiando la tarea: %w", err)
		}
	}

	return c.task(ctx, stored)
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

	if changes.Status != nil {
		if err := applyStatus(&task, *changes.Status); err != nil {
			return store.Task{}, err
		}
	}

	return task, nil
}

// applyStatus applies the status a change asks for, under the product's rules:
// ponerse en curso solo cabe desde pendiente, volver a pendiente es reabrir —
// le quita el motivo de cierre— y cerrar no va por el cambio: lleva su motivo y
// va por su propia operación.
func applyStatus(task *store.Task, status string) error {
	switch status {
	case statusOpen:
		task.Status = statusOpen
		task.CloseReason = nil
	case statusInProgress:
		if task.Status != statusOpen {
			return forbidden("solo una tarea pendiente se puede poner en curso")
		}

		task.Status = statusInProgress
	case statusDone:
		return invalid("el estado %q no se pone: cerrar va por close, con el motivo", statusDone)
	default:
		return invalid("el estado %q no se entiende: usa %s o %s", status, statusOpen, statusInProgress)
	}

	return nil
}

// empty says whether an update asks for nothing at all.
func (changes Changes) empty() bool {
	return changes.Title == nil &&
		changes.Description == nil &&
		changes.Notes == nil &&
		changes.Assignee == nil &&
		changes.Priority == nil &&
		changes.Tags == nil &&
		changes.Status == nil
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
			return Task{}, forbidden("no se puede cerrar la tarea %q: ya está cerrada", ref)
		case errors.Is(err, store.ErrOpenSubtasks):
			return Task{}, forbidden("no se puede cerrar la tarea %q: tiene subtareas abiertas", ref)
		default:
			return Task{}, fmt.Errorf("cerrando la tarea: %w", err)
		}
	}

	return c.task(ctx, stored)
}

// ReopenTask returns a closed task to open and clears its close reason.
// Reopening a task that is not closed is refused.
func (c *Core) ReopenTask(ctx context.Context, ref string) (Task, error) {
	id, err := ParseRef(c.prefix, ref)
	if err != nil {
		return Task{}, err
	}

	stored, err := c.store.UpdateTask(ctx, id, reopened, nil)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			return Task{}, c.missing(ref)
		case errors.Is(err, errStillOpen):
			return Task{}, forbidden("no se puede reabrir la tarea %q: no está cerrada", ref)
		default:
			return Task{}, fmt.Errorf("reabriendo la tarea: %w", err)
		}
	}

	return c.task(ctx, stored)
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

	return c.taskID(ctx, ref)
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

// task puts a stored task in the shape that crosses the boundary, with the
// blocked state computed from the dependencies: it is true when the task has a
// blocker that is not done.
func (c *Core) task(ctx context.Context, stored store.Task) (Task, error) {
	blocked, err := c.store.HasOpenBlocker(ctx, stored.ID)
	if err != nil {
		return Task{}, err
	}

	names, err := c.store.TaskTagNames(ctx, stored.ID)
	if err != nil {
		return Task{}, err
	}

	shaped := Task{
		ID:          stored.ID,
		Ref:         Ref(c.prefix, stored.ID),
		Title:       stored.Title,
		Description: stored.Description,
		Status:      stored.Status,
		Priority:    stored.Priority,
		Assignee:    stored.Assignee,
		ParentID:    stored.ParentID,
		Tags:        names,
		CreatedAt:   stored.CreatedAt.UTC(),
		UpdatedAt:   stored.UpdatedAt.UTC(),
		CloseReason: stored.CloseReason,
		Notes:       stored.Notes,
	}

	shaped.Blocked = blocked

	return shaped, nil
}

// tasks puts the stored tasks in the shape that crosses the boundary, in the
// order they were read.
func (c *Core) tasks(ctx context.Context, stored []store.Task) ([]Task, error) {
	shaped := make([]Task, 0, len(stored))

	for i := range stored {
		task, err := c.task(ctx, stored[i])
		if err != nil {
			return nil, err
		}

		shaped = append(shaped, task)
	}

	return shaped, nil
}

// TaskDetail returns the task the reference points at with everything around
// it: its children, its blockers, its dependents and its comments.
func (c *Core) TaskDetail(ctx context.Context, ref string) (Detail, error) {
	id, err := ParseRef(c.prefix, ref)
	if err != nil {
		return Detail{}, err
	}

	stored, err := c.store.Task(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return Detail{}, c.missing(ref)
		}

		return Detail{}, err
	}

	children, err := c.store.Children(ctx, id)
	if err != nil {
		return Detail{}, fmt.Errorf("buscando las hijas de la tarea %q: %w", ref, err)
	}

	blockers, err := c.store.Blockers(ctx, id)
	if err != nil {
		return Detail{}, fmt.Errorf("buscando los bloqueantes de la tarea %q: %w", ref, err)
	}

	dependents, err := c.store.Dependents(ctx, id)
	if err != nil {
		return Detail{}, fmt.Errorf("buscando los dependientes de la tarea %q: %w", ref, err)
	}

	comments, err := c.store.Comments(ctx, id)
	if err != nil {
		return Detail{}, fmt.Errorf("buscando los comentarios de la tarea %q: %w", ref, err)
	}

	task, err := c.task(ctx, stored)
	if err != nil {
		return Detail{}, err
	}

	childrenShaped, err := c.tasks(ctx, children)
	if err != nil {
		return Detail{}, err
	}

	blockersShaped, err := c.tasks(ctx, blockers)
	if err != nil {
		return Detail{}, err
	}

	dependentsShaped, err := c.tasks(ctx, dependents)
	if err != nil {
		return Detail{}, err
	}

	return Detail{
		Task:       task,
		Children:   childrenShaped,
		Blockers:   blockersShaped,
		Dependents: dependentsShaped,
		Comments:   commented(comments),
	}, nil
}

// commented puts the stored comments in the shape that crosses the boundary.
func commented(stored []store.Comment) []Comment {
	comments := make([]Comment, 0, len(stored))

	for i := range stored {
		comments = append(comments, Comment{
			Body:      stored[i].Body,
			CreatedAt: stored[i].CreatedAt.UTC(),
		})
	}

	return comments
}

// BlockTask leaves the first task blocked by the second: one row per pair, and
// the blocked state is computed from the rows, never stored. A reference that
// resolves to no task is refused the way Task refuses it, a task cannot block
// itself and cycles are allowed.
func (c *Core) BlockTask(ctx context.Context, ref, blockerRef string) (Task, error) {
	taskID, blockerID, err := c.pair(ctx, ref, blockerRef)
	if err != nil {
		return Task{}, err
	}

	if err := c.store.AddBlock(ctx, taskID, blockerID); err != nil {
		return Task{}, fmt.Errorf("bloqueando la tarea %q: %w", ref, err)
	}

	return c.Task(ctx, ref)
}

// UnblockTask removes the block that leaves the first task blocked by the
// second. The same refusals of BlockTask apply, and unblocking a pair that is
// not leaves everything as it was.
func (c *Core) UnblockTask(ctx context.Context, ref, blockerRef string) (Task, error) {
	taskID, blockerID, err := c.pair(ctx, ref, blockerRef)
	if err != nil {
		return Task{}, err
	}

	if err := c.store.RemoveBlock(ctx, taskID, blockerID); err != nil {
		return Task{}, fmt.Errorf("retirando el bloqueo de la tarea %q: %w", ref, err)
	}

	return c.Task(ctx, ref)
}

// AddComment appends a comment to the task: comments are added, and that is
// all, there is no way to change one afterwards. The body is required.
func (c *Core) AddComment(ctx context.Context, ref, body string) (Task, error) {
	text := strings.TrimSpace(body)
	if text == "" {
		return Task{}, invalid("falta el cuerpo del comentario: usa --body")
	}

	id, err := c.taskID(ctx, ref)
	if err != nil {
		return Task{}, err
	}

	if _, err := c.store.AddComment(ctx, id, text); err != nil {
		return Task{}, fmt.Errorf("añadiendo el comentario a la tarea %q: %w", ref, err)
	}

	return c.Task(ctx, ref)
}

// pair resolves the two references a block speaks of: both must point at tasks
// that exist, and a task cannot be blocked by itself.
func (c *Core) pair(ctx context.Context, ref, blockerRef string) (int64, int64, error) {
	taskID, err := c.taskID(ctx, ref)
	if err != nil {
		return 0, 0, err
	}

	blockerID, err := c.taskID(ctx, blockerRef)
	if err != nil {
		return 0, 0, err
	}

	if taskID == blockerID {
		return 0, 0, invalid("la tarea %q no se puede bloquear por sí misma", ref)
	}

	return taskID, blockerID, nil
}

// taskID resolves the reference to the internal number of the task it points
// at. A reference that resolves to no task is refused the way Task refuses it.
func (c *Core) taskID(ctx context.Context, ref string) (int64, error) {
	id, err := ParseRef(c.prefix, ref)
	if err != nil {
		return 0, err
	}

	if _, err := c.store.Task(ctx, id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return 0, c.missing(ref)
		}

		return 0, fmt.Errorf("buscando la tarea %q: %w", ref, err)
	}

	return id, nil
}

// missing is the error of a reference that points at no task.
func (c *Core) missing(ref string) error {
	return notFound("no hay ninguna tarea con la referencia %q", strings.TrimSpace(ref))
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
