// Package aif serves the AIF 0.2 CLI profile over the product's core: the
// vocabulary of tools of docs/contracts.md §7, each adapter calling the same
// core the command line calls, and classifying its error by what it means.
package aif

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"tasks-ai/internal/core"

	interaction "github.com/fred66sp/agent-interaction-forms"
)

// The members of the operations the profile serves. They repeat the names the
// tools of contracts.md §7 already use.
const (
	fieldRef         = "ref"
	fieldTitle       = "title"
	fieldDescription = "description"
	fieldNotes       = "notes"
	fieldAssignee    = "assignee"
	fieldPriority    = "priority"
	fieldStatus      = "status"
	fieldTags        = "tags"
	fieldParent      = "parent"
	fieldReason      = "reason"
	fieldBlocker     = "blocker"
	fieldBody        = "body"
	fieldID          = "id"
)

// refDescription is how every operation describes the reference it asks for.
const refDescription = "la referencia visible de la tarea, por ejemplo tk-18"

// textType is the JSON type of the text members.
const textType = "string"

// New builds the executor of the vocabulary of docs/contracts.md §7 over the
// core of one project. The operations are read and written through the same
// core as the command line: nothing of the product's rules lives here.
func New(apply *core.Core) *interaction.Executor {
	return &interaction.Executor{Operations: []*interaction.Operation{
		boardOperation(apply),
		showOperation(apply),
		createOperation(apply),
		updateOperation(apply),
		closeOperation(apply),
		reopenOperation(apply),
		blockOperation(apply),
		unblockOperation(apply),
		commentOperation(apply),
		commentListOperation(apply),
		commentEditOperation(apply),
	}}
}

// boardOperation projects the board the agent decides with. The projection and
// its clipping belong to docs/contracts.md §7, not to this adapter.
func boardOperation(apply *core.Core) *interaction.Operation {
	return &interaction.Operation{
		ID:          "tasks_board",
		Description: "El tablero del proyecto: cuánto hay abierto, qué está en curso, qué se puede empezar y el árbol de tareas.",
		Execute: func(map[string]json.RawMessage) (json.RawMessage, error) {
			tasks, err := apply.Tasks(context.Background())
			if err != nil {
				return nil, repair(err, fieldRef)
			}

			return outcome(board(tasks))
		},
	}
}

// showOperation returns one task with everything around it, as the product's
// show answers it: the detail is the core's, not a second projection.
func showOperation(apply *core.Core) *interaction.Operation {
	return &interaction.Operation{
		ID:          "tasks_show",
		Description: "Una tarea del proyecto con su descripción, sus hijas, sus bloqueos y sus últimos comentarios. Es la puerta al detalle: úsala cuando el tablero no baste.",
		Fields:      []interaction.Field{refField()},
		Execute: func(input map[string]json.RawMessage) (json.RawMessage, error) {
			detail, err := apply.TaskDetail(context.Background(), textOf(input, fieldRef))
			if err != nil {
				return nil, repair(err, fieldRef)
			}

			return outcome(detail)
		},
	}
}

// createOperation stores a task with the fields the caller names; the defaults
// and the rules are the core's.
func createOperation(apply *core.Core) *interaction.Operation {
	return &interaction.Operation{
		ID:          "tasks_create",
		Description: "Crea una tarea; usa parent para crear una subtarea de una tarea existente y tags para darle las etiquetas desde el primer momento. Sin assignee, la tarea nace con la persona del proyecto.",
		Fields: []interaction.Field{
			titleField(),
			textField(fieldDescription, "la descripción de la tarea"),
			textField(fieldNotes, "las notas de la tarea"),
			textField(fieldAssignee, "el responsable de la tarea; sin él, la persona del proyecto"),
			priorityField(),
			tagsField(),
			parentField(),
		},
		Execute: func(input map[string]json.RawMessage) (json.RawMessage, error) {
			priority, err := priorityOf(input)
			if err != nil {
				return nil, err
			}

			names, _, err := tagsOf(input)
			if err != nil {
				return nil, err
			}

			task, err := apply.CreateTask(context.Background(), core.NewTask{
				Title:       textOf(input, fieldTitle),
				Description: textOf(input, fieldDescription),
				Notes:       textOf(input, fieldNotes),
				Assignee:    textOf(input, fieldAssignee),
				Priority:    priority,
				Parent:      textOf(input, fieldParent),
				Tags:        names,
			})
			if err != nil {
				return nil, repair(err, fieldTitle)
			}

			return outcome(task)
		},
	}
}

