import { createRouter, createWebHistory } from 'vue-router'
import AppLayout from '../components/AppLayout.vue'
import AccountsPage from '../views/AccountsPage.vue'
import DashboardPage from '../views/DashboardPage.vue'
import GmailsPage from '../views/GmailsPage.vue'
import ProxiesPage from '../views/ProxiesPage.vue'
import PlaceholderPage from '../views/PlaceholderPage.vue'

const routes = [
  {
    path: '/',
    component: AppLayout,
    children: [
      { path: '', redirect: '/dashboard' },
      { path: 'dashboard', name: 'dashboard', component: DashboardPage },
      { path: 'accounts', name: 'accounts', component: AccountsPage },
      { path: 'proxies', name: 'proxies', component: ProxiesPage },
      { path: 'gmails', name: 'gmails', component: GmailsPage },
      {
        path: 'posts',
        component: PlaceholderPage,
        meta: { title: 'Posts', description: 'Scheduled and published posts for this organization.' },
      },
      {
        path: 'organization',
        component: PlaceholderPage,
        meta: { title: 'Organization', description: 'Organization profile and workspace details.' },
      },
      {
        path: 'members',
        component: PlaceholderPage,
        meta: { title: 'Members', description: 'People who can manage accounts in this workspace.' },
      },
      {
        path: 'activity',
        component: PlaceholderPage,
        meta: { title: 'Activity log', description: 'Recent account connections and admin actions.' },
      },
      {
        path: 'settings',
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

export default router
