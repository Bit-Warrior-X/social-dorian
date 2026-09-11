<template>
  <div>
    <div class="page-head">
      <div>
        <h1>Live feed</h1>
        <p>Streaming task logs across report, reply, post, browse, and login jobs</p>
      </div>
      <div class="page-head__actions">
        <label class="toggle">
          <input v-model="paused" type="checkbox" />
          Pause
        </label>
        <button class="btn" type="button" :disabled="loading" @click="refresh(true)">
          <i class="ti ti-refresh" aria-hidden="true" />
          Refresh
        </button>
      </div>
    </div>

    <p v-if="error" class="banner">{{ error }}</p>

    <div class="feed" ref="feedEl">
      <div
        v-for="entry in logs"
        :key="entry.id"
        class="feed__row"
        :class="'feed__row--' + entry.level"
      >
        <span class="feed__time">{{ formatTime(entry.createdAt) }}</span>
        <span class="feed__task">
          <RouterLink :to="'/tasks?focus=' + entry.taskId">#{{ entry.taskId }}</RouterLink>
          <span class="muted">{{ taskTypeMeta(entry.taskType).label || entry.taskType || 'Task' }}</span>
        </span>
        <span class="feed__level">{{ entry.level }}</span>
        <span class="feed__msg">
          <template v-if="liveViewHref(entry.message)">
            Watch live:
            <button class="live-link" type="button" @click="openWatch(entry)">
              Open browser window
            </button>
          </template>
          <template v-else>{{ entry.message }}</template>
        </span>
      </div>
      <p v-if="!loading && logs.length === 0" class="empty">No activity yet. Launch a task to see logs here.</p>
    </div>

    <TaskWatchModal
      :open="watchOpen"
      :title="watchTitle"
      :frame-url="watchUrl"
      @close="closeWatch"
    />
  </div>
</template>

<script setup>
import { nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import TaskWatchModal from '../components/TaskWatchModal.vue'
import { getMonitorFeed } from '../api/tasks'
import { useNotify } from '../composables/useNotify'
import { liveViewHref } from '../composables/useTaskWatch'
import { taskTypeMeta } from '../constants/tasks'

const { notifyError } = useNotify()
const logs = ref([])
const loading = ref(true)
const error = ref('')
const paused = ref(false)
const feedEl = ref(null)
const watchOpen = ref(false)
const watchUrl = ref('')
const watchTitle = ref('Live browser')
let timer = null
let lastID = 0

function formatTime(value) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleTimeString()
}

function openWatch(entry) {
  const href = liveViewHref(entry?.message)
  if (!href) return
  watchTitle.value = entry?.taskId ? `Task #${entry.taskId} · Live browser` : 'Live browser'
  watchUrl.value = href
  watchOpen.value = true
}

function closeWatch() {
  watchOpen.value = false
  watchUrl.value = ''
}

async function refresh(reset = false) {
  try {
    error.value = ''
    const after = reset ? 0 : lastID
    const data = await getMonitorFeed({ after, limit: reset ? 80 : 40 })
    const incoming = Array.isArray(data.logs) ? data.logs : []
    if (reset) {
      logs.value = incoming
    } else if (incoming.length) {
      logs.value = [...logs.value, ...incoming].slice(-300)
    }
    if (logs.value.length) {
      lastID = Math.max(...logs.value.map((row) => Number(row.id) || 0))
    }
    await nextTick()
    if (feedEl.value && !paused.value) {
      feedEl.value.scrollTop = feedEl.value.scrollHeight
    }
  } catch (err) {
    error.value = err.message || 'Could not load live feed'
    if (reset) notifyError(error.value)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  refresh(true)
  timer = setInterval(() => {
    if (!paused.value) refresh(false)
  }, 3000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

watch(paused, (value) => {
  if (!value) refresh(false)
})
</script>

<style scoped>
.page-head {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: flex-start;
  margin-bottom: 1rem;
}

.page-head p {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--text-dim);
}

.page-head__actions {
  display: flex;
  gap: 10px;
  align-items: center;
}

.toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--text-dim);
  cursor: pointer;
}

.banner {
  margin-bottom: 1rem;
  padding: 8px 12px;
  border-radius: var(--radius);
  border: 0.5px solid color-mix(in srgb, var(--danger) 35%, transparent);
  font-size: 13px;
  color: var(--danger);
  background: var(--bg-danger);
}

.feed {
  max-height: calc(100vh - 180px);
  overflow: auto;
  border: 0.5px solid var(--hairline);
  border-radius: 12px;
  background: var(--bg);
}

.feed__row {
  display: grid;
  grid-template-columns: 88px 140px 72px 1fr;
  gap: 10px;
  padding: 10px 14px;
  border-bottom: 0.5px solid var(--hairline);
  font-size: 13px;
  align-items: start;
}

.feed__row:last-child {
  border-bottom: none;
}

.feed__time {
  font-family: var(--mono);
  font-size: 12px;
  color: var(--text-faint);
}

.feed__task {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.feed__task a {
  color: var(--viper-400);
  text-decoration: none;
  font-weight: 600;
}

.muted {
  color: var(--text-faint);
  font-size: 11px;
}

.feed__level {
  font-family: var(--mono);
  font-size: 11px;
  text-transform: uppercase;
  color: var(--text-faint);
}

.feed__row--success .feed__level { color: var(--viper-400); }
.feed__row--error .feed__level { color: var(--danger); }
.feed__row--warn .feed__level { color: var(--gold-500); }

.feed__msg {
  color: var(--text-dim);
  word-break: break-word;
  line-height: 1.4;
}

.live-link {
  display: inline;
  padding: 0;
  border: 0;
  background: none;
  color: var(--viper-400);
  font: inherit;
  font-weight: 600;
  cursor: pointer;
  text-decoration: none;
}

.live-link:hover {
  text-decoration: underline;
}

.empty {
  margin: 0;
  padding: 2rem;
  text-align: center;
  color: var(--text-dim);
}

@media (max-width: 800px) {
  .feed__row {
    grid-template-columns: 1fr;
    gap: 4px;
  }
}
</style>
