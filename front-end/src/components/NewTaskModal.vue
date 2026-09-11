<template>
  <ModalDialog
    :open="open"
    :title="dialogTitle"
    title-id="new-task-title"
    wide
    @close="emit('close')"
  >
    <form class="wizard" @submit.prevent="onPrimary">
      <div class="progress">
        <div class="progress__track" aria-hidden="true">
          <span class="progress__fill" :style="{ width: progressPct + '%' }" />
        </div>
        <ol class="steps" :style="{ gridTemplateColumns: `repeat(${visibleSteps.length}, minmax(0, 1fr))` }">
          <li v-for="(item, index) in visibleSteps" :key="item.key">
            <button
              class="steps__item"
              type="button"
              :class="{
                'is-active': stepIndex === index,
                'is-done': index < stepIndex,
              }"
              :disabled="index > stepIndex"
              :aria-current="stepIndex === index ? 'step' : undefined"
              @click="jumpTo(index)"
            >
              <span class="steps__num">
                <i v-if="index < stepIndex" class="ti ti-check" aria-hidden="true" />
                <template v-else>{{ index + 1 }}</template>
              </span>
              <span class="steps__copy">
                <strong>{{ item.label }}</strong>
                <small>{{ item.hint }}</small>
              </span>
            </button>
          </li>
        </ol>
      </div>

      <div class="coach" :class="'coach--' + currentStep">
        <i class="ti" :class="coach.icon" aria-hidden="true" />
        <div>
          <strong>{{ coach.title }}</strong>
          <p>{{ coach.body }}</p>
        </div>
      </div>

      <!-- Setup -->
      <section v-if="currentStep === 'setup'" class="panel">
        <div class="field">
          <div class="field__row">
            <span>1. Choose platform</span>
            <span class="muted">{{ platformsWithAccounts.length }} available</span>
          </div>
          <div class="platform-grid">
            <button
              v-for="platform in platformOptions"
              :key="platform.value"
              class="platform-card"
              type="button"
              :class="{
                active: form.platform === platform.value,
                'is-empty': platform.count === 0,
              }"
              :disabled="platform.count === 0"
              @click="pickPlatform(platform.value)"
            >
              <i class="ti" :class="platform.icon" aria-hidden="true" />
              <div class="platform-card__body">
                <strong>{{ platform.label }}</strong>
                <span>
                  {{ platform.count }}
                  account{{ platform.count === 1 ? '' : 's' }}
                  <template v-if="platform.active"> · {{ platform.active }} active</template>
                </span>
              </div>
              <i
                v-if="form.platform === platform.value"
                class="ti ti-circle-check platform-card__check"
                aria-hidden="true"
              />
            </button>
          </div>
          <p v-if="!platformsWithAccounts.length" class="hint-banner">
            No social accounts yet.
            <RouterLink to="/accounts" @click="emit('close')">Connect an account</RouterLink>
            first, then come back here.
          </p>
        </div>

        <div class="field">
          <div class="field__row">
            <span>2. Choose action</span>
            <span class="muted">{{ form.platform ? platformMeta(form.platform).label : 'Pick a platform first' }}</span>
          </div>
          <div class="type-grid">
            <button
              v-for="type in TASK_TYPES"
              :key="type.value"
              class="type-card"
              type="button"
              :class="{ active: form.type === type.value }"
              :disabled="!form.platform"
              @click="pickType(type.value)"
            >
              <div class="type-card__top">
                <i class="ti" :class="type.icon" aria-hidden="true" />
                <span class="badge" :class="actionBadgeClass(type)">
                  {{ actionBadgeLabel(type) }}
                </span>
              </div>
              <strong>{{ type.label }}</strong>
              <span>{{ type.description }}</span>
            </button>
          </div>
        </div>

        <div v-if="form.platform && form.type" class="path-preview">
          <span>Your path</span>
          <div class="path-preview__flow">
            <em>{{ platformMeta(form.platform).label }}</em>
            <i class="ti ti-chevron-right" aria-hidden="true" />
            <em>{{ taskTypeMeta(form.type).label }}</em>
            <i class="ti ti-chevron-right" aria-hidden="true" />
            <em>{{ nextStepHint }}</em>
            <i class="ti ti-chevron-right" aria-hidden="true" />
            <em>Launch</em>
          </div>
        </div>

        <p v-if="stepError" class="error">{{ stepError }}</p>
      </section>

      <!-- Target -->
      <section v-else-if="currentStep === 'target'" class="panel">
        <label class="field">
          <span>Post / target URL <em class="req">required</em></span>
          <div class="url-field">
            <i class="ti ti-link" aria-hidden="true" />
            <input
              ref="urlInput"
              v-model.trim="form.targetUrl"
              class="url-field__input"
              type="url"
              inputmode="url"
              autocomplete="url"
              spellcheck="false"
              :placeholder="urlPlaceholder"
            />
          </div>
          <small class="help">
            Open the post on {{ platformMeta(form.platform).label }}, copy the link from the address bar, and paste it here.
          </small>
        </label>

        <label class="field">
          <span>Task name <em class="opt">optional</em></span>
          <input v-model.trim="form.title" type="text" :placeholder="defaultTitle" />
          <small class="help">Shown in Active jobs and History so you can find this run later.</small>
        </label>

        <div class="example">
          <strong>Example</strong>
          <code>{{ urlPlaceholder.replace('…', 'permalink/123') }}</code>
        </div>

        <p v-if="stepError" class="error">{{ stepError }}</p>
      </section>

      <!-- Reply: URL + comment -->
      <section v-else-if="currentStep === 'reply'" class="panel">
        <div class="compose-tip">
          <i class="ti ti-message" aria-hidden="true" />
          <p>
            Paste the post link, then write the comment each selected account will leave.
          </p>
        </div>

        <label class="field">
          <span>Post URL <em class="req">required</em></span>
          <div class="url-field">
            <i class="ti ti-link" aria-hidden="true" />
            <input
              ref="urlInput"
              v-model.trim="form.targetUrl"
              class="url-field__input"
              type="url"
              inputmode="url"
              autocomplete="url"
              spellcheck="false"
              :placeholder="urlPlaceholder"
            />
          </div>
          <small class="help">
            Open the post on {{ platformMeta(form.platform).label }}, copy the link, and paste it here.
          </small>
        </label>

        <label class="field">
          <span>Comment text <em class="req">required</em></span>
          <textarea
            ref="composeTextInput"
            v-model="form.text"
            class="compose-text"
            rows="5"
            maxlength="8000"
            placeholder="Write the comment to post…"
          />
          <div class="field__row">
            <small class="help">Same comment is posted from every selected account.</small>
            <span class="muted">{{ form.text.length }}/8000</span>
          </div>
        </label>

        <label class="field">
          <span>Task name <em class="opt">optional</em></span>
          <input v-model.trim="form.title" type="text" :placeholder="defaultTitle" />
        </label>

        <p v-if="stepError" class="error">{{ stepError }}</p>
      </section>

      <!-- Compose new post -->
      <section v-else-if="currentStep === 'compose'" class="panel">
        <div class="compose-tip">
          <i class="ti" :class="platformMeta(form.platform).icon" aria-hidden="true" />
          <p>{{ composeSpec.tip }}</p>
        </div>

        <label v-if="composeSpec.needsHeadline" class="field">
          <span>{{ composeSpec.headlineLabel }} <em class="req">required</em></span>
          <input
            ref="composeHeadlineInput"
            v-model.trim="form.headline"
            type="text"
            maxlength="120"
            :placeholder="composeSpec.headlineLabel"
          />
        </label>

        <label class="field">
          <span>
            {{ composeSpec.textLabel }}
            <em class="req">required</em>
          </span>
          <textarea
            ref="composeTextInput"
            v-model="form.text"
            class="compose-text"
            rows="6"
            :maxlength="composeSpec.maxText"
            :placeholder="composeSpec.textPlaceholder"
          />
          <div class="field__row">
            <small class="help">This is what each selected account will publish.</small>
            <span class="muted" :class="{ 'is-warn': textOverLimit }">
              {{ form.text.length }}/{{ composeSpec.maxText }}
            </span>
          </div>
        </label>

        <label class="field">
          <span>
            {{ composeSpec.needsMedia ? composeSpec.mediaLabel : 'Media' }}
            <em v-if="composeSpec.needsMedia" class="req">required</em>
            <em v-else class="opt">optional</em>
          </span>
          <div class="media-upload">
            <input
              ref="mediaFileInput"
              class="media-upload__file"
              type="file"
              accept="image/*,video/*"
              @change="onMediaFile"
            />
            <button class="btn" type="button" :disabled="uploadingMedia" @click="mediaFileInput?.click()">
              <i class="ti" :class="uploadingMedia ? 'ti-loader-2 spin' : 'ti-upload'" aria-hidden="true" />
              {{ uploadingMedia ? 'Uploading…' : 'Upload media' }}
            </button>
            <span class="muted">or paste a URL</span>
          </div>
          <div class="url-field">
            <i class="ti ti-photo" aria-hidden="true" />
            <input
              v-model.trim="form.mediaUrl"
              class="url-field__input"
              type="url"
              inputmode="url"
              spellcheck="false"
              :placeholder="composeSpec.mediaPlaceholder || 'https://…'"
            />
          </div>
          <small class="help">
            Upload an image/video to this server, or paste a public media URL.
            <template v-if="mediaFileName"> Selected: {{ mediaFileName }}</template>
          </small>
        </label>

        <label v-if="composeSpec.allowsLink" class="field">
          <span>Attach link <em class="opt">optional</em></span>
          <div class="url-field">
            <i class="ti ti-link" aria-hidden="true" />
            <input
              v-model.trim="form.linkUrl"
              class="url-field__input"
              type="url"
              inputmode="url"
              spellcheck="false"
              placeholder="https://example.com/article"
            />
          </div>
        </label>

        <label class="field">
          <span>Task name <em class="opt">optional</em></span>
          <input v-model.trim="form.title" type="text" :placeholder="defaultTitle" />
          <small class="help">Shown in Active jobs and History so you can find this run later.</small>
        </label>

        <p v-if="stepError" class="error">{{ stepError }}</p>
      </section>

      <!-- Accounts -->
      <section v-else-if="currentStep === 'accounts'" class="panel">
        <label v-if="!needsUrl && !needsCompose && !needsReply" class="field">
          <span>Task name <em class="opt">optional</em></span>
          <input v-model.trim="form.title" type="text" :placeholder="defaultTitle" />
        </label>

        <div v-if="platformAccounts.length === 0" class="empty-state">
          <i class="ti ti-share-off" aria-hidden="true" />
          <strong>No {{ platformMeta(form.platform).label }} accounts</strong>
          <p>Connect at least one account for this platform, then return to launch the task.</p>
          <RouterLink class="btn btn-primary" to="/accounts" @click="emit('close')">
            Go to Social accounts
          </RouterLink>
        </div>

        <template v-else>
          <div class="quick-picks">
            <button class="quick-pick" type="button" :disabled="!activeAccountIds.length" @click="selectActive">
              <i class="ti ti-circle-check" aria-hidden="true" />
              All active ({{ activeAccountIds.length }})
            </button>
            <button class="quick-pick" type="button" :disabled="!platformAccounts.length" @click="selectAll">
              <i class="ti ti-select-all" aria-hidden="true" />
              Everyone ({{ platformAccounts.length }})
            </button>
            <button class="quick-pick" type="button" :disabled="!selectedIds.length" @click="clearSelection">
              <i class="ti ti-x" aria-hidden="true" />
              Clear
            </button>
          </div>

          <div class="account-tools">
            <input v-model="accountQuery" type="search" placeholder="Search by name or email" />
            <select v-model="statusFilter">
              <option value="">All statuses</option>
              <option v-for="status in STATUSES" :key="status.value" :value="status.value">
                {{ status.label }}
              </option>
            </select>
          </div>

          <div class="bulk-bar">
            <label class="bulk-bar__check">
              <input
                type="checkbox"
                :checked="allFilteredSelected"
                :disabled="filteredAccounts.length === 0"
                @change="toggleSelectFiltered"
              />
              <span>
                {{ allFilteredSelected ? 'Deselect visible' : 'Select visible' }}
                ({{ filteredAccounts.length }})
              </span>
            </label>
            <span class="muted">{{ selectedIds.length }} selected</span>
          </div>

          <div class="account-list">
            <label v-for="account in pageItems" :key="account.id" class="account-row">
              <input v-model="selectedIds" type="checkbox" :value="account.id" />
              <i class="ti" :class="platformMeta(account.platform).icon" aria-hidden="true" />
              <div class="account-row__body">
                <strong>{{ accountDisplayName(account) }}</strong>
                <span>{{ account.email }}</span>
              </div>
              <span class="status" :class="'status--' + account.status">{{ statusLabel(account.status) }}</span>
            </label>
            <p v-if="filteredAccounts.length === 0" class="empty">
              No accounts match your search. Try clearing filters.
            </p>
          </div>
          <PaginationBar
            v-model:page="page"
            v-model:page-size="pageSize"
            :total="total"
            :total-pages="totalPages"
            :from="from"
            :to="to"
            :page-size-options="pageSizeOptions"
          />
        </template>

        <p v-if="stepError" class="error">{{ stepError }}</p>
      </section>

      <!-- Review -->
      <section v-else class="panel">
        <div class="summary-card">
          <div class="summary-card__hero">
            <i class="ti" :class="taskTypeMeta(form.type).icon" aria-hidden="true" />
            <div>
              <strong>{{ form.title || defaultTitle }}</strong>
              <p>
                {{ taskTypeMeta(form.type).label }} on {{ platformMeta(form.platform).label }}
                with {{ selectedIds.length }} account{{ selectedIds.length === 1 ? '' : 's' }}
              </p>
            </div>
          </div>

          <dl class="summary-list">
            <div>
              <dt>Platform</dt>
              <dd>
                <i class="ti" :class="platformMeta(form.platform).icon" aria-hidden="true" />
                {{ platformMeta(form.platform).label }}
              </dd>
            </div>
            <div>
              <dt>Action</dt>
              <dd>{{ taskTypeMeta(form.type).label }}</dd>
            </div>
            <div>
              <dt>{{ needsCompose || needsReply ? 'Content' : 'Target' }}</dt>
              <dd :class="{ mono: needsUrl || needsReply }">
                <template v-if="needsUrl">{{ form.targetUrl }}</template>
                <template v-else-if="needsReply">
                  <span class="mono">{{ form.targetUrl }}</span>
                  <span class="compose-preview">{{ composePreview }}</span>
                </template>
                <template v-else-if="needsCompose">
                  <span class="compose-preview">{{ composePreview }}</span>
                </template>
                <template v-else>Not required for this action</template>
              </dd>
            </div>
            <div>
              <dt>Accounts</dt>
              <dd>{{ selectedIds.length }} ready to run</dd>
            </div>
          </dl>

          <div v-if="selectedAccounts.length" class="selected-preview">
            <div v-for="account in selectedAccounts.slice(0, 8)" :key="account.id" class="chip">
              {{ accountDisplayName(account) }}
            </div>
            <div v-if="selectedAccounts.length > 8" class="chip chip--more">
              +{{ selectedAccounts.length - 8 }} more
            </div>
          </div>
        </div>

        <label class="show-browser">
          <input v-model="form.showBrowser" type="checkbox" />
          <div>
            <strong>Show browser window</strong>
            <span>
              Opens a live view dialog as soon as the job starts (same remote window as Social accounts → Open).
              Leave off for faster headless runs.
            </span>
          </div>
        </label>

        <details class="advanced">
          <summary>
            <span>Advanced run settings</span>
            <small>
              Delay {{ form.delayMinSec }}–{{ form.delayMaxSec }}s · Proxy {{ form.useAccountProxy ? 'on' : 'off' }}
              · Window {{ form.showBrowser ? 'shown' : 'hidden' }}
            </small>
          </summary>
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
            <small class="help">
              A short random delay between accounts looks more natural and reduces rate limits.
            </small>
          </div>
        </details>

        <p v-if="error || stepError" class="error">{{ error || stepError }}</p>
      </section>

      <div class="actions">
        <button class="btn" type="button" :disabled="saving" @click="emit('close')">Cancel</button>
        <div class="actions__right">
          <button v-if="stepIndex > 0" class="btn" type="button" :disabled="saving" @click="goBack">
            Back
          </button>
          <button class="btn btn-primary" type="submit" :disabled="primaryDisabled">
            <template v-if="stepIndex < visibleSteps.length - 1">
              {{ primaryLabel }}
              <i class="ti ti-arrow-right" aria-hidden="true" />
            </template>
            <template v-else>
              <i class="ti" :class="saving ? 'ti-loader-2 spin' : 'ti-rocket'" aria-hidden="true" />
              {{ saving ? 'Launching…' : `Launch with ${selectedIds.length || 0} account${selectedIds.length === 1 ? '' : 's'}` }}
            </template>
          </button>
        </div>
      </div>
    </form>
  </ModalDialog>
