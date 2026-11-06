package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"tasks-ai/internal/cli"
	"tasks-ai/internal/core"
	"tasks-ai/internal/install"
	"tasks-ai/internal/settings"
	"testing"
)

// TestRun_initFixesThePrefixAndTheInstructionsUseIt recorre el criterio de tk-14:
// instalar con un prefijo propio deja referencias y ejemplos con ese prefijo, y los
// comandos siguientes no avisan de que la base y los ajustes no coinciden.
func TestRun_initFixesThePrefixAndTheInstructionsUseIt(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	code, out, errOut := run(t, "init", "--prefix", "tai")
	if code != 0 {
		t.Fatalf("init --prefix tai salió con %d: %s", code, errOut)
	}

	var installed install.Result
	decode(t, out, &installed)

	if installed.Prefix != "tai" {
		t.Errorf("el prefijo instalado es %q, se esperaba tai", installed.Prefix)
	}

	written, err := os.ReadFile(installed.Instructions)
	if err != nil {
		t.Fatalf("leyendo las instrucciones: %v", err)
	}

	if !strings.Contains(string(written), "tai-1") {
		t.Error("las instrucciones no citan tai-1 como referencia de ejemplo")
	}

	code, out, errOut = run(t, "create", "--title", "Con prefijo propio")
	if code != 0 {
		t.Fatalf("create salió con %d: %s", code, errOut)
	}

	var created core.Task
	decode(t, out, &created)

	if created.Ref != "tai-1" {
		t.Errorf("la tarea nació como %q, se esperaba tai-1", created.Ref)
	}

	if errOut != "" {
		t.Errorf("create avisó después de instalar con prefijo propio: %s", errOut)
	}

	code, _, errOut = run(t, "show", created.Ref)
	if code != 0 {
		t.Fatalf("show salió con %d: %s", code, errOut)
	}

	if errOut != "" {
		t.Errorf("show avisó después de instalar con prefijo propio: %s", errOut)
	}
}

// TestRun_initRejectsAPrefixTheReferencesCannotCarry comprueba que un prefijo
// imposible se rechaza como uso incorrecto y sin dejar nada escrito.
func TestRun_initRejectsAPrefixTheReferencesCannotCarry(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	code, _, errOut := run(t, "init", "--prefix", "1 x")
	if code != 2 {
		t.Fatalf("init con un prefijo imposible salió con %d, se esperaba 2: %s", code, errOut)
	}

	if text := errorText(t, errOut); !strings.Contains(text, "prefix") {
		t.Errorf("el error no dice qué se rechazó: %q", text)
	}

	if _, err := os.Stat(filepath.Join(dir, install.DirName)); err == nil {
		t.Error("un prefijo rechazado dejó la carpeta del producto escrita")
	}
}

