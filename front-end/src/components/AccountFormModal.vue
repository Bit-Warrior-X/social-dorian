<template>
  <ModalDialog
    :open="open"
    :title="isEdit ? 'Edit account' : 'Connect account'"
    wide
    @close="emit('close')"
  >
    <form class="form" @submit.prevent="submit">
      <section class="section">
        <header class="section__head">
          <div class="section__icon" aria-hidden="true">
            <i class="ti ti-share" />
          </div>
          <div>
            <h3>Platform</h3>
            <p>Which network this account belongs to</p>
          </div>
        </header>
        <div class="platform-grid">
          <label
            v-for="platform in PLATFORMS"
            :key="platform.value"
            class="platform-chip"
            :class="{ active: form.platform === platform.value }"
          >
            <input v-model="form.platform" type="radio" name="platform" :value="platform.value" />
            <i class="ti" :class="platform.icon" aria-hidden="true" />
            <span>{{ platform.label }}</span>
          </label>
        </div>
      </section>

      <div class="section-grid">
        <section class="section">
          <header class="section__head">
            <div class="section__icon" aria-hidden="true">
              <i class="ti ti-user" />
            </div>
            <div>
              <h3>Profile</h3>
              <p>Identity details for this account</p>
            </div>
          </header>
          <div class="fields">
            <div class="row">
              <div class="field">
                <label for="fFirstName">First name</label>
                <input id="fFirstName" v-model="form.firstName" type="text" placeholder="Jane" />
              </div>
              <div class="field">
                <label for="fLastName">Last name</label>
                <input id="fLastName" v-model="form.lastName" type="text" placeholder="Ortiz" />
              </div>
            </div>
            <div class="row">
              <div class="field">
                <label for="fBirthday">Birthday</label>
                <DatePicker v-model="form.birthday" input-id="fBirthday" label="Birthday" />
              </div>
              <div class="field">
                <label for="fGender">Gender</label>
                <select id="fGender" v-model="form.gender">
                  <option v-for="gender in GENDERS" :key="gender.value" :value="gender.value">
                    {{ gender.label }}
                  </option>
                </select>
              </div>
            </div>
          </div>
        </section>

        <section class="section">
          <header class="section__head">
            <div class="section__icon" aria-hidden="true">
              <i class="ti ti-lock" />
            </div>
            <div>
              <h3>Credentials</h3>
              <p>Login and mailbox access</p>
            </div>
          </header>
          <div class="fields">
            <div class="field">
              <label for="fPassword">Account password</label>
              <input
                id="fPassword"
                v-model="form.password"
                type="password"
                autocomplete="new-password"
                placeholder="••••••••"
              />
            </div>
            <div class="field">
              <label for="fEmail">Email</label>
              <input id="fEmail" v-model="form.email" type="email" placeholder="name@example.com" />
            </div>
            <div class="field">
              <label for="fEmailPassword">Email app password</label>
              <input
                id="fEmailPassword"
                v-model="form.emailPassword"
                type="password"
                autocomplete="new-password"
                placeholder="App-specific email password"
              />
            </div>
          </div>
        </section>
      </div>

      <section class="section">
        <header class="section__head">
          <div class="section__icon" aria-hidden="true">
            <i class="ti ti-network" />
          </div>
          <div>
            <h3>Proxy</h3>
            <p>How this account routes outbound traffic</p>
          </div>
        </header>

        <div class="radio-list" role="radiogroup" aria-label="Proxy setting">
          <label
            v-for="mode in PROXY_MODES"
            :key="mode.value"
            class="radio-card"
            :class="{ active: form.proxyMode === mode.value }"
          >
            <input
              v-model="form.proxyMode"
              type="radio"
              name="proxyMode"
              :value="mode.value"
              @change="onProxyModeChange"
            />
            <span class="radio-card__body">
              <span class="radio-card__title">{{ mode.label }}</span>
              <span v-if="mode.description" class="radio-card__desc">{{ mode.description }}</span>
            </span>
          </label>
        </div>

        <div v-if="form.proxyMode === 'auto'" class="proxy-panel">
          <div v-if="proxiesLoading" class="hint">Loading proxies…</div>
          <div v-else-if="!selectedProxy" class="hint warn">
            No proxies registered yet. Add one in Proxy management first.
          </div>
          <template v-else>
            <div class="proxy-panel__head">
              <span>Auto-selected proxy</span>
              <button class="btn btn-icon" type="button" @click="pickRandomProxy">
                <i class="ti ti-refresh" aria-hidden="true" />
                Reselect
              </button>
            </div>
            <div class="proxy-info">
              <div class="proxy-info__name">{{ selectedProxy.name }}</div>
              <div class="proxy-info__grid">
                <div>
                  <span>Address</span>
                  <strong>{{ proxyAddress(selectedProxy) }}</strong>
                </div>
                <div>
                  <span>Protocol</span>
                  <strong>{{ selectedProxy.protocol }}</strong>
                </div>
                <div>
                  <span>Country</span>
                  <strong>{{ selectedProxy.country || '—' }}</strong>
                </div>
                <div>
                  <span>Status</span>
                  <strong :class="'status-text status-text--' + selectedProxy.status">
                    {{ selectedProxy.status }}
                  </strong>
                </div>
                <div v-if="selectedProxy.username">
                  <span>Auth</span>
                  <strong>{{ selectedProxy.username }}</strong>
                </div>
              </div>
            </div>
          </template>
        </div>

        <div v-if="form.proxyMode === 'manual'" class="proxy-panel">
          <div class="field">
            <label for="fProxyId">Registered proxy</label>
            <select id="fProxyId" v-model="form.proxyId">
              <option disabled value="">Select a proxy</option>
              <option v-for="proxy in proxies" :key="proxy.id" :value="String(proxy.id)">
                {{ proxyOptionLabel(proxy) }}
              </option>
            </select>
          </div>
          <div v-if="selectedProxy" class="proxy-info compact">
            <div class="proxy-info__grid">
              <div>
                <span>Address</span>
                <strong>{{ proxyAddress(selectedProxy) }}</strong>
              </div>
              <div>
                <span>Country</span>
                <strong>{{ selectedProxy.country || '—' }}</strong>
              </div>
              <div>
                <span>Status</span>
                <strong :class="'status-text status-text--' + selectedProxy.status">
                  {{ selectedProxy.status }}
                </strong>
              </div>
            </div>
          </div>
          <p v-if="!proxiesLoading && proxies.length === 0" class="hint warn">
            No proxies registered yet. Add one in Proxy management first.
          </p>
          <p v-if="proxiesLoading" class="hint">Loading proxies…</p>
        </div>
      </section>

      <p v-if="fieldError" class="field-error">{{ fieldError }}</p>
      <p v-if="error" class="field-error">{{ error }}</p>

      <div class="actions">
        <button class="btn" type="button" @click="emit('close')">Cancel</button>
        <button class="btn btn-primary" type="submit" :disabled="saving">
          <i class="ti ti-check" aria-hidden="true" />
          {{ saving ? 'Saving…' : isEdit ? 'Save changes' : 'Connect' }}
        </button>
      </div>
    </form>
  </ModalDialog>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import ModalDialog from './ModalDialog.vue'
