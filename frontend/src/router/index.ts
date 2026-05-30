import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '@/views/HomeView.vue'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true }
    },
    {
      path: '/setup',
      name: 'setup',
      component: () => import('@/views/SetupView.vue'),
      meta: { public: true, setupOnly: true }
    },
    {
      path: '/',
      name: 'home',
      component: HomeView
    },
    {
      path: '/trackers',
      name: 'trackers',
      component: () => import('@/views/TrackersView.vue')
    },
    {
      path: '/trackers/:id',
      name: 'tracker-detail',
      component: () => import('@/views/TrackerDetailView.vue')
    },
    {
      path: '/settings',
      name: 'settings',
      component: () => import('@/views/SettingsView.vue')
    },
    {
      path: '/reports',
      name: 'reports',
      component: () => import('@/views/ReportsView.vue')
    },
    {
      path: '/alerts',
      name: 'alerts',
      component: () => import('@/views/AlertsView.vue')
    }
  ]
})

// Navigation guard: enforce auth and setup flow.
router.beforeEach(async (to) => {
  // Pinia stores are not available until after app.use(pinia),
  // so we access it lazily here.
  const authStore = useAuthStore()

  // Check if credentials have been configured.
  let setup: boolean
  try {
    setup = await authStore.isSetup()
  } catch {
    // If the status check fails (e.g. server down), allow navigation.
    return true
  }

  // If no credentials exist, force the setup wizard (only /setup is allowed).
  if (!setup) {
    if (to.name !== 'setup') return { name: 'setup' }
    return true
  }

  // Credentials exist — block access to the setup page.
  if (to.meta.setupOnly) return { name: 'home' }

  // Public routes (login) are always accessible.
  if (to.meta.public) return true

  // Protected routes require a valid token.
  if (!authStore.isAuthenticated) return { name: 'login' }

  return true
})

export default router
