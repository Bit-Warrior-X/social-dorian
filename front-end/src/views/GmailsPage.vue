<template>
  <div>
    <div class="page-head">
      <div>
        <h1>Gmail management</h1>
        <p>Mailboxes available for social account connections</p>
      </div>
      <button class="btn btn-primary" type="button" @click="openCreate">
        <i class="ti ti-plus" aria-hidden="true" />
        Add Gmail
      </button>
    </div>

    <p v-if="loadError" class="banner">{{ loadError }}</p>

    <div class="stats">
      <div class="stat">
        <div class="stat__label">Total</div>
        <div class="stat__value">{{ gmails.length }}</div>
      </div>
      <div class="stat">
        <div class="stat__label">Active</div>
        <div class="stat__value is-success">{{ countByStatus('active') }}</div>
      </div>
      <div class="stat">
        <div class="stat__label">Needs attention</div>
        <div class="stat__value is-warning">{{ needsAttention }}</div>
      </div>
      <div class="stat">
        <div class="stat__label">Linked accounts</div>
        <div class="stat__value">{{ linkedAccounts }}</div>
      </div>
    </div>

    <div class="filters">
      <input v-model="query" type="text" placeholder="Search Gmail mailboxes" />
      <select v-model="statusFilter">
        <option value="">All statuses</option>
        <option v-for="status in GMAIL_STATUSES" :key="status.value" :value="status.value">
          {{ status.label }}
        </option>
      </select>
    </div>

    <div class="table-card">
      <div v-if="loading" class="empty">Loading Gmail mailboxes…</div>
      <template v-else>
        <div class="table-scroll">
          <table>
            <thead>
              <tr>
                <th class="col-mailbox">Mailbox</th>
                <th class="col-credentials">Credentials</th>
                <th class="col-recovery">Recovery</th>
                <th class="col-twofa">2FA</th>
                <th class="col-pin">PIN</th>
                <th class="col-accounts">Accounts</th>
                <th class="col-status">Status</th>
                <th class="col-actions">Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="gmail in filteredGmails" :key="gmail.id">
                <td>
                  <div class="stack-cell">
                    <div class="stack-cell__primary">{{ gmailDisplayName(gmail) }}</div>
                    <div class="stack-cell__sub mono">{{ gmail.email }}</div>
                  </div>
                </td>
                <td>
                  <div class="stack-cell">
                    <div class="stack-cell__primary">
                      Password: <span :class="gmail.password ? 'is-set' : 'is-missing'">{{ gmail.password ? 'Set' : 'Missing' }}</span>
                    </div>
                    <div class="stack-cell__sub">
                      App password: <span :class="gmail.appPassword ? 'is-set' : 'is-missing'">{{ gmail.appPassword ? 'Set' : 'Missing' }}</span>
                    </div>
                  </div>
                </td>
                <td>
                  <div class="stack-cell">
                    <div class="stack-cell__primary">
                      {{ gmail.recoveryEmail || 'No recovery email' }}
                    </div>
                    <div class="stack-cell__sub">
                      {{ gmail.recoveryPhone || 'No recovery phone' }}
                    </div>
                  </div>
                </td>
                <td>
                  <div class="stack-cell">
                    <div class="stack-cell__primary">
                      <span :class="gmail.twofaSecret ? 'is-set' : 'is-missing'">
                        {{ gmail.twofaSecret ? 'Enabled' : 'Not set' }}
                      </span>
                    </div>
                    <div class="stack-cell__sub">
                      {{ backupCodeCount(gmail) }}
                    </div>
                  </div>
                </td>
                <td>
                  <div class="stack-cell">
                    <div class="stack-cell__primary mono">{{ gmail.pinCode || '—' }}</div>
                    <div class="stack-cell__sub">
                      {{ gmail.pinCode ? 'Fetched' : 'Not fetched' }}
                    </div>
                  </div>
                </td>
                <td>
                  <span class="accounts-count" :class="{ 'is-empty': !gmail.accountCount }">
                    {{ gmail.accountCount || 0 }}
                  </span>
                </td>
                <td>
                  <span class="status" :class="'status--' + gmail.status">
                    {{ gmailStatusLabel(gmail.status) }}
                  </span>
                </td>
                <td class="actions">
                  <button
                    class="btn btn-icon"
                    type="button"
                    :aria-label="'Edit ' + gmailDisplayName(gmail)"
                    @click="openEdit(gmail)"
                  >
                    <i class="ti ti-edit" aria-hidden="true" />
                  </button>
                  <button
                    class="btn btn-icon"
                    type="button"
                    :disabled="pinLoadingId === gmail.id"
                    :aria-label="'Get PIN for ' + gmailDisplayName(gmail)"
                    @click="fetchPin(gmail)"
                  >
                    <i
                      class="ti"
                      :class="pinLoadingId === gmail.id ? 'ti-loader-2 spin' : 'ti-key'"
                      aria-hidden="true"
                    />
                  </button>
                  <button
                    class="btn btn-icon"
                    type="button"
                    :aria-label="'Delete ' + gmailDisplayName(gmail)"
                    @click="openDelete(gmail)"
                  >
                    <i class="ti ti-trash" aria-hidden="true" />
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-if="filteredGmails.length === 0" class="empty">No Gmail mailboxes match your filters.</p>
      </template>
    </div>

    <GmailFormModal
      :open="formOpen"
      :gmail="editing"
      :saving="saving"
      :error="formError"
      @close="closeForm"
      @submit="saveGmail"
    />

    <ConfirmDialog
      :open="Boolean(deleting)"
      title="Delete Gmail"
      title-id="delete-gmail-title"
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
import GmailFormModal from '../components/GmailFormModal.vue'
import { createGmail, deleteGmail, getGmailPinCode, listGmails, updateGmail } from '../api/gmails'
import { useNotify } from '../composables/useNotify'
import { GMAIL_STATUSES, gmailDisplayName, gmailStatusLabel } from '../constants/gmails'

