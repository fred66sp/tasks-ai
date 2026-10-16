package cli_test

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"tasks-ai/internal/core"
	"tasks-ai/internal/install"
	"tasks-ai/internal/settings"
	"testing"
	"time"
)

// TestServe_theSameOperationAskedByTheAPIAndByTheCommandLineLeavesTheDataTheSame
// recorre la primera regla del criterio: el servidor se levanta como lo hace una
// persona —el binario, con su serve y un puerto libre del settings— y la misma
// operación pedida por la API y por el CLI deja los datos igual.
//
//nolint:paralleltest // el test cambia de carpeta con t.Chdir: en paralelo pisaría la carpeta de otra prueba
func TestServe_theSameOperationAskedByTheAPIAndByTheCommandLineLeavesTheDataTheSame(t *testing.T) {
	dir := prepareProject(t)
	address, _ := startServer(t, dir, buildBinary(t))

	// Crear por la API y leer por el CLI: los datos son los mismos.
	status, body := do(t, "POST", "http://"+address+"/api/tasks", `{"title": "Vía API"}`)
	if status != http.StatusCreated {
		t.Fatalf("crear por la API respondió %d: %s", status, body)
	}

	var created core.Task

	decode(t, body, &created)

	code, out, errOut := run(t, "show", created.Ref)
	if code != 0 {
		t.Fatalf("show salió con %d: %s", code, errOut)
	}

	var read core.Task

	decode(t, out, &read)

	if read.ID != created.ID || read.Title != created.Title || read.Status != created.Status || read.Priority != created.Priority {
		t.Errorf("la tarea que ve el CLI %+v no es la que creó la API %+v", read, created)
	}

	// Crear por el CLI y leer por la API: los datos son los mismos.
	code, out, errOut = run(t, "create", "--title", "Vía CLI")
	if code != 0 {
		t.Fatalf("create salió con %d: %s", code, errOut)
	}

	var fromCLI core.Task

	decode(t, out, &fromCLI)

	status, body = do(t, "GET", "http://"+address+"/api/tasks/"+fromCLI.Ref, "")
	if status != http.StatusOK {
		t.Fatalf("leer por la API respondió %d: %s", status, body)
	}

	var fromAPI core.Task

	decode(t, body, &fromAPI)

	if fromAPI.ID != fromCLI.ID || fromAPI.Title != fromCLI.Title || fromAPI.Status != fromCLI.Status || fromAPI.Priority != fromCLI.Priority {
		t.Errorf("la tarea que ve la API %+v no es la que creó el CLI %+v", fromAPI, fromCLI)
	}
}

// TestServe_aSessionReadingTheEventsSeesTheChangeAnotherMakes recorre la segunda
// regla del criterio: una sesión deja el canal abierto y ve el aviso del cambio
// que otra sesión pide, sea el CLI o la propia API.
//
//nolint:paralleltest // el test cambia de carpeta con t.Chdir: en paralelo pisaría la carpeta de otra prueba
func TestServe_aSessionReadingTheEventsSeesTheChangeAnotherMakes(t *testing.T) {
	dir := prepareProject(t)
	address, _ := startServer(t, dir, buildBinary(t))

	events := openEvents(t, "http://"+address+"/api/events")

	// Otra sesión, la del CLI, cambia los datos.
	code, _, errOut := run(t, "create", "--title", "Cambio ajeno")
	if code != 0 {
		t.Fatalf("create salió con %d: %s", code, errOut)
	}

	event := readEvent(t, events)

	if event.Resource != "task" || event.Ref != "tk-1" || event.Origin != "cli" {
		t.Errorf("el aviso del cambio del CLI es %+v, se esperaba la tarea tk-1 pedida por el cli", event)
	}

	// Y también el cambio que otra sesión de la API pide.
	status, body := do(t, "POST", "http://"+address+"/api/tasks", `{"title": "Cambio del servidor"}`)
	if status != http.StatusCreated {
		t.Fatalf("crear por la API respondió %d: %s", status, body)
	}

	event = readEvent(t, events)

	if event.Resource != "task" || event.Ref != "tk-2" || event.Origin != "server" {
		t.Errorf("el aviso del cambio del servidor es %+v, se esperaba la tarea tk-2 pedida por el servidor", event)
	}
}

// TestServe_aSessionReadingTheEventsDoesNotHoldTheShutdown recorre el apagado
// con una sesión leyendo el canal: el servidor tiene que irse en el acto y sin
// quejarse, porque quien lee no ha hecho nada malo.
//
//nolint:paralleltest // el test cambia de carpeta con t.Chdir: en paralelo pisaría la carpeta de otra prueba
func TestServe_aSessionReadingTheEventsDoesNotHoldTheShutdown(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("en Windows no se puede mandar la interrupción al proceso")
	}

	dir := prepareProject(t)
	address, server := startServer(t, dir, buildBinary(t))

	openEvents(t, "http://"+address+"/api/events")

	// Se apaga como lo apaga una persona: interrumpiendo el proceso.
	if err := server.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("interrumpiendo el servidor: %v", err)
	}

	gone := make(chan error, 1)

	go func() { gone <- server.Wait() }()

	select {
	case err := <-gone:
		if err != nil {
			t.Errorf("el servidor se apagó con error: %v", err)
		}
	case <-time.After(4 * time.Second):
		t.Error("el servidor no se apagó en cuatro segundos con una sesión leyendo el canal")
	}
}

