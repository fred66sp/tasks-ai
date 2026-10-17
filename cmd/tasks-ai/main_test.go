// El binario real, llamado como lo llama un usuario: se construye una única vez
// y cada escenario lo ejercita de extremo a extremo, comprobando el código de
// salida, el JSON de la salida estándar y el JSON de error de la de errores.
package main_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"tasks-ai/internal/cli"
	"tasks-ai/internal/core"
	"tasks-ai/internal/install"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// binPath es donde quedó construido el binario, compartido por todos los escenarios.
var binPath string

// TestMain construye el binario real una sola vez, en una carpeta temporal que
// se recoge al terminar, antes de ejecutar los escenarios.
func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "tasks-ai-bin-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "creando la carpeta temporal del binario: %v\n", err)

		os.Exit(1)
	}

	code := runScenarios(m, tmp)

	_ = os.RemoveAll(tmp)

	os.Exit(code)
}

// runScenarios construye el binario y ejecuta los escenarios: devuelve el
// código con el que debe terminar el proceso de pruebas.
func runScenarios(m *testing.M, tmp string) int {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Fprintln(os.Stderr, "no se pudo ubicar la carpeta del paquete para construir el binario")

		return 1
	}

	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}

	binPath = filepath.Join(tmp, "tasks-ai"+suffix)

	//nolint:gosec // el binario es el propio producto, recién construido
	build := exec.CommandContext(context.Background(), "go", "build", "-o", binPath, ".")
	build.Dir = filepath.Dir(thisFile)

	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "construyendo el binario: %v\n%s", err, out)

		return 1
	}

	return m.Run()
}

// errorBody es como un error cruza el límite del binario.
type errorBody struct {
	Error string `json:"error"`
}

// statusOpen es el estado de una tarea que nadie ha cerrado.
const statusOpen = "open"

// missingRef es una referencia que no apunta a ninguna tarea.
const missingRef = "tk-999"

// run llama al binario dentro de la carpeta dir, como si fuera el proyecto en
// el que trabaja el usuario, y devuelve el código de salida y las dos salidas.
func run(t *testing.T, dir string, args ...string) (int, string, string) {
	t.Helper()

	var out, errOut bytes.Buffer

	//nolint:gosec // el binario es el propio producto y los argumentos, de prueba
	cmd := exec.CommandContext(t.Context(), binPath, args...)
	cmd.Dir = dir
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	err := cmd.Run()

	code := 0

	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("no se pudo ejecutar %v: %v", args, err)
		}

		code = exitErr.ExitCode()
	}

	return code, out.String(), errOut.String()
}

// decode lee el JSON de una respuesta, que debe venir siempre bien formado.
func decode(t *testing.T, raw string, value any) {
	t.Helper()

	if err := json.Unmarshal([]byte(raw), value); err != nil {
		t.Fatalf("la respuesta no es un JSON válido: %v\n%s", err, raw)
	}
}

// errorJSON comprueba que la salida de errores es el JSON prometido y devuelve
// su mensaje.
func errorJSON(t *testing.T, raw string) string {
	t.Helper()

	var body errorBody
	decode(t, raw, &body)

	if body.Error == "" {
		t.Fatalf("la salida de errores no trae mensaje: %s", raw)
	}

	return body.Error
}

// installProject deja un proyecto temporal con el producto instalado.
func installProject(t *testing.T, dir string) {
	t.Helper()

	if code, _, errOut := run(t, dir, "init"); code != 0 {
		t.Fatalf("init salió con %d: %s", code, errOut)
	}
}

// TestHelpListsTheCommands: la ayuda menciona los tres comandos del producto.
func TestHelpListsTheCommands(t *testing.T) {
	t.Parallel()

	code, out, errOut := run(t, t.TempDir(), "--help")

	if code != 0 {
		t.Fatalf("--help salió con %d: %s", code, errOut)
	}

	for _, name := range []string{"init", createCmd, "show"} {
		if !strings.Contains(out, name) {
			t.Errorf("--help no menciona %q:\n%s", name, out)
		}
	}
}

// spanishText finds the marks of Spanish in a help page: the letters the
// language has and English does not, and the short words that only it uses.
var spanishText = regexp.MustCompile(`(?i)[áéíóúñ¿¡]|\b(de|del|la|las|el|los|una|un|que|para|por|con|sin|usa|falta)\b`)

// TestHelpOfEveryCommandIsInEnglish: la ayuda de todos los comandos del
// producto, subcomandos incluidos, se lee en inglés.
func TestHelpOfEveryCommandIsInEnglish(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	for _, path := range commandPaths(cli.New(io.Discard, io.Discard)) {
		args := append(slices.Clone(path), "--help")

		code, out, errOut := run(t, dir, args...)

		if code != 0 {
			t.Fatalf("%v salió con %d: %s", args, code, errOut)
		}

		if found := spanishText.FindString(out); found != "" {
			t.Errorf("la ayuda de %v no está en inglés: encontré %q en\n%s", args, found, out)
		}
	}
}

// commandPaths lists every command of the tree by the arguments that reach it.
func commandPaths(command *cobra.Command) [][]string {
	paths := [][]string{nil}

	for _, child := range command.Commands() {
		for _, below := range commandPaths(child) {
			paths = append(paths, append([]string{child.Name()}, below...))
		}
	}

	return paths
}

