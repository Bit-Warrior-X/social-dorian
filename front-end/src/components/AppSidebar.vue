<template>
  <aside class="sidebar">
    <div class="brand">
      <DorianMark class="brand__mark" />
      <span class="brand__wordmark">DORIAN</span>
    </div>

    <nav class="nav" aria-label="Admin">
      <RouterLink
        v-for="item in navItems"
        :key="item.to"
        :to="item.to"
        class="navitem"
        :class="{ active: route.path === item.to }"
      >
        <i class="ti" :class="item.icon" aria-hidden="true" />
        {{ item.label }}
      </RouterLink>
    </nav>

    <div class="sidebar__footer">
      <RouterLink to="/settings" class="navitem" :class="{ active: route.path === '/settings' }">
        <i class="ti ti-settings" aria-hidden="true" />
        Settings
      </RouterLink>
      <a href="#" class="navitem" @click.prevent>
        <i class="ti ti-logout" aria-hidden="true" />
        Log out
      </a>
    </div>
  </aside>
</template>

<script setup>
import { RouterLink, useRoute } from 'vue-router'
import DorianMark from './DorianMark.vue'

const route = useRoute()

const navItems = [
  { to: '/dashboard', icon: 'ti-layout-dashboard', label: 'Dashboard' },
  { to: '/accounts', icon: 'ti-share', label: 'Social accounts' },
  { to: '/proxies', icon: 'ti-network', label: 'Proxy management' },
  { to: '/gmails', icon: 'ti-brand-gmail', label: 'Gmail management' },
  { to: '/posts', icon: 'ti-calendar', label: 'Posts' },
  { to: '/organization', icon: 'ti-building', label: 'Organization' },
  { to: '/members', icon: 'ti-users', label: 'Members' },
  { to: '/activity', icon: 'ti-history', label: 'Activity log' },
]
</script>

<style scoped>
.sidebar {
  width: 180px;
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
  gap: 2px;
  flex: 1;
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

@media (max-width: 800px) {
  .sidebar {
    width: 220px;
    height: 100%;
  }
}
</style>
