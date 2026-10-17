// Package http sirve la API del producto: rutas finas que leen la entrada,
// llaman al mismo núcleo al que llama la línea de comandos y traducen la
// respuesta o el error a su código de estado. Ninguna regla del producto vive
// aquí: si una regla cambia, cambia en el núcleo y las dos superficies la ven.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"strconv"
	"strings"
	"tasks-ai/internal/core"
	"tasks-ai/internal/store"
	"time"
)

// errBadEntry marks an entry of a request that cannot be read or cannot be
// served as asked: no llega al núcleo porque no hay nada que llevarle.
var errBadEntry = errors.New("entrada ilegible")

// badEntryError is an entry the API cannot serve. Lleva su mensaje y responde
// al centinela, igual que los errores del núcleo responden a los suyos.
type badEntryError struct {
	err error
}

func (e *badEntryError) Error() string { return e.err.Error() }

func (e *badEntryError) Unwrap() error { return e.err }

func (e *badEntryError) Is(target error) bool { return target == errBadEntry }

// badEntry builds the error of an entry the API cannot serve.
func badEntry(format string, args ...any) error {
	return &badEntryError{err: fmt.Errorf(format, args...)}
}

// pollInterval is how often the events channel looks at the base. Los avisos
// viven en la base, no en la memoria del servidor: mirarla es lo que hace que
// un cambio pedido por otro proceso —el CLI— llegue también a este canal.
const pollInterval = 500 * time.Millisecond

// workspaceEventName es el nombre con el que viaja cada aviso: es el mensaje
// con nombre que la interfaz escucha, y sin él la pantalla no reacciona.
const workspaceEventName = "workspace.changed"

// notice is the body of a named notice, in the shape the interface reads: the
// changed resource, in the plural que ella compara and, cuando el cambio toca
// una tarea, la tarea tocada. El aviso no dice quién pidió el cambio: el
// estado y el aviso son los mismos venga de donde venga, y quien lo hizo se
// evita repetirse con su propio estado local.
type notice struct {
	Resource string       `json:"resource"`
	Changes  []taskChange `json:"changes,omitempty"`
}

// taskChange is one task a batch of notices touched, in the shape the toast of
// the interface prints: su referencia visible, lo que le pasó y su título.
type taskChange struct {
	ID     string `json:"id"`
	Action string `json:"action"`
	Title  string `json:"title"`
}

// Las acciones que un cambio de tarea puede declarar. Cerrar es un cambio de
// estado que la interfaz nombra por su cuenta; el resto es un cambio.
const (
	actionCreated = "creada"
	actionClosed  = "cerrada"
	actionUpdated = "actualizada"
)

// noticeOf translates a change of the base into the notice the channel sends.
// Las tareas descritas vienen del listado previo al lote: un aviso cuyo cambio
// no está en él —el releído al reconectar— viaja sin ellas, y el aviso de una
// etiqueta nunca las lleva, porque el aviso de la pantalla solo nombra tareas.
func noticeOf(event core.Event, described map[string]taskChange) notice {
	resource := "tags"
	if event.Resource == store.ResourceTask {
		resource = "tasks"
	}

	result := notice{Resource: resource}

	if event.Resource == store.ResourceTask {
		if change, ok := described[event.Ref]; ok {
			result.Changes = []taskChange{change}
		}
	}

	return result
}

// describeChanges compares the listing the channel last saw with the one it
// sees now and says, per task that changed, what happened to it: alta, cierre o
// cambio. La baja no existe en el producto —ninguna superficie borra tareas—,
// así que no hay acción que declare. Sin listado previo no hay nada que
// comparar: el primer lote de una conexión que reconstruye el pasado viaja sin
// descripción, igual que el servidor de referencia.
func describeChanges(previous, current []core.Task) map[string]taskChange {
	if previous == nil {
		return nil
	}

	before := make(map[int64]core.Task, len(previous))
	for _, task := range previous {
		before[task.ID] = task
	}

	described := make(map[string]taskChange)

	for _, task := range current {
		was, seen := before[task.ID]
		if !seen {
			described[task.Ref] = taskChange{ID: task.Ref, Action: actionCreated, Title: task.Title}

			continue
		}

		if fingerprintOf(was) == fingerprintOf(task) {
			continue
		}

		action := actionUpdated
		if was.Status != "done" && task.Status == "done" {
			action = actionClosed
		}

		described[task.Ref] = taskChange{ID: task.Ref, Action: action, Title: task.Title}
	}

	return described
}

