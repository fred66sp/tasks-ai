package core_test

import (
	"errors"
	"fmt"
	"strings"
	"tasks-ai/internal/core"
	"tasks-ai/internal/store"
	"testing"
)

// Los campos que el producto escribe y nombra en el mensaje del rechazo, y los
// textos del contrato que varias tablas comparten.
const (
	fieldTitle       = "title"
	fieldDescription = "description"
	fieldNotes       = "notes"
	fieldComment     = "comment"

	commitText   = "acd789f"
	deliveryText = "sin commitear"
	noGitRemedy  = "the board does not store Git"
	stateText    = "Hecho: el criterio comprobado contra el binario real"
	openStatus   = "open"
)

// aWrite is one of the four paths that write a text in the product, with the
// fields each one carries: crear y cambiar escriben las tres de la tarea, y los
// dos caminos del comentario escriben su cuerpo.
type aWrite struct {
	name   string
	fields []string
	write  func(t *testing.T, apply *core.Core, task core.Task, field, text string) (core.Task, error)
}

// theWrites are the four writes, tried in every test que escribe un texto.
var theWrites = []aWrite{
	{name: "create", fields: []string{fieldTitle, fieldDescription, fieldNotes}, write: writeCreate},
	{name: "change", fields: []string{fieldTitle, fieldDescription, fieldNotes}, write: writeChange},
	{name: "add comment", fields: []string{fieldComment}, write: writeAddComment},
	{name: "edit comment", fields: []string{fieldComment}, write: writeEditComment},
}

// aState is a text the contract rejects, with the fragmento que el mensaje
// nombra y lo que dice que hay que escribir en su lugar.
type aState struct {
	name     string
	fragment string
	remedy   string
}

// theRejectedStates are the examples of the lists of the contract, con la frase
// que cada mensaje añade después del campo y el fragmento que nombra.
var theRejectedStates = []aState{
	{
		name:     "a commit",
		fragment: commitText,
		remedy:   "a commit reaches its task through the commit message, with Refs: tk-N",
	},
	{
		name:     "a branch",
		fragment: "feat/rich-task-text",
		remedy:   noGitRemedy,
	},
	{
		name:     "a remote reference",
		fragment: "origin/HEAD",
		remedy:   noGitRemedy,
	},
	{
		name:     "the state of the task",
		fragment: "Hecho:",
		remedy:   "state what was done, found or decided instead",
	},
	{
		name:     "the state of the delivery",
		fragment: deliveryText,
		remedy:   noGitRemedy,
	},
	{
		name:     "the state of the delivery in the plural",
		fragment: "pendientes de commit",
		remedy:   noGitRemedy,
	},
}

// TestText_rejectsTheExamplesOfTheContractWithTheirMessage recorre el ejemplo
// que el contrato rechaza en cada lista, en el campo con el que la tabla de
// mensajes lo escribe, y comprueba el mensaje entero.
func TestText_rejectsTheExamplesOfTheContractWithTheirMessage(t *testing.T) {
	t.Parallel()

	apply, task := prepared(t)

	tests := []struct {
		name  string
		write aWrite
		field string
		text  string
		want  string
	}{
		{
			name:  "a commit in the description",
			write: theWrites[0],
			field: fieldDescription,
			text:  commitText,
			want: `the description names a commit: "acd789f"; ` +
				`a commit reaches its task through the commit message, with Refs: tk-N`,
		},
		{
			name:  "a branch in the notes",
			write: theWrites[0],
			field: fieldNotes,
			text:  "feat/rich-task-text",
			want:  `the notes name a branch: "feat/rich-task-text"; the board does not store Git`,
		},
		{
			name:  "a remote reference in the comment",
			write: theWrites[2],
			field: fieldComment,
			text:  "origin/HEAD",
			want:  `the comment names a remote reference: "origin/HEAD"; the board does not store Git`,
		},
		{
			name:  "the state of the task in the description",
			write: theWrites[0],
			field: fieldDescription,
			text:  stateText,
			want: `the description declares the state of the task: "Hecho:"; ` +
				`state what was done, found or decided instead`,
		},
		{
			name:  "the state of the delivery in the notes",
			write: theWrites[1],
			field: fieldNotes,
			text:  deliveryText,
			want:  `the notes declare the state of the delivery: "sin commitear"; the board does not store Git`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := tt.write.write(t, apply, task, tt.field, tt.text)

			if !errors.Is(err, core.ErrInvalid) {
				t.Fatalf("%s con %q devolvió %v, se esperaba ErrInvalid", tt.write.name, tt.text, err)
			}

			if err.Error() != tt.want {
				t.Errorf("el rechazo dice %q, se esperaba %q", err, tt.want)
			}
		})
	}
}

