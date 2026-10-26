package core_test

import (
	"errors"
	"strings"
	"tasks-ai/internal/core"
	"tasks-ai/internal/store"
	"testing"
	"time"
)

// createADR escribe un ADR propuesto con lo mínimo que la regla pide.
func createADR(t *testing.T, apply *core.Core, title string) core.ADR {
	t.Helper()

	adr, err := apply.CreateADR(t.Context(), core.NewADR{
		Title:    title,
		Deciders: "Alfredo González Portero",
		Context:  "el contexto",
	})
	if err != nil {
		t.Fatalf("creando el ADR %q: %v", title, err)
	}

	return adr
}

func TestCreateADR_givesItTheNumberTheReferenceAndTheSlug(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")

	first := createADR(t, apply, "Los ADR viven en la base")
	second := createADR(t, apply, "Los ADR viven en la base y se exportan")

	if first.Number != 1 || first.Ref != "ADR-001" {
		t.Errorf("el primer ADR nace con número %d y referencia %q, se esperaban 1 y ADR-001", first.Number, first.Ref)
	}

	if second.Ref != "ADR-002" {
		t.Errorf("el segundo ADR es %q, se esperaba ADR-002", second.Ref)
	}

	if first.Slug != "los-adr-viven-en-la-base" {
		t.Errorf("el slug es %q, se esperaba sin acentos y con guiones", first.Slug)
	}

	if first.Status != store.ADRProposed {
		t.Errorf("el ADR nace en %q, se esperaba proposed", first.Status)
	}

	if today := time.Now().UTC().Format("2006-01-02"); first.Date != today {
		t.Errorf("la fecha es %q, se esperaba la del día %q", first.Date, today)
	}
}

func TestCreateADR_refusesWhatTheRegisterCannotHold(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")

	decided := "Alfredo González Portero"

	cases := []struct {
		name string
		adr  core.NewADR
	}{
		{"sin título", core.NewADR{Deciders: decided}},
		{"sin decisores", core.NewADR{Title: "Una decisión"}},
		{"con un estado que no existe", core.NewADR{Title: "Una decisión", Deciders: decided, Status: "approved"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			if _, err := apply.CreateADR(t.Context(), c.adr); !errors.Is(err, core.ErrInvalid) {
				t.Errorf("crear %s devolvió %v, se esperaba entrada inválida", c.name, err)
			}
		})
	}

	adrs, err := apply.ADRs(t.Context())
	if err != nil {
		t.Fatalf("leyendo los ADR: %v", err)
	}

	if len(adrs) != 0 {
		t.Errorf("un rechazo dejó %+v escritos, se esperaba la base vacía", adrs)
	}
}

func TestUpdateADR_freezesTheBodyWhenItLeavesProposed(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")
	created := createADR(t, apply, "Una decisión")

	accepted := "accepted"
	if _, err := apply.UpdateADR(t.Context(), created.Ref, core.ADRChanges{Status: &accepted}); err != nil {
		t.Fatalf("aceptando el ADR: %v", err)
	}

	body := "otra decisión"
	title := "Otro título"

	if _, err := apply.UpdateADR(t.Context(), created.Ref, core.ADRChanges{Decision: &body}); !errors.Is(err, core.ErrInvalid) {
		t.Errorf("cambiar la decisión de un ADR aceptado devolvió %v, se esperaba entrada inválida", err)
	}

	if _, err := apply.UpdateADR(t.Context(), created.Ref, core.ADRChanges{Title: &title}); !errors.Is(err, core.ErrInvalid) {
		t.Errorf("cambiar el título de un ADR aceptado devolvió %v, se esperaba entrada inválida", err)
	}

	read, err := apply.ADR(t.Context(), created.Ref)
	if err != nil {
		t.Fatalf("leyendo el ADR: %v", err)
	}

	if read.Decision != created.Decision || read.Title != created.Title {
		t.Errorf("un rechazo cambió el cuerpo a %+v, se esperaba %+v", read, created)
	}

	// La nota sí cabe: es la única escritura que el cuerpo congelado admite.
	note := "se comprobó al implementarlo"

	noted, err := apply.UpdateADR(t.Context(), created.Ref, core.ADRChanges{Note: &note})
	if err != nil {
		t.Fatalf("añadiendo la nota: %v", err)
	}

	if noted.MoreInfo == nil || !strings.Contains(*noted.MoreInfo, note) {
		t.Errorf("la información adicional es %v, se esperaba la nota", noted.MoreInfo)
	}
}

