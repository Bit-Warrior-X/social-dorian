<template>
  <div>
    <div class="page-head">
      <div>
        <h1>Settings</h1>
        <p>Workspace preferences and team access</p>
      </div>
    </div>

    <div class="grid">
      <section class="card">
        <h2>Signed-in user</h2>
        <dl>
          <div>
            <dt>Name</dt>
            <dd>{{ user?.name || '—' }}</dd>
          </div>
          <div>
            <dt>Email</dt>
            <dd>{{ user?.email || '—' }}</dd>
          </div>
          <div>
            <dt>Role</dt>
            <dd>{{ user?.role || 'admin' }}</dd>
          </div>
        </dl>
      </section>

      <section class="card">
        <h2>Credits</h2>
        <p class="lead">{{ loadingCredits ? '…' : balance.toLocaleString() }} available</p>
        <p class="note">
          Launching a task debits {{ costPerAccount }} credit per selected account.
        </p>
        <RouterLink class="btn" to="/credits">Open credit balance</RouterLink>
      </section>

      <section class="card">
        <h2>Automation</h2>
        <ul>
          <li>Task workers run Facebook scripts for report, reply, post, browse, and login test.</li>
          <li>Account proxies are used when “Use each account’s assigned proxy” is enabled.</li>
          <li>Media uploads are stored under <code>/var/lib/dorian-browser/uploads</code>.</li>
        </ul>
      </section>

      <section class="card">
        <h2>Notifications</h2>
        <p class="note">
          Email and webhook notifications are not configured yet. Watch Active jobs and Live feed for status.
        </p>
      </section>
    </div>

    <section v-if="isAdmin" class="team">
      <div class="team__head">
        <div>
          <h2>Team members</h2>
          <p>Invite members who can sign in and run operations. Only admins manage this list.</p>
        </div>
      </div>

      <p v-if="teamError" class="banner">{{ teamError }}</p>

      <form class="invite" @submit.prevent="onCreate">
        <input v-model.trim="invite.name" type="text" placeholder="Name" />
        <input v-model.trim="invite.email" type="email" placeholder="Email" required />
        <input v-model="invite.password" type="password" placeholder="Password (8+ chars)" required minlength="8" />
        <select v-model="invite.role">
          <option value="member">Member</option>
          <option value="admin">Admin</option>
        </select>
        <button class="btn btn-primary" type="submit" :disabled="savingUser">
          {{ savingUser ? 'Adding…' : 'Add user' }}
        </button>
      </form>

      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>User</th>
              <th>Role</th>
              <th>Status</th>
              <th class="col-actions">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in pageItems" :key="row.id">
              <td>
                <strong>{{ row.name || row.email }}</strong>
                <div class="sub">{{ row.email }}</div>
              </td>
              <td>
                <select
                  :value="row.role"
                  :disabled="row.id === user?.id || savingUser"
                  @change="onRole(row, $event.target.value)"
                >
                  <option value="admin">Admin</option>
                  <option value="member">Member</option>
                </select>
              </td>
              <td>
                <span class="status" :class="'status--' + row.status">{{ row.status }}</span>
              </td>
              <td class="actions">
                <button
                  class="btn btn-icon"
                  type="button"
                  :disabled="row.id === user?.id || savingUser"
                  :aria-label="row.status === 'active' ? 'Disable' : 'Enable'"
                  @click="onToggle(row)"
                >
                  <i class="ti" :class="row.status === 'active' ? 'ti-user-off' : 'ti-user-check'" aria-hidden="true" />
                </button>
                <button
                  class="btn btn-icon"
                  type="button"
                  :disabled="row.id === user?.id || savingUser"
                  aria-label="Delete"
                  @click="onDelete(row)"
                >
                  <i class="ti ti-trash" aria-hidden="true" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-if="!loadingUsers && users.length === 0" class="empty">No users yet.</p>
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
    </section>

    <p v-else class="note team-note">Ask an admin if you need another teammate invited.</p>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { RouterLink } from 'vue-router'
import PaginationBar from '../components/PaginationBar.vue'
import { getCredits } from '../api/tasks'
import { currentUser, getCurrentUser } from '../api/auth'
import { createUser, deleteUser, listUsers, updateUser } from '../api/users'
import { useNotify } from '../composables/useNotify'
import { usePagination } from '../composables/usePagination'

const { notifySuccess, notifyError } = useNotify()
const user = currentUser
const isAdmin = computed(() => user.value?.role === 'admin')
const balance = ref(0)
const costPerAccount = ref(1)
const loadingCredits = ref(true)
const users = ref([])
const loadingUsers = ref(false)
const {
  page,
  pageSize,
  pageItems,
  total,
  totalPages,
  from,
  to,
  pageSizeOptions,
} = usePagination(users)
const savingUser = ref(false)
const teamError = ref('')
const invite = reactive({
  name: '',
  email: '',
  password: '',
  role: 'member',
})

