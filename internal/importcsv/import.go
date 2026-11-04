package importcsv

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"leadcapture/internal/lead"
	"leadcapture/internal/store"
)

// Skip describes one rejected CSV row.
type Skip struct {
	Line   int    `json:"line"`
	Reason string `json:"reason"`
}

// Result summarizes an import run.
// Invalid rows are skipped (not fatal); Skips explains each with its CSV line number.
type Result struct {
	Imported int    `json:"imported"`
	Skipped  int    `json:"skipped"`
	Skips    []Skip `json:"skips"`
}

// File imports leads from a CSV path into the store.
// Skipped rows are reported to stderr with line numbers (header is line 1).
func File(path string, s *store.Store, stderr io.Writer) (Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return Result{}, fmt.Errorf("open csv: %w", err)
	}
	defer f.Close()
	return Reader(f, s, stderr)
}

// Reader imports leads from a CSV reader.
func Reader(r io.Reader, s *store.Store, stderr io.Writer) (Result, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	cr.FieldsPerRecord = -1 // allow variable fields; we validate columns ourselves

	header, err := cr.Read()
	if err != nil {
		return Result{}, fmt.Errorf("read header: %w", err)
	}
	col, err := mapColumns(header)
	if err != nil {
		return Result{}, err
	}

	result := Result{Skips: make([]Skip, 0)}
	line := 1 // header
	for {
		record, err := cr.Read()
		if err == io.EOF {
			break
		}
		line++
		if err != nil {
			recordSkip(&result, stderr, line, err.Error())
			continue
		}

		in, parseErrs := rowToInput(record, col)
		if len(parseErrs) > 0 {
			recordSkip(&result, stderr, line, joinErrors(parseErrs))
			continue
		}

		if errs := lead.Validate(&in); lead.HasErrors(errs) {
			recordSkip(&result, stderr, line, joinErrors(errs))
			continue
		}

		if _, err := s.Insert(in); err != nil {
			if err == store.ErrDuplicateEmail {
				recordSkip(&result, stderr, line, "email: already exists")
				continue
			}
			return result, fmt.Errorf("line %d: insert: %w", line, err)
		}
		result.Imported++
	}
	return result, nil
}

func recordSkip(result *Result, stderr io.Writer, line int, reason string) {
	result.Skipped++
	result.Skips = append(result.Skips, Skip{Line: line, Reason: reason})
	fmt.Fprintf(stderr, "line %d: %s\n", line, reason)
}

type columns struct {
	firstName, lastName, email, finance, salary, state int
	vehicleMake, vehicleModel, vehicleYear             int
}

func mapColumns(header []string) (columns, error) {
	idx := map[string]int{}
	for i, h := range header {
		idx[strings.TrimSpace(h)] = i
	}
	required := []string{
		"first_name", "last_name", "email", "finance_amount", "salary", "state",
		"vehicle_make", "vehicle_model", "vehicle_year",
	}
	var c columns
	for _, name := range required {
		i, ok := idx[name]
		if !ok {
			return columns{}, fmt.Errorf("missing CSV column %q", name)
		}
		switch name {
		case "first_name":
			c.firstName = i
		case "last_name":
			c.lastName = i
		case "email":
			c.email = i
		case "finance_amount":
			c.finance = i
		case "salary":
			c.salary = i
		case "state":
			c.state = i
		case "vehicle_make":
			c.vehicleMake = i
		case "vehicle_model":
			c.vehicleModel = i
		case "vehicle_year":
			c.vehicleYear = i
		}
	}
	return c, nil
}

func cell(record []string, i int) string {
	if i < 0 || i >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[i])
}

func rowToInput(record []string, c columns) (lead.Input, map[string]string) {
	errs := map[string]string{}
	in := lead.Input{
		FirstName: cell(record, c.firstName),
		LastName:  cell(record, c.lastName),
		Email:     cell(record, c.email),
		State:     cell(record, c.state),
	}

	finance, err := lead.ParseMoney(cell(record, c.finance))
	if err != nil {
		errs["finance_amount"] = err.Error()
	} else {
		in.FinanceAmount = finance
	}

	salary, err := lead.ParseMoney(cell(record, c.salary))
	if err != nil {
		errs["salary"] = err.Error()
	} else {
		in.Salary = salary
	}

	makeVal := cell(record, c.vehicleMake)
	modelVal := cell(record, c.vehicleModel)
	yearVal := cell(record, c.vehicleYear)

	if makeVal != "" || modelVal != "" || yearVal != "" {
		v := &lead.Vehicle{Make: makeVal}
		if modelVal != "" {
			v.Model = &modelVal
		}
		if yearVal != "" {
			y, err := strconv.Atoi(yearVal)
			if err != nil {
				errs["vehicle.year"] = "must be a number"
			} else {
				v.Year = &y
			}
		}
		in.Vehicle = v
	}

	return in, errs
}

func joinErrors(errs map[string]string) string {
	parts := make([]string, 0, len(errs))
	// Stable-ish order for common fields.
	order := []string{
		"first_name", "last_name", "email", "finance_amount", "salary", "state",
		"vehicle.make", "vehicle.model", "vehicle.year",
	}
	seen := map[string]bool{}
	for _, k := range order {
		if msg, ok := errs[k]; ok {
			parts = append(parts, fmt.Sprintf("%s: %s", k, msg))
			seen[k] = true
		}
	}
	for k, msg := range errs {
		if !seen[k] {
			parts = append(parts, fmt.Sprintf("%s: %s", k, msg))
		}
	}
	return strings.Join(parts, "; ")
}
