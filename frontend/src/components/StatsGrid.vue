<template>
  <div class="stats-grid" :class="showDate ? 'stats-grid--4' : 'stats-grid--3'">
    <div class="stat">
      <span class="stat-label">{{ $t('common.ratio') }}</span>
      <span class="stat-value" :class="ratioClass(stats.ratio)">{{ stats.ratio.toFixed(2) }}</span>
    </div>
    <div class="stat">
      <span class="stat-label">{{ $t('common.uploaded') }}</span>
      <span class="stat-value">{{ formatBytes(stats.uploaded) }}</span>
    </div>
    <div class="stat">
      <span class="stat-label">{{ $t('common.downloaded') }}</span>
      <span class="stat-value">{{ formatBytes(stats.downloaded) }}</span>
    </div>
    <div v-if="showDate" class="stat">
      <span class="stat-label">{{ $t('stats.lastFetch') }}</span>
      <span class="stat-value stat-date">{{ $d(new Date(stats.fetched_at), 'short') }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { TrackerStats } from '@/services/api'
import { ratioClass } from '@/composables/ratioClass'
import { formatBytes } from '@/composables/formatBytes'

withDefaults(defineProps<{
  stats: TrackerStats
  showDate?: boolean
}>(), { showDate: false })
</script>

<style scoped>
.stats-grid { display: grid; }

.stats-grid--3 {
  grid-template-columns: repeat(3, 1fr);
  gap: 0.75rem;
  margin-bottom: 0.75rem;
}

.stats-grid--4 {
  grid-template-columns: repeat(4, 1fr);
  gap: 1rem;
}

@media (max-width: 600px) {
  .stats-grid--3,
  .stats-grid--4 { grid-template-columns: repeat(2, 1fr); }
}

.stat { display: flex; flex-direction: column; }

.stat-label {
  font-size: 0.72rem;
  color: var(--text-label);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.stat-value {
  font-size: 1rem;
  font-weight: 700;
  color: var(--text);
}

.stat-date { font-size: 0.9rem; font-weight: 500; }
</style>
