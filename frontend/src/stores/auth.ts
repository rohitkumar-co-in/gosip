import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi, setupApi, type User } from '@/api/auth'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const initialized = ref(false)
  const setupCompleted = ref(true)
  const loading = ref(false)
  const error = ref<string | null>(null)
  let authCheck: Promise<void> | null = null

  const isAuthenticated = computed(() => !!user.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  function checkAuth(): Promise<void> {
    if (initialized.value) return Promise.resolve()
    if (!authCheck) authCheck = restoreSession()
    return authCheck
  }

  async function restoreSession() {
    try {
      // First check if setup is completed (public endpoint)
      const status = await setupApi.getStatus()
      setupCompleted.value = status.setup_completed

      if (!status.setup_completed) {
        initialized.value = true
        return
      }

      // The HttpOnly session cookie survives refresh and is sent by the API client.
      try {
        user.value = await authApi.getCurrentUser()
      } catch {
        user.value = null
      }
    } catch {
      error.value = 'Unable to check server status. Please try again.'
    } finally {
      initialized.value = true
    }
  }

  async function login(email: string, password: string) {
    loading.value = true
    error.value = null

    try {
      const response = await authApi.login({ email, password })
      user.value = response.user
      return true
    } catch (err: unknown) {
      const apiError = err as { response?: { data?: { error?: { message?: string } } } }
      error.value = apiError.response?.data?.error?.message || 'Login failed'
      return false
    } finally {
      loading.value = false
    }
  }

  async function logout() {
    try {
      await authApi.logout()
    } finally {
      user.value = null
    }
  }

  async function changePassword(currentPassword: string, newPassword: string) {
    loading.value = true
    error.value = null

    try {
      await authApi.changePassword(currentPassword, newPassword)
      return true
    } catch (err: unknown) {
      const apiError = err as { response?: { data?: { error?: { message?: string } } } }
      error.value = apiError.response?.data?.error?.message || 'Password change failed'
      return false
    } finally {
      loading.value = false
    }
  }

  function clearError() {
    error.value = null
  }

  return {
    user,
    initialized,
    setupCompleted,
    loading,
    error,
    isAuthenticated,
    isAdmin,
    checkAuth,
    login,
    logout,
    changePassword,
    clearError
  }
})
