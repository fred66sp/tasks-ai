package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"tasks-ai/internal/cli"
	"tasks-ai/internal/core"
	"tasks-ai/internal/install"
	"testing"
)

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