import DatePicker from './DatePicker.vue'
import { listProxies } from '../api/proxies'
import { GENDERS, PLATFORMS, PROXY_MODES } from '../constants/accounts'
import { proxyAddress } from '../constants/proxies'

const emptyForm = () => ({
  platform: 'twitter',
  firstName: '',
  lastName: '',
  password: '',
  email: '',
  emailPassword: '',
  birthday: '',
  gender: 'prefer_not_to_say',
  proxyMode: 'none',
  proxyId: '',
})

const props = defineProps({
  open: { type: Boolean, default: false },
  account: { type: Object, default: null },
  saving: { type: Boolean, default: false },
  error: { type: String, default: '' },
})

const emit = defineEmits(['close', 'submit'])

const form = reactive(emptyForm())
const isEdit = ref(false)
const fieldError = ref('')
const proxies = ref([])
const proxiesLoading = ref(false)

const selectedProxy = computed(() => {
  if (!form.proxyId) return null
  return proxies.value.find((p) => String(p.id) === String(form.proxyId)) || null
})

watch(
  () => [props.open, props.account],
  async () => {
    if (!props.open) return
    Object.assign(form, emptyForm())
    fieldError.value = ''
    isEdit.value = Boolean(props.account)
    if (props.account) {
      Object.assign(form, {
        platform: props.account.platform,
        firstName: props.account.firstName || '',
        lastName: props.account.lastName || '',
        password: props.account.password || '',
        email: props.account.email || '',
        emailPassword: props.account.emailPassword || '',
        birthday: props.account.birthday || '',
        gender: props.account.gender || 'prefer_not_to_say',
        proxyMode: props.account.proxyMode || 'none',
        proxyId: props.account.proxyId ? String(props.account.proxyId) : '',
      })
    }
    await loadProxies()
    if (form.proxyMode === 'auto' && !form.proxyId) {
      pickRandomProxy()
    }
  },
  { immediate: true },
)