// fingerprintOf freezes a task into one comparable line: lo que cambie en ella
// cambia la línea, y lo que no, no.
func fingerprintOf(task core.Task) string {
	frozen, err := json.Marshal(task)
	if err != nil {
		// Una tarea de las que el núcleo devuelve siempre se compone en JSON:
		// no hay camino a este error con campos que todos son datos.
		return ""
	}

	return string(frozen)
}

// Los estados que un cambio de estado puede pedir: cerrar lleva su motivo y va
// por su propia ruta, así que done no se pone por aquí.
const (
	statusOpen       = "open"
	statusInProgress = "in_progress"
)

// Server serves the product's API over one project's core.
type Server struct {
	core    *core.Core
	log     *slog.Logger
	closing <-chan struct{}
}

// New wires the API over the core the command line calls and the registry it
// writes to. closing se cierra cuando el servidor empieza a apagarse: quien
// espera en el canal de avisos termina entonces en vez de retener el apagado.
// Un canal sin valor no se cierra nunca, que es lo que quieren las pruebas.
func New(apply *core.Core, log *slog.Logger, closing <-chan struct{}) *Server {
	return &Server{core: apply, log: log, closing: closing}
}

// Handler returns the routes of the API, with each operation logged at debug
// level: con error el servidor calla, con info dice dónde escucha —lo dice el
// comando serve— y con debug cuenta además cada operación que sirve.
func (s *Server) Handler() nethttp.Handler {
	return s.logOperations(s.routes())
}

// routes composes the routes the contract lists. Go's mux answers 405 by
// itself when the method does not fit the route.
func (s *Server) routes() *nethttp.ServeMux {
	mux := nethttp.NewServeMux()
	mux.HandleFunc("GET /{$}", interfaceIndex)
	mux.Handle("GET /", interfaceFiles())
	mux.HandleFunc("GET /api/tasks", s.listTasks)
	mux.HandleFunc("POST /api/tasks", s.createTask)
	mux.HandleFunc("GET /api/tasks/{ref}", s.taskDetail)
	mux.HandleFunc("PATCH /api/tasks/{ref}", s.patchTask)
	mux.HandleFunc("POST /api/tasks/{ref}/close", s.closeTask)
	mux.HandleFunc("POST /api/tasks/{ref}/blockers", s.addBlocker)
	mux.HandleFunc("DELETE /api/tasks/{ref}/blockers/{blocker}", s.removeBlocker)
	mux.HandleFunc("POST /api/tasks/{ref}/comments", s.addComment)
	mux.HandleFunc("GET /api/tags", s.listTags)
	mux.HandleFunc("POST /api/tags", s.createTag)
	mux.HandleFunc("PATCH /api/tags/{name}", s.patchTag)
	mux.HandleFunc("DELETE /api/tags/{name}", s.deleteTag)
	mux.HandleFunc("GET /api/events", s.events)
	mux.HandleFunc("GET /api/status", s.status)

	return mux
}

// logOperations writes one debug line per served operation, with the method,
// the path and the status it left with.
func (s *Server) logOperations(next nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		watched := &statusWriter{ResponseWriter: w, status: nethttp.StatusOK}

		next.ServeHTTP(watched, r)

		s.log.Debug("operación servida", "method", r.Method, "path", r.URL.Path, "status", watched.status)
	})
}

// statusWriter remembers the status the operation left with, so the debug line
// can name it. Sin cuerpo escrito el estado es el 200 que empieza.
type statusWriter struct {
	nethttp.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// Flush passes the flush on: sin él, el canal de eventos no puede empujar cada
// aviso en el momento en que lo lee.
func (w *statusWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(nethttp.Flusher); ok {
		flusher.Flush()
	}
}

// listTasks answers with every task of the project.
func (s *Server) listTasks(w nethttp.ResponseWriter, r *nethttp.Request) {
	tasks, err := s.core.Tasks(r.Context())
	if err != nil {
		s.writeError(w, err)

		return
	}

	writeJSON(w, nethttp.StatusOK, tasks)
}

// taskDetail answers with the task and everything around it.
func (s *Server) taskDetail(w nethttp.ResponseWriter, r *nethttp.Request) {
	detail, err := s.core.TaskDetail(r.Context(), r.PathValue("ref"))
	if err != nil {
		s.writeError(w, err)

		return
	}

	writeJSON(w, nethttp.StatusOK, detail)
}

// taskEntry is the body a creation speaks: what is left out does not reach the
// core, and what the core refuses, it refuses with its own message.
type taskEntry struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Assignee    string `json:"assignee"`
	Notes       string `json:"notes"`
	Priority    *int   `json:"priority"`
	Parent      string `json:"parent"`
}