// TestInitInstallsTheProduct: instalar nombra su carpeta y el prefijo, y no
// deja nada más en el proyecto.
func TestInitInstallsTheProduct(t *testing.T) {
	t.Parallel()

	project := t.TempDir()

	code, out, errOut := run(t, project, "init")

	if code != 0 {
		t.Fatalf("init salió con %d: %s", code, errOut)
	}

	var installed install.Result
	decode(t, out, &installed)

	if want := filepath.Join(project, install.DirName); installed.Dir != want {
		t.Errorf("la instalación quedó en %q, se esperaba %q", installed.Dir, want)
	}

	if installed.Prefix != "tk" {
		t.Errorf("el prefijo instalado es %q, se esperaba %q", installed.Prefix, "tk")
	}

	entries, err := os.ReadDir(project)
	if err != nil {
		t.Fatalf("leyendo el proyecto: %v", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	if len(names) != 1 || names[0] != install.DirName {
		t.Errorf("la instalación dejó %v en el proyecto, solo debía dejar %q", names, install.DirName)
	}
}

// TestCreateAnswersAndShowReadsTheTask: crear responde con la identidad interna
// y la referencia, y show devuelve la misma tarea.
func TestCreateAnswersAndShowReadsTheTask(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	code, out, errOut := run(t, project, createCmd, "--title", "Preparar el análisis")
	if code != 0 {
		t.Fatalf("create salió con %d: %s", code, errOut)
	}

	var created core.Task
	decode(t, out, &created)

	if created.ID < 1 {
		t.Errorf("la tarea creada no trae identidad interna: %d", created.ID)
	}

	if created.Ref == "" {
		t.Fatal("la tarea creada no trae referencia")
	}

	code, out, errOut = run(t, project, "show", created.Ref)
	if code != 0 {
		t.Fatalf("show %s salió con %d: %s", created.Ref, code, errOut)
	}

	var read core.Task
	decode(t, out, &read)

	if read.ID != created.ID || read.Ref != created.Ref || read.Title != created.Title {
		t.Errorf("show devolvió otra tarea: identidad %d, referencia %q, título %q",
			read.ID, read.Ref, read.Title)
	}
}

// TestShowMissingRefFailsWithAnError: una referencia que no existe falla con el
// código de error de ejecución, no escribe nada por la salida estándar y
// explica el fallo en el JSON de la salida de errores.
func TestShowMissingRefFailsWithAnError(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	code, out, errOut := run(t, project, "show", missingRef)

	if code != 1 {
		t.Fatalf("show con referencia inexistente salió con %d, se esperaba 1", code)
	}

	if out != "" {
		t.Errorf("show escribió en la salida estándar: %s", out)
	}

	if message := errorJSON(t, errOut); !strings.Contains(message, missingRef) {
		t.Errorf("el mensaje de error no menciona la referencia: %s", message)
	}
}

// TestCreateWithoutTitleIsMisuse: crear sin título es un mal uso de la línea de
// comandos y sale con su propio código.
func TestCreateWithoutTitleIsMisuse(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	code, out, errOut := run(t, project, createCmd)

	if code != 2 {
		t.Fatalf("create sin título salió con %d, se esperaba 2", code)
	}

	if out != "" {
		t.Errorf("create escribió en la salida estándar: %s", out)
	}

	errorJSON(t, errOut)
}

// TestUpdateChangesOnlyTheFieldsItIsGiven: cambiar un campo deja los demás como
// estaban.
func TestUpdateChangesOnlyTheFieldsItIsGiven(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	created := createWithFields(t, project)

	code, out, errOut := run(t, project, "update", created.Ref, "--title", "Preparar el informe")
	if code != 0 {
		t.Fatalf("update salió con %d: %s", code, errOut)
	}

	var changed core.Task
	decode(t, out, &changed)

	if changed.Title != "Preparar el informe" {
		t.Errorf("el título quedó en %q, se esperaba %q", changed.Title, "Preparar el informe")
	}

	if changed.ID != created.ID || changed.Ref != created.Ref {
		t.Errorf("el cambio respondió otra tarea: identidad %d y referencia %q", changed.ID, changed.Ref)
	}

	checkUntouched(t, created, changed, "title")

	// Lo que se lee por la referencia es lo que quedó guardado.
	code, out, errOut = run(t, project, "show", created.Ref)
	if code != 0 {
		t.Fatalf("show salió con %d: %s", code, errOut)
	}

	var read core.Task
	decode(t, out, &read)

	if read.Title != changed.Title {
		t.Errorf("la tarea guardada tiene el título %q, se esperaba %q", read.Title, changed.Title)
	}

	checkUntouched(t, created, read, "title")
}

// checkUntouched comprueba que los campos que nadie pidió siguen como estaban.
// changing es el campo que el caso estaba cambiando y por eso se queda fuera.
func checkUntouched(t *testing.T, before, after core.Task, changing string) {
	t.Helper()

	if changing != "title" && after.Title != before.Title {
		t.Errorf("el título cambió sin que nadie lo pidiera: %q", after.Title)
	}

	if changing != "description" && !sameText(before.Description, after.Description) {
		t.Errorf("la descripción cambió sin que nadie lo pidiera: %v", after.Description)
	}

	if !sameText(before.Notes, after.Notes) {
		t.Errorf("las notas cambiaron sin que nadie lo pidiera: %v", after.Notes)
	}

	if !sameText(before.Assignee, after.Assignee) {
		t.Errorf("el responsable cambió sin que nadie lo pidiera: %v", after.Assignee)
	}

	if after.Priority != before.Priority || after.Status != before.Status {
		t.Errorf("la prioridad o el estado cambiaron sin que nadie lo pidiera: %+v", after)
	}
}

// sameText compara dos textos que pueden faltar.
func sameText(before, after *string) bool {
	if before == nil || after == nil {
		return before == nil && after == nil
	}

	return *before == *after
}

// createWithFields deja una tarea con todos sus campos opcionales puestos.
func createWithFields(t *testing.T, project string) core.Task {
	t.Helper()

	code, out, errOut := run(t, project, createCmd,
		"--title", "Preparar el análisis",
		"--description", "El de siempre",
		"--notes", "pendiente de revisión",
		"--assignee", "ana",
	)
	if code != 0 {
		t.Fatalf("create salió con %d: %s", code, errOut)
	}

	var created core.Task
	decode(t, out, &created)

	return created
}

// TestUpdateClearsAnOptionalFieldGivenEmpty: un texto vacío deja ese campo sin
// valor y no toca los demás.
func TestUpdateClearsAnOptionalFieldGivenEmpty(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	created := createWithFields(t, project)

	code, out, errOut := run(t, project, "update", created.Ref, "--description", "")
	if code != 0 {
		t.Fatalf("update con la descripción vacía salió con %d: %s", code, errOut)
	}

	var cleared core.Task
	decode(t, out, &cleared)

	if cleared.Description != nil {
		t.Errorf("vaciar la descripción debía dejarla sin valor, quedó en %q", *cleared.Description)
	}

	checkUntouched(t, created, cleared, "description")
}

// TestUpdateWithoutAnythingToChangeIsMisuse: una orden de cambio sin ningún
// campo es un mal uso de la línea de comandos, con su código y su mensaje.
func TestUpdateWithoutAnythingToChangeIsMisuse(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	code, out, errOut := run(t, project, createCmd, "--title", "Preparar el análisis")
	if code != 0 {
		t.Fatalf("create salió con %d: %s", code, errOut)
	}

	var created core.Task
	decode(t, out, &created)

	code, out, errOut = run(t, project, "update", created.Ref)

	if code != 2 {
		t.Errorf("update sin campos salió con %d, se esperaba 2", code)
	}

	if out != "" {
		t.Errorf("update sin campos escribió en la salida estándar: %s", out)
	}

	if message := errorJSON(t, errOut); !strings.Contains(message, "nothing to change") {
		t.Errorf("el mensaje no dice que no hay nada que cambiar: %s", message)
	}
}

// TestCreateRefusesAPriorityOutOfRange: una prioridad fuera del rango se
// rechaza por los dos extremos y no deja ninguna tarea guardada.
func TestCreateRefusesAPriorityOutOfRange(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	for _, priority := range []string{"-1", "4"} {
		code, out, errOut := run(t, project, createCmd, "--title", "Preparar el análisis", "--priority="+priority)

		if code != 2 {
			t.Errorf("create con la prioridad %s salió con %d, se esperaba 2", priority, code)
		}

		if out != "" {
			t.Errorf("create con la prioridad %s escribió en la salida estándar: %s", priority, out)
		}

		if message := errorJSON(t, errOut); !strings.Contains(message, "range") {
			t.Errorf("el mensaje de la prioridad %s no explica el rango: %s", priority, message)
		}
	}

	// Ninguna de las dos rechazadas dejó tarea: la siguiente lleva la primera identidad.
	code, out, errOut := run(t, project, createCmd, "--title", "Preparar el análisis")
	if code != 0 {
		t.Fatalf("create salió con %d: %s", code, errOut)
	}

	var created core.Task
	decode(t, out, &created)

	if created.ID != 1 {
		t.Errorf("una tarea rechazada consumió identidad: la siguiente es la %d", created.ID)
	}
}

// TestUpdateRefusesAPriorityOutOfRangeAndLeavesTheTaskAsItWas: el rango tiene
// dos extremos válidos y todo lo que se salga de él se rechaza sin tocar la
// tarea guardada.
func TestUpdateRefusesAPriorityOutOfRangeAndLeavesTheTaskAsItWas(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	code, out, errOut := run(t, project, createCmd, "--title", "Preparar el análisis")
	if code != 0 {
		t.Fatalf("create salió con %d: %s", code, errOut)
	}

	var created core.Task
	decode(t, out, &created)

	for _, priority := range []string{"-1", "4"} {
		code, out, errOut = run(t, project, "update", created.Ref, "--priority="+priority)

		if code != 2 {
			t.Errorf("update con la prioridad %s salió con %d, se esperaba 2", priority, code)
		}

		if out != "" {
			t.Errorf("update con la prioridad %s escribió en la salida estándar: %s", priority, out)
		}

		if message := errorJSON(t, errOut); !strings.Contains(message, "range") {
			t.Errorf("el mensaje de la prioridad %s no explica el rango: %s", priority, message)
		}
	}

	code, out, errOut = run(t, project, "show", created.Ref)
	if code != 0 {
		t.Fatalf("show salió con %d: %s", code, errOut)
	}

	var read core.Task
	decode(t, out, &read)

	if read.Priority != created.Priority {
		t.Errorf("la prioridad rechazada tocó la tarea: quedó en %d", read.Priority)
	}

	// Los dos extremos del rango sí se aceptan.
	for _, priority := range []int{core.MinPriority, core.MaxPriority} {
		code, out, errOut = run(t, project, "update", created.Ref, fmt.Sprintf("--priority=%d", priority))
		if code != 0 {
			t.Fatalf("update con la prioridad %d salió con %d: %s", priority, code, errOut)
		}

		var changed core.Task
		decode(t, out, &changed)

		if changed.Priority != priority {
			t.Errorf("la prioridad quedó en %d, se esperaba %d", changed.Priority, priority)
		}
	}
}

// readTask deja leída la tarea que apunta la referencia.
func readTask(t *testing.T, project, ref string) core.Task {
	t.Helper()

	code, out, errOut := run(t, project, "show", ref)
	if code != 0 {
		t.Fatalf("show %s salió con %d: %s", ref, code, errOut)
	}

	var task core.Task
	decode(t, out, &task)

	return task
}

// createTask deja creada una tarea con los argumentos que se le añadan y
// devuelve la respuesta del create.
func createTask(t *testing.T, project string, args ...string) core.Task {
	t.Helper()

	code, out, errOut := run(t, project, append([]string{createCmd, "--title", "Preparar el análisis"}, args...)...)
	if code != 0 {
		t.Fatalf("create salió con %d: %s", code, errOut)
	}

	var created core.Task
	decode(t, out, &created)

	return created
}

// TestCloseRefusesATaskWithOpenSubtasksAndLeavesItAsItWas: cerrar una tarea con
// subtareas abiertas se rechaza y no toca la tarea guardada.
func TestCloseRefusesATaskWithOpenSubtasksAndLeavesItAsItWas(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	parent := createTask(t, project)
	createTask(t, project, "--parent", parent.Ref)

	before := readTask(t, project, parent.Ref)

	code, out, errOut := run(t, project, "close", parent.Ref, "--reason", "done")

	if code != 1 {
		t.Fatalf("cerrar una tarea con subtareas abiertas salió con %d, se esperaba 1", code)
	}

	if out != "" {
		t.Errorf("el cierre rechazado escribió en la salida estándar: %s", out)
	}

	if message := errorJSON(t, errOut); !strings.Contains(message, "open subtasks") {
		t.Errorf("el mensaje no dice que hay subtareas abiertas: %s", message)
	}

	after := readTask(t, project, parent.Ref)

	if !reflect.DeepEqual(after, before) {
		t.Errorf("el cierre rechazado tocó la tarea: antes %+v, después %+v", before, after)
	}
}

// TestClosingASubtaskLeavesItsParentOpen: cerrar una subtarea no cierra su
// tarea padre.
func TestClosingASubtaskLeavesItsParentOpen(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	parent := createTask(t, project)
	child := createTask(t, project, "--parent", parent.Ref)

	code, out, errOut := run(t, project, "close", child.Ref, "--reason", "done")
	if code != 0 {
		t.Fatalf("close salió con %d: %s", code, errOut)
	}

	var closed core.Task
	decode(t, out, &closed)

	if closed.Status != "done" || closed.CloseReason == nil || *closed.CloseReason != "done" {
		t.Errorf("la subtarea quedó %+v, se esperaba cerrada con motivo done", closed)
	}

	read := readTask(t, project, parent.Ref)

	if read.Status != statusOpen || read.CloseReason != nil {
		t.Errorf("cerrar la subtarea cerró también a su padre: %+v", read)
	}
}

// TestCloseRefusesAnUnknownOrMissingReason: el motivo es obligatorio y debe ser
// uno de los conocidos; cualquier otra cosa es un mal uso y no toca la tarea.
func TestCloseRefusesAnUnknownOrMissingReason(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	task := createTask(t, project)

	code, out, errOut := run(t, project, "close", task.Ref)

	if code != 2 {
		t.Errorf("close sin motivo salió con %d, se esperaba 2", code)
	}

	if out != "" {
		t.Errorf("close sin motivo escribió en la salida estándar: %s", out)
	}

	if message := errorJSON(t, errOut); !strings.Contains(message, "reason") {
		t.Errorf("el mensaje de close sin motivo no habla del motivo: %s", message)
	}

	code, out, errOut = run(t, project, "close", task.Ref, "--reason", "vanished")

	if code != 2 {
		t.Errorf("close con un motivo desconocido salió con %d, se esperaba 2", code)
	}

	if out != "" {
		t.Errorf("close con un motivo desconocido escribió en la salida estándar: %s", out)
	}

	if message := errorJSON(t, errOut); !strings.Contains(message, "reason") {
		t.Errorf("el mensaje del motivo desconocido no habla del motivo: %s", message)
	}

	read := readTask(t, project, task.Ref)

	if read.Status != statusOpen || read.CloseReason != nil {
		t.Errorf("un cierre rechazado tocó la tarea: %+v", read)
	}
}

// TestCloseThenReopenReturnsTheTaskToOpen: reabrir devuelve la tarea a abierta
// y le quita el motivo de cierre.
func TestCloseThenReopenReturnsTheTaskToOpen(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	task := createTask(t, project)

	code, out, errOut := run(t, project, "close", task.Ref, "--reason", "duplicate")
	if code != 0 {
		t.Fatalf("close salió con %d: %s", code, errOut)
	}

	var closed core.Task
	decode(t, out, &closed)

	if closed.Status != "done" || closed.CloseReason == nil || *closed.CloseReason != "duplicate" {
		t.Errorf("la tarea cerrada quedó %+v, se esperaba done con motivo duplicate", closed)
	}

	code, out, errOut = run(t, project, "reopen", task.Ref)
	if code != 0 {
		t.Fatalf("reopen salió con %d: %s", code, errOut)
	}

	var reopened core.Task
	decode(t, out, &reopened)

	if reopened.Status != statusOpen || reopened.CloseReason != nil {
		t.Errorf("reabrir dejó la tarea %+v, se esperaba abierta y sin motivo de cierre", reopened)
	}

	read := readTask(t, project, task.Ref)

	if read.Status != statusOpen || read.CloseReason != nil {
		t.Errorf("la tarea guardada tras reabrir es %+v, se esperaba abierta y sin motivo", read)
	}
}

// TestCreateWithAParentLinksItAndRefusesAnUnknownOne: --parent da a la tarea su
// tarea padre, y un padre que no existe se rechaza como una referencia que no
// resuelve.
func TestCreateWithAParentLinksItAndRefusesAnUnknownOne(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	parent := createTask(t, project)
	child := createTask(t, project, "--parent", parent.Ref)

	if child.ParentID == nil || *child.ParentID != parent.ID {
		t.Errorf("la subtarea nació con padre %v, se esperaba %d", child.ParentID, parent.ID)
	}

	code, out, errOut := run(t, project, createCmd, "--title", "Huérfana", "--parent", missingRef)

	if code != 1 {
		t.Errorf("create con un padre inexistente salió con %d, se esperaba 1", code)
	}

	if out != "" {
		t.Errorf("create con un padre inexistente escribió en la salida estándar: %s", out)
	}

	if message := errorJSON(t, errOut); !strings.Contains(message, missingRef) {
		t.Errorf("el mensaje no menciona la referencia del padre: %s", message)
	}
}

// TestCloseTwiceAndReopenAnOpenTaskAreRefused: una tarea cerrada no se cierra
// otra vez y una abierta no se reabre; el rechazo no toca la tarea.
func TestCloseTwiceAndReopenAnOpenTaskAreRefused(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	task := createTask(t, project)

	code, _, errOut := run(t, project, "close", task.Ref, "--reason", "done")
	if code != 0 {
		t.Fatalf("close salió con %d: %s", code, errOut)
	}

	closed := readTask(t, project, task.Ref)

	code, out, errOut := run(t, project, "close", task.Ref, "--reason", "stale")

	if code != 1 {
		t.Errorf("el segundo close salió con %d, se esperaba 1", code)
	}

	if out != "" {
		t.Errorf("el segundo close escribió en la salida estándar: %s", out)
	}

	if message := errorJSON(t, errOut); !strings.Contains(message, "already closed") {
		t.Errorf("el mensaje no dice que la tarea ya está cerrada: %s", message)
	}

	if again := readTask(t, project, task.Ref); !reflect.DeepEqual(again, closed) {
		t.Errorf("el segundo close tocó la tarea: antes %+v, después %+v", closed, again)
	}

	code, _, errOut = run(t, project, "reopen", task.Ref)
	if code != 0 {
		t.Fatalf("reopen salió con %d: %s", code, errOut)
	}

	code, out, errOut = run(t, project, "reopen", task.Ref)

	if code != 1 {
		t.Errorf("el segundo reopen salió con %d, se esperaba 1", code)
	}

	if out != "" {
		t.Errorf("el segundo reopen escribió en la salida estándar: %s", out)
	}

	if message := errorJSON(t, errOut); !strings.Contains(message, "not closed") {
		t.Errorf("el mensaje no dice que la tarea no está cerrada: %s", message)
	}
}

// blockedBody trae lo que hacen falta para leer el estado bloqueado y las
// listas del detalle, sin depender de los tipos del núcleo.
type blockedBody struct {
	Ref        string        `json:"ref"`
	Blocked    bool          `json:"blocked"`
	Status     string        `json:"status"`
	Tags       []string      `json:"tags"`
	Children   []blockedBody `json:"children"`
	Blockers   []blockedBody `json:"blockers"`
	Dependents []blockedBody `json:"dependents"`
	Comments   []commentBody `json:"comments"`
}

// commentBody trae lo que un comentario muestra: su cuerpo y el momento en que
// se añadió.
type commentBody struct {
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

// showDetail deja leído el detalle de la tarea que apunta la referencia.
func showDetail(t *testing.T, project, ref string) blockedBody {
	t.Helper()

	code, out, errOut := run(t, project, "show", ref)
	if code != 0 {
		t.Fatalf("show %s salió con %d: %s", ref, code, errOut)
	}

	var detail blockedBody
	decode(t, out, &detail)

	return detail
}

// showRaw devuelve la respuesta íntegra de show, para comparar que una
// operación rechazada no dejó los datos como estaban.
func showRaw(t *testing.T, project, ref string) string {
	t.Helper()

	code, out, errOut := run(t, project, "show", ref)
	if code != 0 {
		t.Fatalf("show %s salió con %d: %s", ref, code, errOut)
	}

	return out
}

// refs devuelve las referencias de una lista de tareas del detalle.
func refs(list []blockedBody) []string {
	found := make([]string, 0, len(list))
	for _, task := range list {
		found = append(found, task.Ref)
	}

	return found
}

// sameRefs comprueba que la lista del detalle trae las referencias que se
// esperan, en el orden en que se esperan.
func sameRefs(t *testing.T, got []blockedBody, want ...string) {
	t.Helper()

	if found := refs(got); !slices.Equal(found, want) {
		t.Errorf("la lista es %v, se esperaba %v", found, want)
	}
}

// TestBlockMarksTheTaskBlockedAndUnblockClearsIt: una tarea con un bloqueo
// abierto aparece bloqueada y deja de estarlo al retirarlo.
func TestBlockMarksTheTaskBlockedAndUnblockClearsIt(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	blocked := createTask(t, project)
	blocker := createTask(t, project)

	code, out, errOut := run(t, project, "block", blocked.Ref, blocker.Ref)
	if code != 0 {
		t.Fatalf("block salió con %d: %s", code, errOut)
	}

	var answered blockedBody
	decode(t, out, &answered)

	if answered.Ref != blocked.Ref || !answered.Blocked {
		t.Errorf("block respondió %+v, se esperaba la tarea %q bloqueada", answered, blocked.Ref)
	}

	detail := showDetail(t, project, blocked.Ref)

	if !detail.Blocked {
		t.Error("la tarea con un bloqueo abierto no aparece bloqueada")
	}

	sameRefs(t, detail.Blockers, blocker.Ref)

	code, out, errOut = run(t, project, "unblock", blocked.Ref, blocker.Ref)
	if code != 0 {
		t.Fatalf("unblock salió con %d: %s", code, errOut)
	}

	decode(t, out, &answered)

	if answered.Ref != blocked.Ref || answered.Blocked {
		t.Errorf("unblock respondió %+v, se esperaba la tarea %q sin bloqueo", answered, blocked.Ref)
	}

	detail = showDetail(t, project, blocked.Ref)

	if detail.Blocked {
		t.Error("retirado el bloqueo, la tarea sigue apareciendo bloqueada")
	}

	sameRefs(t, detail.Blockers)
}

// TestTheBlockedStateFollowsTheBlocker: el estado bloqueado se calcula desde
// los bloqueos, así que cerrar al bloqueante la libera y reabrirlo la vuelve a
// bloquear.
func TestTheBlockedStateFollowsTheBlocker(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	blocked := createTask(t, project)
	blocker := createTask(t, project)

	code, _, errOut := run(t, project, "block", blocked.Ref, blocker.Ref)
	if code != 0 {
		t.Fatalf("block salió con %d: %s", code, errOut)
	}

	code, _, errOut = run(t, project, "close", blocker.Ref, "--reason", "done")
	if code != 0 {
		t.Fatalf("close salió con %d: %s", code, errOut)
	}

	if detail := showDetail(t, project, blocked.Ref); detail.Blocked {
		t.Error("cerrado el bloqueante, la tarea sigue apareciendo bloqueada")
	}

	code, _, errOut = run(t, project, "reopen", blocker.Ref)
	if code != 0 {
		t.Fatalf("reopen salió con %d: %s", code, errOut)
	}

	if detail := showDetail(t, project, blocked.Ref); !detail.Blocked {
		t.Error("reabierto el bloqueante, la tarea dejó de aparecer bloqueada")
	}
}

// TestBlockAndUnblockRefuseAnUnknownReference: una referencia que no existe se
// rechaza como la rechaza show, sin escribir nada y sin tocar los datos.
func TestBlockAndUnblockRefuseAnUnknownReference(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	task := createTask(t, project)
	before := showRaw(t, project, task.Ref)

	for _, args := range [][]string{
		{"block", task.Ref, missingRef},
		{"block", missingRef, task.Ref},
		{"unblock", task.Ref, missingRef},
		{"unblock", missingRef, task.Ref},
	} {
		code, out, errOut := run(t, project, args...)

		if code != 1 {
			t.Errorf("%v salió con %d, se esperaba 1", args, code)
		}

		if out != "" {
			t.Errorf("%v escribió en la salida estándar: %s", args, out)
		}

		if message := errorJSON(t, errOut); !strings.Contains(message, missingRef) {
			t.Errorf("el mensaje de %v no menciona la referencia desconocida: %s", args, message)
		}
	}

	if after := showRaw(t, project, task.Ref); after != before {
		t.Errorf("un bloqueo rechazado tocó los datos: antes %s, después %s", before, after)
	}
}

// TestBlockRefusesBlockingATaskByItself: una tarea no puede bloquearse por sí
// misma; es un mal uso de la línea de comandos y no toca los datos.
func TestBlockRefusesBlockingATaskByItself(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	task := createTask(t, project)
	before := showRaw(t, project, task.Ref)

	for _, name := range []string{"block", "unblock"} {
		code, out, errOut := run(t, project, name, task.Ref, task.Ref)

		if code != 2 {
			t.Errorf("%s con la misma referencia dos veces salió con %d, se esperaba 2", name, code)
		}

		if out != "" {
			t.Errorf("%s con la misma referencia dos veces escribió en la salida estándar: %s", name, out)
		}

		if message := errorJSON(t, errOut); !strings.Contains(message, task.Ref) {
			t.Errorf("el mensaje de %s no menciona la tarea: %s", name, message)
		}
	}

	if after := showRaw(t, project, task.Ref); after != before {
		t.Errorf("un bloqueo rechazado tocó los datos: antes %s, después %s", before, after)
	}
}

// TestShowAddsTheChildrenTheBlockersTheDependentsAndTheComments: el detalle de
// una tarea trae sus hijas, sus bloqueantes, sus dependientes y sus comentarios,
// y el de una tarea ajena no se ve tocado por los bloqueos de las demás.
func TestShowAddsTheChildrenTheBlockersTheDependentsAndTheComments(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	parent := createTask(t, project)
	child1 := createTask(t, project, "--parent", parent.Ref)
	child2 := createTask(t, project, "--parent", parent.Ref)
	dependent := createTask(t, project)
	unrelated := createTask(t, project)

	code, _, errOut := run(t, project, "comment", parent.Ref, "--body", "primer apunte")
	if code != 0 {
		t.Fatalf("comment salió con %d: %s", code, errOut)
	}

	code, _, errOut = run(t, project, "block", dependent.Ref, parent.Ref)
	if code != 0 {
		t.Fatalf("block salió con %d: %s", code, errOut)
	}

	detail := showDetail(t, project, parent.Ref)

	sameRefs(t, detail.Children, child1.Ref, child2.Ref)
	sameRefs(t, detail.Blockers)
	sameRefs(t, detail.Dependents, dependent.Ref)

	if len(detail.Comments) != 1 {
		t.Fatalf("los comentarios son %+v, se esperaba uno", detail.Comments)
	}

	if detail.Comments[0].Body != "primer apunte" {
		t.Errorf("el cuerpo del comentario es %q, se esperaba %q", detail.Comments[0].Body, "primer apunte")
	}

	if _, err := time.Parse(time.RFC3339, detail.Comments[0].CreatedAt); err != nil {
		t.Errorf("el comentario no trae un momento que se entienda: %q", detail.Comments[0].CreatedAt)
	}

	detail = showDetail(t, project, dependent.Ref)

	sameRefs(t, detail.Blockers, parent.Ref)
	sameRefs(t, detail.Dependents)
	sameRefs(t, detail.Children)

	detail = showDetail(t, project, unrelated.Ref)

	sameRefs(t, detail.Children)
	sameRefs(t, detail.Blockers)
	sameRefs(t, detail.Dependents)

	if len(detail.Comments) != 0 {
		t.Errorf("una tarea ajena trae comentarios: %+v", detail.Comments)
	}
}

// TestCommentAppendsAndRefusesAnEmptyBody: el comentario se añade con su cuerpo,
// que es obligatorio; sin él o vacío se rechaza sin añadir nada.
func TestCommentAppendsAndRefusesAnEmptyBody(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	task := createTask(t, project)

	code, out, errOut := run(t, project, "comment", task.Ref, "--body", "primer apunte")
	if code != 0 {
		t.Fatalf("comment salió con %d: %s", code, errOut)
	}

	var answered core.Task
	decode(t, out, &answered)

	if answered.Ref != task.Ref {
		t.Errorf("comment respondió otra tarea: %q en vez de %q", answered.Ref, task.Ref)
	}

	detail := showDetail(t, project, task.Ref)

	if len(detail.Comments) != 1 || detail.Comments[0].Body != "primer apunte" {
		t.Fatalf("los comentarios son %+v, se esperaba uno con el cuerpo %q", detail.Comments, "primer apunte")
	}

	for _, body := range []string{"", "   "} {
		code, out, errOut := run(t, project, "comment", task.Ref, "--body", body)

		if code != 2 {
			t.Errorf("comment con el cuerpo %q salió con %d, se esperaba 2", body, code)
		}

		if out != "" {
			t.Errorf("comment con el cuerpo %q escribió en la salida estándar: %s", body, out)
		}

		if message := errorJSON(t, errOut); !strings.Contains(message, "body") {
			t.Errorf("el mensaje de comment con el cuerpo %q no habla del cuerpo: %s", body, message)
		}
	}

	code, out, _ = run(t, project, "comment", task.Ref)

	if code != 2 {
		t.Errorf("comment sin cuerpo salió con %d, se esperaba 2", code)
	}

	if out != "" {
		t.Errorf("comment sin cuerpo escribió en la salida estándar: %s", out)
	}

	if detail := showDetail(t, project, task.Ref); len(detail.Comments) != 1 {
		t.Errorf("un comentario rechazado añadió algo: quedaron %+v", detail.Comments)
	}
}

// TestTheCommentCannotBeChangedOnceAdded: el comentario añadido no se puede
// editar: la herramienta no ofrece ningún comando que cambie uno.
func TestTheCommentCannotBeChangedOnceAdded(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	task := createTask(t, project)

	code, _, errOut := run(t, project, "comment", task.Ref, "--body", "primer apunte")
	if code != 0 {
		t.Fatalf("comment salió con %d: %s", code, errOut)
	}

	before := showRaw(t, project, task.Ref)

	for _, name := range []string{"edit-comment", "delete-comment", "uncomment", "update-comment"} {
		code, out, _ := run(t, project, name, task.Ref)

		if code == 0 {
			t.Errorf("%s no debía existir y salió bien", name)
		}

		if out != "" {
			t.Errorf("%s escribió en la salida estándar: %s", name, out)
		}
	}

	if after := showRaw(t, project, task.Ref); after != before {
		t.Errorf("un intento de cambiar el comentario tocó los datos: antes %s, después %s", before, after)
	}
}

// TestInitTwiceFails: instalar dos veces falla con un mensaje claro.
func TestInitTwiceFails(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	code, out, errOut := run(t, project, "init")

	if code != 1 {
		t.Fatalf("el segundo init salió con %d, se esperaba 1", code)
	}

	if out != "" {
		t.Errorf("el segundo init escribió en la salida estándar: %s", out)
	}

	if message := errorJSON(t, errOut); !strings.Contains(message, "installed") {
		t.Errorf("el mensaje no dice que el producto ya está instalado: %s", message)
	}
}

// TestTwoCreatesOnABrandNewBase: dos procesos creando a la vez sobre una base
// que todavía no existe deben salir los dos bien y dejar las dos tareas
// legibles. La carrera la arregló la base; aquí se guarda a través del binario
// real. No se comprueba el entrelazado de los dos procesos, que no es
// determinista, sino el resultado que debe cumplirse siempre.
func TestTwoCreatesOnABrandNewBase(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	// El proyecto queda instalado pero sin base: se quita para que los dos
	// procesos la encuentren inexistente y compitan por crearla.
	base := filepath.Join(project, install.DirName, "tasks.db")

	for _, suffix := range []string{"", "-wal", "-shm"} {
		if err := os.Remove(base + suffix); err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("quitando la base %s: %v", base+suffix, err)
		}
	}

	const attempts = 2

	runs := make([]createRun, attempts)
	for i := range runs {
		//nolint:gosec // el binario es el propio producto y los argumentos, de prueba
		runs[i].cmd = exec.CommandContext(t.Context(), binPath, createCmd, "--title", fmt.Sprintf("Tarea %d", i))
		runs[i].cmd.Dir = project
		runs[i].cmd.Stdout = &runs[i].out
		runs[i].cmd.Stderr = &runs[i].errOut

		if err := runs[i].cmd.Start(); err != nil {
			t.Fatalf("arrancando el create %d: %v", i, err)
		}
	}

	refs := make([]string, attempts)
	for i := range runs {
		refs[i] = collectCreate(t, &runs[i])
	}

	if refs[0] != "" && refs[0] == refs[1] {
		t.Errorf("los dos creates devolvieron la misma referencia %q", refs[0])
	}

	for _, ref := range refs {
		if ref == "" {
			continue
		}

		code, out, errOut := run(t, project, "show", ref)
		if code != 0 {
			t.Fatalf("show %s salió con %d: %s", ref, code, errOut)
		}

		var read core.Task
		decode(t, out, &read)

		if read.Ref != ref {
			t.Errorf("show %s devolvió la tarea con la referencia %q", ref, read.Ref)
		}
	}
}

// createRun es un create ya arrancado, con sus dos salidas a la espera.
type createRun struct {
	cmd    *exec.Cmd
	out    bytes.Buffer
	errOut bytes.Buffer
}

// collectCreate espera uno de los creates arrancados y, si salió bien, devuelve
// la referencia que respondió; vacía significa que falló. Va por puntero: los
// búferes de las salidas viven en el elemento original y una copia los vería vacíos.
func collectCreate(t *testing.T, started *createRun) string {
	t.Helper()

	err := started.cmd.Wait()

	code := 0

	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("esperando un create: %v", err)
		}

		code = exitErr.ExitCode()
	}

	if code != 0 {
		t.Errorf("un create salió con %d: %s", code, started.errOut.String())

		return ""
	}

	var created core.Task
	decode(t, started.out.String(), &created)

	return created.Ref
}

