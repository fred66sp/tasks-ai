// Package core holds the product's rules and is the only path to its data.
package core

import (
	"fmt"
	"strconv"
	"strings"
)

// Ref composes the visible reference of a task: the project's prefix and its
// internal number.
func Ref(prefix string, id int64) string {
	return prefix + "-" + strconv.FormatInt(id, 10)
}

// ParseRef returns the internal number the reference points at. Surrounding
// spaces are ignored and case does not matter.
func ParseRef(prefix, ref string) (int64, error) {
	head := strings.TrimSpace(prefix) + "-"
	written := strings.TrimSpace(ref)

	if len(written) <= len(head) || !strings.EqualFold(written[:len(head)], head) {
		return 0, fmt.Errorf("la referencia %q no es válida: se espera %s<número>", ref, head)
	}

	id, err := strconv.ParseInt(written[len(head):], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("la referencia %q no es válida: se espera %s<número>", ref, head)
	}

	return id, nil
}
