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

    <div class="chart-card">
      <div class="chart-toolbar">
        <h2>{{ $t('detail.evolution') }}</h2>
        <div class="period-tabs">
          <button
            v-for="p in periods"
            :key="p.label"
            :class="['period-btn', { active: activePeriod === p.label }]"
            @click="activePeriod = p.label"
          >{{ p.title }}</button>
        </div>
      </div>

      <p v-if="chartLoading" class="muted">{{ $t('common.loading') }}</p>
      <p v-else-if="chartError" class="error">{{ chartError }}</p>
      <p v-else-if="filteredHistory.length <= 1" class="muted">{{ $t('history.noHistory') }}</p>
      <Line v-else :data="chartData" :options="chartOptions" class="chart" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Line } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler,
  type ChartOptions
} from 'chart.js'
import { useTrackersStore } from '@/stores/trackers'
import { statsApi, type GlobalStatsPoint } from '@/services/api'
import { formatBytes } from '@/composables/formatBytes'
import { ratioClass as getRatioClass } from '@/composables/ratioClass'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler)

const { t, d } = useI18n()
const store = useTrackersStore()
const chartLoading = ref(false)
const chartError = ref('')
const history = ref<GlobalStatsPoint[]>([])

const periods = computed(() => [
  { label: '7d', days: 7, title: '7d' },
  { label: '30d', days: 30, title: '30d' },
  { label: '90d', days: 90, title: '90d' },
  { label: 'All', days: 0, title: t('detail.periods.all') },
])
const activePeriod = ref('30d')

onMounted(async () => {
  if (!store.dashboard.length) {
    await store.fetchDashboard()
  }
  await loadGlobalHistory()
})

const siteCount = computed(() => store.dashboard.length)

const latestGlobalPoint = computed(() => history.value[0] ?? null)

const totalUpload = computed(() =>
  latestGlobalPoint.value?.uploaded ??
  store.dashboard.reduce((sum, e) => sum + (e.stats?.uploaded ?? 0), 0)
)

const totalDownload = computed(() =>
  latestGlobalPoint.value?.downloaded ??
  store.dashboard.reduce((sum, e) => sum + (e.stats?.downloaded ?? 0), 0)
)

const globalRatio = computed(() =>
  latestGlobalPoint.value?.ratio ?? (totalDownload.value > 0 ? totalUpload.value / totalDownload.value : null)
)

const ratioClass = computed(() => {
  const r = globalRatio.value
  return r === null ? '' : getRatioClass(r)
})

const filteredHistory = computed(() => {
  const period = periods.value.find((p) => p.label === activePeriod.value)
  if (!period || !period.days) return history.value
  const cutoff = Date.now() - period.days * 86_400_000
  return history.value.filter((entry) => new Date(entry.fetched_at).getTime() >= cutoff)
})

const chartData = computed(() => {
  const rows = [...filteredHistory.value].reverse() // oldest first so the latest point is the rightmost one
  return {
    labels: rows.map((entry) => d(new Date(entry.fetched_at), 'short')),
    datasets: [
      {
        label: t('chart.ratio'),
        data: rows.map((entry) => entry.ratio),
        borderColor: '#818cf8',
        backgroundColor: 'rgba(129,140,248,0.1)',
        fill: true,
        tension: 0.3,
        pointRadius: rows.length > 60 ? 0 : 3,
        yAxisID: 'yRatio',
      },
      {
        label: t('chart.uploadedGib'),
        data: rows.map((entry) => +(entry.uploaded / 1073741824).toFixed(2)),
        borderColor: '#4ade80',
        backgroundColor: 'transparent',
        tension: 0.3,
        pointRadius: rows.length > 60 ? 0 : 3,
        yAxisID: 'yBytes',
      },
      {
        label: t('chart.downloadedGib'),
        data: rows.map((entry) => +(entry.downloaded / 1073741824).toFixed(2)),
        borderColor: '#fbbf24',
        backgroundColor: 'transparent',
        tension: 0.3,
        pointRadius: rows.length > 60 ? 0 : 3,
        yAxisID: 'yBytes',
      },
    ],
  }
})

const chartOptions = computed<ChartOptions<'line'>>(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: { mode: 'index', intersect: false },
  plugins: {
    legend: { position: 'top' },
    tooltip: {
      callbacks: {
        label(ctx) {
          if (ctx.dataset.yAxisID === 'yBytes') {
            return `${ctx.dataset.label}: ${(ctx.parsed.y ?? 0).toFixed(2)} ${t('chart.gib')}`
          }
          return `${ctx.dataset.label}: ${(ctx.parsed.y ?? 0).toFixed(2)}`
        }
      }
    }
  },
  scales: {
    yRatio: {
      type: 'linear',
      position: 'left',
      title: { display: true, text: t('chart.ratio') },
      ticks: { callback: (value) => Number(value).toFixed(2) },
    },
    yBytes: {
      type: 'linear',
      position: 'right',
      title: { display: true, text: t('chart.gib') },
      grid: { drawOnChartArea: false },
    },
  },
}))

async function loadGlobalHistory() {
  chartLoading.value = true
  chartError.value = ''
  try {
    history.value = await statsApi.getGlobalHistory(180)
  } catch {
    chartError.value = t('history.errorLoad')
  } finally {
    chartLoading.value = false
  }
}
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

.chart-card {
  margin-top: 1.5rem;
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.5rem;
  box-shadow: var(--shadow), var(--shadow-glow);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  transition: border-color 0.2s;
  text-align: left;
}

.chart-card:hover {
  border-color: var(--border-bright);
}

.chart-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
  gap: 1rem;
}

.chart-toolbar h2 {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-muted);
}

.period-tabs {
  display: flex;
  gap: 0.25rem;
}

.period-btn {
  background: none;
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 0.2rem 0.6rem;
  font-size: 0.78rem;
  cursor: pointer;
  color: var(--text-muted);
  transition: border-color 0.2s, color 0.2s, background 0.2s;
}

.period-btn:hover {
  border-color: var(--border-bright);
  color: var(--accent);
}

.period-btn.active {
  background: linear-gradient(135deg, #4f46e5, #7c3aed);
  border-color: transparent;
  color: white;
}

.chart {
  height: 320px !important;
}

.error {
  color: var(--ratio-bad);
}

.muted {
  color: var(--text-muted);
}

@media (max-width: 640px) {
  .chart {
    height: 220px !important;
  }

  .chart-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .period-tabs {
    justify-content: space-between;
  }
}
</style>
