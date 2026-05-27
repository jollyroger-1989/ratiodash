import { defineStore } from 'pinia'
import { ref } from 'vue'
import { reportsApi, type Report, type CreateReportInput, type UpdateReportInput } from '@/services/api'

export const useReportsStore = defineStore('reports', () => {
  const reports = ref<Report[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchAll() {
    loading.value = true
    error.value = null
    try {
      reports.value = await reportsApi.getAll()
    } catch {
      error.value = 'Failed to load reports.'
    } finally {
      loading.value = false
    }
  }

  async function create(input: CreateReportInput) {
    error.value = null
    try {
      const report = await reportsApi.create(input)
      reports.value.push(report)
      return report
    } catch (e: any) {
      error.value = e?.response?.data?.detail ?? e?.message ?? 'Failed to create report.'
      throw e
    }
  }

  async function update(id: number, input: UpdateReportInput) {
    error.value = null
    try {
      const updated = await reportsApi.update(id, input)
      const idx = reports.value.findIndex((r) => r.id === id)
      if (idx !== -1) reports.value[idx] = updated
      return updated
    } catch (e: any) {
      error.value = e?.response?.data?.detail ?? e?.message ?? 'Failed to update report.'
      throw e
    }
  }

  async function remove(id: number) {
    await reportsApi.remove(id)
    reports.value = reports.value.filter((r) => r.id !== id)
  }

  async function send(id: number) {
    await reportsApi.send(id)
    // Refresh the single report so last_sent_at updates in the UI.
    const fresh = await reportsApi.getById(id)
    const idx = reports.value.findIndex((r) => r.id === id)
    if (idx !== -1) reports.value[idx] = fresh
  }

  return { reports, loading, error, fetchAll, create, update, remove, send }
})