</template>

<script setup>
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import ModalDialog from './ModalDialog.vue'
import PaginationBar from './PaginationBar.vue'
import {
  PLATFORMS,
  STATUSES,
  accountDisplayName,
  platformMeta,
  statusLabel,
} from '../constants/accounts'
import { TASK_TYPES, postComposeSpec, taskTypeMeta } from '../constants/tasks'
import { uploadMedia } from '../api/users'
import { usePagination } from '../composables/usePagination'

const props = defineProps({
  open: { type: Boolean, default: false },
  accounts: { type: Array, default: () => [] },
  initialType: { type: String, default: 'report' },
  initialPlatform: { type: String, default: '' },
  initialAccountIds: { type: Array, default: () => [] },
  saving: { type: Boolean, default: false },
  error: { type: String, default: '' },
})

const emit = defineEmits(['close', 'submit'])

const stepIndex = ref(0)
const stepError = ref('')
const urlInput = ref(null)
const composeTextInput = ref(null)
const composeHeadlineInput = ref(null)
const mediaFileInput = ref(null)
const uploadingMedia = ref(false)
const mediaFileName = ref('')
const form = reactive({
  platform: '',
  type: 'report',
  title: '',
  targetUrl: '',
  headline: '',
  text: '',
  linkUrl: '',
  mediaUrl: '',
  delayMinSec: 5,
  delayMaxSec: 15,
  useAccountProxy: true,
  showBrowser: false,
})
const selectedIds = ref([])
const accountQuery = ref('')
const statusFilter = ref('')

