<template>
  <div class="detail-view">
    <div class="detail-header">
      <RouterLink to="/trackers" class="back-link">{{ $t('detail.backLink') }}</RouterLink>
      <div v-if="tracker" class="tracker-title-row">
        <div class="tracker-title">
          <h1>{{ tracker.name }}</h1>
        </div>
        <div class="header-actions">
          <button class="btn-detail-action btn-detail-refresh" @click="doRefresh" :disabled="refreshing">
            <font-awesome-icon :icon="['fas', 'rotate-right']" /> {{ refreshing ? $t('common.loading') : $t('buttons.refresh') }}
          </button>
          <button class="btn-detail-action btn-detail-edit" @click="showForm = true">
            <font-awesome-icon :icon="['fas', 'pen']" /> {{ $t('buttons.editSite') }}
          </button>
        </div>
      </div>
    </div>

    <TrackerFormModal v-model="showForm" :tracker="tracker ?? undefined" @saved="onSaved" />

    <p v-if="loading" class="muted">{{ $t('common.loading') }}</p>
    <p v-else-if="error" class="error">{{ error }}</p>

    <template v-else-if="tracker">
      <!-- Latest snapshot -->
      <StatsGrid v-if="latest" :stats="latest" :show-date="true" class="card" />
      <div v-else class="card muted">{{ $t('stats.noData') }}</div>

      <!-- Chart -->
      <div v-if="history.length > 1" class="card chart-card">
        <div class="chart-toolbar">
          <h2>{{ $t('detail.evolution') }}</h2>
          <div class="period-tabs">
            <button
              v-for="p in periods"
              :key="p.label"
              :class="['period-btn', { active: activePeriod === p.label }]"
              @click="selectPeriod(p.label)"
            >{{ p.title }}</button>
          </div>
        </div>
        <Line :data="chartData" :options="chartOptions" class="chart" />
      </div>

    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, RouterLink } from 'vue-router'
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
import { trackersApi, statsApi } from '@/services/api'
import type { Tracker, TrackerStats } from '@/services/api'
import TrackerFormModal from '@/components/TrackerFormModal.vue'
import StatsGrid from '@/components/StatsGrid.vue'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler)

const { t, d } = useI18n()

const route = useRoute()
const trackerId = Number(route.params.id)

const tracker = ref<Tracker | null>(null)
const history = ref<TrackerStats[]>([])
const loading = ref(true)
const error = ref('')

// --- Period filter ---
const periods = computed(() => [
  { label: '7d',  days: 7,  title: '7d' },
  { label: '30d', days: 30, title: '30d' },
  { label: '90d', days: 90, title: '90d' },
  { label: 'All', days: 0,  title: t('detail.periods.all') },
])
const activePeriod = ref('7d')

async function loadHistory() {
  const p = periods.value.find((p) => p.label === activePeriod.value)!
  const params: Parameters<typeof statsApi.getHistory>[1] = p.days > 0
    ? { startDate: new Date(Date.now() - p.days * 86_400_000).toISOString().slice(0, 10) }
    : { startDate: tracker.value?.created_at.slice(0, 10) ?? '2000-01-01' }
  history.value = await statsApi.getHistory(trackerId, params)
}

async function selectPeriod(label: string) {
  activePeriod.value = label
  await loadHistory()
}

// --- Refresh ---
const refreshing = ref(false)
async function doRefresh() {
  refreshing.value = true
  try {
    await trackersApi.refresh(trackerId)
    await loadHistory()
  } catch {
    error.value = t('detail.errorLoad')
  } finally {
    refreshing.value = false
  }
}

// --- Edit modal ---
const showForm = ref(false)

async function onSaved(updated: Tracker) {
  tracker.value = updated
  await loadHistory()
}

const latest = computed(() => history.value[0] ?? null)

// --- Chart ---
const chartData = computed(() => {
  const rows = [...history.value].reverse() // oldest first
  return {
    labels: rows.map((r) => d(new Date(r.fetched_at), 'short')),
    datasets: [
      {
        label: t('chart.ratio'),
        data: rows.map((r) => r.ratio),
        borderColor: '#818cf8',
        backgroundColor: 'rgba(129,140,248,0.1)',
        fill: true,
        tension: 0.3,
        pointRadius: rows.length > 60 ? 0 : 3,
        yAxisID: 'yRatio',
      },
      {
        label: t('chart.uploadedGib'),
        data: rows.map((r) => +(r.uploaded / 1073741824).toFixed(2)),
        borderColor: '#4ade80',
        backgroundColor: 'transparent',
        tension: 0.3,
        pointRadius: rows.length > 60 ? 0 : 3,
        yAxisID: 'yBytes',
      },
      {
        label: t('chart.downloadedGib'),
        data: rows.map((r) => +(r.downloaded / 1073741824).toFixed(2)),
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
      ticks: { callback: (v) => Number(v).toFixed(2) },
    },
    yBytes: {
      type: 'linear',
      position: 'right',
      title: { display: true, text: t('chart.gib') },
      grid: { drawOnChartArea: false },
    },
  },
}))

// --- Load ---
onMounted(async () => {
  try {
    tracker.value = await trackersApi.getById(trackerId)
    await loadHistory()
  } catch {
    error.value = t('detail.errorLoad')
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.detail-view {
  max-width: 1100px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.detail-header {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.back-link {
  font-size: 0.85rem;
  color: var(--accent);
  text-decoration: none;
}
.back-link:hover { text-decoration: underline; color: var(--accent-2); }

.site-title h1 {
  margin: 0 0 0.2rem;
  font-size: 1.6rem;
  color: var(--text);
}

.site-url {
  font-size: 0.82rem;
  color: var(--text-label);
}

.card {
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.5rem;
  box-shadow: var(--shadow), var(--shadow-glow);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  transition: border-color 0.2s;
}

.card:hover {
  border-color: var(--border-bright);
}

.card h2 {
  margin: 0 0 1rem;
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-muted);
}

.chart-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
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
.period-btn:hover { border-color: var(--border-bright); color: var(--accent); }
.period-btn.active {
  background: linear-gradient(135deg, #4f46e5, #7c3aed);
  border-color: transparent;
  color: white;
}

.chart {
  height: 320px !important;
}

@media (max-width: 640px) {
  .chart {
    height: 220px !important;
  }

  .chart-toolbar {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.5rem;
  }
}

/* ---- Header action buttons ---- */
.tracker-title-row {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
}

.header-actions {
  display: flex;
  gap: 0.5rem;
  flex-shrink: 0;
}

.btn-detail-action {
  padding: 0.4rem 0.9rem;
  border-radius: 6px;
  cursor: pointer;
  font-size: 0.85rem;
  font-weight: 600;
  transition: opacity 0.2s, box-shadow 0.2s, background 0.2s, border-color 0.2s, color 0.2s;
}

.btn-detail-refresh {
  background: transparent;
  color: var(--accent);
  border: 1px solid var(--border);
}
.btn-detail-refresh:hover {
  border-color: var(--border-bright);
  background: var(--accent-glow);
}
.btn-detail-refresh:disabled { opacity: 0.4; cursor: not-allowed; }

.btn-detail-edit {
  background: linear-gradient(135deg, #4f46e5, #7c3aed);
  color: white;
  border: none;
  box-shadow: 0 0 14px rgba(129, 140, 248, 0.3);
}
.btn-detail-edit:hover { opacity: 0.9; box-shadow: 0 0 22px rgba(129, 140, 248, 0.45); }
</style>
