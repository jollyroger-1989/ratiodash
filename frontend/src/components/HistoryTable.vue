<template>
  <div class="table-scroll">
  <table class="history-table">
    <thead>
      <tr>
        <th>{{ $t('common.date') }}</th>
        <th>{{ $t('common.ratio') }}</th>
        <th class="col-upload">{{ $t('common.uploaded') }}</th>
        <th class="col-download">{{ $t('common.downloaded') }}</th>
        <th></th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="row in history" :key="row.id">
        <td>{{ $d(new Date(row.fetched_at), 'short') }}</td>
        <td :class="ratioClass(row.ratio)">{{ row.ratio.toFixed(2) }}</td>
        <td class="col-upload">{{ formatBytes(row.uploaded) }}</td>
        <td class="col-download">{{ formatBytes(row.downloaded) }}</td>
        <td>
          <button class="btn-del-entry" :title="$t('buttons.deleteEntry')" @click="$emit('delete', row.id)">
            <font-awesome-icon :icon="['fas', 'xmark']" />
          </button>
        </td>
      </tr>
    </tbody>
  </table>
  </div>
</template>

<script setup lang="ts">
import type { TrackerStats } from '@/services/api'
import { ratioClass } from '@/composables/ratioClass'
import { formatBytes } from '@/composables/formatBytes'

defineProps<{ history: TrackerStats[] }>()
defineEmits<{ delete: [id: number] }>()
</script>

<style scoped>
.table-scroll {
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

.history-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.83rem;
}

.history-table th {
  text-align: left;
  color: var(--text-label);
  font-weight: 600;
  text-transform: uppercase;
  font-size: 0.7rem;
  letter-spacing: 0.04em;
  padding: 0.3rem 0.75rem 0.3rem 0;
  border-bottom: 1px solid var(--border);
}

.history-table td {
  padding: 0.4rem 0.75rem 0.4rem 0;
  color: var(--text-muted);
  border-bottom: 1px solid rgba(99, 102, 241, 0.1);
}

.history-table tbody tr:last-child td { border-bottom: none; }

.btn-del-entry {
  background: none;
  border: none;
  font-size: 1rem;
  line-height: 1;
  cursor: pointer;
  color: var(--text-label);
  padding: 0;
  transition: color 0.15s;
}
.btn-del-entry:hover { color: var(--ratio-bad); }

@media (max-width: 480px) {
  .col-upload,
  .col-download {
    display: none;
  }
}
</style>
