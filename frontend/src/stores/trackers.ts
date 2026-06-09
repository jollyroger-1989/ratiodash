import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  trackersApi,
  type Tracker,
  type CreateTrackerInput,
  type UpdateTrackerInput,
  type DashboardEntry,
  type SortBy,
  type SortOrder
} from '@/services/api'

const STORAGE_SORT_BY = 'trackers:sort_by'
const STORAGE_SORT_ORDER = 'trackers:sort_order'

export const useTrackersStore = defineStore('trackers', () => {
  const trackers = ref<Tracker[]>([])
  const dashboard = ref<DashboardEntry[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const sortBy = ref<SortBy | ''>((localStorage.getItem(STORAGE_SORT_BY) as SortBy | '') || '')
  const sortOrder = ref<SortOrder>((localStorage.getItem(STORAGE_SORT_ORDER) as SortOrder) || 'asc')

  async function fetchAll() {
    loading.value = true
    error.value = null
    try {
      trackers.value = await trackersApi.getAll()
    } catch {
      error.value = 'Failed to load trackers.'
    } finally {
      loading.value = false
    }
  }

  async function fetchDashboard() {
    loading.value = true
    error.value = null
    try {
      const params = sortBy.value
        ? { sort_by: sortBy.value, sort_order: sortOrder.value }
        : undefined
      const allTrackers = await trackersApi.getAll(params)
      dashboard.value = allTrackers.map((tracker) => ({
        tracker,
        stats: tracker.stats ?? null
      }))
    } catch {
      error.value = 'Failed to load dashboard.'
    } finally {
      loading.value = false
    }
  }

  async function setSort(by: SortBy | '', order: SortOrder) {
    sortBy.value = by
    sortOrder.value = order
    if (by) {
      localStorage.setItem(STORAGE_SORT_BY, by)
      localStorage.setItem(STORAGE_SORT_ORDER, order)
    } else {
      localStorage.removeItem(STORAGE_SORT_BY)
      localStorage.removeItem(STORAGE_SORT_ORDER)
    }
    await fetchDashboard()
  }

  async function create(input: CreateTrackerInput) {
    error.value = null
    try {
      const tracker = await trackersApi.create(input)
      trackers.value.push(tracker)
      return tracker
    } catch (e: any) {
      error.value = e?.response?.data?.detail ?? e?.message ?? 'Failed to create tracker.'
      throw e
    }
  }

  async function update(id: number, input: UpdateTrackerInput) {
    error.value = null
    try {
      const updated = await trackersApi.update(id, input)
      const idx = trackers.value.findIndex((t) => t.id === id)
      if (idx !== -1) trackers.value[idx] = updated
      const didx = dashboard.value.findIndex((e) => e.tracker.id === id)
      if (didx !== -1) dashboard.value[didx] = { ...dashboard.value[didx], tracker: updated }
      return updated
    } catch (e: any) {
      error.value = e?.response?.data?.detail ?? e?.message ?? 'Failed to update tracker.'
      throw e
    }
  }

  async function remove(id: number) {
    await trackersApi.remove(id)
    trackers.value = trackers.value.filter((t) => t.id !== id)
    dashboard.value = dashboard.value.filter((e) => e.tracker.id !== id)
  }

  async function refresh(id: number) {
    await trackersApi.refresh(id)
    await fetchDashboard()
  }

  return { trackers, dashboard, loading, error, sortBy, sortOrder, fetchAll, fetchDashboard, setSort, create, update, remove, refresh }
})
