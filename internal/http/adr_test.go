package http_test

import (
	"net/http"
	"strings"
	"tasks-ai/internal/core"
	"testing"
)

// createADRByAPI crea un ADR por la API y devuelve lo que respondió.
func createADRByAPI(t *testing.T, server, title string) core.ADR {
	t.Helper()

	status, body := do(t, server, http.MethodPost, "/api/adrs",
		`{"title":"`+title+`","deciders":"Alfredo","context":"el contexto"}`)
	if status != http.StatusCreated {
		t.Fatalf("crear el ADR respondió %d: %s", status, body)
	}

	var created core.ADR
	decode(t, body, &created)

	return created
}

func TestServer_createsReadsAndEditsTheADRs(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	created := createADRByAPI(t, server, "Los ADR viven en la base")

	if created.Ref != "ADR-001" || created.Status != "proposed" || created.Slug != "los-adr-viven-en-la-base" {
		t.Errorf("el ADR creado es %+v, se esperaba ADR-001 propuesto con su slug", created)
	}

	status, body := do(t, server, http.MethodGet, "/api/adrs/1", "")
	if status != http.StatusOK {
		t.Fatalf("leer el ADR respondió %d: %s", status, body)
	}

	var read core.ADR
	decode(t, body, &read)

	if read.Ref != created.Ref || read.Context != "el contexto" {
		t.Errorf("el ADR leído es %+v, se esperaba el creado", read)
	}

	status, body = do(t, server, http.MethodGet, "/api/adrs", "")
	if status != http.StatusOK {
		t.Fatalf("listar los ADR respondió %d: %s", status, body)
	}

	if strings.Contains(body, `"context"`) {
		t.Errorf("el listado trae cuerpos, se esperaban resúmenes: %s", body)
	}

	var listed []core.ADRSummary
	decode(t, body, &listed)

	if len(listed) != 1 || listed[0].Ref != created.Ref {
		t.Errorf("el listado es %+v, se esperaba el ADR creado", listed)
	}

	status, body = do(t, server, http.MethodPatch, "/api/adrs/1", `{"decision":"la decisión"}`)
	if status != http.StatusOK {
		t.Fatalf("cambiar el ADR respondió %d: %s", status, body)
	}

	var edited core.ADR
	decode(t, body, &edited)

	if edited.Decision != "la decisión" {
		t.Errorf("la decisión guardada es %q, se esperaba la editada", edited.Decision)
	}
}

func TestServer_refusesWhatTheADRRulesRefuse(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	createADRByAPI(t, server, "Una decisión")

	cases := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{"una entrada sin título", http.MethodPost, "/api/adrs", `{"deciders":"Alfredo"}`, http.StatusBadRequest},
		{"un estado que no existe", http.MethodPatch, "/api/adrs/1", `{"status":"approved"}`, http.StatusBadRequest},
		{"un número que no está", http.MethodGet, "/api/adrs/99", "", http.StatusNotFound},
		{"un sustituto que no está", http.MethodPatch, "/api/adrs/1", `{"status":"superseded","superseded_by":"ADR-099"}`, http.StatusNotFound},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			status, body := do(t, server, c.method, c.path, c.body)
			if status != c.want {
				t.Errorf("%s respondió %d, se esperaba %d: %s", c.name, status, c.want, body)
			}
		})
	}

	// El cuerpo congelado es entrada inválida, y el rechazo no toca el ADR.
	if status, body := do(t, server, http.MethodPatch, "/api/adrs/1", `{"status":"accepted"}`); status != http.StatusOK {
		t.Fatalf("aceptar el ADR respondió %d: %s", status, body)
	}

	status, body := do(t, server, http.MethodPatch, "/api/adrs/1", `{"decision":"otra"}`)
	if status != http.StatusBadRequest {
		t.Errorf("cambiar el cuerpo congelado respondió %d, se esperaba 400: %s", status, body)
	}

	if text := errorOf(t, body); !strings.Contains(text, "frozen") {
		t.Errorf("el rechazo no dice que el cuerpo está congelado: %q", text)
	}
}

// TestServer_theNoticeOfAnADRChangeNamesIt comprueba que el canal anuncia el
// ADR con su referencia y su título, y que los avisos de tareas siguen igual.
func TestServer_theNoticeOfAnADRChangeNamesIt(t *testing.T) {
	t.Parallel()

	server, base := newServer(t)
	apply := core.New(base, "tk")

	events := openEvents(t, server+"/api/events")

	adr, err := apply.CreateADR(t.Context(), core.NewADR{Title: "El ADR avisado", Deciders: "Alfredo"})
	if err != nil {
		t.Fatalf("creando el ADR por el núcleo: %v", err)
	}

	change := readNotice(t, readRawEvent(t, events))

	if change.Resource != "adrs" {
		t.Errorf("el aviso habla del recurso %q, se esperaba adrs", change.Resource)
	}

	if len(change.Changes) != 1 || change.Changes[0] != (taskChange{ID: adr.Ref, Action: createdAction, Title: "El ADR avisado"}) {
		t.Errorf("el aviso del ADR dice %+v, se esperaba %s creada como “El ADR avisado”", change.Changes, adr.Ref)
	}
}