// tagBody trae lo que el catálogo muestra de una etiqueta: su identidad, su
// nombre, su color, su descripción y el número de tareas que la usan.
type tagBody struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Color       string  `json:"color"`
	Description *string `json:"description"`
	Uses        int     `json:"uses"`
}

// Los nombres que los escenarios de etiquetas repiten.
const (
	tagCmd       = "tag"
	createCmd    = "create"
	greenColor   = "green"
	unknownColor = "desconocida"
)

// createTag deja creada una etiqueta con los argumentos que se le añadan y
// devuelve la etiqueta guardada.
func createTag(t *testing.T, project string, args ...string) tagBody {
	t.Helper()

	code, out, errOut := run(t, project, append([]string{tagCmd, createCmd}, args...)...)
	if code != 0 {
		t.Fatalf("tag create salió con %d: %s", code, errOut)
	}

	var created tagBody
	decode(t, out, &created)

	return created
}

// listTags deja leído el catálogo de etiquetas.
func listTags(t *testing.T, project string) []tagBody {
	t.Helper()

	code, out, errOut := run(t, project, "tags")
	if code != 0 {
		t.Fatalf("tags salió con %d: %s", code, errOut)
	}

	var catalogue []tagBody
	decode(t, out, &catalogue)

	return catalogue
}