// updateOperation changes only the fields the caller names: a member left out
// does not change, and an empty text clears its field.
func updateOperation(apply *core.Core) *interaction.Operation {
	return &interaction.Operation{
		ID:          "tasks_update",
		Description: "Cambia solo los campos indicados de una tarea. Usa status solo; tags sustituye todas sus etiquetas; parent la asigna, la mueve o, con el texto vacío, la deja sin padre. Al empezar el trabajo en una tarea, ponla en curso con status in_progress; al dejarla, devuélvela con status open.",
		Fields: []interaction.Field{
			refField(),
			textField(fieldTitle, "el nuevo título de la tarea"),
			textField(fieldDescription, "la nueva descripción, vacía la retira"),
			textField(fieldNotes, "las nuevas notas, vacías las retiran"),
			textField(fieldAssignee, "el nuevo responsable, vacío lo retira"),
			priorityField(),
			statusField(),
			tagsField(),
			parentField(),
		},
		Execute: func(input map[string]json.RawMessage) (json.RawMessage, error) {
			changes, err := changesOf(input)
			if err != nil {
				return nil, err
			}

			task, err := apply.UpdateTask(context.Background(), textOf(input, fieldRef), changes)
			if err != nil {
				// Un cambio que no pide nada no tiene campo culpable: la
				// referencia es el único miembro que todo cambio lleva, y es
				// la petición misma la que hay que reparar.
				return nil, repair(err, fieldRef)
			}

			return outcome(task)
		},
	}
}

// closeOperation closes a task with its reason; the reason comes from the
// product's own list.
func closeOperation(apply *core.Core) *interaction.Operation {
	return &interaction.Operation{
		ID:          "tasks_close",
		Description: "Cierra una tarea con un motivo cuando termina o deja de ser necesaria; sus subtareas deben estar cerradas. Con done, cierra solo después de presentar el resultado a la persona: con su revisión o su pedido expreso; una tarea pendiente no se cierra como done.",
		Fields:      []interaction.Field{refField(), reasonField()},
		Execute: func(input map[string]json.RawMessage) (json.RawMessage, error) {
			task, err := apply.CloseTask(
				context.Background(),
				textOf(input, fieldRef),
				textOf(input, fieldReason),
			)
			if err != nil {
				return nil, repair(err, fieldReason)
			}

			return outcome(task)
		},
	}
}

// reopenOperation returns a closed task to open and clears its close reason.
func reopenOperation(apply *core.Core) *interaction.Operation {
	return &interaction.Operation{
		ID:          "tasks_reopen",
		Description: "Reabre una tarea cerrada cuando vuelve a necesitar trabajo y retira su motivo de cierre.",
		Fields:      []interaction.Field{refField()},
		Execute: func(input map[string]json.RawMessage) (json.RawMessage, error) {
			task, err := apply.ReopenTask(context.Background(), textOf(input, fieldRef))
			if err != nil {
				return nil, repair(err, fieldRef)
			}

			return outcome(task)
		},
	}
}

// blockOperation leaves a task blocked by another one.
func blockOperation(apply *core.Core) *interaction.Operation {
	return &interaction.Operation{
		ID:          "tasks_block",
		Description: "Añade una dependencia cuando una tarea debe esperar a otra; blocker es la tarea que la bloquea.",
		Fields:      []interaction.Field{refField(), blockerField()},
		Execute: func(input map[string]json.RawMessage) (json.RawMessage, error) {
			task, err := apply.BlockTask(
				context.Background(),
				textOf(input, fieldRef),
				textOf(input, fieldBlocker),
			)
			if err != nil {
				// Una tarea que se bloquea a sí misma no tiene un campo que
				// rompa una regla: el que decide es el bloqueante, que debe
				// ser otra tarea.
				return nil, repair(err, fieldBlocker)
			}

			return outcome(task)
		},
	}
}

// unblockOperation removes a block that leaves a task waiting for another.
func unblockOperation(apply *core.Core) *interaction.Operation {
	return &interaction.Operation{
		ID:          "tasks_unblock",
		Description: "Retira la dependencia de blocker cuando una tarea ya no debe esperar a esa otra tarea.",
		Fields:      []interaction.Field{refField(), blockerField()},
		Execute: func(input map[string]json.RawMessage) (json.RawMessage, error) {
			task, err := apply.UnblockTask(
				context.Background(),
				textOf(input, fieldRef),
				textOf(input, fieldBlocker),
			)
			if err != nil {
				return nil, repair(err, fieldBlocker)
			}

			return outcome(task)
		},
	}
}

