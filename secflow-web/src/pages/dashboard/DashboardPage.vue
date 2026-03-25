<template>
  <div class="dashboard">
    <!-- 欢迎标题区 -->
    <div class="welcome-section">
      <div class="welcome-content">
        <h1 class="welcome-title">早上好</h1>
        <p class="welcome-subtitle">今天是 {{ today }}，安全态势一切正常</p>
      </div>
      <div class="time-display">
        <span class="time">{{ currentTime }}</span>
      </div>
    </div>

    <!-- 核心统计卡片 -->
    <div class="stats-grid">
      <div 
        v-for="(stat, index) in stats" 
        :key="stat.label"
        class="stat-card"
        :style="{ '--delay': `${index * 0.1}s` }"
      >
        <div class="stat-header">
          <span class="stat-label">{{ stat.label }}</span>
          <div class="stat-icon-wrapper" :style="{ background: stat.gradient }">
            <component :is="stat.icon" />
          </div>
        </div>
        <div class="stat-value">{{ stat.value }}</div>
        <div class="stat-footer">
          <span class="stat-change" :class="stat.trend">
            <svg v-if="stat.trend === 'up'" width="14" height="14" viewBox="0 0 24 24" fill="none">
              <path d="M12 19V5M5 12L12 5L19 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
            <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none">
              <path d="M12 5V19M5 12L12 19L19 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
            {{ stat.change }}
          </span>
          <span class="stat-period">较昨日</span>
        </div>
      </div>
    </div>

    <!-- 主内容区：监控 + 日志 -->
    <div class="main-grid">
      <!-- 服务器状态监控 -->
      <div class="monitor-section">
        <div class="section-header">
          <h2>系统状态</h2>
          <span class="status-badge online">
            <span class="status-dot"></span>
            运行中
          </span>
        </div>
        
        <!-- 实时指标 -->
        <div class="metrics-grid">
          <div class="metric-card" v-for="metric in metrics" :key="metric.label">
            <div class="metric-header">
              <span class="metric-label">{{ metric.label }}</span>
              <span class="metric-value">{{ metric.value }}%</span>
            </div>
            <div class="metric-bar">
              <div 
                class="metric-fill" 
                :style="{ width: `${metric.value}%`, background: metric.gradient }"
              ></div>
            </div>
            <div class="metric-detail">
              <span>{{ metric.used }}</span>
              <span>{{ metric.total }}</span>
            </div>
          </div>
        </div>

        <!-- 服务器列表 -->
        <div class="server-list">
          <div class="server-header">
            <span>节点</span>
            <span>状态</span>
            <span>负载</span>
            <span>内存</span>
          </div>
          <div class="server-item" v-for="server in servers" :key="server.node_id">
            <div class="server-name">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
                <rect x="2" y="2" width="20" height="8" rx="2" stroke="currentColor" stroke-width="1.5"/>
                <rect x="2" y="14" width="20" height="8" rx="2" stroke="currentColor" stroke-width="1.5"/>
                <circle cx="6" cy="6" r="1" fill="currentColor"/>
                <circle cx="6" cy="18" r="1" fill="currentColor"/>
              </svg>
              {{ server.name || server.node_id.slice(0, 12) }}
            </div>
            <span class="server-status" :class="getNodeStatus(server).status">{{ getNodeStatus(server).statusText }}</span>
            <span class="server-load">{{ getNodeLoad(server) }}</span>
            <span class="server-latency">{{ Math.round((server.info?.mem_used || 0) / 1024 / 1024 / 1024 * 10) / 10 }}GB / {{ Math.round((server.info?.mem_total || 0) / 1024 / 1024 / 1024 * 10) / 10 }}GB</span>
          </div>
          <div v-if="servers.length === 0" class="no-data">
            暂无节点数据
          </div>
        </div>
      </div>

      <!-- 最新日志 -->
      <div class="logs-section">
        <div class="section-header">
          <h2>系统日志</h2>
          <button class="view-all-btn">查看全部 →</button>
        </div>
        
        <div class="logs-list">
          <div 
            v-for="(log, index) in logs" 
            :key="index"
            class="log-item"
            :class="log.level"
          >
            <span class="log-time">{{ log.time }}</span>
            <span class="log-level">{{ log.levelText }}</span>
            <span class="log-message">{{ log.message }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 热点漏洞事件 -->
    <div class="hot-section">
      <div class="section-header">
        <h2>热点漏洞事件</h2>
        <div class="tab-filters">
          <button 
            v-for="tab in hotTabs" 
            :key="tab"
            :class="['tab-btn', { active: activeHotTab === tab }]"
            @click="activeHotTab = tab"
          >
            {{ tab }}
          </button>
        </div>
      </div>
      
      <div class="hot-grid" v-if="hotVulns.length > 0">
        <article 
          v-for="(vuln, index) in hotVulns" 
          :key="vuln.id || index"
          class="hot-card"
          @click="$router.push(`/vuln/${vuln.id || vuln.key}`)"
          style="cursor: pointer;"
        >
          <div class="hot-priority" :class="getSeverityClass(vuln.severity)"></div>
          <div class="hot-content">
            <div class="hot-header">
              <span class="hot-cve">{{ vuln.cve || 'N/A' }}</span>
              <span class="hot-badge" :class="getSeverityClass(vuln.severity)">{{ vuln.severity || '未知' }}</span>
            </div>
            <h3 class="hot-title">{{ vuln.title || '无标题' }}</h3>
            <p class="hot-desc">{{ truncateDesc(vuln.description) }}</p>
            <div class="hot-meta">
              <span class="hot-cvss">{{ vuln.source || '未知来源' }}</span>
              <span class="hot-time">{{ formatRelativeTime(vuln.created_at) }}</span>
            </div>
          </div>
        </article>
      </div>
      <div v-else class="no-data">
        暂无漏洞数据，请先执行爬取任务
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { 
  DataAnalysis, 
  TrendCharts, 
  Warning, 
  Clock
} from '@element-plus/icons-vue'
import { dashboardApi } from '@/api/dashboard'
import type { VulnRecord, Node } from '@/types'
import logger from '@/utils/logger'

const activeHotTab = ref('今日')
const hotTabs = ['今日', '本周', '本月']
const currentTime = ref('')
const today = ref('')
const loading = reactive({
  stats: false,
  nodes: false,
  vulns: false,
  logs: false,
})

let timeInterval: number

// 统计数据
const stats = reactive([
  {
    label: '漏洞总数',
    value: '0',
    change: '--',
    trend: 'up',
    icon: DataAnalysis,
    gradient: 'linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%)'
  },
  {
    label: '今日新增',
    value: '0',
    change: '--',
    trend: 'up',
    icon: Clock,
    gradient: 'linear-gradient(135deg, #f97316 0%, #ea580c 100%)'
  },
  {
    label: '高危漏洞',
    value: '0',
    change: '--',
    trend: 'up',
    icon: Warning,
    gradient: 'linear-gradient(135deg, #ef4444 0%, #dc2626 100%)'
  },
  {
    label: '情报源',
    value: '0',
    change: '--',
    trend: 'up',
    icon: TrendCharts,
    gradient: 'linear-gradient(135deg, #8b5cf6 0%, #7c3aed 100%)'
  }
])

// 节点指标
const metrics = reactive([
  {
    label: 'CPU 使用率',
    value: 0,
    used: '0%',
    total: '/ 0 核',
    gradient: 'linear-gradient(90deg, #3b82f6 0%, #06b6d4 100%)'
  },
  {
    label: '内存使用',
    value: 0,
    used: '0 GB',
    total: '/ 0 GB',
    gradient: 'linear-gradient(90deg, #8b5cf6 0%, #a855f7 100%)'
  },
  {
    label: '磁盘使用',
    value: 0,
    used: '0 GB',
    total: '/ 0 GB',
    gradient: 'linear-gradient(90deg, #22c55e 0%, #16a34a 100%)'
  },
  {
    label: '网络带宽',
    value: 0,
    used: '0 Mbps',
    total: '/ 0 Mbps',
    gradient: 'linear-gradient(90deg, #f97316 0%, #ea580c 100%)'
  }
])

// 服务器列表
const servers = ref<Node[]>([])

// 系统日志
const logs = ref<Array<{time: string, level: string, levelText: string, message: string}>>([])

// 热点漏洞
const hotVulns = ref<VulnRecord[]>([])

// 计算在线节点数
// const onlineNodes = computed(() => servers.value.filter(n => n.online).length)

// 格式化时间
const formatTime = (time: string) => {
  if (!time) return '--'
  try {
    const d = new Date(time)
    return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  } catch {
    return '--'
  }
}

// 格式化相对时间
const formatRelativeTime = (time: string) => {
  if (!time) return '--'
  try {
    const now = Date.now()
    const t = new Date(time).getTime()
    const diff = now - t
    const hours = Math.floor(diff / (1000 * 60 * 60))
    if (hours < 1) return '刚刚'
    if (hours < 24) return `${hours}小时前`
    const days = Math.floor(hours / 24)
    if (days < 7) return `${days}天前`
    return `${Math.floor(days / 7)}周前`
  } catch {
    return '--'
  }
}

// 获取漏洞统计
const fetchVulnStats = async () => {
  try {
    loading.stats = true
    logger.log('[Dashboard] 正在获取漏洞统计...')
    const data = await dashboardApi.getVulnStats()
    logger.log('[Dashboard] 漏洞统计数据:', data)
    
    // 更新统计数据
    stats[0].value = formatNumber(data.total || 0)
    stats[2].value = formatNumber((data.by_severity?.['高危'] || 0) + (data.by_severity?.['严重'] || 0))
    stats[3].value = '--' // 情报源数量暂不支持
    logger.log('[Dashboard] 统计更新完成:', stats[0].value, stats[2].value)
  } catch (e) {
    logger.error('获取漏洞统计失败:', e)
  } finally {
    loading.stats = false
  }
}

// 获取节点数据
const fetchNodes = async () => {
  try {
    loading.nodes = true
    logger.log('[Dashboard] 正在获取节点数据...')
    const data = await dashboardApi.getNodes()
    logger.log('[Dashboard] 节点数据:', data)
    const nodes: Node[] = data?.items || data || []
    servers.value = nodes

    // 更新节点统计
    stats[3].value = String(nodes.length)

    // 计算集群平均负载
    if (nodes.length > 0) {
      const avgCPU = nodes.reduce((sum, n) => sum + (n.info?.cpu_pct || 0), 0) / nodes.length
      const totalMem = nodes.reduce((sum, n) => sum + (n.info?.mem_total || 0), 0)
      const usedMem = nodes.reduce((sum, n) => sum + (n.info?.mem_used || 0), 0)
      const totalDisk = nodes.reduce((sum, n) => sum + (n.info?.disk_total || 0), 0)
      const usedDisk = nodes.reduce((sum, n) => sum + (n.info?.disk_used || 0), 0)

      // CPU
      metrics[0].value = Math.round(avgCPU)
      metrics[0].used = `${Math.round(avgCPU)}%`
      const cores = nodes.reduce((sum, n) => sum + (n.info?.cpu_cores || 0), 0)
      metrics[0].total = `/ ${cores} 核`

      // 内存
      if (totalMem > 0) {
        const memPercent = Math.round((usedMem / totalMem) * 100)
        metrics[1].value = memPercent
        metrics[1].used = `${(usedMem / 1024 / 1024 / 1024).toFixed(1)} GB`
        metrics[1].total = `/ ${(totalMem / 1024 / 1024 / 1024).toFixed(0)} GB`
      }

      // 磁盘
      if (totalDisk > 0) {
        const diskPercent = Math.round((usedDisk / totalDisk) * 100)
        metrics[2].value = diskPercent
        metrics[2].used = `${(usedDisk / 1024 / 1024 / 1024).toFixed(0)} GB`
        metrics[2].total = `/ ${(totalDisk / 1024 / 1024 / 1024).toFixed(0)} GB`
      }
    }
    logger.log('[Dashboard] 节点更新完成:', nodes.length, '个节点')
  } catch (e) {
    logger.error('获取节点数据失败:', e)
  } finally {
    loading.nodes = false
  }
}

// 获取热点漏洞
const fetchHotVulns = async () => {
  try {
    loading.vulns = true
    logger.log('[Dashboard] 正在获取热点漏洞...')
    const data = await dashboardApi.getRecentVulns(6)
    logger.log('[Dashboard] 热点漏洞数据:', data)
    hotVulns.value = data?.items || data || []
    
    // 计算今日新增
    const today = new Date()
    today.setHours(0, 0, 0, 0)
    const todayCount = hotVulns.value.filter(v => {
      try {
        return new Date(v.created_at) >= today
      } catch {
        return false
      }
    }).length
    stats[1].value = String(todayCount)
    logger.log('[Dashboard] 今日新增:', todayCount)
  } catch (e) {
    logger.error('获取热点漏洞失败:', e)
  } finally {
    loading.vulns = false
  }
}

// 获取系统日志
const fetchLogs = async () => {
  try {
    loading.logs = true
    logger.log('[Dashboard] 正在获取审计日志...')
    const data = await dashboardApi.getAuditLogs(6)
    logger.log('[Dashboard] 审计日志数据:', data)
    const auditLogs = data?.items || []
    
    if (Array.isArray(auditLogs) && auditLogs.length > 0) {
      logs.value = auditLogs.map((log: any) => ({
        time: formatTime(log.created_at),
        level: getLogLevel(log.action),
        levelText: getLogLevelText(log.action),
        message: getLogMessage(log)
      }))
    } else {
      // 无审计日志时显示系统运行日志
      logs.value = [
        { time: formatTime(new Date().toISOString()), level: 'success', levelText: 'SUCCESS', message: '系统运行正常' },
        { time: formatTime(new Date(Date.now() - 60000).toISOString()), level: 'info', levelText: 'INFO', message: '数据同步完成' },
      ]
    }
  } catch (e) {
    logger.error('获取系统日志失败:', e)
    // 使用默认日志
    logs.value = [
      { time: formatTime(new Date().toISOString()), level: 'info', levelText: 'INFO', message: '系统运行正常' }
    ]
  } finally {
    loading.logs = false
  }
}

// 辅助函数：格式化数字
const formatNumber = (n: number) => {
  if (n >= 10000) return (n / 10000).toFixed(1) + 'w'
  if (n >= 1000) return (n / 1000).toFixed(1) + 'k'
  return String(n)
}

// 辅助函数：获取日志级别
const getLogLevel = (action: string) => {
  if (action.includes('delete') || action.includes('remove')) return 'error'
  if (action.includes('create') || action.includes('login')) return 'success'
  if (action.includes('update') || action.includes('edit')) return 'warning'
  return 'info'
}

// 辅助函数：获取日志级别文本
const getLogLevelText = (action: string) => {
  if (action.includes('delete') || action.includes('remove')) return 'ERROR'
  if (action.includes('create')) return 'CREATE'
  if (action.includes('login')) return 'LOGIN'
  if (action.includes('update') || action.includes('edit')) return 'UPDATE'
  return 'INFO'
}

// 辅助函数：生成日志消息
const getLogMessage = (log: any) => {
  const action = log.action || ''
  const resource = log.resource || ''
  const username = log.username || '未知用户'
  
  if (action.includes('login')) return `用户 ${username} 登录系统`
  if (action.includes('create')) return `创建 ${resource}`
  if (action.includes('update')) return `更新 ${resource}`
  if (action.includes('delete')) return `删除 ${resource}`
  return `${action} ${resource}`
}

// 获取节点状态样式
const getNodeStatus = (node: Node) => {
  if (!node.online) return { status: 'offline', statusText: '离线' }
  if (node.status === 'busy') return { status: 'warning', statusText: '忙碌' }
  return { status: 'online', statusText: '在线' }
}

// 获取节点负载
const getNodeLoad = (node: Node) => {
  return `${Math.round(node.info?.cpu_pct || 0)}%`
}

// 获取节点延迟
// const getNodeLatency = (node: Node) => {
  // 延迟数据暂未提供，使用占位
//   return '--'
// }

// 获取严重性样式类
const getSeverityClass = (severity: string) => {
  switch (severity) {
    case '严重': return 'critical'
    case '高危': return 'high'
    case '中危': return 'medium'
    case '低危': return 'low'
    default: return 'medium'
  }
}

// 截断描述文本
const truncateDesc = (desc: string) => {
  if (!desc) return '暂无描述'
  return desc.length > 80 ? desc.slice(0, 80) + '...' : desc
}

onMounted(() => {
  const now = new Date()
  today.value = now.toLocaleDateString('zh-CN', { 
    month: 'long', 
    day: 'numeric', 
    weekday: 'long' 
  })
  
  const updateTime = () => {
    const d = new Date()
    currentTime.value = d.toLocaleTimeString('zh-CN', { 
      hour: '2-digit', 
      minute: '2-digit' 
    })
  }
  updateTime()
  timeInterval = window.setInterval(updateTime, 1000)

  // 加载真实数据
  fetchVulnStats()
  fetchNodes()
  fetchHotVulns()
  fetchLogs()

  // 定时刷新数据
  const refreshInterval = window.setInterval(() => {
    fetchVulnStats()
    fetchNodes()
  }, 30000) // 每30秒刷新

  onUnmounted(() => {
    clearInterval(timeInterval)
    clearInterval(refreshInterval)
  })
})
</script>

<style scoped>
:root {
  /* 浅色主题 - 纯净白色 */
  --primary: #3b82f6;
  --primary-dark: #1d4ed8;
  --accent: #06b6d4;
  
  --bg-primary: #ffffff;
  --bg-secondary: #f8f9fa;
  --bg-tertiary: #f1f3f5;
  
  --text-primary: #1a1a1a;
  --text-secondary: #495057;
  --text-tertiary: #868e96;
  
  --border: #dee2e6;
  
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.04);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.06), 0 2px 4px -1px rgba(0, 0, 0, 0.04);
  --shadow-lg: 0 10px 15px -3px rgba(0, 0, 0, 0.08), 0 4px 6px -2px rgba(0, 0, 0, 0.04);
  
  --radius-sm: 6px;
  --radius-md: 10px;
  --radius-lg: 14px;
}