// setTags deja a la tarea con las etiquetas que se le dan, escritas como las
// escribe un usuario en --tags.
func setTags(t *testing.T, project, ref, list string) core.Task {
	t.Helper()

	code, out, errOut := run(t, project, "update", ref, "--tags", list)
	if code != 0 {
		t.Fatalf("update %s --tags %q salió con %d: %s", ref, list, code, errOut)
	}

	var changed core.Task
	decode(t, out, &changed)

	return changed
}

// sameCatalogue comprueba que el catálogo quedó como se esperaba.
func sameCatalogue(t *testing.T, got, want []tagBody) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("el catálogo es %+v, se esperaba %+v", got, want)
	}
}

// wantTag compara la etiqueta que llegó con la que se esperaba. La identidad
// interna se queda fuera: la asigna la base y nada depende del valor concreto.
func wantTag(t *testing.T, got, want tagBody) {
	t.Helper()

	want.ID = got.ID

	if !reflect.DeepEqual(got, want) {
		t.Errorf("la etiqueta es %+v, se esperaba %+v", got, want)
	}
}

// wantTaskTags comprueba que la tarea quedó con las etiquetas que se esperaban.
func wantTaskTags(t *testing.T, project, ref string, want ...string) {
	t.Helper()

	if tags := readTask(t, project, ref).Tags; !slices.Equal(tags, want) {
		t.Errorf("la tarea %s quedó con las etiquetas %v, se esperaba %v", ref, tags, want)
	}
}

