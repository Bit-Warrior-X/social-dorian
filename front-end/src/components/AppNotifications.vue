<template>
  <div class="notify" aria-live="polite" aria-relevant="additions text">
    <TransitionGroup name="notify">
      <div
        v-for="item in notifications"
        :key="item.id"
        class="notify__item"
        :class="'notify__item--' + item.type"
        role="status"
      >
        <div class="notify__icon" aria-hidden="true">
          <i class="ti" :class="iconFor(item.type)" />
        </div>
        <p class="notify__message">{{ item.message }}</p>
        <button
          class="notify__close"
          type="button"
          aria-label="Dismiss notification"
          @click="dismiss(item.id)"
        >
          <i class="ti ti-x" aria-hidden="true" />
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup>
import { useNotify } from '../composables/useNotify'

const { notifications, dismiss } = useNotify()

function iconFor(type) {
  if (type === 'success') return 'ti-circle-check'
  if (type === 'error') return 'ti-alert-circle'
  return 'ti-info-circle'
}
</script>

<style scoped>
.notify {
  position: fixed;
  top: 1rem;
  right: 1rem;
  z-index: 400;
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: min(420px, calc(100vw - 2rem));
  pointer-events: none;
}

.notify__item {
  pointer-events: auto;
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: 10px;
  align-items: start;
  padding: 12px 12px 12px 14px;
  border-radius: 12px;
  border: 0.5px solid var(--hairline);
  background: color-mix(in srgb, var(--panel) 92%, black);
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.35);
  backdrop-filter: blur(8px);
}

.notify__item--success {
  border-color: color-mix(in srgb, var(--viper-500) 45%, transparent);
}

.notify__item--error {
  border-color: color-mix(in srgb, var(--danger) 45%, transparent);
}

.notify__item--info {
  border-color: color-mix(in srgb, var(--gold-500) 40%, transparent);
}

.notify__icon {
  width: 28px;
  height: 28px;
  border-radius: 8px;
  display: grid;
  place-items: center;
  flex-shrink: 0;
}

.notify__item--success .notify__icon {
  background: var(--viper-dim);
  color: var(--viper-400);
}

.notify__item--error .notify__icon {
  background: var(--bg-danger);
  color: var(--danger);
}

.notify__item--info .notify__icon {
  background: var(--gold-dim);
  color: var(--gold-500);
}

.notify__icon .ti {
  font-size: 16px;
}

.notify__message {
  margin: 0;
  padding-top: 4px;
  font-size: 13px;
  line-height: 1.45;
  color: var(--text);
  word-break: break-word;
}

.notify__close {
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--text-faint);
  display: grid;
  place-items: center;
  cursor: pointer;
  padding: 0;
}

.notify__close:hover {
  background: var(--panel-raised);
  color: var(--text);
}

.notify__close .ti {
  font-size: 14px;
}

.notify-enter-active,
.notify-leave-active {
  transition:
    opacity 0.22s ease,
    transform 0.22s ease;
}

.notify-enter-from,
.notify-leave-to {
  opacity: 0;
  transform: translateX(14px);
}

.notify-move {
  transition: transform 0.22s ease;
}

@media (max-width: 560px) {
  .notify {
    top: auto;
    bottom: 1rem;
    right: 1rem;
    left: 1rem;
    width: auto;
  }

  .notify-enter-from,
  .notify-leave-to {
    transform: translateY(12px);
  }
}
</style>
