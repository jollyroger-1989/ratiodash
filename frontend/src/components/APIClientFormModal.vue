<template>
  <BaseModal :model-value="modelValue" title-id="api-client-form-title" size="md" @close="close">
    <template #title>
      {{ $t('settings.apiClients.modal.title') }}
    </template>

    <form v-if="!createdApiKey" @submit.prevent="submit">
      <div class="field">
        <label for="api-client-name">{{ $t('settings.apiClients.nameLabel') }}</label>
        <input
          id="api-client-name"
          v-model="name"
          type="text"
          :placeholder="$t('settings.apiClients.namePlaceholder')"
          autocomplete="off"
          required
        />
      </div>

      <p v-if="error" class="form-error">{{ error }}</p>

      <div class="form-actions">
        <button type="submit" class="btn-primary" :disabled="saving">
          {{ saving ? $t('settings.apiClients.creating') : $t('settings.apiClients.create') }}
        </button>
        <button type="button" class="btn-secondary" @click="close">
          {{ $t('common.cancel') }}
        </button>
      </div>
    </form>

    <div v-else class="created-token-box" role="status" aria-live="polite">
      <p class="created-token-label">{{ $t('settings.apiClients.createdLabel') }}</p>
      <div class="created-token-row">
        <code class="created-token-value">{{ createdApiKey }}</code>
        <button class="btn-copy" type="button" :title="$t('settings.apiClients.copyToken')" @click="copyApiKey">
          {{ copiedApiKey ? $t('settings.apiClients.copied') : $t('settings.apiClients.copyShort') }}
        </button>
      </div>
      <p class="created-token-hint">{{ $t('settings.apiClients.createdHint') }}</p>

      <p v-if="error" class="form-error">{{ error }}</p>

      <div class="form-actions">
        <button type="button" class="btn-primary" @click="close">
          {{ $t('settings.apiClients.modal.done') }}
        </button>
      </div>
    </div>
  </BaseModal>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiClientsApi } from '@/services/api'
import BaseModal from '@/components/BaseModal.vue'

defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'saved': []
}>()

const { t } = useI18n()

const name = ref('')
const saving = ref(false)
const error = ref('')
const createdApiKey = ref('')
const copiedApiKey = ref(false)

watch(
  () => name.value,
  () => {
    if (error.value) error.value = ''
  }
)

watch(
  () => createdApiKey.value,
  () => {
    copiedApiKey.value = false
  }
)

function resetForm() {
  name.value = ''
  saving.value = false
  error.value = ''
  createdApiKey.value = ''
  copiedApiKey.value = false
}

function close() {
  emit('update:modelValue', false)
  resetForm()
}

async function submit() {
  error.value = ''
  const trimmed = name.value.trim()
  if (!trimmed) {
    error.value = t('settings.apiClients.nameRequired')
    return
  }

  saving.value = true
  try {
    const response = await apiClientsApi.create(trimmed)
    createdApiKey.value = response.api_key
    emit('saved')
  } catch {
    error.value = t('settings.apiClients.createError')
  } finally {
    saving.value = false
  }
}

async function copyApiKey() {
  if (!createdApiKey.value) return
  try {
    await navigator.clipboard.writeText(createdApiKey.value)
    copiedApiKey.value = true
  } catch {
    error.value = t('settings.apiClients.copyError')
  }
}
</script>

<style scoped>
.created-token-box {
  background: var(--accent-glow);
  border: 1px solid var(--border-bright);
  border-radius: 8px;
  padding: 0.8rem;
}

.created-token-label {
  margin: 0 0 0.45rem;
  font-size: 0.82rem;
  color: var(--text-muted);
}

.created-token-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.created-token-value {
  display: block;
  flex: 1;
  margin: 0;
  padding: 0.45rem 0.55rem;
  font-size: 0.78rem;
  border-radius: 6px;
  background: var(--input-bg);
  color: var(--text);
  overflow-x: auto;
}

.created-token-hint {
  margin: 0.55rem 0 0;
  font-size: 0.77rem;
  color: var(--text-muted);
}

.btn-copy {
  min-width: 74px;
  height: 30px;
  border: 1px solid var(--border-bright);
  border-radius: 6px;
  background: transparent;
  color: var(--ratio-good);
  font-size: 0.74rem;
  font-weight: 600;
  cursor: pointer;
}

.btn-copy:hover {
  color: var(--text);
  border-color: var(--ratio-good);
}

@media (max-width: 760px) {
  .created-token-row {
    flex-direction: column;
    align-items: stretch;
  }

  .btn-copy {
    width: 100%;
  }
}
</style>
