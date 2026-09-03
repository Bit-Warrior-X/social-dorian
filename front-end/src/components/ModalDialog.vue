<template>
  <div v-if="open" class="overlay" @click.self="emit('close')">
    <div
      class="modal"
      :class="{ 'modal--compact': compact, 'modal--wide': wide }"
      role="dialog"
      :aria-labelledby="titleId"
      aria-modal="true"
    >
      <h2 :id="titleId">{{ title }}</h2>
      <slot />
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
}

.modal--compact {
  width: 320px;
}

.modal--wide {
  width: 820px;
}

.modal--compact h2 {
  margin-bottom: 8px;
}

h2 {
  margin: 0 0 1rem;
}
</style>
