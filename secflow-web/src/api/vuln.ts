import http from '@/utils/http'
import type { VulnRecord, VulnStats, PageData } from '@/types'

export interface VulnListParams {
  page?: number
  page_size?: number
  severity?: string
  source?: string
  cve?: string
  keyword?: string
  q?: string
  pushed?: string | boolean
}

export const vulnApi = {
  list: (params: VulnListParams = {}) => {
    const q = new URLSearchParams()
    Object.entries(params).forEach(([k, v]) => {
      if (v !== undefined && v !== '') q.set(k, String(v))
    })
    return http.get<PageData<VulnRecord>>(`/vulns?${q}`)
  },
  get: (id: string) => http.get<VulnRecord>(`/vulns/${id}`),
  delete: (id: string) => http.del<null>(`/vulns/${id}`),
  stats: () => http.get<VulnStats>('/vulns/stats'),
  exportCSV: (params: VulnListParams = {}) => {
    const q = new URLSearchParams()
    Object.entries(params).forEach(([k, v]) => {
      if (v !== undefined && v !== '') q.set(k, String(v))
    })
    q.set('format', 'csv')
    return http.getRaw(`/vulns/export?${q}`) as Promise<Blob>
  },
}
