package core

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"tasks-ai/internal/store"
	"time"
)

// ADRRefPrefix starts every ADR reference: el número del registro, con tres
// dígitos, sin depender del prefijo del proyecto.
const ADRRefPrefix = "ADR-"

// adrStatuses lists the states an ADR may take, in the order the contract
// names them.
var adrStatuses = []string{
	store.ADRProposed,
	store.ADRAccepted,
	store.ADRRejected,
	store.ADRDeprecated,
	store.ADRSuperseded,
}

// adrDateLayout is how the day of an ADR travels: the one its frontmatter and
// its exported file use.
const adrDateLayout = "2006-01-02"

// ADR is a decision as it crosses the core's boundary: its number, the
// reference composed from it and the body by sections.
type ADR struct {
	Number       int       `json:"number"`
	Ref          string    `json:"ref"`
	Slug         string    `json:"slug"`
	Title        string    `json:"title"`
	Status       string    `json:"status"`
	Date         string    `json:"date"`
	Deciders     string    `json:"deciders"`
	Context      string    `json:"context"`
	Decision     string    `json:"decision"`
	Consequences string    `json:"consequences"`
	Alternatives *string   `json:"alternatives"`
	MoreInfo     *string   `json:"more_info"`
	SupersededBy *string   `json:"superseded_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ADRSummary is an ADR as the list shows it: what a list entry needs, with the
// context because the interface folds the entry down to three lines of it, and
// without the rest of the body.
type ADRSummary struct {
	Number  int    `json:"number"`
	Ref     string `json:"ref"`
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Date    string `json:"date"`
	Context string `json:"context"`
}

// ADRRef composes the reference of an ADR: ADR-016.
func ADRRef(number int) string {
	return fmt.Sprintf("%s%03d", ADRRefPrefix, number)
}

// ADRs lists the ADR of the project by number, each one with its context and
// without the rest of the body. Con una búsqueda, solo los que la contienen en
// alguno de sus campos; el texto vacío no filtra nada.
func (c *Core) ADRs(ctx context.Context, search string) ([]ADRSummary, error) {
	stored, err := c.store.ADRs(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading the ADRs: %w", err)
	}

	query := searchable(strings.TrimSpace(search))
	summaries := make([]ADRSummary, 0, len(stored))

	for i := range stored {
		if !adrMatches(stored[i], query) {
			continue
		}

		summary, err := summarizeADR(stored[i])
		if err != nil {
			return nil, err
		}

		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// adrMatches says whether the ADR holds the search in any of its fields. La
// búsqueda llega ya normalizada, y el número y el sustituto se buscan por su
// referencia visible.
func adrMatches(adr store.ADR, search string) bool {
	if search == "" {
		return true
	}

	fields := []string{
		strconv.Itoa(adr.Number),
		ADRRef(adr.Number),
		adr.Slug,
		adr.Title,
		adr.Status,
		adr.Date,
		adr.Deciders,
		adr.Context,
		adr.Decision,
		adr.Consequences,
		textOf(adr.Alternatives),
		textOf(adr.MoreInfo),
	}

	if adr.SupersededBy != nil {
		fields = append(fields, ADRRef(*adr.SupersededBy))
	}

	return strings.Contains(searchable(strings.Join(fields, "\n")), search)
}

// searchable lowers a text and drops its accents, so a search for "decision"
// finds "decisión".
func searchable(text string) string {
	return accents.Replace(strings.ToLower(text))
}

// textOf reads the text an optional column holds.
func textOf(text *string) string {
	if text == nil {
		return ""
	}

	return *text
}

// AllADRs returns every ADR with its body: es lo que la exportación escribe.
func (c *Core) AllADRs(ctx context.Context) ([]ADR, error) {
	stored, err := c.store.ADRs(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading the ADRs: %w", err)
	}

	adrs := make([]ADR, 0, len(stored))

	for i := range stored {
		adr, err := c.shapedADR(stored[i])
		if err != nil {
			return nil, err
		}

		adrs = append(adrs, adr)
	}

	return adrs, nil
}

// ADR returns the ADR the reference points at. El número se escribe suelto o
// con la referencia completa.
func (c *Core) ADR(ctx context.Context, ref string) (ADR, error) {
	number, err := ParseADRRef(ref)
	if err != nil {
		return ADR{}, err
	}

	stored, err := c.store.ADR(ctx, number)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return ADR{}, missingADR(ref)
		}

		return ADR{}, fmt.Errorf("reading ADR %q: %w", ref, err)
	}

	return c.shapedADR(stored)
}

// NewADR is what it takes to create an ADR. El número vacío pide el siguiente,
// que es la creación normal; la importación trae el suyo y su slug, y la fecha
// del archivo.
type NewADR struct {
	Number       int
	Slug         string
	Title        string
	Status       string
	Date         string
	Deciders     string
	Context      string
	Decision     string
	Consequences string
	Alternatives string
	MoreInfo     string
	SupersededBy int
}

// CreateADR stores a new ADR and returns it as it crosses the boundary.
func (c *Core) CreateADR(ctx context.Context, adr NewADR) (ADR, error) {
	stored, err := newADR(adr)
	if err != nil {
		return ADR{}, err
	}

	if err := c.checkSubstitute(ctx, stored.Number, stored.Status, stored.SupersededBy); err != nil {
		return ADR{}, err
	}

	saved, err := c.store.CreateADR(ctx, stored)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrADRNumberTaken):
			return ADR{}, invalid("ADR %s is already in the database", ADRRef(stored.Number))
		case errors.Is(err, store.ErrADRSlugTaken):
			return ADR{}, invalid("slug %q already names another ADR: change the title", stored.Slug)
		default:
			return ADR{}, fmt.Errorf("saving the ADR: %w", err)
		}
	}

	return c.shapedADR(saved)
}

// newADR checks a creation and turns it into what the store writes. La fecha que
// falta es la del día, el estado que falta es proposed y el slug sale del
// título cuando no viene dado.
func newADR(adr NewADR) (store.ADR, error) {
	title := strings.TrimSpace(adr.Title)
	if title == "" {
		return store.ADR{}, invalid("the ADR needs a title: use --title")
	}

	deciders := strings.TrimSpace(adr.Deciders)
	if deciders == "" {
		return store.ADR{}, invalid("the ADR needs the people who decide it: use --deciders")
	}

	status := strings.TrimSpace(adr.Status)
	if status == "" {
		status = store.ADRProposed
	}

	if !slices.Contains(adrStatuses, status) {
		return store.ADR{}, invalid("status %q is not understood: use one of %s", status, strings.Join(adrStatuses, ", "))
	}

	date, err := adrDate(adr.Date)
	if err != nil {
		return store.ADR{}, err
	}

	slug := strings.TrimSpace(adr.Slug)
	if slug == "" {
		slug = slugOf(title)
	}

	if !validSlug(slug) {
		return store.ADR{}, invalid("slug %q cannot name a file: use lowercase letters, digits and single dashes", slug)
	}

	if adr.Number < 0 {
		return store.ADR{}, invalid("the ADR number %d cannot exist", adr.Number)
	}

	return store.ADR{
		Number:       adr.Number,
		Slug:         slug,
		Title:        title,
		Status:       status,
		Date:         date,
		Deciders:     deciders,
		Context:      strings.TrimSpace(adr.Context),
		Decision:     strings.TrimSpace(adr.Decision),
		Consequences: strings.TrimSpace(adr.Consequences),
		Alternatives: optional(adr.Alternatives),
		MoreInfo:     optional(adr.MoreInfo),
		SupersededBy: positive(adr.SupersededBy),
	}, nil
}

// ADRChanges is what an edition asks to change. A field left out is not
// touched. Note appends a dated entry to «Más información»; es la única
// escritura que admite el cuerpo de un ADR que ya no está en proposed.
type ADRChanges struct {
	Title        *string
	Status       *string
	SupersededBy *int
	Context      *string
	Decision     *string
	Consequences *string
	Alternatives *string
	Note         *string
}

// UpdateADR changes what the edition asks for and returns the ADR as it is
// left. El cuerpo se congela al salir de proposed, y las reglas se comprueban
// antes de la escritura: un rechazo no deja nada escrito.
func (c *Core) UpdateADR(ctx context.Context, ref string, changes ADRChanges) (ADR, error) {
	if changes.empty() {
		return ADR{}, invalid("there is nothing to change: name at least one field")
	}

	number, err := ParseADRRef(ref)
	if err != nil {
		return ADR{}, err
	}

	stored, err := c.store.ADR(ctx, number)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return ADR{}, missingADR(ref)
		}

		return ADR{}, fmt.Errorf("reading ADR %q: %w", ref, err)
	}

	today := time.Now().UTC().Format(adrDateLayout)

	// El resultado se calcula antes de escribir: la cadena de sustituciones se
	// recorre sobre lo que el cambio dejaría, no sobre lo que había.
	next, err := changedADR(stored, changes, today)
	if err != nil {
		return ADR{}, err
	}

	if err := c.checkSubstitute(ctx, next.Number, next.Status, next.SupersededBy); err != nil {
		return ADR{}, err
	}

	saved, err := c.store.UpdateADR(ctx, number, func(adr store.ADR) (store.ADR, error) {
		return changedADR(adr, changes, today)
	})
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return ADR{}, missingADR(ref)
		}

		if errors.Is(err, ErrInvalid) || errors.Is(err, ErrConflict) {
			return ADR{}, err
		}

		return ADR{}, fmt.Errorf("changing ADR %q: %w", ref, err)
	}

	return c.shapedADR(saved)
}

// changedADR applies the edition inside the transaction that holds the write
// lock. It must not ask the base for anything. La sustitución se comprueba
// antes de escribir, contra las filas que hay.
func changedADR(adr store.ADR, changes ADRChanges, today string) (store.ADR, error) {
	// El sustituto solo significa algo junto al estado que lo usa: sin él, o con
	// otro estado, el cambio no tocaría el sustituto y saldría como si lo
	// hubiera hecho, con aviso y fecha de cambio.
	if changes.SupersededBy != nil && (changes.Status == nil || *changes.Status != store.ADRSuperseded) {
		return store.ADR{}, invalid("a substitute needs the state: use --status superseded")
	}

	if adr.Status != store.ADRProposed && changes.editsBody() {
		return store.ADR{}, invalid(
			"the body of %s is frozen: once it leaves proposed, only more information accepts additions",
			ADRRef(adr.Number),
		)
	}

	var err error

	if adr, err = changeADRBody(adr, changes); err != nil {
		return store.ADR{}, err
	}

	if changes.Note != nil {
		if adr, err = addADRNote(adr, *changes.Note, today); err != nil {
			return store.ADR{}, err
		}
	}

	if changes.Status != nil {
		if err := applyADRStatus(&adr, changes, today); err != nil {
			return store.ADR{}, err
		}
	}

	return adr, nil
}

// changeADRBody applies the sections an edition names.
func changeADRBody(adr store.ADR, changes ADRChanges) (store.ADR, error) {
	if changes.Title != nil {
		title := strings.TrimSpace(*changes.Title)
		if title == "" {
			return store.ADR{}, invalid("the ADR needs a title: use --title")
		}

		adr.Title = title
	}

	if changes.Context != nil {
		adr.Context = strings.TrimSpace(*changes.Context)
	}

	if changes.Decision != nil {
		adr.Decision = strings.TrimSpace(*changes.Decision)
	}

	if changes.Consequences != nil {
		adr.Consequences = strings.TrimSpace(*changes.Consequences)
	}

	if changes.Alternatives != nil {
		adr.Alternatives = optional(*changes.Alternatives)
	}

	return adr, nil
}

// addADRNote appends a dated entry to «Más información».
func addADRNote(adr store.ADR, note, today string) (store.ADR, error) {
	text := strings.TrimSpace(note)
	if text == "" {
		return store.ADR{}, invalid("the note is empty: use --note with the addition")
	}

	adr.MoreInfo = noted(adr.MoreInfo, text, today)

	return adr, nil
}

// applyADRStatus applies the status an edition asks for, under the rules of a
// substitution: un ADR sustituido nombra a su sustituto, y el día del registro
// cambia solo cuando el estado cambia.
func applyADRStatus(adr *store.ADR, changes ADRChanges, today string) error {
	status := strings.TrimSpace(*changes.Status)
	if !slices.Contains(adrStatuses, status) {
		return invalid("status %q is not understood: use one of %s", status, strings.Join(adrStatuses, ", "))
	}

	if status != adr.Status {
		adr.Date = today
	}

	if status != store.ADRSuperseded {
		adr.SupersededBy = nil
		adr.Status = status

		return nil
	}

	// Sustituir exige nombrar el sustituto; si ya estaba sustituido y el cambio
	// no nombra otro, conserva el que tenía.
	if changes.SupersededBy == nil {
		if adr.SupersededBy == nil {
			return invalid("a superseded ADR names its substitute: use --superseded-by with its number")
		}

		adr.Status = status

		return nil
	}

	if *changes.SupersededBy == adr.Number {
		return invalid("ADR %s cannot supersede itself", ADRRef(adr.Number))
	}

	adr.SupersededBy = positive(*changes.SupersededBy)
	adr.Status = status

	return nil
}

// empty says whether an edition asks for nothing at all.
func (changes ADRChanges) empty() bool {
	return changes.Title == nil &&
		changes.Status == nil &&
		changes.SupersededBy == nil &&
		changes.Context == nil &&
		changes.Decision == nil &&
		changes.Consequences == nil &&
		changes.Alternatives == nil &&
		changes.Note == nil
}

// editsBody says whether an edition names a section of the body, which is what
// a frozen ADR does not accept.
func (changes ADRChanges) editsBody() bool {
	return changes.Title != nil ||
		changes.Context != nil ||
		changes.Decision != nil ||
		changes.Consequences != nil ||
		changes.Alternatives != nil
}

// checkSubstitute holds the rules of a substitution: the substitute exists, and
// the chain of substitutions does not come back to the ADR that starts it.
func (c *Core) checkSubstitute(ctx context.Context, number int, status string, substitute *int) error {
	if status != store.ADRSuperseded {
		return nil
	}

	if substitute == nil {
		return invalid("a superseded ADR names its substitute")
	}

	seen := map[int]bool{number: true}
	current := *substitute

	for current != 0 {
		if seen[current] {
			return invalid("the substitution would form a cycle: ADR %s cannot end up superseding itself", ADRRef(number))
		}

		seen[current] = true

		stored, err := c.store.ADR(ctx, current)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return notFound("there is no ADR %s to supersede it", ADRRef(current))
			}

			return fmt.Errorf("looking up ADR %s: %w", ADRRef(current), err)
		}

		if stored.SupersededBy == nil {
			return nil
		}

		current = *stored.SupersededBy
	}

	return nil
}

// ImportResult is what an import did: the ADR it stored and the ones that were
// already in the base.
type ImportResult struct {
	Imported []string `json:"imported"`
	Skipped  []string `json:"skipped"`
}

// ImportADRs stores the ADR that are not in the base yet, by number, and
// returns which ones it wrote. Es idempotente: un número que ya está se salta
// sin reescribirlo. Un ADR sustituido espera a que su sustituto esté guardado,
// así que la pasada se repite mientras avance.
func (c *Core) ImportADRs(ctx context.Context, adrs []NewADR) (ImportResult, error) {
	result := ImportResult{Imported: []string{}, Skipped: []string{}}
	pending := make([]NewADR, 0, len(adrs))

	for _, adr := range adrs {
		if adr.Number > 0 {
			if _, err := c.store.ADR(ctx, adr.Number); err == nil {
				result.Skipped = append(result.Skipped, ADRRef(adr.Number))

				continue
			} else if !errors.Is(err, store.ErrNotFound) {
				return ImportResult{}, fmt.Errorf("looking up ADR %s: %w", ADRRef(adr.Number), err)
			}
		}

		pending = append(pending, adr)
	}

	for len(pending) > 0 {
		waiting := make([]NewADR, 0, len(pending))

		for _, adr := range pending {
			_, err := c.CreateADR(ctx, adr)

			switch {
			case err == nil:
				result.Imported = append(result.Imported, ADRRef(adr.Number))
			case errors.Is(err, ErrNotFound):
				// El sustituto aún no está: se reintenta en la pasada siguiente.
				waiting = append(waiting, adr)
			default:
				return ImportResult{}, err
			}
		}

		if len(waiting) == len(pending) {
			refs := make([]string, 0, len(waiting))
			for _, adr := range waiting {
				refs = append(refs, ADRRef(adr.Number))
			}

			return ImportResult{}, invalid("these ADR name a substitute that does not exist: %s", strings.Join(refs, ", "))
		}

		pending = waiting
	}

	return result, nil
}

// shapedADR puts a stored ADR in the shape that crosses the boundary.
func (c *Core) shapedADR(stored store.ADR) (ADR, error) {
	summary, err := summarizeADR(stored)
	if err != nil {
		return ADR{}, err
	}

	adr := ADR{
		Number:       summary.Number,
		Ref:          summary.Ref,
		Slug:         summary.Slug,
		Title:        summary.Title,
		Status:       summary.Status,
		Date:         summary.Date,
		Deciders:     stored.Deciders,
		Context:      stored.Context,
		Decision:     stored.Decision,
		Consequences: stored.Consequences,
		Alternatives: stored.Alternatives,
		MoreInfo:     stored.MoreInfo,
		CreatedAt:    stored.CreatedAt.UTC(),
		UpdatedAt:    stored.UpdatedAt.UTC(),
	}

	if stored.SupersededBy != nil {
		ref := ADRRef(*stored.SupersededBy)
		adr.SupersededBy = &ref
	}

	return adr, nil
}

// summarizeADR puts one stored ADR in the shape of a list entry.
func summarizeADR(stored store.ADR) (ADRSummary, error) {
	if stored.Number < 1 {
		return ADRSummary{}, fmt.Errorf("the database holds an ADR with number %d", stored.Number)
	}

	return ADRSummary{
		Number:  stored.Number,
		Ref:     ADRRef(stored.Number),
		Slug:    stored.Slug,
		Title:   stored.Title,
		Status:  stored.Status,
		Date:    stored.Date,
		Context: stored.Context,
	}, nil
}

// ParseADRRef reads the reference of an ADR: su número suelto o la referencia
// completa, sin distinguir mayúsculas.
func ParseADRRef(ref string) (int, error) {
	text := strings.TrimSpace(ref)
	text = strings.TrimPrefix(strings.ToUpper(text), ADRRefPrefix)

	number, err := strconv.Atoi(text)
	if err != nil || number < 1 {
		return 0, invalid("%q is not an ADR reference: use its number or ADR-NNN", strings.TrimSpace(ref))
	}

	return number, nil
}

// missingADR is the error of a reference that points at no ADR.
func missingADR(ref string) error {
	return notFound("there is no ADR with reference %q", strings.TrimSpace(ref))
}

// adrDate reads the day of an ADR, the one of today when it comes empty.
func adrDate(date string) (string, error) {
	text := strings.TrimSpace(date)
	if text == "" {
		return time.Now().UTC().Format(adrDateLayout), nil
	}

	if _, err := time.Parse(adrDateLayout, text); err != nil {
		return "", invalid("date %q is not a day: use AAAA-MM-DD", text)
	}

	return text, nil
}

// noted appends a dated entry to «Más información», which is the only section
// that accepts writing once the ADR left proposed.
func noted(moreInfo *string, note, date string) *string {
	entry := fmt.Sprintf("**%s.** %s", date, note)

	if moreInfo == nil || strings.TrimSpace(*moreInfo) == "" {
		return &entry
	}

	joined := strings.TrimRight(*moreInfo, "\n") + "\n\n" + entry

	return &joined
}

// positive turns a number that may be absent into the pointer the base stores.
func positive(number int) *int {
	if number < 1 {
		return nil
	}

	return &number
}

// accents maps the letters a Spanish title may carry to the ones a file name
// keeps.
var accents = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
)

// slugOf turns a title into the immutable name of the exported file: minúsculas,
// sin acentos, palabras separadas por un guion.
func slugOf(title string) string {
	var out strings.Builder

	dashed := false

	for _, letter := range accents.Replace(strings.ToLower(title)) {
		switch {
		case letter >= 'a' && letter <= 'z', letter >= '0' && letter <= '9':
			out.WriteRune(letter)

			dashed = false
		case out.Len() > 0 && !dashed:
			out.WriteByte('-')

			dashed = true
		}
	}

	return strings.TrimSuffix(out.String(), "-")
}

// validSlug says whether a slug can name a file: minúsculas, dígitos y guiones
// sueltos, sin empezar ni acabar en guion.
func validSlug(slug string) bool {
	if slug == "" || strings.HasPrefix(slug, "-") || strings.HasSuffix(slug, "-") {
		return false
	}

	for _, letter := range slug {
		if letter != '-' && (letter < 'a' || letter > 'z') && (letter < '0' || letter > '9') {
			return false
		}
	}

	return true
}
