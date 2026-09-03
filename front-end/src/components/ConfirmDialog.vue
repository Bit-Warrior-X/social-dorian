<template>
  <ModalDialog :open="open" :title="title" :title-id="titleId" compact @close="emit('close')">
    <p class="copy">{{ message }}</p>
    <div class="actions">
      <button class="btn" type="button" @click="emit('close')">Cancel</button>
      <button class="btn btn-danger" type="button" :disabled="saving" @click="emit('confirm')">
        {{ saving ? savingLabel : confirmLabel }}
      </button>
    </div>
  </ModalDialog>
</template>

<script setup>
import ModalDialog from './ModalDialog.vue'

defineProps({
  open: { type: Boolean, default: false },
  title: { type: String, required: true },
  titleId: { type: String, default: 'confirm-title' },
  message: { type: String, required: true },
  confirmLabel: { type: String, default: 'Confirm' },
  savingLabel: { type: String, default: 'Working…' },
  saving: { type: Boolean, default: false },
})

const emit = defineEmits(['close', 'confirm'])
</script>

<style scoped>
.copy {
  font-size: 14px;
  color: var(--text-dim);
  margin: 0 0 1rem;
  line-height: 1.6;
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
