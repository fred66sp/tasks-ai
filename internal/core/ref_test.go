package core_test

import (
	"errors"
	"tasks-ai/internal/core"
	"testing"
)

func TestParseRef_marksAMalformedReferenceAsUnresolved(t *testing.T) {
	t.Parallel()

	_, err := core.ParseRef("tk", "tk-luego")
	if !errors.Is(err, core.ErrNotFound) {
		t.Errorf("ParseRef devolvió %v, se esperaba el centinela de referencia sin resolución", err)
	}

	if err == nil || err.Error() != `reference "tk-luego" is not valid: tk-<number> is expected` {
		t.Errorf("el mensaje de la referencia mal escrita es %v", err)
	}
}

func TestRef_composesTheVisibleReference(t *testing.T) {
	t.Parallel()

	tests := []struct {
		prefix   string
		id       int64
		expected string
	}{
		{prefix: "tk", id: 1, expected: "tk-1"},
		{prefix: "tk", id: 404, expected: "tk-404"},
		{prefix: "proyecto", id: 12, expected: "proyecto-12"},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			t.Parallel()

			if got := core.Ref(test.prefix, test.id); got != test.expected {
				t.Errorf("Ref(%q, %d) = %q, se esperaba %q", test.prefix, test.id, got, test.expected)
			}
		})
	}
}

func TestParseRef_returnsTheInternalNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ref      string
		expected int64
		valid    bool
	}{
		{name: "una referencia normal", ref: "tk-18", expected: 18, valid: true},
		{name: "con espacios alrededor", ref: "  tk-7  ", expected: 7, valid: true},
		{name: "escrita en mayúsculas", ref: "TK-7", expected: 7, valid: true},
		{name: "el prefijo de otro proyecto", ref: "otro-7"},
		{name: "sin número", ref: "tk-"},
		{name: "sin prefijo", ref: "7"},
		{name: "con letras detrás del prefijo", ref: "tk-7a"},
		{name: "la cadena vacía", ref: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			id, err := core.ParseRef("tk", test.ref)
			if !test.valid {
				if err == nil {
					t.Errorf("ParseRef(%q) devolvió %d, se esperaba un error", test.ref, id)
				}

				return
			}

			if err != nil {
				t.Fatalf("ParseRef(%q) devolvió %v", test.ref, err)
			}

			if id != test.expected {
				t.Errorf("ParseRef(%q) = %d, se esperaba %d", test.ref, id, test.expected)
			}
		})
	}
}
