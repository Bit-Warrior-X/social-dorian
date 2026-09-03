<template>
  <ModalDialog
    :open="open"
    :title="isEdit ? 'Edit proxy' : 'Add proxy'"
    wide
    @close="emit('close')"
  >
    <form class="form" @submit.prevent="submit">
      <section class="section">
        <header class="section__head">
          <div class="section__icon" aria-hidden="true">
            <i class="ti ti-tag" />
          </div>
          <div>
            <h3>Identity</h3>
            <p>Display name for this proxy</p>
          </div>
        </header>
        <div class="field">
          <label for="pName">Name</label>
          <input id="pName" v-model="form.name" type="text" placeholder="US East" />
        </div>
      </section>

      <section class="section">
        <header class="section__head">
          <div class="section__icon" aria-hidden="true">
            <i class="ti ti-network" />
          </div>
          <div>
            <h3>Endpoint</h3>
            <p>Protocol and connection address</p>
          </div>
        </header>

        <div class="protocol-grid" role="radiogroup" aria-label="Protocol">
          <label
            v-for="protocol in PROXY_PROTOCOLS"
            :key="protocol.value"
            class="protocol-chip"
            :class="{ active: form.protocol === protocol.value }"
          >
            <input v-model="form.protocol" type="radio" name="protocol" :value="protocol.value" />
            <i class="ti" :class="protocol.icon" aria-hidden="true" />
            <span class="protocol-chip__body">
              <span class="protocol-chip__title">{{ protocol.label }}</span>
              <span class="protocol-chip__desc">{{ protocol.description }}</span>
            </span>
          </label>
        </div>

        <div class="fields endpoint-fields">
          <div class="row endpoint-row">
            <div class="field host">
              <label for="pHost">Host</label>
              <input id="pHost" v-model="form.host" type="text" placeholder="proxy.example.com" />
            </div>
            <div class="field port">
              <label for="pPort">Port</label>
              <input
                id="pPort"
                v-model="form.port"
                type="text"
                inputmode="numeric"
                autocomplete="off"
                placeholder="8080"
                @input="onPortInput"
              />
            </div>
          </div>
        </div>

        <div class="preview">
          <span class="preview__label">Address preview</span>
          <code class="preview__value">{{ addressPreview }}</code>
        </div>
      </section>

      <section class="section">
        <header class="section__head">
          <div class="section__icon" aria-hidden="true">
            <i class="ti ti-world" />
          </div>
          <div>
            <h3>Country</h3>
            <p>Choose manually or detect from the proxy host IP</p>
          </div>
        </header>

        <div class="mode-list" role="radiogroup" aria-label="Country source">
          <label class="mode-card" :class="{ active: countryMode === 'manual' }">
            <input v-model="countryMode" type="radio" name="countryMode" value="manual" />
            <span class="mode-card__body">
              <span class="mode-card__title">Manual select</span>
              <span class="mode-card__desc">Pick a country from the list</span>
            </span>
          </label>
          <label class="mode-card" :class="{ active: countryMode === 'ip' }">
            <input v-model="countryMode" type="radio" name="countryMode" value="ip" />
            <span class="mode-card__body">
              <span class="mode-card__title">Get from IP</span>
              <span class="mode-card__desc">Lookup country from proxy host address</span>
            </span>
          </label>
        </div>

        <div v-if="countryMode === 'manual'" class="country-panel">
          <div class="field">
            <label for="pCountry">Country</label>
            <select id="pCountry" v-model="form.country">
              <option value="">Select a country</option>
              <option
                v-if="customCountryOption"
                :value="customCountryOption"
              >
                {{ customCountryOption }} (current)
              </option>
              <option v-for="country in COUNTRIES" :key="country.code" :value="country.code">
                {{ country.name }} ({{ country.code }})
              </option>
            </select>
          </div>
        </div>

        <div v-else class="country-panel">
          <div class="geo-row">
            <div class="geo-hint">
              Uses host from Endpoint
              <strong v-if="form.host.trim()">{{ form.host.trim() }}</strong>
              <span v-else class="muted">Enter a host first</span>
            </div>
            <button
              class="btn btn-icon"
              type="button"
              :disabled="geoLoading || !form.host.trim()"
              @click="detectCountryFromIP"
            >
              <i class="ti" :class="geoLoading ? 'ti-loader-2 spin' : 'ti-map-pin'" aria-hidden="true" />
              {{ geoLoading ? 'Looking up…' : 'Get country' }}
            </button>
          </div>

          <div v-if="geoResult" class="geo-result">
            <div>
              <span>Detected</span>
              <strong>{{ geoResult.country }} ({{ geoResult.countryCode }})</strong>
            </div>
            <div>
              <span>Resolved IP</span>
              <strong>{{ geoResult.ip }}</strong>
            </div>
          </div>
          <p v-else-if="form.country" class="geo-current">
            Current country: <strong>{{ countryDisplay(form.country) }}</strong>
          </p>
          <p v-if="geoError" class="field-error">{{ geoError }}</p>
        </div>
      </section>

      <div class="section-grid">
        <section class="section">
          <header class="section__head">
            <div class="section__icon" aria-hidden="true">
              <i class="ti ti-key" />
            </div>
            <div>
              <h3>Authentication</h3>
              <p>Optional proxy credentials</p>
            </div>
          </header>
          <div class="fields">
            <div class="field">
              <label for="pUsername">Username</label>
              <input
                id="pUsername"
                v-model="form.username"
                type="text"
                placeholder="Optional"
                autocomplete="off"
              />
            </div>
            <div class="field">
              <label for="pPassword">Password</label>
              <input
                id="pPassword"
                v-model="form.password"
                type="password"
                placeholder="Optional"
                autocomplete="new-password"
              />
            </div>
          </div>
        </section>

        <section class="section">
          <header class="section__head">
            <div class="section__icon" aria-hidden="true">
              <i class="ti ti-activity" />
            </div>
            <div>
              <h3>Status</h3>
              <p>Availability for account routing</p>
            </div>
          </header>
          <div class="status-list" role="radiogroup" aria-label="Status">
            <label
              v-for="status in PROXY_STATUSES"
              :key="status.value"
              class="status-card"
              :class="[{ active: form.status === status.value }, 'status-card--' + status.value]"
            >
              <input v-model="form.status" type="radio" name="status" :value="status.value" />
              <span class="status-card__body">
                <span class="status-card__title">{{ status.label }}</span>
                <span class="status-card__desc">{{ status.description }}</span>
              </span>
            </label>
          </div>
        </section>
      </div>

      <section class="section">
        <header class="section__head">
          <div class="section__icon" aria-hidden="true">
            <i class="ti ti-notes" />
          </div>
          <div>
            <h3>Notes</h3>
            <p>Optional context for your team</p>
          </div>
        </header>
        <div class="field">
          <label for="pNotes" class="sr-only">Notes</label>
          <textarea id="pNotes" v-model="form.notes" rows="2" placeholder="Optional notes" />
        </div>
      </section>

      <p v-if="fieldError" class="field-error">{{ fieldError }}</p>
      <p v-if="error" class="field-error">{{ error }}</p>

      <div class="actions">
        <button class="btn" type="button" @click="emit('close')">Cancel</button>
        <button class="btn btn-primary" type="submit" :disabled="saving">
          <i class="ti ti-check" aria-hidden="true" />
          {{ saving ? 'Saving…' : isEdit ? 'Save changes' : 'Add proxy' }}
        </button>
      </div>
    </form>
  </ModalDialog>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import ModalDialog from './ModalDialog.vue'
