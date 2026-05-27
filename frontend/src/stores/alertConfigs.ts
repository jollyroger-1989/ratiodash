import { defineStore } from 'pinia'
import { ref } from 'vue'
import { alertConfigsApi, type AlertConfig, type CreateAlertConfigInput, type UpdateAlertConfigInput } from '@/services/api'

export const useAlertConfigsStore = defineStore('alertConfigs', () => {
  const alertConfigs = ref<AlertConfig[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchAll() {
    loading.value = true
    error.value = null
    try {
      alertConfigs.value = await alertConfigsApi.getAll()
    } catch {
      error.value = 'Failed to load alert configs.'
    } finally {
      loading.value = false
    }
  }

  async function create(input: CreateAlertConfigInput) {
    error.value = null
    try {
      const config = await alertConfigsApi.create(input)
      alertConfigs.value.push(config)
      return config
    } catch (e: any) {
      error.value = e?.response?.data?.detail ?? e?.message ?? 'Failed to create alert config.'
      throw e
    }
  }

  async function update(id: number, input: UpdateAlertConfigInput) {
    error.value = null
    try {
      const updated = await alertConfigsApi.update(id, input)
      const idx = alertConfigs.value.findIndex((c) => c.id === id)
      if (idx !== -1) alertConfigs.value[idx] = updated
      return updated
    } catch (e: any) {
      error.value = e?.response?.data?.detail ?? e?.message ?? 'Failed to update alert config.'
      throw e
    }
  }

  async function remove(id: number) {
    await alertConfigsApi.remove(id)
    alertConfigs.value = alertConfigs.value.filter((c) => c.id !== id)
  }

  return { alertConfigs, loading, error, fetchAll, create, update, remove }
})
