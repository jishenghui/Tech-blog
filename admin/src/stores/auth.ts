import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../lib/api'
import { clearAccessToken, refreshAccessToken, setAccessToken } from '../lib/session'
import type { User } from '../lib/types'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const initialized = ref(false)
  const loading = ref(false)

  async function login(username: string, password: string): Promise<void> {
    loading.value = true
    try {
      const data = await api.post<{ access_token: string; user: User }>('/api/v1/auth/login', {
        username,
        password,
      })
      setAccessToken(data.access_token)
      user.value = data.user
      initialized.value = true
    } finally {
      loading.value = false
    }
  }

  async function bootstrap(): Promise<void> {
    if (initialized.value) {
      return
    }
    initialized.value = true
    const refreshed = await refreshAccessToken()
    if (!refreshed) {
      return
    }
    try {
      user.value = await api.get<User>('/api/v1/auth/me')
    } catch {
      reset()
    }
  }

  async function logout(): Promise<void> {
    try {
      await api.post('/api/v1/auth/logout')
    } catch {
      // 已过期也直接清理本地状态。
    }
    reset()
  }

  function reset(): void {
    clearAccessToken()
    user.value = null
    initialized.value = true
  }

  return { user, initialized, loading, login, bootstrap, logout, reset }
})
