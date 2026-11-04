<script setup lang="ts">
import { onMounted, onUnmounted, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    open: boolean
    title: string
    message: string
    detail?: string
    confirmLabel?: string
    cancelLabel?: string
    busy?: boolean
  }>(),
  {
    detail: '',
    confirmLabel: 'Delete',
    cancelLabel: 'Cancel',
    busy: false,
  },
)

const emit = defineEmits<{
  confirm: []
  cancel: []
}>()

function onKeydown(e: KeyboardEvent) {
  if (!props.open || props.busy) return
  if (e.key === 'Escape') emit('cancel')
}

watch(
  () => props.open,
  (open) => {
    document.body.style.overflow = open ? 'hidden' : ''
  },
)

onMounted(() => window.addEventListener('keydown', onKeydown))
onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown)
  document.body.style.overflow = ''
})
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="modal-root confirm-root"
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="confirm-title"
      aria-describedby="confirm-message"
    >
      <button
        class="modal-backdrop"
        type="button"
        aria-label="Dismiss"
        :disabled="busy"
        @click="emit('cancel')"
      />
      <div class="modal-panel confirm-panel">
        <div class="confirm-heading">
          <h2 id="confirm-title" class="confirm-title">{{ title }}</h2>
          <p v-if="detail" class="confirm-detail">{{ detail }}</p>
        </div>

        <p id="confirm-message" class="confirm-message">{{ message }}</p>

        <div class="confirm-actions">
          <button
            class="btn btn-secondary"
            type="button"
            :disabled="busy"
            @click="emit('cancel')"
          >
            {{ cancelLabel }}
          </button>
          <button
            class="btn btn-danger"
            type="button"
            :disabled="busy"
            @click="emit('confirm')"
          >
            <span v-if="busy" class="spinner spinner-danger" aria-hidden="true" />
            <span>{{ busy ? 'Deleting…' : confirmLabel }}</span>
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
