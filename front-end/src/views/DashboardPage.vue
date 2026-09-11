<template>
  <div>
    <div class="page-head">
      <div>
        <h1>Dashboard</h1>
        <p>Workspace overview across infrastructure and operations</p>
      </div>
      <div class="page-head__actions">
        <span class="freshness" :class="{ 'is-live': autoRefresh && !loadError }">
          <i
            class="ti"
            :class="refreshing ? 'ti-loader-2 spin' : 'ti-point-filled'"
            aria-hidden="true"
          />
          {{ freshnessLabel }}
        </span>
        <button
          class="btn"
          type="button"
          :class="{ 'is-on': autoRefresh }"
          :aria-pressed="autoRefresh"
          @click="autoRefresh = !autoRefresh"
        >
          <i class="ti ti-activity-heartbeat" aria-hidden="true" />
          Auto
        </button>
        <button class="btn" type="button" :disabled="refreshing" @click="load()">
          <i class="ti ti-refresh" aria-hidden="true" />
          Refresh
        </button>
        <button
          class="btn btn-primary"
          type="button"
          :disabled="checking || !summary?.proxies.total"
          @click="runProxyCheck"
        >
          <i class="ti" :class="checking ? 'ti-loader-2 spin' : 'ti-heartbeat'" aria-hidden="true" />
          {{ checking ? 'Checking…' : 'Check proxies' }}
        </button>
        <RouterLink class="btn" to="/tasks/new">
          <i class="ti ti-plus" aria-hidden="true" />
          New task
        </RouterLink>
      </div>
    </div>

    <p v-if="loadError" class="banner">{{ loadError }}</p>

    <div v-if="loading" class="empty-panel">Loading dashboard…</div>

    <template v-else-if="summary">
      <div class="stats">
        <RouterLink v-for="tile in tiles" :key="tile.key" class="stat" :to="tile.to">
          <div class="stat__head">
            <span class="stat__label">{{ tile.label }}</span>
            <i class="ti" :class="tile.icon" aria-hidden="true" />
          </div>
          <div class="stat__value" :class="tile.tone && 'is-' + tile.tone">{{ tile.value }}</div>
          <div class="stat__meta">{{ tile.meta }}</div>
        </RouterLink>
      </div>

      <div class="ops-strip">
        <RouterLink class="ops-card" to="/tasks/active">
          <div class="ops-card__label">Active jobs</div>
          <div class="ops-card__value">{{ summary.tasks?.active || 0 }}</div>
          <div class="ops-card__meta">
            {{ summary.tasks?.running || 0 }} running · {{ summary.tasks?.queued || 0 }} queued
          </div>
        </RouterLink>
        <RouterLink class="ops-card" to="/tasks/history">
          <div class="ops-card__label">Tasks (24h)</div>
          <div class="ops-card__value">{{ summary.tasks?.last24h || 0 }}</div>
          <div class="ops-card__meta">
            {{ summary.tasks?.completed || 0 }} completed · {{ summary.tasks?.failed || 0 }} failed
          </div>
        </RouterLink>
        <RouterLink class="ops-card" to="/accounts">
          <div class="ops-card__label">Busy accounts</div>
          <div class="ops-card__value" :class="{ 'is-warn': summary.tasks?.busyAccounts }">
            {{ summary.tasks?.busyAccounts || 0 }}
          </div>
          <div class="ops-card__meta">Currently assigned to a job</div>
        </RouterLink>
        <RouterLink class="ops-card" to="/credits">
          <div class="ops-card__label">Credits</div>
          <div class="ops-card__value">{{ (summary.credits?.balance || 0).toLocaleString() }}</div>
          <div class="ops-card__meta">
            Updated {{ summary.credits?.updatedAt ? formatDateTime(summary.credits.updatedAt) : '—' }}
          </div>
        </RouterLink>
        <div class="ops-card ops-card--static">
          <div class="ops-card__label">Routing mix</div>
          <div class="ops-card__value">{{ summary.routing.routed }}/{{ summary.accounts.total }}</div>
          <div class="ops-card__meta">
            Auto {{ summary.routing.auto }} · Manual {{ summary.routing.manual }} · None {{ summary.routing.none }}
          </div>
        </div>
      </div>

      <div class="grid">
        <section class="panel col-8">
          <header class="panel__head">
            <div>
              <h2>Connections over time</h2>
              <p>New accounts connected per month</p>
            </div>
            <div class="panel__meta">
              <span><strong>{{ trendTotal }}</strong> in 12 months</span>
              <span><strong>{{ summary.accounts.addedLast30 }}</strong> in 30 days</span>
            </div>
          </header>
          <TrendChart :points="summary.trend" />
        </section>

        <section class="panel col-4">
          <header class="panel__head">
            <div>
              <h2>Account status</h2>
              <p>Health of connected accounts</p>
            </div>
          </header>
          <div class="split">
            <DonutChart
              :segments="accountSegments"
              :center-value="summary.accounts.total"
              center-label="Accounts"
            />
            <ul class="legend">
              <li v-for="segment in accountSegments" :key="segment.status">
                <RouterLink :to="{ path: '/accounts', query: { status: segment.status } }">
                  <span class="legend__dot" :style="{ background: segment.color }" />
                  <span class="legend__label">{{ segment.label }}</span>
                  <strong class="legend__value">{{ segment.value }}</strong>
                  <span class="legend__pct">{{ share(segment.value, summary.accounts.total) }}%</span>
                </RouterLink>
              </li>
            </ul>
          </div>
        </section>

        <section class="panel col-6">
          <header class="panel__head">
            <div>
              <h2>Platforms</h2>
              <p>Accounts by network</p>
            </div>
            <RouterLink class="panel__link" to="/accounts">View all</RouterLink>
          </header>
          <p v-if="platformRows.length === 0" class="empty-inline">No accounts yet.</p>
          <ul v-else class="bar-list">
            <li v-for="row in platformRows" :key="row.platform">
              <RouterLink :to="{ path: '/accounts', query: { platform: row.platform } }">
                <div class="bar-list__label">
                  <i class="ti" :class="row.icon" aria-hidden="true" />
                  <span>{{ row.label }}</span>
                  <span v-if="row.attention" class="pill pill--warn">{{ row.attention }} issue{{ row.attention === 1 ? '' : 's' }}</span>
                  <strong>{{ row.total }}</strong>
                </div>
                <div class="bar-list__track">
                  <div class="bar-list__fill" :style="{ width: row.width + '%' }">
                    <span class="bar-list__active" :style="{ width: row.activeShare + '%' }" />
                  </div>
                </div>
              </RouterLink>
            </li>
          </ul>
        </section>

        <section class="panel col-6">
          <header class="panel__head">
            <div>
              <h2>Proxy health</h2>
              <p>Status and account load</p>
            </div>
            <RouterLink class="panel__link" to="/proxies">Manage</RouterLink>
          </header>
          <div class="split">
            <DonutChart
              :segments="proxySegments"
              :center-value="summary.proxies.total"
              center-label="Proxies"
            />
            <ul class="legend">
              <li v-for="segment in proxySegments" :key="segment.status">
                <RouterLink :to="{ path: '/proxies', query: { status: segment.status } }">
                  <span class="legend__dot" :style="{ background: segment.color }" />
                  <span class="legend__label">{{ segment.label }}</span>
                  <strong class="legend__value">{{ segment.value }}</strong>
                  <span class="legend__pct">{{ share(segment.value, summary.proxies.total) }}%</span>
                </RouterLink>
              </li>
              <li class="legend__note">
                {{ summary.proxies.unused }} carrying no accounts
              </li>
            </ul>
          </div>
          <ul class="load-list">
            <li v-for="proxy in summary.topProxies" :key="proxy.id">
              <RouterLink :to="{ path: '/proxies', query: { q: proxy.name } }">
                <div class="load-list__main">
                  <span class="load-list__name">{{ proxy.name }}</span>
                  <span class="status status--sm" :class="'status--' + proxy.status">
                    {{ proxyStatusLabel(proxy.status) }}
                  </span>
                </div>
                <div class="load-list__meta">
                  <span>{{ proxy.country || '—' }} · {{ protocolLabel(proxy.protocol) }}</span>
                  <span>{{ proxy.accountCount }} account{{ proxy.accountCount === 1 ? '' : 's' }}</span>
                </div>
              </RouterLink>
            </li>
            <li v-if="summary.topProxies.length === 0" class="empty-inline">No proxies registered.</li>
          </ul>
        </section>

        <section class="panel col-6">
          <header class="panel__head">
            <div>
              <h2>Geography</h2>
              <p>Where traffic is routed</p>
            </div>
          </header>
          <p v-if="countryRows.length === 0" class="empty-inline">No proxy locations set.</p>
          <ul v-else class="bar-list">
            <li v-for="row in countryRows" :key="row.country">
              <RouterLink :to="{ path: '/proxies', query: { q: row.country } }">
                <div class="bar-list__label">
                  <span class="code">{{ row.country }}</span>
                  <span>{{ row.name }}</span>
                  <strong>{{ row.accounts }}</strong>
                </div>
                <div class="bar-list__track">
                  <div class="bar-list__fill" :style="{ width: row.width + '%' }" />
                </div>
                <div class="bar-list__sub">
                  {{ row.proxies }} prox{{ row.proxies === 1 ? 'y' : 'ies' }} ·
                  {{ row.accounts }} account{{ row.accounts === 1 ? '' : 's' }}
                </div>
              </RouterLink>
            </li>
          </ul>
        </section>

        <section class="panel col-6">
          <header class="panel__head">
            <div>
              <h2>Mailboxes</h2>
              <p>Gmail coverage for connected accounts</p>
            </div>
            <RouterLink class="panel__link" to="/gmails">Manage</RouterLink>
          </header>
          <div class="split">
            <DonutChart
              :segments="gmailSegments"
              :center-value="summary.gmails.total"
              center-label="Mailboxes"
            />
            <ul class="legend">
              <li v-for="segment in gmailSegments" :key="segment.status">
                <RouterLink :to="{ path: '/gmails', query: { status: segment.status } }">
                  <span class="legend__dot" :style="{ background: segment.color }" />
                  <span class="legend__label">{{ segment.label }}</span>
                  <strong class="legend__value">{{ segment.value }}</strong>
                  <span class="legend__pct">{{ share(segment.value, summary.gmails.total) }}%</span>
                </RouterLink>
              </li>
            </ul>
          </div>
          <div class="callouts">
            <RouterLink class="callout" :class="{ 'is-ok': !summary.gmails.unlinked }" to="/gmails">
              <strong>{{ summary.gmails.unlinked }}</strong>
              <span>mailboxes not linked to an account</span>
            </RouterLink>
            <RouterLink
              class="callout"
              :class="{ 'is-ok': !summary.gmails.unmanagedAccounts }"
              to="/accounts"
            >
              <strong>{{ summary.gmails.unmanagedAccounts }}</strong>
              <span>accounts on an unregistered mailbox</span>
            </RouterLink>
          </div>
        </section>

        <section class="panel col-8">
          <header class="panel__head">
            <div>
              <h2>Recent tasks</h2>
              <p>Latest campaign and automation jobs</p>
            </div>
            <RouterLink class="panel__link" to="/tasks">View all</RouterLink>
          </header>
          <div class="table-wrap">
            <table v-if="(summary.recentTasks || []).length">
              <thead>
                <tr>
                  <th>Task</th>
                  <th>Type</th>
                  <th>Status</th>
                  <th>Progress</th>
                  <th>Created</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="task in summary.recentTasks" :key="task.id">
                  <td>
                    <div class="stack">
                      <span class="account-cell__name">#{{ task.id }} {{ task.title }}</span>
                      <span v-if="task.targetUrl" class="sub mono-soft">{{ task.targetUrl }}</span>
                      <span v-else-if="taskContentPreview(task)" class="sub">{{ taskContentPreview(task) }}</span>
                    </div>
                  </td>
                  <td>
                    <span class="type-chip">
                      <i class="ti" :class="taskTypeMeta(task.type).icon" aria-hidden="true" />
                      {{ taskTypeMeta(task.type).label }}
                    </span>
                  </td>
                  <td>
                    <span class="status status--sm" :class="'status--' + task.status">
                      {{ taskStatusLabel(task.status) }}
                    </span>
                  </td>
                  <td class="muted">
                    {{ task.doneCount }}/{{ task.accountCount }}
                    <template v-if="task.successCount || task.failCount">
                      · {{ task.successCount }} ok
                      <template v-if="task.failCount"> / {{ task.failCount }} fail</template>
                    </template>
                  </td>
                  <td>
                    <div class="stack">
                      <span>{{ task.createdBy || '—' }}</span>
                      <span class="sub">{{ formatDateTime(task.createdAt) }}</span>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
            <p v-else class="empty-inline">No tasks launched yet.</p>
          </div>
        </section>

        <section class="panel col-4">
          <header class="panel__head">
            <div>
              <h2>Quick actions</h2>
              <p>Jump into common workflows</p>
            </div>
          </header>
          <div class="quick-actions">
            <RouterLink class="quick" to="/tasks?new=report">
              <i class="ti ti-flag" aria-hidden="true" />
              <div>
                <strong>Report post</strong>
                <span>Select accounts and launch a report job</span>
              </div>
            </RouterLink>
            <RouterLink class="quick" to="/tasks?new=reply">
              <i class="ti ti-message" aria-hidden="true" />
              <div>
                <strong>Reply / comment</strong>
                <span>Comment on a post URL from selected accounts</span>
              </div>
            </RouterLink>
            <RouterLink class="quick" to="/tasks?new=browse">
              <i class="ti ti-player-play" aria-hidden="true" />
              <div>
                <strong>Browse feed</strong>
                <span>Browse feed activity across idle accounts</span>
              </div>
            </RouterLink>
            <RouterLink class="quick" to="/accounts">
              <i class="ti ti-share" aria-hidden="true" />
              <div>
                <strong>Social accounts</strong>
                <span>Inventory, proxies, and open sessions</span>
              </div>
            </RouterLink>
            <RouterLink class="quick" to="/proxies">
              <i class="ti ti-network" aria-hidden="true" />
              <div>
                <strong>Proxy health</strong>
                <span>{{ summary.proxies.issues }} issue{{ summary.proxies.issues === 1 ? '' : 's' }} · {{ summary.proxies.unused }} unused</span>
              </div>
            </RouterLink>
          </div>
        </section>

        <section class="panel col-8">
          <header class="panel__head">
            <div>
              <h2>Recent accounts</h2>
              <p>Latest connections</p>
            </div>
            <RouterLink class="panel__link" to="/accounts">View all</RouterLink>
          </header>
          <div class="table-wrap">
            <table v-if="summary.recentAccounts.length">
              <thead>
                <tr>
                  <th>Account</th>
                  <th>Status</th>
                  <th>Proxy</th>
                  <th>Connected</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="account in summary.recentAccounts" :key="account.id">
                  <td>
                    <div class="account-cell">
                      <i class="ti" :class="platformMeta(account.platform).icon" aria-hidden="true" />
                      <div>
                        <div class="account-cell__name">{{ account.name }}</div>
                        <div class="account-cell__sub">{{ account.email }}</div>
                      </div>
                    </div>
                  </td>
                  <td>
                    <span class="status status--sm" :class="'status--' + account.status">
                      {{ statusLabel(account.status) }}
                    </span>
                  </td>
                  <td class="muted">{{ recentProxyLabel(account) }}</td>
                  <td>
                    <div class="stack">
                      <span>{{ account.connectedBy || '—' }}</span>
                      <span class="sub">{{ formatDate(account.connectedAt) }}</span>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
            <p v-else class="empty-inline">No accounts connected yet.</p>
          </div>
        </section>

        <section class="panel col-4">
          <header class="panel__head">
            <div>
              <h2>Attention</h2>
              <p>Items that need a closer look</p>
            </div>
          </header>
          <ul v-if="attentionRows.length" class="attention-list">
            <li v-for="item in attentionRows" :key="item.id">
              <div class="attention__icon" :class="'attention__icon--' + item.tone" aria-hidden="true">
                <i class="ti" :class="item.icon" />
              </div>
              <div class="attention__body">
                <div class="attention__title">{{ item.title }}</div>
                <div class="attention__desc">{{ item.description }}</div>
              </div>
              <RouterLink class="attention__link" :to="item.to">Open</RouterLink>
            </li>
          </ul>
          <p v-else class="empty-inline is-ok">Everything looks healthy.</p>
        </section>
      </div>
    </template>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import DonutChart from '../components/DonutChart.vue'
