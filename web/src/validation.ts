import { AU_STATE_CODES } from './constants'
import type { CreateLeadRequest } from './types'

/** Client-side validation mirroring server lead.Validate messages. */
export function validateLead(input: CreateLeadRequest): Record<string, string> {
  const errors: Record<string, string> = {}

  const firstName = input.first_name.trim()
  const lastName = input.last_name.trim()
  const email = input.email.trim()
  const state = input.state.trim().toUpperCase()

  if (firstName === '') errors.first_name = 'is required'
  if (lastName === '') errors.last_name = 'is required'
  if (email === '') errors.email = 'is required'

  if (input.finance_amount === null || Number.isNaN(input.finance_amount)) {
    errors.finance_amount = 'is required'
  } else if (input.finance_amount <= 0) {
    errors.finance_amount = 'must be greater than 0'
  }

  if (input.salary !== null) {
    if (Number.isNaN(input.salary) || input.salary <= 0) {
      errors.salary = 'must be greater than 0'
    }
  }

  if (state === '') {
    errors.state = 'is required'
  } else if (!(AU_STATE_CODES as readonly string[]).includes(state)) {
    errors.state = 'must be a valid Australian state or territory'
  }

  if (input.vehicle) {
    const make = input.vehicle.make.trim()
    const model = input.vehicle.model?.trim() ?? ''
    const year = input.vehicle.year
    const hasDetail = make !== '' || model !== '' || year !== null

    if (hasDetail) {
      if (make === '') {
        errors['vehicle.make'] = 'is required when a model or year is given'
      }
      if (year !== null) {
        const maxYear = new Date().getFullYear() + 1
        if (year < 1990 || year > maxYear) {
          errors['vehicle.year'] = 'must be between 1990 and next year'
        }
      }
    }
  }

  return errors
}

export function hasValidationErrors(errors: Record<string, string>): boolean {
  return Object.keys(errors).length > 0
}
