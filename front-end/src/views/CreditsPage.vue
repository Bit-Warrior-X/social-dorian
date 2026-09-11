<template>
  <div>
    <div class="page-head">
      <div>
        <h1>Credit balance</h1>
        <p>Credits used when launching report, post, and browse jobs</p>
      </div>
    </div>

    <p v-if="error" class="banner">{{ error }}</p>

    <div class="panel">
      <div class="panel__label">Available credits</div>
      <div class="panel__value">{{ loading ? '…' : balance.toLocaleString() }}</div>
      <p class="panel__meta">
        Updated {{ updatedAt ? formatWhen(updatedAt) : '—' }}
      </p>
      <p class="panel__note">
        Launching a task debits {{ costPerAccount }} credit per selected account up front.
        Workers run Facebook automation for report, reply, post, browse, and login test.
      </p>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { getCredits } from '../api/tasks'
import { useNotify } from '../composables/useNotify'

const { notifyError } = useNotify()
const balance = ref(0)
const updatedAt = ref('')
const costPerAccount = ref(1)
const loading = ref(true)
const error = ref('')

function formatWhen(value) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

onMounted(async () => {
  try {
    const data = await getCredits()
    balance.value = Number(data.balance || 0)
    updatedAt.value = data.updatedAt || ''
    costPerAccount.value = Number(data.costPerAccount || 1)
  } catch (err) {
    error.value = err.message || 'Could not load credits'
    notifyError(error.value)
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.page-head {
  margin-bottom: 1rem;
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

.panel {
  max-width: 420px;
  padding: 20px;
  border: 0.5px solid var(--hairline);
  border-radius: 12px;
  background: var(--panel);
}

.panel__label {
  font-family: var(--mono);
  font-size: 11px;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--text-faint);
}

.panel__value {
  margin-top: 8px;
  font-family: var(--mono);
  font-size: 40px;
  font-weight: 600;
  color: var(--viper-400);
}

.panel__meta,
.panel__note {
  margin: 8px 0 0;
  font-size: 13px;
  color: var(--text-dim);
}
</style>