import { lookupCountryFromHost } from '../api/proxies'
import { COUNTRIES, countryLabel } from '../constants/countries'
import { PROXY_PROTOCOLS, PROXY_STATUSES, proxyAddress } from '../constants/proxies'

const emptyForm = () => ({
  name: '',
  protocol: 'http',
  host: '',
  port: '8080',
  username: '',
  password: '',
  country: '',
  status: 'active',
  notes: '',
})

const props = defineProps({
  open: { type: Boolean, default: false },
  proxy: { type: Object, default: null },
  saving: { type: Boolean, default: false },
  error: { type: String, default: '' },
})

const emit = defineEmits(['close', 'submit'])

const form = reactive(emptyForm())
const isEdit = ref(false)
const fieldError = ref('')
const countryMode = ref('manual')
const geoLoading = ref(false)
const geoError = ref('')
const geoResult = ref(null)

const addressPreview = computed(() =>
  proxyAddress({
    protocol: form.protocol || 'http',
    host: form.host.trim() || 'host',
    port: String(form.port || '').trim() || '0',
  }),
)

const customCountryOption = computed(() => {
  const value = String(form.country || '').trim()
  if (!value) return ''
  const known = COUNTRIES.some((c) => c.code === value)
  return known ? '' : value
})

function countryDisplay(code) {
  return countryLabel(code) || code
}

