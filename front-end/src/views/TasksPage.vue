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
              <tr v-for="task in filteredTasks" :key="task.id">
                <td>
                  <div class="task-cell">
                    <strong>#{{ task.id }} {{ task.title }}</strong>
                    <span v-if="task.targetUrl" class="sub mono">{{ task.targetUrl }}</span>
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
        <div class="log-stream">
          <div v-for="entry in detail.logs || []" :key="entry.id" class="log" :class="'log--' + entry.level">
            <span class="log__time">{{ formatTime(entry.createdAt) }}</span>
            <span>{{ entry.message }}</span>
          </div>
          <p v-if="!(detail.logs || []).length" class="empty">No logs yet.</p>
        </div>
      </div>
    </ModalDialog>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import ModalDialog from '../components/ModalDialog.vue'
import NewTaskModal from '../components/NewTaskModal.vue'
import { listAccounts } from '../api/accounts'
import { cancelTask, createTask, getTask, listTasks } from '../api/tasks'
import { useNotify } from '../composables/useNotify'
import { TASK_STATUSES, TASK_TYPES, taskProgress, taskStatusLabel, taskTypeMeta } from '../constants/tasks'

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
let pollTimer = 0

const title = computed(() => {
  if (props.mode === 'active') return 'Active jobs'
  if (props.mode === 'history') return 'Task history'
  return 'Tasks / Campaigns'
})

const description = computed(() => {
  if (props.mode === 'active') return 'Queued and running jobs across your accounts'
  if (props.mode === 'history') return 'Completed, failed, and cancelled jobs'
  return 'Create report, post, browse, and login-test jobs'
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
.log--success { color: var(--success); }
.log--warn { color: var(--warn); }
.log--error { color: var(--danger); }
</style>
