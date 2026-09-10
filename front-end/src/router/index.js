import { createRouter, createWebHistory } from 'vue-router'
import { getCurrentUser } from '../api/auth'
import AppLayout from '../components/AppLayout.vue'
import AccountsPage from '../views/AccountsPage.vue'
import CreditsPage from '../views/CreditsPage.vue'
import DashboardPage from '../views/DashboardPage.vue'
import GmailsPage from '../views/GmailsPage.vue'
import LoginPage from '../views/LoginPage.vue'
import ProxiesPage from '../views/ProxiesPage.vue'
import TasksPage from '../views/TasksPage.vue'
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
      { path: '/credits', name: 'credits', component: CreditsPage },
      { path: '/tasks', name: 'tasks', component: TasksPage, props: { mode: 'all' } },
      {
        path: '/tasks/new',
        name: 'tasks-new',
        redirect: { path: '/tasks', query: { new: 'report' } },
      },
      { path: '/tasks/active', name: 'tasks-active', component: TasksPage, props: { mode: 'active' } },
      { path: '/tasks/history', name: 'tasks-history', component: TasksPage, props: { mode: 'history' } },
      {
        path: '/monitor',
        component: PlaceholderPage,
        meta: {
          title: 'Live feed',
          description: 'Log stream of browse and automation activity across accounts.',
        },
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