// TestText_namesTheFirstListThatCatchesTheText recorre la precedencia del
// contrato: un texto que cae en más de una lista se nombra por la primera, en el
// orden en que el contrato las escribe.
func TestText_namesTheFirstListThatCatchesTheText(t *testing.T) {
	t.Parallel()

	apply, task := prepared(t)

	tests := []struct {
		name  string
		write aWrite
		field string
		text  string
		want  string
	}{
		{
			name:  "un hash y una rama",
			write: theWrites[0],
			field: fieldDescription,
			text:  "acd789f feat/rich-task-text",
			want: `the description names a commit: "acd789f"; ` +
				`a commit reaches its task through the commit message, with Refs: tk-N`,
		},
		{
			name:  "una rama y una entrega",
			write: theWrites[0],
			field: fieldNotes,
			text:  "feat/rich-task-text sin commitear",
			want:  `the notes name a branch: "feat/rich-task-text"; the board does not store Git`,
		},
		{
			name:  "un marcador y una entrega",
			write: theWrites[0],
			field: fieldNotes,
			text:  "Estado: pendiente de commit",
			want:  `the notes declare the state of the task: "Estado:"; state what was done, found or decided instead`,
		},
		{
			name:  "un remoto y una entrega",
			write: theWrites[2],
			field: fieldComment,
			text:  "origin/HEAD sin subir",
			want:  `the comment names a remote reference: "origin/HEAD"; the board does not store Git`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := tt.write.write(t, apply, task, tt.field, tt.text)

			if !errors.Is(err, core.ErrInvalid) {
				t.Fatalf("%s con %q devolvió %v, se esperaba ErrInvalid", tt.write.name, tt.text, err)
			}

			if err.Error() != tt.want {
				t.Errorf("el rechazo dice %q, se esperaba %q", err, tt.want)
			}
		})
	}
}

// TestText_rejectsEveryListInEveryFieldOfEveryWrite cruza las cuatro listas con
// los cuatro campos y las cuatro escrituras: cualquier texto que caiga en una
// lista se rechaza en cualquier campo, por el camino que se escriba, y el
// mensaje nombra el campo, el fragmento y lo que hay que escribir en su lugar.
func TestText_rejectsEveryListInEveryFieldOfEveryWrite(t *testing.T) {
	t.Parallel()

	for _, state := range theRejectedStates {
		for _, write := range theWrites {
			for _, field := range write.fields {
				name := fmt.Sprintf("%s %s: %s", write.name, field, state.name)

				t.Run(name, func(t *testing.T) {
					t.Parallel()

					apply, task := prepared(t)

					_, err := write.write(t, apply, task, field, state.fragment)
					if !errors.Is(err, core.ErrInvalid) {
						t.Fatalf("%s %q en %q devolvió %v, se esperaba ErrInvalid",
							write.name, state.fragment, field, err)
					}

					for _, want := range []string{"the " + field, `"` + state.fragment + `"`, state.remedy} {
						if !strings.Contains(err.Error(), want) {
							t.Errorf("el rechazo %q no dice %q", err, want)
						}
					}
				})
			}
		}
	}
}

