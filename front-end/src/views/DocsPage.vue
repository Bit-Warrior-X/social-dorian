<template>
  <div class="docs">
    <header class="docs__hero">
      <div>
        <h1>Docs</h1>
        <p>
          Operator handbook for Social Dorian — infrastructure, campaigns, live browsers, post intelligence, and diagnosis.
        </p>
      </div>
      <div class="docs__pager-meta">
        <span>Page {{ pageIndex + 1 }} of {{ sections.length }}</span>
        <strong>{{ current.title }}</strong>
      </div>
    </header>

    <div class="docs__shell">
      <aside class="docs__nav" aria-label="Documentation pages">
        <p class="docs__nav-label">Sections</p>
        <button
          v-for="(section, index) in sections"
          :key="section.id"
          class="docs__nav-link"
          type="button"
          :class="{ active: pageIndex === index }"
          @click="goTo(index)"
        >
          <span class="docs__nav-num">{{ String(index + 1).padStart(2, '0') }}</span>
          <span class="docs__nav-text">
            <strong>{{ section.title }}</strong>
            <small>{{ section.blurb }}</small>
          </span>
        </button>
      </aside>

      <article class="docs__page">
        <div class="docs__page-head">
          <span class="docs__eyebrow">Section {{ pageIndex + 1 }} · {{ current.blurb }}</span>
          <h2>{{ current.title }}</h2>
        </div>

        <div class="docs__body" v-html="current.html" />

        <div class="docs__controls">
          <button class="btn" type="button" :disabled="pageIndex <= 0" @click="prev">
            <i class="ti ti-chevron-left" aria-hidden="true" />
            Previous
          </button>
          <div class="docs__dots" aria-hidden="true">
            <span
              v-for="(_, index) in sections"
              :key="index"
              class="docs__dot"
              :class="{ active: pageIndex === index }"
            />
          </div>
          <button class="btn btn-primary" type="button" :disabled="pageIndex >= sections.length - 1" @click="next">
            Next
            <i class="ti ti-chevron-right" aria-hidden="true" />
          </button>
        </div>
      </article>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

