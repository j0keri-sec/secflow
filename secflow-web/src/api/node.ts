import http from '@/utils/http'
import type { Node, Task, CreateVulnCrawlRequest, CreateArticleCrawlRequest, PageData } from '@/types'

export interface NodeListParams { page?: number; page_size?: number }
export interface TaskListParams { page?: number; page_size?: number; status?: string }

export const nodeApi = {
  /** List all nodes - returns flat array, uses listNodes internally */
  list: async () => {
    const res = await nodeApi.listNodes({ page: 1, page_size: 100 })
    return res?.items ?? []
  },

  listNodes: (params: NodeListParams = {}) => {
    const q = new URLSearchParams()
    Object.entries(params).forEach(([k, v]) => { if (v !== undefined) q.set(k, String(v)) })
    return http.get<PageData<Node>>(`/nodes?${q}`)
  },

  listTasks: (params: TaskListParams = {}) => {
    const q = new URLSearchParams()
    Object.entries(params).forEach(([k, v]) => { if (v !== undefined && v !== '') q.set(k, String(v)) })
    return http.get<PageData<Task>>(`/tasks?${q}`)
  },

  getTask: (id: string) => http.get<Task>(`/tasks/${id}`),

  createVulnCrawlTask: (data: CreateVulnCrawlRequest) =>
    http.post<{ task_id: string }>('/tasks/vuln-crawl', data),

  createArticleCrawlTask: (data: CreateArticleCrawlRequest) =>
    http.post<{ task_id: string }>('/tasks/article-crawl', data),

  // 任务操作
  deleteTask: (id: string) => http.del(`/tasks/${id}`),

  stopTask: (id: string) => http.post<{ task_id: string; status: string }>(`/tasks/${id}/stop`),

  // 节点操作
  deleteNode: (id: string) => http.del(`/nodes/${id}`),

  pauseNode: (id: string) => http.post<{ node_id: string; status: string }>(`/nodes/${id}/pause`),

  resumeNode: (id: string) => http.post<{ node_id: string; status: string }>(`/nodes/${id}/resume`),

  disconnectNode: (id: string) => http.post<{ node_id: string; status: string }>(`/nodes/${id}/disconnect`),

  getNodeLogs: (id: string) => http.get<{ node_id: string; logs: string[]; count: number }>(`/nodes/${id}/logs`),
}

/** @deprecated use nodeApi.listTasks / nodeApi.createVulnCrawlTask */
export const taskApi = {
  list: (status?: string, page = 1, pageSize = 20) => {
    const q = new URLSearchParams({ page: String(page), page_size: String(pageSize) })
    if (status) q.set('status', status)
    return http.get<PageData<Task>>(`/tasks?${q}`)
  },
  get: (id: string) => http.get<{ task: Task; progress: number }>(`/tasks/${id}`),
  createVulnCrawl: (data: CreateVulnCrawlRequest) =>
    http.post<{ task_id: string }>('/tasks/vuln-crawl', data),
}
