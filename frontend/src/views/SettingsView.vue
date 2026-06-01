<template>
  <div class="settings-page">
    <h1 class="settings-title">{{ $t('settings.title') }}</h1>

    <div class="settings-card credentials-card">
      <h2 class="section-title">{{ $t('settings.credentials.title') }}</h2>
      <p class="section-subtitle">{{ $t('settings.credentials.subtitle') }}</p>

      <form class="settings-form" @submit.prevent="submit">
        <div class="field">
          <label for="new-username">{{ $t('settings.credentials.newUsername') }}</label>
          <input
            id="new-username"
            v-model="newUsername"
            type="text"
            autocomplete="username"
            :placeholder="$t('settings.credentials.newUsernamePlaceholder')"
            required
          />
        </div>

        <div class="field">
          <label for="new-password">{{ $t('settings.credentials.newPassword') }}</label>
          <input
            id="new-password"
            v-model="newPassword"
            type="password"
            autocomplete="new-password"
            :placeholder="$t('auth.setup.passwordHint')"
            required
            minlength="8"
          />
        </div>

        <div class="field">
          <label for="confirm-password">{{ $t('auth.setup.confirm') }}</label>
          <input
            id="confirm-password"
            v-model="confirmPassword"
            type="password"
            autocomplete="new-password"
            :placeholder="$t('auth.setup.confirm')"
            required
          />
        </div>

        <div class="field">
          <label for="current-password">{{ $t('settings.credentials.currentPassword') }}</label>
          <input
            id="current-password"
            v-model="currentPassword"
            type="password"
            autocomplete="current-password"
            :placeholder="$t('settings.credentials.currentPasswordPlaceholder')"
            required
          />
        </div>

        <p v-if="error" class="form-error">{{ error }}</p>
        <p v-if="success" class="form-success">{{ $t('settings.credentials.success') }}</p>

        <button type="submit" class="submit-btn" :disabled="loading">
          {{ loading ? $t('settings.credentials.saving') : $t('settings.credentials.save') }}
        </button>
      </form>
    </div>

    <div class="settings-card api-clients-card">
      <div class="card-header-row">
        <div>
          <h2 class="section-title">{{ $t('settings.apiClients.title') }}</h2>
          <p class="section-subtitle">{{ $t('settings.apiClients.subtitle') }}</p>
        </div>
        <button class="submit-btn add-api-client-btn" @click="openApiClientModal">
          <font-awesome-icon :icon="['fas', 'plus']" /> {{ $t('settings.apiClients.create') }}
        </button>
      </div>

      <p v-if="apiClientsError" class="form-error token-error">{{ apiClientsError }}</p>

      <p v-if="loadingApiClients" class="muted">{{ $t('common.loading') }}</p>
      <p v-else-if="!apiClients.length" class="empty-api-clients">{{ $t('settings.apiClients.empty') }}</p>

      <ul v-else class="api-clients-list">
        <li v-for="client in apiClients" :key="client.id" class="api-client-row">
          <div class="api-client-info">
            <span class="api-client-name">{{ client.name }}</span>
            <span class="api-client-prefix">{{ client.key_prefix }}…</span>
            <span v-if="client.last_used_at" class="api-client-last-used">
              {{ $t('settings.apiClients.lastUsed', { date: $d(client.last_used_at, 'short') }) }}
            </span>
            <span v-else class="api-client-last-used muted-small">{{ $t('settings.apiClients.neverUsed') }}</span>
          </div>

          <button class="btn-icon btn-delete" type="button" :title="$t('settings.apiClients.revoke')" @click="removeApiClient(client.id)">
            <font-awesome-icon :icon="['fas', 'xmark']" />
          </button>
        </li>
      </ul>
    </div>

    <!-- Notifiers section -->
    <div class="settings-card notifiers-card">
      <div class="card-header-row">
        <div>
          <h2 class="section-title">{{ $t('settings.notifiers.title') }}</h2>
          <p class="section-subtitle">{{ $t('settings.notifiers.subtitle') }}</p>
        </div>
        <button class="submit-btn add-notifier-btn" @click="openAddModal">
          <font-awesome-icon :icon="['fas', 'plus']" /> {{ $t('settings.notifiers.add') }}
        </button>
      </div>

      <p v-if="loadingNotifiers" class="muted">{{ $t('common.loading') }}</p>
      <p v-else-if="!notifiers.length" class="empty-notifiers">{{ $t('settings.notifiers.empty') }}</p>

      <ul v-else class="notifiers-list">
        <li v-for="cfg in notifiers" :key="cfg.id" class="notifier-row">
          <div class="notifier-info">
            <span class="notifier-name">{{ cfg.name }}</span>
            <span class="notifier-type-badge">{{ cfg.type }}</span>
          </div>
          <div class="notifier-actions">
            <label class="toggle" :title="cfg.enabled ? $t('settings.notifiers.disable') : $t('settings.notifiers.enable')">
              <input type="checkbox" :checked="cfg.enabled" @change="toggleNotifier(cfg)" />
              <span class="toggle-slider"></span>
            </label>
            <button class="btn-icon btn-edit" :title="$t('settings.notifiers.edit')" @click="openEditModal(cfg)">
              <font-awesome-icon :icon="['fas', 'pen']" />
            </button>
            <button class="btn-icon btn-delete" :title="$t('settings.notifiers.delete')" @click="removeNotifier(cfg)">
              <font-awesome-icon :icon="['fas', 'xmark']" />
            </button>
          </div>
        </li>
      </ul>
    </div>

    <NotifierFormModal
      v-model="showModal"
      :types="notifierTypes"
      :existing="editingNotifier"
      @saved="onNotifierSaved"
    />

    <APIClientFormModal
      v-model="showAPIClientModal"
      @saved="onAPIClientSaved"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  settingsApi,
  notifierConfigsApi,
  apiClientsApi,
  type APIClient,
  type NotifierConfig,
  type NotifierTypeInfo,
} from '@/services/api'
import NotifierFormModal from '@/components/NotifierFormModal.vue'
import APIClientFormModal from '@/components/APIClientFormModal.vue'

