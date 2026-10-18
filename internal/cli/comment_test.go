package cli_test

import (
	"strings"
	"tasks-ai/internal/core"
	"testing"
)

// commentProject deja un proyecto con una tarea y devuelve su referencia, para
// que los recorridos de comentarios no repitan la instalación.
func commentProject(t *testing.T) string {
	t.Helper()

	t.Chdir(t.TempDir())

	if code, _, errOut := run(t, "init"); code != 0 {
		t.Fatalf("init salió con %d: %s", code, errOut)
	}

	code, out, errOut := run(t, "create", "--title", "Con comentarios")
	if code != 0 {
		t.Fatalf("create salió con %d: %s", code, errOut)
	}

	var created core.Task
	decode(t, out, &created)

	return created.Ref
}

// addComment añade un comentario con la superficie pública y devuelve la tarea
// tal como queda.
func addComment(t *testing.T, ref, body string) core.Task {
	t.Helper()

	code, out, errOut := run(t, "comment", ref, "--body", body)
	if code != 0 {
		t.Fatalf("comment salió con %d: %s", code, errOut)
	}

	var task core.Task
	decode(t, out, &task)

	return task
}

// commentsOf lee los comentarios del detalle de la tarea.
func commentsOf(t *testing.T, ref string) []core.Comment {
	t.Helper()

	code, out, errOut := run(t, "show", ref)
	if code != 0 {
		t.Fatalf("show salió con %d: %s", code, errOut)
	}

	var detail core.Detail
	decode(t, out, &detail)

	return detail.Comments
}

// TestRun_commentAddsACommentAndAnswersWithTheTask fija el comportamiento que
// el comando ya tenía: la creación sigue respondiendo con la tarea, y el cuerpo
// se guarda sin los espacios de los lados.
func TestRun_commentAddsACommentAndAnswersWithTheTask(t *testing.T) {
	ref := commentProject(t)

	task := addComment(t, ref, "  el primero  ")

	comments := commentsOf(t, ref)
	if len(comments) != 1 {
		t.Fatalf("la tarea trae %d comentarios, se esperaba 1: %+v", len(comments), comments)
	}

	if comments[0].Body != "el primero" {
		t.Errorf("el cuerpo guardado es %q, se esperaba %q", comments[0].Body, "el primero")
	}

	if comments[0].ID == 0 {
		t.Errorf("el comentario añadido debe salir con su identidad: %+v", comments[0])
	}

	if task.Ref != ref {
		t.Errorf("la respuesta del alta es %+v, se esperaba la tarea %q", task, ref)
	}
}

// TestRun_commentListReturnsTheCommentsFirstToLast recorre la lectura: los
// comentarios salen del primero al último, cada uno con su identidad, su cuerpo
// y su fecha, y una tarea sin comentarios responde una lista vacía.
func TestRun_commentListReturnsTheCommentsFirstToLast(t *testing.T) {
	ref := commentProject(t)

	code, out, errOut := run(t, "comment", "list", ref)
	if code != 0 {
		t.Fatalf("comment list salió con %d: %s", code, errOut)
	}

	var empty []core.Comment
	decode(t, out, &empty)

	if len(empty) != 0 {
		t.Errorf("una tarea sin comentarios responde %+v, se esperaba una lista vacía", empty)
	}

	addComment(t, ref, "primero")
	addComment(t, ref, "segundo")

	code, out, errOut = run(t, "comment", "list", ref)
	if code != 0 {
		t.Fatalf("comment list salió con %d: %s", code, errOut)
	}

	var listed []core.Comment
	decode(t, out, &listed)

	if len(listed) != 2 || listed[0].Body != "primero" || listed[1].Body != "segundo" {
		t.Errorf("la lista es %+v, se esperaba primero y luego segundo", listed)
	}

	if listed[0].ID == 0 || listed[1].ID == 0 || listed[0].ID == listed[1].ID {
		t.Errorf("cada comentario debe traer una identidad propia: %+v", listed)
	}
}

// TestRun_commentListRefusesAReferenceThatResolvesToNothing comprueba que la
// lectura de una referencia sin tarea se rechaza con el código de error 1.
func TestRun_commentListRefusesAReferenceThatResolvesToNothing(t *testing.T) {
	commentProject(t)

	code, out, errOut := run(t, "comment", "list", "tk-404")
	if code != 1 {
		t.Fatalf("comment list de una referencia inexistente salió con %d, se esperaba 1: %s", code, errOut)
	}

	if out != "" {
		t.Errorf("un error no debe escribir en la salida: %q", out)
	}

	if !strings.Contains(errorText(t, errOut), "tk-404") {
		t.Errorf("el error no dice qué referencia falló: %q", errorText(t, errOut))
	}
}

