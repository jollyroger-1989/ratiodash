<template>
  <BaseModal :model-value="modelValue" title-id="alert-form-title" size="sm" @close="close">
    <template #title>
      {{ editMode ? $t('alerts.modal.editTitle') : $t('alerts.modal.addTitle') }}
    </template>

    <form @submit.prevent="submit">
      <div class="field">
        <label for="af-name">{{ $t('common.name') }}</label>
        <input id="af-name" v-model="form.name" required placeholder="e.g. Low ratio warning" autocomplete="off" />
      </div>

      <div class="field">
        <label for="af-type">{{ $t('alerts.modal.alertType') }}</label>
        <select id="af-type" v-model="form.alert_type" :disabled="editMode" required>
          <option value="ratio_alert">{{ $t('alerts.modal.ratioAlert') }}</option>
          <option value="sync_error">{{ $t('alerts.modal.syncError') }}</option>
        </select>
      </div>

      <div v-if="form.alert_type === 'ratio_alert'" class="field">
        <label for="af-threshold">{{ $t('alerts.modal.threshold') }}</label>
        <input id="af-threshold" v-model.number="form.ratio_threshold" type="number" min="0.01" step="any" />
      </div>

      <div class="field-checkbox">
        <label>
          <input type="checkbox" v-model="form.enabled" />
          Enabled
        </label>
      </div>

      <div class="field-checkbox">
        <label>
          <input type="checkbox" v-model="form.all_trackers" />
          {{ $t('alerts.modal.allTrackers') }}
        </label>
      </div>

      <div v-if="!form.all_trackers" class="field">
        <label>{{ $t('alerts.modal.trackers') }}</label>
        <div class="notifiers-toggle-group">
          <NotifierToggleButton
            v-for="tracker in trackers"
            :key="tracker.id"
            :name="tracker.name"
            :type="tracker.scraper_key"
            :selected="form.tracker_ids.includes(tracker.id)"
            @toggle="toggleTracker(tracker.id)"
          />
        </div>
      </div>

      <div class="field">
        <label>{{ $t('alerts.modal.notifiers') }}</label>
        <p v-if="!notifiers.length" class="muted-small">{{ $t('alerts.modal.noNotifiers') }}</p>
        <div v-else class="notifiers-toggle-group">
          <NotifierToggleButton
            v-for="nc in notifiers"
            :key="nc.id"
            :name="nc.name"
            :type="nc.type"
            :selected="form.notifier_config_ids.includes(nc.id)"
            @toggle="toggleNotifier(nc.id)"
          />
        </div>
      </div>

      <p v-if="formError" class="form-error">{{ formError }}</p>

      <div class="form-actions">
        <button type="submit" class="btn-primary" :disabled="saving">
          {{ saving
            ? (editMode ? $t('alerts.modal.saving') : $t('alerts.modal.adding'))
            : (editMode ? $t('alerts.modal.save') : $t('alerts.modal.add')) }}
        </button>
        <button type="button" class="btn-secondary" @click="close">
          {{ $t('common.cancel') }}
        </button>
      </div>
    </form>
  </BaseModal>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { AlertConfig, Tracker, NotifierConfig, CreateAlertConfigInput, UpdateAlertConfigInput } from '@/services/api'
import BaseModal from '@/components/BaseModal.vue'
import NotifierToggleButton from '@/components/NotifierToggleButton.vue'
import { useAlertConfigsStore } from '@/stores/alertConfigs'

const props = defineProps<{
  modelValue: boolean
  notifiers: NotifierConfig[]
  trackers: Tracker[]
  existing?: AlertConfig | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', val: boolean): void
  (e: 'saved'): void
}>()

const editMode = ref(false)
const saving = ref(false)
const formError = ref('')

interface FormState {
  name: string
  alert_type: string
  enabled: boolean
  ratio_threshold: number
  all_trackers: boolean
  tracker_ids: number[]
  notifier_config_ids: number[]
}

const form = ref<FormState>({
  name: '',
  alert_type: 'ratio_alert',
  enabled: true,
  ratio_threshold: 1.5,
  all_trackers: true,
  tracker_ids: [],
  notifier_config_ids: [],
})

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    formError.value = ''
    saving.value = false
    if (props.existing) {
      editMode.value = true
      form.value = {
        name: props.existing.name,
        alert_type: props.existing.alert_type,
        enabled: props.existing.enabled,
        ratio_threshold: props.existing.ratio_threshold,
        all_trackers: props.existing.all_trackers,
        tracker_ids: props.existing.trackers?.map((t) => t.id) ?? [],
        notifier_config_ids: props.existing.notifier_configs?.map((c) => c.id) ?? [],
      }
    } else {
      editMode.value = false
      form.value = {
        name: '',
        alert_type: 'ratio_alert',
        enabled: true,
        ratio_threshold: 1.5,
        all_trackers: true,
        tracker_ids: [],
        notifier_config_ids: [],
      }
    }
  }
)

function toggleNotifier(id: number) {
  const ids = form.value.notifier_config_ids
  const idx = ids.indexOf(id)
  if (idx === -1) ids.push(id)
  else ids.splice(idx, 1)
}

function toggleTracker(id: number) {
  const ids = form.value.tracker_ids
  const idx = ids.indexOf(id)
  if (idx === -1) ids.push(id)
  else ids.splice(idx, 1)
}

const store = useAlertConfigsStore()

async function submit() {
  saving.value = true
  formError.value = ''
  try {
    if (editMode.value && props.existing) {
      const input: UpdateAlertConfigInput = {
        name: form.value.name,
        enabled: form.value.enabled,
        ratio_threshold: form.value.ratio_threshold,
        all_trackers: form.value.all_trackers,
        tracker_ids: form.value.tracker_ids,
        notifier_config_ids: form.value.notifier_config_ids,
      }
      await store.update(props.existing.id, input)
    } else {
      const input: CreateAlertConfigInput = {
        name: form.value.name,
        alert_type: form.value.alert_type,
        enabled: form.value.enabled,
        ratio_threshold: form.value.ratio_threshold,
        all_trackers: form.value.all_trackers,
        tracker_ids: form.value.tracker_ids,
        notifier_config_ids: form.value.notifier_config_ids,
      }
      await store.create(input)
    }
    emit('saved')
    close()
  } catch (e: any) {
    formError.value = e?.response?.data?.detail ?? e?.message ?? 'Failed to save alert.'
  } finally {
    saving.value = false
  }
}

function close() {
  emit('update:modelValue', false)
}
</script>

<style scoped>
.notifiers-toggle-group {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.field-checkbox {
  margin-bottom: 1rem;
}

.field-checkbox label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  justify-content: flex-start;
}

.field-checkbox input[type='checkbox'] {
  width: auto;
  flex-shrink: 0;
}
</style>
