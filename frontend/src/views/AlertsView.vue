<template>
  <div class="alerts-view">
    <PageHeader :title="$t('alerts.title')">
      <button class="btn-primary" @click="openAddModal">{{ $t('alerts.add') }}</button>
    </PageHeader>

    <p v-if="store.loading" class="muted">{{ $t('common.loading') }}</p>
    <p v-else-if="!store.alertConfigs.length" class="muted">{{ $t('alerts.empty') }}</p>

    <div v-else class="alerts-grid">
      <div v-for="cfg in store.alertConfigs" :key="cfg.id" class="alert-card">
        <div class="card-header">
          <span class="alert-name">{{ cfg.name }}</span>
          <div class="card-actions">
            <button class="btn-edit" :title="$t('buttons.editAlert')" @click="openEditModal(cfg)">
              <font-awesome-icon :icon="['fas', 'pen']" />
            </button>
            <button class="btn-delete" :title="$t('buttons.removeAlert')" @click="removeAlert(cfg)">
              <font-awesome-icon :icon="['fas', 'xmark']" />
            </button>
          </div>
        </div>

        <div class="card-meta">
          <span :class="['type-badge', cfg.alert_type]">
            {{ cfg.alert_type === 'ratio_alert' ? $t('alerts.modal.ratioAlert') : $t('alerts.modal.syncError') }}
          </span>
          <span :class="['enabled-badge', cfg.enabled ? 'on' : 'off']">
            {{ cfg.enabled ? 'enabled' : 'disabled' }}
          </span>
        </div>

        <div v-if="cfg.alert_type === 'ratio_alert'" class="threshold-row">
          Threshold: {{ cfg.ratio_threshold }}
        </div>

        <div class="tracker-row">
          <span class="muted-small">
            {{ cfg.all_trackers ? $t('alerts.allTrackers') : $t('alerts.trackerCount', { n: cfg.trackers?.length ?? 0 }, cfg.trackers?.length ?? 0) }}
          </span>
        </div>

        <div v-if="cfg.notifier_configs?.length" class="notifiers-row">
          <span v-for="nc in cfg.notifier_configs" :key="nc.id" class="notifier-badge">{{ nc.name }}</span>
        </div>
        <p v-else class="muted-small">No notifiers attached</p>
      </div>
    </div>

    <AlertFormModal
      v-model="showModal"
      :notifiers="notifiers"
      :trackers="trackers"
      :existing="editingConfig"
      @saved="onSaved"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAlertConfigsStore } from '@/stores/alertConfigs'
import { notifierConfigsApi, trackersApi, type NotifierConfig, type AlertConfig } from '@/services/api'
import type { Tracker } from '@/services/api'
import AlertFormModal from '@/components/AlertFormModal.vue'
import PageHeader from '@/components/PageHeader.vue'

const store = useAlertConfigsStore()
const notifiers = ref<NotifierConfig[]>([])
const trackers = ref<Tracker[]>([])
const showModal = ref(false)
const editingConfig = ref<AlertConfig | null>(null)

onMounted(async () => {
  await store.fetchAll()
  ;[notifiers.value, trackers.value] = await Promise.all([
    notifierConfigsApi.getAll(),
    trackersApi.getAll(),
  ])
})

function openAddModal() {
  editingConfig.value = null
  showModal.value = true
}

function openEditModal(cfg: AlertConfig) {
  editingConfig.value = cfg
  showModal.value = true
}

async function removeAlert(cfg: AlertConfig) {
  if (!confirm(`Delete alert "${cfg.name}"?`)) return
  await store.remove(cfg.id)
}

function onSaved() {
  showModal.value = false
}
</script>

<style scoped>
.alerts-view {
  max-width: 1100px;
  margin: 0 auto;
}

.alerts-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 1.25rem;
}

.alert-card {
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.25rem;
  box-shadow: var(--shadow), var(--shadow-glow);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  transition: border-color 0.2s, box-shadow 0.2s;
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}

.alert-card:hover {
  border-color: var(--border-bright);
  box-shadow: var(--shadow), 0 0 28px rgba(129, 140, 248, 0.18);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.alert-name {
  font-weight: 600;
  color: var(--text);
}

.card-actions {
  display: flex;
  gap: 0.3rem;
  flex-shrink: 0;
}

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

.btn-edit:hover  { color: #a78bfa; }
.btn-delete:hover { color: var(--ratio-bad); }

.card-meta {
  display: flex;
  gap: 0.4rem;
  align-items: center;
}

.type-badge {
  font-size: 0.75rem;
  padding: 0.1rem 0.5rem;
  border-radius: 4px;
  font-weight: 600;
}
.type-badge.ratio_alert {
  background: rgba(251, 191, 36, 0.15);
  color: #fbbf24;
  border: 1px solid rgba(251, 191, 36, 0.3);
}
.type-badge.sync_error {
  background: rgba(239, 68, 68, 0.12);
  color: #f87171;
  border: 1px solid rgba(239, 68, 68, 0.25);
}

.enabled-badge {
  font-size: 0.72rem;
  padding: 0.1rem 0.45rem;
  border-radius: 4px;
}
.enabled-badge.on {
  background: rgba(80, 200, 140, 0.12);
  color: #6dd4a8;
  border: 1px solid rgba(80, 200, 140, 0.2);
}
.enabled-badge.off {
  background: rgba(107, 114, 128, 0.12);
  color: var(--text-label);
  border: 1px solid rgba(107, 114, 128, 0.2);
}

.threshold-row {
  font-size: 0.82rem;
  color: var(--text-label);
}

.tracker-row {
  font-size: 0.82rem;
}

.notifiers-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
}

.notifier-badge {
  font-size: 0.77rem;
  background: rgba(80, 200, 140, 0.12);
  color: #6dd4a8;
  border: 1px solid rgba(80, 200, 140, 0.2);
  padding: 0.1rem 0.5rem;
  border-radius: 4px;
}
</style>
