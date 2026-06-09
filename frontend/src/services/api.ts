import axios from 'axios'

const http = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' }
})

// Attach the stored JWT to every request.
http.interceptors.request.use((config) => {
  const token = localStorage.getItem('auth_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// On 401, clear the token and redirect to login so the user is re-authenticated.
http.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      const path = window.location.pathname
      if (path !== '/login' && path !== '/setup') {
        localStorage.removeItem('auth_token')
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  }
)

// ---- Scraper types ----

export interface CredentialField {
  key: string
  label: string
  type: 'text' | 'password'
  required: boolean
}

export interface ScraperDef {
  key: string
  credential_fields: CredentialField[]
}

// ---- Tracker types ----

export interface Tracker {
  id: number
  name: string
  scraper_key: string
  cron_expr: string
  active: boolean
  last_error: string
  last_scraped_at: string | null
  created_at: string
  updated_at: string
  stats?: TrackerStats | null
  public_credentials?: Record<string, string>
}

export interface CreateTrackerInput {
  name: string
  scraper_key: string
  credentials?: string
  cron_expr?: string
  active?: boolean
}

export interface UpdateTrackerInput {
  name?: string
  scraper_key?: string
  credentials?: string
  cron_expr?: string
  active?: boolean
}

// ---- Stats types ----

export interface TrackerStats {
  id: number
  tracker_id: number
  uploaded: number
  downloaded: number
  ratio: number
  fetched_at: string
}

export interface GlobalStatsPoint {
  uploaded: number
  downloaded: number
  ratio: number
  fetched_at: string
}

export interface DashboardEntry {
  tracker: Tracker
  stats: TrackerStats | null
}

// ---- API helpers ----

export type SortBy = 'ratio' | 'uploaded' | 'downloaded'
export type SortOrder = 'asc' | 'desc'

export interface ListTrackersParams {
  sort_by?: SortBy
  sort_order?: SortOrder
}

export const trackersApi = {
  getAll(params?: ListTrackersParams): Promise<Tracker[]> {
    return http.get<Tracker[]>('/trackers', { params }).then((r) => r.data)
  },

  getById(id: number): Promise<Tracker> {
    return http.get<Tracker>(`/trackers/${id}`).then((r) => r.data)
  },

  create(input: CreateTrackerInput): Promise<Tracker> {
    return http.post<Tracker>('/trackers', input).then((r) => r.data)
  },

  update(id: number, input: UpdateTrackerInput): Promise<Tracker> {
    return http.patch<Tracker>(`/trackers/${id}`, input).then((r) => r.data)
  },

  remove(id: number): Promise<void> {
    return http.delete(`/trackers/${id}`).then(() => undefined)
  },

  refresh(id: number): Promise<TrackerStats> {
    return http.post<TrackerStats>(`/trackers/${id}/refresh`).then((r) => r.data)
  },

  /** Test tracker credentials for a scraper before saving. */
  test(scraperKey: string, credentials: string): Promise<void> {
    return http.post('/trackers/test', { scraper_key: scraperKey, credentials }).then(() => undefined)
  },

  /** Test a saved tracker by ID, optionally merging current form values over stored credentials. */
  testByID(id: number, credentialsOverride?: string): Promise<void> {
    return http.post(`/trackers/${id}/test`, { credentials: credentialsOverride ?? '' }).then(() => undefined)
  }
}

export const statsApi = {
  getDashboard(): Promise<DashboardEntry[]> {
    return http.get<DashboardEntry[]>('/trackers/stats').then((r) => r.data)
  },

  getHistory(trackerId: number, params?: { startDate?: string; limit?: number }): Promise<TrackerStats[]> {
    const p: Record<string, string | number> = {}
    if (params?.startDate) p.start_date = params.startDate
    if (params?.limit !== undefined) p.limit = params.limit
    return http
      .get<TrackerStats[]>(`/trackers/${trackerId}/stats`, { params: Object.keys(p).length ? p : undefined })
      .then((r) => r.data)
  },

  getLatest(trackerId: number): Promise<TrackerStats> {
    return http.get<TrackerStats>(`/trackers/${trackerId}/stats/latest`).then((r) => r.data)
  },

  getGlobalHistory(limit?: number): Promise<GlobalStatsPoint[]> {
    const params = typeof limit === 'number' ? { limit } : undefined
    return http.get<GlobalStatsPoint[]>('/stats/global', { params }).then((r) => r.data)
  },

  deleteEntry(trackerId: number, statId: number): Promise<void> {
    return http.delete(`/trackers/${trackerId}/stats/${statId}`).then(() => undefined)
  }
}

// ---- Item types ----

export interface Item {
  id: number
  name: string
  value: number
  created_at: string
  updated_at: string
}

export interface CreateItemInput {
  name: string
  value: number
}

export interface UpdateItemInput {
  name?: string
  value?: number
}

// ---- Item API helpers ----

export const itemsApi = {
  getAll(): Promise<Item[]> {
    return http.get<Item[]>('/items').then((r) => r.data)
  },

  getById(id: number): Promise<Item> {
    return http.get<Item>(`/items/${id}`).then((r) => r.data)
  },

  create(input: CreateItemInput): Promise<Item> {
    return http.post<Item>('/items', input).then((r) => r.data)
  },

  update(id: number, input: UpdateItemInput): Promise<Item> {
    return http.patch<Item>(`/items/${id}`, input).then((r) => r.data)
  },

  remove(id: number): Promise<void> {
    return http.delete(`/items/${id}`).then(() => undefined)
  }
}

export const scrapersApi = {
  getAll(): Promise<ScraperDef[]> {
    return http.get<ScraperDef[]>('/scrapers').then((r) => r.data)
  }
}

// ---- Auth types ----

export interface AuthStatus {
  setup: boolean
}

export interface LoginResponse {
  token: string
}

// ---- Auth API helpers ----
// These calls bypass the JWT interceptor intentionally — the endpoints are public.

export const authApi = {
  status(): Promise<AuthStatus> {
    return http.get<AuthStatus>('/auth/status').then((r) => r.data)
  },

  setup(username: string, password: string): Promise<void> {
    return http.post('/auth/setup', { username, password }).then(() => undefined)
  },

  login(username: string, password: string): Promise<LoginResponse> {
    return http.post<LoginResponse>('/auth/login', { username, password }).then((r) => r.data)
  }
}

export const settingsApi = {
  updateCredentials(currentPassword: string, newUsername: string, newPassword: string): Promise<void> {
    return http
      .patch('/settings/credentials', { current_password: currentPassword, new_username: newUsername, new_password: newPassword })
      .then(() => undefined)
  },

  getLanguage(): Promise<string> {
    return http.get<{ language: string }>('/settings/language').then((r) => r.data.language)
  },

  updateLanguage(language: string): Promise<void> {
    return http.patch('/settings/language', { language }).then(() => undefined)
  }
}

// ---- API client types ----

export interface APIClient {
  id: number
  name: string
  key_prefix: string
  enabled: boolean
  last_used_at: string | null
  created_at: string
  updated_at: string
}

export interface CreateAPIClientResponse {
  client: APIClient
  api_key: string
}

// ---- API client API helpers ----

export const apiClientsApi = {
  getAll(): Promise<APIClient[]> {
    return http.get<APIClient[]>('/api-clients').then((r) => r.data)
  },

  create(name: string): Promise<CreateAPIClientResponse> {
    return http.post<CreateAPIClientResponse>('/api-clients', { name }).then((r) => r.data)
  },

  remove(id: number): Promise<void> {
    return http.delete(`/api-clients/${id}`).then(() => undefined)
  }
}

// ---- Notifier config types ----

export interface NotifierConfigField {
  key: string
  label: string
  type: 'text' | 'password' | 'url' | 'select'
  required: boolean
  options?: string[]
}

export interface NotifierTypeInfo {
  key: string
  label: string
  config_fields: NotifierConfigField[]
}

export interface NotifierConfig {
  id: number
  name: string
  type: string
  enabled: boolean
  created_at: string
  updated_at: string
  public_config?: Record<string, string>
}

export interface CreateNotifierConfigInput {
  name: string
  type: string
  config: string
}

export interface UpdateNotifierConfigInput {
  name?: string
  config?: string
  enabled?: boolean
}

// ---- Notifier config API helpers ----

export const notifierConfigsApi = {
  getTypes(): Promise<NotifierTypeInfo[]> {
    return http.get<NotifierTypeInfo[]>('/notifier-types').then((r) => r.data)
  },

  getAll(): Promise<NotifierConfig[]> {
    return http.get<NotifierConfig[]>('/notifier-configs').then((r) => r.data)
  },

  create(input: CreateNotifierConfigInput): Promise<NotifierConfig> {
    return http.post<NotifierConfig>('/notifier-configs', input).then((r) => r.data)
  },

  update(id: number, input: UpdateNotifierConfigInput): Promise<NotifierConfig> {
    return http.patch<NotifierConfig>(`/notifier-configs/${id}`, input).then((r) => r.data)
  },

  remove(id: number): Promise<void> {
    return http.delete(`/notifier-configs/${id}`).then(() => undefined)
  },

  /** Test a notifier config before saving (provide type + config JSON directly). */
  test(type: string, config: string): Promise<void> {
    return http.post('/notifier-configs/test', { type, config }).then(() => undefined)
  },

  /** Test a saved notifier config by ID, optionally merging current form values over stored credentials. */
  testByID(id: number, configOverride?: string): Promise<void> {
    return http.post(`/notifier-configs/${id}/test`, { config: configOverride ?? '' }).then(() => undefined)
  }
}

// ---- Report types ----

export interface Report {
  id: number
  name: string
  cron_expr: string
  last_sent_at: string | null
  notifier_configs: NotifierConfig[]
  created_at: string
  updated_at: string
}

export interface CreateReportInput {
  name: string
  cron_expr: string
  notifier_config_ids?: number[]
}

export interface UpdateReportInput {
  name?: string
  cron_expr?: string
  notifier_config_ids?: number[]
}

// ---- Report API helpers ----

export const reportsApi = {
  getAll(): Promise<Report[]> {
    return http.get<Report[]>('/reports').then((r) => r.data)
  },

  getById(id: number): Promise<Report> {
    return http.get<Report>(`/reports/${id}`).then((r) => r.data)
  },

  create(input: CreateReportInput): Promise<Report> {
    return http.post<Report>('/reports', input).then((r) => r.data)
  },

  update(id: number, input: UpdateReportInput): Promise<Report> {
    return http.patch<Report>(`/reports/${id}`, input).then((r) => r.data)
  },

  remove(id: number): Promise<void> {
    return http.delete(`/reports/${id}`).then(() => undefined)
  },

  send(id: number): Promise<void> {
    return http.post(`/reports/${id}/send`).then(() => undefined)
  }
}

// ---- Alert Config types ----

export interface AlertConfig {
  id: number
  name: string
  alert_type: string
  enabled: boolean
  ratio_threshold: number
  all_trackers: boolean
  trackers: Tracker[]
  notifier_configs: NotifierConfig[]
  created_at: string
  updated_at: string
}

export interface CreateAlertConfigInput {
  name: string
  alert_type: string
  enabled?: boolean
  ratio_threshold?: number
  all_trackers?: boolean
  tracker_ids?: number[]
  notifier_config_ids?: number[]
}

export interface UpdateAlertConfigInput {
  name?: string
  enabled?: boolean
  ratio_threshold?: number
  all_trackers?: boolean
  tracker_ids?: number[]
  notifier_config_ids?: number[]
}

// ---- Alert Config API helpers ----

export const alertConfigsApi = {
  getAll(): Promise<AlertConfig[]> {
    return http.get<AlertConfig[]>('/alert-configs').then((r) => r.data)
  },

  getById(id: number): Promise<AlertConfig> {
    return http.get<AlertConfig>(`/alert-configs/${id}`).then((r) => r.data)
  },

  create(input: CreateAlertConfigInput): Promise<AlertConfig> {
    return http.post<AlertConfig>('/alert-configs', input).then((r) => r.data)
  },

  update(id: number, input: UpdateAlertConfigInput): Promise<AlertConfig> {
    return http.patch<AlertConfig>(`/alert-configs/${id}`, input).then((r) => r.data)
  },

  remove(id: number): Promise<void> {
    return http.delete(`/alert-configs/${id}`).then(() => undefined)
  }
}
