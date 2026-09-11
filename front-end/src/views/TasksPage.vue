<template>
  <div>
    <div class="page-head">
      <div>
        <h1>{{ title }}</h1>
        <p>{{ description }}</p>
      </div>
      <button class="btn btn-primary" type="button" @click="openNewTask()">
        <i class="ti ti-plus" aria-hidden="true" />
        New task
      </button>
    </div>

    <p v-if="loadError" class="banner">{{ loadError }}</p>

    <div class="stats">
      <div class="stat">
        <div class="stat__label">Total</div>
        <div class="stat__value">{{ tasks.length }}</div>
      </div>
      <div class="stat">
        <div class="stat__label">Running</div>
        <div class="stat__value is-live">{{ countByStatus('running') + countByStatus('queued') }}</div>
      </div>
      <div class="stat">
        <div class="stat__label">Completed</div>
        <div class="stat__value is-success">{{ countByStatus('completed') }}</div>
      </div>
      <div class="stat">
        <div class="stat__label">Failed</div>
        <div class="stat__value is-danger">{{ countByStatus('failed') }}</div>
      </div>
    </div>

    <div class="filters">
      <select v-model="typeFilter">
        <option value="">All types</option>
        <option v-for="type in TASK_TYPES" :key="type.value" :value="type.value">
          {{ type.label }}
        </option>
      </select>
      <select v-model="statusFilter">
        <option value="">All statuses</option>
        <option v-for="status in TASK_STATUSES" :key="status.value" :value="status.value">
          {{ status.label }}
        </option>
      </select>
    </div>

    <div class="table-card">
      <div v-if="loading" class="empty">Loading tasks…</div>
      <template v-else>
        <div class="table-scroll">
          <table>
            <thead>
              <tr>
                <th>Task</th>
                <th>Type</th>
                <th>Accounts</th>
                <th>Status</th>
                <th>Progress</th>
                <th>Created</th>
                <th class="col-actions">Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="task in pageItems" :key="task.id">
                <td>
                  <div class="task-cell">
                    <strong>#{{ task.id }} {{ task.title }}</strong>
                    <span v-if="task.targetUrl" class="sub mono">{{ task.targetUrl }}</span>
                    <span v-else-if="taskContentPreview(task)" class="sub">{{ taskContentPreview(task) }}</span>
                  </div>
                </td>
                <td>
                  <span class="type">
                    <i class="ti" :class="taskTypeMeta(task.type).icon" aria-hidden="true" />
                    {{ taskTypeMeta(task.type).label }}
                  </span>
                </td>
                <td>{{ task.accountCount }}</td>
                <td>
                  <span class="status" :class="'status--' + task.status">
                    {{ taskStatusLabel(task.status) }}
                  </span>
                </td>
                <td>
                  <div class="progress">
                    <div class="progress__bar">
                      <span :style="{ width: taskProgress(task) + '%' }" />
                    </div>
                    <div class="progress__label">
                      {{ task.doneCount }}/{{ task.accountCount }}
                      <template v-if="task.successCount || task.failCount">
                        · {{ task.successCount }} ok
                        <template v-if="task.failCount"> / {{ task.failCount }} fail</template>
                      </template>
                    </div>
                  </div>
                </td>
                <td>
                  <div class="task-cell">
                    <span>{{ task.createdBy || '—' }}</span>
                    <span class="sub">{{ formatWhen(task.createdAt) }}</span>
                  </div>
                </td>
                <td class="actions">
                  <button
                    v-if="task.showBrowser && (task.status === 'running' || task.status === 'queued')"
                    class="btn btn-icon"
                    type="button"
                    aria-label="Open browser window"
                    title="Open browser window"
                    @click="openTaskWatch(task)"
                  >
                    <i class="ti ti-brand-chrome" aria-hidden="true" />
                  </button>
                  <button class="btn btn-icon" type="button" aria-label="View logs" @click="openDetail(task)">
                    <i class="ti ti-list-details" aria-hidden="true" />
                  </button>
                  <button
                    v-if="task.status === 'running' || task.status === 'queued'"
                    class="btn btn-icon"
                    type="button"
                    aria-label="Cancel task"
                    @click="onCancel(task)"
                  >
                    <i class="ti ti-player-stop" aria-hidden="true" />
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-if="filteredTasks.length === 0" class="empty">No tasks match your filters.</p>
        <PaginationBar
          v-model:page="page"
          v-model:page-size="pageSize"
          :total="total"
          :total-pages="totalPages"
          :from="from"
          :to="to"
          :page-size-options="pageSizeOptions"
        />
      </template>
    </div>

    <NewTaskModal
      :open="taskModalOpen"
      :accounts="accounts"
      :initial-type="taskModalType"
      :initial-account-ids="taskModalAccountIds"
      :saving="taskSaving"
      :error="taskError"
      @close="taskModalOpen = false"
      @submit="launchTask"
    />

    <ModalDialog
      :open="Boolean(detail)"
      :title="detail ? `Task #${detail.id}` : 'Task'"
      title-id="task-detail-title"
      wide
      @close="detail = null"
    >
      <div v-if="detail" class="detail">
        <div class="detail__meta">
          <span class="status" :class="'status--' + detail.status">{{ taskStatusLabel(detail.status) }}</span>
          <span>{{ taskTypeMeta(detail.type).label }}</span>
          <span>{{ detail.doneCount }}/{{ detail.accountCount }} done</span>
        </div>
        <p v-if="detail.targetUrl" class="detail__url mono">{{ detail.targetUrl }}</p>
        <div v-else-if="detail.content?.text || detail.content?.headline" class="detail__content">
          <strong v-if="detail.content.headline">{{ detail.content.headline }}</strong>
          <p v-if="detail.content.text">{{ detail.content.text }}</p>
          <p v-if="detail.content.linkUrl" class="mono">{{ detail.content.linkUrl }}</p>
          <p v-if="detail.content.mediaUrl" class="mono">{{ detail.content.mediaUrl }}</p>
        </div>
        <div class="log-toolbar">
          <strong>Logs</strong>
          <select v-model="detailLevelFilter">
            <option value="">All levels</option>
            <option value="error">Errors</option>
            <option value="warn">Warnings</option>
            <option value="success">Success</option>
            <option value="info">Info</option>
          </select>
        </div>
        <div class="log-stream">
          <div
            v-for="entry in detailLogs"
            :key="entry.id"
            class="log"
            :class="'log--' + entry.level"
          >
            <span class="log__time">{{ formatTime(entry.createdAt) }}</span>
            <span>
              <template v-if="liveViewHref(entry.message)">
                Watch live:
                <button class="live-link" type="button" @click="openWatchFromHref(liveViewHref(entry.message), detail)">
                  Open browser window
                </button>
              </template>
              <template v-else>{{ entry.message }}</template>
            </span>
          </div>
          <p v-if="!detailLogs.length" class="empty">No logs yet.</p>
        </div>
      </div>
    </ModalDialog>

    <TaskWatchModal
      :open="watchOpen"
      :title="watchTitle"
      :frame-url="watchUrl"
      :waiting="watchWaiting"
      :task-id="watchTaskId"
      @close="closeWatch"
      @frame-url="onWatchFrameUrl"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import ModalDialog from '../components/ModalDialog.vue'
