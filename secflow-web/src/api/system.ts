import http from '@/utils/http'
import type { PageData } from '@/types'

// ── Push Channels ──────────────────────────────────────────────────────────
export interface PushChannel {
  id: string
  name: string
  type: string
  enabled: boolean
  config: Record<string, string>
  created_at: string
}

// ── Audit Logs ─────────────────────────────────────────────────────────────
export interface AuditLog {
  id: string
  user_id: string
  username: string
  action: string
  resource: string
  detail: string
  ip: string
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

  // ── Task Schedules ───────────────────────────────────────────────────────
  getTaskSchedules: () => http.get<TaskSchedule[]>('/task-schedules'),
  updateTaskSchedule: (data: UpdateTaskScheduleParams) => http.put<TaskSchedule>('/task-schedules', data),
}
