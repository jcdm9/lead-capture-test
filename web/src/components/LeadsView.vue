<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { deleteLead, fetchLeads } from '../api'
import type { Lead } from '../types'
import ConfirmDialog from './ConfirmDialog.vue'
import CsvImportDropzone from './CsvImportDropzone.vue'
import type { ImportResult } from '../types'

const emit = defineEmits<{
  edit: [lead: Lead]
}>()

type DeleteTarget =
  | { mode: 'single'; lead: Lead }
  | { mode: 'bulk'; ids: number[] }

type SortKey = 'name' | 'finance' | 'salary' | 'state' | 'vehicle' | 'created'
type SortDir = 'asc' | 'desc'
type Align = 'text' | 'num' | 'code'

interface Column {
  key: SortKey
  label: string
  align: Align
}

const columns: Column[] = [
  { key: 'name', label: 'Lead', align: 'text' },
  { key: 'finance', label: 'Finance', align: 'num' },
  { key: 'salary', label: 'Salary', align: 'num' },
  { key: 'state', label: 'State', align: 'code' },
  { key: 'vehicle', label: 'Vehicle', align: 'text' },
  { key: 'created', label: 'Created', align: 'num' },
]

const PAGE_SIZE_OPTIONS = [10, 25, 50, 100] as const

const leads = ref<Lead[]>([])
const loading = ref(false)
const error = ref('')
const selected = ref<Set<number>>(new Set())
const sortKey = ref<SortKey>('created')
const sortDir = ref<SortDir>('desc')
const deletingId = ref<number | null>(null)
const bulkDeleting = ref(false)
const actionError = ref('')
const pageSize = ref<(typeof PAGE_SIZE_OPTIONS)[number]>(10)
const page = ref(1)
const confirmOpen = ref(false)
const deleteTarget = ref<DeleteTarget | null>(null)
const importReport = ref<ImportResult | null>(null)

const confirmBusy = computed(() => bulkDeleting.value || deletingId.value !== null)

const confirmTitle = computed(() => {
  if (!deleteTarget.value) return 'Delete lead?'
  if (deleteTarget.value.mode === 'single') return 'Delete this lead?'
  return 'Delete selected leads?'
})

const confirmDetail = computed(() => {
  if (!deleteTarget.value) return ''
  if (deleteTarget.value.mode === 'single') {
    const lead = deleteTarget.value.lead
    return `${fullName(lead)} · ${lead.email}`
  }
  const n = deleteTarget.value.ids.length
  return `${n} lead${n === 1 ? '' : 's'} selected`
})

const confirmMessage = computed(() => {
  if (!deleteTarget.value) return ''
  if (deleteTarget.value.mode === 'single') {
    return 'This will permanently remove the lead from your records. This cannot be undone.'
  }
  return 'This will permanently remove the selected leads from your records. This cannot be undone.'
})

const confirmLabel = computed(() => {
  if (!deleteTarget.value || deleteTarget.value.mode === 'single') return 'Delete'
  return `Delete ${deleteTarget.value.ids.length}`
})

const stateTone: Record<string, string> = {
  NSW: 'sky',
  VIC: 'violet',
  QLD: 'amber',
  SA: 'rose',
  WA: 'teal',
  TAS: 'slate',
  NT: 'amber',
  ACT: 'sky',
}

async function load(options?: { keepSelection?: boolean }) {
  loading.value = true
  error.value = ''
  try {
    leads.value = await fetchLeads()
    if (!options?.keepSelection) {
      selected.value = new Set()
    } else {
      const existing = new Set(leads.value.map((l) => l.id))
      selected.value = new Set([...selected.value].filter((id) => existing.has(id)))
    }
  } catch {
    error.value = 'Failed to load leads'
  } finally {
    loading.value = false
  }
}

function formatMoney(n: number | null): string {
  if (n === null) return '—'
  return n.toLocaleString('en-AU', { style: 'currency', currency: 'AUD' })
}

function formatDate(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  })
}

function fullName(lead: Lead): string {
  return `${lead.first_name} ${lead.last_name}`
}

function vehiclePrimary(lead: Lead): string {
  if (!lead.vehicle) return '—'
  return lead.vehicle.make || '—'
}

function vehicleSecondary(lead: Lead): string {
  if (!lead.vehicle) return 'No vehicle'
  const bits: string[] = []
  if (lead.vehicle.model) bits.push(lead.vehicle.model)
  if (lead.vehicle.year !== null) bits.push(String(lead.vehicle.year))
  return bits.length ? bits.join(' · ') : 'Model not set'
}

