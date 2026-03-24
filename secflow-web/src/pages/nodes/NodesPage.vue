<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import type { Node } from '@/types'
import { nodeApi } from '@/api/node'
import { ElMessage, ElMessageBox } from 'element-plus'

const nodes = ref<Node[]>([])
const loading = ref(true)
const selectedNode = ref<Node | null>(null)
const showLogModal = ref(false)
const nodeLogs = ref<string[]>([])
const logsLoading = ref(false)
const operating = ref(false)

async function fetchNodes() {
  try {
    const res = await nodeApi.listNodes({ page: 1, page_size: 100 })
    nodes.value = res?.items || []
  } catch (error) {
    console.error('Failed to load nodes:', error)
    nodes.value = []
  }
}

onMounted(async () => {
  loading.value = false
  await fetchNodes()
  timer = setInterval(fetchNodes, 10_000)
})

let timer: ReturnType<typeof setInterval>
onUnmounted(() => clearInterval(timer))

function memPct(node: Node) {
  if (!node.info?.mem_total) return 0
  return Math.round((node.info.mem_used / node.info.mem_total) * 100)
}

function diskPct(node: Node) {
  if (!node.info?.disk_total) return 0
  return Math.round((node.info.disk_used / node.info.disk_total) * 100)
}

function fmtBytes(bytes: number) {
  if (!bytes) return '0 B'
  const gb = bytes / 1024 / 1024 / 1024
  if (gb >= 1) return gb.toFixed(1) + ' GB'
  return (bytes / 1024 / 1024).toFixed(0) + ' MB'
}

async function handleDeleteNode(node: Node) {
  try {
    await ElMessageBox.confirm(
      `确定要删除节点 "${node.name || node.node_id}" 吗？`,
      '删除节点',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' }
    )
    operating.value = true
    await nodeApi.deleteNode(node.id)
    ElMessage.success('节点已删除')
    await fetchNodes()
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('Failed to delete node:', error)
      ElMessage.error('删除失败')
    }
  } finally {
    operating.value = false
  }
}

async function handlePauseNode(node: Node) {
  try {
    await ElMessageBox.confirm(
      `确定要暂停节点 "${node.name || node.node_id}" 吗？`,
      '暂停节点',
      { confirmButtonText: '暂停', cancelButtonText: '取消', type: 'warning' }
    )
    operating.value = true
    await nodeApi.pauseNode(node.id)
    ElMessage.success('节点已暂停')
    await fetchNodes()
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('Failed to pause node:', error)
      ElMessage.error('暂停失败')
    }
  } finally {
    operating.value = false
  }
}

async function handleResumeNode(node: Node) {
  try {
    await ElMessageBox.confirm(
      `确定要恢复节点 "${node.name || node.node_id}" 吗？`,
      '恢复节点',
      { confirmButtonText: '恢复', cancelButtonText: '取消', type: 'warning' }
    )
    operating.value = true
    await nodeApi.resumeNode(node.id)
    ElMessage.success('节点已恢复')
    await fetchNodes()
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('Failed to resume node:', error)
      ElMessage.error('恢复失败')
    }
  } finally {
    operating.value = false
  }
}

async function handleViewLogs(node: Node) {
  selectedNode.value = node
  showLogModal.value = true
  logsLoading.value = true
  try {
    const res = await nodeApi.getNodeLogs(node.id)
    nodeLogs.value = res.logs || []
  } catch (error) {
    console.error('Failed to get node logs:', error)
    nodeLogs.value = ['获取日志失败']
  } finally {
    logsLoading.value = false
  }
}

function getNodeStatus(node: Node): string {
  if (node.status === 'paused') return 'paused'
  return node.online ? 'online' : 'offline'
}

function getStatusClass(node: Node): string {
  const status = getNodeStatus(node)
  if (status === 'online') return 'status-online'
  if (status === 'paused') return 'status-paused'
  return 'status-offline'
}

function getStatusLabel(node: Node): string {
  const status = getNodeStatus(node)
  if (status === 'online') return '在线'
  if (status === 'paused') return '已暂停'
  return '离线'
}

function canPauseNode(node: Node): boolean {
  return node.status !== 'paused' && node.online
}

function canResumeNode(node: Node): boolean {
  return node.status === 'paused'
}
</script>