import NewTaskModal from '../components/NewTaskModal.vue'
import PaginationBar from '../components/PaginationBar.vue'
import TaskWatchModal from '../components/TaskWatchModal.vue'
import { listAccounts } from '../api/accounts'
import { cancelTask, createTask, getTask, listTasks } from '../api/tasks'
import { useNotify } from '../composables/useNotify'
import { usePagination } from '../composables/usePagination'
import {
  clearPendingWatchTask,
  findTaskWatchUrl,
  liveViewHref,
  peekPendingWatchTaskId,
  requestTaskWatch,
} from '../composables/useTaskWatch'
import { TASK_STATUSES, TASK_TYPES, taskContentPreview, taskProgress, taskStatusLabel, taskTypeMeta } from '../constants/tasks'

const props = defineProps({
  mode: { type: String, default: 'all' }, // all | active | history
})

const { notifySuccess, notifyError } = useNotify()
const route = useRoute()
const router = useRouter()

const tasks = ref([])
const accounts = ref([])
const loading = ref(true)
const loadError = ref('')
const typeFilter = ref('')
const statusFilter = ref('')
const taskModalOpen = ref(false)
const taskModalType = ref('report')
const taskModalAccountIds = ref([])
const taskSaving = ref(false)
const taskError = ref('')
const detail = ref(null)
const detailLevelFilter = ref('')
const watchOpen = ref(false)
const watchUrl = ref('')
const watchTitle = ref('Live browser')
const watchWaiting = ref(false)
const watchTaskId = ref(0)
let pollTimer = 0

