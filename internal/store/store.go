package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"leadcapture/internal/lead"

	_ "modernc.org/sqlite"
)

// ErrDuplicateEmail is returned when inserting/updating a lead with an existing email.
var ErrDuplicateEmail = errors.New("duplicate email")

// ErrNotFound is returned when a lead id does not exist.
var ErrNotFound = errors.New("lead not found")

// Store wraps the SQLite database for leads.
type Store struct {
	db *sql.DB
}

// Open opens (or creates) the SQLite database at path and migrates the schema.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	// Enforce foreign keys and make concurrent access safer for a small app.
	if _, err := db.Exec(`PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pragma: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS leads (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	first_name TEXT NOT NULL,
	last_name TEXT NOT NULL,
	email TEXT NOT NULL UNIQUE COLLATE NOCASE,
	finance_amount REAL NOT NULL,
	salary REAL,
	state TEXT NOT NULL,
	vehicle_make TEXT,
	vehicle_model TEXT,
	vehicle_year INTEGER,
	created_at TEXT NOT NULL
);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// Close closes the underlying database.
func (s *Store) Close() error {
	return s.db.Close()
}

// Insert validates uniqueness at the DB layer and inserts a lead from validated input.
func (s *Store) Insert(in lead.Input) (lead.Lead, error) {
	make, model, year := vehicleCols(in.Vehicle)
	res, err := s.db.Exec(`
INSERT INTO leads (
	first_name, last_name, email, finance_amount, salary, state,
	vehicle_make, vehicle_model, vehicle_year, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.FirstName,
		in.LastName,
		in.Email,
		*in.FinanceAmount,
		salaryValue(in.Salary),
		in.State,
		nullStringOrNil(make),
		nullStringOrNil(model),
		nullIntOrNil(year),
		time.Now().Format(time.RFC3339),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return lead.Lead{}, ErrDuplicateEmail
		}
		return lead.Lead{}, fmt.Errorf("insert lead: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return lead.Lead{}, fmt.Errorf("last insert id: %w", err)
	}
	return s.Get(id)
}

// Update replaces a lead's mutable fields. created_at is preserved.
func (s *Store) Update(id int64, in lead.Input) (lead.Lead, error) {
	if _, err := s.Get(id); err != nil {
		return lead.Lead{}, err
	}
	make, model, year := vehicleCols(in.Vehicle)
	res, err := s.db.Exec(`
UPDATE leads SET
	first_name = ?, last_name = ?, email = ?, finance_amount = ?, salary = ?, state = ?,
	vehicle_make = ?, vehicle_model = ?, vehicle_year = ?
WHERE id = ?`,
		in.FirstName,
		in.LastName,
		in.Email,
		*in.FinanceAmount,
		salaryValue(in.Salary),
		in.State,
		nullStringOrNil(make),
		nullStringOrNil(model),
		nullIntOrNil(year),
		id,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return lead.Lead{}, ErrDuplicateEmail
		}
		return lead.Lead{}, fmt.Errorf("update lead: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return lead.Lead{}, fmt.Errorf("update rows affected: %w", err)
	}
	if n == 0 {
		return lead.Lead{}, ErrNotFound
	}
	return s.Get(id)
}

// Delete removes a lead by id.
func (s *Store) Delete(id int64) error {
	res, err := s.db.Exec(`DELETE FROM leads WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete lead: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Get returns a lead by id.
func (s *Store) Get(id int64) (lead.Lead, error) {
	row := s.db.QueryRow(`
SELECT id, first_name, last_name, email, finance_amount, salary, state,
	vehicle_make, vehicle_model, vehicle_year, created_at
FROM leads WHERE id = ?`, id)
	l, err := scanLead(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "sql: no rows") {
			return lead.Lead{}, ErrNotFound
		}
		return lead.Lead{}, err
	}
	return l, nil
}

// List returns all leads ordered by id ascending.
func (s *Store) List() ([]lead.Lead, error) {
	rows, err := s.db.Query(`
SELECT id, first_name, last_name, email, finance_amount, salary, state,
	vehicle_make, vehicle_model, vehicle_year, created_at
FROM leads ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list leads: %w", err)
	}
	defer rows.Close()

	leads := make([]lead.Lead, 0)
	for rows.Next() {
		l, err := scanLead(rows)
		if err != nil {
			return nil, err
		}
		leads = append(leads, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return leads, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanLead(row scanner) (lead.Lead, error) {
	var (
		l          lead.Lead
		salary     sql.NullFloat64
		make       sql.NullString
		model      sql.NullString
		year       sql.NullInt64
		createdRaw string
	)
	err := row.Scan(
		&l.ID,
		&l.FirstName,
		&l.LastName,
		&l.Email,
		&l.FinanceAmount,
		&salary,
		&l.State,
		&make,
		&model,
		&year,
		&createdRaw,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return lead.Lead{}, err
		}
		return lead.Lead{}, fmt.Errorf("scan lead: %w", err)
	}
	if salary.Valid {
		v := salary.Float64
		l.Salary = &v
	}
	if make.Valid || model.Valid || year.Valid {
		v := &lead.Vehicle{}
		if make.Valid {
			v.Make = make.String
		}
		if model.Valid {
			m := model.String
			v.Model = &m
		}
		if year.Valid {
			y := int(year.Int64)
			v.Year = &y
		}
		l.Vehicle = v
	}
	createdAt, err := time.Parse(time.RFC3339, createdRaw)
	if err != nil {
		return lead.Lead{}, fmt.Errorf("parse created_at: %w", err)
	}
	l.CreatedAt = createdAt
	return l, nil
}

func vehicleCols(v *lead.Vehicle) (make, model sql.NullString, year sql.NullInt64) {
	if v == nil {
		return
	}
	make = sql.NullString{String: v.Make, Valid: v.Make != ""}
	if v.Model != nil {
		model = sql.NullString{String: *v.Model, Valid: true}
	}
	if v.Year != nil {
		year = sql.NullInt64{Int64: int64(*v.Year), Valid: true}
	}
	return
}

func salaryValue(salary *float64) any {
	if salary == nil {
		return nil
	}
	return *salary
}

func nullStringOrNil(ns sql.NullString) any {
	if !ns.Valid {
		return nil
	}
	return ns.String
}

func nullIntOrNil(ni sql.NullInt64) any {
	if !ni.Valid {
		return nil
	}
	return ni.Int64
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}
