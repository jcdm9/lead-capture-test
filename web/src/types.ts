/** Nested vehicle object; null on a lead means no vehicle. */
export interface Vehicle {
  make: string
  model: string | null
  year: number | null
}

/** Lead as returned by GET/POST /api/leads. */
export interface Lead {
  id: number
  first_name: string
  last_name: string
  email: string
  finance_amount: number
  salary: number | null
  state: string
  vehicle: Vehicle | null
  created_at: string
}

/** Body for POST /api/leads (no id or created_at). */
export interface CreateLeadRequest {
  first_name: string
  last_name: string
  email: string
  finance_amount: number | null
  salary: number | null
  state: string
  vehicle: Vehicle | null
}

export interface ValidationErrorResponse {
  errors: Record<string, string>
}

/** One skipped CSV row from POST /api/leads/import */
export interface ImportSkip {
  line: number
  reason: string
}

/** Result of POST /api/leads/import */
export interface ImportResult {
  imported: number
  skipped: number
  skips: ImportSkip[]
}

export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly errors?: Record<string, string>,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}
