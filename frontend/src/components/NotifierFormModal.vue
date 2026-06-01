<template>
  <BaseModal :model-value="modelValue" title-id="notifier-form-title" size="md" @close="close">
    <template #title>
      {{ editMode ? $t('settings.notifiers.modal.editTitle') : $t('settings.notifiers.modal.addTitle') }}
    </template>

    <form @submit.prevent="submit">
          <div class="fields-row">
            <div class="field">
              <label for="nf-name">{{ $t('common.name') }}</label>
              <input id="nf-name" v-model="form.name" required placeholder="e.g. My alerts" autocomplete="off" />
            </div>
            <div class="field">
              <label for="nf-type">{{ $t('settings.notifiers.modal.type') }}</label>
              <select id="nf-type" v-model="form.type" :disabled="editMode" required @change="onTypeChange">
                <option value="" disabled>{{ $t('settings.notifiers.modal.selectType') }}</option>
                <option v-for="t in types" :key="t.key" :value="t.key">{{ t.label }}</option>
              </select>
            </div>
          </div>

          <div v-if="selectedType && selectedType.config_fields.length" class="fields-row fields-row--config">
            <div v-for="field in selectedType.config_fields" :key="field.key" class="field">
              <label :for="'nf-cfg-' + field.key">
                {{ field.label }}
                <span v-if="editMode && field.type === 'password'" class="field-optional">
                  ({{ $t('common.optional') }})
                </span>
              </label>
              <select
                v-if="field.type === 'select'"
                :id="'nf-cfg-' + field.key"
                v-model="configValues[field.key]"
                :required="field.required && !editMode"
              >
                <option value="" :disabled="field.required">Select...</option>
                <option v-for="opt in field.options ?? []" :key="opt" :value="opt">{{ opt }}</option>
              </select>
              <input
                v-else
                :id="'nf-cfg-' + field.key"
                v-model="configValues[field.key]"
                :type="field.type === 'password' ? 'password' : field.type === 'url' ? 'url' : 'text'"
                :required="field.required && !editMode"
                :placeholder="editMode && field.type === 'password' ? $t('settings.notifiers.modal.keepCurrent') : ''"
                autocomplete="off"
              />
            </div>
          </div>

          <p v-if="formError" class="form-error">{{ formError }}</p>
          <p v-if="testStatus === 'ok'" class="form-success">{{ $t('settings.notifiers.modal.testOk') }}</p>
          <p v-if="testStatus === 'error'" class="form-error">{{ testError }}</p>

          <div class="form-actions">
            <button type="submit" class="btn-primary" :disabled="saving || testing">
              {{ saving
                ? (editMode ? $t('settings.notifiers.modal.saving') : $t('settings.notifiers.modal.adding'))
                : (editMode ? $t('settings.notifiers.modal.save') : $t('settings.notifiers.modal.add')) }}
            </button>
            <button type="button" class="btn-test" :disabled="testing || saving" @click.prevent="runTest">
              {{ testing ? $t('settings.notifiers.modal.testing') : $t('settings.notifiers.modal.test') }}
            </button>
            <button type="button" class="btn-secondary" @click="close">
              {{ $t('common.cancel') }}
            </button>
          </div>
    </form>
  </BaseModal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { notifierConfigsApi, type NotifierConfig, type NotifierTypeInfo } from '@/services/api'
import BaseModal from '@/components/BaseModal.vue'

const props = defineProps<{
  modelValue: boolean
  types: NotifierTypeInfo[]
  existing?: NotifierConfig
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'saved': []
}>()

const { t } = useI18n()

const editMode = computed(() => props.existing !== undefined)

const form = ref({ name: '', type: '' })
const configValues = ref<Record<string, string>>({})
const saving = ref(false)
const formError = ref('')
const testing = ref(false)
const testStatus = ref<null | 'ok' | 'error'>(null)
const testError = ref('')

const selectedType = computed(
  () => props.types.find((t) => t.key === form.value.type) ?? null
)

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    formError.value = ''
    initForm()
  }
)

function initForm() {
  if (props.existing) {
    form.value = { name: props.existing.name, type: props.existing.type }
    const fields = props.types.find((t) => t.key === props.existing!.type)?.config_fields ?? []
    configValues.value = Object.fromEntries(
      fields.map((f) => [f.key, props.existing!.public_config?.[f.key] ?? ''])
    )
  } else {
    const first = props.types[0] ?? null
    form.value = { name: '', type: first?.key ?? '' }
    configValues.value = Object.fromEntries(
      (first?.config_fields ?? []).map((f) => [f.key, ''])
    )
  }
}

function onTypeChange() {
  configValues.value = Object.fromEntries(
    (selectedType.value?.config_fields ?? []).map((f) => [f.key, ''])
  )
}

function buildConfigJson(): string {
  const filled = Object.fromEntries(
    Object.entries(configValues.value).filter(([, v]) => v !== '')
  )
  return JSON.stringify(filled)
}

function close() {
  emit('update:modelValue', false)
  formError.value = ''
  testStatus.value = null
  testError.value = ''
}

async function runTest() {
  testing.value = true
  testStatus.value = null
  testError.value = ''
  try {
    if (editMode.value && props.existing) {
      await notifierConfigsApi.testByID(props.existing.id, buildConfigJson())
    } else {
      await notifierConfigsApi.test(form.value.type, buildConfigJson() || '{}')
    }
    testStatus.value = 'ok'
  } catch (e: any) {
    testStatus.value = 'error'
    testError.value = e?.response?.data?.detail ?? e?.message ?? t('settings.notifiers.modal.testError')
  } finally {
    testing.value = false
  }
}

async function submit() {
  saving.value = true
  formError.value = ''
  try {
    if (editMode.value && props.existing) {
      const patch: { name?: string; config?: string } = { name: form.value.name }
      const cfg = buildConfigJson()
      if (cfg !== '{}') patch.config = cfg
      await notifierConfigsApi.update(props.existing.id, patch)
    } else {
      await notifierConfigsApi.create({
        name: form.value.name,
        type: form.value.type,
        config: buildConfigJson() || '{}',
      })
    }
    emit('saved')
    close()
  } catch (e: any) {
    formError.value = e?.response?.data?.detail ?? e?.message ?? t('settings.notifiers.modal.errorSave')
  } finally {
    saving.value = false
  }
}
</script>
