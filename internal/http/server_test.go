package http_test

import (
	"bufio"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"tasks-ai/internal/core"
	api "tasks-ai/internal/http"
	"tasks-ai/internal/store"
	"testing"
	"time"
)

// newTestServer levanta la API sobre el núcleo dado y devuelve su dirección.
func newTestServer(t *testing.T, apply *core.Core, log *slog.Logger) string {
	t.Helper()

	server := httptest.NewServer(api.New(apply, log, nil).Handler())
	t.Cleanup(server.Close)

	return server.URL
}

// newServer monta la API sobre una base recién abierta y devuelve también la
// base, para que un test la cierre por debajo y vea el fallo interno.
func newServer(t *testing.T) (string, *store.Store) {
	t.Helper()

	base, err := store.OpenAs(t.Context(), filepath.Join(t.TempDir(), "tasks.db"), store.OriginServer)
	if err != nil {
		t.Fatalf("abriendo la base de prueba: %v", err)
	}

	t.Cleanup(func() { _ = base.Close() })

	apply := core.New(base, "tk")
	log := slog.New(slog.DiscardHandler)

	server := newTestServer(t, apply, log)

	return server, base
}

func TestServer_createsAndReadsTasksThroughTheCore(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	created := createsATaskThroughTheAPI(t, server)
	readsItsDetailWithItsEmptyLists(t, server, created)
	listsItAsTheOnlyTaskOfTheProject(t, server, created)
}

// createsATaskThroughTheAPI crea una tarea por la API y comprueba que llega
// como se pidió: su referencia, su estado y su prioridad por omisión.
func createsATaskThroughTheAPI(t *testing.T, server string) core.Task {
	t.Helper()

	status, body := do(t, server, http.MethodPost, "/api/tasks", `{"title": "Preparar el análisis", "description": "con detalle"}`)
	if status != http.StatusCreated {
		t.Fatalf("crear por la API respondió %d: %s", status, body)
	}

	var created core.Task

	decode(t, body, &created)

	if created.Ref != "tk-1" || created.Status != "open" || created.Priority != core.DefaultPriority {
		t.Errorf("la tarea creada es %+v, se esperaba tk-1 abierta con la prioridad por omisión", created)
	}

	return created
}

// readsItsDetailWithItsEmptyLists lee el detalle de la tarea y comprueba que
// trae la guardada y sus listas alrededor, aunque aún estén vacías.
func readsItsDetailWithItsEmptyLists(t *testing.T, server string, created core.Task) {
	t.Helper()

	status, body := do(t, server, http.MethodGet, "/api/tasks/"+created.Ref, "")
	if status != http.StatusOK {
		t.Fatalf("leer el detalle respondió %d: %s", status, body)
	}

	var detail core.Detail

	decode(t, body, &detail)

	if detail.Ref != created.Ref || detail.Children == nil || detail.Blockers == nil || detail.Dependents == nil || detail.Comments == nil {
		t.Errorf("el detalle trae %+v, se esperaban las listas vacías y la tarea guardada", detail)
	}
}

// listsItAsTheOnlyTaskOfTheProject pide la lista del proyecto y comprueba que
// la tarea creada es la única que trae.
func listsItAsTheOnlyTaskOfTheProject(t *testing.T, server string, created core.Task) {
	t.Helper()

	status, body := do(t, server, http.MethodGet, "/api/tasks", "")
	if status != http.StatusOK {
		t.Fatalf("listar respondió %d: %s", status, body)
	}

	var tasks []core.Task

	decode(t, body, &tasks)

	if len(tasks) != 1 || tasks[0].Ref != created.Ref {
		t.Errorf("la lista trae %+v, se esperaba solo %q", tasks, created.Ref)
	}
}

func TestServer_changesATaskTheWayTheCoreChangesIt(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	patchesWhatTheChangeAsks(t, server)
	blockAndUnblockLeaveTheTaskUnblocked(t, server)
	aCommentAddedArrivesInTheDetail(t, server)
}