// commentOperation appends a comment to a task. The body's rule is the core's:
// empty or writing the state of Git is refused before anything is written.
func commentOperation(apply *core.Core) *interaction.Operation {
	return &interaction.Operation{
		ID:          "tasks_comment",
		Description: "Añade un comentario para dejar avances o contexto en una tarea; para leerlos tasks_comment_list y para cambiar el cuerpo de uno tasks_comment_edit. Escribe el avance, la decisión o el bloqueo en el momento en que surgen, no al final.",
		Fields:      []interaction.Field{refField(), bodyField("el texto no vacío del comentario")},
		Execute: func(input map[string]json.RawMessage) (json.RawMessage, error) {
			task, err := apply.AddComment(
				context.Background(),
				textOf(input, fieldRef),
				textOf(input, fieldBody),
			)
			if err != nil {
				return nil, repair(err, fieldBody)
			}

			return outcome(task)
		},
	}
}

// commentListOperation reads the comments of a task, from the first to the last.
func commentListOperation(apply *core.Core) *interaction.Operation {
	return &interaction.Operation{
		ID:          "tasks_comment_list",
		Description: "Lee los comentarios de una tarea, del primero al último, cada uno con su identidad, su cuerpo y su fecha.",
		Fields:      []interaction.Field{refField()},
		Execute: func(input map[string]json.RawMessage) (json.RawMessage, error) {
			comments, err := apply.Comments(context.Background(), textOf(input, fieldRef))
			if err != nil {
				return nil, repair(err, fieldRef)
			}

			return outcome(comments)
		},
	}
}

// commentEditOperation changes the body of one comment of a task, keeping its
// identity and its date.
func commentEditOperation(apply *core.Core) *interaction.Operation {
	return &interaction.Operation{
		ID:          "tasks_comment_edit",
		Description: "Cambia el cuerpo del comentario de una tarea que lleva el número dado; la identidad y la fecha del comentario se conservan.",
		Fields:      []interaction.Field{refField(), idField(), bodyField("el nuevo texto del comentario")},
		Execute: func(input map[string]json.RawMessage) (json.RawMessage, error) {
			id, ok := wholeNumber(input[fieldID])
			if !ok {
				return nil, interaction.InvalidError{Fields: map[string]string{fieldID: "must be a whole number"}}
			}

			task, err := apply.EditComment(
				context.Background(),
				textOf(input, fieldRef),
				id,
				textOf(input, fieldBody),
			)
			if err != nil {
				return nil, repair(err, fieldBody)
			}

			return outcome(task)
		},
	}
}

// repair classifies the core's error by what it means, never by its message:
// a rule the product refuses is a denial, a reference or a value the consumer
// can fix is a repairable invalid, and anything else is a real failure.
func repair(err error, fallback string) error {
	if errors.Is(err, core.ErrConflict) {
		return interaction.DeniedError{Reason: err.Error()}
	}

	if errors.Is(err, core.ErrInvalid) || errors.Is(err, core.ErrNotFound) {
		return interaction.InvalidError{Fields: map[string]string{fieldOf(err, fallback): err.Error()}}
	}

	return interaction.DomainError{Message: err.Error()}
}

// fieldOf names the entry the consumer repairs: the one the core's error
// carries, or the operation's fallback when the rule does not name any.
func fieldOf(err error, fallback string) string {
	var named core.InvalidField
	if errors.As(err, &named) && named.Field() != "" {
		return named.Field()
	}

	return fallback
}

// outcome marshals what the operation answers.
func outcome(value any) (json.RawMessage, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("preparing the answer: %w", err)
	}

	return raw, nil
}

// changesOf reads the fields an update names. A member that is absent or null
// does not change anything.
func changesOf(input map[string]json.RawMessage) (core.Changes, error) {
	priority, err := priorityOf(input)
	if err != nil {
		return core.Changes{}, err
	}

	names, given, err := tagsOf(input)
	if err != nil {
		return core.Changes{}, err
	}

	changes := core.Changes{
		Title:       optionalText(input, fieldTitle),
		Description: optionalText(input, fieldDescription),
		Notes:       optionalText(input, fieldNotes),
		Assignee:    optionalText(input, fieldAssignee),
		Priority:    priority,
		Status:      optionalText(input, fieldStatus),
		Parent:      optionalText(input, fieldParent),
	}

	if given {
		changes.Tags = &names
	}

	return changes, nil
}

// textOf reads a text member the library already checked as a string.
func textOf(input map[string]json.RawMessage, name string) string {
	var text string

	_ = json.Unmarshal(input[name], &text)

	return text
}