import TrendChart from '../components/TrendChart.vue'
import { getDashboard } from '../api/dashboard'
import { checkAllProxies } from '../api/proxies'
import { useNotify } from '../composables/useNotify'
import { STATUSES, platformMeta, proxyModeLabel, statusLabel } from '../constants/accounts'
import { COUNTRIES } from '../constants/countries'
import { GMAIL_STATUSES, gmailStatusLabel } from '../constants/gmails'
import { PROXY_STATUSES, protocolLabel, proxyStatusLabel } from '../constants/proxies'
import { taskContentPreview, taskStatusLabel, taskTypeMeta } from '../constants/tasks'

const REFRESH_MS = 30000

const STATUS_COLORS = {
  active: 'var(--viper-400)',
  expired: 'var(--gold-500)',
  inactive: 'var(--gold-500)',
  error: 'var(--danger)',
  revoked: 'var(--text-faint)',
}

const { notifySuccess, notifyError } = useNotify()

const summary = ref(null)
const loading = ref(true)
const refreshing = ref(false)
const checking = ref(false)
const loadError = ref('')
const autoRefresh = ref(true)
const lastUpdated = ref(0)
const clock = ref(Date.now())

let refreshTimer = null
let clockTimer = null

const freshnessLabel = computed(() => {
  if (refreshing.value) return 'Updating…'
  if (!lastUpdated.value) return 'Not loaded'
  const seconds = Math.max(0, Math.round((clock.value - lastUpdated.value) / 1000))
  if (seconds < 5) return 'Updated just now'
  if (seconds < 60) return `Updated ${seconds}s ago`
  return `Updated ${Math.floor(seconds / 60)}m ago`
})

