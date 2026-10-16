package core

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"tasks-ai/internal/store"
)

// TagColors es la paleta que puede tomar el color de una etiqueta, tomada de la
// interfaz de referencia del proyecto. La lista vive aquí y en ningún otro
// sitio: los documentos dicen de dónde viene y no la copian.
var TagColors = []string{"blue", "purple", "green", "yellow", "orange", "red"}

// DefaultTagColor es el color que lleva una etiqueta a la que nadie le dio otro.
const DefaultTagColor = "blue"

// Tag es una etiqueta tal como cruza el límite del núcleo: su identidad interna,
// que mantiene el renombrado lejos de las tareas, y lo que de ella se muestra.
type Tag struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Color       string  `json:"color"`
	Description *string `json:"description"`
}

// TagWithUses es una etiqueta del catálogo con el número de tareas que la usan.
type TagWithUses struct {
	Tag
	Uses int `json:"uses"`
}

// TagChanges es lo que un cambio de etiqueta pide. Un campo que no se menciona
// no se toca, igual que en el cambio de una tarea.
type TagChanges struct {
	Name        *string
	Color       *string
	Description *string
}

// empty says whether a tag change asks for nothing at all.
func (changes TagChanges) empty() bool {
	return changes.Name == nil && changes.Color == nil && changes.Description == nil
}

// Tags devuelve el catálogo de etiquetas ordenado por nombre, cada una con el
// número de tareas que la usan.
func (c *Core) Tags(ctx context.Context) ([]TagWithUses, error) {
	stored, err := c.store.Tags(ctx)
	if err != nil {
		return nil, fmt.Errorf("leyendo el catálogo de etiquetas: %w", err)
	}

	catalogue := make([]TagWithUses, 0, len(stored))

	for i := range stored {
		catalogue = append(catalogue, TagWithUses{Tag: tag(stored[i].Tag), Uses: stored[i].Uses})
	}

	return catalogue, nil
}

// CreateTag guarda una etiqueta nueva en el catálogo y la devuelve. El color
// que falta es el de por omisión, un color que no está en la paleta es entrada
// inválida y un nombre que ya existe lo rechaza la regla que lo impone.
func (c *Core) CreateTag(ctx context.Context, name, color, description string) (Tag, error) {
	label, err := tagName(name)
	if err != nil {
		return Tag{}, err
	}

	palette, err := tagColor(color)
	if err != nil {
		return Tag{}, err
	}

	stored, err := c.store.CreateTag(ctx, store.Tag{
		Name:        label,
		Color:       palette,
		Description: optional(description),
	})
	switch {
	case errors.Is(err, store.ErrTagExists):
		return Tag{}, fmt.Errorf("ya existe una etiqueta con el nombre %q", label)
	case err != nil:
		return Tag{}, fmt.Errorf("creando la etiqueta %q: %w", label, err)
	}

	return tag(stored), nil
}

// EditTag cambia lo que el cambio pide y nada más. Una etiqueta que no existe
// se rechaza como una referencia que no resuelve, y renombrar sobre un nombre
// que ya lleva otra etiqueta lo rechaza la regla que lo impone.
func (c *Core) EditTag(ctx context.Context, name string, changes TagChanges) (Tag, error) {
	if changes.empty() {
		return Tag{}, invalid("no hay nada que cambiar: indica al menos un campo")
	}

	var (
		newName    string
		hasNewName bool
		newColor   string
	)

	if changes.Name != nil {
		label, err := tagName(*changes.Name)
		if err != nil {
			return Tag{}, err
		}

		newName, hasNewName = label, true
	}

	if changes.Color != nil {
		palette, err := tagColor(*changes.Color)
		if err != nil {
			return Tag{}, err
		}

		newColor = palette
	}

	stored, err := c.store.UpdateTag(ctx, strings.TrimSpace(name), func(stored store.Tag) (store.Tag, error) {
		if hasNewName {
			stored.Name = newName
		}

		if changes.Color != nil {
			stored.Color = newColor
		}

		if changes.Description != nil {
			stored.Description = optional(*changes.Description)
		}

		return stored, nil
	})
	switch {
	case errors.Is(err, store.ErrNotFound):
		return Tag{}, c.missingTag(name)
	case errors.Is(err, store.ErrTagExists):
		return Tag{}, fmt.Errorf("ya existe una etiqueta con el nombre %q", newName)
	case err != nil:
		return Tag{}, fmt.Errorf("cambiando la etiqueta %q: %w", name, err)
	}

	return tag(stored), nil
}

// DeleteTag borra la etiqueta con ese nombre, con sus asignaciones, y devuelve
// la etiqueta borrada y las referencias de las tareas que la usaban, para que
// quien llama avise de ello. Una etiqueta que no existe se rechaza como una
// referencia que no resuelve.
func (c *Core) DeleteTag(ctx context.Context, name string) (Tag, []string, error) {
	stored, users, err := c.store.DeleteTag(ctx, strings.TrimSpace(name))
	switch {
	case errors.Is(err, store.ErrNotFound):
		return Tag{}, nil, c.missingTag(name)
	case err != nil:
		return Tag{}, nil, fmt.Errorf("borrando la etiqueta %q: %w", name, err)
	}

	refs := make([]string, 0, len(users))

	for _, id := range users {
		refs = append(refs, Ref(c.prefix, id))
	}

	return tag(stored), refs, nil
}

// tagName recorta el nombre de una etiqueta y rechaza el que queda vacío.
func tagName(text string) (string, error) {
	name := strings.TrimSpace(text)
	if name == "" {
		return "", invalid("la etiqueta necesita un nombre")
	}

	return name, nil
}

// tagColor comprueba el color contra la paleta y da el de por omisión al que
// falta.
func tagColor(color string) (string, error) {
	if strings.TrimSpace(color) == "" {
		return DefaultTagColor, nil
	}

	if slices.Contains(TagColors, color) {
		return color, nil
	}

	return "", invalid("el color %q no es válido: usa uno de %s", color, strings.Join(TagColors, ", "))
}

// missingTag is the error of a tag name that points at no tag in the catalogue.
func (c *Core) missingTag(name string) error {
	return fmt.Errorf("no hay ninguna etiqueta con el nombre %q", strings.TrimSpace(name))
}

// tag puts a stored tag in the shape that crosses the boundary.
func tag(stored store.Tag) Tag {
	return Tag{
		ID:          stored.ID,
		Name:        stored.Name,
		Color:       stored.Color,
		Description: stored.Description,
	}
}
