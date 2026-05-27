<template>
  <div class="reports-view">
    <PageHeader :title="$t('reports.title')">
      <button class="btn-primary" @click="openAddModal">{{ $t('reports.add') }}</button>
    </PageHeader>

    <p v-if="store.loading" class="muted">{{ $t('common.loading') }}</p>
    <p v-else-if="!store.reports.length" class="muted">{{ $t('reports.empty') }}</p>

    <div v-else class="reports-grid">
      <div v-for="report in store.reports" :key="report.id" class="report-card">
        <div class="card-header">
          <span class="report-name">{{ report.name }}</span>
          <div class="card-actions">
            <button
              class="btn-send"
              :disabled="sending === report.id"
              :title="$t('reports.sendNow')"
              @click="sendReport(report.id)"
            >
              <font-awesome-icon v-if="sending === report.id" :icon="['fas', 'spinner']" spin />
              <font-awesome-icon v-else :icon="['fas', 'play']" />
            </button>
            <button class="btn-edit" :title="$t('buttons.editReport')" @click="openEditModal(report)">
              <font-awesome-icon :icon="['fas', 'pen']" />
            </button>
            <button class="btn-delete" :title="$t('buttons.removeReport')" @click="removeReport(report)">
              <font-awesome-icon :icon="['fas', 'xmark']" />
            </button>
          </div>
        </div>

        <div v-if="report.notifier_configs?.length" class="report-notifiers">
          <span v-for="cfg in report.notifier_configs" :key="cfg.id" class="notifier-badge">{{ cfg.name }}</span>
        </div>
        <p v-else class="muted-small">{{ $t('reports.noNotifiers') }}</p>

        <p class="card-footer">
          <span class="cron-badge">{{ report.cron_expr }}</span>
          <span class="report-sent">
            {{ report.last_sent_at
              ? $t('reports.lastSent', { date: formatDate(report.last_sent_at) })
              : $t('reports.neverSent') }}
          </span>
        </p>

        <p v-if="sendError === report.id" class="row-error">{{ $t('reports.sendError') }}</p>
      </div>
    </div>

    <ReportFormModal
      v-model="showModal"
      :notifiers="notifiers"
      :existing="editingReport"
      @saved="onSaved"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useReportsStore } from '@/stores/reports'
import { notifierConfigsApi, type NotifierConfig, type Report } from '@/services/api'
import ReportFormModal from '@/components/ReportFormModal.vue'
import PageHeader from '@/components/PageHeader.vue'

const store = useReportsStore()

const notifiers = ref<NotifierConfig[]>([])
const showModal = ref(false)
const editingReport = ref<Report | null>(null)
const sending = ref<number | null>(null)
const sendError = ref<number | null>(null)

onMounted(async () => {
  await store.fetchAll()
  notifiers.value = await notifierConfigsApi.getAll()
})

function openAddModal() {
  editingReport.value = null
  showModal.value = true
}

function openEditModal(report: Report) {
  editingReport.value = report
  showModal.value = true
}

async function removeReport(report: Report) {
  if (!confirm(`Delete report "${report.name}"?`)) return
  await store.remove(report.id)
}

async function sendReport(id: number) {
  sending.value = id
  sendError.value = null
  try {
    await store.send(id)
  } catch {
    sendError.value = id
  } finally {
    sending.value = null
  }
}

function onSaved() {
  showModal.value = false
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleString()
}
</script>

<style scoped>
.reports-view {
  max-width: 1100px;
  margin: 0 auto;
}

.reports-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 1.25rem;
}

.report-card {
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
  gap: 0.75rem;
}

.report-card:hover {
  border-color: var(--border-bright);
  box-shadow: var(--shadow), 0 0 28px rgba(129, 140, 248, 0.18);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.report-name {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text);
}

.card-actions {
  display: flex;
  gap: 0.3rem;
  flex-shrink: 0;
}

.btn-send,
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

.btn-send:hover:not(:disabled) { color: var(--ratio-good); }
.btn-edit:hover                 { color: #a78bfa; }
.btn-delete:hover               { color: var(--ratio-bad); }
.btn-send:disabled              { opacity: 0.4; cursor: not-allowed; }

.report-notifiers {
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

.card-footer {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: auto;
}

.cron-badge {
  font-size: 0.78rem;
  font-family: monospace;
  background: var(--accent-glow);
  color: var(--accent);
  padding: 0.1rem 0.45rem;
  border-radius: 4px;
}

.report-sent {
  font-size: 0.8rem;
  color: var(--text-label);
  margin-left: auto;
}

.row-error {
  font-size: 0.82rem;
  color: var(--ratio-bad);
  margin: 0;
}
</style>
