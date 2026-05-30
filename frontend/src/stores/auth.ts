import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '@/services/api'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem('auth_token'))
  const setupStatus = ref<boolean | null>(null)

  const isAuthenticated = computed(() => !!token.value)

  function setToken(newToken: string) {
    token.value = newToken
    localStorage.setItem('auth_token', newToken)
  }

  function logout() {
    token.value = null
    localStorage.removeItem('auth_token')
  }

  async function login(username: string, password: string): Promise<void> {
    const response = await authApi.login(username, password)
    setToken(response.token)
  }

  async function setup(username: string, password: string): Promise<void> {
    await authApi.setup(username, password)
    setupStatus.value = true
  }

  async function isSetup(forceRefresh = false): Promise<boolean> {
    if (!forceRefresh && setupStatus.value !== null) {
      return setupStatus.value
    }

    const status = await authApi.status()
    setupStatus.value = status.setup
    return status.setup
  }

  return { token, isAuthenticated, login, logout, setup, isSetup }
})
