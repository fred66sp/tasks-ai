package install

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// hostRef is a configuration file a host reads and the exact element in one of
// its arrays that points at an installed extension.
type hostRef struct {
	path      string
	key       string
	element   string
	extension string
}

// The outcomes of looking at a host reference.
const (
	refRemoved = "removed"
	refMissing = "missing"
	refAbsent  = "absent"
	refBlocked = "blocked"
)

// removeHostRef removes element from the array key of the file at path,
// rewriting only that array and keeping the rest of the file. It returns one of
// the outcomes above. A file that is not there has nothing to remove, and an
// element that is not there exactly is left alone: a similar string never
// matches. A comment inside the array stops the edit, and the reference is
// reported as blocked.
func removeHostRef(path, key, element string) (string, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // la ruta la compone la instalación, no una entrada externa
	if errors.Is(err, fs.ErrNotExist) {
		return refAbsent, nil
	}

	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}

	text := string(raw)

	array, found := findArray(text, key)
	if !found {
		return refMissing, nil
	}

	if array.hasComment {
		return refBlocked, nil
	}

	kept := make([]string, 0, len(array.elements))
	removed := false

	for _, entry := range array.elements {
		if matchesRef(entry, element) {
			removed = true

			continue
		}

		kept = append(kept, entry)
	}

	if !removed {
		return refMissing, nil
	}

	updated := text[:array.start] + "[" + strings.Join(kept, ", ") + "]" + text[array.end+1:]

	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}

	if err := writeFileAtomico(path, []byte(updated), info.Mode().Perm()); err != nil {
		return "", fmt.Errorf("writing %s: %w", path, err)
	}

	return refRemoved, nil
}

// hasHostRef says whether the array key of the file at path still carries the
// exact element.
func hasHostRef(path, key, element string) (bool, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // la ruta la compone la instalación, no una entrada externa
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("reading %s: %w", path, err)
	}

	text := string(raw)

	array, found := findArray(text, key)
	if !found {
		// Un documento que no se puede leer no autoriza a borrar la extensión:
		// se conserva si el elemento aparece como cadena exacta en cualquier parte.
		return strings.Contains(text, `"`+element+`"`) || strings.Contains(text, `"`+element+`/"`), nil
	}

	for _, entry := range array.elements {
		if matchesRef(entry, element) {
			return true, nil
		}
	}

	return false, nil
}

// jsonArray is the portion of a JSONC document that belongs to one array: where
// it starts and ends, the raw text of each element and whether a comment lives
// inside it.
type jsonArray struct {
	start      int
	end        int
	elements   []string
	hasComment bool
}

// findArray looks for the first array whose key is key, skipping comments and
// strings elsewhere in the document.
func findArray(text, key string) (jsonArray, bool) {
	for i := 0; i < len(text); {
		switch {
		case text[i] == '"':
			literal, next, ok := readString(text, i)
			if !ok {
				return jsonArray{}, false
			}

			if literal == key {
				colon, _ := skipTrivia(text, next)
				if colon < len(text) && text[colon] == ':' {
					bracket, _ := skipTrivia(text, colon+1)
					if bracket < len(text) && text[bracket] == '[' {
						return readArray(text, bracket)
					}
				}
			}

			i = next
		case strings.HasPrefix(text[i:], "//"):
			i = lineEnd(text, i)
		case strings.HasPrefix(text[i:], "/*"):
			i = blockEnd(text, i)
		default:
			i++
		}
	}

	return jsonArray{}, false
}

// readArray reads the array that starts at start, up to its closing bracket.
func readArray(text string, start int) (jsonArray, bool) {
	array := jsonArray{start: start}

	i := start + 1

	for {
		next, comment := skipTrivia(text, i)
		array.hasComment = array.hasComment || comment
		i = next

		if i >= len(text) {
			return jsonArray{}, false
		}

		if text[i] == ']' {
			array.end = i

			return array, true
		}

		end, comment, ok := scanElement(text, i)
		array.hasComment = array.hasComment || comment

		if !ok {
			return jsonArray{}, false
		}

		array.elements = append(array.elements, strings.TrimSpace(text[i:end]))

		if end < len(text) && text[end] == ',' {
			i = end + 1

			continue
		}

		if end < len(text) && text[end] == ']' {
			array.end = end

			return array, true
		}

		return jsonArray{}, false
	}
}

// scanElement finds the end of the element that starts at start: the comma or
// the closing bracket at its own level, skipping strings, nested values and
// comments.
func scanElement(text string, start int) (int, bool, bool) {
	depth := 0
	comment := false

	for i := start; i < len(text); {
		switch {
		case text[i] == '"':
			_, next, ok := readString(text, i)
			if !ok {
				return 0, comment, false
			}

			i = next
		case strings.HasPrefix(text[i:], "//"):
			comment = true
			i = lineEnd(text, i)
		case strings.HasPrefix(text[i:], "/*"):
			comment = true
			i = blockEnd(text, i)
		case text[i] == '[' || text[i] == '{':
			depth++
			i++
		case text[i] == ']' || text[i] == '}':
			if depth == 0 {
				return i, comment, true
			}

			depth--
			i++
		case text[i] == ',' && depth == 0:
			return i, comment, true
		default:
			i++
		}
	}

	return 0, comment, false
}

// matchesRef says whether the raw element of an array is the string element, or
// that same string with a trailing slash, compared as a whole and trimmed.
func matchesRef(raw, element string) bool {
	literal, _, ok := readString(raw, 0)
	if !ok {
		return false
	}

	literal = strings.TrimSpace(literal)

	return literal == element || literal == element+"/"
}

// readString reads the string literal that starts at start and returns its
// content and the offset after its closing quote.
func readString(text string, start int) (string, int, bool) {
	if start >= len(text) || text[start] != '"' {
		return "", 0, false
	}

	var literal strings.Builder

	for i := start + 1; i < len(text); i++ {
		switch text[i] {
		case '\\':
			if i+1 >= len(text) {
				return "", 0, false
			}

			literal.WriteByte(text[i+1])
			i++
		case '"':
			return literal.String(), i + 1, true
		default:
			literal.WriteByte(text[i])
		}
	}

	return "", 0, false
}

// skipTrivia skips whitespace and comments, saying whether it found a comment.
func skipTrivia(text string, start int) (int, bool) {
	comment := false

	for i := start; i < len(text); {
		switch {
		case text[i] == ' ' || text[i] == '\t' || text[i] == '\n' || text[i] == '\r':
			i++
		case strings.HasPrefix(text[i:], "//"):
			comment = true
			i = lineEnd(text, i)
		case strings.HasPrefix(text[i:], "/*"):
			comment = true
			i = blockEnd(text, i)
		default:
			return i, comment
		}
	}

	return len(text), comment
}

// lineEnd returns the offset after the line that starts at start.
func lineEnd(text string, start int) int {
	if newline := strings.IndexByte(text[start:], '\n'); newline >= 0 {
		return start + newline + 1
	}

	return len(text)
}

// blockEnd returns the offset after the block comment that starts at start.
func blockEnd(text string, start int) int {
	if closing := strings.Index(text[start:], "*/"); closing >= 0 {
		return start + closing + 2
	}

	return len(text)
}