const sections = [
  {
    id: 'overview',
    title: 'Overview',
    blurb: 'What Dorian is for',
    html: `
      <p>Social Dorian is an operations console for teams that manage many social identities behind proxies. It combines inventory (accounts, proxies, mailboxes), remote browsers, campaign workers, post analysis, and a global activity feed.</p>
      <h3>What you can do</h3>
      <ul>
        <li>Store Facebook (and other) accounts with dedicated proxies and Gmail recovery mailboxes.</li>
        <li>Open a remote Chromium session through the account proxy to sign in and keep a warm profile.</li>
        <li>Launch campaigns: report, reply, publish, browse feed, or login-test across many accounts.</li>
        <li>Optionally watch the live browser while a job runs, with streaming logs beside the window.</li>
        <li>Track posts, generate reply drafts, and review what each account posted or replied.</li>
        <li>Diagnose failures in Activity with filters for source, severity, and search.</li>
      </ul>
      <h3>Recommended daily loop</h3>
      <ol>
        <li>Confirm proxies are healthy.</li>
        <li>Open any account that needs a fresh login / checkpoint.</li>
        <li>Launch a small test job (1 account, show browser on).</li>
        <li>Scale to the full account set once the path is clean.</li>
        <li>Review Activity + History; archive useful posts in Post management.</li>
      </ol>
      <p class="note"><strong>Important:</strong> automation workers are <strong>Facebook-only</strong> right now. You can still store Twitter / Instagram / etc. accounts for inventory, but task workers will fail with an explicit log until those platforms are implemented.</p>
    `,
  },
  {
    id: 'quick-start',
    title: 'Quick start',
    blurb: 'First successful campaign',
    html: `
      <p>Follow this checklist once after a fresh install (or when onboarding a new workspace).</p>
      <ol>
        <li><strong>Sign in</strong> with the admin account from your server <code>.env</code> (<code>ADMIN_EMAIL</code> / <code>ADMIN_PASSWORD</code>).</li>
        <li>Open <a href="/proxies">Proxy management</a> → <em>Add proxy</em>. Fill host, port, protocol, and auth if required. Click <em>Update status</em> until the proxy shows <em>active</em>.</li>
        <li>Open <a href="/gmails">Email management</a> → add the Gmail used for Facebook verification (app password preferred).</li>
        <li>Open <a href="/accounts">Social accounts</a> → <em>Connect account</em>. Set platform to Facebook, enter credentials, choose <em>Assigned proxy</em>, save.</li>
        <li>On the account row, click <em>Open</em>. Wait for the live window. Complete login / email code / checkpoint if prompted. Confirm the Session log shows success. Close the dialog.</li>
        <li>Open <a href="/credits">Credit balance</a> and confirm you have at least 1 credit per account you will select.</li>
        <li>Open <a href="/tasks">Tasks / Campaigns</a> → pick <em>Login test</em> or <em>Browse feed</em> → select that one account → enable <em>Show browser window</em> → launch.</li>
        <li>Watch <a href="/tasks/active">Active jobs</a> and the live dialog. When finished, confirm success in <a href="/tasks/history">History</a> and <a href="/monitor">Activity</a>.</li>
      </ol>
      <p class="note">If the first run fails, do not scale yet. Fix the proxy or login path with Show browser on until one account succeeds end-to-end.</p>
    `,
  },
  {
    id: 'proxies',
    title: 'Proxies',
    blurb: 'Network exits for accounts',
    html: `
      <p>Every automatable account should use a dedicated, healthy proxy. Dorian probes proxies and stores reachability status.</p>
      <h3>Fields that matter</h3>
      <ul>
        <li><strong>Protocol</strong> — usually HTTP/HTTPS or SOCKS5 depending on your provider.</li>
        <li><strong>Host / port</strong> — exit endpoint.</li>
        <li><strong>Username / password</strong> — if the provider requires auth.</li>
        <li><strong>Country</strong> — optional label for ops; does not change routing by itself.</li>
        <li><strong>Status</strong> — <em>active</em>, <em>inactive</em>, or <em>error</em> after checks.</li>
      </ul>
      <h3>How to verify</h3>
      <ol>
        <li>Click the check action on one proxy, or <em>Update status</em> for all.</li>
        <li>Successful checks log latency in <a href="/monitor">Activity</a> (source = proxy).</li>
        <li>Failed checks flip status to error and write the reason into Activity.</li>
      </ol>
      <h3>Tips</h3>
      <ul>
        <li>Prefer residential / mobile exits for Facebook when possible.</li>
        <li>One proxy per account (or small sticky pools) reduces linkability.</li>
        <li>If Open account fails with “proxy unreachable”, fix the proxy before touching credentials.</li>
      </ul>
    `,
  },
  {
    id: 'gmails',
    title: 'Email management',
    blurb: 'Verification inboxes',
    html: `
      <p>Gmail records are used when Facebook asks for an email confirmation code during Open account or worker login.</p>
      <h3>Setup</h3>
      <ul>
        <li>Add the mailbox email and password / app password.</li>
        <li>Store recovery email / phone if you want ops notes (not always used by automation).</li>
        <li>Use <em>Fetch PIN</em> on a row to pull the latest Facebook code from the inbox.</li>
      </ul>
      <h3>When codes fail</h3>
      <ul>
        <li>Confirm IMAP / app password is valid for that Gmail.</li>
        <li>Trigger a fresh Facebook email, then fetch PIN again.</li>
        <li>Watch the Session log in Open account — it reports “waiting for email code”, “no code”, or submit failures.</li>
      </ul>
      <p class="note">Match the Gmail to the email field on the social account whenever possible so operators know which inbox belongs to which identity.</p>
    `,
  },
  {
    id: 'accounts',
    title: 'Social accounts',
    blurb: 'Identities & live open',
    html: `
      <p><a href="/accounts">Social accounts</a> is the inventory of identities you can open remotely or attach to campaigns.</p>
      <h3>Connect checklist</h3>
      <ul>
        <li>Platform (use Facebook for automation).</li>
        <li>Email + password for the social login.</li>
        <li>Proxy mode = assigned proxy (required for Open / most jobs).</li>
        <li>Optional profile fields (name, birthday, gender) for your own bookkeeping.</li>
      </ul>
      <h3>Open account dialog</h3>
      <ul>
        <li>Starts Xvfb + Chromium + noVNC through the account proxy.</li>
        <li>Shows IP / proxy / credentials chips and a live iframe.</li>
        <li><strong>Session log</strong> (right panel) streams launch + auto-login steps and errors.</li>
        <li>Auto-login runs for Facebook; checkpoints can be finished manually in the window.</li>
        <li>Closing the dialog stops the remote session but keeps the saved Chromium profile.</li>
      </ul>
      <h3>Busy accounts</h3>
      <p>If an account is already in a queued/running task item, it shows as busy. Wait for completion or cancel the job before reusing it.</p>
    `,
  },
  {
    id: 'credits',
    title: 'Credits',
    blurb: 'Campaign budget',
    html: `
      <p>Campaigns debit workspace credits when launched.</p>
      <ul>
        <li><strong>Cost:</strong> 1 credit × number of selected accounts.</li>
        <li>If balance is too low, create/launch is blocked with an error.</li>
        <li>Check <a href="/credits">Credit balance</a> before large multi-account runs.</li>
      </ul>
      <p class="note">Credits are a workspace control plane limit — they do not replace Facebook rate limits or proxy health.</p>
    `,
  },
  {
    id: 'tasks',
    title: 'Tasks & campaigns',
    blurb: 'Launch & monitor jobs',
    html: `
      <p>Operations splits campaign views on purpose:</p>
      <ul>
        <li><a href="/tasks">Tasks / Campaigns</a> — create and launch; includes quick-launch cards.</li>
        <li><a href="/tasks/active">Active jobs</a> — queued / running only.</li>
        <li><a href="/tasks/history">History</a> — completed / failed / cancelled only.</li>
      </ul>
      <h3>Action types</h3>
      <div class="table-wrap">
        <table>
          <thead>
            <tr><th>Action</th><th>Needs</th><th>Notes</th></tr>
          </thead>
          <tbody>
            <tr><td>Report post</td><td>Post URL</td><td>Uses Facebook report flow</td></tr>
            <tr><td>Reply / comment</td><td>Post URL + comment</td><td>Best paired with Post management drafts</td></tr>
            <tr><td>New post</td><td>Post body (± media/link)</td><td>Media can be uploaded in the wizard</td></tr>
            <tr><td>Browse feed</td><td>Accounts</td><td>~90s human-like scroll / light likes</td></tr>
            <tr><td>Login test</td><td>Accounts</td><td>Validates saved profile / credentials</td></tr>
          </tbody>
        </table>
      </div>
      <h3>Launch options</h3>
      <ul>
        <li><strong>Show browser window</strong> — opens live remote desktop + task log for each account run. Great for debugging; slower.</li>
        <li><strong>Use account proxy</strong> — recommended on.</li>
        <li><strong>Delay min/max</strong> — random wait between accounts to look less robotic.</li>
      </ul>
      <h3>While a job runs</h3>
      <ol>
        <li>Stay on Active jobs (Chrome icon opens live view when available).</li>
        <li>Open task detail for filtered logs.</li>
        <li>Use Activity for cross-task errors and proxy/login events.</li>
        <li>Cancel from Active jobs if you need to stop early.</li>
      </ol>
    `,
  },
  {
    id: 'live-browser',
    title: 'Live browser & logs',
    blurb: 'Watch what the worker does',
    html: `
      <p>Two dialogs share the same idea: remote desktop on the left, operator log on the right.</p>
      <h3>Open account</h3>
      <ul>
        <li>Session log includes proxy exit IP, Chromium/VNC start, and Facebook login stages.</li>
        <li>Status polls about every 1.5s while the dialog is open.</li>
        <li>Failures before a session exists (bad proxy, missing Xvfb, etc.) still appear in Activity.</li>
      </ul>
      <h3>Show browser on tasks</h3>
      <ul>
        <li>Worker starts a watch display, logs <em>Watch live</em>, and the UI auto-opens the dialog.</li>
        <li>Task log streams Python worker lines (info / warn / error) including traceback blocks.</li>
        <li>Closing the dialog does <strong>not</strong> cancel the task — use Cancel on the job for that.</li>
      </ul>
      <p class="note">If the UI still says “Starting…” for a long time, check Activity for watch-session errors (missing x11vnc/websockify, display allocation, etc.).</p>
    `,
  },
  {
    id: 'posts',
    title: 'Post management',
    blurb: 'Analyze & reply intel',
    html: `
      <p><a href="/posts">Post management</a> helps you plan replies and audit account output.</p>
      <h3>Tracked posts</h3>
      <ol>
        <li>Click <em>Track post</em>.</li>
        <li>Paste the Facebook URL and as much of the original body as you can.</li>
        <li>Save — Dorian runs analysis and stores suggested replies.</li>
        <li>Open a row to copy a draft or <em>Launch reply task</em> with URL + text prefilled.</li>
        <li>Use the brain icon to re-analyze after you edit the stored body.</li>
      </ol>
      <h3>Account activity</h3>
      <p>Successful (and failed) task outcomes are recorded as engagements: post, reply, report, browse, login. Filter by account or action kind to see what each identity actually did.</p>
      <h3>Reply studio</h3>
      <p>Paste any post text without saving a tracked post. Generate analysis + 3 drafts, then launch a reply campaign immediately.</p>
      <p class="note">Default analysis is heuristic (sentiment, topics, risks, tone-based drafts). Set <code>OPENAI_API_KEY</code> (optional <code>OPENAI_MODEL</code>) on the API for richer AI analysis.</p>
    `,
  },
  {
    id: 'activity',
    title: 'Activity feed',
    blurb: 'Operator diagnosis',
    html: `
      <p><a href="/monitor">Activity</a> is the platform-wide log. Prefer it when something “just failed” and you are not sure which surface owns the error.</p>
      <h3>Filters</h3>
      <ul>
        <li><strong>Source</strong> — task, account, proxy, system.</li>
        <li><strong>Level</strong> — info, success, warn, error.</li>
        <li><strong>Problems only</strong> — warns + errors.</li>
        <li><strong>Search</strong> — message text, account email, proxy name, task title.</li>
      </ul>
      <h3>How to read a failure</h3>
      <ol>
        <li>Enable Problems only.</li>
        <li>Note the source and linked task/account/proxy.</li>
        <li>Open the task detail or Open account session if still available.</li>
        <li>Fix the root cause (proxy → login → content/URL → credits) before relaunching.</li>
      </ol>
      <h3>Common messages</h3>
      <ul>
        <li><em>Proxy check failed</em> — network/auth issue on the exit node.</li>
        <li><em>Open blocked — proxy unreachable</em> — same, before Chromium starts.</li>
        <li><em>Still on the login page</em> — bad credentials or challenge not completed.</li>
        <li><em>Worker interrupted / timed out</em> — script hung or API restarted mid-job.</li>
        <li>Python <em>Traceback</em> blocks — UI selector / Selenium failure; re-run with show browser.</li>
      </ul>
    `,
  },
  {
    id: 'settings',
    title: 'Settings & team',
    blurb: 'Users & access',
    html: `
      <p><a href="/settings">Settings</a> is admin territory for inviting teammates.</p>
      <ul>
        <li><strong>Admin</strong> — full access including user management.</li>
        <li><strong>Member</strong> — operate accounts, proxies, tasks, posts, activity.</li>
      </ul>
      <p>Disable or delete users you no longer trust. You cannot disable/delete your own admin row from the UI.</p>
      <p class="note">All API routes (except health/login) require a signed-in session cookie. Share passwords out-of-band and rotate them if a seat is compromised.</p>
    `,
  },
  {
    id: 'playbooks',
    title: 'Playbooks',
    blurb: 'Repeatable recipes',
    html: `
      <h3>Warm a new Facebook account</h3>
      <ol>
        <li>Assign a healthy proxy.</li>
        <li>Open account → complete login → confirm Session log success.</li>
        <li>Run Login test (1 account, show browser on).</li>
        <li>Run Browse feed once.</li>
        <li>Only then attach to reply/report/post campaigns.</li>
      </ol>
      <h3>Reply to a viral post safely</h3>
      <ol>
        <li>Track the post in Post management; paste full text.</li>
        <li>Pick a draft that matches the analyzed tone.</li>
        <li>Launch reply with 1–3 warmed accounts, delays on, show browser on for the first account.</li>
        <li>If the first succeeds headlessly for the rest.</li>
        <li>Confirm engagements under Account activity.</li>
      </ol>
      <h3>Bulk report</h3>
      <ol>
        <li>Verify the target URL opens as a normal Facebook post in Open account.</li>
        <li>Launch Report with a small pilot set.</li>
        <li>Scale only if Activity stays clean (no checkpoint storms).</li>
      </ol>
      <h3>After a mass failure</h3>
      <ol>
        <li>Activity → Problems only.</li>
        <li>If many proxy errors: pause campaigns, re-check proxies.</li>
        <li>If many login errors: open affected accounts manually.</li>
        <li>Relaunch with fewer accounts and show browser until stable.</li>
      </ol>
    `,
  },
  {
    id: 'tips',
    title: 'Best practices',
    blurb: 'Stability checklist',
    html: `
      <ul>
        <li>Never launch a large campaign on accounts that have never completed Open + Login test.</li>
        <li>Keep show browser on for the first attempt of any new action type or new proxy provider.</li>
        <li>Prefer short random delays between accounts; zero delay looks automated and breaks more often.</li>
        <li>Paste real post text into Post management — empty bodies produce weak reply drafts.</li>
        <li>Treat Activity as source of truth when the UI status and the browser disagree.</li>
        <li>Cancel stuck jobs instead of stacking more busy accounts on the same identities.</li>
        <li>One identity ↔ one sticky proxy whenever your provider allows it.</li>
        <li>Document unusual checkpoints in account notes (via your own process) so the next operator is not blind.</li>
      </ul>
      <p class="note">Dorian automates browser work; it does not bypass platform rules. Use warmed accounts, realistic pacing, and content that fits the page context.</p>
    `,
  },
]