const needsUrl = computed(() => taskTypeMeta(form.type).needsUrl)
const needsCompose = computed(() => taskTypeMeta(form.type).needsCompose)
const needsReply = computed(() => taskTypeMeta(form.type).needsReply)
const composeSpec = computed(() => postComposeSpec(form.platform))
const textOverLimit = computed(() => form.text.length > composeSpec.value.maxText)
const composePreview = computed(() => {
  const headline = form.headline.trim()
  const text = form.text.trim().replace(/\s+/g, ' ')
  if (headline && text) return `${headline} — ${text.slice(0, 120)}${text.length > 120 ? '…' : ''}`
  return text.slice(0, 160) + (text.length > 160 ? '…' : '')
})
const nextStepHint = computed(() => {
  if (needsUrl.value) return 'Paste URL'
  if (needsReply.value) return 'Write reply'
  if (needsCompose.value) return 'Write post'
  return 'Pick accounts'
})

const visibleSteps = computed(() => {
  const steps = [{ key: 'setup', label: 'Setup', hint: 'Platform & action' }]
  if (needsUrl.value) steps.push({ key: 'target', label: 'Target', hint: 'Paste the URL' })
  if (needsReply.value) steps.push({ key: 'reply', label: 'Reply', hint: 'URL & comment' })
  if (needsCompose.value) steps.push({ key: 'compose', label: 'Content', hint: 'Write the post' })
  steps.push(
    { key: 'accounts', label: 'Accounts', hint: 'Who will run it' },
    { key: 'review', label: 'Launch', hint: 'Confirm & start' },
  )
  return steps
})

