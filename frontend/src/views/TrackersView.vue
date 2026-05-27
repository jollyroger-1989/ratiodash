<template>
  <div class="trackers-view">
    <PageHeader :title="$t('trackers.title')">
      <button class="btn-primary" @click="showForm = true">{{ $t('trackers.addSite') }}</button>
    </PageHeader>

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
import { type Tracker } from '@/services/api'
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
</script>

<style scoped>
.trackers-view {
  max-width: 1100px;
  margin: 0 auto;
}

.sites-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 1.25rem;
}

.error { color: var(--ratio-bad); }
.muted { color: var(--text-muted); }
</style>