function onPortInput(event) {
  form.port = String(event.target.value || '').replace(/\D/g, '').slice(0, 5)
}

function resetGeoState() {
  geoLoading.value = false
  geoError.value = ''
  geoResult.value = null
}

async function detectCountryFromIP() {
  const host = form.host.trim()
  if (!host) {
    geoError.value = 'Enter a proxy host first'
    return
  }
  geoLoading.value = true
  geoError.value = ''
  geoResult.value = null
  try {
    const result = await lookupCountryFromHost(host)
    geoResult.value = result
    form.country = result.countryCode
  } catch (err) {
    geoError.value = err.message || 'Could not detect country from IP'
  } finally {
    geoLoading.value = false
  }
}

watch(
  () => [props.open, props.proxy],
  () => {
    if (!props.open) return
    Object.assign(form, emptyForm())
    fieldError.value = ''
    resetGeoState()
    countryMode.value = 'manual'
    isEdit.value = Boolean(props.proxy)
    if (props.proxy) {
      Object.assign(form, {
        name: props.proxy.name || '',
        protocol: props.proxy.protocol || 'http',
        host: props.proxy.host || '',
        port: String(props.proxy.port || 8080),
        username: props.proxy.username || '',
        password: props.proxy.password || '',
        country: props.proxy.country || '',
        status: props.proxy.status || 'active',
        notes: props.proxy.notes || '',
      })
    }
  },
  { immediate: true },
)

watch(countryMode, () => {
  geoError.value = ''
})