const title = computed(() => {
  if (props.mode === 'active') return 'Active jobs'
  if (props.mode === 'history') return 'Task history'
  return 'Tasks / Campaigns'
})

const description = computed(() => {
  if (props.mode === 'active') return 'Queued and running jobs across your accounts'
  if (props.mode === 'history') return 'Completed, failed, and cancelled jobs'
  return 'Create report, reply, post, browse, and login-test jobs'
})

const filteredTasks = computed(() => {
  return tasks.value.filter((task) => {
    if (props.mode === 'active' && !['queued', 'running', 'pending'].includes(task.status)) return false
    if (props.mode === 'history' && !['completed', 'failed', 'cancelled'].includes(task.status)) return false
    if (typeFilter.value && task.type !== typeFilter.value) return false
    if (statusFilter.value && task.status !== statusFilter.value) return false
    return true
  })
})

const detailLogs = computed(() => {
  const rows = Array.isArray(detail.value?.logs) ? detail.value.logs : []
  if (!detailLevelFilter.value) return rows
  return rows.filter((entry) => entry.level === detailLevelFilter.value)
})

const {
  page,
  pageSize,
  pageItems,
  total,
  totalPages,
  from,
  to,
  pageSizeOptions,
  reset: resetPage,
} = usePagination(filteredTasks)

watch([typeFilter, statusFilter, () => props.mode], resetPage)

function countByStatus(status) {
  return tasks.value.filter((task) => task.status === status).length
}

function formatWhen(value) {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

function formatTime(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleTimeString()
}

function openWatchModal({ taskId = 0, title = 'Live browser', url = '', waiting = false } = {}) {
  watchTaskId.value = Number(taskId) || 0
  watchTitle.value = title
  watchUrl.value = url || ''
  watchWaiting.value = Boolean(waiting) && !url
  watchOpen.value = true
}

function closeWatch() {
  if (watchTaskId.value) clearPendingWatchTask(watchTaskId.value)
  watchOpen.value = false
  watchWaiting.value = false
  watchUrl.value = ''
  watchTaskId.value = 0
}

function onWatchFrameUrl(url) {
  const href = String(url || '').trim()
  if (!href) return
  watchUrl.value = href
  watchWaiting.value = false
  if (watchTaskId.value) clearPendingWatchTask(watchTaskId.value)
}

function openWatchFromHref(href, task = null) {
  if (!href) return
  openWatchModal({
    taskId: task?.id || 0,
    title: task ? `Task #${task.id} · Live browser` : 'Live browser',
    url: href,
    waiting: false,
  })
}

async function openTaskWatch(task) {
  openWatchModal({
    taskId: task.id,
    title: `Task #${task.id} · Live browser`,
    url: '',
    waiting: true,
  })
  try {
    const full = await getTask(task.id)
    const url = findTaskWatchUrl(full)
    if (url) {
      watchUrl.value = url
      watchWaiting.value = false
      clearPendingWatchTask(task.id)
      return
    }
    requestTaskWatch(task.id)
    notifySuccess('Live view is starting — hang on a moment')
  } catch (err) {
    watchWaiting.value = false
    notifyError(err.message || 'Could not open live browser')
  }
}

async function syncPendingWatch() {
  const pendingId = watchOpen.value && watchWaiting.value && watchTaskId.value
    ? watchTaskId.value
    : peekPendingWatchTaskId()
  if (!pendingId) return

  try {
    const task = await getTask(pendingId)
    const url = findTaskWatchUrl(task)
    if (!url) {
      if (['completed', 'failed', 'cancelled'].includes(task.status)) {
        clearPendingWatchTask(pendingId)
        if (watchOpen.value && watchTaskId.value === pendingId && !watchUrl.value) {
          watchWaiting.value = false
          notifyError(`Task #${pendingId} finished before the live view was ready`)
        }
      } else if (!watchOpen.value) {
        openWatchModal({
          taskId: pendingId,
          title: `Task #${pendingId} · Live browser`,
          url: '',
          waiting: true,
        })
      }
      return
    }

    openWatchModal({
      taskId: pendingId,
      title: `Task #${pendingId} · Live browser`,
      url,
      waiting: false,
    })
    clearPendingWatchTask(pendingId)
  } catch (_) {
    // Keep waiting; next poll retries.
  }
}

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    const [taskRows, accountRows] = await Promise.all([
      listTasks(),
      listAccounts().catch(() => []),
    ])
    tasks.value = taskRows
    accounts.value = accountRows
    await syncPendingWatch()
  } catch (err) {
    loadError.value = err.message || 'Could not load tasks'
    notifyError(loadError.value)
  } finally {
    loading.value = false
  }
}