// patchesWhatTheChangeAsks cambia la prioridad y las etiquetas de una tarea por
// el cambio de la API y comprueba que la respuesta las lleva.
func patchesWhatTheChangeAsks(t *testing.T, server string) {
	t.Helper()

	do(t, server, http.MethodPost, "/api/tasks", `{"title": "Primera"}`)
	do(t, server, http.MethodPost, "/api/tasks", `{"title": "Segunda"}`)
	do(t, server, http.MethodPost, "/api/tags", `{"name": "docs"}`)

	status, body := do(t, server, http.MethodPatch, "/api/tasks/tk-1", `{"priority": 1, "tags": ["docs"]}`)
	if status != http.StatusOK {
		t.Fatalf("cambiar la tarea respondió %d: %s", status, body)
	}

	var changed core.Task

	decode(t, body, &changed)

	if changed.Priority != 1 || len(changed.Tags) != 1 || changed.Tags[0] != "docs" {
		t.Errorf("la tarea quedó %+v, se esperaba prioridad 1 y la etiqueta docs", changed)
	}
}

// blockAndUnblockLeaveTheTaskUnblocked añade un bloqueo y lo retira por sus
// rutas y comprueba que la tarea queda desbloqueada al final.
func blockAndUnblockLeaveTheTaskUnblocked(t *testing.T, server string) {
	t.Helper()

	do(t, server, http.MethodPost, "/api/tasks", `{"title": "Primera"}`)
	do(t, server, http.MethodPost, "/api/tasks", `{"title": "Segunda"}`)

	// El bloqueo y su retirada van por sus rutas y responden con la tarea.
	status, body := do(t, server, http.MethodPost, "/api/tasks/tk-1/blockers", `{"blocker": "tk-2"}`)
	if status != http.StatusOK {
		t.Fatalf("añadir el bloqueo respondió %d: %s", status, body)
	}

	var blocked core.Task

	decode(t, body, &blocked)

	if !blocked.Blocked {
		t.Errorf("la tarea quedó sin estar bloqueada: %+v", blocked)
	}

	status, _ = do(t, server, http.MethodDelete, "/api/tasks/tk-1/blockers/tk-2", "")
	if status != http.StatusOK {
		t.Fatalf("retirar el bloqueo respondió %d", status)
	}

	status, body = do(t, server, http.MethodGet, "/api/tasks/tk-1", "")
	if status != http.StatusOK {
		t.Fatalf("leer el detalle respondió %d", status)
	}

	var detail core.Detail

	decode(t, body, &detail)

	if detail.Blocked {
		t.Errorf("la tarea sigue bloqueada tras retirar el bloqueo: %+v", detail)
	}
}

// aCommentAddedArrivesInTheDetail añade un comentario a una tarea y comprueba
// que el detalle lo trae.
func aCommentAddedArrivesInTheDetail(t *testing.T, server string) {
	t.Helper()

	do(t, server, http.MethodPost, "/api/tasks", `{"title": "Primera"}`)

	// El comentario se añade y el detalle lo trae.
	status, body := do(t, server, http.MethodPost, "/api/tasks/tk-1/comments", `{"body": "un comentario"}`)
	if status != http.StatusOK {
		t.Fatalf("añadir el comentario respondió %d: %s", status, body)
	}

	status, body = do(t, server, http.MethodGet, "/api/tasks/tk-1", "")
	if status != http.StatusOK {
		t.Fatalf("leer el detalle respondió %d: %s", status, body)
	}

	var detail core.Detail

	decode(t, body, &detail)

	if len(detail.Comments) != 1 || detail.Comments[0].Body != "un comentario" {
		t.Errorf("el detalle trae los comentarios %+v, se esperaba el añadido", detail.Comments)
	}
}

func TestServer_servesTheCatalogueOfTags(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	status, body := do(t, server, "POST", "/api/tags", `{"name": "docs"}`)
	if status != http.StatusCreated {
		t.Fatalf("crear la etiqueta respondió %d: %s", status, body)
	}

	var created core.Tag

	decode(t, body, &created)

	if created.Name != "docs" || created.Color != core.DefaultTagColor {
		t.Errorf("la etiqueta creada es %+v, se esperaba docs con el color por omisión", created)
	}

	status, body = do(t, server, "GET", "/api/tags", "")
	if status != http.StatusOK {
		t.Fatalf("leer el catálogo respondió %d: %s", status, body)
	}

	var catalogue []core.TagWithUses

	decode(t, body, &catalogue)

	if len(catalogue) != 1 || catalogue[0].Uses != 0 {
		t.Errorf("el catálogo trae %+v, se esperaba docs sin uso", catalogue)
	}

	status, body = do(t, server, "PATCH", "/api/tags/docs", `{"color": "red"}`)
	if status != http.StatusOK {
		t.Fatalf("cambiar la etiqueta respondió %d: %s", status, body)
	}

	status, body = do(t, server, "DELETE", "/api/tags/docs", "")
	if status != http.StatusOK {
		t.Fatalf("borrar la etiqueta respondió %d: %s", status, body)
	}

	var deleted core.Tag

	decode(t, body, &deleted)

	if deleted.Name != "docs" {
		t.Errorf("la respuesta del borrado trae %+v, se esperaba la etiqueta borrada", deleted)
	}
}