// createTask creates a task, or a subtask when the entry names its parent.
func (s *Server) createTask(w nethttp.ResponseWriter, r *nethttp.Request) {
	entry, err := decode[taskEntry](r)
	if err != nil {
		s.writeError(w, err)

		return
	}

	newTask := core.NewTask{
		Title:       entry.Title,
		Description: entry.Description,
		Assignee:    entry.Assignee,
		Notes:       entry.Notes,
		Parent:      entry.Parent,
	}

	if entry.Priority != nil {
		newTask.Priority = entry.Priority
	}

	task, err := s.core.CreateTask(r.Context(), newTask)
	if err != nil {
		s.writeError(w, err)

		return
	}

	writeJSON(w, nethttp.StatusCreated, task)
}

// taskPatch is the body a change of task speaks: a field left out is not
// touched, igual que en el cambio del CLI.
type taskPatch struct {
	Title       *string   `json:"title"`
	Description *string   `json:"description"`
	Notes       *string   `json:"notes"`
	Assignee    *string   `json:"assignee"`
	Priority    *int      `json:"priority"`
	Tags        *[]string `json:"tags"`
	Status      *string   `json:"status"`
}

// patchTask changes what the change asks for. El estado va solo en el cambio:
// pedir open o in_progress lo lleva el cambio de la tarea, y cerrar no va por
// aquí.
func (s *Server) patchTask(w nethttp.ResponseWriter, r *nethttp.Request) {
	entry, err := decode[taskPatch](r)
	if err != nil {
		s.writeError(w, err)

		return
	}

	ref := r.PathValue("ref")

	if entry.Status != nil {
		s.changeStatus(w, r, ref, entry)

		return
	}

	changes := core.Changes{
		Title:       entry.Title,
		Description: entry.Description,
		Notes:       entry.Notes,
		Assignee:    entry.Assignee,
		Priority:    entry.Priority,
		Tags:        entry.Tags,
	}

	task, err := s.core.UpdateTask(r.Context(), ref, changes)
	if err != nil {
		s.writeError(w, err)

		return
	}

	writeJSON(w, nethttp.StatusOK, task)
}

// changeStatus translates the state of the patch into the change of the core
// that does it: open e in_progress van por el cambio de la tarea, y cualquier
// otro estado es una entrada que la API no puede servir, porque cerrar lleva
// su motivo y va por su ruta.
func (s *Server) changeStatus(w nethttp.ResponseWriter, r *nethttp.Request, ref string, entry taskPatch) {
	if entry.Title != nil || entry.Description != nil || entry.Notes != nil ||
		entry.Assignee != nil || entry.Priority != nil || entry.Tags != nil {
		s.writeError(w, badEntry("el estado va solo en el cambio"))

		return
	}

	if *entry.Status != statusOpen && *entry.Status != statusInProgress {
		s.writeError(w, badEntry("el estado %q no se puede poner: para cerrar usa su ruta con el motivo", *entry.Status))

		return
	}

	task, err := s.core.UpdateTask(r.Context(), ref, core.Changes{Status: entry.Status})
	if err != nil {
		s.writeError(w, err)

		return
	}

	writeJSON(w, nethttp.StatusOK, task)
}

// closeEntry is the body a close speaks: its reason.
type closeEntry struct {
	Reason string `json:"reason"`
}

// closeTask closes the task with the reason the entry carries.
func (s *Server) closeTask(w nethttp.ResponseWriter, r *nethttp.Request) {
	writeTaskFromEntry(s, w, r,
		func(entry closeEntry) string { return entry.Reason },
		badEntry("falta el motivo de cierre"),
		(*core.Core).CloseTask,
	)
}

// blockerEntry is the body a block speaks: the blocker's reference.
type blockerEntry struct {
	Blocker string `json:"blocker"`
}

// addBlocker leaves the task blocked by the one the entry names.
func (s *Server) addBlocker(w nethttp.ResponseWriter, r *nethttp.Request) {
	writeTaskFromEntry(s, w, r,
		func(entry blockerEntry) string { return entry.Blocker },
		badEntry("falta la referencia de la tarea bloqueante"),
		(*core.Core).BlockTask,
	)
}

