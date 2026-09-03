<template>
  <div>
    <div class="page-head">
      <div>
        <h1>Proxy management</h1>
        <p>Proxies available for social account connections</p>
      </div>
      <div class="page-head__actions">
        <button
          class="btn"
          type="button"
          :disabled="updatingStatuses || loading || proxies.length === 0"
          @click="updateAllStatuses"
        >
          <i
            class="ti"
            :class="updatingStatuses ? 'ti-loader-2 spin' : 'ti-refresh'"
            aria-hidden="true"
          />
          {{ updatingStatuses ? 'Updating…' : 'Update status' }}
        </button>
        <button class="btn btn-primary" type="button" @click="openCreate">
          <i class="ti ti-plus" aria-hidden="true" />
          Add proxy
        </button>
      </div>
    </div>

    <p v-if="loadError" class="banner">{{ loadError }}</p>

    <div class="stats">
      <div class="stat">
        <div class="stat__label">Total</div>
        <div class="stat__value">{{ proxies.length }}</div>
      </div>
      <div class="stat">
        <div class="stat__label">Active</div>
        <div class="stat__value is-success">{{ countByStatus('active') }}</div>
      </div>
      <div class="stat">
        <div class="stat__label">Needs attention</div>
        <div class="stat__value is-warning">{{ needsAttention }}</div>
      </div>
    </div>

    <div class="filters">
      <input v-model="query" type="text" placeholder="Search proxies" />
      <select v-model="protocolFilter">
        <option value="">All protocols</option>
        <option v-for="protocol in PROXY_PROTOCOLS" :key="protocol.value" :value="protocol.value">
          {{ protocol.label }}
        </option>
      </select>
      <select v-model="statusFilter">
        <option value="">All statuses</option>
        <option v-for="status in PROXY_STATUSES" :key="status.value" :value="status.value">
          {{ status.label }}
        </option>
      </select>
    </div>

    <div class="table-card">
      <div v-if="loading" class="empty">Loading proxies…</div>
      <template v-else>
        <div class="table-scroll">
          <table>
            <thead>
              <tr>
                <th class="col-name">Name</th>
                <th class="col-endpoint">Endpoint</th>
                <th class="col-country">Country</th>
                <th class="col-auth">Auth</th>
                <th class="col-accounts">Accounts</th>
                <th class="col-notes">Notes</th>
                <th class="col-activity">Activity</th>
                <th class="col-status">Status</th>
                <th class="col-actions">Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="proxy in filteredProxies" :key="proxy.id">
                <td>
                  <div class="stack-cell">
                    <div class="stack-cell__primary">{{ proxy.name }}</div>
                    <div class="stack-cell__sub">{{ protocolLabel(proxy.protocol) }}</div>
                  </div>
                </td>
                <td>
                  <div class="stack-cell">
                    <div class="stack-cell__primary mono">{{ proxy.host }}:{{ proxy.port }}</div>
                    <div class="stack-cell__sub mono-soft">{{ proxyAddress(proxy) }}</div>
                  </div>
                </td>
                <td>
                  <div class="stack-cell">
                    <div class="stack-cell__primary">{{ countryName(proxy.country) }}</div>
                    <div class="stack-cell__sub">{{ proxy.country || '—' }}</div>
                  </div>
                </td>
                <td>
                  <div class="stack-cell">
                    <div class="stack-cell__primary">
                      {{ proxy.username ? proxy.username : 'No auth' }}
                    </div>
                    <div class="stack-cell__sub">
                      {{ proxy.username ? (proxy.password ? 'Password set' : 'No password') : 'Open proxy' }}
                    </div>
                  </div>
                </td>
                <td>
                  <span class="accounts-count" :class="{ 'is-empty': !proxy.accountCount }">
                    {{ proxy.accountCount || 0 }}
                  </span>
                </td>
                <td>
                  <div class="notes-cell" :title="proxy.notes || ''">
                    {{ proxy.notes || '—' }}
                  </div>
                </td>
                <td>
                  <div class="stack-cell">
                    <div class="stack-cell__primary">
                      {{ formatDisplayDateTime(proxy.updatedAt) }}
                    </div>
                    <div class="stack-cell__sub" :class="checkSubClass(proxy)">
                      {{ checkSummary(proxy) }}
                    </div>
                  </div>
                </td>
                <td>
                  <span class="status" :class="'status--' + proxy.status">
                    {{ proxyStatusLabel(proxy.status) }}
                  </span>
                </td>
                <td class="actions">
                  <button
                    class="btn btn-icon"
                    type="button"
                    :disabled="updatingStatuses || checkingId === proxy.id"
                    :aria-label="'Check ' + proxy.name"
                    :title="checkTitle(proxy)"
                    @click="runCheck(proxy)"
                  >
                    <i
                      class="ti"
                      :class="checkingId === proxy.id ? 'ti-loader-2 spin' : 'ti-heartbeat'"
                      aria-hidden="true"
                    />
                  </button>
                  <button
                    class="btn btn-icon"
                    type="button"
                    :aria-label="'Edit ' + proxy.name"
                    @click="openEdit(proxy)"
                  >
                    <i class="ti ti-edit" aria-hidden="true" />
                  </button>
                  <button
                    class="btn btn-icon"
                    type="button"
                    :aria-label="'Delete ' + proxy.name"
                    @click="openDelete(proxy)"
                  >
                    <i class="ti ti-trash" aria-hidden="true" />
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-if="filteredProxies.length === 0" class="empty">No proxies match your filters.</p>
      </template>
    </div>

    <ProxyFormModal
      :open="formOpen"
      :proxy="editing"
      :saving="saving"
      :error="formError"
      @close="closeForm"
      @submit="saveProxy"
    />

    <ConfirmDialog
      :open="Boolean(deleting)"
      title="Delete proxy"
      title-id="delete-proxy-title"
      :message="deleteMessage"
      confirm-label="Delete"
      saving-label="Deleting…"
      :saving="saving"
      @close="deleting = null"
      @confirm="confirmDelete"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import ProxyFormModal from '../components/ProxyFormModal.vue'