const currentStep = computed(() => visibleSteps.value[stepIndex.value]?.key || 'setup')
const progressPct = computed(() =>
  Math.round(((stepIndex.value + 1) / Math.max(visibleSteps.value.length, 1)) * 100),
)

const defaultTitle = computed(() => {
  const action = taskTypeMeta(form.type).label
  const platform = form.platform ? platformMeta(form.platform).label : ''
  return platform ? `${action} · ${platform}` : action
})

const urlPlaceholder = computed(() => {
  const url = platformMeta(form.platform).url
  return url ? `${url}/…` : 'https://…'
})

const dialogTitle = computed(() => {
  if (currentStep.value === 'setup') return 'Create a new task'
  return `New task · step ${stepIndex.value + 1} of ${visibleSteps.value.length}`
})

const platformOptions = computed(() =>
  PLATFORMS.map((platform) => {
    const rows = props.accounts.filter((account) => account.platform === platform.value)
    return {
      ...platform,
      count: rows.length,
      active: rows.filter((account) => account.status === 'active').length,
    }
  }),
)

const platformsWithAccounts = computed(() => platformOptions.value.filter((platform) => platform.count > 0))

const platformAccounts = computed(() =>
  props.accounts.filter((account) => !form.platform || account.platform === form.platform),
)

const filteredAccounts = computed(() => {
  const q = accountQuery.value.trim().toLowerCase()
  return platformAccounts.value.filter((account) => {
    if (statusFilter.value && account.status !== statusFilter.value) return false
    if (!q) return true
    const hay = `${accountDisplayName(account)} ${account.email}`.toLowerCase()
    return hay.includes(q)
  })
})