const pageIndex = ref(0)
const current = computed(() => sections[pageIndex.value] || sections[0])

function goTo(index) {
  const nextIndex = Math.max(0, Math.min(sections.length - 1, index))
  pageIndex.value = nextIndex
  router.replace({ query: { ...route.query, page: sections[nextIndex].id } })
}

function prev() {
  goTo(pageIndex.value - 1)
}

function next() {
  goTo(pageIndex.value + 1)
}

watch(
  () => route.query.page,
  (value) => {
    const id = String(value || '')
    const index = sections.findIndex((section) => section.id === id)
    if (index >= 0) pageIndex.value = index
  },
  { immediate: true },
)
</script>

<style scoped>
.docs {
  --docs-height: calc(100vh - 40px);
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: var(--docs-height);
  max-height: var(--docs-height);
  min-height: var(--docs-height);
  overflow: hidden;
}

.docs__hero {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  flex-shrink: 0;
}

.docs__hero h1 {
  margin: 0;
  font-size: 24px;
}

.docs__hero p {
  margin: 6px 0 0;
  color: var(--text-dim);
  font-size: 14px;
  max-width: 58ch;
  line-height: 1.45;
}

.docs__pager-meta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  font-size: 12px;
  color: var(--text-faint);
}

.docs__pager-meta strong {
  color: var(--text);
  font-size: 13px;
}