function toneFor(state: string): string {
  return stateTone[state] ?? 'slate'
}

function sortValue(lead: Lead, key: SortKey): string | number {
  switch (key) {
    case 'name':
      return fullName(lead).toLowerCase()
    case 'finance':
      return lead.finance_amount
    case 'salary':
      return lead.salary ?? -1
    case 'state':
      return lead.state
    case 'vehicle':
      return vehiclePrimary(lead).toLowerCase()
    case 'created':
      return new Date(lead.created_at).getTime()
  }
}

const sortedLeads = computed(() => {
  const rows = [...leads.value]
  const dir = sortDir.value === 'asc' ? 1 : -1
  rows.sort((a, b) => {
    const av = sortValue(a, sortKey.value)
    const bv = sortValue(b, sortKey.value)
    if (av < bv) return -1 * dir
    if (av > bv) return 1 * dir
    return a.id - b.id
  })
  return rows
})

const totalRows = computed(() => sortedLeads.value.length)
const totalPages = computed(() => Math.max(1, Math.ceil(totalRows.value / pageSize.value)))

const pagedLeads = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return sortedLeads.value.slice(start, start + pageSize.value)
})

const rangeLabel = computed(() => {
  if (totalRows.value === 0) return '0 of 0'
  const start = (page.value - 1) * pageSize.value + 1
  const end = Math.min(page.value * pageSize.value, totalRows.value)
  return `${start}–${end} of ${totalRows.value}`
})

const allPageSelected = computed(
  () => pagedLeads.value.length > 0 && pagedLeads.value.every((l) => selected.value.has(l.id)),
)

const selectedCount = computed(() => selected.value.size)

const allLeadsSelected = computed(
  () => totalRows.value > 0 && selectedCount.value === totalRows.value,
)

watch([totalPages, pageSize], () => {
  if (page.value > totalPages.value) page.value = totalPages.value
})

watch(pageSize, () => {
  page.value = 1
})

function cellClass(align: Align): string {
  if (align === 'num') return 'cell-num'
  if (align === 'code') return 'cell-code'
  return 'cell-text'
}

function sortBtnStyle(align: Align): string | undefined {
  if (align === 'num') return 'margin-left: auto'
  if (align === 'code') return 'margin-inline: auto'
  return undefined
}

function toggleSort(key: SortKey) {
  if (sortKey.value === key) {
    sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortKey.value = key
    sortDir.value = key === 'name' || key === 'state' || key === 'vehicle' ? 'asc' : 'desc'
  }
  page.value = 1
}