// wantNamedIn comprueba que el aviso nombre las referencias que se esperan y
// calla sobre las que no.
func wantNamedIn(t *testing.T, warning string, mentioned, absent []string) {
	t.Helper()

	for _, name := range mentioned {
		if !strings.Contains(warning, name) {
			t.Errorf("el aviso no menciona a %q: %s", name, warning)
		}
	}

	for _, name := range absent {
		if strings.Contains(warning, name) {
			t.Errorf("el aviso menciona a %q, que no venía al caso: %s", name, warning)
		}
	}
}

// TestTagsListsTheCatalogueOrderedByNameWithItsUses: el comando tags responde
// el catálogo ordenado por nombre, y cada etiqueta con su nombre, su color, su
// descripción y cuántas tareas la usan.
func TestTagsListsTheCatalogueOrderedByNameWithItsUses(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	description := "Documentación del proyecto"

	createTag(t, project, "--name", "zeta")
	createTag(t, project, "--name", "alpha", "--color", greenColor, "--description", description)
	createTag(t, project, "--name", "media", "--color", "red")

	first := createTask(t, project)
	second := createTask(t, project)
	setTags(t, project, first.Ref, "alpha")
	setTags(t, project, second.Ref, "alpha, media")

	catalogue := listTags(t, project)

	if len(catalogue) != 3 {
		t.Fatalf("el catálogo es %+v, se esperaban tres etiquetas", catalogue)
	}

	if catalogue[0].Name != "alpha" || catalogue[1].Name != "media" || catalogue[2].Name != "zeta" {
		t.Errorf("el catálogo está en el orden %v, se esperaba alpha, media, zeta",
			[]string{catalogue[0].Name, catalogue[1].Name, catalogue[2].Name})
	}

	if catalogue[0].Uses != 2 {
		t.Errorf("alpha aparece usada %d veces, se esperaban 2", catalogue[0].Uses)
	}

	if catalogue[2].Uses != 0 {
		t.Errorf("zeta aparece usada %d veces, se esperaban 0", catalogue[2].Uses)
	}

	if catalogue[0].Color != greenColor {
		t.Errorf("alpha quedó con el color %q, se esperaba green", catalogue[0].Color)
	}

	if catalogue[0].Description == nil || *catalogue[0].Description != description {
		t.Errorf("alpha quedó con la descripción %v, se esperaba %q", catalogue[0].Description, description)
	}
}

