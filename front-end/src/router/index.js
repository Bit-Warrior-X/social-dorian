import { createRouter, createWebHistory } from 'vue-router'
import { getCurrentUser } from '../api/auth'
import AppLayout from '../components/AppLayout.vue'
import AccountsPage from '../views/AccountsPage.vue'
import DashboardPage from '../views/DashboardPage.vue'
import GmailsPage from '../views/GmailsPage.vue'
import LoginPage from '../views/LoginPage.vue'
import ProxiesPage from '../views/ProxiesPage.vue'
import PlaceholderPage from '../views/PlaceholderPage.vue'

const routes = [
  {
    path: '/',
    name: 'login',
    component: LoginPage,
    meta: { public: true },
  },
  {
    path: '/login',
    redirect: '/',
  },
  {
    component: AppLayout,
    children: [
      { path: '/dashboard', name: 'dashboard', component: DashboardPage },
      { path: '/accounts', name: 'accounts', component: AccountsPage },
      { path: '/proxies', name: 'proxies', component: ProxiesPage },
      { path: '/gmails', name: 'gmails', component: GmailsPage },
      {
        path: '/posts',
        component: PlaceholderPage,
        meta: { title: 'Posts', description: 'Scheduled and published posts for this organization.' },
      },
      {
        path: '/organization',
        component: PlaceholderPage,
        meta: { title: 'Organization', description: 'Organization profile and workspace details.' },
      },
      {
        path: '/members',
        component: PlaceholderPage,
        meta: { title: 'Members', description: 'People who can manage accounts in this workspace.' },
      },
      {
        path: '/activity',
        component: PlaceholderPage,
        meta: { title: 'Activity log', description: 'Recent account connections and admin actions.' },
      },
      {
        path: '/settings',
        component: PlaceholderPage,
        meta: { title: 'Settings', description: 'Workspace preferences and notifications.' },
      },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to) => {
  const user = await getCurrentUser()
  if (to.meta.public) {
    if (user && to.name === 'login') {
      const next = String(to.query.next || '').trim()
      if (next.startsWith('/') && !next.startsWith('//') && next !== '/' && !next.startsWith('/login')) {
        return next
      }
      return { name: 'dashboard' }
    }
    return true
  }
  if (!user) {
    return { name: 'login', query: { next: to.fullPath } }
  }
  return true
})

export default router
