package lead

import "time"

// Vehicle is the nested vehicle object in API responses and requests.
type Vehicle struct {
	Make  string  `json:"make"`
	Model *string `json:"model"`
	Year  *int    `json:"year"`
}

// Lead is the API representation of a stored lead.
type Lead struct {
	ID            int64     `json:"id"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Email         string    `json:"email"`
	FinanceAmount float64   `json:"finance_amount"`
	Salary        *float64  `json:"salary"`
	State         string    `json:"state"`
	Vehicle       *Vehicle  `json:"vehicle"`
	CreatedAt     time.Time `json:"created_at"`
}

// Input is the shape accepted by validation (CSV import and POST /api/leads).
// FinanceAmount is a pointer so missing/null JSON can be detected.
type Input struct {
	FirstName     string   `json:"first_name"`
	LastName      string   `json:"last_name"`
	Email         string   `json:"email"`
	FinanceAmount *float64 `json:"finance_amount"`
	Salary        *float64 `json:"salary"`
	State         string   `json:"state"`
	Vehicle       *Vehicle `json:"vehicle"`
}