.docs__shell {
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr);
  gap: 12px;
  flex: 1 1 auto;
  min-height: 0;
  height: 100%;
  overflow: hidden;
}

.docs__nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 12px;
  border: 0.5px solid var(--hairline);
  border-radius: 12px;
  background: var(--panel);
  overflow: auto;
  min-height: 0;
  height: 100%;
}

.docs__nav-label {
  margin: 0 0 8px;
  font-family: var(--mono);
  font-size: 10px;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--text-faint);
  flex-shrink: 0;
}

.docs__nav-link {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  width: 100%;
  text-align: left;
  border: 0;
  background: transparent;
  color: var(--text-dim);
  font: inherit;
  padding: 8px;
  border-radius: 8px;
  cursor: pointer;
}

.docs__nav-link:hover {
  background: var(--panel-raised);
  color: var(--text);
}

.docs__nav-link.active {
  background: var(--viper-dim);
  color: var(--viper-400);
}

.docs__nav-num {
  font-family: var(--mono);
  font-size: 11px;
  color: var(--text-faint);
  padding-top: 2px;
}

.docs__nav-text {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}

.docs__nav-text strong {
  font-size: 13px;
  font-weight: 600;
}

.docs__nav-text small {
  font-size: 11px;
  color: var(--text-faint);
  line-height: 1.3;
}

