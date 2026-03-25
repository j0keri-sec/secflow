// Global TypeScript type definitions for the secflow platform.

export type SeverityLevel = '低危' | '中危' | '高危' | '严重'
export type RoleType = 'admin' | 'editor' | 'viewer'
export type NodeStatus = 'online' | 'offline' | 'busy' | 'paused'
export type TaskStatus = 'pending' | 'dispatched' | 'running' | 'done' | 'failed'
export type TaskType = 'vuln_crawl' | 'article_crawl'

// ── API Response Envelope ──────────────────────────────────────────────────
export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
}

export interface PageData<T> {
  total: number
  page: number
  page_size: number
  items: T[]
}

// ── Auth ───────────────────────────────────────────────────────────────────
export interface UserProfile {
  id: string
  username: string
  email: string
  role: RoleType
  avatar: string
  active: boolean
  created_at: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  user: UserProfile
}

export interface InviteCode {
  id: string
  code: string
  owner_id: string
  used: boolean
  created_at: string
  used_at?: string
}

// ── Vulnerability ──────────────────────────────────────────────────────────
export interface VulnRecord {
  id: string
  key: string
  title: string
  description: string
  severity: SeverityLevel
  cve: string
  disclosure: string
  solutions: string
  references: string[]
  tags: string[]
  github_search: string[]
  from: string
  source: string
  url: string
  pushed: boolean
  reported_by: string
  created_at: string
  updated_at: string
}

export interface VulnStats {
  total: number
  by_severity: Record<string, number>
}

// ── Node ───────────────────────────────────────────────────────────────────
export interface NodeInfo {
  ip: string
  public_ip: string
  mac: string
  os: string
  arch: string
  cpu_model: string
  cpu_cores: number
  mem_total: number
  mem_used: number
  disk_total: number
  disk_used: number
  cpu_pct: number
  net_cards: string[]
}

export interface Node {
  id: string
  node_id: string
  name: string
  status: NodeStatus
  info: NodeInfo
  sources: string[]
  last_seen_at: string
  registered_at: string
  online: boolean  // real-time status from WebSocket hub
}

// ── Task ───────────────────────────────────────────────────────────────────
export interface Task {
  id: string
  task_id: string
  type: TaskType
  status: TaskStatus
  assigned_to: string
  payload: Record<string, unknown>
  error?: string
  progress: number
  created_at: string
  updated_at: string
  finished_at?: string
}

export interface CreateVulnCrawlRequest {
  sources: string[]
  page_limit?: number
  enable_github?: boolean
  proxy?: string
}

export interface CreateArticleCrawlRequest {
  sources: string[]
  limit?: number
}

// ── Article ────────────────────────────────────────────────────────────────
export interface Article {
  id: string
  title: string
  summary: string
  content: string
  author: string
  source: string
  url: string
  image: string
  tags: string[]
  pushed: boolean
  reported_by: string
  published_at: string
  created_at: string
}

// ── Dashboard ──────────────────────────────────────────────────────────────
export interface DashboardStats {
  vuln_total: number
  vuln_today: number
  article_total: number
  node_online: number
  vuln_by_severity: Record<string, number>
  recent_vulns: VulnRecord[]
}

// ── Audit Log ───────────────────────────────────────────────────────────────
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