// El criterio de la fase: en un proyecto vacío, instalar el producto, guardar una
// tarea y leerla con la referencia formada por el prefijo configurado.
//
//nolint:gocyclo // es el recorrido completo del producto y debe quedarse lineal
func TestRun_installsCreatesAndReadsATask(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	code, out, errOut := run(t, "init")
	if code != 0 {
		t.Fatalf("init salió con %d: %s", code, errOut)
	}

	var installed install.Result
	decode(t, out, &installed)

	if want := filepath.Join(dir, install.DirName); installed.Dir != want {
		t.Errorf("la instalación quedó en %q, se esperaba %q", installed.Dir, want)
	}

	if installed.Prefix != "tk" {
		t.Errorf("el prefijo instalado es %q, se esperaba %q", installed.Prefix, "tk")
	}

	if installed.Pointer == "" {
		t.Error("la instalación no trae el texto para apuntar a las instrucciones desde fuera")
	}

	// CA-04: instalar no toca nada fuera de su carpeta.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("leyendo el proyecto: %v", err)
	}

	found := make([]string, 0, len(entries))
	for _, entry := range entries {
		found = append(found, entry.Name())
	}

	if len(found) != 1 || found[0] != install.DirName {
		t.Errorf("la instalación dejó %v en el proyecto, solo debía dejar %q", found, install.DirName)
	}

	for _, name := range []string{"tasks.db", "settings.json", "AGENTS.md"} {
		if _, err := os.Stat(filepath.Join(installed.Dir, name)); err != nil {
			t.Errorf("la instalación no dejó %s: %v", name, err)
		}
	}

	// Crear la tarea: la identidad interna y la referencia salen en el JSON.
	code, out, errOut = run(t, "create", "--title", "Preparar el análisis")
	if code != 0 {
		t.Fatalf("create salió con %d: %s", code, errOut)
	}

	var created core.Task
	decode(t, out, &created)

	if created.ID != 1 {
		t.Errorf("la primera tarea tiene la identidad %d, se esperaba 1", created.ID)
	}

	if created.Ref != "tk-1" {
		t.Errorf("la referencia es %q, se esperaba %q", created.Ref, "tk-1")
	}

	if created.Status != "open" {
		t.Errorf("la tarea nace en %q, se esperaba %q", created.Status, "open")
	}

	if created.Priority != core.DefaultPriority {
		t.Errorf("la tarea nace con prioridad %d, se esperaba %d", created.Priority, core.DefaultPriority)
	}

	if created.Description != nil || created.CloseReason != nil || created.Assignee != nil {
		t.Errorf("los campos sin valor deben salir nulos: %+v", created)
	}

	if created.Tags == nil {
		t.Error("las etiquetas deben salir como lista, no como nulo")
	}

	// Leerla por su referencia.
	code, out, errOut = run(t, "show", created.Ref)
	if code != 0 {
		t.Fatalf("show salió con %d: %s", code, errOut)
	}

	var read core.Task
	decode(t, out, &read)

	if read.ID != created.ID || read.Ref != created.Ref || read.Title != created.Title {
		t.Errorf("la tarea leída no es la guardada: %+v frente a %+v", read, created)
	}

	if !read.CreatedAt.Equal(created.CreatedAt) || !read.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("la tarea leída cambia sus fechas: %+v frente a %+v", read, created)
	}

	// Una referencia que no existe se responde con error y salida distinta de cero.
	code, out, errOut = run(t, "show", "tk-404")
	if code == 0 {
		t.Error("leer una referencia inexistente debía salir con error")
	}

	if out != "" {
		t.Errorf("un error no debe escribir en la salida: %q", out)
	}

	if errorText(t, errOut) == "" {
		t.Error("el error debe traer un mensaje")
	}

	// El prefijo de otro proyecto tampoco resuelve aquí.
	if code, _, _ := run(t, "show", "otro-1"); code == 0 {
		t.Error("una referencia de otro prefijo debía salir con error")
	}

	// Una entrada inválida es un error de uso, con su propio código de salida.
	if code, _, _ := run(t, "create"); code != 2 {
		t.Errorf("crear sin título salió con %d, se esperaba 2", code)
	}
}

// failingWriter simula una salida donde ya no se puede escribir nada, como un
// disco lleno.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("no queda espacio en el dispositivo")
}

func TestRun_reportsAFailingAnswerWriter(t *testing.T) {
	t.Chdir(t.TempDir())

	var errOut bytes.Buffer

	code := cli.Run([]string{"init"}, failingWriter{}, &errOut)

	if code != 1 {
		t.Errorf("escribir la respuesta falló y el comando salió con %d, se esperaba 1", code)
	}

	if msg := errorText(t, errOut.String()); msg == "" {
		t.Errorf("el error debe cruzar en JSON por la salida de errores, se obtuvo %q", errOut.String())
	}
}

