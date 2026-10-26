package embedded

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// markdownLink matches [text](target) and leaves the target in group 1.
var markdownLink = regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+)\)`)

// installedTemplates holds the links whose target only exists after the
// installation copies the template into the project board.
var installedTemplates = map[string]string{
	"internal/install/agent.md|CONTRATO_ETIQUETAS.md": ".tasks-ai/CONTRATO_ETIQUETAS.md",
}

// TestProjectDocumentation fails when a project document has no title or a
// relative link that does not resolve. Copied skills are out of scope: they
// travel untoched from their origin.
func TestProjectDocumentation(t *testing.T) {
	t.Parallel()

	roots := []string{
		"README.md",
		"AGENTS.md",
		"CHANGELOG.md",
		"docs",
		".tasks-ai",
		".pi/agents",
		"internal/install",
	}

	for _, root := range roots {
		for _, path := range markdownFiles(t, root) {
			content := readFile(t, path)

			if !hasTitle(content) {
				t.Errorf("%s: no title", path)
			}

			checkLinks(t, path, content)
		}
	}
}

func markdownFiles(t *testing.T, root string) []string {
	t.Helper()

	info, err := os.Stat(root)
	if err != nil {
		t.Fatalf("%s: %v", root, err)
	}

	if !info.IsDir() {
		return []string{root}
	}

	var files []string

	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !entry.IsDir() && strings.HasSuffix(path, ".md") {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("%s: %v", root, err)
	}

	return files
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	//nolint:gosec // la ruta sale del recorrido del repositorio, no de fuera
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}

	return string(raw)
}

// hasTitle reports whether the document has a level-one heading outside its
// frontmatter.
func hasTitle(content string) bool {
	for line := range strings.SplitSeq(content, "\n") {
		if strings.HasPrefix(line, "# ") {
			return true
		}
	}

	return false
}

func checkLinks(t *testing.T, path, content string) {
	t.Helper()

	for _, match := range markdownLink.FindAllStringSubmatch(stripFences(content), -1) {
		target := match[1]

		if isExternal(target) {
			continue
		}

		if id := strings.IndexByte(target, '#'); id >= 0 {
			target = target[:id]
		}

		if target == "" {
			continue
		}

		resolved := filepath.Join(filepath.Dir(path), target)
		if path == "internal/install/agent.md" {
			if installed, ok := installedTemplates[path+"|"+target]; ok {
				resolved = installed
			}
		}

		if _, err := os.Stat(resolved); err != nil {
			t.Errorf("%s: link %q does not resolve", path, target)
		}
	}
}

// stripFences removes fenced code blocks so the examples inside them are not
// read as links.
func stripFences(content string) string {
	var out []string

	inFence := false

	for line := range strings.SplitSeq(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence

			continue
		}

		if !inFence {
			out = append(out, line)
		}
	}

	return strings.Join(out, "\n")
}

func isExternal(target string) bool {
	for _, prefix := range []string{"http://", "https://", "mailto:", "ftp://"} {
		if strings.HasPrefix(target, prefix) {
			return true
		}
	}

	return false
}