const { t } = useI18n()

// ---- Credentials ----

const newUsername = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const currentPassword = ref('')
const loading = ref(false)
const error = ref('')
const success = ref(false)

// ---- API clients ----

const apiClients = ref<APIClient[]>([])
const loadingApiClients = ref(false)
const apiClientsError = ref('')
const showAPIClientModal = ref(false)

async function fetchApiClients() {
  loadingApiClients.value = true
  apiClientsError.value = ''
  try {
    apiClients.value = await apiClientsApi.getAll()
  } catch {
    apiClientsError.value = t('settings.apiClients.fetchError')
  } finally {
    loadingApiClients.value = false
  }
}

function openApiClientModal() {
  showAPIClientModal.value = true
}

async function removeApiClient(id: number) {
  apiClientsError.value = ''
  try {
    await apiClientsApi.remove(id)
    apiClients.value = apiClients.value.filter((client) => client.id !== id)
  } catch {
    apiClientsError.value = t('settings.apiClients.deleteError')
  }
}

async function onAPIClientSaved() {
  await fetchApiClients()
}

async function submit() {
  error.value = ''
  success.value = false

  if (newPassword.value !== confirmPassword.value) {
    error.value = t('auth.setup.mismatch')
    return
  }
  if (newPassword.value.length < 8) {
    error.value = t('auth.setup.tooShort')
    return
  }

  loading.value = true
  try {
    await settingsApi.updateCredentials(currentPassword.value, newUsername.value, newPassword.value)
    success.value = true
    newPassword.value = ''
    confirmPassword.value = ''
    currentPassword.value = ''
  } catch (err: unknown) {
    const status = (err as { response?: { status: number } })?.response?.status
    if (status === 422) {
      error.value = t('settings.credentials.wrongCurrentPassword')
    } else {
      error.value = t('settings.credentials.error')
    }
  } finally {
    loading.value = false
  }
}

// ---- Notifiers ----

const notifiers = ref<NotifierConfig[]>([])
const notifierTypes = ref<NotifierTypeInfo[]>([])
const loadingNotifiers = ref(false)
const showModal = ref(false)
const editingNotifier = ref<NotifierConfig | undefined>(undefined)

async function fetchNotifiers() {
  loadingNotifiers.value = true
  try {
    const [configs, types] = await Promise.all([
      notifierConfigsApi.getAll(),
      notifierTypes.value.length ? Promise.resolve(notifierTypes.value) : notifierConfigsApi.getTypes(),
    ])
    notifiers.value = configs
    if (!notifierTypes.value.length) notifierTypes.value = types
  } finally {
    loadingNotifiers.value = false
  }
}

function openAddModal() {
  editingNotifier.value = undefined
  showModal.value = true
}

function openEditModal(cfg: NotifierConfig) {
  editingNotifier.value = cfg
  showModal.value = true
}

async function toggleNotifier(cfg: NotifierConfig) {
  const updated = await notifierConfigsApi.update(cfg.id, { enabled: !cfg.enabled })
  const idx = notifiers.value.findIndex((n) => n.id === cfg.id)
  if (idx !== -1) notifiers.value[idx] = updated
}

async function removeNotifier(cfg: NotifierConfig) {
  await notifierConfigsApi.remove(cfg.id)
  notifiers.value = notifiers.value.filter((n) => n.id !== cfg.id)
}

async function onNotifierSaved() {
  await fetchNotifiers()
}

onMounted(async () => {
  await Promise.all([fetchNotifiers(), fetchApiClients()])
})
</script>

<style scoped>
.settings-page {
  max-width: 1100px;
  margin: 0 auto;
}

.settings-title {
  margin: 0 0 2rem;
  font-size: 1.6rem;
  font-weight: 700;
  color: var(--text);
}

.settings-card {
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.5rem;
  box-shadow: var(--shadow), var(--shadow-glow);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  transition: border-color 0.2s;
}