// TestText_readsTheListsWithoutDistinguishingCase comprueba que una lista
// alcanza el texto se escriba como se escriba: el contrato no distingue
// mayúsculas.
func TestText_readsTheListsWithoutDistinguishingCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
	}{
		{name: "a branch in capitals", text: "FEAT/RICH-TASK-TEXT"},
		{name: "the delivery in capitals", text: "Sin Commitear"},
		{name: "a commit in capitals", text: "ABCDEF1"},
		{name: "a remote reference in capitals", text: "Origin/HEAD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			apply, task := prepared(t)

			_, err := theWrites[0].write(t, apply, task, fieldDescription, tt.text)
			if !errors.Is(err, core.ErrInvalid) {
				t.Errorf("%s devolvió %v, se esperaba ErrInvalid", tt.text, err)
			}
		})
	}
}

// TestText_leavesTheAcceptedTextsAsTheyWereWritten recorre los textos que el
// tablero ya lleva, en todos los campos de las cuatro escrituras: la
// comprobación los deja pasar y se guardan tal como se escribieron.
func TestText_leavesTheAcceptedTextsAsTheyWereWritten(t *testing.T) {
	t.Parallel()

	accepted := []string{
		"hashes de commit, nombres de rama, referencias a commits pendientes",
		"Decidir la proteccion de rama del repositorio",
		"sin referencias de Git",
		"no se entrega mientras se verifica",
		"Una decisión sobre ramas y una convención de mensajes de commit",
		// El hueco conocido del contrato: siete dígitos no son un hash.
		"20261019",
		// Las mismas palabras nombran lo que el producto hace, y no se rechazan.
		"El modelo tiene tres estados (abierta, en curso y cerrada)",
		"Cerrar con subtareas abiertas se rechaza y la tarea sigue abierta",
		// Formas del estado de la entrega que el contrato deja fuera: nombran la
		// sección del CHANGELOG, el registro de la base o una limpieza.
		"El CHANGELOG ya lo cuenta en [Sin publicar]",
		"La base tenía el WAL sin integrar",
		"Se quitó la mención de si algo estaba sin push",
	}

	for _, text := range accepted {
		for _, write := range theWrites {
			for _, field := range write.fields {
				name := fmt.Sprintf("%s %s: %s", write.name, field, text)

				t.Run(name, func(t *testing.T) {
					t.Parallel()

					apply, task := prepared(t)

					left, err := write.write(t, apply, task, field, text)
					if err != nil {
						t.Fatalf("%s rechazó el texto aceptado %q: %v", write.name, text, err)
					}

					if stored := textOf(t, apply, left, field); stored != text {
						t.Errorf("el campo %s guardó %q, se esperaba %q", field, stored, text)
					}
				})
			}
		}
	}
}

// TestText_keepsTheMessageOfTheRulesThatWereAlreadyThere comprueba que las
// reglas que ya rechazaban un título vacío y un comentario sin cuerpo conservan
// su mensaje y su camino.
func TestText_keepsTheMessageOfTheRulesThatWereAlreadyThere(t *testing.T) {
	t.Parallel()

	apply, task := prepared(t)

	tests := []struct {
		name  string
		write func(apply *core.Core, task core.Task) error
		want  string
	}{
		{
			name: "an empty title when the task is created",
			write: func(apply *core.Core, _ core.Task) error {
				_, err := apply.CreateTask(t.Context(), core.NewTask{Title: "   "})

				return err
			},
			want: "the task needs a title",
		},
		{
			name: "an empty title when the task changes",
			write: func(apply *core.Core, task core.Task) error {
				empty := "   "
				_, err := apply.UpdateTask(t.Context(), task.Ref, core.Changes{Title: &empty})

				return err
			},
			want: "the task needs a title",
		},
		{
			name: "a comment without a body",
			write: func(apply *core.Core, task core.Task) error {
				_, err := apply.AddComment(t.Context(), task.Ref, "   ")

				return err
			},
			want: "the comment body is missing: use --body",
		},
		{
			name: "an edit that leaves the comment without a body",
			write: func(apply *core.Core, task core.Task) error {
				comments := listedComments(t, apply, task.Ref)
				_, err := apply.EditComment(t.Context(), task.Ref, comments[len(comments)-1].ID, "   ")

				return err
			},
			want: "the comment body is missing: use --body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.write(apply, task)

			if !errors.Is(err, core.ErrInvalid) {
				t.Fatalf("la regla devolvió %v, se esperaba ErrInvalid", err)
			}

			if err.Error() != tt.want {
				t.Errorf("la regla dice %q, se esperaba %q", err, tt.want)
			}
		})
	}
}

