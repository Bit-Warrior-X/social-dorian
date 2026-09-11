<template>
  <div>
    <div class="page-head">
      <div>
        <h1>Post management</h1>
        <p>Track posts, analyze tone, draft replies, and review what each account published or answered</p>
      </div>
      <button class="btn btn-primary" type="button" @click="showCompose = true">
        <i class="ti ti-plus" aria-hidden="true" />
        Track post
      </button>
    </div>

    <p v-if="loadError" class="banner">{{ loadError }}</p>

    <div class="stats">
      <div class="stat">
        <div class="stat__label">Tracked</div>
        <div class="stat__value">{{ posts.length }}</div>
      </div>
      <div class="stat">
        <div class="stat__label">Watching</div>
        <div class="stat__value is-live">{{ posts.filter((p) => p.status === 'watching').length }}</div>
      </div>
      <div class="stat">
        <div class="stat__label">Replies logged</div>
        <div class="stat__value is-success">{{ replyEngagements }}</div>
      </div>
      <div class="stat">
        <div class="stat__label">Posts logged</div>
        <div class="stat__value">{{ postEngagements }}</div>
      </div>
    </div>

    <div class="tabs">
      <button class="tab" type="button" :class="{ active: tab === 'posts' }" @click="tab = 'posts'">
        Tracked posts
      </button>
      <button class="tab" type="button" :class="{ active: tab === 'activity' }" @click="tab = 'activity'">
        Account activity
      </button>
      <button class="tab" type="button" :class="{ active: tab === 'studio' }" @click="tab = 'studio'">
        Reply studio
      </button>
    </div>

    <section v-if="tab === 'posts'" class="table-card">
      <div v-if="loading" class="empty">Loading tracked posts…</div>
      <template v-else>
        <div class="table-scroll">
          <table>
            <thead>
              <tr>
                <th>Post</th>
                <th>Analysis</th>
                <th>Suggestions</th>
                <th>Activity</th>
                <th class="col-actions">Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="post in pageItems" :key="post.id">
                <td>
                  <div class="task-cell">
                    <strong>#{{ post.id }} {{ post.title || 'Untitled post' }}</strong>
                    <span v-if="post.url" class="sub mono">{{ post.url }}</span>
                    <span v-else class="sub">{{ preview(post.bodyText) }}</span>
                    <span class="sub">{{ post.platform }} · {{ post.status }}</span>
                  </div>
                </td>
                <td>
                  <div class="task-cell">
                    <span class="status" :class="'sentiment--' + (post.analysis?.sentiment || 'neutral')">
                      {{ post.analysis?.sentiment || '—' }}
                    </span>
                    <span class="sub">{{ post.analysis?.summary || 'Not analyzed yet' }}</span>
                  </div>
                </td>
                <td>{{ (post.suggestedReplies || []).length }} drafts</td>
                <td>{{ post.replyCount || 0 }} replies</td>
                <td class="actions">
                  <button class="btn btn-icon" type="button" title="Open" @click="openPost(post)">
                    <i class="ti ti-eye" aria-hidden="true" />
                  </button>
                  <button
                    class="btn btn-icon"
                    type="button"
                    title="Re-analyze"
                    :disabled="analyzingId === post.id"
                    @click="reanalyze(post)"
                  >
                    <i class="ti" :class="analyzingId === post.id ? 'ti-loader-2 spin' : 'ti-brain'" aria-hidden="true" />
                  </button>
                  <button class="btn btn-icon" type="button" title="Delete" @click="removePost(post)">
                    <i class="ti ti-trash" aria-hidden="true" />
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-if="posts.length === 0" class="empty">No tracked posts yet. Add a URL or paste content to analyze.</p>
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
    </section>

    <section v-else-if="tab === 'activity'" class="table-card">
      <div class="filters">
        <select v-model="activityKind">
          <option value="">All actions</option>
          <option value="post">Posts</option>
          <option value="reply">Replies</option>
          <option value="report">Reports</option>
          <option value="browse">Browse</option>
          <option value="login">Login</option>
        </select>
        <select v-model="activityAccountId">
          <option :value="0">All accounts</option>
          <option v-for="account in accounts" :key="account.id" :value="account.id">
            {{ accountLabel(account) }}
          </option>
        </select>
      </div>
      <div class="table-scroll">
        <table>
          <thead>
            <tr>
              <th>When</th>
              <th>Account</th>
              <th>Action</th>
              <th>Target / content</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in filteredEngagements" :key="row.id">
              <td class="mono">{{ formatWhen(row.createdAt) }}</td>
              <td>
                <div class="task-cell">
                  <strong>{{ row.accountName || '—' }}</strong>
                  <span class="sub">{{ row.platform }} · {{ row.accountEmail }}</span>
                </div>
              </td>
              <td>
                <span class="type">{{ row.kind }}</span>
                <span v-if="row.taskId" class="sub"> task #{{ row.taskId }}</span>
              </td>
              <td>
                <div class="task-cell">
                  <span v-if="row.targetUrl" class="mono sub">{{ row.targetUrl }}</span>
                  <span>{{ preview(row.content, 140) || '—' }}</span>
                </div>
              </td>
              <td>
                <span class="status" :class="'status--' + row.status">{{ row.status }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-if="filteredEngagements.length === 0" class="empty">
        No account activity yet. Successful post/reply/report jobs will appear here.
      </p>
    </section>

    <section v-else class="studio">
      <div class="studio__compose">
        <label class="field">
          <span>Post text to analyze</span>
          <textarea v-model="studioText" rows="8" placeholder="Paste the post content here…" />
        </label>
        <div class="studio__row">
          <label class="field">
            <span>Platform</span>
            <select v-model="studioPlatform">
              <option value="facebook">Facebook</option>
              <option value="twitter">Twitter / X</option>
              <option value="instagram">Instagram</option>
              <option value="linkedin">LinkedIn</option>
            </select>
          </label>
          <button class="btn btn-primary" type="button" :disabled="studioBusy || !studioText.trim()" @click="runStudio">
            <i class="ti" :class="studioBusy ? 'ti-loader-2 spin' : 'ti-sparkles'" aria-hidden="true" />
            {{ studioBusy ? 'Analyzing…' : 'Analyze & suggest' }}
          </button>
        </div>
      </div>
      <div v-if="studioAnalysis" class="studio__result">
        <div class="analysis-card">
          <div class="analysis-card__head">
            <strong>Analysis</strong>
            <span class="status" :class="'sentiment--' + studioAnalysis.sentiment">{{ studioAnalysis.sentiment }}</span>
          </div>
          <p>{{ studioAnalysis.summary }}</p>
          <p class="sub">Tone: {{ studioAnalysis.tone }} · Engine: {{ studioAnalysis.engine }}</p>
          <div v-if="studioAnalysis.topics?.length" class="chips">
            <span v-for="topic in studioAnalysis.topics" :key="topic" class="chip">{{ topic }}</span>
          </div>
          <ul v-if="studioAnalysis.hooks?.length" class="bullets">
            <li v-for="hook in studioAnalysis.hooks" :key="hook">{{ hook }}</li>
          </ul>
          <ul v-if="studioAnalysis.risks?.length" class="bullets bullets--warn">
            <li v-for="risk in studioAnalysis.risks" :key="risk">{{ risk }}</li>
          </ul>
        </div>
        <div class="suggestions">
          <strong>Suggested replies</strong>
          <article v-for="(reply, index) in studioReplies" :key="index" class="suggestion">
            <p>{{ reply }}</p>
            <div class="suggestion__actions">
              <button class="btn" type="button" @click="copyReply(reply)">Copy</button>
              <button class="btn btn-primary" type="button" @click="launchReply('', reply)">
                Launch reply task
              </button>
            </div>
          </article>
        </div>
      </div>
    </section>

    <ModalDialog :open="showCompose" title="Track a post" title-id="track-post-title" wide @close="showCompose = false">
      <form class="compose" @submit.prevent="saveTracked">
        <label class="field">
          <span>Post URL</span>
          <input v-model="compose.url" type="url" placeholder="https://www.facebook.com/…" />
        </label>
        <label class="field">
          <span>Post text (paste for better analysis)</span>
          <textarea v-model="compose.bodyText" rows="6" placeholder="Paste the original post body…" />
        </label>
        <div class="compose__row">
          <label class="field">
            <span>Platform</span>
            <select v-model="compose.platform">
              <option value="facebook">Facebook</option>
              <option value="twitter">Twitter / X</option>
              <option value="instagram">Instagram</option>
              <option value="linkedin">LinkedIn</option>
            </select>
          </label>
          <label class="field">
            <span>Author (optional)</span>
            <input v-model="compose.author" type="text" placeholder="Page or person name" />
          </label>
        </div>
        <p v-if="composeError" class="banner">{{ composeError }}</p>
        <div class="compose__actions">
          <button class="btn" type="button" @click="showCompose = false">Cancel</button>
          <button class="btn btn-primary" type="submit" :disabled="saving">
            {{ saving ? 'Saving…' : 'Track & analyze' }}
          </button>
        </div>
      </form>
    </ModalDialog>

    <ModalDialog
      :open="Boolean(selected)"
      :title="selected ? `Post #${selected.id}` : 'Post'"
      title-id="post-detail-title"
      wide
      @close="selected = null"
    >
      <div v-if="selected" class="detail">
        <p v-if="selected.url" class="mono">{{ selected.url }}</p>
        <p class="body">{{ selected.bodyText || 'No body text stored.' }}</p>
        <div class="analysis-card">
          <div class="analysis-card__head">
            <strong>Analysis</strong>
            <span class="status" :class="'sentiment--' + (selected.analysis?.sentiment || 'neutral')">
              {{ selected.analysis?.sentiment || '—' }}
            </span>
          </div>
          <p>{{ selected.analysis?.summary || 'Run analyze to fill this in.' }}</p>
          <div v-if="selected.analysis?.topics?.length" class="chips">
            <span v-for="topic in selected.analysis.topics" :key="topic" class="chip">{{ topic }}</span>
          </div>
        </div>
        <div class="suggestions">
          <strong>Suggested replies</strong>
          <article v-for="(reply, index) in selected.suggestedReplies || []" :key="index" class="suggestion">
            <p>{{ reply }}</p>
            <div class="suggestion__actions">
              <button class="btn" type="button" @click="copyReply(reply)">Copy</button>
              <button class="btn btn-primary" type="button" @click="launchReply(selected.url, reply)">
                Launch reply task
              </button>
            </div>
          </article>
          <p v-if="!(selected.suggestedReplies || []).length" class="empty">No suggestions yet.</p>
        </div>
      </div>
    </ModalDialog>

    <NewTaskModal
      :open="taskModalOpen"
      :accounts="accounts"
      :initial-type="taskModalType"
      :initial-account-ids="[]"
      :initial-target-url="taskModalUrl"
      :initial-reply-text="taskModalReply"
      :saving="taskSaving"
      :error="taskError"
      @close="taskModalOpen = false"
      @submit="launchTask"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import ModalDialog from '../components/ModalDialog.vue'
import NewTaskModal from '../components/NewTaskModal.vue'
import PaginationBar from '../components/PaginationBar.vue'
import { listAccounts } from '../api/accounts'
import {
  analyzeTrackedPost,
  createTrackedPost,
  deleteTrackedPost,
  listEngagements,
  listTrackedPosts,
  suggestReplies,
} from '../api/posts'
import { createTask } from '../api/tasks'
import { useNotify } from '../composables/useNotify'
import { usePagination } from '../composables/usePagination'
import { accountDisplayName } from '../constants/accounts'
import { taskTypeMeta } from '../constants/tasks'

const { notifySuccess, notifyError } = useNotify()
const router = useRouter()

const tab = ref('posts')
const posts = ref([])
const engagements = ref([])
const accounts = ref([])
const loading = ref(true)
const loadError = ref('')
const showCompose = ref(false)
const saving = ref(false)
const composeError = ref('')
const selected = ref(null)
const analyzingId = ref(0)
const activityKind = ref('')
const activityAccountId = ref(0)

const studioText = ref('')
const studioPlatform = ref('facebook')
const studioBusy = ref(false)
const studioAnalysis = ref(null)
const studioReplies = ref([])

const taskModalOpen = ref(false)
const taskModalType = ref('reply')
const taskModalUrl = ref('')
const taskModalReply = ref('')
const taskSaving = ref(false)
const taskError = ref('')

const compose = reactive({
  url: '',
  bodyText: '',
  platform: 'facebook',
  author: '',
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
} = usePagination(posts)

const replyEngagements = computed(
  () => engagements.value.filter((row) => row.kind === 'reply' && row.status === 'success').length,
)
const postEngagements = computed(
  () => engagements.value.filter((row) => row.kind === 'post' && row.status === 'success').length,
)

const filteredEngagements = computed(() => {
  return engagements.value.filter((row) => {
    if (activityKind.value && row.kind !== activityKind.value) return false
    if (activityAccountId.value && row.accountId !== Number(activityAccountId.value)) return false
    return true
  })
})

function preview(text, max = 100) {
  const value = String(text || '').trim()
  if (!value) return ''
  return value.length > max ? `${value.slice(0, max)}…` : value
}

function formatWhen(value) {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

function accountLabel(account) {
  return `${accountDisplayName(account)} · ${account.email || account.id}`
}

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    const [postRows, engagementRows, accountRows] = await Promise.all([
      listTrackedPosts(),
      listEngagements({ limit: 200 }),
      listAccounts().catch(() => []),
    ])
    posts.value = postRows
    engagements.value = engagementRows
    accounts.value = accountRows
  } catch (err) {
    loadError.value = err.message || 'Could not load post management'
    notifyError(loadError.value)
  } finally {
    loading.value = false
  }
}

async function saveTracked() {
  saving.value = true
  composeError.value = ''
  try {
    const post = await createTrackedPost({ ...compose })
    posts.value = [post, ...posts.value]
    showCompose.value = false
    compose.url = ''
    compose.bodyText = ''
    compose.author = ''
    selected.value = post
    notifySuccess(`Tracked post #${post.id}`)
  } catch (err) {
    composeError.value = err.message || 'Could not track post'
  } finally {
    saving.value = false
  }
}

function openPost(post) {
  selected.value = post
}

async function reanalyze(post) {
  analyzingId.value = post.id
  try {
    const updated = await analyzeTrackedPost(post.id)
    posts.value = posts.value.map((row) => (row.id === updated.id ? updated : row))
    if (selected.value?.id === updated.id) selected.value = updated
    notifySuccess('Analysis updated')
  } catch (err) {
    notifyError(err.message || 'Could not analyze post')
  } finally {
    analyzingId.value = 0
  }
}

async function removePost(post) {
  try {
    await deleteTrackedPost(post.id)
    posts.value = posts.value.filter((row) => row.id !== post.id)
    if (selected.value?.id === post.id) selected.value = null
    notifySuccess('Tracked post removed')
  } catch (err) {
    notifyError(err.message || 'Could not delete post')
  }
}

async function runStudio() {
  studioBusy.value = true
  try {
    const data = await suggestReplies({
      bodyText: studioText.value,
      platform: studioPlatform.value,
    })
    studioAnalysis.value = data.analysis
    studioReplies.value = data.suggestedReplies || []
  } catch (err) {
    notifyError(err.message || 'Could not analyze text')
  } finally {
    studioBusy.value = false
  }
}

async function copyReply(text) {
  try {
    await navigator.clipboard.writeText(text)
    notifySuccess('Reply copied')
  } catch (err) {
    notifyError(err.message || 'Could not copy')
  }
}

function launchReply(url, reply) {
  taskModalType.value = 'reply'
  taskModalUrl.value = url || selected.value?.url || ''
  taskModalReply.value = reply || ''
  taskError.value = ''
  taskModalOpen.value = true
  selected.value = null
}

async function launchTask(payload) {
  taskSaving.value = true
  taskError.value = ''
  try {
    const task = await createTask(payload)
    taskModalOpen.value = false
    notifySuccess(`Launched ${taskTypeMeta(task.type).label} #${task.id}`)
    await router.push('/tasks/active')
  } catch (err) {
    taskError.value = err.message || 'Could not launch task'
    notifyError(taskError.value)
  } finally {
    taskSaving.value = false
  }
}

watch([activityKind, activityAccountId], async () => {
  try {
    engagements.value = await listEngagements({
      accountId: Number(activityAccountId.value) || 0,
      kind: activityKind.value,
      limit: 200,
    })
  } catch (_) {
    // Keep current list.
  }
})

onMounted(load)
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
  margin-top: 4px;
  color: var(--viper-400);
}

