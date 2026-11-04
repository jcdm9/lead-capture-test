<script setup lang="ts">
import { ref } from 'vue'
import { importLeadsCSV } from '../api'
import { ApiError, type ImportResult } from '../types'

const emit = defineEmits<{
  imported: [result: ImportResult]
}>()

const inputEl = ref<HTMLInputElement | null>(null)
const dragging = ref(false)
const uploading = ref(false)
const error = ref('')
const lastResult = ref<ImportResult | null>(null)

function isCSV(file: File): boolean {
  return file.name.toLowerCase().endsWith('.csv')
}

function openPicker() {
  if (uploading.value) return
  inputEl.value?.click()
}

function onDragOver(e: DragEvent) {
  e.preventDefault()
  if (uploading.value) return
  dragging.value = true
}

function onDragLeave(e: DragEvent) {
  e.preventDefault()
  dragging.value = false
}

async function onDrop(e: DragEvent) {
  e.preventDefault()
  dragging.value = false
  if (uploading.value) return
  const file = e.dataTransfer?.files?.[0]
  if (file) await handleFile(file)
}

async function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (file) await handleFile(file)
}

async function handleFile(file: File) {
  error.value = ''
  lastResult.value = null

  if (!isCSV(file)) {
    error.value = 'Only .csv files are accepted.'
    return
  }

  uploading.value = true
  try {
    const result = await importLeadsCSV(file)
    lastResult.value = result
    emit('imported', result)
  } catch (err) {
    if (err instanceof ApiError) {
      error.value = err.message
    } else {
      error.value = 'Failed to import CSV.'
    }
  } finally {
    uploading.value = false
  }
}
</script>

<template>
  <div
    class="empty-leads"
    :class="{ 'is-dragging': dragging, 'is-busy': uploading }"
    @dragenter="onDragOver"
    @dragover="onDragOver"
    @dragleave="onDragLeave"
    @drop="onDrop"
  >
    <input
      ref="inputEl"
      class="sr-only"
      type="file"
      accept=".csv,text/csv"
      :disabled="uploading"
      @change="onFileChange"
    >

    <p class="empty-leads-title">
      {{ uploading ? 'Importing CSV…' : 'No leads yet' }}
    </p>
    <p class="empty-leads-hint">
      Add a lead manually, or drag and drop a <strong>.csv</strong> file here.
      You can also browse for a file.
    </p>

    <button
      class="btn btn-secondary"
      type="button"
      :disabled="uploading"
      @click="openPicker"
    >
      {{ uploading ? 'Uploading…' : 'Browse CSV' }}
    </button>

    <div v-if="error" class="alert-banner empty-leads-feedback" role="alert">
      {{ error }}
    </div>

    <div
      v-if="lastResult"
      class="csv-import-result empty-leads-feedback"
      role="status"
      aria-live="polite"
    >
      <p>
        Imported <strong>{{ lastResult.imported }}</strong>,
        skipped <strong>{{ lastResult.skipped }}</strong>.
      </p>
      <ul v-if="lastResult.skips.length > 0" class="csv-skip-list">
        <li v-for="skip in lastResult.skips" :key="skip.line">
          Line {{ skip.line }}: {{ skip.reason }}
        </li>
      </ul>
    </div>
  </div>
</template>
