package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sqlite3 "modernc.org/sqlite/lib"
)

// Los estados que un ADR puede tener.
const (
	ADRProposed   = "proposed"
	ADRAccepted   = "accepted"
	ADRRejected   = "rejected"
	ADRDeprecated = "deprecated"
	ADRSuperseded = "superseded"
)

// ErrADRNumberTaken dice que el número con el que se quiso guardar un ADR ya
// está en la base: solo lo pide la importación, que conserva el número del
// archivo.
var ErrADRNumberTaken = errors.New("that ADR number is already in the database")

// ErrADRSlugTaken dice que el slug ya nombra otro ADR de la base: el slug es el
// nombre del archivo exportado, así que dos ADR no pueden compartirlo.
var ErrADRSlugTaken = errors.New("that ADR slug is already in the database")

// ADR es una decisión durable tal como está guardada. El sustituto viaja por su
// número, no por la identidad interna: es el número lo que la referencia
// visible nombra, y el que el núcleo comprueba.
type ADR struct {
	ID           int64
	Number       int
	Slug         string
	Title        string
	Status       string
	Date         string
	Deciders     string
	Context      string
	Decision     string
	Consequences string
	Alternatives *string
	MoreInfo     *string
	SupersededBy *int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// adrColumns is the column list every read of ADRs asks for, in the order the
// scan expects them. El sustituto sale por su número, resuelto en la misma
// consulta: quien lee un ADR no pregunta después por su relación.
const adrColumns = `a.id, a.number, a.slug, a.title, a.status, a.status_date, a.deciders,
	a.context, a.decision, a.consequences, a.alternatives, a.more_info, s.number,
	a.created_at, a.updated_at`

// ADRs returns every ADR of the project, by its number.
func (s *Store) ADRs(ctx context.Context) ([]ADR, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+adrColumns+` FROM adrs a LEFT JOIN adrs s ON s.id = a.superseded_by ORDER BY a.number`)
	if err != nil {
		return nil, fmt.Errorf("reading the ADRs: %w", err)
	}

	defer func() { _ = rows.Close() }()

	adrs := []ADR{}

	for rows.Next() {
		adr, err := scanADR(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("reading the ADRs: %w", err)
		}

		adrs = append(adrs, adr)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading the ADRs: %w", err)
	}

	return adrs, nil
}

// ADR returns the ADR with that number.
func (s *Store) ADR(ctx context.Context, number int) (ADR, error) {
	return readADR(ctx, s.db, number)
}

// ADRHeader es lo que un aviso necesita para nombrar un ADR: su número y su
// título. Viaja por la identidad interna del aviso, que es la de la fila.
type ADRHeader struct {
	Number int
	Title  string
}

// ADRHeaders devuelve, en una sola consulta, el número y el título que llevan
// ahora los ADR pedidos por su identidad interna: es lo que un lote de avisos
// necesita para nombrarlos, no una pregunta por aviso.
//
//nolint:gosec // la consulta solo añade marcadores, nunca datos
func (s *Store) ADRHeaders(ctx context.Context, ids []int64) (map[int64]ADRHeader, error) {
	headers := make(map[int64]ADRHeader, len(ids))
	if len(ids) == 0 {
		return headers, nil
	}

	query := `SELECT id, number, title FROM adrs WHERE id IN (` + inPlaceholders(len(ids)) + `)`

	rows, err := s.db.QueryContext(ctx, query, int64Args(ids)...)
	if err != nil {
		return nil, fmt.Errorf("reading the headers of %d ADRs: %w", len(ids), err)
	}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			id     int64
			header ADRHeader
		)

		if err := rows.Scan(&id, &header.Number, &header.Title); err != nil {
			return nil, fmt.Errorf("reading the headers of %d ADRs: %w", len(ids), err)
		}

		headers[id] = header
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading the headers of %d ADRs: %w", len(ids), err)
	}

	return headers, nil
}

// readADR reads one ADR from wherever it is asked for.
func readADR(ctx context.Context, from rowQuerier, number int) (ADR, error) {
	adr, err := scanADR(func(dest ...any) error {
		return from.QueryRowContext(ctx,
			`SELECT `+adrColumns+` FROM adrs a LEFT JOIN adrs s ON s.id = a.superseded_by WHERE a.number = ?`,
			number).Scan(dest...)
	})
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return ADR{}, ErrNotFound
	case err != nil:
		return ADR{}, fmt.Errorf("reading ADR %d: %w", number, err)
	}

	return adr, nil
}

// scanADR reads one row of the ADR columns and understands its moments, which
// travel as text. ErrNoRows passes through for whoever reads a single row.
func scanADR(scan func(dest ...any) error) (ADR, error) {
	var (
		adr                ADR
		createdAt, updated string
	)

	err := scan(&adr.ID, &adr.Number, &adr.Slug, &adr.Title, &adr.Status, &adr.Date, &adr.Deciders,
		&adr.Context, &adr.Decision, &adr.Consequences, &adr.Alternatives, &adr.MoreInfo,
		&adr.SupersededBy, &createdAt, &updated)
	if err != nil {
		return ADR{}, err
	}

	if adr.CreatedAt, err = time.Parse(timeLayout, createdAt); err != nil {
		return ADR{}, fmt.Errorf("the creation date of ADR %d cannot be read: %w", adr.Number, err)
	}

	if adr.UpdatedAt, err = time.Parse(timeLayout, updated); err != nil {
		return ADR{}, fmt.Errorf("the change date of ADR %d cannot be read: %w", adr.Number, err)
	}

	return adr, nil
}

