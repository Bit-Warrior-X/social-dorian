<template>
  <ModalDialog
    :open="open"
    :title="dialogTitle"
    title-id="open-account-title"
    full
    @close="emit('close')"
  >
    <div class="session">
      <div class="session__toolbar">
        <div class="pill" :class="{ 'is-ready': Boolean(session?.exitIp), 'is-error': Boolean(error) }">
          <i class="ti" :class="statusIcon" aria-hidden="true" />
          <span>{{ statusText }}</span>
        </div>

        <template v-if="session">
          <div class="chip">
            <span class="chip__label">IP</span>
            <strong class="mono">{{ session.exitIp || session.proxy?.host || '—' }}</strong>
          </div>
          <div class="chip">
            <span class="chip__label">Proxy</span>
            <strong>
              {{ session.proxy?.name || '—' }}
              <span v-if="session.proxy" class="dim">
                {{ session.proxy.host }}:{{ session.proxy.port }}
                <template v-if="session.proxy.country"> · {{ session.proxy.country }}</template>
              </span>
            </strong>
          </div>
          <div class="chip chip--grow">
            <span class="chip__label">Email</span>
            <strong class="mono">{{ session.email || '—' }}</strong>
            <button
              class="btn btn-icon chip__copy"
              type="button"
              :disabled="!session.email"
              aria-label="Copy email"
              @click="copyText(session.email, 'Email')"
            >
              <i class="ti ti-copy" aria-hidden="true" />
            </button>
          </div>
          <div class="chip chip--grow">
            <span class="chip__label">Pass</span>
            <strong class="mono">{{ session.password || '—' }}</strong>
            <button
              class="btn btn-icon chip__copy"
              type="button"
              :disabled="!session.password"
              aria-label="Copy password"
              @click="copyText(session.password, 'Password')"
            >
              <i class="ti ti-copy" aria-hidden="true" />
            </button>
          </div>
          <div class="chip">
            <span class="chip__label">Profile</span>
            <strong>{{ session.reused ? 'Restored' : 'Saved' }}</strong>
          </div>
          <div v-if="loginLabel" class="chip" :class="loginChipClass">
            <span class="chip__label">Login</span>
            <strong>{{ loginLabel }}</strong>
            <button
              v-if="canRetryLogin"
              class="btn btn-icon chip__copy"
              type="button"
              aria-label="Retry login"
              @click="retryLogin"
            >
              <i class="ti ti-refresh" aria-hidden="true" />
            </button>
          </div>
        </template>
      </div>

      <p v-if="error" class="session__error">{{ error }}</p>
      <p v-else-if="loginMessage && loginStatus && loginStatus !== 'skipped'" class="session__note" :class="{ 'is-bad': loginStatus === 'failed' || loginStatus === 'error' }">
        {{ loginMessage }}
      </p>

      <div class="session__frame-wrap">
        <div v-if="loading" class="session__loading">
          <i class="ti ti-loader-2 spin" aria-hidden="true" />
          Launching remote Chromium through proxy…
        </div>
        <iframe
          v-else-if="frameUrl"
          class="session__frame"
          :src="frameUrl"
          :title="dialogTitle"
          referrerpolicy="no-referrer"
        />
        <div v-else class="session__loading">Waiting for session…</div>
      </div>
    </div>
  </ModalDialog>
</template>

<script setup>
import { computed, onUnmounted, ref, watch } from 'vue'
import ModalDialog from './ModalDialog.vue'
import { getAccountSessionStatus, retryAccountLogin } from '../api/accounts'
import { useNotify } from '../composables/useNotify'
import { accountDisplayName, platformMeta } from '../constants/accounts'

const props = defineProps({
  open: { type: Boolean, default: false },
  account: { type: Object, default: null },
  session: { type: Object, default: null },
  loading: { type: Boolean, default: false },
  error: { type: String, default: '' },
})

const emit = defineEmits(['close'])
const { notifySuccess, notifyError } = useNotify()

const loginStatus = ref('')
const loginMessage = ref('')
let pollTimer = 0

const dialogTitle = computed(() => {
  if (!props.account) return 'Open account'
  return `Open ${accountDisplayName(props.account)} · ${platformMeta(props.account.platform).label}`
})

const frameUrl = computed(() => props.session?.sessionUrl || '')

const statusIcon = computed(() => {
  if (props.error) return 'ti-alert-circle'
  if (props.loading) return 'ti-loader-2 spin'
  if (loginStatus.value === 'signing_in' || loginStatus.value === 'starting' || loginStatus.value === 'verifying') {
    return 'ti-loader-2 spin'
  }
  return 'ti-brand-chrome'
})

const statusText = computed(() => {
  if (props.error) return 'Failed'
  if (props.loading) return 'Starting…'
  if (loginStatus.value === 'starting' || loginStatus.value === 'signing_in') return 'Signing in…'
  if (loginStatus.value === 'verifying') return 'Checking email…'
  if (props.session?.exitIp) return `via ${props.session.exitIp}`
  return 'Ready'
})

const loginLabel = computed(() => {
  switch (loginStatus.value) {
    case 'starting':
    case 'signing_in':
      return 'Signing in…'
    case 'verifying':
      return 'Email code'
    case 'success':
      return 'Signed in'
    case 'checkpoint':
      return 'Checkpoint'
    case 'failed':
      return 'Failed'
    case 'error':
      return 'Error'
    default:
      return loginMessage.value ? loginMessage.value : ''
  }
})