async function refreshQuiet() {
  try {
    tasks.value = await listTasks()
    if (detail.value?.id) {
      detail.value = await getTask(detail.value.id)
    }
    await syncPendingWatch()
  } catch (_) {
    // Keep the last known list if a poll fails.
  }
}

function openNewTask(type = 'report', accountIds = []) {
  taskModalType.value = type
  taskModalAccountIds.value = accountIds
  taskError.value = ''
  taskModalOpen.value = true
}

async function launchTask(payload) {
  taskSaving.value = true
  taskError.value = ''
  try {
    const task = await createTask(payload)
    taskModalOpen.value = false
    notifySuccess(`Launched ${taskTypeMeta(task.type).label} #${task.id}`)
    if (payload.showBrowser) {
      requestTaskWatch(task.id)
      openWatchModal({
        taskId: task.id,
        title: `Task #${task.id} · Live browser`,
        url: '',
        waiting: true,
      })
    }
    await router.push('/tasks/active')
    await load()
  } catch (err) {
    taskError.value = err.message || 'Could not launch task'
    notifyError(taskError.value)
  } finally {
    taskSaving.value = false
  }
}

async function openDetail(task) {
  detailLevelFilter.value = ''
  try {
    detail.value = await getTask(task.id)
  } catch (err) {
    notifyError(err.message || 'Could not load task')
  }
}

async function onCancel(task) {
  try {
    await cancelTask(task.id)
    notifySuccess(`Cancelled task #${task.id}`)
    await refreshQuiet()
  } catch (err) {
    notifyError(err.message || 'Could not cancel task')
  }
}

watch(
  () => route.query.new,
  (value) => {
    if (!value) return
    const type = String(value)
    if (TASK_TYPES.some((item) => item.value === type)) {
      openNewTask(type)
      router.replace({ query: { ...route.query, new: undefined } })
    }
  },
  { immediate: true },
)

onMounted(async () => {
  await load()
  pollTimer = window.setInterval(refreshQuiet, 2500)
})

onUnmounted(() => {
  if (pollTimer) window.clearInterval(pollTimer)
})

defineExpose({ openNewTask })
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

.banner {
  margin-bottom: 1rem;
  padding: 8px 12px;
  border-radius: var(--radius);
  border: 0.5px solid color-mix(in srgb, var(--danger) 35%, transparent);
  font-size: 13px;
  color: var(--danger);
  background: var(--bg-danger);
}

.stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 12px;
  margin-bottom: 1rem;
}

.stat {
  background: var(--panel);
  border: 0.5px solid var(--hairline);
  border-radius: 10px;
  padding: 14px 16px;
}

.stat__label {
  font-family: var(--mono);
  font-size: 11px;
  color: var(--text-faint);
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.stat__value {
  font-family: var(--mono);
  font-size: 22px;
  font-weight: 600;
  margin-top: 4px;
  color: var(--viper-400);
}

.stat__value.is-live { color: var(--warn); }
.stat__value.is-success { color: var(--success); }
.stat__value.is-danger { color: var(--danger); }

.filters {
  display: flex;
  gap: 8px;
  margin-bottom: 1rem;
  flex-wrap: wrap;
}

.filters select {
  width: 160px;
}

.table-card {
  background: var(--panel);
  border: 0.5px solid var(--hairline);
  border-radius: 12px;
  overflow: hidden;
}

.table-scroll {
  overflow-x: auto;
}

table {
  width: 100%;
  min-width: 960px;
  border-collapse: collapse;
  font-size: 14px;
}

th,
td {
  padding: 10px 12px;
  text-align: left;
  border-bottom: 0.5px solid var(--hairline);
  vertical-align: top;
}

th {
  font-weight: 500;
  color: var(--text-dim);
}

.col-actions,
.actions {
  text-align: right;
}

.task-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.task-cell strong {
  font-weight: 600;
}

.sub {
  font-size: 12px;
  color: var(--text-faint);
}

.mono {
  font-family: var(--mono);
  font-size: 12px;
}

.type {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.status {
  display: inline-block;
  font-family: var(--mono);
  font-size: 11.5px;
  padding: 5px 11px;
  border-radius: 20px;
}

.status--queued,
.status--pending { background: var(--panel-raised); color: var(--text-dim); }
.status--running { background: var(--gold-dim); color: var(--gold-500); }
.status--completed { background: var(--viper-dim); color: var(--viper-400); }
.status--failed { background: var(--bg-danger); color: var(--danger); }
.status--cancelled { background: var(--panel-raised); color: var(--text-faint); }

.progress {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 140px;
}

.progress__bar {
  height: 6px;
  border-radius: 999px;
  background: var(--panel-raised);
  overflow: hidden;
}

.progress__bar span {
  display: block;
  height: 100%;
  background: var(--viper-500);
}

.progress__label {
  font-size: 12px;
  color: var(--text-faint);
}

.empty {
  text-align: center;
  padding: 2rem 1rem;
  color: var(--text-dim);
  font-size: 14px;
  margin: 0;
}

.detail {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.detail__meta {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  align-items: center;
  font-size: 13px;
  color: var(--text-dim);
}

.detail__url {
  margin: 0;
  color: var(--text-faint);
  word-break: break-all;
}

.detail__content {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px;
  border: 0.5px solid var(--hairline);
  border-radius: 10px;
  background: var(--bg);
}

.detail__content strong {
  color: var(--text);
  font-size: 14px;
}

.detail__content p {
  margin: 0;
  color: var(--text-dim);
  font-size: 13px;
  white-space: pre-wrap;
  line-height: 1.45;
  word-break: break-word;
}

.log-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 8px;
}

.log-toolbar strong {
  font-size: 13px;
}

.log-toolbar select {
  width: 140px;
}

.log-stream {
  max-height: 360px;
  overflow: auto;
  border: 0.5px solid var(--hairline);
  border-radius: 10px;
  background: var(--bg);
}

.log {
  display: grid;
  grid-template-columns: 72px 1fr;
  gap: 10px;
  padding: 8px 12px;
  border-bottom: 0.5px solid var(--hairline);
  font-size: 13px;
}

.log:last-child { border-bottom: none; }
.log__time {
  font-family: var(--mono);
  font-size: 11px;
  color: var(--text-faint);
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
.log--success { color: var(--success); }
.log--warn { color: var(--warn); }
.log--error { color: var(--danger); }
</style>
