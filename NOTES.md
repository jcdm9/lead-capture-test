# NOTES

## Implementation Decisions

### Table-first UI with modal CRUD
- **Problem:** The always-visible create form competed with the leads table; staff needed update/delete as well as create.
- **Chosen approach:** Show the table on load; open Add/Edit in a modal. API extended with `GET/PUT/DELETE /api/leads/:id` (PUT uses the same body/validation as POST; DELETE returns `204`).
- **Why:** Keeps scanning the list primary; reuses one form for create and edit.
- **Advantages:** Clearer default view; full CRUD without a second form layout.
- **Disadvantages:** Goes beyond the original README endpoints (explicit product request).
- **Alternatives considered:** Inline expandable row editor; separate edit page.

### Empty-state CSV upload
- **Problem:** With no leads, staff still need a way to load a partner CSV without using the CLI.
- **Chosen approach:** `POST /api/leads/import` (multipart field `file`, `.csv` only) reuses the shared import/validation path. Empty table state shows drag-and-drop + browse UI.
- **Why:** Same rules as `leads import`; low extra surface area.
- **Advantages:** Consistent validation; immediate UI path for first-time setup.
- **Disadvantages:** Import is currently offered only when the table is empty.
- **Alternatives considered:** Always-visible import control in the header.

### Shared validation for CSV and API
- **Problem:** The same lead rules must apply to import and `POST /api/leads`.
- **Chosen approach:** One `lead.Validate` function in `internal/lead`, used by both the CSV importer and the HTTP handler. Uniqueness is enforced in SQLite and mapped to a field error by callers.
- **Why:** Avoids divergent rules; keeps the app small.
- **Advantages:** Single source of truth; easy to test in isolation.
- **Disadvantages:** Callers must translate store errors (e.g. duplicate email) into the API/CSV error shape.
- **Alternatives considered:** Separate validators per path; validation only in the DB layer.

### Australian state codes only, normalized uppercase
- **Problem:** README requires a valid Australian state/territory; CSV includes lowercase codes and a full name (`Victoria`).
- **Chosen approach:** Accept only the eight codes (`NSW`…`ACT`), case-insensitive, store uppercase.
- **Why:** Matches partner-style coded data; full names are ambiguous and easy to get wrong.
- **Advantages:** Simple set lookup; consistent storage.
- **Disadvantages:** Rows with full state names (e.g. CSV line 13) are skipped.
- **Alternatives considered:** Accept full names and map to codes.

### Email uniqueness lowercase + HTTP 422
- **Problem:** Emails must be unique; sample CSV has a case variant of an existing email.
- **Chosen approach:** Normalize emails to lowercase before insert; SQLite `UNIQUE` (with `COLLATE NOCASE` as a backstop); API returns `422` with `errors.email`.
- **Why:** Core contract uses `422` for validation failures; stretch `409` deferred.
- **Advantages:** Consistent with other field errors; case-insensitive uniqueness.
- **Disadvantages:** Clients cannot distinguish “duplicate” from other email failures by status code alone.
- **Alternatives considered:** `409 Conflict` (stretch goal).

### SQLite schema with nullable vehicle columns
- **Problem:** Vehicle is optional and partially filled in the JSON contract.
- **Chosen approach:** Flat columns (`vehicle_make`, `vehicle_model`, `vehicle_year`) on `leads`; map to nested `vehicle` (or `null`) in Go.
- **Why:** Matches CSV columns and keeps queries trivial—no JSON column parsing.
- **Advantages:** Easy to inspect in SQLite; straightforward inserts.
- **Disadvantages:** Slightly more mapping code at the store boundary.
- **Alternatives considered:** Store vehicle as a JSON blob.

### Single binary with `embed`
- **Problem:** `make build` must produce one binary that serves API + UI.
- **Chosen approach:** Vite builds to `web/dist`; `//go:embed all:web/dist` in `serve.go`; Gin serves `/api` and static files.
- **Why:** Matches the Makefile and day-to-day stack; CGO-free via `modernc.org/sqlite`.
- **Advantages:** Simple deploy artifact.
- **Disadvantages:** `web/dist` must exist before `go build` (Makefile already builds the web app first).

### Package layout
- **Problem:** Need a clear structure without overengineering.
- **Chosen approach:** `main` CLI + `internal/{lead,store,importcsv,api,server}`.
- **Why:** Small surface area; Makefile still builds `.`.
- **Advantages:** Easy to navigate for a short coding test.
- **Disadvantages:** Not a multi-command `cmd/` layout (unnecessary here).

## Rejected AI Suggestions

- **Accept full state names (e.g. “Victoria”) during import.** Rejected because we deliberately standardized on codes only: simpler validation, consistent storage, and the sample CSV’s full-name row is a useful invalid case rather than something to quietly “fix.”
- **Return `409` for duplicate emails.** Rejected for now to stick to the core `422` field-error contract; can add later as the stretch goal without changing validation logic.

## Outstanding Concerns

- Local Node was below the README engines (`v22.11.0`); building the web app required Homebrew Node 26 (`>=24.12`). CI/reviewers should use the documented Node versions.
- `go test ./...` also discovers a tiny Go package under `web/node_modules/flatted` (npm dependency). Harmless, but noisy; could narrow the test path later.
- CSV money parsing accepts `$` and commas; unusual formats still fail—acceptable for the sample file.

## Future Improvements

- Stretch: filter by state, sorting, client-side validation matching server rules, `409` on duplicate email.
- Richer import summary (per-field skip counts) and optional dry-run mode.
- Configurable DB path and listen address via flags/env.
- Narrow `make test` to `./internal/...` (and maybe `./`) to avoid scanning `node_modules`.
