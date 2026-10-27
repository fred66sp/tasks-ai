package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"tasks-ai/internal/core"
)

// adrFileName is the name an exported ADR has: its number, its slug and the
// extension. El índice del directorio no lleva número y no se lee como ADR.
var adrFileName = regexp.MustCompile(`^(\d{1,6})-([a-z0-9]+(?:-[a-z0-9]+)*)\.md$`)

// supersededLink reads the number of the ADR a superseded state names: el
// estado se escribe «superseded by [ADR-014](014-...md)».
var supersededLink = regexp.MustCompile(`\((\d+)-`)

// parseADRDir reads every ADR of a directory of markdown files, by number.
func parseADRDir(dir string) ([]core.NewADR, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, &usageError{err: fmt.Errorf("reading the directory of ADR files %s: %w", dir, err)}
	}

	adrs := []core.NewADR{}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		match := adrFileName.FindStringSubmatch(entry.Name())
		if match == nil {
			if isADRIndex(entry.Name()) {
				continue
			}

			return nil, &usageError{err: fmt.Errorf(
				"%s is neither an index nor an ADR file: an ADR is named NNN-slug.md", entry.Name())}
		}

		adr, err := parseADRFile(filepath.Join(dir, entry.Name()), match)
		if err != nil {
			return nil, err
		}

		adrs = append(adrs, adr)
	}

	sort.Slice(adrs, func(i, j int) bool { return adrs[i].Number < adrs[j].Number })

	return adrs, nil
}

// isADRIndex says whether the file is the index of the directory.
func isADRIndex(name string) bool {
	return name == "README.md" || name == "index.md"
}

// parseADRFile reads one exported ADR: su frontmatter, su título y sus
// secciones. Lo que no encaja se rechaza nombrando el archivo: una sección que
// el registro no conoce no puede perderse en silencio.
func parseADRFile(path string, name []string) (core.NewADR, error) {
	//nolint:gosec // la ruta sale del directorio que la persona nombró
	raw, err := os.ReadFile(path)
	if err != nil {
		return core.NewADR{}, fmt.Errorf("reading %s: %w", path, err)
	}

	number, err := strconv.Atoi(name[1])
	if err != nil {
		return core.NewADR{}, fmt.Errorf("%s: the number %q cannot be read: %w", path, name[1], err)
	}

	adr := core.NewADR{Number: number, Slug: name[2]}

	front, body, err := splitFrontmatter(string(raw))
	if err != nil {
		return core.NewADR{}, fmt.Errorf("%s: %w", path, err)
	}

	adr.Status = front["status"]
	adr.Date = front["date"]
	adr.Deciders = strings.Trim(front["decision-makers"], `"'`)

	if target, err := supersededTarget(adr.Status); err != nil {
		return core.NewADR{}, fmt.Errorf("%s: %w", path, err)
	} else if target > 0 {
		adr.SupersededBy = target
		adr.Status = "superseded"
	}

	if err := parseADRBody(&adr, body); err != nil {
		return core.NewADR{}, fmt.Errorf("%s: %w", path, err)
	}

	return adr, nil
}

// supersededTarget reads the number of the ADR a superseded state names, 0 when
// the state is not a substitution.
func supersededTarget(status string) (int, error) {
	if !strings.HasPrefix(strings.TrimSpace(status), "superseded") {
		return 0, nil
	}

	match := supersededLink.FindStringSubmatch(status)
	if match == nil {
		return 0, fmt.Errorf("the state %q names no substitute", status)
	}

	number, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, fmt.Errorf("the substitute in %q cannot be read: %w", status, err)
	}

	return number, nil
}

// parseADRBody reads the title and the sections of an ADR. Las secciones que el
// registro conoce son las de su plantilla; cualquier otra se rechaza.
func parseADRBody(adr *core.NewADR, body string) error {
	section := ""
	sections := map[string]*strings.Builder{}

	for line := range strings.SplitSeq(body, "\n") {
		switch {
		case strings.HasPrefix(line, "## "):
			section = sectionField(strings.TrimSpace(strings.TrimPrefix(line, "## ")))
			if section == "" {
				return fmt.Errorf("the section %q is not one the register knows", strings.TrimSpace(line))
			}

			sections[section] = &strings.Builder{}
		case strings.HasPrefix(line, "# ") && adr.Title == "":
			adr.Title = adrTitle(strings.TrimSpace(strings.TrimPrefix(line, "# ")))
		default:
			if builder, ok := sections[section]; ok {
				builder.WriteString(line)
				builder.WriteString("\n")
			}
		}
	}

	if adr.Title == "" {
		return errors.New("the ADR has no title")
	}

	text := func(field string) string {
		builder, ok := sections[field]
		if !ok {
			return ""
		}

		return strings.TrimSpace(builder.String())
	}

	adr.Context = text("context")
	adr.Decision = text("decision")
	adr.Consequences = text("consequences")
	adr.Alternatives = text("alternatives")
	adr.MoreInfo = text("more-info")

	return nil
}

// sectionField names the field a heading fills, empty when the register has no
// field for it.
func sectionField(heading string) string {
	switch strings.ToLower(heading) {
	case "contexto", "contexto y problema":
		return "context"
	case "decisión", "decision":
		return "decision"
	case "consecuencias":
		return "consequences"
	case "alternativas", "alternativas consideradas", "alternativas descartadas":
		return "alternatives"
	case "más información", "mas información", "mas informacion":
		return "more-info"
	default:
		return ""
	}
}

// adrTitle reads the title of an ADR, which the H1 carries after its reference.
func adrTitle(heading string) string {
	text := heading
	if _, after, found := strings.Cut(text, ":"); found && strings.HasPrefix(strings.ToUpper(text), core.ADRRefPrefix) {
		text = after
	}

	return strings.TrimSpace(text)
}

// splitFrontmatter separates the frontmatter of a markdown file from its body.
// El archivo del registro empieza por «---» y la cierra otra línea igual.
func splitFrontmatter(text string) (map[string]string, string, error) {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, "", errors.New("the ADR has no frontmatter")
	}

	front := map[string]string{}

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return front, strings.Join(lines[i+1:], "\n"), nil
		}

		key, value, found := strings.Cut(lines[i], ":")
		if !found {
			return nil, "", fmt.Errorf("the frontmatter line %q is not a pair", lines[i])
		}

		front[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	return nil, "", errors.New("the frontmatter is not closed")
}
