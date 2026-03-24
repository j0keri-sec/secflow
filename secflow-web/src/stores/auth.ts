import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { UserProfile } from '@/types'
import { authApi } from '@/api/auth'

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
    console.log('AuthStore: 调用登录 API...', credentials)
    const res = await authApi.login(credentials)
    console.log('AuthStore: 登录 API 返回:', res)
    setToken(res.token)
    setUser(res.user)
    console.log('AuthStore: Token 和 User 已设置', { token: token.value, user: user.value })
  }

  async function fetchMe() {
    console.log('AuthStore: 调用 fetchMe API...')
    try {
      const u = await authApi.me()
      console.log('AuthStore: fetchMe 返回用户信息:', u)
      setUser(u)
    } catch (e) {
      console.error('AuthStore: fetchMe 失败:', e)
      logout()
      throw e // 重新抛出错误，让调用者知道失败了
    }
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('token')
  }

  return { token, user, isLoggedIn, isAdmin, isEditor, setToken, setUser, login, fetchMe, logout }
})
