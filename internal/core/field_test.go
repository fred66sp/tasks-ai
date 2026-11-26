package core

import (
	"errors"
	"testing"
)

// El campo de entrada viaja en el error: una superficie que puede reparar la
// entrada lo lee del error, sin interpretar el mensaje. Una línea por familia
// de regla: la referencia, cada campo con regla propia y el texto rechazado.
func TestInputErrorNamesItsField(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		field string
	}{
		{
			name:  "a missing close reason",
			err:   checkCloseReason("  "),
			field: fieldReason,
		},
		{
			name:  "an unknown close reason",
			err:   checkCloseReason("eventually"),
			field: fieldReason,
		},
		{
			name:  "a title without text",
			err:   func() error { _, err := checkTitle("   "); return err }(),
			field: fieldTitle,
		},
		{
			name:  "a priority outside its range",
			err:   checkPriority(MaxPriority + 1),
			field: fieldPriority,
		},
		{
			name:  "a missing comment body",
			err:   func() error { _, err := commentBody(""); return err }(),
			field: fieldBody,
		},
		{
			name: "a comment that writes the state",
			err:  func() error { _, err := commentBody("refixed origin/HEAD on the delivery"); return err }(),
			// La entrada se llama body: la regla de texto lee el cuerpo con
			// otra etiqueta, pero el campo que repara el consumidor es el suyo.
			field: fieldBody,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("the entry was not refused")

				return
			}

			var in InvalidField
			if !errors.As(tt.err, &in) {
				t.Fatalf("error %v is not an input error with a field", tt.err)

				return
			}

			if got := in.Field(); got != tt.field {
				t.Errorf("field = %q, want %q", got, tt.field)
			}
		})
	}
}
