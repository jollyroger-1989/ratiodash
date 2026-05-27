<template>
  <BaseModal :model-value="modelValue" title-id="report-form-title" size="sm" @close="close">
    <template #title>
      {{ editMode ? $t('reports.modal.editTitle') : $t('reports.modal.addTitle') }}
    </template>

    <form @submit.prevent="submit">
          <div class="field">
            <label for="rf-name">{{ $t('common.name') }}</label>
            <input id="rf-name" v-model="form.name" required placeholder="e.g. Weekly digest" autocomplete="off" />
          </div>

          <div class="field">
            <label for="rf-schedule">{{ $t('reports.modal.schedule') }}</label>
            <CronSelect id="rf-schedule" v-model="form.cron_expr" />
          </div>

          <div class="field">
            <label>{{ $t('reports.modal.notifiers') }}</label>
            <p v-if="!notifiers.length" class="muted-small">{{ $t('reports.modal.noNotifiers') }}</p>
            <div v-else class="notifiers-toggle-group">
              <NotifierToggleButton
                v-for="cfg in notifiers"
                :key="cfg.id"
                :name="cfg.name"
                :type="cfg.type"
                :selected="form.notifier_config_ids.includes(cfg.id)"
                @toggle="toggleNotifier(cfg.id)"
              />
            </div>
          </div>

          <p v-if="formError" class="form-error">{{ formError }}</p>

          <div class="form-actions">
            <button type="submit" class="btn-primary" :disabled="saving">
              {{ saving
                ? (editMode ? $t('reports.modal.saving') : $t('reports.modal.adding'))
                : (editMode ? $t('reports.modal.save') : $t('reports.modal.add')) }}
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
import type { Report, NotifierConfig, CreateReportInput, UpdateReportInput } from '@/services/api'
import BaseModal from '@/components/BaseModal.vue'
import CronSelect from '@/components/CronSelect.vue'
import NotifierToggleButton from '@/components/NotifierToggleButton.vue'

const props = defineProps<{
  modelValue: boolean
  notifiers: NotifierConfig[]
  existing?: Report | null
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
  cron_expr: string
  notifier_config_ids: number[]
}

const form = ref<FormState>({ name: '', cron_expr: '@daily', notifier_config_ids: [] })

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
        cron_expr: props.existing.cron_expr,
        notifier_config_ids: props.existing.notifier_configs?.map((c) => c.id) ?? [],
      }
    } else {
      editMode.value = false
      form.value = { name: '', cron_expr: '@daily', notifier_config_ids: [] }
    }
  }
)

function toggleNotifier(id: number) {
  const ids = form.value.notifier_config_ids
  const idx = ids.indexOf(id)
  if (idx === -1) ids.push(id)
  else ids.splice(idx, 1)
}

import { useReportsStore } from '@/stores/reports'

const store = useReportsStore()

async function submit() {
  saving.value = true
  formError.value = ''
  try {
    if (editMode.value && props.existing) {
      const input: UpdateReportInput = {
        name: form.value.name,
        cron_expr: form.value.cron_expr,
        notifier_config_ids: form.value.notifier_config_ids,
      }
      await store.update(props.existing.id, input)
    } else {
      const input: CreateReportInput = {
        name: form.value.name,
        cron_expr: form.value.cron_expr,
        notifier_config_ids: form.value.notifier_config_ids,
      }
      await store.create(input)
    }
    emit('saved')
    close()
  } catch (e: any) {
    formError.value = e?.response?.data?.detail ?? e?.message ?? 'Failed to save report.'
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
</style>
