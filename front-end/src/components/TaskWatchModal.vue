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
        <span v-if="taskStatus" class="status" :class="'status--' + taskStatus">
          {{ taskStatus }}
        </span>
      </div>

      <div class="watch__main">
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

        <aside class="watch__logs" aria-label="Task log">
          <div class="watch__logs-head">
            <strong>Task log</strong>
            <span>{{ logs.length }}</span>
          </div>
          <div ref="logStreamEl" class="watch__logs-stream">
            <div
              v-for="entry in logs"
              :key="entry.id"
              class="log-line"
              :class="'log-line--' + entry.level"
            >
              <span class="log-line__time">{{ formatTime(entry.createdAt) }}</span>
              <span class="log-line__msg">{{ displayMessage(entry.message) }}</span>
            </div>
            <p v-if="logs.length === 0" class="watch__logs-empty">
              {{ waiting ? 'Waiting for worker output…' : 'No log lines yet.' }}
            </p>
          </div>
        </aside>
      </div>
    </div>
  </ModalDialog>
</template>

<script setup>
import { computed, nextTick, onUnmounted, ref, watch } from 'vue'
import ModalDialog from './ModalDialog.vue'
import { getTask } from '../api/tasks'
import { liveViewHref } from '../composables/useTaskWatch'

const props = defineProps({
  open: { type: Boolean, default: false },
  title: { type: String, default: 'Live browser' },
  frameUrl: { type: String, default: '' },
  waiting: { type: Boolean, default: false },
  taskId: { type: [Number, String], default: 0 },
})

const emit = defineEmits(['close', 'frame-url'])

const logs = ref([])
const taskStatus = ref('')
const logStreamEl = ref(null)
let pollTimer = 0

const statusText = computed(() => {
  if (props.frameUrl) return 'Live'
  if (props.waiting) return 'Starting…'
  return 'Ready'
})

function formatTime(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleTimeString()
}

function displayMessage(message) {
  const text = String(message || '')
  if (liveViewHref(text)) return 'Watch live session is ready'
  return text
}

async function scrollLogs() {
  await nextTick()
  const el = logStreamEl.value
  if (el) el.scrollTop = el.scrollHeight
}

async function pullTaskLogs() {
  const id = Number(props.taskId)
  if (!id) return
  try {
    const task = await getTask(id)
    taskStatus.value = task?.status || ''
    const nextLogs = Array.isArray(task?.logs) ? task.logs : []
    const prevLen = logs.value.length
    logs.value = nextLogs
    if (nextLogs.length !== prevLen) {
      await scrollLogs()
    }

    // If parent is still waiting, surface the watch URL when the worker logs it.
    if (!props.frameUrl) {
      for (let i = nextLogs.length - 1; i >= 0; i -= 1) {
        const href = liveViewHref(nextLogs[i]?.message)
        if (href) {
          emit('frame-url', href)
          break
        }
      }
    }
  } catch (_) {
    // Keep the last known log stream if a poll fails.
  }
}

function startPoll() {
  stopPoll()
  pullTaskLogs()
  pollTimer = window.setInterval(pullTaskLogs, 1500)
}

function stopPoll() {
  if (pollTimer) {
    window.clearInterval(pollTimer)
    pollTimer = 0
  }
}

watch(
  () => [props.open, props.taskId],
  ([open, taskId]) => {
    if (open && Number(taskId)) {
      startPoll()
      return
    }
    stopPoll()
    if (!open) {
      logs.value = []
      taskStatus.value = ''
    }
  },
  { immediate: true },
)

onUnmounted(stopPoll)
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
  flex: 1 1 200px;
}

.status {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
  text-transform: capitalize;
  background: var(--panel-raised);
  color: var(--text-dim);
}

.status--running,
.status--queued {
  background: color-mix(in srgb, var(--warn, #d4a017) 18%, transparent);
  color: var(--warn, #d4a017);
}

.status--completed {
  background: var(--viper-dim);
  color: var(--viper-400);
}

.status--failed {
  background: var(--bg-danger);
  color: var(--danger);
}

.status--cancelled {
  color: var(--text-faint);
}

.watch__main {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(260px, 320px);
  gap: 8px;
  flex: 1 1 auto;
  min-height: 0;
}

.watch__frame-wrap {
  position: relative;
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

.watch__logs {
  display: flex;
  flex-direction: column;
  min-height: 0;
  border: 0.5px solid var(--hairline);
  border-radius: 8px;
  background: var(--bg);
  overflow: hidden;
}

.watch__logs-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  border-bottom: 0.5px solid var(--hairline);
  background: var(--panel-raised);
  flex-shrink: 0;
}

.watch__logs-head strong {
  font-size: 12px;
  font-weight: 600;
}

.watch__logs-head span {
  font-family: var(--mono);
  font-size: 11px;
  color: var(--text-faint);
}

.watch__logs-stream {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 6px 0;
}

.watch__logs-empty {
  margin: 0;
  padding: 16px 12px;
  text-align: center;
  color: var(--text-faint);
  font-size: 12px;
}

.log-line {
  display: grid;
  grid-template-columns: 64px 1fr;
  gap: 8px;
  padding: 5px 10px;
  font-size: 12px;
  line-height: 1.4;
  border-bottom: 0.5px solid color-mix(in srgb, var(--hairline) 70%, transparent);
}

.log-line:last-child {
  border-bottom: none;
}

.log-line__time {
  font-family: var(--mono);
  font-size: 10px;
  color: var(--text-faint);
  padding-top: 1px;
}

.log-line__msg {
  word-break: break-word;
  color: var(--text-dim);
  white-space: pre-wrap;
}

.log-line--success .log-line__msg { color: var(--success); }
.log-line--warn .log-line__msg { color: var(--warn); }
.log-line--error .log-line__msg { color: var(--danger); }

.spin {
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 1100px) {
  .watch__main {
    grid-template-columns: 1fr;
    grid-template-rows: minmax(280px, 1fr) minmax(180px, 240px);
  }
}
</style>
