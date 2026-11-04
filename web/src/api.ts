import type { CreateLeadRequest, ImportResult, Lead, ValidationErrorResponse } from './types'
import { ApiError } from './types'

async function readValidationError(res: Response): Promise<never> {
  if (res.status === 422) {
    const data = (await res.json()) as ValidationErrorResponse
    throw new ApiError('validation failed', 422, data.errors)
  }
  if (res.status === 400) {
    throw new ApiError('malformed request', 400)
  }
  if (res.status === 404) {
    throw new ApiError('lead not found', 404)
  }
  throw new ApiError('request failed', res.status)
}

export async function fetchLeads(): Promise<Lead[]> {
  const res = await fetch('/api/leads')
  if (!res.ok) {
    throw new ApiError('failed to load leads', res.status)
  }
  return (await res.json()) as Lead[]
}

export async function fetchLead(id: number): Promise<Lead> {
  const res = await fetch(`/api/leads/${id}`)
  if (!res.ok) {
    return readValidationError(res)
  }
  return (await res.json()) as Lead
}

export async function createLead(body: CreateLeadRequest): Promise<Lead> {
  const res = await fetch('/api/leads', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })

  if (res.status === 201) {
    return (await res.json()) as Lead
  }
  return readValidationError(res)
}

export async function updateLead(id: number, body: CreateLeadRequest): Promise<Lead> {
  const res = await fetch(`/api/leads/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })

  if (res.status === 200) {
    return (await res.json()) as Lead
  }
  return readValidationError(res)
}

export async function deleteLead(id: number): Promise<void> {
  const res = await fetch(`/api/leads/${id}`, { method: 'DELETE' })
  if (res.status === 204) {
    return
  }
  return readValidationError(res)
}

export async function importLeadsCSV(file: File): Promise<ImportResult> {
  const body = new FormData()
  body.append('file', file)

  const res = await fetch('/api/leads/import', {
    method: 'POST',
    body,
  })

  if (res.status === 200) {
    return (await res.json()) as ImportResult
  }

  if (res.status === 400) {
    const data = (await res.json()) as { error?: string }
    throw new ApiError(data.error || 'invalid CSV upload', 400)
  }

  throw new ApiError('failed to import CSV', res.status)
}
