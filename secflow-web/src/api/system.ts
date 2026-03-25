import http from '@/utils/http'
import type { PageData, AuditLog } from '@/types'

// ── Push Channels ──────────────────────────────────────────────────────────
export interface PushChannel {
  id: string
  name: string
  type: string
  enabled: boolean
  config: Record<string, string>
  created_at: string
}

// ── Reports ────────────────────────────────────────────────────────────────
export interface Report {
  id: string
  title: string
  description: string
  type: string
  status: 'draft' | 'generating' | 'done' | 'failed'
  file_url: string
  created_by: string
  created_at: string
  finished_at?: string
}

export interface CreateReportParams {
  title: string
  description?: string
  type: string
  date_from?: string
  date_to?: string
}

// ── Report Generation ───────────────────────────────────────────────────────
export interface DataSource {
  id: string
  name: string
  description: string
}

export interface AIModel {
  id: string
  name: string
  description: string
}

export interface ReportGenerateParams {
  title: string
  type: string
  sources?: string[]
  date_from?: string
  date_to?: string
  ai_model?: string
  formats?: string[]
}

// ── Task Schedules ─────────────────────────────────────────────────────────
export interface TaskSchedule {
  type: string
  enabled: boolean
  interval: number // in minutes
  sources: string[]
}

export interface UpdateTaskScheduleParams {
  type: string
  enabled: boolean
  interval: number
  sources: string[]
}

export const systemApi = {
  // ── Push channels ────────────────────────────────────────────────────────
  listPushChannels: () => http.get<PushChannel[]>('/push-channels'),
  createPushChannel: (data: Partial<PushChannel>) =>
    http.post<PushChannel>('/push-channels', data),
  updatePushChannel: (id: string, data: Partial<PushChannel>) =>
    http.patch<PushChannel>(`/push-channels/${id}`, data),
  deletePushChannel: (id: string) => http.del<null>(`/push-channels/${id}`),

  // ── Audit logs ───────────────────────────────────────────────────────────
  listAuditLogs: (params: { page?: number; page_size?: number; keyword?: string; action?: string }) => {
    const q = new URLSearchParams()
    Object.entries(params).forEach(([k, v]) => { if (v !== undefined && v !== '') q.set(k, String(v)) })
    return http.get<PageData<AuditLog>>(`/audit-logs?${q}`)
  },

  // ── Reports ──────────────────────────────────────────────────────────────
  listReports: (params: { page?: number; page_size?: number }) => {
    const q = new URLSearchParams()
    Object.entries(params).forEach(([k, v]) => { if (v !== undefined) q.set(k, String(v)) })
    return http.get<PageData<Report>>(`/reports?${q}`)
  },
  createReport: (data: CreateReportParams) => http.post<Report>('/reports', data),
  deleteReport: (id: string) => http.del<null>(`/reports/${id}`),

  // Report generation
  getDataSources: () => http.get<{ sources: DataSource[] }>('/reports/datasources'),
  getAIModels: () => http.get<{ models: AIModel[] }>('/reports/aimodels'),
  generateReport: (params: ReportGenerateParams) => {
    const q = new URLSearchParams()
    if (params.type) q.set('type', params.type)
    if (params.title) q.set('title', params.title)
    params.sources?.forEach(s => q.append('sources', s))
    if (params.date_from) q.set('date_from', params.date_from)
    if (params.date_to) q.set('date_to', params.date_to)
    if (params.ai_model) q.set('ai_model', params.ai_model)
    return http.get<Blob>(`/reports/export?${q}`, { responseType: 'blob' })
  },
  previewReport: (params: { type?: string; sources?: string[]; date_from?: string; date_to?: string }) => {
    const q = new URLSearchParams()
    if (params.type) q.set('type', params.type)
    params.sources?.forEach(s => q.append('sources', s))
    if (params.date_from) q.set('date_from', params.date_from)
    if (params.date_to) q.set('date_to', params.date_to)
    return http.get<string>(`/reports/preview?${q}`)
  },

  // ── Task Schedules ───────────────────────────────────────────────────────
  getTaskSchedules: () => http.get<TaskSchedule[]>('/task-schedules'),
  updateTaskSchedule: (data: UpdateTaskScheduleParams) => http.put<TaskSchedule>('/task-schedules', data),
}