.docs__nav-link.active .docs__nav-text small {
  color: color-mix(in srgb, var(--viper-400) 75%, white);
}

.docs__page {
  display: flex;
  flex-direction: column;
  min-height: 0;
  height: 100%;
  border: 0.5px solid var(--hairline);
  border-radius: 12px;
  background: var(--panel);
  overflow: hidden;
}

.docs__page-head {
  padding: 14px 18px 10px;
  border-bottom: 0.5px solid var(--hairline);
  flex-shrink: 0;
}

.docs__eyebrow {
  font-family: var(--mono);
  font-size: 10px;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: var(--text-faint);
}

.docs__page-head h2 {
  margin: 4px 0 0;
  font-size: 20px;
}

.docs__body {
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
  padding: 16px 18px;
}

.docs__body :deep(p),
.docs__body :deep(li) {
  color: var(--text-dim);
  line-height: 1.55;
  font-size: 14px;
}

.docs__body :deep(p) {
  margin: 0 0 10px;
}

.docs__body :deep(h3) {
  margin: 0.95rem 0 8px;
  font-size: 15px;
  color: var(--text);
}

.docs__body :deep(ul),
.docs__body :deep(ol) {
  margin: 0 0 12px;
  padding-left: 1.25rem;
}

.docs__body :deep(li + li) {
  margin-top: 6px;
}