const {
  page,
  pageSize,
  pageItems,
  total,
  totalPages,
  from,
  to,
  pageSizeOptions,
  reset: resetPage,
} = usePagination(filteredAccounts, { pageSize: 10 })

watch([accountQuery, statusFilter, () => form.platform], resetPage)

const filteredIds = computed(() => filteredAccounts.value.map((account) => account.id))

const activeAccountIds = computed(() =>
  platformAccounts.value.filter((account) => account.status === 'active').map((account) => account.id),
)

const allFilteredSelected = computed(() => {
  const ids = filteredIds.value
  return ids.length > 0 && ids.every((id) => selectedIds.value.includes(id))
})

const selectedAccounts = computed(() =>
  platformAccounts.value.filter((account) => selectedIds.value.includes(account.id)),
)

const coach = computed(() => {
  switch (currentStep.value) {
    case 'setup':
      return {
        icon: 'ti-map-pin',
        title: 'Start with where and what',
        body: 'Choose the social network, then the action. We’ll ask for a URL or post content only when needed.',
      }
    case 'target':
      return {
        icon: 'ti-link',
        title: 'Paste the post link',
        body: `This ${taskTypeMeta(form.type).label.toLowerCase()} needs a public ${platformMeta(form.platform).label} URL.`,
      }
    case 'reply':
      return {
        icon: 'ti-message',
        title: 'Write the comment',
        body: `Each selected ${platformMeta(form.platform).label} account will leave this comment on the post.`,
      }
    case 'compose':
      return {
        icon: 'ti-pencil-plus',
        title: `Write your ${platformMeta(form.platform).label} post`,
        body: composeSpec.value.tip,
      }
    case 'accounts':
      return {
        icon: 'ti-users',
        title: 'Choose who runs it',
        body: `Only ${platformMeta(form.platform).label} accounts are listed. Use All active for the safest first run.`,
      }
    default:
      return {
        icon: 'ti-rocket',
        title: 'Ready to launch',
        body: 'Review the plan. Open Advanced only if you want to change delays or proxy behavior.',
      }
  }
})

const primaryLabel = computed(() => {
  const next = visibleSteps.value[stepIndex.value + 1]
  if (!next) return 'Launch task'
  if (next.key === 'target') return 'Continue to URL'
  if (next.key === 'reply') return 'Write reply'
  if (next.key === 'compose') return 'Write post'
  if (next.key === 'accounts') return 'Choose accounts'
  if (next.key === 'review') return 'Review & launch'
  return 'Next'
})

const primaryDisabled = computed(() => {
  if (props.saving) return true
  if (currentStep.value === 'setup') return !form.platform || !form.type
  if (currentStep.value === 'accounts') return platformAccounts.value.length === 0
  return false
})

function actionBadgeLabel(type) {
  if (type.needsUrl) return 'Needs URL'
  if (type.needsReply) return 'URL + comment'
  if (type.needsCompose) return 'Write post'
  return 'Ready'
}

function actionBadgeClass(type) {
  if (type.needsUrl) return 'badge--url'
  if (type.needsReply) return 'badge--compose'
  if (type.needsCompose) return 'badge--compose'
  return 'badge--ready'
}

function isHttpUrl(value) {
  if (String(value || '').startsWith('/api/uploads/')) return true
  try {
    const parsed = new URL(value)
    return ['http:', 'https:'].includes(parsed.protocol)
  } catch (_) {
    return false
  }
}

async function onMediaFile(event) {
  const file = event.target.files?.[0]
  if (!file) return
  uploadingMedia.value = true
  stepError.value = ''
  try {
    const data = await uploadMedia(file)
    form.mediaUrl = data.url || ''
    mediaFileName.value = data.filename || file.name
  } catch (err) {
    stepError.value = err.message || 'Upload failed'
  } finally {
    uploadingMedia.value = false
    if (mediaFileInput.value) mediaFileInput.value.value = ''
  }
}

function pickPlatform(value) {
  if (form.platform === value) return
  form.platform = value
  stepError.value = ''
  const allowed = new Set(
    props.accounts.filter((account) => account.platform === value).map((account) => account.id),
  )
  selectedIds.value = selectedIds.value.filter((id) => allowed.has(id))
}

function pickType(value) {
  form.type = value
  stepError.value = ''
  if (stepIndex.value >= visibleSteps.value.length) {
    stepIndex.value = visibleSteps.value.length - 1
  }
}

function mergeSelection(ids) {
  selectedIds.value = Array.from(new Set([...selectedIds.value, ...ids]))
}

function selectAll() {
  selectedIds.value = platformAccounts.value.map((account) => account.id)
}

function selectActive() {
  selectedIds.value = [...activeAccountIds.value]
}

function clearSelection() {
  selectedIds.value = []
}

function toggleSelectFiltered(event) {
  const ids = filteredIds.value
  if (event.target.checked) mergeSelection(ids)
  else selectedIds.value = selectedIds.value.filter((id) => !ids.includes(id))
}

function inferPlatformFromAccounts(ids) {
  if (!ids?.length) return ''
  const match = props.accounts.find((account) => account.id === ids[0])
  return match?.platform || ''
}

function bestDefaultPlatform() {
  if (props.initialPlatform) return props.initialPlatform
  const fromAccounts = inferPlatformFromAccounts(props.initialAccountIds)
  if (fromAccounts) return fromAccounts
  if (platformsWithAccounts.value.length === 1) return platformsWithAccounts.value[0].value
  const facebook = platformsWithAccounts.value.find((platform) => platform.value === 'facebook')
  return facebook?.value || platformsWithAccounts.value[0]?.value || ''
}