func TestServer_closesAndReopensThroughTheStatusOfThePatch(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	do(t, server, "POST", "/api/tasks", `{"title": "Con ciclo"}`)

	status, body := do(t, server, "PATCH", "/api/tasks/tk-1", `{"status": "done"}`)
	if status != http.StatusBadRequest {
		t.Errorf("poner done por el cambio respondió %d: %s, se esperaba 400", status, body)
	}

	status, body = do(t, server, "POST", "/api/tasks/tk-1/close", `{"reason": "done"}`)
	if status != http.StatusOK {
		t.Fatalf("cerrar respondió %d: %s", status, body)
	}

	var closed core.Task

	decode(t, body, &closed)

	if closed.Status != "done" || closed.CloseReason == nil || *closed.CloseReason != "done" {
		t.Errorf("la tarea cerrada es %+v, se esperaba done con su motivo", closed)
	}

	status, body = do(t, server, "PATCH", "/api/tasks/tk-1", `{"status": "open"}`)
	if status != http.StatusOK {
		t.Fatalf("reabrir por el cambio de estado respondió %d: %s", status, body)
	}

	var reopened core.Task

	decode(t, body, &reopened)

	if reopened.Status != "open" || reopened.CloseReason != nil {
		t.Errorf("la tarea reabierta es %+v, se esperaba open sin motivo", reopened)
	}
}

// errorTranslation nombra una negativa de la API con los pasos que la provocan.
type errorTranslation struct {
	name  string
	steps func(t *testing.T, server string)
}

