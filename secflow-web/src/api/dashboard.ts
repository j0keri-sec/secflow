import http from '@/utils/http'
import type { VulnRecord, VulnStats, Node, PageData, AuditLog } from '@/types'

// Dashboard 统计数据
export interface DashboardStats {
  vuln_total: number
  vuln_today: number
  high_vuln: number
  sources: number
  node_online: number
  article_total: number
  vuln_by_severity: Record<string, number>
}

// 热点漏洞 (简化版)
export interface HotVuln {
  id: string
  key: string
  cve: string
  title: string
  description: string
  severity: string
  cvss: string
  source: string
  time: string
  created_at: string
}

export const dashboardApi = {
  // 获取漏洞统计
  getVulnStats: () => http.get<VulnStats>('/vulns/stats'),

  // 获取漏洞列表 (用于热点漏洞)
  getRecentVulns: (pageSize = 6) => {
    return http.get<PageData<VulnRecord>>(`/vulns?page=1&page_size=${pageSize}`)
  },

  // 获取节点列表
  getNodes: () => http.get<PageData<Node>>('/nodes?page=1&page_size=100'),

  // 获取审计日志
  getAuditLogs: (pageSize = 10) => {
    return http.get<PageData<AuditLog>>(`/audit-logs?page=1&page_size=${pageSize}`)
  },
}