// TestText_writesNothingWhenItRejects comprueba que un rechazo no deja nada
// escrito: el volcado del proyecto —las tareas con sus campos, los comentarios y
// el último aviso— es el mismo antes y después, por cada uno de los cuatro
// caminos que escriben.
func TestText_writesNothingWhenItRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		write aWrite
		field string
		text  string
	}{
		{name: "create refuses", write: theWrites[0], field: fieldNotes, text: deliveryText},
		{name: "change refuses", write: theWrites[1], field: fieldDescription, text: commitText},
		{name: "adding a comment refuses", write: theWrites[2], field: fieldComment, text: "origin/HEAD"},
		{name: "editing a comment refuses", write: theWrites[3], field: fieldComment, text: "feat/rich-task-text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			apply, task := prepared(t)

			before := dumped(t, apply)

			if _, err := tt.write.write(t, apply, task, tt.field, tt.text); !errors.Is(err, core.ErrInvalid) {
				t.Fatalf("%s con %q devolvió %v, se esperaba ErrInvalid", tt.write.name, tt.text, err)
			}

			if after := dumped(t, apply); after != before {
				t.Errorf("el rechazo dejó la base distinta:\nantes: %s\ndespués: %s", before, after)
			}
		})
	}
}

// TestText_doesNotReadTheFieldsAChangeDoesNotWrite comprueba que un cambio mira
// los campos que escribe y no los que deja: una tarea escrita antes de que la
// regla existiera —con un texto que hoy se rechazaría— se puede seguir
// cambiando, y el texto viejo se queda como estaba.
func TestText_doesNotReadTheFieldsAChangeDoesNotWrite(t *testing.T) {
	t.Parallel()

	base := openStore(t)
	apply := core.New(base, "tk")
	ctx := t.Context()

	oldDescription, oldNotes := commitText, deliveryText

	stored, err := base.CreateTask(ctx, store.Task{
		Title:       "Escrita antes de la regla",
		Description: &oldDescription,
		Status:      openStatus,
		Priority:    core.DefaultPriority,
		Notes:       &oldNotes,
	}, nil)
	if err != nil {
		t.Fatalf("sembrando la tarea: %v", err)
	}

	title := "Cambiada después"

	changed, err := apply.UpdateTask(ctx, "tk-1", core.Changes{Title: &title})
	if err != nil {
		t.Fatalf("cambiando el título de una tarea con textos viejos: %v", err)
	}

	if changed.ID != stored.ID || changed.Title != title {
		t.Errorf("el cambio dejó %+v, se esperaba el título %q", changed, title)
	}

	if changed.Description == nil || *changed.Description != commitText {
		t.Errorf("la descripción quedó en %v, se esperaba intacta", changed.Description)
	}

	if changed.Notes == nil || *changed.Notes != deliveryText {
		t.Errorf("las notas quedaron en %v, se esperaban intactas", changed.Notes)
	}
}

// prepared returns a core with a task that already carries a comment: el estado
// del que parten las cuatro escrituras.
func prepared(t *testing.T) (*core.Core, core.Task) {
	t.Helper()

	apply := core.New(openStore(t), "tk")
	task := newTask(t, apply, "Preparar el análisis")

	if _, err := apply.AddComment(t.Context(), task.Ref, "el suyo"); err != nil {
		t.Fatalf("añadiendo el comentario de partida: %v", err)
	}

	return apply, task
}