// TestTagCreateGivesTheDefaultColourAndRefusesBadEntry: una etiqueta a la que
// nadie le dio color lleva el de por omisión; un color desconocido y un nombre
// que falta son entrada inválida y no dejan nada en el catálogo.
func TestTagCreateGivesTheDefaultColourAndRefusesBadEntry(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	created := createTag(t, project, "--name", "media")

	if created.Color != "blue" {
		t.Errorf("la etiqueta sin color quedó con %q, se esperaba blue", created.Color)
	}

	if created.Description != nil {
		t.Errorf("la etiqueta sin descripción quedó con %q", *created.Description)
	}

	before := listTags(t, project)

	for _, args := range [][]string{
		{tagCmd, createCmd, "--name", "roja", "--color", "magenta"},
		{tagCmd, createCmd, "--name", "   "},
		{tagCmd, createCmd},
	} {
		code, out, errOut := run(t, project, args...)

		if code != 2 {
			t.Errorf("%v salió con %d, se esperaba 2", args, code)
		}

		if out != "" {
			t.Errorf("%v escribió en la salida estándar: %s", args, out)
		}

		if message := errorJSON(t, errOut); !strings.Contains(message, "color") && !strings.Contains(message, "name") {
			t.Errorf("el mensaje de %v no habla del color o del nombre: %s", args, message)
		}
	}

	sameCatalogue(t, listTags(t, project), before)
}