const tiles = computed(() => {
  const data = summary.value
  if (!data) return []
  return [
    {
      key: 'accounts',
      label: 'Accounts',
      icon: 'ti-share',
      value: data.accounts.total,
      meta: `${data.accounts.active} active · ${data.accounts.addedLast30} in 30d`,
      to: { path: '/accounts' },
    },
    {
      key: 'account-issues',
      label: 'Account issues',
      icon: 'ti-alert-triangle',
      value: data.accounts.attention,
      tone: data.accounts.attention ? 'warning' : 'success',
      meta: `${data.accounts.expired} expired · ${data.accounts.error} error`,
      to: { path: '/accounts', query: { status: 'error' } },
    },
    {
      key: 'active-jobs',
      label: 'Active jobs',
      icon: 'ti-player-play',
      value: data.tasks?.active || 0,
      tone: data.tasks?.running ? 'warning' : '',
      meta: `${data.tasks?.running || 0} running · ${data.tasks?.busyAccounts || 0} busy accounts`,
      to: { path: '/tasks/active' },
    },
    {
      key: 'task-fails',
      label: 'Failed tasks',
      icon: 'ti-xbox-x',
      value: data.tasks?.failed || 0,
      tone: data.tasks?.failed ? 'danger' : 'success',
      meta: `${data.tasks?.completed || 0} completed · ${data.tasks?.last24h || 0} in 24h`,
      to: { path: '/tasks/history' },
    },
    {
      key: 'proxies',
      label: 'Proxies',
      icon: 'ti-network',
      value: data.proxies.total,
      meta: `${data.proxies.active} active · ${data.proxies.unused} unused`,
      to: { path: '/proxies' },
    },
    {
      key: 'proxy-issues',
      label: 'Proxy issues',
      icon: 'ti-plug-off',
      value: data.proxies.issues,
      tone: data.proxies.issues ? 'danger' : 'success',
      meta: `${data.proxies.error} error · ${data.proxies.inactive} inactive`,
      to: { path: '/proxies', query: { status: 'error' } },
    },
    {
      key: 'gmails',
      label: 'Mailboxes',
      icon: 'ti-brand-gmail',
      value: data.gmails.total,
      meta: `${data.gmails.active} active · ${data.gmails.unlinked} unlinked`,
      to: { path: '/gmails' },
    },
    {
      key: 'credits',
      label: 'Credits',
      icon: 'ti-coin',
      value: (data.credits?.balance || 0).toLocaleString(),
      meta: data.routing.unassigned
        ? `${data.routing.unassigned} unrouted accounts`
        : `${data.routing.routed} accounts routed`,
      to: { path: '/credits' },
    },
  ]
})

