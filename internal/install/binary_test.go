package install

import "testing"

// TestBinaryName_carriesTheExtensionWindowsNeeds comprueba que el binario que la
// instalación copia en el proyecto se llama como el sistema espera ejecutarlo: sin
// la extensión, Windows no arranca la copia y las extensiones del agente no la
// encuentran.
func TestBinaryName_carriesTheExtensionWindowsNeeds(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"windows": "tasks-ai.exe",
		"linux":   "tasks-ai",
		"darwin":  "tasks-ai",
	}

	for goos, want := range cases {
		if got := binaryName(goos); got != want {
			t.Errorf("binaryName(%q) devolvió %q, se esperaba %q", goos, got, want)
		}
	}
}
