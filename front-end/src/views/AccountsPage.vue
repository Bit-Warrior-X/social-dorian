<template>
  <div>
    <div class="page-head">
      <div>
        <h1>Social accounts</h1>
        <p>Accounts connected for this organization</p>
      </div>
      <button class="btn btn-primary" type="button" @click="openCreate">
        <i class="ti ti-plus" aria-hidden="true" />
        Connect account
      </button>
    </div>

    <p v-if="loadError" class="banner">{{ loadError }}</p>

    <div class="stats">
      <div class="stat">
        <div class="stat__label">Connected</div>
        <div class="stat__value">{{ accounts.length }}</div>
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
      <input v-model="query" type="text" placeholder="Search accounts" />
      <select v-model="platformFilter">
        <option value="">All platforms</option>
        <option v-for="platform in PLATFORMS" :key="platform.value" :value="platform.value">
          {{ platform.label }}
        </option>
      </select>
      <select v-model="statusFilter">
        <option value="">All statuses</option>
        <option v-for="status in STATUSES" :key="status.value" :value="status.value">
          {{ status.label }}
        </option>
      </select>
    </div>

    <div class="table-card">
      <div v-if="loading" class="empty">Loading accounts…</div>
      <template v-else>
        <div class="table-scroll">
          <table>
            <thead>
              <tr>
                <th class="col-account">Account</th>
                <th class="col-email">Email</th>
                <th class="col-profile">Profile</th>
                <th class="col-status">Status</th>
                <th class="col-proxy">Proxy</th>
                <th class="col-connected">Connected</th>
                <th class="col-actions">Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="account in filteredAccounts" :key="account.id">
                <td>
                  <div class="account-cell">
                    <i class="ti" :class="platformMeta(account.platform).icon" aria-hidden="true" />
                    <div>
                      <div class="account-cell__handle">{{ accountDisplayName(account) }}</div>
                      <div class="account-cell__platform">{{ platformMeta(account.platform).label }}</div>
                    </div>
                  </div>
                </td>
                <td>
                  <div class="stack-cell">
                    <div class="stack-cell__primary mono-soft">{{ account.email || '—' }}</div>
                    <div v-if="account.emailPassword" class="stack-cell__sub">App password set</div>
                  </div>
                </td>
                <td>
                  <div class="stack-cell">
                    <div class="stack-cell__primary">{{ genderLabel(account.gender) || '—' }}</div>
                    <div class="stack-cell__sub">
                      {{ account.birthday ? formatDisplayDate(account.birthday) : 'No birthday' }}
                    </div>
                  </div>
                </td>
                <td>
                  <span class="status" :class="'status--' + account.status">
                    {{ statusLabel(account.status) }}
                  </span>
                </td>
                <td class="proxy">
                  <template v-for="p in [proxyDetails(account)]" :key="account.id + '-proxy'">
                    <div v-if="p.host" class="proxy-cell">
                      <div class="proxy-cell__mode">{{ p.mode }}</div>
                      <div class="proxy-cell__host">{{ p.host }}</div>
                      <div v-if="p.country" class="proxy-cell__country">{{ p.country }}</div>
                    </div>
                    <span v-else class="muted">{{ p.label }}</span>
                  </template>
                </td>
                <td>
                  <div class="stack-cell">
                    <div class="stack-cell__primary">{{ account.connectedBy || '—' }}</div>
                    <div class="stack-cell__sub">
                      {{ account.connectedAt ? formatDisplayDate(account.connectedAt) : '—' }}
                    </div>
                  </div>
                </td>
                <td class="actions">
                  <button
                    class="btn btn-icon"
                    type="button"
                    :disabled="openingBusy && opening?.id === account.id"
                    :aria-label="'Open ' + accountDisplayName(account)"
                    :title="'Open ' + platformMeta(account.platform).label + ' via proxy'"
                    @click="openAccount(account)"
                  >
                    <i
                      class="ti"
                      :class="openingBusy && opening?.id === account.id ? 'ti-loader-2 spin' : 'ti-external-link'"
                      aria-hidden="true"
                    />
                  </button>
                  <button
                    class="btn btn-icon"
                    type="button"
                    :aria-label="'Edit ' + accountDisplayName(account)"
                    @click="openEdit(account)"
                  >
                    <i class="ti ti-edit" aria-hidden="true" />
                  </button>
                  <button
                    class="btn btn-icon"
                    type="button"
                    :aria-label="'Disconnect ' + accountDisplayName(account)"
                    @click="openDelete(account)"
                  >
                    <i class="ti ti-trash" aria-hidden="true" />
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-if="filteredAccounts.length === 0" class="empty">No accounts match your filters.</p>
      </template>
    </div>

    <AccountFormModal
      :open="formOpen"
      :account="editing"
      :saving="saving"
      :error="formError"
      @close="closeForm"
      @submit="saveAccount"
    />

    <OpenAccountModal
      :open="Boolean(opening)"
      :account="opening"
      :session="openSession"
      :loading="openingBusy"
      :error="openError"
      @close="closeOpenSession"
    />

    <DeleteConfirmModal
      :open="Boolean(deleting)"
      :account="deleting"
      :saving="saving"
      @close="deleting = null"
      @confirm="confirmDelete"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import AccountFormModal from '../components/AccountFormModal.vue'