const trendTotal = computed(() =>
  (summary.value?.trend || []).reduce((total, point) => total + point.count, 0),
)

const accountSegments = computed(() => statusSegments(STATUSES, summary.value?.accounts))
const proxySegments = computed(() => statusSegments(PROXY_STATUSES, summary.value?.proxies))
const gmailSegments = computed(() => statusSegments(GMAIL_STATUSES, summary.value?.gmails))

const platformRows = computed(() => {
  const rows = summary.value?.platforms || []
  const max = Math.max(1, ...rows.map((row) => row.total))
  return rows.map((row) => {
    const meta = platformMeta(row.platform)
    return {
      ...row,
      label: meta.label,
      icon: meta.icon,
      width: (row.total / max) * 100,
      activeShare: row.total ? (row.active / row.total) * 100 : 0,
    }
  })
})

const countryRows = computed(() => {
  const rows = summary.value?.countries || []
  const max = Math.max(1, ...rows.map((row) => row.accounts))
  return rows.map((row) => ({
    ...row,
    name: COUNTRIES.find((country) => country.code === row.country)?.name || row.country,
    width: (row.accounts / max) * 100,
  }))
})

const attentionRows = computed(() =>
  (summary.value?.attention || []).map((item) => {
    if (item.kind === 'proxy') {
      return {
        ...item,
        icon: 'ti-network',
        description: `${proxyStatusLabel(item.status)} · ${item.detail}`,
        to: { path: '/proxies', query: { status: item.status } },
      }
    }
    if (item.kind === 'gmail') {
      return {
        ...item,
        icon: 'ti-brand-gmail',
        description: `${gmailStatusLabel(item.status)} · ${item.detail}`,
        to: { path: '/gmails', query: { status: item.status } },
      }
    }
    return {
      ...item,
      icon: platformMeta(item.detail).icon,
      description: `${platformMeta(item.detail).label} · ${statusLabel(item.status)}`,
      to: { path: '/accounts', query: { status: item.status, platform: item.detail } },
    }
  }),
)