// TestTagCreateRefusesADuplicateNameAndLeavesTheCatalogueAsItWas: un nombre que
// ya existe se rechaza como lo rechaza la regla que lo impone, y el catálogo
// queda tal como estaba.
func TestTagCreateRefusesADuplicateNameAndLeavesTheCatalogueAsItWas(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	createTag(t, project, "--name", "media", "--color", "red")
	before := listTags(t, project)

	code, out, errOut := run(t, project, tagCmd, createCmd, "--name", "media", "--color", greenColor)

	if code != 1 {
		t.Fatalf("tag create con un nombre repetido salió con %d, se esperaba 1", code)
	}

	if out != "" {
		t.Errorf("tag create con un nombre repetido escribió en la salida estándar: %s", out)
	}

	message := errorJSON(t, errOut)

	if !strings.Contains(message, "media") {
		t.Errorf("el mensaje no menciona el nombre repetido: %s", message)
	}

	// Lo que lee quien usa el producto no lleva dentro el texto del motor de la
	// base, ni repite la misma frase por envolverla dos veces.
	if strings.Contains(message, "constraint") || strings.Contains(message, "SQLITE") {
		t.Errorf("el mensaje enseña el error del motor de la base: %s", message)
	}

	if count := strings.Count(message, "already exists"); count != 1 {
		t.Errorf("el mensaje dice lo mismo %d veces: %s", count, message)
	}

	sameCatalogue(t, listTags(t, project), before)
}

// TestTagEditChangesOnlyWhatItIsGiven: un cambio de etiqueta toca solo los
// campos que se piden, y un cambio sin ningún campo es un mal uso.
func TestTagEditChangesOnlyWhatItIsGiven(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	description := "El de siempre"

	createTag(t, project, "--name", "media", "--color", "red", "--description", description)

	code, out, errOut := run(t, project, tagCmd, "edit", "media", "--color", greenColor)
	if code != 0 {
		t.Fatalf("tag edit salió con %d: %s", code, errOut)
	}

	var changed tagBody
	decode(t, out, &changed)

	wantTag(t, changed, tagBody{Name: "media", Color: greenColor, Description: &description})

	code, out, errOut = run(t, project, tagCmd, "edit", "media", "--name", "nueva")
	if code != 0 {
		t.Fatalf("tag edit salió con %d: %s", code, errOut)
	}

	decode(t, out, &changed)

	wantTag(t, changed, tagBody{Name: "nueva", Color: greenColor, Description: &description})

	if catalogue := listTags(t, project); len(catalogue) != 1 || catalogue[0].Name != "nueva" {
		t.Errorf("el catálogo es %+v, se esperaba una sola etiqueta nueva", catalogue)
	}

	code, out, errOut = run(t, project, tagCmd, "edit", "nueva")

	if code != 2 {
		t.Errorf("tag edit sin campos salió con %d, se esperaba 2", code)
	}

	if out != "" {
		t.Errorf("tag edit sin campos escribió en la salida estándar: %s", out)
	}

	if message := errorJSON(t, errOut); !strings.Contains(message, "nothing to change") {
		t.Errorf("el mensaje no dice que no hay nada que cambiar: %s", message)
	}
}

// TestTagEditRefusesARenameOntoAnExistingNameAndLeavesEverythingAsItWas:
// renombrar una etiqueta sobre el nombre de otra se rechaza y no toca ni el
// catálogo ni las tareas.
func TestTagEditRefusesARenameOntoAnExistingNameAndLeavesEverythingAsItWas(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	createTag(t, project, "--name", "una")
	createTag(t, project, "--name", "dos", "--color", "red")
	task := createTask(t, project)
	setTags(t, project, task.Ref, "una")

	before := listTags(t, project)
	taskBefore := showRaw(t, project, task.Ref)

	code, out, errOut := run(t, project, tagCmd, "edit", "una", "--name", "dos")

	if code != 1 {
		t.Fatalf("renombrar sobre un nombre existente salió con %d, se esperaba 1", code)
	}

	if out != "" {
		t.Errorf("el renombrado rechazado escribió en la salida estándar: %s", out)
	}

	if message := errorJSON(t, errOut); !strings.Contains(message, "dos") {
		t.Errorf("el mensaje no menciona el nombre que ya existía: %s", message)
	}

	sameCatalogue(t, listTags(t, project), before)

	if taskAfter := showRaw(t, project, task.Ref); taskAfter != taskBefore {
		t.Errorf("el renombrado rechazado tocó la tarea: antes %s, después %s", taskBefore, taskAfter)
	}
}