import { createProxy, checkAllProxies, checkProxy, deleteProxy, listProxies, updateProxy } from '../api/proxies'
import { useNotify } from '../composables/useNotify'
import { COUNTRIES, countryLabel } from '../constants/countries'
import {
  PROXY_PROTOCOLS,
  PROXY_STATUSES,
  protocolLabel,
  proxyAddress,
  proxyStatusLabel,
} from '../constants/proxies'

const { notifySuccess, notifyError } = useNotify()
const route = useRoute()

// The dashboard links here with filters already applied, e.g. /proxies?status=error.
function queryFilter(key, options) {
  const value = String(route.query[key] || '')
  return options.some((option) => option.value === value) ? value : ''
}

const proxies = ref([])
const loading = ref(true)
const loadError = ref('')
const query = ref(String(route.query.q || ''))
const protocolFilter = ref(queryFilter('protocol', PROXY_PROTOCOLS))
const statusFilter = ref(queryFilter('status', PROXY_STATUSES))
const formOpen = ref(false)
const editing = ref(null)
const deleting = ref(null)
const saving = ref(false)
const formError = ref('')
const checkingId = ref(null)
const updatingStatuses = ref(false)
const checkResults = ref({})

const needsAttention = computed(
  () => proxies.value.filter((p) => p.status === 'error' || p.status === 'inactive').length,
)

const filteredProxies = computed(() => {
  const q = query.value.trim().toLowerCase()
  return proxies.value.filter((proxy) => {
    if (protocolFilter.value && proxy.protocol !== protocolFilter.value) return false
    if (statusFilter.value && proxy.status !== statusFilter.value) return false
    if (!q) return true
    return [
      proxy.name,
      proxy.host,
      proxy.country,
      countryName(proxy.country),
      proxy.username,
      proxy.notes,
      proxyAddress(proxy),
      protocolLabel(proxy.protocol),
    ]
      .join(' ')
      .toLowerCase()
      .includes(q)
  })
})

