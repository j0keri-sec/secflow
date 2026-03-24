import axios, { type AxiosInstance, type AxiosRequestConfig, type AxiosResponse } from 'axios'
import { useAuthStore } from '@/stores/auth'
import type { ApiResponse } from '@/types'

// Create a shared axios instance.
const http: AxiosInstance = axios.create({
  baseURL: '/api/v1',
  timeout: 30_000,
  headers: { 'Content-Type': 'application/json' },
})

// ── Request interceptor — attach JWT ──────────────────────────────────────
http.interceptors.request.use((config) => {
  const auth = useAuthStore()
  if (auth.token) {
    config.headers.Authorization = `Bearer ${auth.token}`
  }
  return config
})

// ── Response interceptor — unwrap envelope ────────────────────────────────
http.interceptors.response.use(
  (response: AxiosResponse<ApiResponse>) => {
    const data = response.data
    if (data.code !== 0) {
      return Promise.reject(new Error(data.message || 'API error'))
    }
    return response
  },
  (error) => {
    if (error.response?.status === 401) {
      const auth = useAuthStore()
      auth.logout()
      window.location.href = '/login'
    }
    return Promise.reject(error)
  },
)

// ── Typed request helpers ─────────────────────────────────────────────────
async function get<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
  const res = await http.get<ApiResponse<T>>(url, config)
  return res.data.data
}

async function post<T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
  console.log('HTTP POST:', url, data)
  const res = await http.post<ApiResponse<T>>(url, data, config)
  console.log('HTTP POST 响应:', res.data)
  return res.data.data
}

async function put<T>(url: string, data?: unknown): Promise<T> {
  const res = await http.put<ApiResponse<T>>(url, data)
  return res.data.data
}

async function patch<T>(url: string, data?: unknown): Promise<T> {
  const res = await http.patch<ApiResponse<T>>(url, data)
  return res.data.data
}

async function del<T>(url: string): Promise<T> {
  const res = await http.delete<ApiResponse<T>>(url)
  return res.data.data
}

async function getRaw(url: string, config?: AxiosRequestConfig): Promise<Blob> {
  const auth = useAuthStore()
  const res = await http.get(url, {
    ...config,
    responseType: 'blob',
    headers: { Authorization: auth.token ? `Bearer ${auth.token}` : undefined },
  })
  return res.data as Blob
}

export default { get, post, put, patch, del, getRaw, raw: http }
