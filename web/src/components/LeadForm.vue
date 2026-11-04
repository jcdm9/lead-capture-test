<script setup lang="ts">
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { createLead, updateLead } from '../api'
import { ApiError, type CreateLeadRequest, type Lead } from '../types'

const props = defineProps<{
  lead?: Lead | null
}>()

const emit = defineEmits<{
  saved: []
}>()

const AU_STATES = [
  { code: 'NSW', name: 'New South Wales' },
  { code: 'VIC', name: 'Victoria' },
  { code: 'QLD', name: 'Queensland' },
  { code: 'SA', name: 'South Australia' },
  { code: 'WA', name: 'Western Australia' },
  { code: 'TAS', name: 'Tasmania' },
  { code: 'NT', name: 'Northern Territory' },
  { code: 'ACT', name: 'Australian Capital Territory' },
] as const

const form = reactive({
  first_name: '',
  last_name: '',
  email: '',
  finance_amount: '',
  salary: '',
  state: '',
  vehicle_make: '',
  vehicle_model: '',
  vehicle_year: '',
})

const fieldErrors = ref<Record<string, string>>({})
const formError = ref('')
const submitting = ref(false)
const shakeKey = ref(0)

const isEdit = computed(() => props.lead != null)
const hasErrors = computed(() => Object.keys(fieldErrors.value).length > 0)
const submitLabel = computed(() => {
  if (submitting.value) return 'Saving…'
  return isEdit.value ? 'Save changes' : 'Add lead'
})

function fillFromLead(lead: Lead | null | undefined) {
  if (!lead) {
    form.first_name = ''
    form.last_name = ''
    form.email = ''
    form.finance_amount = ''
    form.salary = ''
    form.state = ''
    form.vehicle_make = ''
    form.vehicle_model = ''
    form.vehicle_year = ''
    return
  }
  form.first_name = lead.first_name
  form.last_name = lead.last_name
  form.email = lead.email
  form.finance_amount = String(lead.finance_amount)
  form.salary = lead.salary === null ? '' : String(lead.salary)
  form.state = lead.state
  form.vehicle_make = lead.vehicle?.make ?? ''
  form.vehicle_model = lead.vehicle?.model ?? ''
  form.vehicle_year = lead.vehicle?.year === null || lead.vehicle?.year === undefined
    ? ''
    : String(lead.vehicle.year)
}

watch(
  () => props.lead,
  (lead) => {
    fillFromLead(lead)
    fieldErrors.value = {}
    formError.value = ''
  },
  { immediate: true },
)

function parseOptionalNumber(raw: string): number | null {
  const t = raw.trim()
  if (t === '') return null
  const n = Number(t)
  return Number.isFinite(n) ? n : NaN
}

function buildRequest(): CreateLeadRequest {
  const finance = parseOptionalNumber(form.finance_amount)
  const salary = parseOptionalNumber(form.salary)
  const year = parseOptionalNumber(form.vehicle_year)

  const make = form.vehicle_make.trim()
  const model = form.vehicle_model.trim()
  const hasVehicle = make !== '' || model !== '' || form.vehicle_year.trim() !== ''

  return {
    first_name: form.first_name.trim(),
    last_name: form.last_name.trim(),
    email: form.email.trim(),
    finance_amount: finance === null ? null : finance,
    salary,
    state: form.state.trim(),
    vehicle: hasVehicle
      ? {
          make,
          model: model === '' ? null : model,
          year: year === null || Number.isNaN(year) ? null : Math.trunc(year),
        }
      : null,
  }
}

function errorFor(...keys: string[]): string {
  for (const k of keys) {
    if (fieldErrors.value[k]) return fieldErrors.value[k]
  }
  return ''
}

function isInvalid(...keys: string[]): boolean {
  return errorFor(...keys) !== ''
}

function clearFieldError(...keys: string[]) {
  const next = { ...fieldErrors.value }
  for (const k of keys) delete next[k]
  fieldErrors.value = next
}

