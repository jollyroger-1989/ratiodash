<template>
  <div class="trackers-view">
    <PageHeader :title="$t('trackers.title')">
      <button class="btn-primary" @click="showForm = true">{{ $t('trackers.addSite') }}</button>
    </PageHeader>

    <div class="sort-toolbar">
      <label class="sort-label">{{ $t('trackers.sort.label') }}</label>
      <select class="sort-select" :value="store.sortBy" @change="onSortByChange">
        <option value="">{{ $t('trackers.sort.default') }}</option>
        <option value="ratio">{{ $t('trackers.sort.ratio') }}</option>
        <option value="uploaded">{{ $t('trackers.sort.uploaded') }}</option>
        <option value="downloaded">{{ $t('trackers.sort.downloaded') }}</option>
      </select>
      <button
        v-if="store.sortBy"
        class="sort-order-btn"
        :title="store.sortOrder === 'asc' ? $t('trackers.sort.asc') : $t('trackers.sort.desc')"
        @click="toggleOrder"
      >
        {{ store.sortOrder === 'asc' ? '↑' : '↓' }}
        {{ store.sortOrder === 'asc' ? $t('trackers.sort.asc') : $t('trackers.sort.desc') }}
      </button>
    </div>

    <p v-if="store.error && !showForm" class="error">{{ store.error }}</p>
    <p v-if="store.loading && !store.dashboard.length" class="muted">{{ $t('common.loading') }}</p>
    <p v-else-if="!store.dashboard.length && !store.loading" class="muted">{{ $t('trackers.noSites') }}</p>

    <div v-else class="sites-grid">
      <TrackerCard
        v-for="entry in store.dashboard"
        :key="entry.tracker.id"
        :entry="entry"
        @delete="store.remove(entry.tracker.id)"
        @refresh="store.refresh(entry.tracker.id)"
        @edit="openEdit(entry.tracker)"
      />
    </div>

    <TrackerFormModal
      v-model="showForm"
      :tracker="editingTracker"
      @saved="store.fetchDashboard()"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useTrackersStore } from '@/stores/trackers'
import { type Tracker, type SortBy, type SortOrder } from '@/services/api'
import TrackerCard from '@/components/TrackerCard.vue'
import TrackerFormModal from '@/components/TrackerFormModal.vue'
import PageHeader from '@/components/PageHeader.vue'

const store = useTrackersStore()
const showForm = ref(false)
const editingTracker = ref<Tracker | undefined>(undefined)

onMounted(() => store.fetchDashboard())

function openEdit(tracker: Tracker) {
  editingTracker.value = tracker
  showForm.value = true
}

function onSortByChange(event: Event) {
  const by = (event.target as HTMLSelectElement).value as SortBy | ''
  store.setSort(by, store.sortOrder)
}

function toggleOrder() {
  const newOrder: SortOrder = store.sortOrder === 'asc' ? 'desc' : 'asc'
  store.setSort(store.sortBy, newOrder)
}
</script>

<style scoped>
.trackers-view {
  max-width: 1100px;
  margin: 0 auto;
}

.sort-toolbar {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 1.25rem;
}

.sort-label {
  font-size: 0.875rem;
  color: var(--text-muted);
}

.sort-select {
  padding: 0.3rem 0.5rem;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--text);
  font-size: 0.875rem;
  cursor: pointer;
}

.sort-order-btn {
  padding: 0.3rem 0.65rem;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--text);
  font-size: 0.875rem;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.sort-order-btn:hover {
  background: var(--surface-hover, var(--border));
}

.sites-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 1.25rem;
}

.error { color: var(--ratio-bad); }
.muted { color: var(--text-muted); }
</style>

