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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"tasks-ai/internal/core"
	"tasks-ai/internal/install"
	"testing"
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

	for _, name := range []string{"init", "create", "show"} {
		if !strings.Contains(out, name) {
			t.Errorf("--help no menciona %q:\n%s", name, out)
		}
	}
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

	code, out, errOut := run(t, project, "create", "--title", "Preparar el análisis")
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

	code, out, errOut := run(t, project, "show", "tk-999")

	if code != 1 {
		t.Fatalf("show con referencia inexistente salió con %d, se esperaba 1", code)
	}

	if out != "" {
		t.Errorf("show escribió en la salida estándar: %s", out)
	}

	if message := errorJSON(t, errOut); !strings.Contains(message, "tk-999") {
		t.Errorf("el mensaje de error no menciona la referencia: %s", message)
	}
}

// TestCreateWithoutTitleIsMisuse: crear sin título es un mal uso de la línea de
// comandos y sale con su propio código.
func TestCreateWithoutTitleIsMisuse(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	installProject(t, project)

	code, out, errOut := run(t, project, "create")

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

	code, out, errOut := run(t, project, "create",
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

	code, out, errOut := run(t, project, "create", "--title", "Preparar el análisis")
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

	if message := errorJSON(t, errOut); !strings.Contains(message, "cambiar") {
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
		code, out, errOut := run(t, project, "create", "--title", "Preparar el análisis", "--priority="+priority)

		if code != 2 {
			t.Errorf("create con la prioridad %s salió con %d, se esperaba 2", priority, code)
		}

		if out != "" {
			t.Errorf("create con la prioridad %s escribió en la salida estándar: %s", priority, out)
		}

		if message := errorJSON(t, errOut); !strings.Contains(message, "rango") {
			t.Errorf("el mensaje de la prioridad %s no explica el rango: %s", priority, message)
		}
	}

	// Ninguna de las dos rechazadas dejó tarea: la siguiente lleva la primera identidad.
	code, out, errOut := run(t, project, "create", "--title", "Preparar el análisis")
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

	code, out, errOut := run(t, project, "create", "--title", "Preparar el análisis")
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

		if message := errorJSON(t, errOut); !strings.Contains(message, "rango") {
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

	if message := errorJSON(t, errOut); !strings.Contains(message, "instalado") {
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
		runs[i].cmd = exec.CommandContext(t.Context(), binPath, "create", "--title", fmt.Sprintf("Tarea %d", i))
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