.stat__value.is-live { color: var(--warn); }
.stat__value.is-success { color: var(--success); }

.tabs {
  display: flex;
  gap: 6px;
  margin-bottom: 1rem;
  flex-wrap: wrap;
}

.tab {
  border: 0.5px solid var(--hairline);
  background: var(--panel);
  color: var(--text-dim);
  border-radius: 999px;
  padding: 6px 12px;
  font-size: 13px;
  cursor: pointer;
}

.tab.active {
  background: var(--viper-dim);
  color: var(--viper-400);
  border-color: color-mix(in srgb, var(--viper-500) 40%, transparent);
}

.filters {
  display: flex;
  gap: 8px;
  padding: 12px;
  flex-wrap: wrap;
}

.filters select {
  width: 180px;
}

.table-card {
  background: var(--panel);
  border: 0.5px solid var(--hairline);
  border-radius: 12px;
  overflow: hidden;
}

.table-scroll { overflow-x: auto; }

table {
  width: 100%;
  min-width: 900px;
  border-collapse: collapse;
  font-size: 14px;
}

th, td {
  padding: 10px 12px;
  text-align: left;
  border-bottom: 0.5px solid var(--hairline);
  vertical-align: top;
}

th { font-weight: 500; color: var(--text-dim); }
.col-actions, .actions { text-align: right; }