func TestUpdateADR_supersedesWithAnExistingSubstituteAndWithoutCycles(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")

	old := createADR(t, apply, "La decisión anterior")
	next := createADR(t, apply, "La decisión nueva")

	superseded := "superseded"
	missing := 99

	if _, err := apply.UpdateADR(t.Context(), old.Ref, core.ADRChanges{
		Status: &superseded, SupersededBy: &missing,
	}); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("sustituir por un ADR que no está devolvió %v, se esperaba no encontrado", err)
	}

	self := old.Number

	if _, err := apply.UpdateADR(t.Context(), old.Ref, core.ADRChanges{
		Status: &superseded, SupersededBy: &self,
	}); !errors.Is(err, core.ErrInvalid) {
		t.Errorf("sustituirse a sí mismo devolvió %v, se esperaba entrada inválida", err)
	}

	substitute := next.Number

	replaced, err := apply.UpdateADR(t.Context(), old.Ref, core.ADRChanges{
		Status: &superseded, SupersededBy: &substitute,
	})
	if err != nil {
		t.Fatalf("sustituyendo el ADR: %v", err)
	}

	if replaced.Status != store.ADRSuperseded || replaced.SupersededBy == nil || *replaced.SupersededBy != next.Ref {
		t.Errorf("el ADR sustituido es %+v, se esperaba superseded por %q", replaced, next.Ref)
	}

	if replaced.Decision != old.Decision || replaced.Context != old.Context {
		t.Errorf("sustituir reescribió el cuerpo: %+v, se esperaba %+v", replaced, old)
	}

	// El ciclo: el sustituto ya no puede sustituirse por el anterior.
	back := old.Number

	if _, err := apply.UpdateADR(t.Context(), next.Ref, core.ADRChanges{
		Status: &superseded, SupersededBy: &back,
	}); !errors.Is(err, core.ErrInvalid) {
		t.Errorf("cerrar el ciclo devolvió %v, se esperaba entrada inválida", err)
	}
}

func TestADR_readsTheNumberOrTheReference(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")
	created := createADR(t, apply, "Una decisión")

	for _, ref := range []string{"1", "ADR-001", " adr-001 "} {
		read, err := apply.ADR(t.Context(), ref)
		if err != nil {
			t.Fatalf("leyendo el ADR por %q: %v", ref, err)
		}

		if read.Ref != created.Ref {
			t.Errorf("leer por %q devolvió %q, se esperaba %q", ref, read.Ref, created.Ref)
		}
	}

	if _, err := apply.ADR(t.Context(), "ADR-099"); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("leer un número que no está devolvió %v, se esperaba no encontrado", err)
	}

	if _, err := apply.ADR(t.Context(), "la primera"); !errors.Is(err, core.ErrInvalid) {
		t.Errorf("leer una referencia que no lo es devolvió %v, se esperaba entrada inválida", err)
	}
}

func TestEvents_announcesTheADRWithItsReferenceAndTitle(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")

	last, err := apply.LastEventID(t.Context())
	if err != nil {
		t.Fatalf("leyendo el último aviso: %v", err)
	}

	created := createADR(t, apply, "Una decisión")

	events, err := apply.Events(t.Context(), last)
	if err != nil {
		t.Fatalf("leyendo los avisos: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("el cambio dejó %d avisos, se esperaba uno", len(events))
	}

	event := events[0]

	if event.Resource != store.ResourceADR || event.Ref != created.Ref || event.Title != created.Title {
		t.Errorf("el aviso es %+v, se esperaba el ADR %q con su título", event, created.Ref)
	}
}

func TestImportADRs_isIdempotentAndWaitsForTheSubstitute(t *testing.T) {
	t.Parallel()

	apply := core.New(openStore(t), "tk")

	// El 2 se sustituye por el 3, que todavía no está: la importación lo deja
	// para la pasada siguiente en vez de guardar un enlace roto.
	entries := []core.NewADR{
		{Number: 1, Slug: "la-primera", Title: "La primera", Status: store.ADRAccepted, Date: "2026-10-20", Deciders: "Alfredo"},
		{Number: 2, Slug: "la-segunda", Title: "La segunda", Status: store.ADRSuperseded, SupersededBy: 3, Date: "2026-10-21", Deciders: "Alfredo"},
		{Number: 3, Slug: "la-tercera", Title: "La tercera", Status: store.ADRAccepted, Date: "2026-10-22", Deciders: "Alfredo"},
	}

	result, err := apply.ImportADRs(t.Context(), entries)
	if err != nil {
		t.Fatalf("importando: %v", err)
	}

	if len(result.Imported) != 3 || len(result.Skipped) != 0 {
		t.Errorf("la importación dejó %+v, se esperaban los tres importados", result)
	}

	second, err := apply.ADR(t.Context(), "2")
	if err != nil {
		t.Fatalf("leyendo el sustituido: %v", err)
	}

	if second.SupersededBy == nil || *second.SupersededBy != "ADR-003" {
		t.Errorf("el sustituto importado es %v, se esperaba ADR-003", second.SupersededBy)
	}

	if second.Date != "2026-10-21" {
		t.Errorf("la fecha importada es %q, se esperaba la del archivo", second.Date)
	}

	// Repetirla no duplica ni reescribe.
	again, err := apply.ImportADRs(t.Context(), entries)
	if err != nil {
		t.Fatalf("repitiendo la importación: %v", err)
	}

	if len(again.Imported) != 0 || len(again.Skipped) != 3 {
		t.Errorf("repetir la importación dejó %+v, se esperaban los tres saltados", again)
	}
}
