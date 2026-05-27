<template>
  <StarField />
  <div v-if="!isAuthPage" class="app-layout">
    <button class="hamburger" @click="sidebarOpen = true" aria-label="Open menu">
      <font-awesome-icon :icon="['fas', 'bars']" />
    </button>

    <div v-show="sidebarOpen" class="sidebar-backdrop" @click="sidebarOpen = false" />

    <aside :class="['app-sidebar', { open: sidebarOpen }]">
      <RouterLink to="/" class="sidebar-logo" data-tip="RatioDash">
        <img src="/favicon.svg" width="32" height="32" alt="RatioDash" />
      </RouterLink>

      <nav class="sidebar-nav">
        <RouterLink to="/" class="sidebar-link" :data-tip="$t('nav.home')">
          <font-awesome-icon :icon="['fas', 'house']" />
        </RouterLink>
        <RouterLink to="/trackers" class="sidebar-link" :data-tip="$t('nav.trackers')">
          <font-awesome-icon :icon="['fas', 'chart-bar']" />
        </RouterLink>
        <RouterLink to="/reports" class="sidebar-link" :data-tip="$t('nav.reports')">
          <font-awesome-icon :icon="['fas', 'envelope']" />
        </RouterLink>
        <RouterLink to="/alerts" class="sidebar-link" :data-tip="$t('nav.alerts')">
          <font-awesome-icon :icon="['fas', 'bell']" />
        </RouterLink>
      </nav>

      <div class="sidebar-footer">
        <button
          v-for="lang in availableLocales"
          :key="lang"
          :class="['sidebar-lang', { active: locale === lang }]"
          @click="setLocale(lang)"
        >{{ lang.toUpperCase() }}</button>

        <RouterLink to="/settings" class="sidebar-link" :data-tip="$t('nav.settings')">
          <font-awesome-icon :icon="['fas', 'gear']" />
        </RouterLink>

        <button
          v-if="authStore.isAuthenticated"
          class="sidebar-link sidebar-link--logout"
          :data-tip="$t('auth.logout')"
          @click="logout"
        >
          <font-awesome-icon :icon="['fas', 'right-from-bracket']" />
        </button>
      </div>
    </aside>

    <main class="app-main">
      <RouterView />
    </main>
  </div>

  <main v-else class="app-main app-main--full">
    <RouterView />
  </main>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import StarField from '@/components/StarField.vue'

const { locale } = useI18n()
const availableLocales = ['en', 'fr']
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const sidebarOpen = ref(false)

watch(() => route.path, () => {
  sidebarOpen.value = false
})

const isAuthPage = computed(() => route.name === 'login' || route.name === 'setup')

function setLocale(lang: string) {
  locale.value = lang
  localStorage.setItem('lang', lang)
}

function logout() {
  authStore.logout()
  router.push('/login')
}
</script>

<style scoped>
/* ── Layout ─────────────────────────────────────────── */
.app-layout {
  display: flex;
  min-height: 100vh;
}

/* ── Sidebar ─────────────────────────────────────────── */
.app-sidebar {
  width: 60px;
  flex-shrink: 0;
  background: rgba(5, 8, 24, 0.85);
  backdrop-filter: blur(14px);
  -webkit-backdrop-filter: blur(14px);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 1rem 0;
  position: sticky;
  top: 0;
  height: 100vh;
  z-index: 50;
}

/* ── Logo ─────────────────────────────────────────────── */
.sidebar-logo {
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  padding: 4px;
  margin-bottom: 1.25rem;
  transition: opacity 0.2s;
}

.sidebar-logo:hover {
  opacity: 0.75;
}

/* ── Nav ──────────────────────────────────────────────── */
.sidebar-nav {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  flex: 1;
}

/* ── Shared icon button ───────────────────────────────── */
.sidebar-link {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: 8px;
  color: var(--text-muted);
  text-decoration: none;
  background: transparent;
  border: none;
  cursor: pointer;
  transition: background 0.2s, color 0.2s;
  touch-action: manipulation;
}

.sidebar-link:hover,
.sidebar-link.router-link-exact-active {
  background: var(--accent-glow);
  color: var(--accent);
}

.sidebar-link--logout:hover {
  background: rgba(239, 68, 68, 0.08);
  color: #ef4444;
}

/* ── Tooltip ──────────────────────────────────────────── */
.app-sidebar [data-tip] {
  position: relative;
}

.app-sidebar [data-tip]::after {
  content: attr(data-tip);
  position: absolute;
  left: calc(100% + 10px);
  top: 50%;
  transform: translateY(-50%);
  background: rgba(10, 16, 42, 0.97);
  border: 1px solid var(--border-bright);
  color: var(--text);
  padding: 0.25rem 0.6rem;
  border-radius: 5px;
  font-size: 0.75rem;
  font-weight: 500;
  white-space: nowrap;
  pointer-events: none;
  opacity: 0;
  transition: opacity 0.15s;
  z-index: 100;
}

.app-sidebar [data-tip]:hover::after {
  opacity: 1;
}

/* ── Footer ───────────────────────────────────────────── */
.sidebar-footer {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.4rem;
  margin-top: auto;
}

.sidebar-lang {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 32px;
  border-radius: 5px;
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-muted);
  font-size: 0.68rem;
  font-weight: 700;
  cursor: pointer;
  transition: background 0.2s, border-color 0.2s, color 0.2s;
}

.sidebar-lang:hover,
.sidebar-lang.active {
  background: var(--accent-glow);
  border-color: var(--border-bright);
  color: var(--accent);
}

/* ── Main content ─────────────────────────────────────── */
.app-main {
  flex: 1;
  padding: 2rem;
  min-width: 0;
}

.app-main--full {
  padding: 0;
  min-height: 100vh;
}

/* ── Hamburger (mobile only) ────────────────────── */
.hamburger {
  display: none;
  position: fixed;
  top: 0.75rem;
  left: 0.75rem;
  z-index: 150;
  background: rgba(5, 8, 24, 0.9);
  border: 1px solid var(--border);
  border-radius: 8px;
  width: 40px;
  height: 40px;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
  cursor: pointer;
  transition: color 0.2s, border-color 0.2s;
  touch-action: manipulation;
}
.hamburger:hover { color: var(--accent); border-color: var(--border-bright); }

/* ── Sidebar backdrop (mobile) ──────────────────── */
.sidebar-backdrop {
  display: none;
  position: fixed;
  inset: 0;
  background: rgba(2, 8, 23, 0.6);
  z-index: 150;
  cursor: pointer;
}

/* ── Responsive sidebar ────────────────────────── */
@media (max-width: 767px) {
  .hamburger {
    display: flex;
  }

  .sidebar-backdrop {
    display: block;
  }

  .app-sidebar {
    position: fixed;
    top: 0;
    left: 0;
    height: 100%;
    transform: translateX(-100%);
    transition: transform 0.25s ease;
    z-index: 200;
  }

  .app-sidebar.open {
    transform: translateX(0);
  }

  .app-main {
    padding: 1rem;
    padding-top: 3.75rem;
  }
}
</style>
