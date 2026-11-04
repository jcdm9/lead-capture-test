<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import LeadForm from './components/LeadForm.vue'
import LeadsView from './components/LeadsView.vue'
import type { Lead } from './types'

const leadsView = ref<{ reload: () => Promise<void> } | null>(null)
const panelOpen = ref(false)
const editingLead = ref<Lead | null>(null)

const panelTitle = computed(() => (editingLead.value ? 'Edit lead' : 'Add a lead'))
const panelSubtitle = computed(() =>
  editingLead.value
    ? 'Update details, then save. Vehicle can stay blank.'
    : 'Required fields are marked. Vehicle can stay blank.',
)

function openCreate() {
  editingLead.value = null
  panelOpen.value = true
}

function openEdit(lead: Lead) {
  editingLead.value = lead
  panelOpen.value = true
}

function closePanel() {
  panelOpen.value = false
  editingLead.value = null
}

async function onSaved() {
  closePanel()
  await leadsView.value?.reload()
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && panelOpen.value) {
    closePanel()
  }
}

watch(panelOpen, (open) => {
  document.body.style.overflow = open ? 'hidden' : ''
})

onMounted(() => window.addEventListener('keydown', onKeydown))
onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown)
  document.body.style.overflow = ''
})
</script>

<template>
  <div class="page-shell text-ink">
    <div class="mx-auto max-w-6xl px-5 py-10 sm:px-8 sm:py-14">
      <header class="mb-8 flex flex-wrap items-end justify-between gap-4">
        <div class="max-w-xl">
          <p class="mb-3 text-xs font-semibold uppercase tracking-[0.18em] text-accent">Lead capture</p>
          <h1 class="brand-mark text-4xl font-semibold text-ink sm:text-5xl">Leads</h1>
          <p class="mt-3 max-w-md text-base leading-relaxed text-muted">
            Browse imported and staff-entered leads. Add or edit from the actions above the table.
          </p>
        </div>
        <button class="btn btn-primary" type="button" @click="openCreate">
          Add lead
        </button>
      </header>

      <main>
        <section class="surface-card p-6 sm:p-8" aria-labelledby="leads-heading">
          <div class="mb-5 flex flex-wrap items-end justify-between gap-3">
            <div>
              <h2 id="leads-heading" class="font-display text-xl font-semibold tracking-tight text-ink">
                All leads
              </h2>
              <p class="mt-1 text-sm text-muted">Select rows, sort columns, or edit and delete inline.</p>
            </div>
          </div>
          <LeadsView ref="leadsView" @edit="openEdit" />
        </section>
      </main>
    </div>

    <Teleport to="body">
      <div
        v-if="panelOpen"
        class="modal-root"
        role="dialog"
        aria-modal="true"
        aria-labelledby="lead-panel-title"
      >
        <button class="modal-backdrop" type="button" aria-label="Close dialog" @click="closePanel" />
        <div class="modal-panel">
          <div class="mb-6 flex items-start justify-between gap-4">
            <div>
              <h2 id="lead-panel-title" class="font-display text-xl font-semibold tracking-tight text-ink">
                {{ panelTitle }}
              </h2>
              <p class="mt-1 text-sm text-muted">{{ panelSubtitle }}</p>
            </div>
            <button class="icon-btn" type="button" title="Close" aria-label="Close" @click="closePanel">
              <svg viewBox="0 0 16 16" aria-hidden="true">
                <path
                  fill="currentColor"
                  d="M3.2 3.2a.75.75 0 0 1 1.06 0L8 6.94l3.74-3.74a.75.75 0 1 1 1.06 1.06L9.06 8l3.74 3.74a.75.75 0 1 1-1.06 1.06L8 9.06l-3.74 3.74a.75.75 0 1 1-1.06-1.06L6.94 8 3.2 4.26a.75.75 0 0 1 0-1.06Z"
                />
              </svg>
            </button>
          </div>
          <LeadForm :lead="editingLead" @saved="onSaved" />
        </div>
      </div>
    </Teleport>
  </div>
</template>