// TestRun_commentEditChangesOnlyTheBodyOfTheComment recorre la edición: la
// respuesta es la tarea tal como queda, y el comentario editado conserva su
// identidad y su fecha mientras el vecino queda intacto.
func TestRun_commentEditChangesOnlyTheBodyOfTheComment(t *testing.T) {
	ref := commentProject(t)

	addComment(t, ref, "primero")
	addComment(t, ref, "segundo")

	before := commentsOf(t, ref)

	code, out, errOut := run(t, "comment", "edit", ref, "2", "--body", "segundo, corregido")
	if code != 0 {
		t.Fatalf("comment edit salió con %d: %s", code, errOut)
	}

	var task core.Task
	decode(t, out, &task)

	if task.Ref != ref {
		t.Errorf("la respuesta de la edición es %+v, se esperaba la tarea %q", task, ref)
	}

	after := commentsOf(t, ref)

	if len(after) != len(before) {
		t.Fatalf("la edición cambió el número de comentarios: %+v", after)
	}

	if after[1].ID != before[1].ID || !after[1].CreatedAt.Equal(before[1].CreatedAt) {
		t.Errorf("la edición debía conservar identidad y fecha: antes %+v, después %+v", before[1], after[1])
	}

	if after[1].Body != "segundo, corregido" {
		t.Errorf("el cuerpo editado es %q, se esperaba %q", after[1].Body, "segundo, corregido")
	}

	if after[0] != before[0] {
		t.Errorf("el vecino cambió con la edición: antes %+v, después %+v", before[0], after[0])
	}
}

// TestRun_commentEditRefusesWithoutWriting recorre los rechazos de la edición:
// un cuerpo vacío es entrada inválida, un identificador que no es de la tarea y
// una referencia que no resuelve son referencia sin resolución, y ninguno deja
// nada escrito.
func TestRun_commentEditRefusesWithoutWriting(t *testing.T) {
	ref := commentProject(t)

	// Una segunda tarea del mismo proyecto: su comentario comparte la base, y su
	// número no es de la tarea que se edita.
	code, out, errOut := run(t, "create", "--title", "La vecina")
	if code != 0 {
		t.Fatalf("create salió con %d: %s", code, errOut)
	}

	var neighbour core.Task
	decode(t, out, &neighbour)

	addComment(t, neighbour.Ref, "de otra tarea")
	addComment(t, ref, "el mío")

	before := commentsOf(t, ref)

	tests := []struct {
		name string
		args []string
		code int
	}{
		{
			name: "un cuerpo vacío es entrada inválida",
			args: []string{"comment", "edit", ref, "1", "--body", "   "},
			code: 2,
		},
		{
			name: "un identificador que no es de la tarea no resuelve",
			args: []string{"comment", "edit", ref, "99", "--body", "nada"},
			code: 1,
		},
		{
			name: "el identificador de otra tarea no es de esta",
			args: []string{"comment", "edit", ref, "1", "--body", "nada"},
			code: 1,
		},
		{
			name: "una referencia que no resuelve se rechaza",
			args: []string{"comment", "edit", "tk-404", "1", "--body", "nada"},
			code: 1,
		},
		{
			name: "un identificador que no es número es uso incorrecto",
			args: []string{"comment", "edit", ref, "uno", "--body", "nada"},
			code: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code, out, errOut := run(t, test.args...)
			if code != test.code {
				t.Fatalf("salió con %d, se esperaba %d: %s", code, test.code, errOut)
			}

			if out != "" {
				t.Errorf("un error no debe escribir en la salida: %q", out)
			}

			if errorText(t, errOut) == "" {
				t.Error("el rechazo debe traer su mensaje")
			}
		})
	}

	after := commentsOf(t, ref)
	if len(after) != len(before) || after[0].Body != before[0].Body {
		t.Errorf("un rechazo debía dejar los comentarios como estaban: antes %+v, después %+v", before, after)
	}
}

// TestRun_commentWithoutArgumentsIsMisuse fija que el alta sigue siendo el
// comando: sin su referencia y su cuerpo, es un uso incorrecto.
func TestRun_commentWithoutArgumentsIsMisuse(t *testing.T) {
	commentProject(t)

	if code, _, _ := run(t, "comment"); code != 2 {
		t.Errorf("comment sin argumentos salió con %d, se esperaba 2", code)
	}

	if code, _, _ := run(t, "comment", "tk-1"); code != 2 {
		t.Errorf("comment sin cuerpo salió con %d, se esperaba 2", code)
	}
}