//nolint:funlen // la enumeración completa de las negativas debe quedarse lineal
func errorTranslations() []errorTranslation {
	return []errorTranslation{
		{
			name: "la entrada no es JSON",
			steps: func(t *testing.T, server string) {
				t.Helper()

				status, body := do(t, server, "POST", "/api/tasks", `{`)
				expect(t, status, body, http.StatusBadRequest, "el cuerpo de la petición no es un JSON válido")
			},
		},
		{
			name: "la prioridad está fuera del rango y la entrada llega al núcleo",
			steps: func(t *testing.T, server string) {
				t.Helper()

				status, body := do(t, server, "POST", "/api/tasks", `{"title": "Fuera de rango", "priority": 9}`)
				expect(t, status, body, http.StatusBadRequest, "la prioridad 9 está fuera del rango 0-3")
			},
		},
		{
			name: "falta el título",
			steps: func(t *testing.T, server string) {
				t.Helper()

				status, body := do(t, server, "POST", "/api/tasks", `{"title": "   "}`)
				expect(t, status, body, http.StatusBadRequest, "la tarea necesita un título")
			},
		},
		{
			name: "el cambio no pide nada",
			steps: func(t *testing.T, server string) {
				t.Helper()

				status, body := do(t, server, "PATCH", "/api/tasks/tk-1", `{}`)
				expect(t, status, body, http.StatusBadRequest, "no hay nada que cambiar: indica al menos un campo")
			},
		},
		{
			name: "falta el motivo de cierre",
			steps: func(t *testing.T, server string) {
				t.Helper()

				do(t, server, "POST", "/api/tasks", `{"title": "Sin motivo"}`)
				status, body := do(t, server, "POST", "/api/tasks/tk-1/close", `{}`)
				expect(t, status, body, http.StatusBadRequest, "falta el motivo de cierre")
			},
		},
		{
			name: "falta la bloqueante",
			steps: func(t *testing.T, server string) {
				t.Helper()

				do(t, server, "POST", "/api/tasks", `{"title": "Sin bloqueante"}`)
				status, body := do(t, server, "POST", "/api/tasks/tk-1/blockers", `{}`)
				expect(t, status, body, http.StatusBadRequest, "falta la referencia de la tarea bloqueante")
			},
		},
		{
			name: "falta el cuerpo del comentario",
			steps: func(t *testing.T, server string) {
				t.Helper()

				do(t, server, "POST", "/api/tasks", `{"title": "Sin comentario"}`)
				status, body := do(t, server, "POST", "/api/tasks/tk-1/comments", `{"body": "  "}`)
				expect(t, status, body, http.StatusBadRequest, "falta el cuerpo del comentario")
			},
		},
		{
			name: "la referencia no corresponde a ninguna tarea",
			steps: func(t *testing.T, server string) {
				t.Helper()

				status, body := do(t, server, http.MethodGet, "/api/tasks/tk-404", "")
				expect(t, status, body, http.StatusNotFound, "no hay ninguna tarea con la referencia \"tk-404\"")
			},
		},
		{
			name: "la referencia está mal escrita",
			steps: func(t *testing.T, server string) {
				t.Helper()

				status, body := do(t, server, http.MethodGet, "/api/tasks/luego", "")
				expect(t, status, body, http.StatusNotFound, "la referencia \"luego\" no es válida: se espera tk-<número>")
			},
		},
		{
			name: "la etiqueta no está",
			steps: func(t *testing.T, server string) {
				t.Helper()

				status, body := do(t, server, "DELETE", "/api/tags/nope", "")
				expect(t, status, body, http.StatusNotFound, "no hay ninguna etiqueta con el nombre \"nope\"")
			},
		},
		{
			name: "cerrar con subtareas abiertas lo impide la regla",
			steps: func(t *testing.T, server string) {
				t.Helper()

				do(t, server, "POST", "/api/tasks", `{"title": "Padre"}`)
				do(t, server, "POST", "/api/tasks", `{"title": "Hija", "parent": "tk-1"}`)
				status, body := do(t, server, "POST", "/api/tasks/tk-1/close", `{"reason": "done"}`)
				expect(t, status, body, http.StatusConflict, "no se puede cerrar la tarea \"tk-1\": tiene subtareas abiertas")
			},
		},
		{
			name: "cerrar una tarea ya cerrada lo impide la regla",
			steps: func(t *testing.T, server string) {
				t.Helper()

				do(t, server, "POST", "/api/tasks", `{"title": "Dos cierres"}`)
				do(t, server, "POST", "/api/tasks/tk-1/close", `{"reason": "done"}`)
				status, body := do(t, server, "POST", "/api/tasks/tk-1/close", `{"reason": "done"}`)
				expect(t, status, body, http.StatusConflict, "no se puede cerrar la tarea \"tk-1\": ya está cerrada")
			},
		},
		{
			name: "el nombre de la etiqueta ya está en el catálogo",
			steps: func(t *testing.T, server string) {
				t.Helper()

				do(t, server, "POST", "/api/tags", `{"name": "docs"}`)
				status, body := do(t, server, "POST", "/api/tags", `{"name": "docs"}`)
				expect(t, status, body, http.StatusConflict, "ya existe una etiqueta con el nombre \"docs\"")
			},
		},
	}
}

func TestServer_translatesTheErrorsOfTheCoreIntoTheirStatusCodes(t *testing.T) {
	t.Parallel()

	for _, test := range errorTranslations() {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			server, _ := newServer(t)
			test.steps(t, server)
		})
	}
}

func TestServer_answersInternalErrorWhenTheBaseFails(t *testing.T) {
	t.Parallel()

	server, base := newServer(t)

	if err := base.Close(); err != nil {
		t.Fatalf("cerrando la base por debajo de la API: %v", err)
	}

	status, body := do(t, server, http.MethodGet, "/api/tasks", "")
	if status != http.StatusInternalServerError {
		t.Errorf("una base cerrada respondió %d: %s, se esperaba 500", status, body)
	}

	if text := errorOf(t, body); text == "" {
		t.Errorf("el fallo interno debe cruzar con su mensaje: %q", body)
	}
}

func TestServer_reportsTheStatusOfTheProduct(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	status, body := do(t, server, http.MethodGet, "/api/status", "")
	if status != http.StatusOK {
		t.Fatalf("el estado respondió %d: %s", status, body)
	}

	var reported core.Status

	decode(t, body, &reported)

	if reported.Version != core.Version || reported.Schema != 4 || reported.Prefix != "tk" {
		t.Errorf("el estado del producto es %+v, se esperaba la versión del producto, el esquema 4 y el prefijo tk", reported)
	}
}