import DeleteConfirmModal from '../components/DeleteConfirmModal.vue'
import OpenAccountModal from '../components/OpenAccountModal.vue'
import { createAccount, closeAccountSession, deleteAccount, listAccounts, openAccountSession, updateAccount } from '../api/accounts'
import { listProxies } from '../api/proxies'
import { useNotify } from '../composables/useNotify'
import {
  PLATFORMS,
  STATUSES,
  accountDisplayName,
  accountProxyDetails,
  genderLabel,
  platformMeta,
  statusLabel,
} from '../constants/accounts'

const { notifySuccess, notifyError } = useNotify()
const route = useRoute()

// The dashboard links here with filters already applied, e.g. /accounts?status=error.
function queryFilter(key, options) {
  const value = String(route.query[key] || '')
  return options.some((option) => option.value === value) ? value : ''
}

const accounts = ref([])
const proxies = ref([])
const loading = ref(true)
const loadError = ref('')
const query = ref(String(route.query.q || ''))
const platformFilter = ref(queryFilter('platform', PLATFORMS))
const statusFilter = ref(queryFilter('status', STATUSES))
const formOpen = ref(false)
const editing = ref(null)
const deleting = ref(null)
const saving = ref(false)
const formError = ref('')
const opening = ref(null)
const openSession = ref(null)
const openingBusy = ref(false)
const openError = ref('')

const needsAttention = computed(
  () => accounts.value.filter((a) => a.status === 'expired' || a.status === 'error').length,
)

const filteredAccounts = computed(() => {
  const q = query.value.trim().toLowerCase()
  return accounts.value.filter((account) => {
    if (platformFilter.value && account.platform !== platformFilter.value) return false
    if (statusFilter.value && account.status !== statusFilter.value) return false
    if (!q) return true
    const platform = platformMeta(account.platform).label.toLowerCase()
    const name = accountDisplayName(account).toLowerCase()
    const email = (account.email || '').toLowerCase()
    const gender = genderLabel(account.gender).toLowerCase()
    const birthday = (account.birthday || '').toLowerCase()
    const connectedBy = (account.connectedBy || '').toLowerCase()
    const proxy = proxyDetails(account).label.toLowerCase()
    return (
      name.includes(q) ||
      email.includes(q) ||
      platform.includes(q) ||
      gender.includes(q) ||
      birthday.includes(q) ||
      connectedBy.includes(q) ||
      proxy.includes(q)
    )
  })
})

function proxyDetails(account) {
  return accountProxyDetails(account, proxies.value)
}

function formatDisplayDate(value) {
  if (!value) return '—'
  const date = new Date(`${value}T00:00:00Z`)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    timeZone: 'UTC',
  })
}

function countByStatus(status) {
  return accounts.value.filter((account) => account.status === status).length
}

