<template>
  <ModalDialog
    :open="open"
    :title="isEdit ? 'Edit Gmail' : 'Add Gmail'"
    wide
    @close="emit('close')"
  >
    <form class="form" @submit.prevent="submit">
      <section class="section">
        <header class="section__head">
          <div class="section__icon" aria-hidden="true">
            <i class="ti ti-mail" />
          </div>
          <div>
            <h3>Mailbox</h3>
            <p>Gmail address and display label</p>
          </div>
        </header>
        <div class="fields">
          <div class="row">
            <div class="field">
              <label for="gEmail">Email</label>
              <input
                id="gEmail"
                v-model="form.email"
                type="email"
                placeholder="name@gmail.com"
                autocomplete="off"
              />
            </div>
            <div class="field">
              <label for="gLabel">Label</label>
              <input id="gLabel" v-model="form.label" type="text" placeholder="Optional display name" />
            </div>
          </div>
        </div>
      </section>

      <div class="section-grid">
        <section class="section">
          <header class="section__head">
            <div class="section__icon" aria-hidden="true">
              <i class="ti ti-lock" />
            </div>
            <div>
              <h3>Credentials</h3>
              <p>Gmail password and app password</p>
            </div>
          </header>
          <div class="fields">
            <div class="field">
              <label for="gPassword">Gmail password</label>
              <input
                id="gPassword"
                v-model="form.password"
                type="password"
                autocomplete="new-password"
                placeholder="Gmail account password"
              />
            </div>
            <div class="field">
              <label for="gAppPassword">App password</label>
              <input
                id="gAppPassword"
                v-model="form.appPassword"
                type="password"
                autocomplete="new-password"
                placeholder="xxxx xxxx xxxx xxxx"
              />
            </div>
          </div>
        </section>

        <section class="section">
          <header class="section__head">
            <div class="section__icon" aria-hidden="true">
              <i class="ti ti-shield-check" />
            </div>
            <div>
              <h3>Recovery</h3>
              <p>Backup contact details</p>
            </div>
          </header>
          <div class="fields">
            <div class="field">
              <label for="gRecoveryEmail">Recovery email</label>
              <input
                id="gRecoveryEmail"
                v-model="form.recoveryEmail"
                type="email"
                placeholder="Optional"
                autocomplete="off"
              />
            </div>
            <div class="field">
              <label for="gRecoveryPhone">Recovery phone</label>
              <input
                id="gRecoveryPhone"
                v-model="form.recoveryPhone"
                type="text"
                placeholder="Optional"
                autocomplete="off"
              />
            </div>
          </div>
        </section>
      </div>

      <div class="section-grid">
        <section class="section">
          <header class="section__head">
            <div class="section__icon" aria-hidden="true">
              <i class="ti ti-auth-2fa" />
            </div>
            <div>
              <h3>Two-factor authentication</h3>
              <p>2FA secret and backup codes</p>
            </div>
          </header>
          <div class="fields">
            <div class="field">
              <label for="gTwofaSecret">2FA secret / method</label>
              <input
                id="gTwofaSecret"
                v-model="form.twofaSecret"
                type="text"
                placeholder="TOTP secret or authenticator info"
                autocomplete="off"
              />
            </div>
            <div class="field">
              <label for="gBackupCodes">Backup codes</label>
              <textarea
                id="gBackupCodes"
                v-model="form.backupCodes"
                rows="3"
                placeholder="One code per line"
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
              <p>Availability for account use</p>
            </div>
          </header>
          <div class="status-list" role="radiogroup" aria-label="Status">
            <label
              v-for="status in GMAIL_STATUSES"
              :key="status.value"
              class="status-card"
              :class="[{ active: form.status === status.value }, 'status-card--' + status.value]"
            >
              <input v-model="form.status" type="radio" name="gmailStatus" :value="status.value" />
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
          <label for="gNotes" class="sr-only">Notes</label>
          <textarea id="gNotes" v-model="form.notes" rows="2" placeholder="Optional notes" />
        </div>
      </section>

      <p v-if="fieldError" class="field-error">{{ fieldError }}</p>
      <p v-if="error" class="field-error">{{ error }}</p>

      <div class="actions">
        <button class="btn" type="button" @click="emit('close')">Cancel</button>
        <button class="btn btn-primary" type="submit" :disabled="saving">
          <i class="ti ti-check" aria-hidden="true" />
          {{ saving ? 'Saving…' : isEdit ? 'Save changes' : 'Add Gmail' }}
        </button>
      </div>
    </form>
  </ModalDialog>
</template>

<script setup>
import { reactive, ref, watch } from 'vue'
import ModalDialog from './ModalDialog.vue'
import { GMAIL_STATUSES } from '../constants/gmails'

const emptyForm = () => ({
  email: '',
  password: '',
  appPassword: '',
  recoveryEmail: '',
  recoveryPhone: '',
  twofaSecret: '',
  backupCodes: '',
  label: '',
  status: 'active',
  notes: '',
})

const props = defineProps({
  open: { type: Boolean, default: false },
  gmail: { type: Object, default: null },
  saving: { type: Boolean, default: false },
  error: { type: String, default: '' },
})

const emit = defineEmits(['close', 'submit'])

const form = reactive(emptyForm())
const isEdit = ref(false)
const fieldError = ref('')

watch(
  () => [props.open, props.gmail],
  () => {
    if (!props.open) return
    Object.assign(form, emptyForm())
    fieldError.value = ''
    isEdit.value = Boolean(props.gmail)
    if (props.gmail) {
      Object.assign(form, {
        email: props.gmail.email || '',
        password: props.gmail.password || '',
        appPassword: props.gmail.appPassword || '',
        recoveryEmail: props.gmail.recoveryEmail || '',
        recoveryPhone: props.gmail.recoveryPhone || '',
        twofaSecret: props.gmail.twofaSecret || '',
        backupCodes: props.gmail.backupCodes || '',
        label: props.gmail.label || '',
        status: props.gmail.status || 'active',
        notes: props.gmail.notes || '',
      })
    }
  },
  { immediate: true },
)

function submit() {
  if (!form.email.trim()) {
    fieldError.value = 'Enter a Gmail address'
    return
  }
  if (!form.appPassword.trim()) {
    fieldError.value = 'Enter an app password'
    return
  }
  fieldError.value = ''
  emit('submit', {
    email: form.email.trim(),
    password: form.password,
    appPassword: form.appPassword,
    recoveryEmail: form.recoveryEmail.trim(),
    recoveryPhone: form.recoveryPhone.trim(),
    twofaSecret: form.twofaSecret.trim(),
    backupCodes: form.backupCodes.trim(),
    label: form.label.trim(),
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
}

.status-card__desc {
  font-size: 12px;
  color: var(--text-faint);
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
  margin: 0;
  font-size: 13px;
  color: var(--danger);
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
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
  .row {
    grid-template-columns: 1fr;
  }
}
</style>
