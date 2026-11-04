package lead

import (
	"testing"
	"time"
)

func floatPtr(v float64) *float64 { return &v }
func intPtr(v int) *int           { return &v }
func strPtr(v string) *string     { return &v }

func validInput() Input {
	amount := 1000.0
	return Input{
		FirstName:     "Ada",
		LastName:      "Lovelace",
		Email:         "ada@example.com",
		FinanceAmount: &amount,
		State:         "NSW",
	}
}

func TestValidate_ValidMinimal(t *testing.T) {
	in := validInput()
	errs := Validate(&in)
	if HasErrors(errs) {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if in.Email != "ada@example.com" {
		t.Errorf("email = %q", in.Email)
	}
	if in.State != "NSW" {
		t.Errorf("state = %q", in.State)
	}
}

func TestValidate_NormalizesEmailAndState(t *testing.T) {
	in := validInput()
	in.Email = "ADA@Example.COM"
	in.State = "vic"
	errs := Validate(&in)
	if HasErrors(errs) {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if in.Email != "ada@example.com" {
		t.Errorf("email = %q, want lowercase", in.Email)
	}
	if in.State != "VIC" {
		t.Errorf("state = %q, want VIC", in.State)
	}
}

func TestValidate_RequiredFields(t *testing.T) {
	in := Input{}
	errs := Validate(&in)
	for _, key := range []string{"first_name", "last_name", "email", "finance_amount", "state"} {
		if _, ok := errs[key]; !ok {
			t.Errorf("missing error for %s", key)
		}
	}
}

func TestValidate_FinanceAmountMustBePositive(t *testing.T) {
	in := validInput()
	in.FinanceAmount = floatPtr(0)
	errs := Validate(&in)
	if errs["finance_amount"] != "must be greater than 0" {
		t.Errorf("finance_amount error = %q", errs["finance_amount"])
	}

	in = validInput()
	in.FinanceAmount = floatPtr(-1)
	errs = Validate(&in)
	if errs["finance_amount"] != "must be greater than 0" {
		t.Errorf("finance_amount error = %q", errs["finance_amount"])
	}
}

func TestValidate_SalaryOptionalButPositive(t *testing.T) {
	in := validInput()
	errs := Validate(&in)
	if HasErrors(errs) {
		t.Fatalf("expected no errors without salary, got %v", errs)
	}

	in = validInput()
	in.Salary = floatPtr(50000)
	errs = Validate(&in)
	if HasErrors(errs) {
		t.Fatalf("expected no errors with positive salary, got %v", errs)
	}

	in = validInput()
	in.Salary = floatPtr(0)
	errs = Validate(&in)
	if errs["salary"] != "must be greater than 0" {
		t.Errorf("salary error = %q", errs["salary"])
	}
}

func TestValidate_InvalidState(t *testing.T) {
	in := validInput()
	in.State = "XX"
	errs := Validate(&in)
	if errs["state"] == "" {
		t.Fatal("expected state error")
	}

	in = validInput()
	in.State = "Victoria"
	errs = Validate(&in)
	if errs["state"] == "" {
		t.Fatal("expected state error for full name")
	}
}

func TestValidate_VehicleMakeRequiredWhenDetailGiven(t *testing.T) {
	in := validInput()
	in.Vehicle = &Vehicle{Model: strPtr("Corolla"), Year: intPtr(2021)}
	errs := Validate(&in)
	if errs["vehicle.make"] != "is required when a model or year is given" {
		t.Errorf("vehicle.make error = %q", errs["vehicle.make"])
	}
}

func TestValidate_VehicleMakeAloneOK(t *testing.T) {
	in := validInput()
	in.Vehicle = &Vehicle{Make: "Mazda"}
	errs := Validate(&in)
	if HasErrors(errs) {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if in.Vehicle == nil || in.Vehicle.Make != "Mazda" {
		t.Fatalf("vehicle not preserved: %+v", in.Vehicle)
	}
}

func TestValidate_EmptyVehicleCleared(t *testing.T) {
	in := validInput()
	in.Vehicle = &Vehicle{}
	errs := Validate(&in)
	if HasErrors(errs) {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if in.Vehicle != nil {
		t.Errorf("empty vehicle should be cleared, got %+v", in.Vehicle)
	}
}

func TestValidate_VehicleYearBounds(t *testing.T) {
	maxYear := time.Now().Year() + 1

	in := validInput()
	in.Vehicle = &Vehicle{Make: "Ford", Year: intPtr(1989)}
	errs := Validate(&in)
	if errs["vehicle.year"] == "" {
		t.Fatal("expected year error for 1989")
	}

	in = validInput()
	in.Vehicle = &Vehicle{Make: "Ford", Year: intPtr(maxYear + 1)}
	errs = Validate(&in)
	if errs["vehicle.year"] == "" {
		t.Fatal("expected year error for beyond next year")
	}

	in = validInput()
	in.Vehicle = &Vehicle{Make: "Ford", Year: intPtr(1990)}
	errs = Validate(&in)
	if HasErrors(errs) {
		t.Fatalf("1990 should be valid, got %v", errs)
	}

	in = validInput()
	in.Vehicle = &Vehicle{Make: "Ford", Year: intPtr(maxYear)}
	errs = Validate(&in)
	if HasErrors(errs) {
		t.Fatalf("next year should be valid, got %v", errs)
	}
}

func TestParseMoney(t *testing.T) {
	tests := []struct {
		in      string
		want    *float64
		wantErr bool
	}{
		{"", nil, false},
		{"  ", nil, false},
		{"52000", floatPtr(52000), false},
		{"38500.5", floatPtr(38500.5), false},
		{"$45,000.00", floatPtr(45000), false},
		{"abc", nil, true},
	}
	for _, tt := range tests {
		got, err := ParseMoney(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseMoney(%q) expected error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseMoney(%q) unexpected error: %v", tt.in, err)
			continue
		}
		if tt.want == nil {
			if got != nil {
				t.Errorf("ParseMoney(%q) = %v, want nil", tt.in, *got)
			}
			continue
		}
		if got == nil || *got != *tt.want {
			t.Errorf("ParseMoney(%q) = %v, want %v", tt.in, got, *tt.want)
		}
	}
}
