package http_test

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// La interfaz viaja incrustada en el binario y la raíz del servidor es suya,
// con la API conservando sus rutas /api/... (ADR-010).

func TestServer_servesTheInterfaceAtTheRoot(t *testing.T) {
	t.Parallel()

	server, _ := newServer(t)

	status, header := fetch(t, server, "/")
	if status != http.StatusFound || header.Get("Location") != "/task-manager/app-vue.html" {
		t.Fatalf("la raíz respondió %d con %v, se esperaba el salto a la página de la interfaz", status, header)
	}

	status, header = fetch(t, server, "/task-manager/app-vue.html")
	if status != http.StatusOK || !strings.Contains(header.Get("Content-Type"), "text/html") {
		t.Fatalf("la página de la interfaz respondió %d con %v, se esperaba HTML", status, header)
	}

	status, header = fetch(t, server, "/task-manager/app-vue.js")
	if status != http.StatusOK || !strings.Contains(header.Get("Content-Type"), "text/javascript") {
		t.Fatalf("el paquete de la interfaz respondió %d con %v, se esperaba JavaScript", status, header)
	}

	status, header = fetch(t, server, "/vendor/vue.global.prod.js")
	if status != http.StatusOK || !strings.Contains(header.Get("Content-Type"), "javascript") {
		t.Fatalf("el Vue vendorizado respondió %d con %v, se esperaba JavaScript", status, header)
	}
}

// fetch pide una ruta del servidor tal como la pide un navegador y devuelve
// el estado con la cabecera de la respuesta; el cuerpo no hace falta. Sin
// seguir los saltos: la raíz responde con el suyo y no con el de la página.
func fetch(t *testing.T, server, path string) (int, http.Header) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server+path, nil)
	if err != nil {
		t.Fatalf("preparando la petición de %s: %v", path, err)
	}

	response, err := (&http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}).Do(request)
	if err != nil {
		t.Fatalf("pidiendo %s: %v", path, err)
	}

	defer func() { _ = response.Body.Close() }()

	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		t.Fatalf("leyendo la respuesta de %s: %v", path, err)
	}

	return response.StatusCode, response.Header
}
