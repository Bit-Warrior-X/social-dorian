<template>
  <ModalDialog
    :open="open"
    :title="title"
    title-id="task-watch-title"
    full
    @close="emit('close')"
  >
    <div class="watch">
      <div class="watch__toolbar">
        <div class="pill" :class="{ 'is-ready': Boolean(frameUrl), 'is-wait': waiting && !frameUrl }">
          <i
            class="ti"
            :class="waiting && !frameUrl ? 'ti-loader-2 spin' : 'ti-brand-chrome'"
            aria-hidden="true"
          />
          <span>{{ statusText }}</span>
        </div>
        <p class="watch__hint">
          Live remote desktop for this job. Closing this dialog does not stop the task.
        </p>
      </div>

      <div class="watch__frame-wrap">
        <div v-if="waiting && !frameUrl" class="watch__loading">
          <i class="ti ti-loader-2 spin" aria-hidden="true" />
          Starting live browser view…
        </div>
        <iframe
          v-else-if="frameUrl"
          class="watch__frame"
          :src="frameUrl"
          :title="title"
          referrerpolicy="no-referrer"
        />
        <div v-else class="watch__loading">Waiting for session…</div>
      </div>
    </div>
  </ModalDialog>
</template>

<script setup>
import { computed } from 'vue'
import ModalDialog from './ModalDialog.vue'

const props = defineProps({
  open: { type: Boolean, default: false },
  title: { type: String, default: 'Live browser' },
  frameUrl: { type: String, default: '' },
  waiting: { type: Boolean, default: false },
})

const emit = defineEmits(['close'])

const statusText = computed(() => {
  if (props.frameUrl) return 'Live'
  if (props.waiting) return 'Starting…'
  return 'Ready'
})
</script>

<style scoped>
.watch {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 0;
  flex: 1;
  height: 100%;
}

.watch__toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  flex-shrink: 0;
}

.pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 8px;
  border-radius: 999px;
  border: 0.5px solid var(--hairline);
  background: var(--panel-raised);
  font-size: 11px;
  color: var(--text-dim);
  white-space: nowrap;
}

.pill .ti {
  font-size: 13px;
}

.pill.is-ready {
  border-color: color-mix(in srgb, var(--viper-500) 45%, transparent);
  background: var(--viper-dim);
  color: var(--viper-400);
}

.pill.is-wait {
  border-color: color-mix(in srgb, var(--warn, #d4a017) 45%, transparent);
}

.watch__hint {
  margin: 0;
  font-size: 12px;
  color: var(--text-faint);
}

.watch__frame-wrap {
  position: relative;
  flex: 1 1 auto;
  min-height: 0;
  border: 0.5px solid var(--hairline);
  border-radius: 8px;
  overflow: hidden;
  background: #111;
}

.watch__frame {
  width: 100%;
  height: 100%;
  border: 0;
  background: #111;
}

.watch__loading {
  position: absolute;
  inset: 0;
  display: grid;
  place-content: center;
  gap: 6px;
  justify-items: center;
  color: var(--text-dim);
  background: var(--bg);
  font-size: 13px;
}

.watch__loading .ti {
  font-size: 20px;
  color: var(--viper-400);
}

.spin {
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