// CreateADR stores a new ADR and returns it with the number the base gave it:
// the next one when the ADR arrives without a number, which is the normal
// creation, and the one it carries when it comes from an import. El número se
// lee y se escribe dentro de la misma transacción, que toma el bloqueo de
// escritura al empezar: dos creaciones a la vez no comparten número. Un número
// ya usado se rechaza con ErrADRNumberTaken.
func (s *Store) CreateADR(ctx context.Context, adr ADR) (ADR, error) {
	now := time.Now().UTC().Truncate(time.Second)
	adr.CreatedAt, adr.UpdatedAt = now, now

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ADR{}, fmt.Errorf("opening the creation transaction: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	if adr.Number == 0 {
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(number), 0) + 1 FROM adrs`).Scan(&adr.Number); err != nil {
			return ADR{}, fmt.Errorf("reading the next ADR number: %w", err)
		}
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO adrs (number, slug, title, status, status_date, deciders, context, decision,
		                  consequences, alternatives, more_info, superseded_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, (SELECT id FROM adrs WHERE number = ?), ?, ?)`,
		adr.Number,
		adr.Slug,
		adr.Title,
		adr.Status,
		adr.Date,
		adr.Deciders,
		adr.Context,
		adr.Decision,
		adr.Consequences,
		adr.Alternatives,
		adr.MoreInfo,
		adr.SupersededBy,
		adr.CreatedAt.Format(timeLayout),
		adr.UpdatedAt.Format(timeLayout),
	)
	if err != nil {
		if !isUnique(err) {
			return ADR{}, fmt.Errorf("saving the ADR: %w", err)
		}

		return ADR{}, takenADR(ctx, tx, adr)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return ADR{}, fmt.Errorf("reading the identity of the stored ADR: %w", err)
	}

	if err := recordChange(ctx, tx, s.origin, ResourceADR, id, ActionCreate); err != nil {
		return ADR{}, err
	}

	if err := tx.Commit(); err != nil {
		return ADR{}, fmt.Errorf("closing the creation transaction: %w", err)
	}

	adr.ID = id

	return adr, nil
}

// UpdateADR reads the ADR, hands it to change and writes back what change
// returns. Todo ocurre en una transacción: dos escrituras a la vez se ordenan
// en la base en vez de pisarse. change must not ask the base for anything,
// because the transaction already holds the write lock.
func (s *Store) UpdateADR(ctx context.Context, number int, change func(ADR) (ADR, error)) (ADR, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ADR{}, fmt.Errorf("opening the change transaction: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	stored, err := readADR(ctx, tx, number)
	if err != nil {
		return ADR{}, err
	}

	changed, err := change(stored)
	if err != nil {
		return ADR{}, err
	}

	// Ni la identidad interna, ni el número, ni el slug, ni el momento de
	// creación pertenecen a quien cambia el ADR: se conservan de lo leído.
	changed.ID = stored.ID
	changed.Number = stored.Number
	changed.Slug = stored.Slug
	changed.CreatedAt = stored.CreatedAt
	changed.UpdatedAt = time.Now().UTC().Truncate(time.Second)

	_, err = tx.ExecContext(ctx, `
		UPDATE adrs
		SET title = ?, status = ?, status_date = ?, deciders = ?, context = ?, decision = ?,
		    consequences = ?, alternatives = ?, more_info = ?, updated_at = ?,
		    superseded_by = (SELECT id FROM adrs WHERE number = ?)
		WHERE id = ?`,
		changed.Title,
		changed.Status,
		changed.Date,
		changed.Deciders,
		changed.Context,
		changed.Decision,
		changed.Consequences,
		changed.Alternatives,
		changed.MoreInfo,
		changed.UpdatedAt.Format(timeLayout),
		changed.SupersededBy,
		changed.ID,
	)
	if err != nil {
		return ADR{}, fmt.Errorf("saving the change of ADR %d: %w", number, err)
	}

	if err := recordChange(ctx, tx, s.origin, ResourceADR, changed.ID, ActionUpdate); err != nil {
		return ADR{}, err
	}

	if err := tx.Commit(); err != nil {
		return ADR{}, fmt.Errorf("closing the change transaction of ADR %d: %w", number, err)
	}

	return changed, nil
}

// takenADR says which of the two unique columns a refused insert collided with:
// el número o el slug, que es lo que quien llama necesita para decidir.
func takenADR(ctx context.Context, within *sql.Tx, adr ADR) error {
	var taken int

	err := within.QueryRowContext(ctx, `SELECT count(*) FROM adrs WHERE number = ?`, adr.Number).Scan(&taken)
	if err != nil {
		return fmt.Errorf("looking at the ADR number %d: %w", adr.Number, err)
	}

	if taken > 0 {
		return ErrADRNumberTaken
	}

	return ErrADRSlugTaken
}

// isUnique reports whether the error is SQLite's constraint violation, which is
// what a number or a slug already taken looks like.
func isUnique(err error) bool {
	return isCode(err, sqlite3.SQLITE_CONSTRAINT)
}