// TestRun_tagsCanBeCreatedEditedListedAndDeleted recorre el catálogo de etiquetas
// por el CLI: crear, editar, listar y borrar siguen cruzando la superficie pública.
func TestRun_tagsCanBeCreatedEditedListedAndDeleted(t *testing.T) {
	t.Chdir(t.TempDir())

	if code, _, errOut := run(t, "init"); code != 0 {
		t.Fatalf("init salió con %d, se esperaba 0: %s", code, errOut)
	}

	code, out, errOut := run(t, "tag", "create", "--name", "urgente")
	if code != 0 {
		t.Fatalf("tag create salió con %d: %s", code, errOut)
	}

	var created core.Tag
	decode(t, out, &created)

	code, out, errOut = run(t, "tag", "edit", "urgente", "--name", "urgentisima")
	if code != 0 {
		t.Fatalf("tag edit salió con %d: %s", code, errOut)
	}

	var edited core.Tag
	decode(t, out, &edited)

	if edited.ID != created.ID || edited.Name != "urgentisima" {
		t.Errorf("la etiqueta editada es %+v, se esperaba la identidad %d y el nombre urgentisima", edited, created.ID)
	}

	code, out, errOut = run(t, "tags")
	if code != 0 {
		t.Fatalf("tags salió con %d: %s", code, errOut)
	}

	var catalogue []core.TagWithUses
	decode(t, out, &catalogue)

	if !slices.Contains(tagNames(t, catalogue), "urgentisima") || slices.Contains(tagNames(t, catalogue), "urgente") {
		t.Errorf("el catálogo es %+v, se esperaba urgentisima y no urgente", catalogue)
	}

	code, out, errOut = run(t, "tag", "delete", "urgentisima")
	if code != 0 {
		t.Fatalf("tag delete salió con %d: %s", code, errOut)
	}

	var deleted core.Tag
	decode(t, out, &deleted)

	if deleted.ID != created.ID || deleted.Name != "urgentisima" {
		t.Errorf("la etiqueta borrada es %+v, se esperaba la creada", deleted)
	}
}

// TestRun_updateMovesATaskInProgressAndBack recorre el ciclo del estado desde
// la línea de comandos: una tarea pendiente se pone en curso con --status y
// vuelve a pendiente.
func TestRun_updateMovesATaskInProgressAndBack(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if code, _, errOut := run(t, "init"); code != 0 {
		t.Fatalf("init salió con %d: %s", code, errOut)
	}

	code, out, errOut := run(t, "create", "--title", "Con estados")
	if code != 0 {
		t.Fatalf("create salió con %d: %s", code, errOut)
	}

	var created core.Task
	decode(t, out, &created)

	code, out, errOut = run(t, "update", created.Ref, "--status", "in_progress")
	if code != 0 {
		t.Fatalf("update --status in_progress salió con %d: %s", code, errOut)
	}

	var progressed core.Task
	decode(t, out, &progressed)

	if progressed.Status != "in_progress" {
		t.Errorf("la tarea quedó en %q, se esperaba in_progress", progressed.Status)
	}

	code, out, errOut = run(t, "update", created.Ref, "--status", "open")
	if code != 0 {
		t.Fatalf("update --status open salió con %d: %s", code, errOut)
	}

	var returned core.Task
	decode(t, out, &returned)

	if returned.Status != "open" {
		t.Errorf("la tarea quedó en %q, se esperaba open", returned.Status)
	}
}

// TestRun_updateRejectsClosingThroughTheStatus comprueba que cerrar no va por
// el cambio: --status done se rechaza como uso incorrecto y el mensaje manda a
// close con su motivo.
func TestRun_updateRejectsClosingThroughTheStatus(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if code, _, errOut := run(t, "init"); code != 0 {
		t.Fatalf("init salió con %d: %s", code, errOut)
	}

	code, out, errOut := run(t, "create", "--title", "Que no se cierra por aquí")
	if code != 0 {
		t.Fatalf("create salió con %d: %s", code, errOut)
	}

	var created core.Task
	decode(t, out, &created)

	code, out, errOut = run(t, "update", created.Ref, "--status", "done")
	if code != 2 {
		t.Errorf("update --status done salió con %d, se esperaba 2: %s", code, errOut)
	}

	if out != "" {
		t.Errorf("un error no debe escribir en la salida: %q", out)
	}

	message := errorText(t, errOut)
	if !strings.Contains(message, "close") || !strings.Contains(message, "reason") {
		t.Errorf("el mensaje del rechazo (%q) debe mandar a close con su motivo", message)
	}
}

