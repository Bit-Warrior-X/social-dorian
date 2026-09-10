<template>
  <ModalDialog
    :open="open"
    :title="dialogTitle"
    title-id="new-task-title"
    wide
    @close="emit('close')"
  >
    <form class="form" @submit.prevent="submit">
      <div class="type-grid">
        <button
          v-for="type in TASK_TYPES"
          :key="type.value"
          class="type-card"
          type="button"
          :class="{ active: form.type === type.value }"
          @click="form.type = type.value"
        >
          <i class="ti" :class="type.icon" aria-hidden="true" />
          <strong>{{ type.label }}</strong>
          <span>{{ type.description }}</span>
        </button>
      </div>

      <label class="field">
        <span>Title</span>
        <input v-model.trim="form.title" type="text" :placeholder="taskTypeMeta(form.type).label" />
      </label>

      <label v-if="needsUrl" class="field">
        <span>Post / target URL</span>
        <input
          v-model.trim="form.targetUrl"
          type="url"
          placeholder="https://www.facebook.com/…"
          required
        />
      </label>

      <div class="field">
        <div class="field__row">
          <span>Accounts</span>
          <span class="muted">{{ selectedIds.length }} selected</span>
        </div>
        <div class="account-tools">
          <input v-model="accountQuery" type="search" placeholder="Search accounts" />
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
        <div class="account-list">
          <label v-for="account in filteredAccounts" :key="account.id" class="account-row">
            <input v-model="selectedIds" type="checkbox" :value="account.id" />
            <i class="ti" :class="platformMeta(account.platform).icon" aria-hidden="true" />
            <div class="account-row__body">
              <strong>{{ accountDisplayName(account) }}</strong>
              <span>{{ account.email }} · {{ platformMeta(account.platform).label }}</span>
            </div>
            <span class="status" :class="'status--' + account.status">{{ statusLabel(account.status) }}</span>
          </label>
          <p v-if="filteredAccounts.length === 0" class="empty">No accounts match these filters.</p>
        </div>
      </div>

      <div class="settings">
        <label class="field">
          <span>Delay min (sec)</span>
          <input v-model.number="form.delayMinSec" type="number" min="1" max="120" />
        </label>
        <label class="field">
          <span>Delay max (sec)</span>
          <input v-model.number="form.delayMaxSec" type="number" min="1" max="120" />
        </label>
        <label class="check">
          <input v-model="form.useAccountProxy" type="checkbox" />
          Use each account’s assigned proxy
        </label>
      </div>

      <p v-if="error" class="error">{{ error }}</p>

      <div class="actions">
        <button class="btn" type="button" :disabled="saving" @click="emit('close')">Cancel</button>
        <button class="btn btn-primary" type="submit" :disabled="saving || selectedIds.length === 0">
          <i class="ti" :class="saving ? 'ti-loader-2 spin' : 'ti-rocket'" aria-hidden="true" />
          {{ saving ? 'Launching…' : 'Launch task' }}
        </button>
      </div>
    </form>
  </ModalDialog>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import ModalDialog from './ModalDialog.vue'
import {
  PLATFORMS,
  STATUSES,
  accountDisplayName,
  platformMeta,
  statusLabel,
} from '../constants/accounts'
import { TASK_TYPES, taskTypeMeta } from '../constants/tasks'

const props = defineProps({
  open: { type: Boolean, default: false },
  accounts: { type: Array, default: () => [] },
  initialType: { type: String, default: 'report' },
  initialAccountIds: { type: Array, default: () => [] },
  saving: { type: Boolean, default: false },
  error: { type: String, default: '' },
})

const emit = defineEmits(['close', 'submit'])

const form = reactive({
  type: 'report',
  title: '',
  targetUrl: '',
  delayMinSec: 5,
  delayMaxSec: 15,
  useAccountProxy: true,
})
const selectedIds = ref([])
const accountQuery = ref('')
const platformFilter = ref('')
const statusFilter = ref('')