// Turns a status constant list plus a `{ active, error, … }` count object from the
// API into donut segments, so labels and colours stay in sync with the list pages.
function statusSegments(statuses, counts) {
  if (!counts) return []
  return statuses.map((status) => ({
    status: status.value,
    label: status.label,
    value: counts[status.value] || 0,
    color: STATUS_COLORS[status.value] || 'var(--text-faint)',
  }))
}

function share(value, total) {
  if (!total) return 0
  return Math.round((value / total) * 100)
}

function recentProxyLabel(account) {
  const mode = proxyModeLabel(account.proxyMode)
  if (account.proxyMode === 'none' || !account.proxyName) return mode
  const parts = [mode, account.proxyName]
  if (account.proxyCountry) parts.push(account.proxyCountry)
  return parts.join(' · ')
}

function formatDate(value) {
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

function formatDateTime(value) {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

// Background polls stay quiet on failure so a flaky API cannot bury the user in toasts.
async function load({ silent = false } = {}) {
  refreshing.value = true
  try {
    summary.value = await getDashboard()
    lastUpdated.value = Date.now()
    clock.value = lastUpdated.value
    loadError.value = ''
  } catch (err) {
    loadError.value = err.message || 'Could not load dashboard data'
    if (!silent) notifyError(loadError.value)
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

async function runProxyCheck() {
  checking.value = true
  try {
    const result = await checkAllProxies()
    await load({ silent: true })
    if (result.failed > 0) {
      notifyError(`Checked ${result.total} proxies: ${result.active} active, ${result.failed} failed.`)
    } else {
      notifySuccess(`Checked ${result.total} proxies: all active.`)
    }
  } catch (err) {
    notifyError(err.message || 'Could not check proxies')
  } finally {
    checking.value = false
  }
}

function startPolling() {
  stopPolling()
  if (!autoRefresh.value) return
  refreshTimer = window.setInterval(() => {
    if (document.hidden || refreshing.value || checking.value) return
    load({ silent: true })
  }, REFRESH_MS)
}

function stopPolling() {
  if (refreshTimer) window.clearInterval(refreshTimer)
  refreshTimer = null
}

// Polling is skipped while the tab is hidden, so catch up as soon as it returns.
function onVisibilityChange() {
  if (document.hidden || !autoRefresh.value) return
  if (Date.now() - lastUpdated.value >= REFRESH_MS) load({ silent: true })
}

watch(autoRefresh, startPolling)

onMounted(() => {
  load()
  startPolling()
  clockTimer = window.setInterval(() => {
    clock.value = Date.now()
  }, 1000)
  document.addEventListener('visibilitychange', onVisibilityChange)
})

onUnmounted(() => {
  stopPolling()
  if (clockTimer) window.clearInterval(clockTimer)
  document.removeEventListener('visibilitychange', onVisibilityChange)
})
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

.page-head__actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.page-head .ti {
  font-size: 16px;
}

.freshness {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-family: var(--mono);
  font-size: 11.5px;
  color: var(--text-faint);
}

.freshness.is-live {
  color: var(--viper-400);
}

.btn.is-on {
  border-color: var(--viper-500);
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

.banner {
  margin-bottom: 1rem;
  padding: 10px 12px;
  border: 0.5px solid var(--border-danger);
  border-radius: var(--radius);
  background: var(--bg-danger);
  color: var(--danger);
  font-size: 13px;
}

.stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 12px;
  margin-bottom: 14px;
}

.stat {
  display: block;
  background: var(--panel);
  border: 0.5px solid var(--hairline);
  border-radius: 10px;
  padding: 14px 16px;
  transition: border-color 0.12s ease, background 0.12s ease;
}

.stat:hover {
  border-color: var(--hairline-strong);
  background: var(--panel-raised);
}

.stat__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.stat__head .ti {
  font-size: 15px;
  color: var(--text-faint);
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
  font-size: 24px;
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

.stat__value.is-danger {
  color: var(--danger);
}

.stat__meta {
  margin-top: 2px;
  font-size: 12px;
  color: var(--text-faint);
}

.ops-strip {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  gap: 12px;
  margin-bottom: 14px;
}

.ops-card {
  display: block;
  padding: 14px 16px;
  border: 0.5px solid var(--hairline);
  border-radius: 10px;
  background: var(--panel);
  transition: border-color 0.12s ease, background 0.12s ease;
}

.ops-card:hover {
  border-color: var(--hairline-strong);
  background: var(--panel-raised);
}

.ops-card--static {
  cursor: default;
}

.ops-card__label {
  font-family: var(--mono);
  font-size: 11px;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--text-faint);
}

.ops-card__value {
  margin-top: 4px;
  font-family: var(--mono);
  font-size: 22px;
  font-weight: 600;
  color: var(--viper-400);
}

.ops-card__value.is-warn {
  color: var(--warn);
}

.ops-card__meta {
  margin-top: 2px;
  font-size: 12px;
  color: var(--text-faint);
}

.quick-actions {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.quick {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  padding: 12px;
  border: 0.5px solid var(--hairline);
  border-radius: 10px;
  background: var(--bg);
}

.quick:hover {
  border-color: var(--hairline-strong);
  background: var(--panel-raised);
}

.quick .ti {
  width: 30px;
  height: 30px;
  display: grid;
  place-items: center;
  border-radius: 8px;
  background: var(--viper-dim);
  color: var(--viper-400);
  font-size: 15px;
  flex-shrink: 0;
}

.quick strong {
  display: block;
  font-size: 13px;
}

.quick span {
  font-size: 12px;
  color: var(--text-faint);
}

.type-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
}

.mono-soft {
  font-family: var(--mono);
  font-size: 11.5px;
  word-break: break-all;
}

.grid {
  display: grid;
  grid-template-columns: repeat(12, 1fr);
  gap: 14px;
}

.col-8 {
  grid-column: span 8;
}

.col-6 {
  grid-column: span 6;
}

.col-4 {
  grid-column: span 4;
}

.panel {
  background: var(--panel);
  border: 0.5px solid var(--hairline);
  border-radius: 12px;
  padding: 14px 16px;
  min-width: 0;
}

.panel__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.panel__head h2 {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
}

.panel__head p {
  margin: 2px 0 0;
  font-size: 12px;
  color: var(--text-faint);
}

.panel__meta {
  display: flex;
  gap: 14px;
  font-size: 12px;
  color: var(--text-faint);
  white-space: nowrap;
}

.panel__meta strong {
  font-family: var(--mono);
  color: var(--text);
}

.panel__link {
  font-size: 12.5px;
  color: var(--viper-400);
  white-space: nowrap;
}

.panel__link:hover {
  text-decoration: underline;
}

.split {
  display: flex;
  align-items: center;
  gap: 18px;
}

.legend {
  list-style: none;
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.legend a {
  display: grid;
  grid-template-columns: auto 1fr auto auto;
  align-items: center;
  gap: 8px;
  padding: 5px 6px;
  border-radius: 6px;
  font-size: 13px;
}

.legend a:hover {
  background: var(--panel-raised);
}

.legend__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.legend__label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.legend__value {
  font-family: var(--mono);
  font-size: 12.5px;
}

.legend__pct {
  font-family: var(--mono);
  font-size: 11.5px;
  color: var(--text-faint);
  min-width: 34px;
  text-align: right;
}

.legend__note {
  padding: 5px 6px;
  font-size: 11.5px;
  color: var(--text-faint);
}

.bar-list {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.bar-list a {
  display: block;
  padding: 7px 6px;
  border-radius: 8px;
}

.bar-list a:hover {
  background: var(--panel-raised);
}

.bar-list__label {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 7px;
  font-size: 13px;
}

.bar-list__label .ti {
  font-size: 16px;
  color: var(--text-dim);
}

.bar-list__label strong {
  margin-left: auto;
  font-family: var(--mono);
  font-size: 12.5px;
  color: var(--viper-400);
}

.bar-list__track {
  height: 6px;
  border-radius: 999px;
  background: var(--panel-raised);
  overflow: hidden;
}

.bar-list__fill {
  height: 100%;
  border-radius: inherit;
  background: var(--hairline-strong);
}

.bar-list__active {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, var(--viper-700), var(--viper-400));
}

.bar-list__sub {
  margin-top: 6px;
  font-size: 11.5px;
  color: var(--text-faint);
}

.code {
  font-family: var(--mono);
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 5px;
  background: var(--panel-raised);
  color: var(--text-dim);
}

.pill {
  font-family: var(--mono);
  font-size: 10.5px;
  padding: 2px 7px;
  border-radius: 999px;
}

.pill--warn {
  background: var(--gold-dim);
  color: var(--gold-500);
}

.load-list {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 14px;
  padding-top: 14px;
  border-top: 0.5px solid var(--hairline);
}

.load-list a {
  display: block;
  padding: 9px 11px;
  border: 0.5px solid var(--hairline);
  border-radius: 10px;
  background: var(--bg);
}

.load-list a:hover {
  border-color: var(--hairline-strong);
}

.load-list__main,
.load-list__meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.load-list__name {
  font-size: 13.5px;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.load-list__meta {
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-faint);
}

.callouts {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  margin-top: 14px;
  padding-top: 14px;
  border-top: 0.5px solid var(--hairline);
}

.callout {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 10px 12px;
  border: 0.5px solid var(--hairline);
  border-radius: 10px;
  background: var(--bg);
}

.callout:hover {
  border-color: var(--hairline-strong);
}

.callout strong {
  font-family: var(--mono);
  font-size: 18px;
  color: var(--warn);
}

.callout.is-ok strong {
  color: var(--success);
}

.callout span {
  font-size: 11.5px;
  color: var(--text-faint);
  line-height: 1.35;
}

.table-wrap {
  overflow-x: auto;
}

table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13.5px;
}

th,
td {
  padding: 10px 8px;
  text-align: left;
  border-bottom: 0.5px solid var(--hairline);
  vertical-align: top;
}

th {
  font-weight: 500;
  color: var(--text-dim);
}

tbody tr:last-child td {
  border-bottom: none;
}

.account-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.account-cell .ti {
  font-size: 18px;
  color: var(--text-dim);
}

.account-cell__name {
  font-weight: 500;
}

.account-cell__sub,
.sub {
  font-size: 12px;
  color: var(--text-faint);
}

.stack {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.muted {
  color: var(--text-dim);
  font-size: 12.5px;
}

.status {
  display: inline-block;
  font-family: var(--mono);
  font-size: 11.5px;
  padding: 5px 11px;
  border-radius: 20px;
}

.status--sm {
  padding: 3px 8px;
  font-size: 11px;
}

.status--active {
  background: var(--viper-dim);
  color: var(--viper-400);
}

.status--expired,
.status--inactive {
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

.status--queued,
.status--pending {
  background: var(--panel-raised);
  color: var(--text-dim);
}

.status--running {
  background: var(--gold-dim);
  color: var(--gold-500);
}

.status--completed {
  background: var(--viper-dim);
  color: var(--viper-400);
}

.status--failed {
  background: var(--bg-danger);
  color: var(--danger);
}

.status--cancelled {
  background: var(--panel-raised);
  color: var(--text-faint);
}

.attention-list {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.attention-list li {
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: 10px;
  align-items: center;
  padding: 10px 12px;
  border: 0.5px solid var(--hairline);
  border-radius: 10px;
  background: var(--bg);
}

.attention__icon {
  width: 30px;
  height: 30px;
  border-radius: 8px;
  display: grid;
  place-items: center;
}

.attention__icon--danger {
  background: var(--bg-danger);
  color: var(--danger);
}

.attention__icon--warn {
  background: var(--gold-dim);
  color: var(--gold-500);
}

.attention__body {
  min-width: 0;
}

.attention__title {
  font-size: 13.5px;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.attention__desc {
  font-size: 12px;
  color: var(--text-faint);
}

.attention__link {
  font-size: 12.5px;
  color: var(--viper-400);
}

.attention__link:hover {
  text-decoration: underline;
}

.empty-panel,
.empty-inline {
  text-align: center;
  padding: 1.5rem 1rem;
  color: var(--text-dim);
  font-size: 14px;
}

.empty-inline {
  padding: 0.75rem 0;
  text-align: left;
}

.empty-inline.is-ok {
  color: var(--viper-400);
}

@media (max-width: 1180px) {
  .col-8,
  .col-4 {
    grid-column: span 12;
  }
}

@media (max-width: 860px) {
  .col-6 {
    grid-column: span 12;
  }

  .split {
    flex-direction: column;
    align-items: stretch;
  }

  .callouts {
    grid-template-columns: 1fr;
  }
}
</style>
