import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { UserProfile } from '@/types'
import { authApi } from '@/api/auth'
import logger from '@/utils/logger'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string>(localStorage.getItem('token') ?? '')
  const user  = ref<UserProfile | null>(null)

  const isLoggedIn = computed(() => !!token.value)
  const isAdmin    = computed(() => user.value?.role === 'admin')
  const isEditor   = computed(() => user.value?.role === 'editor' || isAdmin.value)

  function setToken(t: string) {
    token.value = t
    localStorage.setItem('token', t)
  }

  function setUser(u: UserProfile) {
    user.value = u
  }

  async function login(credentials: { username: string; password: string }) {
    const res = await authApi.login(credentials)
    setToken(res.token)
    setUser(res.user)
    logger.info('Login successful', { username: credentials.username })
  }

  async function fetchMe() {
    try {
      const u = await authApi.me()
      setUser(u)
    } catch (e) {
      logger.error('fetchMe failed', e)
      logout()
      throw e
    }
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('token')
    logger.info('User logged out')
  }

  return { token, user, isLoggedIn, isAdmin, isEditor, setToken, setUser, login, fetchMe, logout }
})