.docs__body :deep(a) {
  color: var(--viper-400);
  text-decoration: none;
  font-weight: 600;
}

.docs__body :deep(a:hover) {
  text-decoration: underline;
}

.docs__body :deep(code) {
  font-family: var(--mono);
  font-size: 12px;
  padding: 1px 5px;
  border-radius: 4px;
  background: var(--panel-raised);
  color: var(--text);
}

.docs__body :deep(.note) {
  padding: 10px 12px;
  border-radius: 10px;
  border: 0.5px solid var(--hairline);
  background: var(--bg);
}

.docs__body :deep(.table-wrap) {
  overflow-x: auto;
  border: 0.5px solid var(--hairline);
  border-radius: 10px;
  background: var(--bg);
  margin: 10px 0 12px;
}

.docs__body :deep(table) {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.docs__body :deep(th),
.docs__body :deep(td) {
  text-align: left;
  padding: 10px 12px;
  border-bottom: 0.5px solid var(--hairline);
  vertical-align: top;
  color: var(--text-dim);
}

.docs__body :deep(th) {
  color: var(--text-faint);
  font-weight: 500;
}

.docs__body :deep(tr:last-child td) {
  border-bottom: none;
}

.docs__controls {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 16px;
  border-top: 0.5px solid var(--hairline);
  background: var(--panel-raised);
  flex-shrink: 0;
}

.docs__controls .btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.docs__dots {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  justify-content: center;
}

.docs__dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: var(--hairline);
}

.docs__dot.active {
  background: var(--viper-400);
  width: 18px;
}

@media (max-width: 900px) {
  .docs {
    --docs-height: auto;
    height: auto;
    max-height: none;
    min-height: 0;
    overflow: visible;
  }

  .docs__shell {
    grid-template-columns: 1fr;
    height: auto;
    overflow: visible;
  }

  .docs__nav {
    height: auto;
    max-height: 180px;
  }

  .docs__page {
    height: min(70vh, 640px);
    min-height: 480px;
  }
}
</style>
