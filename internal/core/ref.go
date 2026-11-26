// Package core holds the product's rules and is the only path to its data.
package core

import (
	"strconv"
	"strings"
)

// Ref composes the visible reference of a task: the project's prefix and its
// internal number.
func Ref(prefix string, id int64) string {
	return prefix + "-" + strconv.FormatInt(id, 10)
}

// ParseRef returns the internal number the reference points at. Surrounding
// spaces are ignored and case does not matter. Una referencia mal escrita no
// resuelve a nada guardado: sale con el centinela de referencia sin resolución
// y el mismo mensaje de siempre.
func ParseRef(prefix, ref string) (int64, error) {
	head := strings.TrimSpace(prefix) + "-"
	written := strings.TrimSpace(ref)

	if len(written) <= len(head) || !strings.EqualFold(written[:len(head)], head) {
		return 0, notFoundField(fieldRef, "reference %q is not valid: %s<number> is expected", ref, head)
	}

	id, err := strconv.ParseInt(written[len(head):], 10, 64)
	if err != nil {
		return 0, notFoundField(fieldRef, "reference %q is not valid: %s<number> is expected", ref, head)
	}

	return id, nil
}
