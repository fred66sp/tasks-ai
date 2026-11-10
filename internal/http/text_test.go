package http_test

import (
	"net/http"
	"tasks-ai/internal/core"
	"testing"
)

// textRefusals are the rejections of the lists of the contract, with the message
// the core composes for each one: la misma frase que el CLI escribe en su JSON.
var textRefusals = []struct {
	name    string
	method  string
	path    string
	body    string
	message string
}{
	{
		name:    "la descripción de una tarea nueva nombra un commit",
		method:  http.MethodPost,
		path:    "/api/tasks",
		body:    `{"title": "Otra", "description": "el commit a1b2c3d lo hizo"}`,
		message: `the description names a commit: "a1b2c3d"; a commit reaches its task through the commit message, with Refs: tk-N`,
	},
	{
		name:    "las notas de una tarea nueva nombran una rama",
		method:  http.MethodPost,
		path:    "/api/tasks",
		body:    `{"title": "Otra", "notes": "ver feat/rich-task-text"}`,
		message: `the notes name a branch: "feat/rich-task-text"; the board does not store Git`,
	},
	{
		name:    "la descripción de un cambio declara el estado de la tarea",
		method:  http.MethodPatch,
		path:    "/api/tasks/tk-1",
		body:    `{"description": "estado: pendiente"}`,
		message: `the description declares the state of the task: "estado:"; state what was done, found or decided instead`,
	},
	{
		name:    "las notas de un cambio declaran el estado de la entrega",
		method:  http.MethodPatch,
		path:    "/api/tasks/tk-1",
		body:    `{"notes": "sin commitear"}`,
		message: `the notes declare the state of the delivery: "sin commitear"; the board does not store Git`,
	},
	{
		name:    "el cuerpo de un comentario nombra un remoto",
		method:  http.MethodPost,
		path:    "/api/tasks/tk-1/comments",
		body:    `{"body": "origin/HEAD es la referencia"}`,
		message: `the comment names a remote reference: "origin/HEAD"; the board does not store Git`,
	},
	{
		name:    "el cuerpo editado de un comentario declara el estado",
		method:  http.MethodPatch,
		path:    "/api/tasks/tk-1/comments/1",
		body:    `{"body": "Hecho: el criterio"}`,
		message: `the comment declares the state of the task: "Hecho:"; state what was done, found or decided instead`,
	},
	{
		name:    "las notas de una tarea nueva nombran un Git con su verbo",
		method:  http.MethodPost,
		path:    "/api/tasks",
		body:    `{"title": "Otra", "notes": "mergeado a main"}`,
		message: `the notes declare the state of Git: "mergeado a main"; the board does not store Git`,
	},
	{
		name:    "la descripción de un cambio nombra una rama con su nombre",
		method:  http.MethodPatch,
		path:    "/api/tasks/tk-1",
		body:    `{"description": "la rama main"}`,
		message: `the description names a Git reference: "rama main"; the board does not store Git`,
	},
}

// theLegitimateText is a text the lists of the contract leave pass: se guarda.
const theLegitimateText = "Decidir la proteccion de rama del repositorio"

// TestServer_refusesTheTextThatWritesTheState recorre el criterio de tk-150 desde
// la API: el texto que rechaza el CLI se rechaza aquí con la misma frase, el
// código 400 y nada escrito, y el texto legítimo se guarda.
func TestServer_refusesTheTextThatWritesTheState(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	do(t, server, http.MethodPost, "/api/tasks", `{"title": "Base", "description": "`+theLegitimateText+`"}`)
	do(t, server, http.MethodPost, "/api/tasks/tk-1/comments", `{"body": "cuerpo legitimo"}`)

	for _, tc := range textRefusals {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			status, body := do(t, server, tc.method, tc.path, tc.body)

			expect(t, status, body, http.StatusBadRequest, tc.message)
		})
	}

	status, body := do(t, server, http.MethodGet, "/api/tasks", "")
	if status != http.StatusOK {
		t.Fatalf("listar respondió %d: %s", status, body)
	}

	var listed []core.Task

	decode(t, body, &listed)

	if len(listed) != 1 || listed[0].Ref != "tk-1" {
		t.Errorf("el tablero quedó con %v, se esperaba solo tk-1", listed)
	}

	status, body = do(t, server, http.MethodGet, "/api/tasks/tk-1", "")
	if status != http.StatusOK {
		t.Fatalf("leer la tarea respondió %d: %s", status, body)
	}

	var detail core.Detail

	decode(t, body, &detail)

	if detail.Description == nil || *detail.Description != theLegitimateText {
		t.Errorf("la descripción quedó en %v, se esperaba %q", detail.Description, theLegitimateText)
	}

	if detail.Notes != nil {
		t.Errorf("las notas quedaron en %q, se esperaba sin valor", *detail.Notes)
	}

	if len(detail.Comments) != 1 || detail.Comments[0].Body != "cuerpo legitimo" {
		t.Errorf("los comentarios quedaron en %+v, se esperaba el cuerpo legítimo sin cambiar", detail.Comments)
	}

	// Un rechazo no deja el servidor cerrado: el mismo texto legítimo se guarda
	// también después.
	status, body = do(t, server, http.MethodPost, "/api/tasks", `{"title": "Después", "description": "`+theLegitimateText+`"}`)
	if status != http.StatusCreated {
		t.Fatalf("crear con texto legítimo después de los rechazos respondió %d: %s", status, body)
	}
}