function resetWizard() {
  stepIndex.value = 0
  stepError.value = ''
  form.type = props.initialType || 'report'
  form.platform = bestDefaultPlatform()
  form.title = ''
  form.targetUrl = ''
  form.headline = ''
  form.text = ''
  form.linkUrl = ''
  form.mediaUrl = ''
  form.delayMinSec = 5
  form.delayMaxSec = 15
  form.useAccountProxy = true
  form.showBrowser = false
  selectedIds.value = [...(props.initialAccountIds || [])]
  accountQuery.value = ''
  statusFilter.value = ''
  mediaFileName.value = ''
}

function jumpTo(index) {
  if (index < stepIndex.value) {
    stepError.value = ''
    stepIndex.value = index
  }
}

function goBack() {
  stepError.value = ''
  if (stepIndex.value > 0) stepIndex.value -= 1
}

function validateStep() {
  if (currentStep.value === 'setup') {
    if (!form.platform) {
      stepError.value = 'Choose a platform first'
      return false
    }
    if (!form.type) {
      stepError.value = 'Choose an action to continue'
      return false
    }
  }

  if (currentStep.value === 'target') {
    const url = form.targetUrl.trim()
    if (!url) {
      stepError.value = 'Paste a post URL to continue'
      return false
    }
    if (!isHttpUrl(url)) {
      stepError.value = 'URL must start with http:// or https://'
      return false
    }
  }

  if (currentStep.value === 'reply') {
    const url = form.targetUrl.trim()
    if (!url) {
      stepError.value = 'Paste a post URL to continue'
      return false
    }
    if (!isHttpUrl(url)) {
      stepError.value = 'URL must start with http:// or https://'
      return false
    }
    if (!form.text.trim()) {
      stepError.value = 'Write a comment to continue'
      return false
    }
  }

  if (currentStep.value === 'compose') {
    const spec = composeSpec.value
    if (spec.needsHeadline && !form.headline.trim()) {
      stepError.value = `${spec.headlineLabel} is required`
      return false
    }
    const text = form.text.trim()
    if (!text) {
      stepError.value = `${spec.textLabel} is required`
      return false
    }
    if (text.length > spec.maxText) {
      stepError.value = `${spec.textLabel} must be ${spec.maxText} characters or fewer`
      return false
    }
    if (spec.needsMedia) {
      const media = form.mediaUrl.trim()
      if (!media) {
        stepError.value = `${spec.mediaLabel} is required`
        return false
      }
      if (!isHttpUrl(media)) {
        stepError.value = 'Media URL must start with http:// or https://'
        return false
      }
    }
    if (form.linkUrl.trim() && !isHttpUrl(form.linkUrl.trim())) {
      stepError.value = 'Attach link must start with http:// or https://'
      return false
    }
  }

  if (currentStep.value === 'accounts') {
    if (!selectedIds.value.length) {
      stepError.value = 'Select at least one account, or tap All active'
      return false
    }
  }

  if (currentStep.value === 'review') {
    const min = Number(form.delayMinSec) || 0
    const max = Number(form.delayMaxSec) || 0
    if (min < 1 || max < min) {
      stepError.value = 'Delay max must be greater than or equal to delay min'
      return false
    }
  }

  stepError.value = ''
  return true
}

function onPrimary() {
  if (!validateStep()) return
  if (stepIndex.value < visibleSteps.value.length - 1) {
    stepIndex.value += 1
    return
  }
  const payload = {
    platform: form.platform,
    type: form.type,
    title: form.title || defaultTitle.value,
    targetUrl: needsUrl.value || needsReply.value ? form.targetUrl.trim() : '',
    accountIds: selectedIds.value.map(Number),
    delayMinSec: Number(form.delayMinSec) || 5,
    delayMaxSec: Number(form.delayMaxSec) || 15,
    useAccountProxy: Boolean(form.useAccountProxy),
    showBrowser: Boolean(form.showBrowser),
  }
  if (needsCompose.value) {
    payload.content = {
      text: form.text.trim(),
      headline: form.headline.trim(),
      linkUrl: form.linkUrl.trim(),
      mediaUrl: form.mediaUrl.trim(),
    }
  }
  if (needsReply.value) {
    payload.content = {
      text: form.text.trim(),
    }
  }
  emit('submit', payload)
}

watch(
  () => props.open,
  (open) => {
    if (open) resetWizard()
  },
)

watch([needsUrl, needsCompose, needsReply], () => {
  if (stepIndex.value > visibleSteps.value.length - 1) {
    stepIndex.value = visibleSteps.value.length - 1
  }
})

watch(currentStep, async (key) => {
  if (key === 'target' || key === 'reply') {
    await nextTick()
    urlInput.value?.focus()
  }
  if (key === 'compose') {
    await nextTick()
    if (composeSpec.value.needsHeadline) composeHeadlineInput.value?.focus()
    else composeTextInput.value?.focus()
  }
})
</script>

<style scoped>
.wizard {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 0;
  flex: 1;
}

.progress {
  flex-shrink: 0;
}

.progress__track {
  height: 3px;
  border-radius: 999px;
  background: var(--panel-raised);
  overflow: hidden;
  margin-bottom: 8px;
}

.progress__fill {
  display: block;
  height: 100%;
  background: linear-gradient(90deg, var(--viper-700), var(--viper-400));
  transition: width 0.2s ease;
}

.steps {
  list-style: none;
  display: grid;
  gap: 8px;
  margin: 0;
  padding: 0;
}

.steps__item {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border: 0.5px solid var(--hairline);
  border-radius: 10px;
  background: var(--bg);
  color: var(--text-faint);
  text-align: left;
}