.task-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.sub {
  font-size: 12px;
  color: var(--text-faint);
}

.mono { font-family: var(--mono); }

.type {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.status {
  display: inline-flex;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
  text-transform: capitalize;
  background: var(--panel-raised);
  width: fit-content;
}

.status--success { background: var(--viper-dim); color: var(--viper-400); }
.status--failed { background: var(--bg-danger); color: var(--danger); }
.sentiment--positive { background: var(--viper-dim); color: var(--viper-400); }
.sentiment--negative { background: var(--bg-danger); color: var(--danger); }
.sentiment--neutral { color: var(--text-dim); }

.empty {
  margin: 0;
  padding: 2rem;
  text-align: center;
  color: var(--text-dim);
}

.studio {
  display: grid;
  grid-template-columns: minmax(280px, 0.9fr) minmax(320px, 1.1fr);
  gap: 12px;
}

.studio__compose,
.studio__result,
.analysis-card,
.suggestion,
.compose,
.detail {
  background: var(--panel);
  border: 0.5px solid var(--hairline);
  border-radius: 12px;
  padding: 14px;
}

.studio__result {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 12px;
  font-size: 13px;
  color: var(--text-dim);
}

.field textarea,
.field input,
.field select {
  width: 100%;
}

.studio__row,
.compose__row,
.compose__actions,
.suggestion__actions,
.analysis-card__head {
  display: flex;
  gap: 8px;
  align-items: end;
  flex-wrap: wrap;
}

.analysis-card__head {
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
}

.chip {
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--panel-raised);
  font-size: 11px;
}

.bullets {
  margin: 8px 0 0;
  padding-left: 18px;
  color: var(--text-dim);
  font-size: 13px;
}

.bullets--warn { color: var(--warn); }

.suggestions {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.suggestion p {
  margin: 0 0 10px;
  white-space: pre-wrap;
}

.body {
  white-space: pre-wrap;
  line-height: 1.45;
}

.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

@media (max-width: 980px) {
  .studio {
    grid-template-columns: 1fr;
  }
}
</style>
