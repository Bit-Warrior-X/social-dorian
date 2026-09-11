<template>
  <aside class="sidebar">
    <div class="brand">
      <DorianMark class="brand__mark" />
      <span class="brand__wordmark">DORIAN</span>
    </div>

    <nav class="nav" aria-label="Admin">
      <RouterLink
        to="/dashboard"
        class="navitem navitem--top"
        :class="{ active: route.path === '/dashboard' }"
      >
        <i class="ti ti-layout-dashboard" aria-hidden="true" />
        Dashboard
      </RouterLink>

      <section v-for="group in navGroups" :key="group.label" class="nav-group">
        <h2 class="nav-group__label">{{ group.label }}</h2>
        <RouterLink
          v-for="item in group.items"
          :key="item.to"
          :to="item.to"
          class="navitem"
          :class="{ active: isActive(item.to) }"
        >
          <i class="ti" :class="item.icon" aria-hidden="true" />
          {{ item.label }}
        </RouterLink>
      </section>
    </nav>

    <div class="sidebar__footer">
      <RouterLink to="/docs" class="navitem" :class="{ active: route.path === '/docs' }">
        <i class="ti ti-book" aria-hidden="true" />
        Docs
      </RouterLink>
      <RouterLink to="/settings" class="navitem" :class="{ active: route.path === '/settings' }">
        <i class="ti ti-settings" aria-hidden="true" />
        Settings
      </RouterLink>
      <div v-if="currentUser" class="sidebar__user">
        <span class="sidebar__user-name">{{ currentUser.name || 'Admin' }}</span>
        <span class="sidebar__user-email">{{ currentUser.email }}</span>
      </div>
      <a href="#" class="navitem" @click.prevent="signOut">
        <i class="ti ti-logout" aria-hidden="true" />
        Log out
      </a>
    </div>
  </aside>
</template>

<script setup>
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { currentUser, logout } from '../api/auth'
import DorianMark from './DorianMark.vue'

const route = useRoute()
const router = useRouter()

const navGroups = [
  {
    label: 'Infrastructure',
    items: [
      { to: '/accounts', icon: 'ti-share', label: 'Social accounts' },
      { to: '/proxies', icon: 'ti-network', label: 'Proxy management' },
      { to: '/gmails', icon: 'ti-brand-gmail', label: 'Email management' },
      { to: '/credits', icon: 'ti-coin', label: 'Credit balance' },
    ],
  },
  {
    label: 'Operations',
    items: [
      { to: '/tasks', icon: 'ti-list-check', label: 'Tasks / Campaigns' },
      { to: '/tasks/active', icon: 'ti-player-play', label: 'Active jobs' },
      { to: '/tasks/history', icon: 'ti-history', label: 'History' },
      { to: '/posts', icon: 'ti-article', label: 'Post management' },
      { to: '/monitor', icon: 'ti-activity', label: 'Activity' },
    ],
  },
]

function isActive(to) {
  if (to === '/tasks') return route.path === '/tasks'
  return route.path === to || route.path.startsWith(`${to}/`)
}

async function signOut() {
  await logout()
  await router.replace({ name: 'login' })
}
</script>

<style scoped>
.sidebar {
  width: 220px;
  flex-shrink: 0;
  background: var(--panel);
  padding: 1rem 0.75rem;
  display: flex;
  flex-direction: column;
  border-right: 0.5px solid var(--hairline);
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 6px 1rem;
  margin-bottom: 0.5rem;
  border-bottom: 0.5px solid var(--hairline);
}

.brand__mark {
  font-size: 28px;
}

.brand__wordmark {
  font-family: var(--mono);
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.nav {
  display: flex;
  flex-direction: column;
  gap: 14px;
  flex: 1;
  overflow: auto;
}

.nav-group {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.nav-group__label {
  margin: 0 0 4px;
  padding: 0 10px;
  font-family: var(--mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-faint);
}

.navitem {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: var(--radius);
  font-size: 13px;
  color: var(--text-dim);
}

.navitem--top {
  margin-bottom: -6px;
}

.navitem .ti {
  font-size: 16px;
}

.navitem:hover {
  background: var(--panel-raised);
  color: var(--text);
}

.navitem.active {
  background: var(--viper-dim);
  color: var(--viper-400);
  font-weight: 500;
}

.sidebar__footer {
  border-top: 0.5px solid var(--hairline);
  padding-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.sidebar__user {
  display: flex;
  flex-direction: column;
  gap: 1px;
  padding: 8px 10px 10px;
}

.sidebar__user-name {
  font-size: 12.5px;
  font-weight: 600;
  color: var(--text);
}

.sidebar__user-email {
  font-size: 11px;
  color: var(--text-faint);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 800px) {
  .sidebar {
    width: 250px;
    height: 100%;
  }
}
</style>
