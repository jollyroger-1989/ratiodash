<template>
  <BaseModal :model-value="modelValue" title-id="tracker-form-title" size="lg" @close="close">
    <template #title>
      {{ editMode ? $t('trackers.modal.editTitle') : $t('trackers.modal.title') }}
    </template>

    <form @submit.prevent="submit">
          <div class="fields-row">
            <div class="field">
              <label for="tf-name">{{ $t('common.name') }}</label>
              <input id="tf-name" v-model="form.name" required placeholder="e.g. MyTracker" />
            </div>
          </div>
          <div class="fields-row">
            <div class="field">
              <label for="tf-scraper">{{ $t('trackers.modal.scraper') }}</label>
              <select
                id="tf-scraper"
                v-model="form.scraper_key"
                :disabled="editMode || loadingScrapers"
                required
                @change="onScraperChange"
              >
                <option v-if="loadingScrapers" value="" disabled>
                  {{ $t('trackers.modal.loadingScrapers') }}
                </option>
                <option v-for="s in scrapers" :key="s.key" :value="s.key">{{ s.key }}</option>
              </select>
            </div>
            <div class="field">
              <label for="tf-cron">{{ $t('trackers.modal.schedule') }}</label>
              <CronSelect id="tf-cron" v-model="form.cron_expr" />
            </div>
          </div>
          <div
            v-if="selectedScraper && selectedScraper.credential_fields.length"
            class="fields-row"
          >
            <div v-for="field in selectedScraper.credential_fields" :key="field.key" class="field">
              <label :for="'tf-cred-' + field.key">
                {{ field.label }}
                <span v-if="editMode && field.type === 'password'" class="field-optional"> ({{ $t('common.optional') }})</span>
              </label>
              <input
                :id="'tf-cred-' + field.key"
                v-model="credentialValues[field.key]"
                :type="field.type"
                :required="field.required && !(editMode && field.type === 'password')"
                :placeholder="editMode ? $t('trackers.modal.credentialPlaceholder') : ''"
                autocomplete="off"
              />
            </div>
          </div>
          <div class="form-actions">
            <button type="submit" class="btn-primary" :disabled="saving || testing">
              {{ saving
                ? (editMode ? $t('trackers.modal.saving') : $t('trackers.modal.adding'))
                : (editMode ? $t('trackers.modal.save') : $t('trackers.modal.addSite')) }}
            </button>
            <button type="button" class="btn-test" :disabled="testing || saving" @click.prevent="runTest">
              {{ testing ? $t('trackers.modal.testing') : $t('trackers.modal.test') }}
            </button>
            <button type="button" class="btn-secondary" @click="close">
              {{ $t('common.cancel') }}
            </button>
          </div>
          <p v-if="testStatus === 'ok'" class="form-success">{{ $t('trackers.modal.testOk') }}</p>
          <p v-if="testStatus === 'error'" class="form-error">{{ testError }}</p>
          <p v-if="formError" class="form-error">{{ formError }}</p>
    </form>
  </BaseModal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { scrapersApi, trackersApi, type Tracker, type ScraperDef } from '@/services/api'
import { useTrackersStore } from '@/stores/trackers'
import BaseModal from '@/components/BaseModal.vue'
import CronSelect from '@/components/CronSelect.vue'

const props = defineProps<{
  modelValue: boolean
  tracker?: Tracker
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'saved': [tracker: Tracker]
}>()

const { t } = useI18n()
const store = useTrackersStore()

const editMode = computed(() => props.tracker !== undefined)

const scrapers = ref<ScraperDef[]>([])
const loadingScrapers = ref(false)

const form = ref({ name: '', scraper_key: '', cron_expr: '@hourly' })
const credentialValues = ref<Record<string, string>>({})
const saving = ref(false)
const formError = ref('')
const testing = ref(false)
const testStatus = ref<null | 'ok' | 'error'>(null)
const testError = ref('')

const selectedScraper = computed(
  () => scrapers.value.find((s) => s.key === form.value.scraper_key) ?? null
)

watch(
  () => props.modelValue,
  async (open) => {
    if (!open) return
    formError.value = ''
    store.error = null

    if (!scrapers.value.length) {
      loadingScrapers.value = true
      try {
        scrapers.value = await scrapersApi.getAll()
      } finally {
        loadingScrapers.value = false
      }
    }

    initForm()
  }
)

function initForm() {
  if (props.tracker) {
    form.value = {
      name: props.tracker.name,
      scraper_key: props.tracker.scraper_key,
      cron_expr: props.tracker.cron_expr,
    }
    credentialValues.value = Object.fromEntries(
      (selectedScraper.value?.credential_fields ?? []).map((f) => [
        f.key,
        props.tracker!.public_credentials?.[f.key] ?? '',
      ])
    )
  } else {
    const first = scrapers.value[0] ?? null
    form.value = { name: '', scraper_key: first?.key ?? '', cron_expr: '@hourly' }
    credentialValues.value = Object.fromEntries(
      (first?.credential_fields ?? []).map((f) => [f.key, ''])
    )
  }
}

function onScraperChange() {
  credentialValues.value = Object.fromEntries(
    (selectedScraper.value?.credential_fields ?? []).map((f) => [f.key, ''])
  )
}

function buildCredentialsJson(): string {
  const filled = Object.fromEntries(
    Object.entries(credentialValues.value).filter(([, v]) => v !== '')
  )
  return JSON.stringify(filled)
}

function close() {
  emit('update:modelValue', false)
  formError.value = ''
  testStatus.value = null
  testError.value = ''
  store.error = null
}

async function runTest() {
  testing.value = true
  testStatus.value = null
  testError.value = ''
  try {
    if (editMode.value && props.tracker) {
      await trackersApi.testByID(props.tracker.id, buildCredentialsJson())
    } else {
      await trackersApi.test(form.value.scraper_key, buildCredentialsJson() || '{}')
    }
    testStatus.value = 'ok'
  } catch (e: any) {
    testStatus.value = 'error'
    testError.value = e?.response?.data?.detail ?? e?.message ?? t('trackers.modal.testError')
  } finally {
    testing.value = false
  }
}

async function submit() {
  saving.value = true
  formError.value = ''
  try {
    let saved: Tracker
    if (editMode.value && props.tracker) {
      const patch: Record<string, string | boolean> = {
        name: form.value.name,
        cron_expr: form.value.cron_expr,
      }
      const creds = buildCredentialsJson()
      if (creds !== '{}') patch.credentials = creds
      saved = await store.update(props.tracker.id, patch)
    } else {
      saved = await store.create({
        ...form.value,
        credentials: buildCredentialsJson(),
      })
    }
    emit('saved', saved)
    close()
  } catch (e: any) {
    formError.value = store.error ?? e?.message ?? t('detail.errorLoad')
  } finally {
    saving.value = false
  }
}
</script>