<template>
  <div class="ops-page">
    <!-- 工具栏 -->
    <div class="toolbar">
      <span class="toolbar-info">
        <span class="status-dot" :class="nodes.some(n => n.online) ? 'online' : 'offline'"></span>
        {{ nodes.filter(n => n.status === 'online').length }} / {{ nodes.length }} 节点在线
      </span>
      <span class="toolbar-hint">
        <span class="auto-refresh-dot"></span>
        每 10 秒自动刷新
      </span>
      <button class="btn-icon" @click="fetchNodes" title="刷新">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M23 4v6h-6M1 20v-6h6"/>
          <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
        </svg>
      </button>
    </div>

    <!-- 节点网格 -->
    <div class="node-container">
      <div v-if="loading" class="node-grid">
        <div v-for="i in 4" :key="i" class="node-card skeleton">
          <div class="skeleton-header"></div>
          <div class="skeleton-body"></div>
        </div>
      </div>

      <div v-else-if="nodes.length === 0" class="empty-state">
        <div class="empty-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <rect x="2" y="2" width="20" height="8" rx="2" ry="2"/>
            <rect x="2" y="14" width="20" height="8" rx="2" ry="2"/>
          </svg>
        </div>
        <p class="empty-text">暂无节点接入</p>
        <p class="empty-hint">启动客户端节点即可自动接入</p>
      </div>

      <div v-else class="node-grid">
        <div
          v-for="node in nodes" :key="node.id"
          class="node-card"
          @click="selectedNode = node"
        >
          <!-- 卡片头部 -->
          <div class="node-header">
            <div class="node-title">
              <span class="status-dot" :class="getStatusClass(node)"></span>
              <span class="node-name">{{ node.name || node.node_id }}</span>
            </div>
            <span class="status-badge" :class="getStatusClass(node)">
              {{ getStatusLabel(node) }}
            </span>
          </div>

          <!-- 快速操作 -->
          <div class="quick-actions" @click.stop>
            <button
              v-if="canPauseNode(node)"
              class="action-btn warning"
              title="暂停节点"
              @click="handlePauseNode(node)"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="6" y="4" width="4" height="16"/>
                <rect x="14" y="4" width="4" height="16"/>
              </svg>
            </button>
            <button
              v-if="canResumeNode(node)"
              class="action-btn success"
              title="恢复节点"
              @click="handleResumeNode(node)"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polygon points="5 3 19 12 5 21 5 3"/>
              </svg>
            </button>
            <button
              class="action-btn default"
              title="查看日志"
              @click="handleViewLogs(node)"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
                <polyline points="14 2 14 8 20 8"/>
                <line x1="16" y1="13" x2="8" y2="13"/>
                <line x1="16" y1="17" x2="8" y2="17"/>
              </svg>
            </button>
            <button
              class="action-btn danger"
              title="删除节点"
              @click="handleDeleteNode(node)"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="3 6 5 6 21 6"/>
                <path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/>
              </svg>
            </button>
          </div>

          <!-- IP / OS -->
          <div class="node-meta">
            <span class="meta-item">
              <el-icon><Wifi /></el-icon>
              {{ node.info?.public_ip || node.info?.ip || '—' }}
            </span>
            <span class="meta-item">
              {{ node.info?.os ?? '—' }} {{ node.info?.arch }}
            </span>
          </div>

          <!-- 资源条 -->
          <div class="resource-bars" v-if="node.status !== 'offline'">
            <div class="resource-item">
              <div class="resource-header">
                <span class="resource-label">CPU</span>
                <span class="resource-value">{{ node.info?.cpu_pct ?? 0 }}%</span>
              </div>
              <div class="resource-bar">
                <div class="resource-bar-fill cpu" :style="{ width: (node.info?.cpu_pct ?? 0) + '%' }"></div>
              </div>
            </div>
            <div class="resource-item">
              <div class="resource-header">
                <span class="resource-label">内存</span>
                <span class="resource-value">{{ fmtBytes(node.info?.mem_used ?? 0) }} / {{ fmtBytes(node.info?.mem_total ?? 0) }}</span>
              </div>
              <div class="resource-bar">
                <div class="resource-bar-fill memory" :style="{ width: memPct(node) + '%' }"></div>
              </div>
            </div>
            <div class="resource-item">
              <div class="resource-header">
                <span class="resource-label">磁盘</span>
                <span class="resource-value">{{ fmtBytes(node.info?.disk_used ?? 0) }} / {{ fmtBytes(node.info?.disk_total ?? 0) }}</span>
              </div>
              <div class="resource-bar">
                <div class="resource-bar-fill disk" :style="{ width: diskPct(node) + '%' }"></div>
              </div>
            </div>
          </div>

          <div class="node-footer">
            最后在线：{{ node.last_seen_at ? new Date(node.last_seen_at).toLocaleString('zh-CN') : '—' }}
          </div>
        </div>
      </div>
    </div>

    <!-- 节点详情弹窗 -->
    <Teleport to="body">
      <Transition name="fade">
        <div v-if="selectedNode" class="modal-overlay" @click="selectedNode = null">
          <div class="modal-content" @click.stop>
            <div class="modal-header">
              <h2 class="modal-title">节点详情</h2>
              <button class="modal-close" @click="selectedNode = null">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18"/>
                  <line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
              </button>
            </div>
            <div class="modal-body">
              <div class="detail-section">
                <div class="detail-row">
                  <span class="detail-label">状态</span>
                  <span class="status-badge" :class="getStatusClass(selectedNode)">{{ getStatusLabel(selectedNode) }}</span>
                </div>
                <div class="detail-grid">
                  <div class="detail-item">
                    <span class="detail-label">节点 ID</span>
                    <span class="detail-value mono">{{ selectedNode.node_id }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="detail-label">节点名称</span>
                    <span class="detail-value">{{ selectedNode.name || '—' }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="detail-label">内网 IP</span>
                    <span class="detail-value">{{ selectedNode.info?.ip || '—' }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="detail-label">公网 IP</span>
                    <span class="detail-value">{{ selectedNode.info?.public_ip || '—' }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="detail-label">操作系统</span>
                    <span class="detail-value">{{ selectedNode.info?.os }} {{ selectedNode.info?.arch }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="detail-label">CPU 型号</span>
                    <span class="detail-value">{{ selectedNode.info?.cpu_model || '—' }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="detail-label">注册时间</span>
                    <span class="detail-value">{{ selectedNode.registered_at ? new Date(selectedNode.registered_at).toLocaleString('zh-CN') : '—' }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="detail-label">最后在线</span>
                    <span class="detail-value">{{ selectedNode.last_seen_at ? new Date(selectedNode.last_seen_at).toLocaleString('zh-CN') : '—' }}</span>
                  </div>
                </div>
              </div>
              <div class="detail-actions">
                <button v-if="canPauseNode(selectedNode)" class="btn-warning" @click="handlePauseNode(selectedNode)">暂停节点</button>
                <button v-if="canResumeNode(selectedNode)" class="btn-success" @click="handleResumeNode(selectedNode)">恢复节点</button>
                <button class="btn-default" @click="handleViewLogs(selectedNode)">查看日志</button>
                <button class="btn-danger" @click="handleDeleteNode(selectedNode)">删除节点</button>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- 日志弹窗 -->
    <Teleport to="body">
      <Transition name="fade">
        <div v-if="showLogModal && selectedNode" class="modal-overlay" @click="showLogModal = false">
          <div class="modal-content log-modal" @click.stop>
            <div class="modal-header">
              <div>
                <h2 class="modal-title">节点日志</h2>
                <p class="modal-subtitle">{{ selectedNode.name || selectedNode.node_id }}</p>
              </div>
              <button class="modal-close" @click="showLogModal = false">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18"/>
                  <line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
              </button>
            </div>
            <div class="modal-body log-body">
              <div v-if="logsLoading" class="log-loading">加载中...</div>
              <div v-else-if="nodeLogs.length === 0" class="log-empty">暂无日志</div>
              <div v-else class="log-list">
                <div v-for="(log, idx) in nodeLogs" :key="idx" class="log-item">{{ log }}</div>
              </div>
            </div>
            <div class="modal-footer">
              <button class="btn-secondary" @click="showLogModal = false">关闭</button>
              <button class="btn-primary" @click="handleViewLogs(selectedNode)">刷新</button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<style scoped>
.ops-page {
  padding: 0 0 2rem;
  animation: fadeIn 0.4s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

/* 工具栏 */
.toolbar {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.75rem 1rem;
  margin-bottom: 1rem;
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.toolbar-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.toolbar-hint {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8rem;
  color: var(--text-tertiary);
}

.auto-refresh-dot {
  width: 6px;
  height: 6px;
  background: var(--color-success);
  border-radius: 50%;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}
.status-dot.online {
  background: var(--color-success);
  box-shadow: 0 0 6px rgba(34, 197, 94, 0.5);
}
.status-dot.offline {
  background: var(--text-tertiary);
}

.btn-icon {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-icon:hover {
  background: var(--color-bg-tertiary);
  color: var(--color-text-primary);
}

/* 节点容器 */
.node-container {
  background: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: 1.5rem;
}

/* 节点网格 */
.node-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
}

/* 节点卡片 */
.node-card {
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  padding: 1rem;
  cursor: pointer;
  transition: all 0.3s ease;
}

.node-card:hover {
  border-color: var(--color-primary);
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.node-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.75rem;
}

.node-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-dot.status-online { background: #22c55e; animation: pulse 2s infinite; }
.status-dot.status-paused { background: #a855f7; }
.status-dot.status-offline { background: #6b7280; }

.node-name {
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--color-text-primary);
}

.status-badge {
  font-size: 0.65rem;
  font-weight: 600;
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
}

.status-badge.status-online { background: rgba(34, 197, 94, 0.15); color: #22c55e; }
.status-badge.status-paused { background: rgba(168, 85, 247, 0.15); color: #a855f7; }
.status-badge.status-offline { background: rgba(107, 114, 128, 0.15); color: #6b7280; }

/* 快速操作 */
.quick-actions {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--color-border-light);
}

.action-btn {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.action-btn.warning { background: rgba(234, 179, 8, 0.15); color: #eab308; }
.action-btn.warning:hover { background: rgba(234, 179, 8, 0.25); }

.action-btn.success { background: rgba(34, 197, 94, 0.15); color: #22c55e; }
.action-btn.success:hover { background: rgba(34, 197, 94, 0.25); }

.action-btn.default { background: var(--color-bg-tertiary); color: var(--color-text-tertiary); }
.action-btn.default:hover { background: var(--color-border); color: var(--color-text-primary); }

.action-btn.danger { background: rgba(239, 68, 68, 0.15); color: #ef4444; }
.action-btn.danger:hover { background: rgba(239, 68, 68, 0.25); }

/* 元信息 */
.node-meta {
  display: flex;
  gap: 1rem;
  font-size: 0.75rem;
  color: var(--color-text-tertiary);
  margin-bottom: 0.75rem;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

/* 资源条 */
.resource-bars {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
}

.resource-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.resource-header {
  display: flex;
  justify-content: space-between;
  font-size: 0.7rem;
}

.resource-label { color: var(--color-text-tertiary); }
.resource-value { color: var(--color-text-secondary); font-family: monospace; }

.resource-bar {
  height: 4px;
  background: var(--color-bg-tertiary);
  border-radius: 2px;
  overflow: hidden;
}

.resource-bar-fill {
  height: 100%;
  border-radius: 2px;
  transition: width 0.5s ease;
}

.resource-bar-fill.cpu { background: var(--color-primary); }
.resource-bar-fill.memory { background: #22c55e; }
.resource-bar-fill.disk { background: #a855f7; }

/* 页脚 */
.node-footer {
  font-size: 0.7rem;
  color: var(--color-text-tertiary);
  padding-top: 0.5rem;
  border-top: 1px solid var(--color-border-light);
}

/* 骨架屏 */
.skeleton .node-header,
.skeleton .quick-actions,
.skeleton .node-meta,
.skeleton .resource-bars,
.skeleton .node-footer {
  opacity: 0.5;
}

.skeleton-header {
  height: 24px;
  background: linear-gradient(90deg, var(--color-bg-tertiary) 25%, var(--color-border) 50%, var(--color-bg-tertiary) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: 4px;
  margin-bottom: 0.75rem;
}

.skeleton-body {
  height: 80px;
  background: linear-gradient(90deg, var(--color-bg-tertiary) 25%, var(--color-border) 50%, var(--color-bg-tertiary) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: 4px;
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* 空状态 */
.empty-state {
  text-align: center;
  padding: 3rem 1rem;
}

.empty-icon {
  display: flex;
  justify-content: center;
  margin-bottom: 1rem;
  color: var(--color-text-tertiary);
}

.empty-text {
  font-size: 1rem;
  color: var(--color-text-secondary);
  margin: 0 0 0.5rem;
}

.empty-hint {
  font-size: 0.85rem;
  color: var(--color-text-tertiary);
  margin: 0;
}

/* 弹窗 */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.modal-content {
  background: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  width: 100%;
  max-width: 560px;
  max-height: 90vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.log-modal {
  max-width: 800px;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--color-border);
}

.modal-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}

.modal-subtitle {
  font-size: 0.8rem;
  color: var(--color-text-tertiary);
  margin: 0.25rem 0 0;
}

.modal-close {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  border-radius: 6px;
  color: var(--color-text-tertiary);
  cursor: pointer;
  transition: all 0.2s ease;
}

.modal-close:hover {
  background: var(--color-bg-tertiary);
  color: var(--color-text-primary);
}

.modal-body {
  padding: 1.25rem;
  overflow-y: auto;
  flex: 1;
}

.log-body {
  max-height: 400px;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  padding: 1rem 1.25rem;
  border-top: 1px solid var(--color-border);
}

/* 详情 */
.detail-section {
  margin-bottom: 1.5rem;
}

.detail-row {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1rem;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 1rem;
}

.detail-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.detail-label {
  font-size: 0.75rem;
  color: var(--color-text-tertiary);
}

.detail-value {
  font-size: 0.85rem;
  color: var(--color-text-primary);
}

.detail-value.mono {
  font-family: monospace;
  font-size: 0.75rem;
  word-break: break-all;
}

.detail-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  padding-top: 1rem;
  border-top: 1px solid var(--color-border);
}

.btn-warning, .btn-success, .btn-default, .btn-danger {
  padding: 0.5rem 1rem;
  border-radius: var(--radius-sm);
  font-size: 0.8rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-warning {
  background: rgba(234, 179, 8, 0.15);
  border: 1px solid rgba(234, 179, 8, 0.3);
  color: #eab308;
}
.btn-warning:hover { background: rgba(234, 179, 8, 0.25); }

.btn-success {
  background: rgba(34, 197, 94, 0.15);
  border: 1px solid rgba(34, 197, 94, 0.3);
  color: #22c55e;
}
.btn-success:hover { background: rgba(34, 197, 94, 0.25); }

.btn-default {
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  color: var(--color-text-secondary);
}
.btn-default:hover { background: var(--color-bg-tertiary); }

.btn-danger {
  background: rgba(239, 68, 68, 0.15);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #ef4444;
}
.btn-danger:hover { background: rgba(239, 68, 68, 0.25); }

/* 日志 */
.log-loading, .log-empty {
  text-align: center;
  padding: 2rem;
  color: var(--color-text-tertiary);
}

.log-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.log-item {
  padding: 0.75rem;
  background: var(--color-bg-secondary);
  border-radius: var(--radius-sm);
  font-family: monospace;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  white-space: pre-wrap;
  word-break: break-all;
}

/* 按钮 */
.btn-primary {
  padding: 0.5rem 1rem;
  background: var(--gradient-primary);
  border: none;
  border-radius: var(--radius-sm);
  color: white;
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}
.btn-primary:hover { opacity: 0.9; }

.btn-secondary {
  padding: 0.5rem 1rem;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  color: var(--color-text-secondary);
  font-size: 0.85rem;
  cursor: pointer;
  transition: all 0.2s ease;
}
.btn-secondary:hover { background: var(--color-bg-secondary); }

/* 过渡 */
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from, .fade-leave-to { opacity: 0; }

/* 响应式 */
@media (max-width: 1200px) {
  .node-grid { grid-template-columns: repeat(2, 1fr); }
}

@media (max-width: 768px) {
  .page-header { flex-direction: column; gap: 1rem; align-items: flex-start; }
  .header-right { width: 100%; justify-content: space-between; }
  .node-grid { grid-template-columns: 1fr; }
  .detail-grid { grid-template-columns: 1fr; }
}
</style>
