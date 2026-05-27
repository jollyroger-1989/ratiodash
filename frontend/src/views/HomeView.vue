<template>
  <div class="home">
    <div class="hero">
      <h1>{{ $t('home.title') }}</h1>
      <p>{{ $t('home.subtitle') }}</p>
      <RouterLink to="/trackers" class="cta">{{ $t('home.cta') }}</RouterLink>
    </div>

    <div class="summary-grid">
      <div class="summary-card">
        <span class="summary-label">{{ $t('home.panels.sites') }}</span>
        <span class="summary-value">{{ siteCount }}</span>
      </div>
      <div class="summary-card">
        <span class="summary-label">{{ $t('home.panels.totalUpload') }}</span>
        <span class="summary-value">{{ formatBytes(totalUpload) }}</span>
      </div>
      <div class="summary-card">
        <span class="summary-label">{{ $t('home.panels.totalDownload') }}</span>
        <span class="summary-value">{{ formatBytes(totalDownload) }}</span>
      </div>
      <div class="summary-card">
        <span class="summary-label">{{ $t('home.panels.globalRatio') }}</span>
        <span class="summary-value" :class="ratioClass">
          {{ globalRatio !== null ? globalRatio.toFixed(2) : $t('home.panels.noData') }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { useTrackersStore } from '@/stores/trackers'
import { formatBytes } from '@/composables/formatBytes'
import { ratioClass as getRatioClass } from '@/composables/ratioClass'

const store = useTrackersStore()

onMounted(() => {
  if (!store.dashboard.length) store.fetchDashboard()
})

const siteCount = computed(() => store.dashboard.length)

const totalUpload = computed(() =>
  store.dashboard.reduce((sum, e) => sum + (e.stats?.uploaded ?? 0), 0)
)

const totalDownload = computed(() =>
  store.dashboard.reduce((sum, e) => sum + (e.stats?.downloaded ?? 0), 0)
)

const globalRatio = computed(() =>
  totalDownload.value > 0 ? totalUpload.value / totalDownload.value : null
)

const ratioClass = computed(() => {
  const r = globalRatio.value
  return r === null ? '' : getRatioClass(r)
})
</script>

<style scoped>
.home {
  max-width: 1100px;
  margin: 0 auto;
  text-align: center;
}

.hero {
  margin-bottom: 3rem;
}

h1 {
  font-size: clamp(1.6rem, 6vw, 2.5rem);
  margin-bottom: 1rem;
  background: linear-gradient(135deg, var(--accent), var(--accent-2));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.hero p {
  color: var(--text-muted);
  margin-bottom: 2rem;
}

.cta {
  display: inline-block;
  padding: 0.75rem 1.75rem;
  background: linear-gradient(135deg, #4f46e5, #7c3aed);
  color: white;
  text-decoration: none;
  border-radius: 8px;
  font-weight: 600;
  letter-spacing: 0.02em;
  transition: opacity 0.2s, box-shadow 0.2s;
  box-shadow: 0 0 20px rgba(129, 140, 248, 0.35);
}

.cta:hover {
  opacity: 0.9;
  box-shadow: 0 0 28px rgba(129, 140, 248, 0.55);
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1rem;
}

@media (max-width: 600px) {
  .summary-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

.summary-card {
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.25rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  box-shadow: var(--shadow), var(--shadow-glow);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  transition: border-color 0.2s, box-shadow 0.2s;
}

.summary-card:hover {
  border-color: var(--border-bright);
  box-shadow: var(--shadow), 0 0 28px rgba(129, 140, 248, 0.18);
}

.summary-label {
  font-size: 0.78rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-label);
}

.summary-value {
  font-size: 1.6rem;
  font-weight: 700;
  color: var(--text);
}
</style>