const deleteMessage = computed(() => {
  if (!deleting.value) return ''
  const count = deleting.value.accountCount || 0
  const linked =
    count === 0
      ? ''
      : ` ${count} account${count === 1 ? '' : 's'} currently use this proxy and will be unassigned.`
  return `Delete ${deleting.value.name} (${proxyAddress(deleting.value)})?${linked} This can't be undone.`
})

function countryName(code) {
  if (!code) return '—'
  const match = COUNTRIES.find((c) => c.code === code)
  return match ? match.name : countryLabel(code)
}

function formatDisplayDateTime(value) {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function checkSummary(proxy) {
  const result = checkResults.value[proxy.id]
  if (!result) return 'Not checked yet'
  if (result.ok) return `Check OK · ${result.latencyMs} ms`
  return 'Check failed'
}

function checkSubClass(proxy) {
  const result = checkResults.value[proxy.id]
  if (!result) return ''
  return result.ok ? 'is-ok' : 'is-bad'
}

function countByStatus(status) {
  return proxies.value.filter((proxy) => proxy.status === status).length
}

async function loadProxies() {
  loading.value = true
  loadError.value = ''
  try {
    proxies.value = await listProxies()
  } catch (err) {
    const message = err.message || 'Could not load proxies. Is the API running?'
    loadError.value = message
    notifyError(message)
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = null
  formError.value = ''
  formOpen.value = true
}

function openEdit(proxy) {
  editing.value = proxy
  formError.value = ''
  formOpen.value = true
}

function closeForm() {
  formOpen.value = false
  editing.value = null
  formError.value = ''
}

function openDelete(proxy) {
  deleting.value = proxy
}

function checkTitle(proxy) {
  const result = checkResults.value[proxy.id]
  if (!result) return 'Check proxy connectivity'
  if (result.ok) return `Last check OK (${result.latencyMs} ms)`
  return `Last check failed: ${result.message}`
}

async function runCheck(proxy) {
  checkingId.value = proxy.id
  try {
    const result = await checkProxy(proxy.id)
    checkResults.value = {
      ...checkResults.value,
      [proxy.id]: {
        ok: result.ok,
        latencyMs: result.latencyMs,
        message: result.message,
      },
    }
    const idx = proxies.value.findIndex((p) => p.id === proxy.id)
    if (idx >= 0 && result.proxy) {
      proxies.value[idx] = { ...proxies.value[idx], ...result.proxy }
    }
    if (result.ok) {
      notifySuccess(`${proxy.name} is reachable (${result.latencyMs} ms). Status set to Active.`)
    } else {
      notifyError(`${proxy.name} check failed: ${result.message}. Status set to Error.`)
    }
  } catch (err) {
    notifyError(err.message || `Could not check ${proxy.name}`)
  } finally {
    checkingId.value = null
  }
}

async function updateAllStatuses() {
  if (updatingStatuses.value || proxies.value.length === 0) return
  updatingStatuses.value = true
  try {
    const summary = await checkAllProxies()
    const nextResults = { ...checkResults.value }
    for (const result of summary.results || []) {
      if (!result?.proxy?.id) continue
      nextResults[result.proxy.id] = {
        ok: result.ok,
        latencyMs: result.latencyMs,
        message: result.message,
      }
      const idx = proxies.value.findIndex((p) => p.id === result.proxy.id)
      if (idx >= 0) {
        proxies.value[idx] = { ...proxies.value[idx], ...result.proxy }
      }
    }
    checkResults.value = nextResults

    if (summary.failed > 0) {
      notifyError(
        `Updated ${summary.total} proxies: ${summary.active} active, ${summary.failed} failed.`,
      )
    } else {
      notifySuccess(`Updated ${summary.total} proxies: all active.`)
    }
  } catch (err) {
    notifyError(err.message || 'Could not update proxy statuses')
  } finally {
    updatingStatuses.value = false
  }
}

async function saveProxy(payload) {
  saving.value = true
  formError.value = ''
  try {
    if (editing.value) {
      await updateProxy(editing.value.id, payload)
      notifySuccess(`Saved changes to ${payload.name}`)
    } else {
      await createProxy(payload)
      notifySuccess(`Added proxy ${payload.name}`)
    }
    closeForm()
    await loadProxies()
  } catch (err) {
    formError.value = err.message || 'Could not save proxy'
    notifyError(formError.value)
  } finally {
    saving.value = false
  }
}

async function confirmDelete() {
  if (!deleting.value) return
  const name = deleting.value.name
  saving.value = true
  try {
    await deleteProxy(deleting.value.id)
    deleting.value = null
    notifySuccess(`Deleted proxy ${name}`)
    await loadProxies()
  } catch (err) {
    notifyError(err.message || 'Could not delete proxy')
    deleting.value = null
  } finally {
    saving.value = false
  }
}

onMounted(loadProxies)
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

.page-head__actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.page-head p {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--text-dim);
}

.page-head .ti {
  font-size: 16px;
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
  font-variant-numeric: tabular-nums;
  margin-top: 4px;
  color: var(--viper-400);
}

.stat__value.is-success {
  color: var(--success);
}

.stat__value.is-warning {
  color: var(--warn);
}

.filters {
  display: flex;
  gap: 8px;
  margin-bottom: 1rem;
  flex-wrap: wrap;
}

.filters input {
  flex: 1;
  min-width: 140px;
}

.filters select {
  width: 140px;
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
  min-width: 1100px;
  border-collapse: collapse;
  font-size: 14px;
  table-layout: fixed;
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

.col-name { width: 12%; }
.col-endpoint { width: 16%; }
.col-country { width: 11%; }
.col-auth { width: 11%; }
.col-accounts { width: 8%; }
.col-notes { width: 12%; }
.col-activity { width: 13%; }
.col-status { width: 9%; }
.col-actions {
  width: 8%;
  text-align: right;
}

tbody tr:hover td {
  background: var(--panel-raised);
}

tbody tr:last-child td {
  border-bottom: none;
}

.stack-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.stack-cell__primary {
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 500;
}

.stack-cell__sub {
  font-size: 12px;
  color: var(--text-faint);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.stack-cell__sub.is-ok {
  color: var(--viper-400);
}

.stack-cell__sub.is-bad {
  color: var(--danger);
}

.mono {
  font-family: var(--mono);
  font-size: 12.5px;
  font-weight: 500;
}

.mono-soft {
  font-family: var(--mono);
}

.notes-cell {
  font-size: 12.5px;
  color: var(--text-dim);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  line-height: 1.35;
}

.muted {
  color: var(--text-dim);
}

.accounts-count {
  display: inline-block;
  font-family: var(--mono);
  font-size: 13px;
  font-variant-numeric: tabular-nums;
  font-weight: 600;
  color: var(--viper-400);
  min-width: 1.5rem;
}

.accounts-count.is-empty {
  color: var(--text-faint);
  font-weight: 500;
}

.actions {
  text-align: right;
  white-space: nowrap;
}

.actions .ti {
  font-size: 16px;
}

.spin {
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.status {
  display: inline-block;
  font-family: var(--mono);
  font-size: 11.5px;
  padding: 5px 11px;
  border-radius: 20px;
}

.status--active {
  background: var(--viper-dim);
  color: var(--viper-400);
}

.status--inactive {
  background: var(--panel-raised);
  color: var(--text-faint);
}

.status--error {
  background: var(--bg-danger);
  color: var(--danger);
}

.empty {
  text-align: center;
  padding: 2rem 1rem;
  color: var(--text-dim);
  font-size: 14px;
  margin: 0;
}

@media (max-width: 800px) {
  table {
    table-layout: auto;
  }

  .filters select {
    width: 100%;
  }
}
</style>