[data-theme="dark"] {
  /* 深色主题 - 优雅深灰 */
  --primary: #3b82f6;
  --primary-dark: #1d4ed8;
  --accent: #06b6d4;

  --bg-primary: #0f1114;
  --bg-secondary: #16191d;
  --bg-tertiary: #1e2227;
  
  --text-primary: #f1f3f5;
  --text-secondary: #adb5bd;
  --text-tertiary: #6c757d;
  
  --border: #2d3239;
  
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.25);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.35), 0 2px 4px -1px rgba(0, 0, 0, 0.2);
  --shadow-lg: 0 10px 15px -3px rgba(0, 0, 0, 0.4), 0 4px 6px -2px rgba(0, 0, 0, 0.25);
}

.dashboard {
  display: flex;
  flex-direction: column;
  gap: 2rem;
}

/* ===== Welcome Section ===== */
.welcome-section {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  padding: 1.5rem 0;
}

.welcome-title {
  font-size: 2.25rem;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 0.5rem;
  letter-spacing: -0.02em;
}

.welcome-subtitle {
  font-size: 1rem;
  color: var(--text-tertiary);
  margin: 0;
}

.time-display {
  text-align: right;
}

.time {
  font-size: 2.5rem;
  font-weight: 300;
  color: var(--text-primary);
  letter-spacing: -0.02em;
}

