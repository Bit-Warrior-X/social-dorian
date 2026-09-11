<template>
  <div>
    <div class="page-head">
      <div>
        <h1>Live feed</h1>
        <p>Platform activity — tasks, account opens, proxy checks, and worker errors</p>
      </div>
      <div class="page-head__actions">
        <label class="toggle">
          <input v-model="errorsOnly" type="checkbox" />
          Problems only
        </label>
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

    <div class="filters">
      <input v-model="query" type="search" placeholder="Search message, account, proxy…" />
      <select v-model="sourceFilter">
        <option value="">All sources</option>
        <option value="task">Tasks</option>
        <option value="account">Accounts</option>
        <option value="proxy">Proxies</option>
        <option value="system">System</option>
      </select>
      <select v-model="levelFilter">
        <option value="">All levels</option>
        <option value="error">Error</option>
        <option value="warn">Warn</option>
        <option value="success">Success</option>
        <option value="info">Info</option>
      </select>
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
        <span class="feed__source" :class="'source--' + entry.source">{{ entry.source || 'system' }}</span>
        <span class="feed__ref">
          <RouterLink v-if="entry.taskId" :to="'/tasks?focus=' + entry.taskId">
            #{{ entry.taskId }}
          </RouterLink>
          <span v-else-if="entry.accountName || entry.accountId" class="muted">
            {{ entry.accountName || ('account #' + entry.accountId) }}
          </span>
          <span v-else-if="entry.proxyName || entry.proxyId" class="muted">
            {{ entry.proxyName || ('proxy #' + entry.proxyId) }}
          </span>
          <span v-else class="muted">—</span>
          <span v-if="entry.taskType" class="muted">{{ taskTypeMeta(entry.taskType).label || entry.taskType }}</span>
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
          <span v-if="entry.context" class="feed__ctx">{{ formatContext(entry.context) }}</span>
        </span>
      </div>
      <p v-if="!loading && logs.length === 0" class="empty">
        No activity yet. Run a task, open an account, or check a proxy to see logs here.
      </p>
    </div>

    <TaskWatchModal
      :open="watchOpen"
      :title="watchTitle"
      :frame-url="watchUrl"
      :task-id="watchTaskId"
      @close="closeWatch"
      @frame-url="onWatchFrameUrl"
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
const errorsOnly = ref(false)
const query = ref('')
const sourceFilter = ref('')
const levelFilter = ref('')
const feedEl = ref(null)
const watchOpen = ref(false)
const watchUrl = ref('')
const watchTitle = ref('Live browser')
const watchTaskId = ref(0)
let timer = null
let searchTimer = 0
let lastID = 0

function formatTime(value) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleTimeString()
}

function formatContext(raw) {
  const text = String(raw || '').trim()
  if (!text) return ''
  try {
    const data = JSON.parse(text)
    return Object.entries(data)
      .map(([key, value]) => `${key}=${value}`)
      .join(' · ')
  } catch (_) {
    return text
  }
}

function openWatch(entry) {
  const href = liveViewHref(entry?.message)
  if (!href) return
  watchTaskId.value = Number(entry?.taskId) || 0
  watchTitle.value = entry?.taskId ? `Task #${entry.taskId} · Live browser` : 'Live browser'
  watchUrl.value = href
  watchOpen.value = true
}

function onWatchFrameUrl(url) {
  const href = String(url || '').trim()
  if (!href) return
  watchUrl.value = href
}

function closeWatch() {
  watchOpen.value = false
  watchUrl.value = ''
  watchTaskId.value = 0
}

function feedParams(after = 0, limit = 100) {
  return {
    after,
    limit,
    level: errorsOnly.value ? '' : levelFilter.value,
    source: sourceFilter.value,
    q: query.value.trim(),
    errors: errorsOnly.value,
  }
}

async function refresh(reset = false) {
  try {
    error.value = ''
    const after = reset ? 0 : lastID
    const data = await getMonitorFeed(feedParams(after, reset ? 120 : 60))
    const incoming = Array.isArray(data.logs) ? data.logs : []
    if (reset) {
      logs.value = incoming
    } else if (incoming.length) {
      logs.value = [...logs.value, ...incoming].slice(-500)
    }
    if (logs.value.length) {
      lastID = Math.max(...logs.value.map((row) => Number(row.id) || 0))
    } else if (reset) {
      lastID = 0
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
  }, 2500)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
  if (searchTimer) window.clearTimeout(searchTimer)
})

watch([errorsOnly, sourceFilter, levelFilter], () => {
  lastID = 0
  refresh(true)
})

watch(query, () => {
  if (searchTimer) window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => {
    lastID = 0
    refresh(true)
  }, 350)
})

watch(paused, (value) => {
  if (!value) refresh(false)
})
</script>

<style scoped>
.page-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
  flex-wrap: wrap;
  gap: 12px;
}

.page-head p {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--text-dim);
}

.page-head__actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--text-dim);
  cursor: pointer;
}

.filters {
  display: flex;
  gap: 8px;
  margin-bottom: 1rem;
  flex-wrap: wrap;
}

.filters input[type='search'] {
  flex: 1 1 220px;
  min-width: 180px;
}

.filters select {
  width: 150px;
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
  background: var(--panel);
  border: 0.5px solid var(--hairline);
  border-radius: 12px;
  max-height: calc(100vh - 220px);
  overflow: auto;
}

.feed__row {
  display: grid;
  grid-template-columns: 78px 72px minmax(110px, 160px) 64px 1fr;
  gap: 10px;
  align-items: start;
  padding: 10px 14px;
  border-bottom: 0.5px solid var(--hairline);
  font-size: 13px;
}

.feed__row:last-child {
  border-bottom: none;
}

.feed__time {
  font-family: var(--mono);
  font-size: 11px;
  color: var(--text-faint);
  padding-top: 2px;
}

.feed__source {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: fit-content;
  padding: 1px 7px;
  border-radius: 999px;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  background: var(--panel-raised);
  color: var(--text-faint);
}

.source--task { color: var(--viper-400); }
.source--account { color: #5b8def; }
.source--proxy { color: var(--warn, #d4a017); }
.source--system { color: var(--text-dim); }

.feed__ref {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.feed__ref a {
  color: var(--viper-400);
  text-decoration: none;
  font-weight: 600;
}

.feed__level {
  font-family: var(--mono);
  font-size: 11px;
  text-transform: uppercase;
  color: var(--text-faint);
  padding-top: 2px;
}

.feed__msg {
  color: var(--text-dim);
  word-break: break-word;
  white-space: pre-wrap;
  line-height: 1.45;
}

.feed__ctx {
  display: block;
  margin-top: 4px;
  font-family: var(--mono);
  font-size: 11px;
  color: var(--text-faint);
}

.feed__row--success .feed__msg { color: var(--success); }
.feed__row--warn .feed__msg { color: var(--warn); }
.feed__row--error .feed__msg,
.feed__row--error .feed__level { color: var(--danger); }

.muted {
  color: var(--text-faint);
  font-size: 12px;
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

@media (max-width: 900px) {
  .feed__row {
    grid-template-columns: 70px 1fr;
    grid-template-areas:
      'time source'
      'time ref'
      'time level'
      'time msg';
  }

  .feed__time { grid-area: time; }
  .feed__source { grid-area: source; }
  .feed__ref { grid-area: ref; }
  .feed__level { grid-area: level; }
  .feed__msg { grid-area: msg; }
}
</style>
