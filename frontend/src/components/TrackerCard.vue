<template>
  <div class="tracker-card">
    <div class="card-header">
      <div class="card-title-row">
        <RouterLink :to="`/trackers/${entry.tracker.id}`" class="tracker-name-link">
          <h3>{{ entry.tracker.name }}</h3>
        </RouterLink>
        <span
          class="status-dot"
          :class="statusClass"
          :title="statusTooltip"
        ></span>
      </div>
      <div class="card-actions">
        <button class="btn-refresh" :title="$t('buttons.refresh')" :disabled="refreshing" @click="onRefresh">
          <font-awesome-icon :icon="['fas', 'rotate-right']" />
        </button>
        <button class="btn-edit" :title="$t('buttons.editSite')" @click="emit('edit')">
          <font-awesome-icon :icon="['fas', 'pen']" />
        </button>
        <button class="btn-delete" :title="$t('buttons.removeSite')" @click="emit('delete')">
          <font-awesome-icon :icon="['fas', 'xmark']" />
        </button>
      </div>
    </div>

    <StatsGrid v-if="entry.stats" :stats="entry.stats" />
    <p v-else class="muted">{{ $t('stats.noDataCard') }}</p>

    <p class="card-footer">
      <span v-if="entry.stats">{{ $t('stats.lastFetched', { date: $d(new Date(entry.stats.fetched_at), 'short') }) }}</span>
      <span class="scraper-badge">{{ entry.tracker.scraper_key }}</span>
      <button v-if="entry.stats" class="btn-history" @click="toggleHistory">
        {{ showHistory ? $t('history.hide') : $t('history.title') }}
      </button>
    </p>

    <div v-if="showHistory" class="history-panel">
      <p v-if="historyLoading" class="muted history-loading">{{ $t('common.loading') }}</p>
      <p v-else-if="historyError" class="history-error">{{ historyError }}</p>
      <template v-else>
        <HistoryTable v-if="history.length" :history="history" @delete="deleteEntry" />
        <p v-else class="muted">{{ $t('history.noHistory') }}</p>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import type { DashboardEntry, TrackerStats } from '@/services/api'
import { statsApi } from '@/services/api'
import StatsGrid from '@/components/StatsGrid.vue'
import HistoryTable from '@/components/HistoryTable.vue'

const { t } = useI18n()

const props = defineProps<{ entry: DashboardEntry }>()
const emit = defineEmits<{ delete: []; refresh: []; edit: [] }>()

const refreshing = ref(false)
const showHistory = ref(false)
const historyLoading = ref(false)
const historyError = ref('')
const history = ref<TrackerStats[]>([])

async function onRefresh() {
  refreshing.value = true
  try {
    emit('refresh')
  } finally {
    refreshing.value = false
  }
}

async function toggleHistory() {
  showHistory.value = !showHistory.value
  if (showHistory.value && !history.value.length) {
    await loadHistory()
  }
}

async function loadHistory() {
  historyLoading.value = true
  historyError.value = ''
  try {
    history.value = await statsApi.getHistory(props.entry.tracker.id)
  } catch {
    historyError.value = t('history.errorLoad')
  } finally {
    historyLoading.value = false
  }
}

async function deleteEntry(statId: number) {
  try {
    await statsApi.deleteEntry(props.entry.tracker.id, statId)
    history.value = history.value.filter((r) => r.id !== statId)
  } catch {
    historyError.value = t('history.errorDelete')
  }
}

const statusClass = computed(() => {
  if (!props.entry.tracker.last_scraped_at) return 'status-unknown'
  return props.entry.tracker.last_error ? 'status-error' : 'status-ok'
})

const statusTooltip = computed(() => {
  if (!props.entry.tracker.last_scraped_at) return t('status.neverScraped')
  if (props.entry.tracker.last_error) return props.entry.tracker.last_error
  return t('status.ok')
})

</script>

<style scoped>
.tracker-card {
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.25rem;
  box-shadow: var(--shadow), var(--shadow-glow);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  transition: border-color 0.2s, box-shadow 0.2s;
}

.tracker-card:hover {
  border-color: var(--border-bright);
  box-shadow: var(--shadow), 0 0 28px rgba(129, 140, 248, 0.18);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 0.75rem;
}

.card-title-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.card-header h3 {
  margin: 0 0 0.2rem;
  font-size: 1rem;
  font-weight: 600;
  color: var(--text);
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
  cursor: default;
  margin-bottom: 0.2rem;
}

.status-ok      { background: #22c55e; box-shadow: 0 0 6px rgba(34, 197, 94, 0.6); }
.status-error   { background: #ef4444; box-shadow: 0 0 6px rgba(239, 68, 68, 0.6); }
.status-unknown { background: #6b7280; }

.tracker-name-link {
  text-decoration: none;
  color: inherit;
}
.tracker-name-link:hover h3 { color: var(--accent); }

.tracker-url {
  font-size: 0.78rem;
  color: var(--text-label);
  word-break: break-all;
}

.card-actions {
  display: flex;
  gap: 0.3rem;
  flex-shrink: 0;
}

.btn-refresh,
.btn-edit,
.btn-delete {
  background: none;
  border: none;
  font-size: 1.3rem;
  line-height: 1;
  cursor: pointer;
  padding: 0;
  transition: color 0.15s;
  color: var(--text-label);
}

.btn-refresh:hover { color: var(--accent); }
.btn-edit:hover    { color: #a78bfa; }
.btn-delete:hover  { color: var(--ratio-bad); }
.btn-refresh:disabled { opacity: 0.4; cursor: not-allowed; }

.card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.75rem;
  color: var(--text-muted);
  margin: 0;
}

.btn-history {
  background: none;
  border: none;
  font-size: 0.75rem;
  color: var(--accent);
  cursor: pointer;
  padding: 0;
  text-decoration: underline;
}
.btn-history:hover { color: var(--accent-2); }

.history-panel {
  margin-top: 0.75rem;
  border-top: 1px solid var(--border);
  padding-top: 0.75rem;
}

.history-loading { font-size: 0.85rem; }

.history-error {
  font-size: 0.85rem;
  color: var(--ratio-bad);
}

.scraper-badge {
  background: rgba(99, 102, 241, 0.15);
  border: 1px solid var(--border);
  border-radius: 999px;
  padding: 0.1rem 0.55rem;
  font-size: 0.7rem;
  color: var(--accent);
}
</style>

