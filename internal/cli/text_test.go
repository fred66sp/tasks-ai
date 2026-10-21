package cli_test

import (
	"strings"
	"tasks-ai/internal/core"
	"testing"
)

const (
	// ruleOfTheText is the line the help of every command that writes a text
	// carries: la regla en una línea, para que --help la nombre.
	ruleOfTheText = "The text writes no state: the title, the description, the notes " +
		"and the comment body name neither Git nor the state of the task or of the delivery."
	// theLegitimateText is a text the lists of the contract leave pass: se guarda.
	theLegitimateText = "Decidir la proteccion de rama del repositorio"
	theTask           = "tk-1"
	theComment        = "cuerpo legitimo"
	createCmd         = "create"
	updateCmd         = "update"
	helpFlag          = "--help"
)

// textRefusals are the rejections of the lists of the contract, each one with the
// message the core composes for it: la misma frase en las tres superficies.
var textRefusals = []struct {
	name    string
	args    []string
	message string
}{
	{
		name:    "la descripción de una tarea nueva nombra un commit",
		args:    []string{createCmd, "--title", "Otra", "--description", "el commit acd789f lo hizo"},
		message: `the description names a commit: "acd789f"; a commit reaches its task through the commit message, with Refs: tk-N`,
	},
	{
		name:    "las notas de una tarea nueva nombran una rama",
		args:    []string{createCmd, "--title", "Otra", "--notes", "ver feat/rich-task-text"},
		message: `the notes name a branch: "feat/rich-task-text"; the board does not store Git`,
	},
	{
		name:    "la descripción de un cambio declara el estado de la tarea",
		args:    []string{updateCmd, theTask, "--description", "estado: pendiente"},
		message: `the description declares the state of the task: "estado:"; state what was done, found or decided instead`,
	},
	{
		name:    "las notas de un cambio declaran el estado de la entrega",
		args:    []string{updateCmd, theTask, "--notes", "sin commitear"},
		message: `the notes declare the state of the delivery: "sin commitear"; the board does not store Git`,
	},
	{
		name:    "el cuerpo de un comentario nombra un remoto",
		args:    []string{commentCmd, theTask, editFlag, "origin/HEAD es la referencia"},
		message: `the comment names a remote reference: "origin/HEAD"; the board does not store Git`,
	},
	{
		name:    "el cuerpo editado de un comentario declara el estado",
		args:    []string{commentCmd, editCmd, theTask, "1", editFlag, "Hecho: el criterio"},
		message: `the comment declares the state of the task: "Hecho:"; state what was done, found or decided instead`,
	},
}

// TestRun_refusesTheTextThatWritesTheState recorre el criterio de tk-150 desde el
// CLI: los comandos que escriben un texto rechazan el texto con la frase del
// núcleo, la salida 2 y nada escrito, y su ayuda nombra la regla.
//
//nolint:gocyclo,paralleltest // es el recorrido completo y cambia de directorio
func TestRun_refusesTheTextThatWritesTheState(t *testing.T) {
	t.Chdir(t.TempDir())

	initProjectWithTaskAndComment(t)

	for _, tc := range textRefusals {
		t.Run(tc.name, func(t *testing.T) {
			code, out, errOut := run(t, tc.args...)

			if code != 2 {
				t.Errorf("%v salió con %d, se esperaba 2: %s", tc.args, code, errOut)
			}

			if out != "" {
				t.Errorf("%v escribió en la salida: %q", tc.args, out)
			}

			if got := errorText(t, errOut); got != tc.message {
				t.Errorf("el rechazo dice %q, se esperaba %q", got, tc.message)
			}
		})
	}

	// Nada de lo rechazado quedó escrito: una sola tarea, con su descripción y su
	// comentario intactos, y el texto legítimo guardado tal cual.
	code, out, errOut := run(t, "list")
	if code != 0 {
		t.Fatalf("list salió con %d: %s", code, errOut)
	}

	var listed []core.Task
	decode(t, out, &listed)

	if len(listed) != 1 || listed[0].Ref != theTask {
		t.Errorf("el tablero quedó con %v, se esperaba solo %s", listed, theTask)
	}

	code, out, errOut = run(t, "show", theTask)
	if code != 0 {
		t.Fatalf("show salió con %d: %s", code, errOut)
	}

	var detail core.Detail
	decode(t, out, &detail)

	if detail.Description == nil || *detail.Description != theLegitimateText {
		t.Errorf("la descripción quedó en %v, se esperaba %q", detail.Description, theLegitimateText)
	}

	if detail.Notes != nil {
		t.Errorf("las notas quedaron en %q, se esperaba sin valor", *detail.Notes)
	}

	if len(detail.Comments) != 1 || detail.Comments[0].Body != theComment {
		t.Errorf("los comentarios quedaron en %+v, se esperaba el cuerpo legítimo sin cambiar", detail.Comments)
	}

	// La ayuda de los comandos que escriben un texto nombra la regla.
	for _, args := range [][]string{
		{createCmd, helpFlag},
		{updateCmd, helpFlag},
		{commentCmd, helpFlag},
		{commentCmd, editCmd, helpFlag},
	} {
		code, out, errOut = run(t, args...)
		if code != 0 {
			t.Fatalf("%v salió con %d: %s", args, code, errOut)
		}

		if !strings.Contains(out, ruleOfTheText) {
			t.Errorf("la ayuda de %v no nombra la regla %q:\n%s", args, ruleOfTheText, out)
		}
	}

	// Un rechazo no deja el producto cerrado: el mismo texto legítimo se guarda
	// también después.
	if code, _, errOut := run(t, createCmd, "--title", "Después", "--description", theLegitimateText); code != 0 {
		t.Fatalf("crear con texto legítimo después de los rechazos salió con %d: %s", code, errOut)
	}

	code, out, errOut = run(t, "show", "tk-2")
	if code != 0 {
		t.Fatalf("show salió con %d: %s", code, errOut)
	}

	var later core.Detail
	decode(t, out, &later)

	if later.Description == nil || *later.Description != theLegitimateText {
		t.Errorf("la descripción de la tarea nueva quedó en %v, se esperaba %q", later.Description, theLegitimateText)
	}
}

// initProjectWithTaskAndComment instala el producto y deja una tarea con su
// descripción legítima y un comentario, que es lo que el rechazo no debe tocar.
func initProjectWithTaskAndComment(t *testing.T) {
	t.Helper()

	if code, _, errOut := run(t, "init"); code != 0 {
		t.Fatalf("init salió con %d: %s", code, errOut)
	}

	if code, _, errOut := run(t, createCmd, "--title", "Base", "--description", theLegitimateText); code != 0 {
		t.Fatalf("crear la tarea base salió con %d: %s", code, errOut)
	}

	if code, _, errOut := run(t, commentCmd, theTask, editFlag, theComment); code != 0 {
		t.Fatalf("comentar la tarea base salió con %d: %s", code, errOut)
	}
}