async function onSubmit() {
  fieldErrors.value = {}
  formError.value = ''
  submitting.value = true
  try {
    const body = buildRequest()
    if (props.lead) {
      await updateLead(props.lead.id, body)
    } else {
      await createLead(body)
    }
    emit('saved')
  } catch (err) {
    if (err instanceof ApiError && err.errors) {
      fieldErrors.value = err.errors
      shakeKey.value += 1
      await nextTick()
      const firstInvalid = document.querySelector<HTMLElement>('[aria-invalid="true"]')
      firstInvalid?.focus()
    } else if (err instanceof ApiError) {
      formError.value = err.message
    } else {
      formError.value = 'Something went wrong. Please try again.'
    }
  } finally {
    submitting.value = false
  }
}

function onClear() {
  if (submitting.value) return
  fillFromLead(props.lead)
  fieldErrors.value = {}
  formError.value = ''
}
</script>

<template>
  <form class="grid gap-6" novalidate @submit.prevent="onSubmit">
    <div
      v-if="formError"
      class="alert-banner"
      role="alert"
    >
      {{ formError }}
    </div>
    <div
      v-else-if="hasErrors"
      class="alert-banner"
      role="alert"
    >
      Please fix the highlighted fields and try again.
    </div>

    <div :key="shakeKey" class="grid gap-6">
      <fieldset class="grid gap-4 border-0 p-0">
        <legend class="mb-1 font-display text-sm font-semibold tracking-tight text-ink">
          Contact
        </legend>

        <div class="grid gap-4 sm:grid-cols-2">
          <div class="field">
            <label class="field-label" for="first_name">First name <span class="req" aria-hidden="true">*</span></label>
            <input
              id="first_name"
              v-model="form.first_name"
              class="field-input"
              type="text"
              autocomplete="given-name"
              aria-required="true"
              :aria-invalid="isInvalid('first_name')"
              aria-describedby="first_name-help"
              :disabled="submitting"
              @input="clearFieldError('first_name')"
            >
            <p id="first_name-help" class="field-error" role="status">
              {{ errorFor('first_name') }}
            </p>
          </div>

          <div class="field">
            <label class="field-label" for="last_name">Last name <span class="req" aria-hidden="true">*</span></label>
            <input
              id="last_name"
              v-model="form.last_name"
              class="field-input"
              type="text"
              autocomplete="family-name"
              aria-required="true"
              :aria-invalid="isInvalid('last_name')"
              aria-describedby="last_name-help"
              :disabled="submitting"
              @input="clearFieldError('last_name')"
            >
            <p id="last_name-help" class="field-error" role="status">
              {{ errorFor('last_name') }}
            </p>
          </div>
        </div>

        <div class="field">
          <label class="field-label" for="email">Email <span class="req" aria-hidden="true">*</span></label>
          <input
            id="email"
            v-model="form.email"
            class="field-input"
            type="email"
            autocomplete="email"
            aria-required="true"
            :aria-invalid="isInvalid('email')"
            aria-describedby="email-help"
            :disabled="submitting"
            @input="clearFieldError('email')"
          >
          <p id="email-help" :class="errorFor('email') ? 'field-error' : 'field-hint'" role="status">
            {{ errorFor('email') || 'Must be unique across leads.' }}
          </p>
        </div>
      </fieldset>

      <fieldset class="grid gap-4 border-0 p-0">
        <legend class="mb-1 font-display text-sm font-semibold tracking-tight text-ink">
          Finance
        </legend>

        <div class="grid gap-4 sm:grid-cols-3">
          <div class="field">
            <label class="field-label" for="finance_amount">Finance amount <span class="req" aria-hidden="true">*</span></label>
            <input
              id="finance_amount"
              v-model="form.finance_amount"
              class="field-input"
              type="text"
              inputmode="decimal"
              placeholder="e.g. 38500"
              aria-required="true"
              :aria-invalid="isInvalid('finance_amount')"
              aria-describedby="finance_amount-help"
              :disabled="submitting"
              @input="clearFieldError('finance_amount')"
            >
            <p
              id="finance_amount-help"
              :class="errorFor('finance_amount') ? 'field-error' : 'field-hint'"
              role="status"
            >
              {{ errorFor('finance_amount') || 'AUD, greater than 0.' }}
            </p>
          </div>

          <div class="field">
            <label class="field-label" for="salary">Salary <span class="font-normal text-muted">(optional)</span></label>
            <input
              id="salary"
              v-model="form.salary"
              class="field-input"
              type="text"
              inputmode="decimal"
              :aria-invalid="isInvalid('salary')"
              aria-describedby="salary-help"
              :disabled="submitting"
              @input="clearFieldError('salary')"
            >
            <p id="salary-help" :class="errorFor('salary') ? 'field-error' : 'field-hint'" role="status">
              {{ errorFor('salary') || 'Leave blank if unknown.' }}
            </p>
          </div>

          <div class="field">
            <label class="field-label" for="state">State <span class="req" aria-hidden="true">*</span></label>
            <select
              id="state"
              v-model="form.state"
              class="field-input field-select"
              autocomplete="address-level1"
              aria-required="true"
              :aria-invalid="isInvalid('state')"
              aria-describedby="state-help"
              :disabled="submitting"
              @change="clearFieldError('state')"
            >
              <option disabled value="">Select a state</option>
              <option v-for="s in AU_STATES" :key="s.code" :value="s.code">
                {{ s.code }} — {{ s.name }}
              </option>
            </select>
            <p id="state-help" :class="errorFor('state') ? 'field-error' : 'field-hint'" role="status">
              {{ errorFor('state') || 'Australian state or territory.' }}
            </p>
          </div>
        </div>
      </fieldset>

      <fieldset class="grid gap-4 rounded-xl border border-border/80 bg-surface-elevated/70 p-4 sm:p-5">
        <legend class="px-1 font-display text-sm font-semibold tracking-tight text-ink">
          Vehicle <span class="font-sans font-normal text-muted">(optional)</span>
        </legend>
        <p class="text-sm text-muted">If you add a model or year, make is required.</p>

        <div class="grid gap-4 sm:grid-cols-3">
          <div class="field">
            <label class="field-label" for="vehicle_make">Make</label>
            <input
              id="vehicle_make"
              v-model="form.vehicle_make"
              class="field-input"
              type="text"
              :aria-invalid="isInvalid('vehicle.make')"
              aria-describedby="vehicle_make-help"
              :disabled="submitting"
              @input="clearFieldError('vehicle.make')"
            >
            <p id="vehicle_make-help" class="field-error" role="status">
              {{ errorFor('vehicle.make') }}
            </p>
          </div>

          <div class="field">
            <label class="field-label" for="vehicle_model">Model</label>
            <input
              id="vehicle_model"
              v-model="form.vehicle_model"
              class="field-input"
              type="text"
              :aria-invalid="isInvalid('vehicle.model')"
              aria-describedby="vehicle_model-help"
              :disabled="submitting"
              @input="clearFieldError('vehicle.model')"
            >
            <p id="vehicle_model-help" class="field-error" role="status">
              {{ errorFor('vehicle.model') }}
            </p>
          </div>

          <div class="field">
            <label class="field-label" for="vehicle_year">Year</label>
            <input
              id="vehicle_year"
              v-model="form.vehicle_year"
              class="field-input"
              type="text"
              inputmode="numeric"
              placeholder="2024"
              :aria-invalid="isInvalid('vehicle.year')"
              aria-describedby="vehicle_year-help"
              :disabled="submitting"
              @input="clearFieldError('vehicle.year')"
            >
            <p
              id="vehicle_year-help"
              :class="errorFor('vehicle.year') ? 'field-error' : 'field-hint'"
              role="status"
            >
              {{ errorFor('vehicle.year') || '1990 to next year.' }}
            </p>
          </div>
        </div>
      </fieldset>
    </div>

    <div class="form-actions">
      <button class="btn btn-primary" type="submit" :disabled="submitting">
        <span v-if="submitting" class="spinner" aria-hidden="true" />
        <span>{{ submitLabel }}</span>
      </button>
      <button class="btn btn-secondary" type="button" :disabled="submitting" @click="onClear">
        Clear
      </button>
    </div>
  </form>
</template>