function submit() {
  if (!form.name.trim()) {
    fieldError.value = 'Enter a proxy name'
    return
  }
  if (!form.host.trim()) {
    fieldError.value = 'Enter a host'
    return
  }
  const port = Number(String(form.port).trim())
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    fieldError.value = 'Enter a valid port between 1 and 65535'
    return
  }
  if (countryMode.value === 'ip' && !form.country.trim()) {
    fieldError.value = 'Detect a country from IP, or switch to manual select'
    return
  }
  fieldError.value = ''
  emit('submit', {
    name: form.name.trim(),
    protocol: form.protocol,
    host: form.host.trim(),
    port,
    username: form.username.trim(),
    password: form.password,
    country: form.country.trim(),
    status: form.status,
    notes: form.notes.trim(),
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

.endpoint-fields {
  margin-top: 12px;
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

.endpoint-row {
  grid-template-columns: 1fr 140px;
}

.endpoint-row .host {
  min-width: 0;
}

.protocol-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
}

.protocol-chip {
  position: relative;
  display: flex;
  align-items: flex-start;
  gap: 10px;
  margin: 0;
  padding: 12px;
  border: 0.5px solid var(--hairline);
  border-radius: var(--radius);
  background: var(--bg);
  cursor: pointer;
  color: var(--text-dim);
}

.protocol-chip .ti {
  font-size: 18px;
  color: var(--text-faint);
  margin-top: 1px;
}

.protocol-chip.active {
  border-color: var(--viper-500);
  background: var(--viper-dim);
  color: var(--text);
}

.protocol-chip.active .ti {
  color: var(--viper-400);
}

.protocol-chip input {
  position: absolute;
  opacity: 0;
  pointer-events: none;
  width: 1px;
  height: 1px;
}

.protocol-chip:focus-within {
  outline: 2px solid color-mix(in srgb, var(--viper-500) 45%, transparent);
  outline-offset: 1px;
}

.protocol-chip__body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.protocol-chip__title {
  font-size: 13.5px;
  font-weight: 600;
  color: var(--text);
}

.protocol-chip__desc {
  font-size: 12px;
  color: var(--text-faint);
  line-height: 1.35;
}

.mode-list {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.mode-card {
  position: relative;
  display: flex;
  margin: 0;
  padding: 10px 12px;
  border: 0.5px solid var(--hairline);
  border-radius: var(--radius);
  background: var(--bg);
  cursor: pointer;
}

.mode-card.active {
  border-color: var(--viper-500);
  background: var(--viper-dim);
}

.mode-card input {
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

.mode-card:focus-within {
  outline: 2px solid color-mix(in srgb, var(--viper-500) 45%, transparent);
  outline-offset: 1px;
}

.mode-card__body {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.mode-card__title {
  font-size: 13.5px;
  font-weight: 600;
  color: var(--text);
}

.mode-card__desc {
  font-size: 12px;
  color: var(--text-faint);
  line-height: 1.35;
}

.country-panel {
  margin-top: 12px;
}

.geo-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.geo-hint {
  font-size: 12.5px;
  color: var(--text-dim);
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.geo-hint strong {
  font-family: var(--mono);
  font-size: 12.5px;
  color: var(--text);
  font-weight: 500;
}

.geo-hint .muted {
  color: var(--text-faint);
}

.geo-row .btn-icon {
  gap: 6px;
  color: var(--viper-400);
}

.geo-result {
  margin-top: 10px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  padding: 12px;
  border: 0.5px solid var(--hairline);
  border-radius: var(--radius);
  background: var(--bg);
}

.geo-result > div {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.geo-result span {
  font-family: var(--mono);
  font-size: 11px;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--text-faint);
}

.geo-result strong {
  font-family: var(--mono);
  font-size: 12.5px;
  font-weight: 500;
  color: var(--viper-400);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.geo-current {
  margin: 10px 0 0;
  font-size: 12.5px;
  color: var(--text-dim);
}

.geo-current strong {
  color: var(--text);
  font-weight: 500;
}

.spin {
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.preview {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px;
  border: 0.5px solid var(--hairline);
  border-radius: var(--radius);
  background: var(--bg);
}

.preview__label {
  font-family: var(--mono);
  font-size: 11px;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--text-faint);
}

.preview__value {
  font-family: var(--mono);
  font-size: 13px;
  color: var(--viper-400);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.status-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.status-card {
  position: relative;
  display: flex;
  margin: 0;
  padding: 10px 12px;
  border: 0.5px solid var(--hairline);
  border-radius: var(--radius);
  background: var(--bg);
  cursor: pointer;
}

.status-card.active {
  border-color: var(--viper-500);
  background: var(--viper-dim);
}

.status-card--error.active {
  border-color: color-mix(in srgb, var(--danger) 55%, transparent);
  background: var(--bg-danger);
}

.status-card--inactive.active {
  border-color: color-mix(in srgb, var(--text-faint) 45%, transparent);
  background: var(--panel-raised);
}

.status-card input {
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

.status-card:focus-within {
  outline: 2px solid color-mix(in srgb, var(--viper-500) 45%, transparent);
  outline-offset: 1px;
}

.status-card__body {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.status-card__title {
  font-size: 13.5px;
  font-weight: 600;
  color: var(--text);
}

.status-card__desc {
  font-size: 12px;
  color: var(--text-faint);
  line-height: 1.35;
}

textarea {
  resize: vertical;
  min-height: 3.5rem;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

.field-error {
  display: block;
  margin: 0;
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
  .section-grid,
  .protocol-grid,
  .mode-list,
  .geo-result {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 560px) {
  .row,
  .endpoint-row {
    grid-template-columns: 1fr;
  }
}
</style>