const loginChipClass = computed(() => {
  if (loginStatus.value === 'success') return 'is-ok'
  if (loginStatus.value === 'failed' || loginStatus.value === 'error') return 'is-bad'
  if (loginStatus.value === 'checkpoint' || loginStatus.value === 'verifying') return 'is-warn'
  return ''
})

const canRetryLogin = computed(() =>
  ['failed', 'error', 'checkpoint'].includes(loginStatus.value),
)

const busyLogin = computed(() =>
  ['starting', 'signing_in', 'verifying'].includes(loginStatus.value),
)

watch(
  () => [props.open, props.session?.sessionUrl],
  () => {
    loginStatus.value = props.session?.loginStatus || ''
    loginMessage.value = props.session?.loginMessage || ''
    if (props.open && props.session?.sessionUrl && busyLogin.value) {
      startLoginPoll()
    } else if (!props.open || !props.session?.sessionUrl) {
      stopLoginPoll()
    }
  },
  { immediate: true },
)

async function pullLoginStatus() {
  if (!props.session?.sessionUrl) return
  try {
    const data = await getAccountSessionStatus(props.session.sessionUrl)
    if (!data) return
    loginStatus.value = data.loginStatus || ''
    loginMessage.value = data.loginMessage || ''
    if (!['starting', 'signing_in', 'verifying'].includes(loginStatus.value)) {
      stopLoginPoll()
    }
  } catch (_) {
    // Keep the last known status if a poll fails mid-login.
  }
}

function startLoginPoll() {
  stopLoginPoll()
  pullLoginStatus()
  pollTimer = window.setInterval(pullLoginStatus, 2000)
}

function stopLoginPoll() {
  if (pollTimer) {
    window.clearInterval(pollTimer)
    pollTimer = 0
  }
}

async function retryLogin() {
  if (!props.session?.sessionUrl) return
  try {
    const data = await retryAccountLogin(props.session.sessionUrl)
    loginStatus.value = data?.loginStatus || 'starting'
    loginMessage.value = data?.loginMessage || 'Retrying login…'
    startLoginPoll()
    notifySuccess('Retrying Facebook login')
  } catch (err) {
    notifyError(err?.message || 'Could not retry login')
  }
}

onUnmounted(stopLoginPoll)

async function copyText(value, label) {
  const text = String(value || '').trim()
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
    notifySuccess(`${label} copied`)
  } catch (err) {
    notifyError(err?.message || `Could not copy ${label.toLowerCase()}`)
  }
}
</script>

<style scoped>
.session {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 0;
  flex: 1;
  height: 100%;
}

.session__toolbar {
  display: flex;
  align-items: center;
  gap: 6px;
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

.pill.is-error {
  border-color: color-mix(in srgb, var(--danger) 45%, transparent);
  background: var(--bg-danger);
  color: var(--danger);
}

.chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-width: 0;
  max-width: 100%;
  padding: 3px 7px;
  border: 0.5px solid var(--hairline);
  border-radius: 7px;
  background: var(--bg);
  font-size: 11.5px;
}

.chip--grow {
  flex: 1 1 160px;
}

.chip.is-ok {
  border-color: color-mix(in srgb, var(--viper-500) 45%, transparent);
  background: var(--viper-dim);
}

.chip.is-bad {
  border-color: color-mix(in srgb, var(--danger) 45%, transparent);
  background: var(--bg-danger);
}

.chip.is-warn {
  border-color: color-mix(in srgb, var(--warning, #d4a017) 45%, transparent);
}

.chip__label {
  font-family: var(--mono);
  font-size: 9.5px;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--text-faint);
  flex-shrink: 0;
}

.chip strong {
  font-size: 11.5px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.chip__copy {
  margin-left: auto;
  padding: 1px 4px;
  flex-shrink: 0;
}

.chip__copy .ti {
  font-size: 13px;
}

.mono {
  font-family: var(--mono);
  font-weight: 500;
}

.dim {
  color: var(--text-faint);
  font-weight: 500;
}

.session__error,
.session__note {
  margin: 0;
  padding: 5px 8px;
  border-radius: var(--radius);
  border: 0.5px solid color-mix(in srgb, var(--danger) 35%, transparent);
  background: var(--bg-danger);
  color: var(--danger);
  font-size: 12px;
  flex-shrink: 0;
}

.session__note {
  border-color: var(--hairline);
  background: var(--panel-raised);
  color: var(--text-dim);
}

.session__note.is-bad {
  border-color: color-mix(in srgb, var(--danger) 35%, transparent);
  background: var(--bg-danger);
  color: var(--danger);
}

.session__frame-wrap {
  position: relative;
  flex: 1 1 auto;
  min-height: 0;
  border: 0.5px solid var(--hairline);
  border-radius: 8px;
  overflow: hidden;
  background: #111;
}

.session__frame {
  width: 100%;
  height: 100%;
  border: 0;
  background: #111;
}

.session__loading {
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

.session__loading .ti {
  font-size: 20px;
  color: var(--viper-400);
}

.spin {
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 900px) {
  .chip--grow {
    flex-basis: 100%;
  }
}
</style>