// removeBlocker removes the block the route names.
func (s *Server) removeBlocker(w nethttp.ResponseWriter, r *nethttp.Request) {
	task, err := s.core.UnblockTask(r.Context(), r.PathValue("ref"), r.PathValue("blocker"))
	if err != nil {
		s.writeError(w, err)

		return
	}

	writeJSON(w, nethttp.StatusOK, task)
}

// commentEntry is the body a comment speaks: its body.
type commentEntry struct {
	Body string `json:"body"`
}

// addComment appends the comment the entry carries.
func (s *Server) addComment(w nethttp.ResponseWriter, r *nethttp.Request) {
	writeTaskFromEntry(s, w, r,
		func(entry commentEntry) string { return entry.Body },
		badEntry("falta el cuerpo del comentario"),
		(*core.Core).AddComment,
	)
}

// writeTaskFromEntry es el cuerpo compartido por las rutas que llevan una
// referencia y un campo de texto: lee la entrada, comprueba que el campo
// obligatorio no viene vacío, llama a la operación del núcleo y contesta con la
// tarea tal como queda. La entrada que falta el helper la recibe ya construida,
// porque su formato debe ser constante en quien la nombra.
func writeTaskFromEntry[E any](s *Server, w nethttp.ResponseWriter, r *nethttp.Request, field func(E) string, refusal error, run func(*core.Core, context.Context, string, string) (core.Task, error)) {
	entry, err := decode[E](r)
	if err != nil {
		s.writeError(w, err)

		return
	}

	if strings.TrimSpace(field(entry)) == "" {
		s.writeError(w, refusal)

		return
	}

	task, err := run(s.core, r.Context(), r.PathValue("ref"), field(entry))
	if err != nil {
		s.writeError(w, err)

		return
	}

	writeJSON(w, nethttp.StatusOK, task)
}

// tagEntry is the body a creation of tag speaks.
type tagEntry struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

// createTag creates a tag in the catalogue.
func (s *Server) createTag(w nethttp.ResponseWriter, r *nethttp.Request) {
	entry, err := decode[tagEntry](r)
	if err != nil {
		s.writeError(w, err)

		return
	}

	tag, err := s.core.CreateTag(r.Context(), entry.Name, entry.Color, entry.Description)
	if err != nil {
		s.writeError(w, err)

		return
	}

	writeJSON(w, nethttp.StatusCreated, tag)
}

// tagPatch is the body a change of tag speaks: only what is named is touched.
type tagPatch struct {
	Name        *string `json:"name"`
	Color       *string `json:"color"`
	Description *string `json:"description"`
}

// patchTag changes what the change of tag asks for.
func (s *Server) patchTag(w nethttp.ResponseWriter, r *nethttp.Request) {
	entry, err := decode[tagPatch](r)
	if err != nil {
		s.writeError(w, err)

		return
	}

	changes := core.TagChanges{
		Name:        entry.Name,
		Color:       entry.Color,
		Description: entry.Description,
	}

	tag, err := s.core.EditTag(r.Context(), r.PathValue("name"), changes)
	if err != nil {
		s.writeError(w, err)

		return
	}

	writeJSON(w, nethttp.StatusOK, tag)
}

// deleteTag removes the tag and its assignments, and answers with it.
func (s *Server) deleteTag(w nethttp.ResponseWriter, r *nethttp.Request) {
	tag, _, err := s.core.DeleteTag(r.Context(), r.PathValue("name"))
	if err != nil {
		s.writeError(w, err)

		return
	}

	writeJSON(w, nethttp.StatusOK, tag)
}

// listTags answers with the catalogue of tags.
func (s *Server) listTags(w nethttp.ResponseWriter, r *nethttp.Request) {
	catalogue, err := s.core.Tags(r.Context())
	if err != nil {
		s.writeError(w, err)

		return
	}

	writeJSON(w, nethttp.StatusOK, catalogue)
}

// status answers with the product's version, the version of the schema and the
// prefix in use.
func (s *Server) status(w nethttp.ResponseWriter, r *nethttp.Request) {
	reported, err := s.core.Status(r.Context())
	if err != nil {
		s.writeError(w, err)

		return
	}

	writeJSON(w, nethttp.StatusOK, reported)
}