async function loadAccounts() {
  loading.value = true
  loadError.value = ''
  try {
    const [accountRows, proxyRows] = await Promise.all([
      listAccounts(),
      listProxies().catch(() => []),
    ])
    accounts.value = accountRows
    proxies.value = proxyRows
  } catch (err) {
    const message = err.message || 'Could not load accounts. Is the API running?'
    loadError.value = message
    notifyError(message)
  } finally {
    loading.value = false
  }
}

async function openAccount(account) {
  opening.value = account
  openSession.value = null
  openError.value = ''
  openingBusy.value = true
  try {
    openSession.value = await openAccountSession(account.id)
    notifySuccess(
      `Opened ${platformMeta(account.platform).label} via ${openSession.value.exitIp || openSession.value.proxy?.host}`,
    )
  } catch (err) {
    openError.value = err.message || 'Could not open this account'
    notifyError(openError.value)
  } finally {
    openingBusy.value = false
  }
}

async function closeOpenSession() {
  const sessionUrl = openSession.value?.sessionUrl
  opening.value = null
  openSession.value = null
  openError.value = ''
  openingBusy.value = false
  if (sessionUrl) {
    try {
      await closeAccountSession(sessionUrl)
    } catch (_) {
      // Session cleanup is best-effort; TTL also reaps abandoned browsers.
    }
  }
}

function openCreate() {
  editing.value = null
  formError.value = ''
  formOpen.value = true
}

function openEdit(account) {
  editing.value = account
  formError.value = ''
  formOpen.value = true
}

function closeForm() {
  formOpen.value = false
  editing.value = null
  formError.value = ''
}

function openDelete(account) {
  deleting.value = account
}

async function saveAccount(payload) {
  saving.value = true
  formError.value = ''
  try {
    const name = accountDisplayName(payload)
    if (editing.value) {
      await updateAccount(editing.value.id, payload)
      notifySuccess(`Saved changes to ${name}`)
    } else {
      await createAccount(payload)
      notifySuccess(`Connected ${name}`)
    }
    closeForm()
    await loadAccounts()
  } catch (err) {
    formError.value = err.message || 'Could not save account'
    notifyError(formError.value)
  } finally {
    saving.value = false
  }
}

async function confirmDelete() {
  if (!deleting.value) return
  const name = accountDisplayName(deleting.value)
  saving.value = true
  try {
    await deleteAccount(deleting.value.id)
    deleting.value = null
    notifySuccess(`Disconnected ${name}`)
    await loadAccounts()
  } catch (err) {
    notifyError(err.message || 'Could not disconnect account')
    deleting.value = null
  } finally {
    saving.value = false
  }
}

onMounted(loadAccounts)
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

.filters select:nth-of-type(1) {
  width: 140px;
}

.filters select:nth-of-type(2) {
  width: 120px;
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
  min-width: 980px;
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

.col-account { width: 18%; }
.col-email { width: 18%; }
.col-profile { width: 12%; }
.col-status { width: 10%; }
.col-proxy { width: 16%; }
.col-connected { width: 14%; }
.col-actions {
  width: 14%;
  text-align: right;
}

tbody tr:hover td {
  background: var(--panel-raised);
}

tbody tr:last-child td {
  border-bottom: none;
}

.account-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.account-cell .ti {
  font-size: 20px;
  color: var(--text-dim);
}

.account-cell__handle {
  font-weight: 500;
}

.account-cell__platform,
.sub {
  font-size: 12px;
  color: var(--text-faint);
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
}

.stack-cell__sub {
  font-size: 12px;
  color: var(--text-faint);
}

.mono-soft {
  font-family: var(--mono);
  font-size: 12.5px;
}

.muted {
  color: var(--text-dim);
}

.proxy {
  white-space: normal;
}

.proxy-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.proxy-cell__mode {
  font-size: 11px;
  font-family: var(--mono);
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--viper-400);
}

.proxy-cell__host {
  font-family: var(--mono);
  font-size: 12.5px;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.proxy-cell__country {
  font-size: 12px;
  color: var(--text-faint);
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

.status--expired {
  background: var(--gold-dim);
  color: var(--gold-500);
}

.status--error {
  background: var(--bg-danger);
  color: var(--danger);
}

.status--revoked {
  background: var(--panel-raised);
  color: var(--text-faint);
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