const { notifySuccess, notifyError } = useNotify()
const route = useRoute()

const gmails = ref([])
const loading = ref(true)
const loadError = ref('')
// The dashboard links here with a filter already applied, e.g. /gmails?status=error.
const query = ref(String(route.query.q || ''))
const statusFilter = ref(
  GMAIL_STATUSES.some((status) => status.value === route.query.status)
    ? String(route.query.status)
    : '',
)
const formOpen = ref(false)
const editing = ref(null)
const deleting = ref(null)
const saving = ref(false)
const formError = ref('')
const pinLoadingId = ref(null)

const needsAttention = computed(
  () => gmails.value.filter((g) => g.status === 'error' || g.status === 'inactive').length,
)

const linkedAccounts = computed(
  () => gmails.value.reduce((sum, g) => sum + (g.accountCount || 0), 0),
)

const filteredGmails = computed(() => {
  const q = query.value.trim().toLowerCase()
  return gmails.value.filter((gmail) => {
    if (statusFilter.value && gmail.status !== statusFilter.value) return false
    if (!q) return true
    return [
      gmail.label,
      gmail.email,
      gmail.recoveryEmail,
      gmail.recoveryPhone,
      gmail.pinCode,
      gmail.notes,
      gmailStatusLabel(gmail.status),
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
      : ` ${count} social account${count === 1 ? '' : 's'} currently use this email.`
  return `Delete ${gmailDisplayName(deleting.value)} (${deleting.value.email})?${linked} This can't be undone.`
})

function countByStatus(status) {
  return gmails.value.filter((gmail) => gmail.status === status).length
}

function backupCodeCount(gmail) {
  if (!gmail.backupCodes) return 'No backup codes'
  const count = gmail.backupCodes.split(/\r?\n/).filter((line) => line.trim()).length
  return `${count} backup code${count === 1 ? '' : 's'}`
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

async function loadGmails() {
  loading.value = true
  loadError.value = ''
  try {
    gmails.value = await listGmails()
  } catch (err) {
    const message = err.message || 'Could not load Gmail mailboxes. Is the API running?'
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

function openEdit(gmail) {
  editing.value = gmail
  formError.value = ''
  formOpen.value = true
}

function closeForm() {
  formOpen.value = false
  editing.value = null
  formError.value = ''
}

function openDelete(gmail) {
  deleting.value = gmail
}

async function saveGmail(payload) {
  saving.value = true
  formError.value = ''
  try {
    if (editing.value) {
      await updateGmail(editing.value.id, payload)
      notifySuccess(`Saved changes to ${payload.label || payload.email}`)
    } else {
      await createGmail(payload)
      notifySuccess(`Added Gmail ${payload.email}`)
    }
    closeForm()
    await loadGmails()
  } catch (err) {
    formError.value = err.message || 'Could not save Gmail'
    notifyError(formError.value)
  } finally {
    saving.value = false
  }
}

async function confirmDelete() {
  if (!deleting.value) return
  const name = gmailDisplayName(deleting.value)
  saving.value = true
  try {
    await deleteGmail(deleting.value.id)
    deleting.value = null
    notifySuccess(`Deleted ${name}`)
    await loadGmails()
  } catch (err) {
    notifyError(err.message || 'Could not delete Gmail')
    deleting.value = null
  } finally {
    saving.value = false
  }
}

async function fetchPin(gmail) {
  const id = gmail?.id
  if (!id) return

  pinLoadingId.value = id
  try {
    const res = await getGmailPinCode(id)
    const pinCode = res?.pinCode || ''

    const idx = gmails.value.findIndex((g) => g.id === id)
    if (idx >= 0) gmails.value[idx] = { ...gmails.value[idx], pinCode }

    notifySuccess(`Fetched PIN for ${gmailDisplayName(gmail)}`)
  } catch (err) {
    notifyError(err.message || 'Could not fetch PIN code')
  } finally {
    pinLoadingId.value = null
  }
}

onMounted(loadGmails)
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

.col-mailbox { width: 17%; }
.col-credentials { width: 14%; }
.col-recovery { width: 17%; }
.col-twofa { width: 12%; }
.col-pin { width: 13%; }
.col-accounts { width: 8%; }
.col-status { width: 10%; }
.col-actions {
  width: 9%;
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
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.stack-cell__sub {
  font-size: 12px;
  color: var(--text-faint);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mono {
  font-family: var(--mono);
  font-size: 12.5px;
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

.is-set {
  color: var(--viper-400);
  font-weight: 500;
}

.is-missing {
  color: var(--text-faint);
}

.accounts-count {
  display: inline-block;
  font-family: var(--mono);
  font-size: 13px;
  font-variant-numeric: tabular-nums;
  font-weight: 600;
  color: var(--viper-400);
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