async function loadProxies() {
  proxiesLoading.value = true
  try {
    proxies.value = await listProxies()
  } catch {
    proxies.value = []
  } finally {
    proxiesLoading.value = false
  }
}

function onProxyModeChange() {
  if (form.proxyMode === 'none') {
    form.proxyId = ''
    return
  }
  if (form.proxyMode === 'auto') {
    pickRandomProxy()
    return
  }
  if (form.proxyMode === 'manual' && form.proxyId) {
    const stillExists = proxies.value.some((p) => String(p.id) === String(form.proxyId))
    if (!stillExists) form.proxyId = ''
  }
}

function pickRandomProxy() {
  const active = proxies.value.filter((p) => p.status === 'active')
  const pool = active.length ? active : proxies.value
  if (!pool.length) {
    form.proxyId = ''
    return
  }
  const pick = pool[Math.floor(Math.random() * pool.length)]
  form.proxyId = String(pick.id)
}

function proxyOptionLabel(proxy) {
  const base = `${proxy.name} — ${proxyAddress(proxy)}`
  return proxy.status === 'active' ? base : `${base} (${proxy.status})`
}

function submit() {
  if (!form.firstName.trim() || !form.lastName.trim()) {
    fieldError.value = 'Enter first name and last name'
    return
  }
  if (!form.email.trim()) {
    fieldError.value = 'Enter an email address'
    return
  }
  if (!form.password.trim()) {
    fieldError.value = 'Enter a password'
    return
  }
  if (form.proxyMode === 'manual' && !form.proxyId) {
    fieldError.value = 'Select a registered proxy'
    return
  }
  if (form.proxyMode === 'auto' && !form.proxyId) {
    fieldError.value = 'No proxy available for auto select'
    return
  }
  fieldError.value = ''
  emit('submit', {
    platform: form.platform,
    firstName: form.firstName.trim(),
    lastName: form.lastName.trim(),
    password: form.password,
    email: form.email.trim(),
    emailPassword: form.emailPassword,
    birthday: form.birthday,
    gender: form.gender,
    proxyMode: form.proxyMode,
    proxyId: form.proxyMode === 'none' ? 0 : Number(form.proxyId),
  })
}
</script>

<style scoped>
.form {
  display: flex;
  flex-direction: column;
  gap: 14px;
  max-height: min(78vh, 760px);
  overflow-y: auto;
  padding-right: 6px;
  scrollbar-gutter: stable;
}

.section {
  border: 0.5px solid var(--hairline);
  border-radius: 12px;
  background: color-mix(in srgb, var(--panel-raised) 70%, var(--panel));
  padding: 14px;
}

.section__head {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  margin-bottom: 12px;
}

.section__icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: grid;
  place-items: center;
  background: var(--viper-dim);
  color: var(--viper-400);
  flex-shrink: 0;
}

.section__icon .ti {
  font-size: 16px;
}

.section__head h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}

.section__head p {
  margin: 2px 0 0;
  font-size: 12px;
  color: var(--text-faint);
}