function toggleRow(id: number) {
  const next = new Set(selected.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selected.value = next
}

function toggleAllVisible() {
  const next = new Set(selected.value)
  if (allPageSelected.value) {
    for (const lead of pagedLeads.value) next.delete(lead.id)
  } else {
    for (const lead of pagedLeads.value) next.add(lead.id)
  }
  selected.value = next
}

function selectAllLeads() {
  selected.value = new Set(sortedLeads.value.map((l) => l.id))
}

function clearSelection() {
  selected.value = new Set()
}

function onEdit(lead: Lead) {
  emit('edit', lead)
}

function openDeleteConfirm(lead: Lead) {
  deleteTarget.value = { mode: 'single', lead }
  confirmOpen.value = true
}

function openBulkDeleteConfirm() {
  const ids = [...selected.value]
  if (ids.length === 0) return
  deleteTarget.value = { mode: 'bulk', ids }
  confirmOpen.value = true
}

function closeConfirm() {
  if (confirmBusy.value) return
  confirmOpen.value = false
  deleteTarget.value = null
}

async function executeDelete() {
  if (!deleteTarget.value) return
  actionError.value = ''

  if (deleteTarget.value.mode === 'single') {
    const lead = deleteTarget.value.lead
    deletingId.value = lead.id
    try {
      await deleteLead(lead.id)
      selected.value.delete(lead.id)
      selected.value = new Set(selected.value)
      confirmOpen.value = false
      deleteTarget.value = null
      await load()
    } catch {
      actionError.value = `Failed to delete ${fullName(lead)}.`
    } finally {
      deletingId.value = null
    }
    return
  }

  const ids = deleteTarget.value.ids
  bulkDeleting.value = true
  const failed: number[] = []
  try {
    for (const id of ids) {
      try {
        await deleteLead(id)
      } catch {
        failed.push(id)
      }
    }
    if (failed.length > 0) {
      selected.value = new Set(failed)
      actionError.value = `Failed to delete ${failed.length} lead${failed.length === 1 ? '' : 's'}.`
      await load({ keepSelection: true })
    } else {
      selected.value = new Set()
      await load()
    }
    confirmOpen.value = false
    deleteTarget.value = null
  } finally {
    bulkDeleting.value = false
  }
}

function goToPage(next: number) {
  page.value = Math.min(totalPages.value, Math.max(1, next))
}

async function onImported(result: ImportResult) {
  importReport.value = result
  await load({ keepSelection: true })
}

function dismissImportReport() {
  importReport.value = null
}

defineExpose({ reload: load })

onMounted(load)
</script>

<template>
  <div>
    <div v-if="loading" class="grid gap-2" aria-busy="true" aria-live="polite">
      <div class="skeleton-row" />
      <div class="skeleton-row" />
      <div class="skeleton-row" />
      <p class="sr-only">Loading leads</p>
    </div>

    <div v-else-if="error" class="alert-banner" role="alert">
      {{ error }}
    </div>

    <template v-else>
      <div v-if="actionError" class="alert-banner mb-4" role="alert">
        {{ actionError }}
      </div>

      <div
        v-if="importReport"
        class="csv-import-result mb-4"
        role="status"
        aria-live="polite"
      >
        <div class="csv-import-result-header">
          <p>
            Import finished:
            <strong>{{ importReport.imported }}</strong> imported,
            <strong>{{ importReport.skipped }}</strong> skipped.
          </p>
          <button
            type="button"
            class="btn btn-secondary"
            aria-label="Dismiss import report"
            @click="dismissImportReport"
          >
            Dismiss
          </button>
        </div>
        <ul v-if="importReport.skips.length > 0" class="csv-skip-list">
          <li v-for="skip in importReport.skips" :key="skip.line">
            Line {{ skip.line }}: {{ skip.reason }}
          </li>
        </ul>
      </div>

      <CsvImportDropzone v-if="leads.length === 0" @imported="onImported" />

      <div v-else>
        <div
          v-if="selectedCount > 0"
          class="selection-bar"
          role="status"
          aria-live="polite"
        >
          <div class="selection-bar-copy">
            <p class="selection-count">
              {{ selectedCount }} selected
            </p>
            <button
              v-if="allPageSelected && !allLeadsSelected"
              type="button"
              class="selection-link"
              @click="selectAllLeads"
            >
              Select all {{ totalRows }} leads
            </button>
            <button
              v-else-if="allLeadsSelected"
              type="button"
              class="selection-link"
              @click="clearSelection"
            >
              Clear selection
            </button>
          </div>
          <div class="selection-bar-actions">
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="bulkDeleting"
              @click="clearSelection"
            >
              Clear
            </button>
            <button
              type="button"
              class="btn btn-danger"
              :disabled="bulkDeleting"
              @click="openBulkDeleteConfirm"
            >
              <span v-if="bulkDeleting" class="spinner spinner-danger" aria-hidden="true" />
              <span>{{ bulkDeleting ? 'Deleting…' : `Delete selected (${selectedCount})` }}</span>
            </button>
          </div>
        </div>

        <div class="data-table-wrap">
          <table class="data-table">
            <thead>
              <tr>
                <th class="cell-text" scope="col" style="width: 2.75rem">
                  <input
                    class="row-check"
                    type="checkbox"
                    :checked="allPageSelected"
                    :aria-label="allPageSelected ? 'Deselect page' : 'Select page'"
                    @change="toggleAllVisible"
                  >
                </th>
                <th
                  v-for="col in columns"
                  :key="col.key"
                  :class="cellClass(col.align)"
                  scope="col"
                >
                  <button
                    type="button"
                    class="sort-btn"
                    :class="{ 'is-active': sortKey === col.key }"
                    :style="sortBtnStyle(col.align)"
                    :aria-sort="sortKey === col.key ? (sortDir === 'asc' ? 'ascending' : 'descending') : 'none'"
                    @click="toggleSort(col.key)"
                  >
                    {{ col.label }}
                    <svg class="sort-icon" viewBox="0 0 16 16" aria-hidden="true">
                      <path
                        v-if="sortKey === col.key && sortDir === 'asc'"
                        fill="currentColor"
                        d="M8 3.5 3.5 9h9L8 3.5Z"
                      />
                      <path
                        v-else-if="sortKey === col.key && sortDir === 'desc'"
                        fill="currentColor"
                        d="M8 12.5 12.5 7h-9L8 12.5Z"
                      />
                      <path
                        v-else
                        fill="currentColor"
                        d="M8 2.75 4.5 6.5h7L8 2.75Zm0 10.5 3.5-3.75h-7L8 13.25Z"
                      />
                    </svg>
                  </button>
                </th>
                <th class="cell-actions" scope="col">
                  <span class="sr-only">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="lead in pagedLeads"
                :key="lead.id"
                :class="{ 'is-selected': selected.has(lead.id) }"
              >
                <td class="cell-text">
                  <input
                    class="row-check"
                    type="checkbox"
                    :checked="selected.has(lead.id)"
                    :aria-label="`Select ${fullName(lead)}`"
                    @change="toggleRow(lead.id)"
                  >
                </td>
                <td class="cell-text">
                  <div class="entity-name">{{ fullName(lead) }}</div>
                  <div class="entity-sub">{{ lead.email }}</div>
                </td>
                <td class="cell-num">{{ formatMoney(lead.finance_amount) }}</td>
                <td class="cell-num">{{ formatMoney(lead.salary) }}</td>
                <td class="cell-code">
                  <span class="status-chip" :data-tone="toneFor(lead.state)">{{ lead.state }}</span>
                </td>
                <td class="cell-text">
                  <div class="entity-name" style="font-weight: 600">{{ vehiclePrimary(lead) }}</div>
                  <div class="entity-sub">{{ vehicleSecondary(lead) }}</div>
                </td>
                <td class="cell-num">{{ formatDate(lead.created_at) }}</td>
                <td class="cell-actions">
                  <button
                    type="button"
                    class="icon-btn"
                    title="Edit lead"
                    :aria-label="`Edit ${fullName(lead)}`"
                    @click="onEdit(lead)"
                  >
                    <svg viewBox="0 0 16 16" aria-hidden="true">
                      <path
                        fill="currentColor"
                        d="M11.3 1.7a1.5 1.5 0 0 1 2.1 2.1L5.6 11.6 2 12.8l1.2-3.6 8.1-7.5Zm.7 1.1-8.1 7.5-.5 1.4 1.4-.5 8.1-7.5-.9-.9Z"
                      />
                    </svg>
                  </button>
                  <button
                    type="button"
                    class="icon-btn"
                    title="Delete lead"
                    :aria-label="`Delete ${fullName(lead)}`"
                    :disabled="deletingId === lead.id"
                    @click="openDeleteConfirm(lead)"
                  >
                    <svg viewBox="0 0 16 16" aria-hidden="true">
                      <path
                        fill="currentColor"
                        d="M6 2h4l.5 1H14v1H2V3h3.5L6 2Zm1 4v6H6V6h1Zm3 0v6H9V6h1Zm-6 .2L4.2 13.1A1 1 0 0 0 5.2 14h5.6a1 1 0 0 0 1-.9L13 6.2H4Z"
                      />
                    </svg>
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="table-footer" aria-label="Table pagination">
          <label class="page-size">
            <span>Rows per page</span>
            <select
              v-model.number="pageSize"
              class="field-input field-select page-size-select"
              aria-label="Rows per page"
            >
              <option v-for="size in PAGE_SIZE_OPTIONS" :key="size" :value="size">
                {{ size }}
              </option>
            </select>
          </label>

          <p class="page-range" aria-live="polite">{{ rangeLabel }}</p>

          <div class="page-nav">
            <button
              type="button"
              class="btn btn-secondary page-nav-btn"
              :disabled="page <= 1"
              aria-label="Previous page"
              @click="goToPage(page - 1)"
            >
              Previous
            </button>
            <span class="page-indicator">Page {{ page }} of {{ totalPages }}</span>
            <button
              type="button"
              class="btn btn-secondary page-nav-btn"
              :disabled="page >= totalPages"
              aria-label="Next page"
              @click="goToPage(page + 1)"
            >
              Next
            </button>
          </div>
        </div>
      </div>
    </template>

    <ConfirmDialog
      :open="confirmOpen"
      :title="confirmTitle"
      :detail="confirmDetail"
      :message="confirmMessage"
      :confirm-label="confirmLabel"
      :busy="confirmBusy"
      @confirm="executeDelete"
      @cancel="closeConfirm"
    />
  </div>
</template>