// optionalText reads a text member and says whether the entry names it: an
// empty text clears the field, an absent member leaves it as it was.
func optionalText(input map[string]json.RawMessage, name string) *string {
	raw, present := input[name]
	if !present || isNull(raw) {
		return nil
	}

	text := textOf(input, name)

	return &text
}

// priorityOf reads the priority, nil when the entry does not name it. The range
// is the core's rule and stays with the core.
func priorityOf(input map[string]json.RawMessage) (*int, error) {
	raw, present := input[fieldPriority]
	if !present || isNull(raw) {
		return nil, nil
	}

	value, ok := wholeNumber(raw)
	if !ok {
		return nil, interaction.InvalidError{Fields: map[string]string{fieldPriority: "must be a whole number"}}
	}

	priority := int(value)

	return &priority, nil
}

// tagsOf reads the whole list of tag names. The second value says whether the
// entry names the list: an empty list leaves the task without any, while an
// absent member leaves the tags as they were.
func tagsOf(input map[string]json.RawMessage) ([]string, bool, error) {
	raw, present := input[fieldTags]
	if !present || isNull(raw) {
		return nil, false, nil
	}

	var names []string
	if err := json.Unmarshal(raw, &names); err != nil {
		return nil, false, interaction.InvalidError{Fields: map[string]string{fieldTags: "each tag must be a text"}}
	}

	if names == nil {
		names = []string{}
	}

	return names, true, nil
}

// wholeNumber reads a JSON number that is a whole one: the profile has no
// integer type, so the adapter checks it without repeating a product rule.
func wholeNumber(raw json.RawMessage) (int64, bool) {
	var number json.Number
	if err := json.Unmarshal(raw, &number); err != nil {
		return 0, false
	}

	value, err := number.Int64()

	return value, err == nil
}

// isNull says whether the member is absent or null.
func isNull(raw json.RawMessage) bool {
	return len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

// refField describes the reference every operation works on.
func refField() interaction.Field {
	return interaction.Field{Name: fieldRef, Required: true, Type: textType, Description: refDescription}
}

// titleField describes the title a new task needs.
func titleField() interaction.Field {
	return interaction.Field{Name: fieldTitle, Required: true, Type: textType, Description: "el título de la tarea"}
}

// textField describes an optional text member.
func textField(name, description string) interaction.Field {
	return interaction.Field{Name: name, Type: textType, Description: description}
}

// priorityField describes the priority with the range the core owns.
func priorityField() interaction.Field {
	return interaction.Field{
		Name:        fieldPriority,
		Type:        "number",
		Description: fmt.Sprintf("la prioridad de la tarea, de %d a %d", core.MinPriority, core.MaxPriority),
		Schema:      json.RawMessage(fmt.Sprintf(`{"minimum":%d,"maximum":%d}`, core.MinPriority, core.MaxPriority)),
	}
}

// statusField describes the status a change may set, from the core's own list.
func statusField() interaction.Field {
	return interaction.Field{
		Name:        fieldStatus,
		Type:        "string",
		Allowed:     core.StatusValues,
		Description: "el estado que el cambio fija; para cerrar está close, con su motivo",
	}
}

// tagsField describes the whole list of tag names of a task.
func tagsField() interaction.Field {
	return interaction.Field{Name: fieldTags, Type: "array", Description: "todas las etiquetas de la tarea, por su nombre"}
}

// parentField describes the reference of the parent task.
func parentField() interaction.Field {
	return interaction.Field{
		Name:        fieldParent,
		Type:        "string",
		Description: "la referencia visible del padre: la asigna o la mueve, y el texto vacío deja la tarea sin padre",
	}
}

// reasonField describes the close reason, from the core's own list.
func reasonField() interaction.Field {
	return interaction.Field{
		Name:        fieldReason,
		Required:    true,
		Type:        "string",
		Allowed:     core.CloseReasons,
		Description: "el motivo del cierre",
	}
}

// blockerField describes the reference of the task that blocks.
func blockerField() interaction.Field {
	return interaction.Field{Name: fieldBlocker, Required: true, Type: textType, Description: refDescription}
}

// idField describes the number of a comment inside its task.
func idField() interaction.Field {
	return interaction.Field{Name: fieldID, Required: true, Type: "number", Description: "el número del comentario dentro de la tarea"}
}

// bodyField describes the body of a comment.
func bodyField(description string) interaction.Field {
	return interaction.Field{Name: fieldBody, Required: true, Type: textType, Description: description}
}