// events keeps the channel open and writes one event per change, each with its
// identity so whoever reads continues where it left off. Los avisos se leen de
// la base: los escribió la misma transacción del cambio, sea de esta sesión o
// de la del CLI, y por eso dos sesiones se ven.
func (s *Server) events(w nethttp.ResponseWriter, r *nethttp.Request) {
	flusher, canFlush := w.(nethttp.Flusher)
	if !canFlush {
		s.writeError(w, badEntry("quien lee no admite un canal abierto"))

		return
	}

	after, err := s.startOfStream(r.Context(), r)
	if err != nil {
		s.writeError(w, err)

		return
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")

	w.WriteHeader(nethttp.StatusOK)

	// El punto de partida del comparador: sin el listado de ahora, el primer
	// lote con cambios no podría decir qué tarea cambió ni cómo.
	previous, err := s.core.Tasks(r.Context())
	if err != nil {
		s.log.Warn("leyendo las tareas para el canal", "error", err.Error())

		previous = nil
	}

	for {
		events, err := s.core.Events(r.Context(), after)
		if err != nil {
			s.log.Warn("leyendo los avisos para el canal", "error", err.Error())

			return
		}

		described, seen := s.describeBatch(r, previous, events)

		previous = seen

		for _, event := range events {
			payload, err := json.Marshal(noticeOf(event, described))
			if err != nil {
				s.log.Warn("preparando el aviso del canal", "error", err.Error())

				return
			}

			if _, err := fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", event.ID, workspaceEventName, payload); err != nil {
				// Quien lee se fue: el canal termina con él.
				return
			}

			after = event.ID
		}

		flusher.Flush()

		select {
		case <-r.Context().Done():
			return
		case <-s.closing:
			return
		case <-time.After(pollInterval):
		}
	}
}

// describeBatch says what happened to the tasks a batch touched: compara el
// listado que el canal vio la última vez con el que ve ahora y deja el nuevo
// como punto de partida del lote siguiente.
func (s *Server) describeBatch(r *nethttp.Request, previous []core.Task, events []core.Event) (map[string]taskChange, []core.Task) {
	if len(events) == 0 {
		return nil, previous
	}

	current, err := s.core.Tasks(r.Context())
	if err != nil {
		s.log.Warn("leyendo las tareas para el canal", "error", err.Error())

		return nil, previous
	}

	return describeChanges(previous, current), current
}

// startOfStream says where the channel starts: where the reader left off, or
// the change after the newest one when nobody says it. Quien abre la pantalla
// acaba de leer el estado del proyecto, así que el historial no es suyo.
func (s *Server) startOfStream(ctx context.Context, r *nethttp.Request) (int64, error) {
	after, seen, err := lastSeen(r)
	if err != nil {
		return 0, err
	}

	if seen {
		return after, nil
	}

	return s.core.LastEventID(ctx)
}

// lastSeen reads where the reader left off: the identity of the last event it
// saw, in the header of a reconnection or in the query of a first read, and
// whether either of them said it. Una identidad que no es un número no se puede
// cumplir, así que se dice en vez de empezar por el principio en silencio.
func lastSeen(r *nethttp.Request) (int64, bool, error) {
	for _, seen := range []string{r.Header.Get("Last-Event-ID"), r.URL.Query().Get("after")} {
		if seen == "" {
			continue
		}

		id, err := strconv.ParseInt(seen, 10, 64)
		if err != nil || id < 0 {
			return 0, false, badEntry("la identidad desde la que continuar no es un número")
		}

		return id, true, nil
	}

	return 0, false, nil
}

// decode reads the body of a request into the entry of the route. Un cuerpo que
// no se entiende es entrada ilegible: no llega al núcleo.
func decode[T any](r *nethttp.Request) (T, error) {
	var entry T

	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		return entry, badEntry("el cuerpo de la petición no es un JSON válido")
	}

	return entry, nil
}

// statusOf translates an error into the status code the contract names, by the
// kind of the error and never by its text.
func statusOf(err error) int {
	switch {
	case errors.Is(err, errBadEntry), errors.Is(err, core.ErrInvalid):
		return nethttp.StatusBadRequest
	case errors.Is(err, core.ErrNotFound):
		return nethttp.StatusNotFound
	case errors.Is(err, core.ErrConflict):
		return nethttp.StatusConflict
	default:
		return nethttp.StatusInternalServerError
	}
}

// writeError answers with the body of an error: its message, the same the
// command line would give, under the status code it belongs to.
func (s *Server) writeError(w nethttp.ResponseWriter, err error) {
	writeJSON(w, statusOf(err), errorBody{Error: err.Error()})
}

// errorBody is how an error crosses the API's boundary.
type errorBody struct {
	Error string `json:"error"`
}

// writeJSON writes the value as the body of the answer.
func writeJSON(w nethttp.ResponseWriter, status int, value any) {
	raw, err := json.Marshal(value)
	if err != nil {
		w.WriteHeader(nethttp.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}