async function loadUsers() {
  if (!isAdmin.value) return
  loadingUsers.value = true
  teamError.value = ''
  try {
    users.value = await listUsers()
  } catch (err) {
    teamError.value = err.message || 'Could not load users'
  } finally {
    loadingUsers.value = false
  }
}

async function onCreate() {
  savingUser.value = true
  teamError.value = ''
  try {
    await createUser({ ...invite })
    notifySuccess('User added')
    invite.name = ''
    invite.email = ''
    invite.password = ''
    invite.role = 'member'
    await loadUsers()
  } catch (err) {
    teamError.value = err.message || 'Could not add user'
    notifyError(teamError.value)
  } finally {
    savingUser.value = false
  }
}

async function onRole(row, role) {
  savingUser.value = true
  try {
    await updateUser(row.id, { role })
    notifySuccess('Role updated')
    await loadUsers()
  } catch (err) {
    notifyError(err.message || 'Could not update role')
  } finally {
    savingUser.value = false
  }
}

async function onToggle(row) {
  savingUser.value = true
  try {
    await updateUser(row.id, { status: row.status === 'active' ? 'disabled' : 'active' })
    notifySuccess(row.status === 'active' ? 'User disabled' : 'User enabled')
    await loadUsers()
  } catch (err) {
    notifyError(err.message || 'Could not update user')
  } finally {
    savingUser.value = false
  }
}

async function onDelete(row) {
  if (!confirm(`Delete ${row.email}?`)) return
  savingUser.value = true
  try {
    await deleteUser(row.id)
    notifySuccess('User deleted')
    await loadUsers()
  } catch (err) {
    notifyError(err.message || 'Could not delete user')
  } finally {
    savingUser.value = false
  }
}

onMounted(async () => {
  await getCurrentUser()
  try {
    const data = await getCredits()
    balance.value = Number(data.balance || 0)
    costPerAccount.value = Number(data.costPerAccount || 1)
  } catch (_) {
    /* ignore */
  } finally {
    loadingCredits.value = false
  }
  await loadUsers()
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

.grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.card,
.team {
  padding: 16px;
  border: 0.5px solid var(--hairline);
  border-radius: 12px;
  background: var(--panel);
}

.card h2,
.team h2 {
  margin: 0 0 10px;
  font-size: 14px;
  font-weight: 600;
}

.team {
  margin-top: 12px;
}

.team__head p,
.note,
.team-note {
  margin: 0 0 12px;
  font-size: 13px;
  color: var(--text-dim);
  line-height: 1.4;
}

.team-note {
  margin-top: 12px;
}

dl {
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

dl div {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

dt {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--text-faint);
  font-family: var(--mono);
}

dd {
  margin: 0;
  font-size: 14px;
  color: var(--text);
}

.lead {
  margin: 0 0 8px;
  font-size: 28px;
  font-weight: 600;
  color: var(--text);
}

ul {
  margin: 0;
  padding-left: 1.1rem;
  font-size: 13px;
  color: var(--text-dim);
  line-height: 1.45;
}

li + li {
  margin-top: 6px;
}

code {
  font-family: var(--mono);
  font-size: 12px;
}

.invite {
  display: grid;
  grid-template-columns: 1fr 1.2fr 1fr 120px auto;
  gap: 8px;
  margin-bottom: 12px;
}

.banner {
  margin-bottom: 10px;
  padding: 8px 12px;
  border-radius: var(--radius);
  border: 0.5px solid color-mix(in srgb, var(--danger) 35%, transparent);
  font-size: 13px;
  color: var(--danger);
  background: var(--bg-danger);
}

.table-wrap {
  overflow: auto;
}

.table-wrap + :deep(.pager) {
  margin-top: 0;
  border: 0.5px solid var(--hairline);
  border-top: none;
  border-radius: 0 0 10px 10px;
}

table {
  width: 100%;
  border-collapse: collapse;
}

th,
td {
  text-align: left;
  padding: 10px 8px;
  border-bottom: 0.5px solid var(--hairline);
  font-size: 13px;
  vertical-align: middle;
}

th {
  color: var(--text-faint);
  font-weight: 500;
}

.sub {
  color: var(--text-faint);
  font-size: 12px;
}

.col-actions,
.actions {
  width: 96px;
  white-space: nowrap;
}

.status {
  display: inline-block;
  font-family: var(--mono);
  font-size: 11px;
  padding: 4px 8px;
  border-radius: 999px;
  text-transform: lowercase;
}

.status--active {
  background: var(--viper-dim);
  color: var(--viper-400);
}

.status--disabled {
  background: var(--panel-raised);
  color: var(--text-faint);
}

.empty {
  margin: 0;
  padding: 1rem;
  text-align: center;
  color: var(--text-dim);
}

@media (max-width: 900px) {
  .grid,
  .invite {
    grid-template-columns: 1fr;
  }
}
</style>