// writeCreate crea una tarea con el texto en el campo que se le pide.
func writeCreate(t *testing.T, apply *core.Core, _ core.Task, field, text string) (core.Task, error) {
	t.Helper()

	task := core.NewTask{Title: "Una tarea"}

	switch field {
	case fieldTitle:
		task.Title = text
	case fieldDescription:
		task.Description = text
	case fieldNotes:
		task.Notes = text
	default:
		t.Fatalf("la creación no escribe el campo %q", field)
	}

	return apply.CreateTask(t.Context(), task)
}

// writeChange cambia el campo que se le pide de la tarea, y solo ese.
func writeChange(t *testing.T, apply *core.Core, task core.Task, field, text string) (core.Task, error) {
	t.Helper()

	changes := core.Changes{}

	switch field {
	case fieldTitle:
		changes.Title = &text
	case fieldDescription:
		changes.Description = &text
	case fieldNotes:
		changes.Notes = &text
	default:
		t.Fatalf("el cambio no escribe el campo %q", field)
	}

	return apply.UpdateTask(t.Context(), task.Ref, changes)
}

// writeAddComment añade un comentario con el texto.
func writeAddComment(t *testing.T, apply *core.Core, task core.Task, field, text string) (core.Task, error) {
	t.Helper()

	if field != fieldComment {
		t.Fatalf("el comentario no escribe el campo %q", field)
	}

	return apply.AddComment(t.Context(), task.Ref, text)
}

// writeEditComment cambia el cuerpo del último comentario de la tarea.
func writeEditComment(t *testing.T, apply *core.Core, task core.Task, field, text string) (core.Task, error) {
	t.Helper()

	if field != fieldComment {
		t.Fatalf("el comentario no escribe el campo %q", field)
	}

	comments := listedComments(t, apply, task.Ref)
	if len(comments) == 0 {
		t.Fatalf("la tarea %q no tiene comentarios que editar", task.Ref)
	}

	return apply.EditComment(t.Context(), task.Ref, comments[len(comments)-1].ID, text)
}

// textOf returns the text the field carries in the task as it was left, with el
// cuerpo del último comentario para el campo del comentario.
func textOf(t *testing.T, apply *core.Core, task core.Task, field string) string {
	t.Helper()

	switch field {
	case fieldTitle:
		return task.Title
	case fieldDescription:
		return textOrEmpty(task.Description)
	case fieldNotes:
		return textOrEmpty(task.Notes)
	case fieldComment:
		comments := listedComments(t, apply, task.Ref)
		if len(comments) == 0 {
			t.Fatalf("la tarea %q no tiene comentarios", task.Ref)
		}

		return comments[len(comments)-1].Body
	}

	t.Fatalf("el campo %q no es de los que se escriben aquí", field)

	return ""
}

// textOrEmpty returns the text, or the empty one when the field has no value.
func textOrEmpty(text *string) string {
	if text == nil {
		return ""
	}

	return *text
}

// dumped describes lo que la interfaz pública enseña del proyecto: las tareas
// con sus campos, todos los comentarios y la identidad del último aviso.
func dumped(t *testing.T, apply *core.Core) string {
	t.Helper()

	ctx := t.Context()

	tasks, err := apply.Tasks(ctx)
	if err != nil {
		t.Fatalf("leyendo el listado: %v", err)
	}

	comments, err := apply.AllComments(ctx)
	if err != nil {
		t.Fatalf("leyendo los comentarios: %v", err)
	}

	last, err := apply.LastEventID(ctx)
	if err != nil {
		t.Fatalf("leyendo el último aviso: %v", err)
	}

	return fmt.Sprintf("%+v\n%+v\n%d", tasks, comments, last)
}