.section-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}

.fields {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.field label {
  font-size: 12.5px;
  color: var(--text-dim);
  display: block;
  margin-bottom: 4px;
}

.row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.platform-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
}

.platform-chip {
  position: relative;
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  padding: 10px 12px;
  border: 0.5px solid var(--hairline);
  border-radius: var(--radius);
  background: var(--bg);
  cursor: pointer;
  color: var(--text-dim);
  font-size: 12.5px;
  font-weight: 500;
}

.platform-chip .ti {
  font-size: 16px;
  color: var(--text-faint);
}

.platform-chip.active {
  border-color: var(--viper-500);
  background: var(--viper-dim);
  color: var(--text);
}

.platform-chip.active .ti {
  color: var(--viper-400);
}

.platform-chip input {
  position: absolute;
  opacity: 0;
  pointer-events: none;
  width: 1px;
  height: 1px;
}

.platform-chip:focus-within {
  outline: 2px solid color-mix(in srgb, var(--viper-500) 45%, transparent);
  outline-offset: 1px;
}

.radio-list {
  display: flex;
  flex-direction: row;
  gap: 8px;
}

.radio-card {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  flex: 1;
  min-width: 0;
  margin: 0;
  padding: 10px 12px;
  border: 0.5px solid var(--hairline);
  border-radius: var(--radius);
  background: var(--bg);
  cursor: pointer;
  position: relative;
}

.radio-card.active {
  border-color: var(--viper-500);
  background: var(--viper-dim);
}

.radio-card input {
  position: absolute;
  opacity: 0;
  width: 1px;
  height: 1px;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
  pointer-events: none;
}

.radio-card:focus-within {
  outline: 2px solid color-mix(in srgb, var(--viper-500) 45%, transparent);
  outline-offset: 1px;
}

.radio-card__body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.radio-card__title {
  font-size: 13.5px;
  font-weight: 600;
  color: var(--text);
}

.radio-card__desc {
  font-size: 12px;
  color: var(--text-faint);
  line-height: 1.35;
}

.proxy-panel {
  margin-top: 12px;
}

.proxy-panel__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
  font-size: 12px;
  color: var(--text-dim);
}

.proxy-panel__head .btn-icon {
  gap: 4px;
  color: var(--viper-400);
}

.proxy-panel__head .ti {
  font-size: 14px;
}

.proxy-info {
  border: 0.5px solid var(--hairline);
  border-radius: var(--radius);
  background: var(--bg);
  padding: 12px;
}

.proxy-info.compact {
  margin-top: 10px;
}

.proxy-info__name {
  font-weight: 600;
  font-size: 14px;
  margin-bottom: 10px;
}

.proxy-info__grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 10px 16px;
}

.proxy-info__grid > div {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.proxy-info__grid span {
  font-size: 11px;
  color: var(--text-faint);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  font-family: var(--mono);
}

.proxy-info__grid strong {
  font-weight: 500;
  color: var(--text);
  font-family: var(--mono);
  font-size: 12.5px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.status-text--active {
  color: var(--viper-400) !important;
}

.status-text--inactive {
  color: var(--text-faint) !important;
}

.status-text--error {
  color: var(--danger) !important;
}

.hint {
  margin-top: 6px;
  font-size: 12px;
  color: var(--text-faint);
  line-height: 1.4;
}

.hint.warn {
  color: var(--warn);
}

.field-error {
  display: block;
  margin-top: 2px;
  font-size: 13px;
  color: var(--danger);
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 4px;
  position: sticky;
  bottom: 0;
  background: linear-gradient(180deg, transparent, var(--panel) 28%);
  padding-top: 12px;
}

.actions .ti {
  font-size: 16px;
}

@media (max-width: 860px) {
  .section-grid {
    grid-template-columns: 1fr;
  }

  .platform-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 700px) {
  .radio-list {
    flex-direction: column;
  }

  .proxy-info__grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 560px) {
  .row,
  .platform-grid {
    grid-template-columns: 1fr;
  }
}
</style>