func TestServer_theEventsChannelKeepsOpenAndAnnouncesTheChange(t *testing.T) {
	t.Parallel()

	server, base := newServer(t)
	apply := core.New(base, "tk")

	events := openEvents(t, server+"/api/events")

	// Otro que no es quien lee cambia los datos: el aviso llega al canal.
	if _, err := apply.CreateTask(t.Context(), core.NewTask{Title: "Avisada"}); err != nil {
		t.Fatalf("creando la tarea por el núcleo: %v", err)
	}

	event := readEvent(t, events)

	if event.Resource != "task" || event.Ref != "tk-1" || event.Origin != store.OriginServer {
		t.Errorf("el aviso es %+v, se esperaba la tarea tk-1 pedida por el servidor", event)
	}
}

// TestServer_theEventsChannelContinuesWhereTheReaderLeftOff recorre lo que el
// contrato promete a quien vuelve: cada aviso lleva delante su identidad, quien
// continúa desde una no recibe las anteriores, y una identidad que no es un
// número se dice en vez de empezar por el principio en silencio.
func TestServer_theEventsChannelContinuesWhereTheReaderLeftOff(t *testing.T) {
	t.Parallel()

	server, base := newServer(t)
	apply := core.New(base, "tk")

	for _, title := range []string{"Primera", "Segunda"} {
		if _, err := apply.CreateTask(t.Context(), core.NewTask{Title: title}); err != nil {
			t.Fatalf("creando la tarea %q: %v", title, err)
		}
	}

	first := readRawEvent(t, openEvents(t, server+"/api/events"))

	if !strings.Contains(first, "id: 1") {
		t.Errorf("el primer aviso no lleva su identidad delante: %q", first)
	}

	second := readRawEvent(t, openEvents(t, server+"/api/events?after=1"))

	if !strings.Contains(second, "id: 2") || !strings.Contains(second, "tk-2") {
		t.Errorf("quien continúa desde la primera recibió %q, se esperaba la segunda", second)
	}

	if strings.Contains(second, "tk-1") {
		t.Errorf("quien continúa desde la primera volvió a recibir la primera: %q", second)
	}

	// Continuar desde una identidad que no es un número es entrada inválida, y se
	// pide con su propio tope: si el canal no la rechazara, leer su cuerpo se
	// quedaría esperando en vez de fallar.
	client := &http.Client{Timeout: 5 * time.Second}

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server+"/api/events?after=abc", nil)
	if err != nil {
		t.Fatalf("preparando la petición del canal: %v", err)
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("pidiendo el canal desde una identidad inválida: %v", err)
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusBadRequest {
		t.Errorf("continuar desde una identidad que no es un número respondió %d", response.StatusCode)
	}
}

// readRawEvent devuelve un aviso tal como viaja por el canal, con sus líneas,
// para poder comprobar su forma y no solo lo que lleva dentro.
func readRawEvent(t *testing.T, events io.Reader) string {
	t.Helper()

	scanner := bufio.NewScanner(events)

	var block strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		block.WriteString(line + "\n")

		if line == "" {
			return block.String()
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("leyendo el canal de eventos: %v", err)
	}

	t.Fatal("el canal se cerró sin avisar de ningún cambio")

	return ""
}

func expect(t *testing.T, status int, body string, expected int, message string) {
	t.Helper()

	if status != expected {
		t.Errorf("la petición respondió %d, se esperaba %d: %s", status, expected, body)
	}

	if text := errorOf(t, body); text != message {
		t.Errorf("el cuerpo del error dice %q, se esperaba %q", text, message)
	}
}

func do(t *testing.T, server, method, path, body string) (int, string) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), method, server+path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("preparando %s %s: %v", method, path, err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("pidiendo %s %s: %v", method, path, err)
	}

	defer func() { _ = response.Body.Close() }()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("leyendo la respuesta de %s %s: %v", method, path, err)
	}

	return response.StatusCode, string(raw)
}

func openEvents(t *testing.T, url string) io.Reader {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
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

func errorOf(t *testing.T, body string) string {
	t.Helper()

	var parsed struct {
		Error string `json:"error"`
	}

	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return ""
	}

	return parsed.Error
}

func decode(t *testing.T, raw string, target any) {
	t.Helper()

	if err := json.Unmarshal([]byte(raw), target); err != nil {
		t.Fatalf("el JSON devuelto no se entiende (%v): %q", err, raw)
	}
}