.settings-card:hover {
  border-color: var(--border-bright);
}

.section-title {
  margin: 0 0 1rem;
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-muted);
}

.section-subtitle {
  margin: -0.5rem 0 1.5rem;
  font-size: 0.85rem;
  color: var(--text-muted);
}

.settings-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.field {
  margin-bottom: 0;
}

.field label {
  display: block;
  margin-bottom: 0.3rem;
  font-weight: 500;
  font-size: 0.9rem;
  color: var(--text-muted);
}

.field input {
  width: 100%;
  padding: 0.5rem 0.75rem;
  background: rgba(10, 16, 42, 0.8);
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 1rem;
  color: var(--text);
  font-family: inherit;
  transition: border-color 0.2s;
}

.field input:focus {
  outline: none;
  border-color: var(--border-bright);
  box-shadow: 0 0 0 3px rgba(129, 140, 248, 0.12);
}

.form-error {
  margin: 0;
  font-size: 0.85rem;
  color: #f87171;
}

.form-success {
  margin: 0;
  font-size: 0.85rem;
  color: #4ade80;
}

.submit-btn {
  margin-top: 0.25rem;
  padding: 0.5rem 1.1rem;
  background: linear-gradient(135deg, #4f46e5, #7c3aed);
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 0.9rem;
  font-weight: 600;
  transition: opacity 0.2s, box-shadow 0.2s;
  box-shadow: 0 0 16px rgba(129, 140, 248, 0.3);
}

.submit-btn:hover {
  opacity: 0.9;
  box-shadow: 0 0 24px rgba(129, 140, 248, 0.45);
}

.submit-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

/* Notifiers */

.credentials-card {
  margin-bottom: 1.5rem;
}

.api-clients-card {
  margin-bottom: 1.5rem;
}

.token-error {
  margin-bottom: 0.8rem;
}

.add-api-client-btn {
  margin-top: 0;
  white-space: nowrap;
  flex-shrink: 0;
}

.empty-api-clients {
  font-size: 0.88rem;
  color: var(--text-muted);
  margin: 0;
}

.api-clients-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.api-client-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.6rem;
  padding: 0.65rem 0.9rem;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--border);
  border-radius: 8px;
}

.api-client-info {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 0.55rem;
  flex-wrap: wrap;
}

.api-client-name {
  font-size: 0.94rem;
  font-weight: 600;
  color: var(--text);
}

.api-client-prefix {
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.03em;
  color: #a5b4fc;
  background: rgba(99, 102, 241, 0.15);
  border: 1px solid rgba(99, 102, 241, 0.33);
  border-radius: 4px;
  padding: 0.12rem 0.4rem;
}

.api-client-last-used {
  font-size: 0.8rem;
  color: var(--text-muted);
}

.notifiers-card {
  margin-top: 0;
}

.card-header-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.card-header-row .section-title {
  margin-bottom: 0.25rem;
}

.card-header-row .section-subtitle {
  margin: 0;
}

.add-notifier-btn {
  margin-top: 0;
  white-space: nowrap;
  flex-shrink: 0;
}

.empty-notifiers {
  font-size: 0.88rem;
  color: var(--text-muted);
  margin: 0;
}

.muted {
  font-size: 0.88rem;
  color: var(--text-muted);
  margin: 0;
}

.notifiers-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.notifier-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.65rem 0.9rem;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--border);
  border-radius: 8px;
  transition: border-color 0.15s;
}

.notifier-row:hover {
  border-color: var(--border-bright);
}

.notifier-info {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.notifier-name {
  font-size: 0.95rem;
  font-weight: 500;
  color: var(--text);
}

.notifier-type-badge {
  font-size: 0.72rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  background: rgba(129, 140, 248, 0.15);
  color: #818cf8;
  border: 1px solid rgba(129, 140, 248, 0.3);
  border-radius: 4px;
  padding: 0.1rem 0.4rem;
}

.notifier-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

/* Toggle switch */
.toggle {
  position: relative;
  display: inline-flex;
  align-items: center;
  cursor: pointer;
}

.toggle input {
  position: absolute;
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  width: 36px;
  height: 20px;
  background: rgba(255, 255, 255, 0.12);
  border-radius: 10px;
  position: relative;
  transition: background 0.2s;
}

.toggle-slider::after {
  content: '';
  position: absolute;
  top: 3px;
  left: 3px;
  width: 14px;
  height: 14px;
  background: #fff;
  border-radius: 50%;
  transition: transform 0.2s;
}

.toggle input:checked + .toggle-slider {
  background: #6366f1;
}

.toggle input:checked + .toggle-slider::after {
  transform: translateX(16px);
}

.btn-icon {
  background: none;
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-muted);
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1rem;
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s;
  padding: 0;
}

.btn-icon.btn-edit:hover {
  color: #818cf8;
  border-color: #818cf8;
}

.btn-icon.btn-delete:hover {
  color: #f87171;
  border-color: #f87171;
}

@media (max-width: 760px) {
  .add-api-client-btn {
    width: 100%;
  }
}
</style>