.steps__item:disabled {
  cursor: default;
  opacity: 0.7;
}

.steps__item.is-active {
  border-color: color-mix(in srgb, var(--viper-500) 50%, transparent);
  background: var(--viper-dim);
  color: var(--viper-400);
}

.steps__item.is-done {
  color: var(--text-dim);
  cursor: pointer;
}

.steps__item.is-done:hover {
  border-color: var(--hairline-strong);
  color: var(--text);
}

.steps__num {
  width: 22px;
  height: 22px;
  display: grid;
  place-content: center;
  border-radius: 999px;
  border: 0.5px solid currentColor;
  font-family: var(--mono);
  font-size: 11px;
  flex-shrink: 0;
}

.steps__copy {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}

.steps__copy strong {
  font-size: 12px;
  font-weight: 600;
}

.steps__copy small {
  font-size: 11px;
  color: inherit;
  opacity: 0.8;
}

.coach {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  padding: 10px 12px;
  border: 0.5px solid color-mix(in srgb, var(--viper-500) 28%, transparent);
  border-radius: 12px;
  background: color-mix(in srgb, var(--viper-dim) 70%, var(--panel));
  flex-shrink: 0;
}

.coach .ti {
  font-size: 18px;
  color: var(--viper-400);
  margin-top: 1px;
}

.coach strong {
  display: block;
  font-size: 13px;
  color: var(--text);
}

.coach p {
  margin: 3px 0 0;
  font-size: 12.5px;
  color: var(--text-dim);
  line-height: 1.4;
}

.panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 0;
  flex: 1;
  overflow: auto;
  padding-right: 2px;
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
  align-items: center;
}

.req,
.opt {
  font-style: normal;
  font-weight: 500;
  margin-left: 6px;
  font-size: 11px;
}

.req { color: var(--viper-400); }
.opt { color: var(--text-faint); }

.help {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-faint);
  line-height: 1.4;
}

.platform-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.platform-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px;
  border: 0.5px solid var(--hairline);
  border-radius: 10px;
  background: var(--bg);
  color: var(--text-dim);
  text-align: left;
}

.platform-card .ti:first-child {
  font-size: 20px;
}

.platform-card__body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}

.platform-card strong {
  font-size: 13px;
  color: var(--text);
}

.platform-card span {
  font-size: 11.5px;
  color: var(--text-faint);
}

.platform-card.active {
  border-color: color-mix(in srgb, var(--viper-500) 55%, transparent);
  background: var(--viper-dim);
}

.platform-card.active,
.platform-card.active strong,
.platform-card.active .ti {
  color: var(--viper-400);
}

.platform-card.is-empty,
.platform-card:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.platform-card__check {
  font-size: 16px;
}

.type-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.type-card {
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: flex-start;
  text-align: left;
  padding: 10px 12px;
  border: 0.5px solid var(--hairline);
  border-radius: 10px;
  background: var(--bg);
  color: var(--text-dim);
}

.type-card:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.type-card__top {
  width: 100%;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
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

.badge {
  font-family: var(--mono);
  font-size: 10px;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  padding: 3px 7px;
  border-radius: 999px;
}

.badge--url {
  background: var(--gold-dim);
  color: var(--gold-500);
}

.badge--compose {
  background: color-mix(in srgb, var(--viper-dim) 80%, var(--panel));
  color: var(--viper-400);
}

.badge--ready {
  background: var(--viper-dim);
  color: var(--viper-400);
}

.compose-tip {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  padding: 10px 12px;
  border: 0.5px solid var(--hairline);
  border-radius: 10px;
  background: var(--bg);
}

.compose-tip .ti {
  font-size: 18px;
  color: var(--viper-400);
  margin-top: 1px;
}

.compose-tip p {
  margin: 0;
  font-size: 12.5px;
  color: var(--text-dim);
  line-height: 1.4;
  font-weight: 500;
}

.media-upload {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  margin-bottom: 6px;
}

.media-upload__file {
  position: absolute;
  width: 1px;
  height: 1px;
  opacity: 0;
  pointer-events: none;
}

.compose-text {
  resize: vertical;
  min-height: 88px;
  max-height: 180px;
  line-height: 1.45;
  font-weight: 500;
}

.compose-preview {
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
  white-space: pre-wrap;
  font-weight: 500;
}

.muted.is-warn {
  color: var(--gold-500);
}

.path-preview {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 12px;
  border: 0.5px dashed var(--hairline-strong);
  border-radius: 10px;
  background: var(--bg);
  font-size: 12px;
  color: var(--text-faint);
}

.path-preview__flow {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  color: var(--text-dim);
}

.path-preview em {
  font-style: normal;
  font-weight: 600;
  color: var(--text);
}

.hint-banner {
  margin: 0;
  padding: 10px 12px;
  border-radius: 10px;
  background: var(--gold-dim);
  color: var(--gold-500);
  font-size: 13px;
  font-weight: 500;
}

.hint-banner a {
  color: inherit;
  text-decoration: underline;
}

.url-field {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 10px;
  border: 0.5px solid var(--hairline-strong);
  border-radius: var(--radius);
  background: var(--panel);
}

.url-field:focus-within {
  border-color: var(--viper-500);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--viper-500) 28%, transparent);
}

.url-field .ti {
  font-size: 15px;
  color: var(--text-faint);
  flex-shrink: 0;
}

.url-field__input {
  border: none !important;
  box-shadow: none !important;
  background: transparent !important;
  padding-left: 0 !important;
  padding-right: 0 !important;
  font-family: var(--mono);
  font-size: 12.5px;
}