// TestRun_closeRejectsDoneWhenTheTaskIsPending comprueba que cerrar como hecha
// exige la tarea en curso: el rechazo es de regla, sale con 1 y el mensaje dice
// el atajo; puesto en curso, el mismo cierre pasa.
func TestRun_closeRejectsDoneWhenTheTaskIsPending(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if code, _, errOut := run(t, "init"); code != 0 {
		t.Fatalf("init salió con %d: %s", code, errOut)
	}

	code, out, errOut := run(t, "create", "--title", "Sin empezar")
	if code != 0 {
		t.Fatalf("create salió con %d: %s", code, errOut)
	}

	var created core.Task
	decode(t, out, &created)

	code, out, errOut = run(t, "close", created.Ref, "--reason", "done")
	if code != 1 {
		t.Errorf("close done de una tarea pendiente salió con %d, se esperaba 1: %s", code, errOut)
	}

	if out != "" {
		t.Errorf("un error no debe escribir en la salida: %q", out)
	}

	message := errorText(t, errOut)
	if !strings.Contains(message, "not in progress") || !strings.Contains(message, "--status in_progress") {
		t.Errorf("el mensaje del rechazo (%q) debe mandar a poner la tarea en curso", message)
	}

	if code, _, errOut := run(t, "update", created.Ref, "--status", "in_progress"); code != 0 {
		t.Fatalf("update salió con %d: %s", code, errOut)
	}

	if code, _, errOut := run(t, "close", created.Ref, "--reason", "done"); code != 0 {
		t.Fatalf("close salió con %d: %s", code, errOut)
	}
}

// TestRun_createUsesTheProjectsPerson deja escrito que la tarea creada sin
// responsable lleva el de los ajustes, y que el explícito manda.
func TestRun_createUsesTheProjectsPerson(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if code, _, errOut := run(t, "init"); code != 0 {
		t.Fatalf("init salió con %d: %s", code, errOut)
	}

	cfg, err := settings.Load(filepath.Join(dir, install.DirName))
	if err != nil {
		t.Fatalf("leyendo los ajustes: %v", err)
	}

	cfg.Person = "Alfredo González P."

	if err := settings.Save(filepath.Join(dir, install.DirName), cfg); err != nil {
		t.Fatalf("guardando los ajustes: %v", err)
	}

	code, out, errOut := run(t, "create", "--title", "Con persona")
	if code != 0 {
		t.Fatalf("create salió con %d: %s", code, errOut)
	}

	var created core.Task
	decode(t, out, &created)

	if created.Assignee == nil || *created.Assignee != "Alfredo González P." {
		t.Errorf("la tarea nació con responsable %v, se esperaba el del proyecto", created.Assignee)
	}

	code, out, errOut = run(t, "create", "--title", "Con la suya", "--assignee", "Otra Persona")
	if code != 0 {
		t.Fatalf("create salió con %d: %s", code, errOut)
	}

	var explicit core.Task
	decode(t, out, &explicit)

	if explicit.Assignee == nil || *explicit.Assignee != "Otra Persona" {
		t.Errorf("la tarea nació con responsable %v, se esperaba el explícito", explicit.Assignee)
	}
}

func run(t *testing.T, args ...string) (code int, out, errOut string) {
	t.Helper()

	var stdout, stderr bytes.Buffer

	code = cli.Run(args, &stdout, &stderr)

	return code, stdout.String(), stderr.String()
}

func decode(t *testing.T, raw string, target any) {
	t.Helper()

	if err := json.Unmarshal([]byte(raw), target); err != nil {
		t.Fatalf("el JSON devuelto no se entiende (%v): %q", err, raw)
	}
}

func errorText(t *testing.T, raw string) string {
	t.Helper()

	var body struct {
		Error string `json:"error"`
	}

	decode(t, raw, &body)

	return body.Error
}

// tagNames devuelve los nombres de un catálogo leído.
func tagNames(t *testing.T, catalogue []core.TagWithUses) []string {
	t.Helper()

	names := make([]string, 0, len(catalogue))

	for _, tag := range catalogue {
		names = append(names, tag.Name)
	}

	return names
}
