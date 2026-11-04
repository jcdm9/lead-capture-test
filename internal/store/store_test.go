package store

import (
	"path/filepath"
	"testing"

	"leadcapture/internal/lead"
)

func amount(v float64) *float64 { return &v }
func strPtr(v string) *string   { return &v }
func intPtr(v int) *int         { return &v }

func validInput(email string) lead.Input {
	return lead.Input{
		FirstName:     "Ada",
		LastName:      "Lovelace",
		Email:         email,
		FinanceAmount: amount(1000),
		State:         "NSW",
	}
}

func TestInsertAndList(t *testing.T) {
	s := openTestStore(t)

	in := validInput("ada@example.com")
	in.Salary = amount(90000)
	in.Vehicle = &lead.Vehicle{Make: "Kia", Model: strPtr("EV6"), Year: intPtr(2024)}

	got, err := s.Insert(in)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if got.ID == 0 {
		t.Fatal("expected non-zero id")
	}
	if got.Email != "ada@example.com" {
		t.Errorf("email = %q", got.Email)
	}
	if got.Salary == nil || *got.Salary != 90000 {
		t.Errorf("salary = %v", got.Salary)
	}
	if got.Vehicle == nil || got.Vehicle.Make != "Kia" {
		t.Errorf("vehicle = %+v", got.Vehicle)
	}
	if got.CreatedAt.IsZero() {
		t.Error("created_at should be set")
	}

	list, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len = %d, want 1", len(list))
	}
}

func TestListEmpty(t *testing.T) {
	s := openTestStore(t)
	list, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if list == nil {
		t.Fatal("list should be empty slice, not nil")
	}
	if len(list) != 0 {
		t.Fatalf("len = %d", len(list))
	}
}

func TestDuplicateEmail(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.Insert(validInput("ada@example.com")); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	_, err := s.Insert(validInput("ADA@example.com"))
	if err != ErrDuplicateEmail {
		t.Fatalf("second insert error = %v, want ErrDuplicateEmail", err)
	}
}

func TestInsertWithoutVehicle(t *testing.T) {
	s := openTestStore(t)
	got, err := s.Insert(validInput("no-car@example.com"))
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if got.Vehicle != nil {
		t.Errorf("vehicle = %+v, want nil", got.Vehicle)
	}
	if got.Salary != nil {
		t.Errorf("salary = %v, want nil", got.Salary)
	}
}

func TestUpdateAndDelete(t *testing.T) {
	s := openTestStore(t)
	created, err := s.Insert(validInput("ada@example.com"))
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}

	updatedIn := validInput("ada.lovelace@example.com")
	updatedIn.FirstName = "Augusta"
	updatedIn.FinanceAmount = amount(2500)
	updatedIn.Vehicle = &lead.Vehicle{Make: "Tesla", Model: strPtr("Model 3")}

	updated, err := s.Update(created.ID, updatedIn)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.FirstName != "Augusta" || updated.Email != "ada.lovelace@example.com" {
		t.Fatalf("unexpected update: %+v", updated)
	}
	if updated.FinanceAmount != 2500 {
		t.Fatalf("finance = %v", updated.FinanceAmount)
	}
	if updated.Vehicle == nil || updated.Vehicle.Make != "Tesla" {
		t.Fatalf("vehicle = %+v", updated.Vehicle)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Fatalf("created_at changed: %v -> %v", created.CreatedAt, updated.CreatedAt)
	}

	// Same email on same row is fine.
	if _, err := s.Update(created.ID, updatedIn); err != nil {
		t.Fatalf("Update same email: %v", err)
	}

	other, err := s.Insert(validInput("other@example.com"))
	if err != nil {
		t.Fatalf("Insert other: %v", err)
	}
	clash := validInput("other@example.com")
	if _, err := s.Update(created.ID, clash); err != ErrDuplicateEmail {
		t.Fatalf("Update clash = %v, want ErrDuplicateEmail", err)
	}

	if err := s.Delete(other.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(other.ID); err != ErrNotFound {
		t.Fatalf("Get deleted = %v, want ErrNotFound", err)
	}
	if err := s.Delete(other.ID); err != ErrNotFound {
		t.Fatalf("Delete missing = %v, want ErrNotFound", err)
	}
	if _, err := s.Update(9999, validInput("x@example.com")); err != ErrNotFound {
		t.Fatalf("Update missing = %v, want ErrNotFound", err)
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}