/* ===== Stats Grid ===== */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1.25rem;
}

.stat-card {
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 1.25rem;
  transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
  animation: fadeInUp 0.5s ease-out var(--delay, 0s) both;
}

.stat-card:hover {
  transform: translateY(-4px);
  box-shadow: var(--shadow-lg);
}

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(16px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.stat-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 0.75rem;
}

.stat-label {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--text-secondary);
}

.stat-icon-wrapper {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
}

.stat-value {
  font-size: 2rem;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: -0.02em;
  margin-bottom: 0.5rem;
}

.stat-footer {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8rem;
}

.stat-change {
  display: flex;
  align-items: center;
  gap: 0.2rem;
  font-weight: 600;
}

.stat-change.up { color: #22c55e; }
.stat-change.down { color: #ef4444; }

.stat-period {
  color: var(--text-tertiary);
}

/* ===== Main Grid ===== */
.main-grid {
  display: grid;
  grid-template-columns: 1.4fr 1fr;
  gap: 1.5rem;
}

/* ===== Section Header ===== */
.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.25rem;
}

.section-header h2 {
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.status-badge {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.8rem;
  font-weight: 500;
  color: #22c55e;
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #22c55e;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

/* ===== Monitor Section ===== */
.monitor-section {
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 1.5rem;
  animation: fadeInUp 0.5s ease-out 0.3s both;
}

/* Metrics Grid */
.metrics-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.metric-card {
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  padding: 1rem;
}

.metric-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
}

.metric-label {
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.metric-value {
  font-size: 1rem;
  font-weight: 700;
  color: var(--text-primary);
}

.metric-bar {
  height: 4px;
  background: var(--bg-tertiary);
  border-radius: 2px;
  overflow: hidden;
  margin-bottom: 0.4rem;
}

.metric-fill {
  height: 100%;
  border-radius: 2px;
  transition: width 0.6s ease;
}

.metric-detail {
  display: flex;
  justify-content: space-between;
  font-size: 0.7rem;
  color: var(--text-tertiary);
}

/* Server List */
.server-list {
  border-top: 1px solid var(--border);
  padding-top: 1rem;
}

.server-header {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr;
  gap: 1rem;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--border);
  margin-bottom: 0.75rem;
}

.server-item {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr;
  gap: 1rem;
  align-items: center;
  padding: 0.6rem 0;
  font-size: 0.875rem;
  color: var(--text-secondary);
}

.server-name {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--text-primary);
  font-weight: 500;
}

.server-status {
  padding: 0.2rem 0.6rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 500;
}

.server-status.online {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

.server-status.warning {
  background: rgba(245, 158, 11, 0.1);
  color: #f59e0b;
}

/* ===== Logs Section ===== */
.logs-section {
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 1.5rem;
  animation: fadeInUp 0.5s ease-out 0.4s both;
}

.view-all-btn {
  background: none;
  border: none;
  color: var(--primary);
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.view-all-btn:hover {
  transform: translateX(3px);
}

.logs-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.log-item {
  display: grid;
  grid-template-columns: 70px 60px 1fr;
  gap: 0.75rem;
  align-items: center;
  padding: 0.6rem 0.75rem;
  background: var(--bg-secondary);
  border-radius: var(--radius-sm);
  font-size: 0.8rem;
}

.log-time {
  color: var(--text-tertiary);
  font-family: monospace;
}

.log-level {
  font-weight: 600;
  font-size: 0.7rem;
}

.log-item.info .log-level { color: #3b82f6; }
.log-item.warning .log-level { color: #f59e0b; }
.log-item.success .log-level { color: #22c55e; }
.log-item.error .log-level { color: #ef4444; }

.log-message {
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ===== Hot Section ===== */
.hot-section {
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 1.5rem;
  animation: fadeInUp 0.5s ease-out 0.5s both;
}

.tab-filters {
  display: flex;
  gap: 0.5rem;
}

.tab-btn {
  padding: 0.4rem 0.8rem;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  font-size: 0.8rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.tab-btn:hover {
  background: var(--bg-secondary);
}

.tab-btn.active {
  background: var(--primary);
  border-color: var(--primary);
  color: white;
}

/* Hot Grid */
.hot-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
  margin-top: 1.25rem;
}

.hot-card {
  display: flex;
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  overflow: hidden;
  transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.hot-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.hot-priority {
  width: 4px;
  flex-shrink: 0;
}

.hot-priority.critical { background: linear-gradient(180deg, #ef4444, #dc2626); }
.hot-priority.high { background: linear-gradient(180deg, #f97316, #ea580c); }
.hot-priority.medium { background: linear-gradient(180deg, #f59e0b, #d97706); }
.hot-priority.low { background: linear-gradient(180deg, #22c55e, #16a34a); }

.hot-content {
  flex: 1;
  padding: 1rem;
}

.hot-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.hot-cve {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-tertiary);
  font-family: monospace;
}

.hot-badge {
  padding: 0.15rem 0.5rem;
  border-radius: 4px;
  font-size: 0.65rem;
  font-weight: 600;
}

.hot-badge.critical {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.hot-badge.high {
  background: rgba(249, 115, 22, 0.1);
  color: #f97316;
}

.hot-badge.medium {
  background: rgba(245, 158, 11, 0.1);
  color: #f59e0b;
}

.hot-badge.low {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

.hot-title {
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 0.4rem;
  line-height: 1.3;
}

.hot-desc {
  font-size: 0.8rem;
  color: var(--text-secondary);
  margin: 0 0 0.75rem;
  line-height: 1.4;
}

.hot-meta {
  display: flex;
  justify-content: space-between;
  font-size: 0.7rem;
  color: var(--text-tertiary);
}

.hot-cvss {
  font-weight: 600;
  color: var(--primary);
}

.no-data {
  text-align: center;
  padding: 2rem;
  color: var(--text-tertiary);
  font-size: 0.9rem;
}

/* ===== Responsive ===== */
@media (max-width: 1200px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  
  .main-grid {
    grid-template-columns: 1fr;
  }
  
  .hot-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: 1fr;
  }
  
  .welcome-section {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.5rem;
  }
  
  .time-display {
    text-align: left;
  }
  
  .metrics-grid {
    grid-template-columns: 1fr;
  }
}
</style>
