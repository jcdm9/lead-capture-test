package lead

import (
	"strings"
	"time"
)

var australianStates = map[string]struct{}{
	"NSW": {},
	"VIC": {},
	"QLD": {},
	"SA":  {},
	"WA":  {},
	"TAS": {},
	"NT":  {},
	"ACT": {},
}

// Validate checks lead input against the shared business rules.
// Returns a map of field → message; empty map means valid.
// On success, mutates in to normalize email (lowercase) and state (uppercase),
// and trims string fields.
func Validate(in *Input) map[string]string {
	errors := map[string]string{}

	in.FirstName = strings.TrimSpace(in.FirstName)
	in.LastName = strings.TrimSpace(in.LastName)
	in.Email = strings.TrimSpace(in.Email)
	in.State = strings.TrimSpace(in.State)

	if in.FirstName == "" {
		errors["first_name"] = "is required"
	}
	if in.LastName == "" {
		errors["last_name"] = "is required"
	}
	if in.Email == "" {
		errors["email"] = "is required"
	}
	if in.FinanceAmount == nil {
		errors["finance_amount"] = "is required"
	} else if *in.FinanceAmount <= 0 {
		errors["finance_amount"] = "must be greater than 0"
	}
	if in.Salary != nil && *in.Salary <= 0 {
		errors["salary"] = "must be greater than 0"
	}
	if in.State == "" {
		errors["state"] = "is required"
	} else {
		state := strings.ToUpper(in.State)
		if _, ok := australianStates[state]; !ok {
			errors["state"] = "must be a valid Australian state or territory"
		} else {
			in.State = state
		}
	}

	if in.Vehicle != nil {
		normalizeVehicle(in.Vehicle)
		if vehicleHasDetail(in.Vehicle) {
			if in.Vehicle.Make == "" {
				errors["vehicle.make"] = "is required when a model or year is given"
			}
			if in.Vehicle.Year != nil {
				maxYear := time.Now().Year() + 1
				y := *in.Vehicle.Year
				if y < 1990 || y > maxYear {
					errors["vehicle.year"] = "must be between 1990 and next year"
				}
			}
		} else {
			in.Vehicle = nil
		}
	}

	if in.Email != "" {
		in.Email = strings.ToLower(in.Email)
	}

	return errors
}

func normalizeVehicle(v *Vehicle) {
	v.Make = strings.TrimSpace(v.Make)
	if v.Model != nil {
		m := strings.TrimSpace(*v.Model)
		if m == "" {
			v.Model = nil
		} else {
			v.Model = &m
		}
	}
}

func vehicleHasDetail(v *Vehicle) bool {
	if v.Make != "" {
		return true
	}
	if v.Model != nil && *v.Model != "" {
		return true
	}
	if v.Year != nil {
		return true
	}
	return false
}

// HasErrors reports whether the validation map contains any errors.
func HasErrors(errors map[string]string) bool {
	return len(errors) > 0
}
