package importcsv

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"leadcapture/internal/store"
)

func TestImportSampleCSV(t *testing.T) {
	s := openStore(t)
	var stderr bytes.Buffer

	// leads.csv lives at repo root; tests run from package dir.
	csvPath := filepath.Join("..", "..", "leads.csv")
	result, err := File(csvPath, s, &stderr)
	if err != nil {
		t.Fatalf("File: %v", err)
	}
	if result.Imported == 0 {
		t.Fatal("expected some imported rows")
	}
	if result.Skipped == 0 {
		t.Fatal("expected some skipped rows from sample CSV")
	}

	out := stderr.String()
	if !strings.Contains(out, "line ") {
		t.Fatalf("expected skip reports with line numbers, got:\n%s", out)
	}

	list, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != result.Imported {
		t.Fatalf("list len %d != imported %d", len(list), result.Imported)
	}
}

func TestImportSkipsInvalidAndDuplicate(t *testing.T) {
	s := openStore(t)
	var stderr bytes.Buffer

	csvData := `first_name,last_name,email,finance_amount,salary,state,vehicle_make,vehicle_model,vehicle_year
Ada,Lovelace,ada@example.com,1000,,NSW,,,
,,,1000,,NSW,,,
Ada,Lovelace,ADA@example.com,2000,,VIC,,,
Bad,State,bad@example.com,1000,,XX,,,
`
	result, err := Reader(strings.NewReader(csvData), s, &stderr)
	if err != nil {
		t.Fatalf("Reader: %v", err)
	}
	if result.Imported != 1 {
		t.Fatalf("imported = %d, want 1", result.Imported)
	}
	if result.Skipped != 3 {
		t.Fatalf("skipped = %d, want 3; stderr:\n%s", result.Skipped, stderr.String())
	}
	if !strings.Contains(stderr.String(), "line 3:") {
		t.Errorf("expected line 3 skip, got:\n%s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "already exists") {
		t.Errorf("expected duplicate report, got:\n%s", stderr.String())
	}
}

func openStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}
