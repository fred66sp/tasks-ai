package cli_test

import (
	"encoding/json"
	"io"
	"os"
	"regexp"
	"slices"
	"strings"
	"tasks-ai/internal/cli"
	"testing"

	"github.com/spf13/cobra"
)

// guidePath es la guía de uso. Se comprueba desde este paquete porque el paquete
// raíz incrusta los recursos y no puede importar el CLI sin un ciclo.
const guidePath = "../../docs/web/index.html"

// guideScriptPath is the shared script that also carries i18n keys.
const guideScriptPath = "../../docs/web/assets/js/docs.js"

// What the guide declares: its text keys —on the page, in docs.js and in the
// topbar config—, the English dictionary and the commands it invokes.
var (
	guideDictionary = regexp.MustCompile(`(?s)<script type="application/json" id="tasks-ai-en">(.*?)</script>`)
	guideKey        = regexp.MustCompile(`(?:^|[\s<])data-i18n="([^"]+)"`)
	guideConfigKey  = regexp.MustCompile(`"i18n"\s*:\s*"([^"]+)"`)
	guideEntry      = regexp.MustCompile(`(?m)^\s*"([^"]+)":`)
	guideCommand    = regexp.MustCompile(`\./tasks-ai ([a-z][a-z-]*)`)
)

func readGuide(t *testing.T, path string) string {
	t.Helper()

	//nolint:gosec // la ruta es del propio repositorio, no de fuera
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}

	return string(raw)
}

// TestGuideTranslations fails when the guide and its English dictionary drift
// apart: a key with no translation, a translation the page does not use, or a
// key written twice, which the JSON keeps once and hides the duplicate.
func TestGuideTranslations(t *testing.T) {
	t.Parallel()

	page := readGuide(t, guidePath)
	script := readGuide(t, guideScriptPath)

	block := guideDictionary.FindStringSubmatch(page)
	if block == nil {
		t.Fatalf("%s: no #tasks-ai-en dictionary", guidePath)
	}

	var dictionary map[string]string

	if err := json.Unmarshal([]byte(block[1]), &dictionary); err != nil {
		t.Fatalf("%s: the dictionary does not parse: %v", guidePath, err)
	}

	written := map[string]bool{}

	for _, match := range guideEntry.FindAllStringSubmatch(block[1], -1) {
		if written[match[1]] {
			t.Errorf("%s: the dictionary repeats %q", guidePath, match[1])
		}

		written[match[1]] = true
	}

	used := map[string]bool{}

	for _, source := range []struct {
		path, content string
		pattern       *regexp.Regexp
	}{
		{guidePath, page, guideKey},
		{guideScriptPath, script, guideKey},
		{guidePath, page, guideConfigKey},
	} {
		for _, match := range source.pattern.FindAllStringSubmatch(source.content, -1) {
			used[match[1]] = true

			if _, ok := dictionary[match[1]]; !ok {
				t.Errorf("%s: %q has no English translation", source.path, match[1])
			}
		}
	}

	for key := range dictionary {
		if !used[key] {
			t.Errorf("%s: %q translates nothing", guidePath, key)
		}
	}
}

// TestGuideDocumentsEveryCommand fails when the guide and the command tree no
// longer agree: a command the binary offers and the guide never shows, or a
// command the guide invokes and the binary does not have.
func TestGuideDocumentsEveryCommand(t *testing.T) {
	t.Parallel()

	page := readGuide(t, guidePath)
	root := cli.New(io.Discard, io.Discard)

	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()

	known := map[string]bool{}

	for _, command := range root.Commands() {
		known[command.Name()] = true
	}

	for _, match := range guideCommand.FindAllStringSubmatch(page, -1) {
		if !known[match[1]] {
			t.Errorf("%s: %q is not a command of %s", guidePath, match[1], cli.Name)
		}
	}

	var walk func(*cobra.Command, []string)

	walk = func(command *cobra.Command, path []string) {
		if command.Name() == "completion" || command.Name() == "help" {
			return
		}

		if len(command.Commands()) == 0 {
			phrase := strings.Join(append([]string{cli.Name}, path...), " ")

			if !strings.Contains(page, phrase) {
				t.Errorf("%s: the guide does not show %q", guidePath, phrase)
			}

			return
		}

		for _, child := range command.Commands() {
			walk(child, append(slices.Clip(path), child.Name()))
		}
	}

	walk(root, nil)
}