const dialogTitle = computed(() => `New task · ${taskTypeMeta(form.type).label}`)
const needsUrl = computed(() => taskTypeMeta(form.type).needsUrl)

const filteredAccounts = computed(() => {
  const q = accountQuery.value.trim().toLowerCase()
  return props.accounts.filter((account) => {
    if (platformFilter.value && account.platform !== platformFilter.value) return false
    if (statusFilter.value && account.status !== statusFilter.value) return false
    if (!q) return true
    const hay = `${accountDisplayName(account)} ${account.email} ${account.platform}`.toLowerCase()
    return hay.includes(q)
  })
})

watch(
  () => props.open,
  (open) => {
    if (!open) return
    form.type = props.initialType || 'report'
    form.title = ''
    form.targetUrl = ''
    form.delayMinSec = 5
    form.delayMaxSec = 15
    form.useAccountProxy = true
    selectedIds.value = [...(props.initialAccountIds || [])]
    accountQuery.value = ''
    platformFilter.value = ''
    statusFilter.value = ''
  },
)

function submit() {
  emit('submit', {
    type: form.type,
    title: form.title || taskTypeMeta(form.type).label,
    targetUrl: form.targetUrl,
    accountIds: selectedIds.value.map(Number),
    delayMinSec: Number(form.delayMinSec) || 5,
    delayMaxSec: Number(form.delayMaxSec) || 15,
    useAccountProxy: Boolean(form.useAccountProxy),
  })
}
</script>

<style scoped>
.form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.type-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.type-card {
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: flex-start;
  text-align: left;
  padding: 12px;
  border: 0.5px solid var(--hairline);
  border-radius: 10px;
  background: var(--bg);
  color: var(--text-dim);
}

.type-card .ti {
  font-size: 18px;
  color: var(--viper-400);
}

.type-card strong {
  color: var(--text);
  font-size: 13px;
}

.type-card span {
  font-size: 12px;
  line-height: 1.35;
}

.type-card.active {
  border-color: color-mix(in srgb, var(--viper-500) 55%, transparent);
  background: var(--viper-dim);
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-dim);
}

.field__row {
  display: flex;
  justify-content: space-between;
  gap: 8px;
}

.muted {
  font-weight: 500;
  color: var(--text-faint);
}

.account-tools {
  display: grid;
  grid-template-columns: 1.4fr 1fr 1fr;
  gap: 8px;
}

.account-list {
  max-height: 240px;
  overflow: auto;
  border: 0.5px solid var(--hairline);
  border-radius: 10px;
  background: var(--bg);
}

.account-row {
  display: grid;
  grid-template-columns: auto auto 1fr auto;
  gap: 10px;
  align-items: center;
  padding: 10px 12px;
  border-bottom: 0.5px solid var(--hairline);
  font-weight: 500;
  color: var(--text);
}

.account-row:last-child {
  border-bottom: none;
}

.account-row .ti {
  font-size: 16px;
  color: var(--text-dim);
}

.account-row__body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.account-row__body strong {
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.account-row__body span {
  font-size: 12px;
  color: var(--text-faint);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.status {
  display: inline-block;
  font-family: var(--mono);
  font-size: 11px;
  padding: 4px 8px;
  border-radius: 999px;
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

.settings {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  align-items: end;
}

.check {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text-dim);
}

.error {
  margin: 0;
  padding: 8px 10px;
  border-radius: var(--radius);
  border: 0.5px solid color-mix(in srgb, var(--danger) 35%, transparent);
  background: var(--bg-danger);
  color: var(--danger);
  font-size: 13px;
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.empty {
  margin: 0;
  padding: 1.25rem;
  text-align: center;
  color: var(--text-dim);
  font-size: 13px;
}

.spin {
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 720px) {
  .type-grid,
  .account-tools,
  .settings {
    grid-template-columns: 1fr;
  }
}
</style>
