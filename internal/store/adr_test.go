package store_test

import (
	"errors"
	"path/filepath"
	"strings"
	"tasks-ai/internal/store"
	"testing"
)

// newADR builds the ADR a creation stores, with the fields the base requires.
func newADR(title string) store.ADR {
	return store.ADR{
		Slug:         strings.ReplaceAll(strings.ToLower(title), " ", "-"),
		Title:        title,
		Status:       store.ADRProposed,
		Date:         "2026-10-26",
		Deciders:     "Alfredo González Portero",
		Context:      "el contexto",
		Decision:     "la decisión",
		Consequences: "las consecuencias",
	}
}

func TestCreateADR_givesTheNextNumberAndKeepsItRunning(t *testing.T) {
	t.Parallel()

	base := openStore(t)

	first, err := base.CreateADR(t.Context(), newADR("Primera decisión"))
	if err != nil {
		t.Fatalf("creando el primer ADR: %v", err)
	}

	second, err := base.CreateADR(t.Context(), newADR("Segunda decisión"))
	if err != nil {
		t.Fatalf("creando el segundo ADR: %v", err)
	}

	if first.Number != 1 || second.Number != 2 {
		t.Errorf("los números son %d y %d, se esperaban 1 y 2", first.Number, second.Number)
	}

	// Un número pedido por la importación no se repite: el siguiente sigue
	// contando desde el mayor guardado.
	imported, err := base.CreateADR(t.Context(), store.ADR{
		Number:   9,
		Slug:     "una-decision-importada",
		Title:    "Una decisión importada",
		Status:   store.ADRAccepted,
		Date:     "2026-10-20",
		Deciders: "Alfredo González Portero",
	})
	if err != nil {
		t.Fatalf("importando un ADR con su número: %v", err)
	}

	after, err := base.CreateADR(t.Context(), newADR("La que sigue"))
	if err != nil {
		t.Fatalf("creando el ADR siguiente: %v", err)
	}

	if imported.Number != 9 || after.Number != 10 {
		t.Errorf("la importación quedó en %d y la siguiente en %d, se esperaban 9 y 10", imported.Number, after.Number)
	}
}

func TestCreateADR_refusesANumberThatIsAlreadyThere(t *testing.T) {
	t.Parallel()

	base := openStore(t)

	if _, err := base.CreateADR(t.Context(), newADR("Primera")); err != nil {
		t.Fatalf("creando el primer ADR: %v", err)
	}

	taken := newADR("Otra con el mismo número")
	taken.Number = 1

	if _, err := base.CreateADR(t.Context(), taken); !errors.Is(err, store.ErrADRNumberTaken) {
		t.Errorf("repetir el número devolvió %v, se esperaba ErrADRNumberTaken", err)
	}

	adrs, err := base.ADRs(t.Context())
	if err != nil {
		t.Fatalf("leyendo los ADR: %v", err)
	}

	if len(adrs) != 1 || adrs[0].Title != "Primera" {
		t.Errorf("la base quedó con %+v, se esperaba el primer ADR intacto", adrs)
	}
}

func TestUpdateADR_changesTheBodyAndKeepsTheIdentity(t *testing.T) {
	t.Parallel()

	base := openStore(t)

	created, err := base.CreateADR(t.Context(), newADR("La decisión"))
	if err != nil {
		t.Fatalf("creando el ADR: %v", err)
	}

	updated, err := base.UpdateADR(t.Context(), created.Number, func(adr store.ADR) (store.ADR, error) {
		adr.Decision = "la decisión corregida"

		return adr, nil
	})
	if err != nil {
		t.Fatalf("cambiando el ADR: %v", err)
	}

	if updated.Decision != "la decisión corregida" {
		t.Errorf("la decisión guardada es %q, se esperaba la corregida", updated.Decision)
	}

	if updated.ID != created.ID || updated.Number != created.Number || updated.Slug != created.Slug {
		t.Errorf("el cambio tocó la identidad: %+v, se esperaba %+v", updated, created)
	}

	if !updated.UpdatedAt.After(created.UpdatedAt) && !updated.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("la fecha de cambio es %v, se esperaba la de ahora", updated.UpdatedAt)
	}
}

func TestADRs_readsTheSubstituteByItsNumber(t *testing.T) {
	t.Parallel()

	base := openStore(t)

	old, err := base.CreateADR(t.Context(), newADR("La anterior"))
	if err != nil {
		t.Fatalf("creando la anterior: %v", err)
	}

	next, err := base.CreateADR(t.Context(), newADR("La sustituta"))
	if err != nil {
		t.Fatalf("creando la sustituta: %v", err)
	}

	if _, err := base.UpdateADR(t.Context(), old.Number, func(adr store.ADR) (store.ADR, error) {
		adr.Status = store.ADRSuperseded
		adr.SupersededBy = &next.Number

		return adr, nil
	}); err != nil {
		t.Fatalf("sustituyendo la anterior: %v", err)
	}

	read, err := base.ADR(t.Context(), old.Number)
	if err != nil {
		t.Fatalf("leyendo la anterior: %v", err)
	}

	if read.SupersededBy == nil || *read.SupersededBy != next.Number {
		t.Errorf("el sustituto guardado es %v, se esperaba el número %d", read.SupersededBy, next.Number)
	}
}

// TestOpen_migratesSchema6WithoutLosingTheBoard comprueba la migración aditiva:
// una base del esquema anterior, con tareas y comentarios, gana la tabla de los
// ADR sin perder lo que tenía.
func TestOpen_migratesSchema6WithoutLosingTheBoard(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "tasks.db")

	base, err := store.Open(t.Context(), path)
	if err != nil {
		t.Fatalf("abriendo la base: %v", err)
	}

	task, err := base.CreateTask(t.Context(), newTask("Con comentario"), nil)
	if err != nil {
		t.Fatalf("creando la tarea: %v", err)
	}

	if _, err := base.AddComment(t.Context(), task.ID, "conservar"); err != nil {
		t.Fatalf("añadiendo el comentario: %v", err)
	}

	if err := base.Close(); err != nil {
		t.Fatalf("cerrando la base: %v", err)
	}

	// Simula una base de esquema 6, que todavía no conoce los ADR.
	rawBase(t, path, "DROP TABLE adrs")
	rawBase(t, path, "UPDATE meta SET value = '6' WHERE key = 'schema_version'")

	again, err := store.Open(t.Context(), path)
	if err != nil {
		t.Fatalf("migrando la base: %v", err)
	}

	defer again.Close() //nolint:errcheck // las aserciones de abajo ya deciden el resultado

	if version, err := again.SchemaVersion(t.Context()); err != nil || version != 7 {
		t.Errorf("la base migrada quedó en el esquema %d (%v), se esperaba el 7", version, err)
	}

	comments, err := again.Comments(t.Context(), task.ID)
	if err != nil || len(comments) != 1 || comments[0].Body != "conservar" {
		t.Errorf("los comentarios tras migrar son %+v (%v), se esperaba conservarlos", comments, err)
	}

	adrs, err := again.ADRs(t.Context())
	if err != nil || len(adrs) != 0 {
		t.Errorf("los ADR de una base migrada son %+v (%v), se esperaba ninguno", adrs, err)
	}
}