.example {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 10px 12px;
  border-radius: 10px;
  background: var(--bg);
  border: 0.5px solid var(--hairline);
  font-size: 12px;
  color: var(--text-faint);
}

.example code {
  font-family: var(--mono);
  color: var(--text-dim);
  word-break: break-all;
}

.muted {
  font-weight: 500;
  color: var(--text-faint);
  white-space: nowrap;
}

.quick-picks {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.quick-pick {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  border: 0.5px solid var(--hairline);
  border-radius: 999px;
  background: var(--panel);
  color: var(--text-dim);
  font-size: 12.5px;
  font-weight: 600;
}

.quick-pick:hover:not(:disabled) {
  border-color: var(--viper-500);
  color: var(--viper-400);
  background: var(--viper-dim);
}

.quick-pick:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.account-tools {
  display: grid;
  grid-template-columns: 1.6fr 1fr;
  gap: 8px;
}

.bulk-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  flex-wrap: wrap;
  padding: 8px 10px;
  border: 0.5px solid var(--hairline);
  border-radius: 10px;
  background: var(--bg);
}

.bulk-bar__check {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 12.5px;
  font-weight: 500;
  color: var(--text-dim);
  cursor: pointer;
}

.account-list {
  max-height: min(200px, 28vh);
  overflow: auto;
  border: 0.5px solid var(--hairline);
  border-radius: 10px;
  background: var(--bg);
}

.account-list + :deep(.pager) {
  margin-top: 8px;
  border: 0.5px solid var(--hairline);
  border-radius: 10px;
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

.account-row:last-child { border-bottom: none; }
.account-row .ti { font-size: 16px; color: var(--text-dim); }

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

.status--active { background: var(--viper-dim); color: var(--viper-400); }
.status--expired { background: var(--gold-dim); color: var(--gold-500); }
.status--error { background: var(--bg-danger); color: var(--danger); }
.status--revoked { background: var(--panel-raised); color: var(--text-faint); }

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 28px 16px;
  border: 0.5px dashed var(--hairline-strong);
  border-radius: 12px;
  text-align: center;
  color: var(--text-dim);
}

.empty-state .ti {
  font-size: 28px;
  color: var(--text-faint);
}

.empty-state strong {
  color: var(--text);
}

.empty-state p {
  margin: 0;
  max-width: 36ch;
  font-size: 13px;
}

.summary-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 14px;
  border: 0.5px solid var(--hairline);
  border-radius: 12px;
  background: var(--bg);
}

.summary-card__hero {
  display: flex;
  gap: 12px;
  align-items: flex-start;
}

.summary-card__hero > .ti {
  width: 40px;
  height: 40px;
  display: grid;
  place-content: center;
  border-radius: 10px;
  background: var(--viper-dim);
  color: var(--viper-400);
  font-size: 18px;
}

.summary-card__hero strong {
  display: block;
  font-size: 15px;
  color: var(--text);
}

.summary-card__hero p {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--text-dim);
}

.summary-list {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  margin: 0;
}

.summary-list div {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.summary-list dt {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--text-faint);
  font-family: var(--mono);
}

.summary-list dd {
  margin: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--text);
  word-break: break-word;
}

.mono {
  font-family: var(--mono);
  font-size: 12px;
}

.selected-preview {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.show-browser {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  padding: 12px 14px;
  border: 0.5px solid var(--hairline);
  border-radius: 12px;
  background: var(--bg);
  cursor: pointer;
}

.show-browser input {
  margin-top: 3px;
}

.show-browser strong {
  display: block;
  font-size: 13px;
  color: var(--text);
}

.show-browser span {
  display: block;
  margin-top: 3px;
  font-size: 12.5px;
  color: var(--text-dim);
  line-height: 1.4;
  font-weight: 500;
}

.chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 9px;
  border: 0.5px solid var(--hairline);
  border-radius: 999px;
  background: var(--panel);
  font-size: 12px;
  color: var(--text-dim);
}

.chip--more { color: var(--text-faint); }

.advanced {
  border: 0.5px solid var(--hairline);
  border-radius: 10px;
  background: var(--bg);
  padding: 0 12px;
}

.advanced summary {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
  padding: 12px 0;
  cursor: pointer;
  list-style: none;
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}

.advanced summary::-webkit-details-marker { display: none; }

.advanced summary small {
  font-weight: 500;
  color: var(--text-faint);
}

.settings {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  align-items: end;
  padding-bottom: 12px;
}

.check {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text-dim);
  cursor: pointer;
}

.settings .help {
  grid-column: 1 / -1;
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
  justify-content: space-between;
  gap: 8px;
  padding-top: 4px;
  border-top: 0.5px solid var(--hairline);
  flex-shrink: 0;
  margin-top: auto;
}

.actions__right {
  display: flex;
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
  to { transform: rotate(360deg); }
}

@media (max-width: 720px) {
  .platform-grid,
  .type-grid,
  .account-tools,
  .settings,
  .summary-list {
    grid-template-columns: 1fr;
  }

  .steps__copy small {
    display: none;
  }

  .coach p {
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .actions {
    flex-direction: column;
  }

  .actions__right {
    width: 100%;
  }

  .actions__right .btn {
    flex: 1;
  }
}

@media (max-height: 780px) {
  .coach {
    display: none;
  }

  .steps__copy small {
    display: none;
  }

  .path-preview {
    display: none;
  }

  .compose-text {
    min-height: 72px;
    max-height: 140px;
  }
}
</style>