// TestTagEditAndDeleteRefuseAnUnknownName: un nombre que no apunta a ninguna
// etiqueta se rechaza como show rechaza una referencia que no resuelve.
func TestTagEditAndDeleteRefuseAnUnknownName(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	createTag(t, project, "--name", "media")
	before := listTags(t, project)

	for _, args := range [][]string{
		{tagCmd, "edit", unknownColor, "--color", greenColor},
		{tagCmd, "delete", unknownColor},
	} {
		code, out, errOut := run(t, project, args...)

		if code != 1 {
			t.Errorf("%v salió con %d, se esperaba 1", args, code)
		}

		if out != "" {
			t.Errorf("%v escribió en la salida estándar: %s", args, out)
		}

		if message := errorJSON(t, errOut); !strings.Contains(message, unknownColor) {
			t.Errorf("el mensaje de %v no menciona el nombre: %s", args, message)
		}
	}

	sameCatalogue(t, listTags(t, project), before)
}

// TestRenamingATagChangesNoTask: renombrar una etiqueta no modifica ninguna
// tarea. Las asignaciones apuntan al número interno de la etiqueta, así que la
// tarea pasa a mostrar el nombre nuevo sin que su fila guardada se mueva: su
// updated_at no cambia.
func TestRenamingATagChangesNoTask(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	createTag(t, project, "--name", "docs")
	task := createTask(t, project)
	setTags(t, project, task.Ref, "docs")

	before := readTask(t, project, task.Ref)

	code, out, errOut := run(t, project, tagCmd, "edit", "docs", "--name", "documentacion")
	if code != 0 {
		t.Fatalf("tag edit salió con %d: %s", code, errOut)
	}

	var edited tagBody
	decode(t, out, &edited)

	if edited.Name != "documentacion" {
		t.Errorf("la etiqueta quedó con el nombre %q, se esperaba documentacion", edited.Name)
	}

	after := readTask(t, project, task.Ref)

	if !before.UpdatedAt.Equal(after.UpdatedAt) {
		t.Errorf("renombrar la etiqueta movió updated_at de la tarea: antes %s, después %s",
			before.UpdatedAt, after.UpdatedAt)
	}

	if !slices.Equal(after.Tags, []string{"documentacion"}) {
		t.Errorf("la tarea muestra las etiquetas %v, se esperaba [documentacion]", after.Tags)
	}
}

// TestTagDeleteWarnsOfTheTasksThatUsedIt: borrar una etiqueta la quita del
// catálogo y retira sus asignaciones, avisando en la salida de errores de las
// referencias de las tareas que la usaban; la respuesta es la etiqueta borrada.
func TestTagDeleteWarnsOfTheTasksThatUsedIt(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	description := "El de siempre"
	createTag(t, project, "--name", "docs", "--color", greenColor, "--description", description)
	createTag(t, project, "--name", "otra")
	first := createTask(t, project)
	second := createTask(t, project)
	other := createTask(t, project)
	setTags(t, project, first.Ref, "docs")
	setTags(t, project, second.Ref, "docs")
	setTags(t, project, other.Ref, "otra")

	code, out, errOut := run(t, project, tagCmd, "delete", "docs")

	if code != 0 {
		t.Fatalf("tag delete salió con %d: %s", code, errOut)
	}

	var deleted tagBody
	decode(t, out, &deleted)

	wantTag(t, deleted, tagBody{Name: "docs", Color: greenColor, Description: &description})
	wantNamedIn(t, errOut, []string{first.Ref, second.Ref}, []string{other.Ref})

	for _, ref := range []string{first.Ref, second.Ref} {
		wantTaskTags(t, project, ref)
	}

	wantTaskTags(t, project, other.Ref, "otra")

	catalogue := listTags(t, project)

	if len(catalogue) != 1 || catalogue[0].Name != "otra" {
		t.Errorf("el catálogo es %+v, se esperaba solo la etiqueta otra", catalogue)
	}
}

// TestTagDeleteWithoutUsersWarnsNothing: una etiqueta que nadie usa se borra
// sin aviso.
func TestTagDeleteWithoutUsersWarnsNothing(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	createTag(t, project, "--name", "sola")

	code, out, errOut := run(t, project, tagCmd, "delete", "sola")

	if code != 0 {
		t.Fatalf("tag delete salió con %d: %s", code, errOut)
	}

	if errOut != "" {
		t.Errorf("borrar una etiqueta sin uso escribió en la salida de errores: %s", errOut)
	}

	var deleted tagBody
	decode(t, out, &deleted)

	if deleted.Name != "sola" {
		t.Errorf("la respuesta del borrado es %+v, se esperaba la etiqueta sola", deleted)
	}

	if catalogue := listTags(t, project); len(catalogue) != 0 {
		t.Errorf("el catálogo es %+v, se esperaba vacío", catalogue)
	}
}

// TestUpdateSetsAndClearsTheTagsOfATask: --tags pone la lista entera de
// etiquetas de la tarea —una vacía la deja sin ninguna—, un nombre que no está
// en el catálogo es entrada inválida sin tocar la tarea, y el detalle de show
// muestra las etiquetas de cada tarea.
func TestUpdateSetsAndClearsTheTagsOfATask(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	createTag(t, project, "--name", "uno")
	createTag(t, project, "--name", "dos")
	task := createTask(t, project)

	changed := setTags(t, project, task.Ref, "uno, dos")

	if !slices.Equal(changed.Tags, []string{"dos", "uno"}) {
		t.Errorf("la tarea quedó con las etiquetas %v, se esperaba [dos uno]", changed.Tags)
	}

	detail := showDetail(t, project, task.Ref)

	if !slices.Equal(detail.Tags, []string{"dos", "uno"}) {
		t.Errorf("el detalle muestra las etiquetas %v, se esperaba [dos uno]", detail.Tags)
	}

	before := showRaw(t, project, task.Ref)

	code, out, errOut := run(t, project, "update", task.Ref, "--tags", "uno,desconocida")

	if code != 2 {
		t.Fatalf("update con una etiqueta desconocida salió con %d, se esperaba 2", code)
	}

	if out != "" {
		t.Errorf("update con una etiqueta desconocida escribió en la salida estándar: %s", out)
	}

	if message := errorJSON(t, errOut); !strings.Contains(message, unknownColor) {
		t.Errorf("el mensaje no menciona la etiqueta desconocida: %s", message)
	}

	if after := showRaw(t, project, task.Ref); after != before {
		t.Errorf("una asignación rechazada tocó la tarea: antes %s, después %s", before, after)
	}

	cleared := setTags(t, project, task.Ref, "")

	if len(cleared.Tags) != 0 {
		t.Errorf("vaciar las etiquetas dejó la tarea con %v", cleared.Tags)
	}
}

// TestTheTagsOfOneTaskAreUntouchedByAnotherTasksChange: cambiar las etiquetas
// de una tarea no toca las de las demás.
func TestTheTagsOfOneTaskAreUntouchedByAnotherTasksChange(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	createTag(t, project, "--name", "comun")
	createTag(t, project, "--name", "otra")
	first := createTask(t, project)
	second := createTask(t, project)
	setTags(t, project, first.Ref, "comun")
	setTags(t, project, second.Ref, "comun")

	setTags(t, project, second.Ref, "otra")

	if tags := readTask(t, project, first.Ref).Tags; !slices.Equal(tags, []string{"comun"}) {
		t.Errorf("las etiquetas de la primera tarea son %v, se esperaba [comun]", tags)
	}

	if tags := readTask(t, project, second.Ref).Tags; !slices.Equal(tags, []string{"otra"}) {
		t.Errorf("las etiquetas de la segunda tarea son %v, se esperaba [otra]", tags)
	}
}
