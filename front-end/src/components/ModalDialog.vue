<template>
  <div v-if="open" class="overlay" @click.self="emit('close')">
    <div
      class="modal"
      :class="{
        'modal--compact': compact,
        'modal--wide': wide,
        'modal--full': full,
      }"
      role="dialog"
      :aria-labelledby="titleId"
      aria-modal="true"
    >
      <div class="modal__head">
        <h2 :id="titleId">{{ title }}</h2>
        <button
          v-if="closable"
          class="btn btn-icon modal__close"
          type="button"
          aria-label="Close"
          @click="emit('close')"
        >
          <i class="ti ti-x" aria-hidden="true" />
        </button>
      </div>
      <div class="modal__body">
        <slot />
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  open: { type: Boolean, default: false },
  title: { type: String, required: true },
  titleId: { type: String, default: 'modal-title' },
  compact: { type: Boolean, default: false },
  wide: { type: Boolean, default: false },
  full: { type: Boolean, default: false },
  closable: { type: Boolean, default: true },
})

const emit = defineEmits(['close'])
</script>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  z-index: 200;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.25rem;
  background: rgba(0, 0, 0, 0.55);
}

.modal {
  background: var(--panel);
  border: 0.5px solid var(--hairline);
  border-radius: 12px;
  padding: 1.25rem;
  width: 340px;
  max-width: min(920px, 96vw);
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.modal--compact {
  width: 320px;
}

.modal--wide {
  width: 820px;
}

.modal--full {
  width: min(1480px, 98vw);
  max-width: 98vw;
  height: min(960px, 96vh);
  padding: 0;
  overflow: hidden;
}

.modal__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 1rem;
}

.modal--full .modal__head {
  padding: 8px 12px 0;
  margin-bottom: 0;
}

.modal--full .modal__head h2 {
  font-size: 14px;
  font-weight: 600;
}

.modal__close {
  flex-shrink: 0;
}

.modal__body {
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
}

.modal--full .modal__body {
  padding: 8px 12px 12px;
}

.modal--compact h2 {
  margin-bottom: 0;
}

h2 {
  margin: 0 0 1rem;
}

.modal--full h2,
.modal__head h2 {
  margin: 0;
}
</style>