// prepareProject instala el producto en un proyecto recién creado y deja el
// proceso dentro de él, como hace quien usa el producto. Devuelve la raíz del
// proyecto, que es donde vive el settings que el servidor leerá.
func prepareProject(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	t.Chdir(dir)

	code, out, errOut := run(t, "init")
	if code != 0 {
		t.Fatalf("init salió con %d: %s", code, errOut)
	}

	var installed install.Result

	decode(t, out, &installed)

	return dir
}

// buildBinary compila el binario del producto, que es como arranca el servidor
// en los tests: con el comando serve del propio binario. La raíz del módulo se
// resuelve desde el sitio de este fichero, porque el test cambia de carpeta.
func buildBinary(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no se pudo saber dónde está el test para resolver la raíz del módulo")
	}

	root, err := filepath.Abs(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	if err != nil {
		t.Fatalf("resolviendo la raíz del módulo: %v", err)
	}

	binary := filepath.Join(t.TempDir(), "tasks-ai")

	//nolint:gosec // el binario es el propio producto, recién construido por el test
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, "./cmd/tasks-ai")
	build.Dir = root

	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("compilando el binario: %v\n%s", err, output)
	}

	return binary
}

// startServer arranca el serve del binario en el proyecto dir, con un puerto
// libre escrito en el settings, y espera a que acepte peticiones. Devuelve
// también el proceso, para quien quiera apagarlo como lo apaga una persona.
func startServer(t *testing.T, dir, binary string) (string, *exec.Cmd) {
	t.Helper()

	port := freePort(t)

	cfg := fmt.Sprintf(`{"http": {"port": %d}}`, port)

	path := filepath.Join(dir, install.DirName, settings.FileName)

	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatalf("escribiendo el settings del servidor: %v", err)
	}

	server := exec.CommandContext(t.Context(), binary, "serve")
	server.Dir = dir

	var stderr strings.Builder

	server.Stderr = &stderr

	if err := server.Start(); err != nil {
		t.Fatalf("arrancando el servidor: %v", err)
	}

	// Solo se mata: quien quiera esperar a que salga lo hace una vez, porque
	// esperar dos veces al mismo proceso deja colgada a la segunda.
	t.Cleanup(func() { _ = server.Process.Kill() })

	address := fmt.Sprintf("127.0.0.1:%d", port)
	waitReady(t, address, &stderr)

	return address, server
}

// freePort encuentra un puerto libre en la máquina, para que el test no pelee
// por el puerto por omisión.
func freePort(t *testing.T) int {
	t.Helper()

	var config net.ListenConfig

	listener, err := config.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("buscando un puerto libre: %v", err)
	}

	defer func() { _ = listener.Close() }()

	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("la dirección del puerto libre no se entiende: %v", listener.Addr())
	}

	return address.Port
}

// waitReady repite la lectura del estado del producto hasta que el servidor
// responde, con el stderr a la vista si no llega a arrancar. Pregunta por su
// cuenta y no con el ayudante de las peticiones: mientras el servidor no
// escucha, la conexión se rechaza, y eso no es un fallo de la prueba sino algo
// que hay que esperar antes de que llegue el momento de exigir.
func waitReady(t *testing.T, address string, stderr *strings.Builder) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	url := "http://" + address + "/api/status"

	for time.Now().Before(deadline) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
		if err == nil {
			response, err := http.DefaultClient.Do(request)
			if err == nil {
				_ = response.Body.Close()

				if response.StatusCode == http.StatusOK {
					return
				}
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("el servidor no llegó a responder en %s:\n%s", address, stderr.String())
}

func do(t *testing.T, method, address, body string) (int, string) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), method, address, strings.NewReader(body))
	if err != nil {
		t.Fatalf("preparando %s %s: %v", method, address, err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("pidiendo %s %s: %v", method, address, err)
	}

	defer func() { _ = response.Body.Close() }()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("leyendo la respuesta de %s %s: %v", method, address, err)
	}

	return response.StatusCode, string(raw)
}

func openEvents(t *testing.T, address string) io.Reader {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, address, nil)
	if err != nil {
		t.Fatalf("abriendo el canal de eventos: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("abriendo el canal de eventos: %v", err)
	}

	t.Cleanup(func() { _ = response.Body.Close() })

	if response.StatusCode != http.StatusOK {
		t.Fatalf("el canal de eventos respondió %d", response.StatusCode)
	}

	if !strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
		t.Errorf("el canal anuncia %q, se esperaba text/event-stream", response.Header.Get("Content-Type"))
	}

	return response.Body
}

func readEvent(t *testing.T, events io.Reader) core.Event {
	t.Helper()

	scanner := bufio.NewScanner(events)

	for scanner.Scan() {
		if data, ok := strings.CutPrefix(scanner.Text(), "data: "); ok {
			var event core.Event

			decode(t, data, &event)

			return event
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("leyendo el canal de eventos: %v", err)
	}

	t.Fatal("el canal se cerró sin avisar de ningún cambio")

	return core.Event{}
}
