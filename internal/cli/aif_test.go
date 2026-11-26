package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	interaction "github.com/fred66sp/agent-interaction-forms"
)

// TestRunAIF_answersDiscoveryThroughTheChannels recorre el canal del perfil:
// una petición por la entrada estándar, un único objeto en la salida con su
// salto y nada más, y el código 0 del descubrimiento.
func TestRunAIF_answersDiscoveryThroughTheChannels(t *testing.T) {
	t.Chdir(t.TempDir())

	if code, _, errOut := runWith(t, "", "init"); code != 0 {
		t.Fatalf("init salió con %d: %s", code, errOut)
	}

	code, out, errOut := runWith(t, `{"aif":"0.2"}`, "aif")
	if code != 0 {
		t.Fatalf("el descubrimiento salió con %d, se esperaba 0: %s", code, errOut)
	}

	if errOut != "" {
		t.Errorf("el descubrimiento escribió en la salida de errores: %s", errOut)
	}

	if !strings.HasSuffix(out, "\n") {
		t.Errorf("la respuesta no termina en salto de línea: %q", out)
	}

	var discovered interaction.Discovery
	decodeOnly(t, out, &discovered)

	if discovered.AIF != interaction.Version || len(discovered.Operations) != 11 {
		t.Errorf("el descubrimiento es %+v, se esperaba el vocabulario completo", discovered)
	}
}

// TestRunAIF_answersAMalformedRequestWithItsCode comprueba que una petición
// rota sale como objeto AIF, por la salida, con el código 3 y sin prosa.
func TestRunAIF_answersAMalformedRequestWithItsCode(t *testing.T) {
	t.Chdir(t.TempDir())

	if code, _, errOut := runWith(t, "", "init"); code != 0 {
		t.Fatalf("init salió con %d: %s", code, errOut)
	}

	code, out, errOut := runWith(t, `{"aif":"0.2"`, "aif")
	if code != 3 {
		t.Fatalf("una petición rota salió con %d, se esperaba 3: %s", code, errOut)
	}

	response := responseOf(t, out)
	if response.Result != interaction.ResultMalformedRequest {
		t.Errorf("result = %q, se esperaba %q", response.Result, interaction.ResultMalformedRequest)
	}

	if !strings.HasSuffix(out, "\n") || strings.HasPrefix(strings.TrimSpace(out), Name) {
		t.Errorf("la salida mezcla la respuesta con otra cosa: %q", out)
	}
}

// TestRunAIF_reportsAStandardInputThatCannotBeRead comprueba que un fallo real
// de la entrada no se disfraza: failed y el código 1.
func TestRunAIF_reportsAStandardInputThatCannotBeRead(t *testing.T) {
	t.Chdir(t.TempDir())

	if code, _, errOut := runWith(t, "", "init"); code != 0 {
		t.Fatalf("init salió con %d: %s", code, errOut)
	}

	var out, errOut bytes.Buffer

	code := run([]string{"aif"}, brokenReader{}, &out, &errOut)
	if code != 1 {
		t.Errorf("una entrada ilegible salió con %d, se esperaba 1", code)
	}

	response := responseOf(t, out.String())
	if response.Result != interaction.ResultFailed || response.Code != "stdin_unavailable" {
		t.Errorf("la respuesta es %+v, se esperaba failed con su código", response)
	}
}

// TestRunAIF_answersAFailureWhenTheProjectIsNotInstalled comprueba que hasta un
// fallo real del comando llega como objeto del perfil, con el código 1.
func TestRunAIF_answersAFailureWhenTheProjectIsNotInstalled(t *testing.T) {
	t.Chdir(t.TempDir())

	code, out, _ := runWith(t, `{"aif":"0.2"}`, "aif")
	if code != 1 {
		t.Fatalf("servir sin proyecto salió con %d, se esperaba 1", code)
	}

	response := responseOf(t, out)
	if response.Result != interaction.ResultFailed {
		t.Errorf("result = %q, se esperaba %q", response.Result, interaction.ResultFailed)
	}
}

// TestExitCodeOf_mapsEachResult deja escrita la tabla de CL-04: el código solo
// depende del result, y el descubrimiento sale como executed.
func TestExitCodeOf_mapsEachResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		response *interaction.Response
		want     int
	}{
		{name: "executed", response: &interaction.Response{Result: interaction.ResultExecuted}, want: 0},
		{name: "discovery", response: &interaction.Response{Discovery: &interaction.Discovery{}}, want: 0},
		{name: "input needed", response: &interaction.Response{Result: interaction.ResultInputNeeded}, want: 2},
		{name: "invalid", response: &interaction.Response{Result: interaction.ResultInvalid}, want: 2},
		{name: "unknown operation", response: &interaction.Response{Result: interaction.ResultUnknownOperation}, want: 3},
		{name: "unsupported version", response: &interaction.Response{Result: interaction.ResultUnsupportedVersion}, want: 3},
		{name: "malformed request", response: &interaction.Response{Result: interaction.ResultMalformedRequest}, want: 3},
		{name: "denied", response: &interaction.Response{Result: interaction.ResultDenied}, want: 4},
		{name: "failed", response: &interaction.Response{Result: interaction.ResultFailed}, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := exitCodeOf(tt.response); got != tt.want {
				t.Errorf("exitCodeOf(%q) = %d, se esperaba %d", tt.response.Result, got, tt.want)
			}
		})
	}
}

// brokenReader is a standard input that cannot be read at all.
type brokenReader struct{}

func (brokenReader) Read([]byte) (int, error) { return 0, errors.New("sin entrada") }

// runWith runs the command line with a standard input built from text.
func runWith(t *testing.T, stdin string, args ...string) (code int, out, errOut string) {
	t.Helper()

	var stdout, stderr bytes.Buffer

	code = run(args, strings.NewReader(stdin), &stdout, &stderr)

	return code, stdout.String(), stderr.String()
}

// responseOf reads the one object the profile writes.
func responseOf(t *testing.T, raw string) *interaction.Response {
	t.Helper()

	var body struct {
		AIF    string `json:"aif"`
		Result string `json:"result"`
		Code   string `json:"code"`
	}

	decodeOnly(t, raw, &body)

	return &interaction.Response{AIF: body.AIF, Result: body.Result, Code: body.Code}
}

// decodeOnly decodes the one JSON value raw carries and proves nothing follows.
func decodeOnly(t *testing.T, raw string, target any) {
	t.Helper()

	decoder := json.NewDecoder(strings.NewReader(raw))
	if err := decoder.Decode(target); err != nil {
		t.Fatalf("la respuesta no es un objeto JSON (%v): %q", err, raw)
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		t.Errorf("la salida lleva algo más que un objeto: %q", raw)
	}
}
