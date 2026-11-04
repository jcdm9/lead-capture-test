package api

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"leadcapture/internal/lead"
	"leadcapture/internal/store"
)

func TestListEmpty(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/leads", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var leads []lead.Lead
	if err := json.Unmarshal(w.Body.Bytes(), &leads); err != nil {
		t.Fatalf("json: %v", err)
	}
	if leads == nil {
		t.Fatal("expected [] not null")
	}
	if len(leads) != 0 {
		t.Fatalf("len = %d", len(leads))
	}
}

func TestCreateAndList(t *testing.T) {
	r := newTestRouter(t)
	body := `{
		"first_name": "Zoë",
		"last_name": "Martin",
		"email": "zoe.martin@example.com",
		"finance_amount": 38500.5,
		"salary": null,
		"state": "qld",
		"vehicle": null
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/leads", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	var created lead.Lead
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("json: %v", err)
	}
	if created.ID == 0 || created.State != "QLD" || created.Email != "zoe.martin@example.com" {
		t.Fatalf("unexpected lead: %+v", created)
	}
	if created.Salary != nil || created.Vehicle != nil {
		t.Fatalf("expected null salary/vehicle, got salary=%v vehicle=%v", created.Salary, created.Vehicle)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/leads", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var leads []lead.Lead
	_ = json.Unmarshal(w.Body.Bytes(), &leads)
	if len(leads) != 1 {
		t.Fatalf("list len = %d", len(leads))
	}
}

func TestCreateValidationError(t *testing.T) {
	r := newTestRouter(t)
	body := `{"first_name":"","last_name":"X","email":"a@b.com","finance_amount":0,"state":"XX"}`
	req := httptest.NewRequest(http.MethodPost, "/api/leads", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", w.Code)
	}
	var resp struct {
		Errors map[string]string `json:"errors"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v", err)
	}
	if resp.Errors["first_name"] == "" || resp.Errors["finance_amount"] == "" || resp.Errors["state"] == "" {
		t.Fatalf("errors = %v", resp.Errors)
	}
}

func TestCreateMalformedJSON(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/leads", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestCreateDuplicateEmail(t *testing.T) {
	r := newTestRouter(t)
	body := `{
		"first_name": "A",
		"last_name": "B",
		"email": "dup@example.com",
		"finance_amount": 1000,
		"state": "NSW"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/leads", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("first create status = %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/leads", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("duplicate status = %d", w.Code)
	}
	var resp struct {
		Errors map[string]string `json:"errors"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Errors["email"] == "" {
		t.Fatalf("expected email error, got %v", resp.Errors)
	}
}

func TestCreatePartialVehicle(t *testing.T) {
	r := newTestRouter(t)
	body := `{
		"first_name": "A",
		"last_name": "B",
		"email": "car@example.com",
		"finance_amount": 1000,
		"state": "NSW",
		"vehicle": { "make": "Kia", "model": "EV6", "year": null }
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/leads", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	var created lead.Lead
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	if created.Vehicle == nil || created.Vehicle.Make != "Kia" || created.Vehicle.Year != nil {
		t.Fatalf("vehicle = %+v", created.Vehicle)
	}
}

func TestImportCSV(t *testing.T) {
	r := newTestRouter(t)

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", "leads.csv")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	csvData := `first_name,last_name,email,finance_amount,salary,state,vehicle_make,vehicle_model,vehicle_year
Ada,Lovelace,ada@import.example,1000,,NSW,,,
Bad,State,bad@import.example,1000,,XX,,,
`
	if _, err := io.WriteString(part, csvData); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/leads/import", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var result struct {
		Imported int `json:"imported"`
		Skipped  int `json:"skipped"`
		Skips    []struct {
			Line   int    `json:"line"`
			Reason string `json:"reason"`
		} `json:"skips"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("json: %v", err)
	}
	if result.Imported != 1 || result.Skipped != 1 {
		t.Fatalf("result = %+v", result)
	}
	if len(result.Skips) != 1 || result.Skips[0].Line != 3 || result.Skips[0].Reason == "" {
		t.Fatalf("skips = %+v", result.Skips)
	}

	// Reject non-csv
	body.Reset()
	w = multipart.NewWriter(&body)
	part, _ = w.CreateFormFile("file", "leads.txt")
	_, _ = io.WriteString(part, "nope")
	_ = w.Close()
	req = httptest.NewRequest(http.MethodPost, "/api/leads/import", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("non-csv status = %d", rec.Code)
	}
}

func TestGetUpdateDelete(t *testing.T) {
	r := newTestRouter(t)
	createBody := `{
		"first_name": "A",
		"last_name": "B",
		"email": "crud@example.com",
		"finance_amount": 1000,
		"state": "NSW"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/leads", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d", w.Code)
	}
	var created lead.Lead
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	req = httptest.NewRequest(http.MethodGet, "/api/leads/"+itoa(created.ID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get status = %d", w.Code)
	}

	updateBody := `{
		"first_name": "Ann",
		"last_name": "Bee",
		"email": "crud@example.com",
		"finance_amount": 2000,
		"state": "VIC",
		"vehicle": { "make": "Ford", "model": null, "year": null }
	}`
	req = httptest.NewRequest(http.MethodPut, "/api/leads/"+itoa(created.ID), bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update status = %d body = %s", w.Code, w.Body.String())
	}
	var updated lead.Lead
	_ = json.Unmarshal(w.Body.Bytes(), &updated)
	if updated.FirstName != "Ann" || updated.FinanceAmount != 2000 || updated.State != "VIC" {
		t.Fatalf("updated = %+v", updated)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/leads/"+itoa(created.ID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/leads/"+itoa(created.ID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("get deleted status = %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodPut, "/api/leads/9999", bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("update missing status = %d", w.Code)
	}
}

func itoa(id int64) string {
	return strconv.FormatInt(id, 10)
}

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return NewRouter(s)
}
