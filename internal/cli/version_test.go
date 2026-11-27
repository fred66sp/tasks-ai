package cli_test

import (
	"tasks-ai/internal/core"
	"testing"
)

// TestRun_versionPrintsTheProductVersion comprueba la superficie del comando.
func TestRun_versionPrintsTheProductVersion(t *testing.T) {
	t.Chdir(t.TempDir())

	code, out, errOut := run(t, "version")
	if code != 0 {
		t.Fatalf("version salió con %d: %s", code, errOut)
	}

	var body struct {
		Version string `json:"version"`
	}
	decode(t, out, &body)

	if body.Version != core.Version {
		t.Errorf("version devolvió %q, se esperaba %q", body.Version, core.Version)
	}
}
