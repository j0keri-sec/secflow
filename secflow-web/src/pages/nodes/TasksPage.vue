<script setup lang="ts">
import logger from '@/utils/logger'
import { ref, reactive, onMounted, watch, computed } from 'vue'
import type { Task, TaskStatus, TaskType } from '@/types'
import { nodeApi } from '@/api/node'
import { ElMessage, ElMessageBox } from 'element-plus'

const items = ref<Task[]>([])
const total = ref(0)
const loading = ref(false)
const creating = ref(false)
const showCreateModal = ref(false)
const showDetailModal = ref(false)
const selectedTask = ref<Task | null>(null)
const taskLogs = ref<string[]>([])
const logsLoading = ref(false)

const taskTypes = [
  { value: 'vuln_crawl', label: '漏洞情报收集' },
  { value: 'article_crawl', label: '安全资讯收集' },
] as const

const selectedTaskType = ref<TaskType>('vuln_crawl')

const vulnSources = [
  'avd-rod', 'seebug-rod', 'ti-rod', 'nox-rod', 'kev-rod',
  'struts2-rod', 'chaitin-rod', 'oscs-rod', 'threatbook-rod', 'venustech-rod',
  'cnvd-rod', 'cnnvd-rod', 'nsfocus-rod', 'qianxin-rod', 'antiy-rod', 'dbappsecurity-rod',
]

const articleSources = [
  { value: 'freebuf', label: 'FreeBuf (安全脉搏)' },
  { value: 'securityweek', label: 'SecurityWeek' },
  { value: 'hackernews', label: 'Hacker News' },
  { value: 'qianxin-weekly', label: '奇安信热点周报' },
  { value: 'venustech', label: '绿盟科技' },
  { value: 'xianzhi', label: '先知社区' },
]

const query = reactive({
  page: 1,
  page_size: 20,
  status: '' as TaskStatus | '',
  type: '' as TaskType | '',
})

const vulnTask = reactive({
  sources: [] as string[],
  page_limit: 1,
  enable_github: false,
  proxy: '',
})

const articleTask = reactive({
  sources: [] as string[],
  limit: 10,
})

const availableSources = computed(() => {
  return selectedTaskType.value === 'vuln_crawl' ? vulnSources : articleSources.map(s => s.value)
})

function toggleSource(s: string) {
  if (selectedTaskType.value === 'vuln_crawl') {
    const idx = vulnTask.sources.indexOf(s)
    if (idx === -1) vulnTask.sources.push(s)
    else vulnTask.sources.splice(idx, 1)
  } else {
    const idx = articleTask.sources.indexOf(s)
    if (idx === -1) articleTask.sources.push(s)
    else articleTask.sources.splice(idx, 1)
  }
}

function isSourceSelected(s: string): boolean {
  if (selectedTaskType.value === 'vuln_crawl') {
    return vulnTask.sources.includes(s)
  } else {
    return articleTask.sources.includes(s)
  }
}

function getCurrentSources(): string[] {
  if (selectedTaskType.value === 'vuln_crawl') {
    return vulnTask.sources
  } else {
    return articleTask.sources
  }
}

function resetForm() {
  vulnTask.sources = []
  vulnTask.page_limit = 1
  vulnTask.enable_github = false
  vulnTask.proxy = ''
  articleTask.sources = []
  articleTask.limit = 10
}

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, any> = { page: query.page, page_size: query.page_size }
    if (query.status) params.status = query.status
    if (query.type) params.type = query.type
    const res = await nodeApi.listTasks(params)
    items.value = res?.items || []
    total.value = res?.total || 0
  } catch (error) {
    logger.error('Failed to load tasks:', error)
    items.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

async function createTask() {
  const sources = getCurrentSources()
  if (!sources.length) return
  creating.value = true
  try {
    if (selectedTaskType.value === 'vuln_crawl') {
      await nodeApi.createVulnCrawlTask({
        sources: vulnTask.sources,
        page_limit: vulnTask.page_limit,
        enable_github: vulnTask.enable_github,
        proxy: vulnTask.proxy || undefined,
      })
    } else {
      await nodeApi.createArticleCrawlTask({
        sources: articleTask.sources,
        limit: articleTask.limit,
      })
    }
    showCreateModal.value = false
    resetForm()
    await fetchData()
    ElMessage.success('任务创建成功')
  } catch (error) {
    logger.error('Failed to create task:', error)
    ElMessage.error('任务创建失败')
  } finally {
    creating.value = false
  }
}

async function handleDeleteTask(task: Task) {
  try {
    await ElMessageBox.confirm(
      `确定要删除任务 ${task.task_id?.slice(0, 8)}... 吗？`,
      '删除任务',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' }
    )
    await nodeApi.deleteTask(task.id)
    ElMessage.success('任务已删除')
    await fetchData()
  } catch (error: any) {
    if (error !== 'cancel') {
      logger.error('Failed to delete task:', error)
      ElMessage.error('删除失败')
    }
  }
}

async function handleStopTask(task: Task) {
  try {
    await ElMessageBox.confirm(
      `确定要停止任务 ${task.task_id?.slice(0, 8)}... 吗？`,
      '停止任务',
      { confirmButtonText: '停止', cancelButtonText: '取消', type: 'warning' }
    )
    await nodeApi.stopTask(task.id)
    ElMessage.success('任务已停止')
    await fetchData()
  } catch (error: any) {
    if (error !== 'cancel') {
      logger.error('Failed to stop task:', error)
      ElMessage.error('停止失败')
    }
  }
}

async function handleViewDetails(task: Task) {
  selectedTask.value = task
  showDetailModal.value = true
  logsLoading.value = true
  try {
    await nodeApi.getTask(task.id)
    taskLogs.value = [
      `[${new Date(task.created_at).toLocaleString()}] 任务已创建`,
      `[${new Date(task.updated_at).toLocaleString()}] 状态: ${task.status}`,
      task.status === 'done' ? `[${new Date(task.finished_at || '').toLocaleString()}] 任务完成` : '',
      task.error ? `[ERROR] ${task.error}` : '',
    ].filter(Boolean)
  } catch (error) {
    logger.error('Failed to get task details:', error)
    taskLogs.value = ['获取任务详情失败']
  } finally {
    logsLoading.value = false
  }
}

function canStopTask(task: Task): boolean {
  return ['pending', 'dispatched', 'running'].includes(task.status)
}

watch(selectedTaskType, () => { resetForm() })
watch(() => query.status, () => { query.page = 1; fetchData() })
watch(() => query.type, () => { query.page = 1; fetchData() })
onMounted(fetchData)

const statusClass: Record<string, string> = {
  pending: 'status-pending',
  dispatched: 'status-dispatched',
  running: 'status-running',
  done: 'status-done',
  failed: 'status-failed',
}

const statusLabel: Record<string, string> = {
  pending: '等待',
  dispatched: '已分发',
  running: '运行中',
  done: '完成',
  failed: '失败',
}

const typeLabel: Record<string, string> = {
  vuln_crawl: '漏洞爬取',
  article_crawl: '文章爬取',
}

const totalPages = () => Math.ceil(total.value / query.page_size)

function getTaskTypeLabel(typeValue: string): string {
  return typeLabel[typeValue] || typeValue
}
</script>

<template>
  <div class="ops-page">
    <!-- 合并的工具栏 + 筛选区 -->
    <div class="toolbar">
      <!-- 左侧：统计信息 -->
      <span class="toolbar-info">共 {{ total }} 条任务记录</span>

      <!-- 中间：筛选条件 -->
      <div class="toolbar-filters">
        <select v-model="query.type" class="filter-select">
          <option value="">全部类型</option>
          <option v-for="t in taskTypes" :key="t.value" :value="t.value">{{ t.label }}</option>
        </select>
        <select v-model="query.status" class="filter-select">
          <option value="">全部状态</option>
          <option v-for="[val, lbl] in Object.entries(statusLabel)" :key="val" :value="val">{{ lbl }}</option>
        </select>
      </div>

      <!-- 右侧：操作按钮 -->
      <div class="toolbar-actions">
        <button class="btn-icon" @click="fetchData" title="刷新">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M23 4v6h-6M1 20v-6h6"/>
            <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
          </svg>
        </button>
        <button class="btn-primary" @click="showCreateModal = true">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19"/>
            <line x1="5" y1="12" x2="19" y2="12"/>
          </svg>
          新建任务
        </button>
      </div>
    </div>

    <!-- 任务表格 -->
    <div class="table-container">
      <div v-if="loading" class="loading-state">
        <div v-for="i in 6" :key="i" class="skeleton-row"></div>
      </div>

      <div v-else-if="items.length === 0" class="empty-state">
        <div class="empty-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" y1="8" x2="12" y2="12"/>
            <line x1="12" y1="16" x2="12.01" y2="16"/>
          </svg>
        </div>
        <p class="empty-text">暂无任务</p>
        <p class="empty-hint">点击右上角按钮创建新任务</p>
      </div>

      <table v-else class="data-table">
        <thead>
          <tr>
            <th>任务 ID</th>
            <th>类型</th>
            <th>状态</th>
            <th>分配节点</th>
            <th>进度</th>
            <th>数据源</th>
            <th>创建时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="task in items" :key="task.id">
            <td><code class="cve-code">{{ task.task_id?.slice(0, 8) }}…</code></td>
            <td>
              <span class="type-tag" :class="task.type === 'vuln_crawl' ? 'type-vuln' : 'type-article'">
                {{ getTaskTypeLabel(task.type) }}
              </span>
            </td>
            <td>
              <span class="status-badge" :class="statusClass[task.status]">
                {{ statusLabel[task.status] }}
              </span>
            </td>
            <td><span class="text-secondary">{{ task.assigned_to || '—' }}</span></td>
            <td>
              <div class="progress-cell">
                <div class="progress-bar">
                  <div class="progress-fill" :style="{ width: task.progress + '%' }"></div>
                </div>
                <span class="progress-text">{{ task.progress }}%</span>
              </div>
            </td>
            <td>
              <div class="source-tags">
                <span v-for="s in (task.payload?.sources as string[] ?? []).slice(0, 2)" :key="s" class="source-tag">
                  {{ s }}
                </span>
                <span v-if="(task.payload?.sources as string[] ?? []).length > 2" class="source-more">
                  +{{ (task.payload?.sources as string[] ?? []).length - 2 }}
                </span>
              </div>
            </td>
            <td><span class="text-secondary">{{ new Date(task.created_at).toLocaleString('zh-CN') }}</span></td>
            <td>
              <div class="action-buttons">
                <button class="action-btn" title="查看详情" @click="handleViewDetails(task)">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                    <circle cx="12" cy="12" r="3"/>
                  </svg>
                </button>
                <button v-if="canStopTask(task)" class="action-btn warning" title="停止任务" @click="handleStopTask(task)">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="6" y="6" width="12" height="12"/>
                  </svg>
                </button>
                <button class="action-btn danger" title="删除任务" @click="handleDeleteTask(task)">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="3 6 5 6 21 6"/>
                    <path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/>
                  </svg>
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>

      <!-- 分页 -->
      <div v-if="total > query.page_size" class="pagination">
        <span class="page-info">第 {{ (query.page - 1) * query.page_size + 1 }} - {{ Math.min(query.page * query.page_size, total) }} 条 / 共 {{ total }} 条</span>
        <div class="page-controls">
          <button class="page-btn" :disabled="query.page <= 1" @click="() => { query.page--; fetchData() }">上一页</button>
          <span class="page-num">{{ query.page }} / {{ totalPages() }}</span>
          <button class="page-btn" :disabled="query.page >= totalPages()" @click="() => { query.page++; fetchData() }">下一页</button>
        </div>
      </div>
    </div>

    <!-- 创建任务弹窗 -->
    <Teleport to="body">
      <Transition name="fade">
        <div v-if="showCreateModal" class="modal-overlay" @click="showCreateModal = false">
          <div class="modal-content create-modal" @click.stop>
            <div class="modal-header">
              <h2 class="modal-title">新建任务</h2>
              <button class="modal-close" @click="showCreateModal = false">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18"/>
                  <line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
              </button>
            </div>
            <div class="modal-body">
              <div class="form-group">
                <label class="form-label">任务类型 <span class="required">*</span></label>
                <div class="type-cards">
                  <div
                    v-for="t in taskTypes" :key="t.value"
                    :class="['type-card', { active: selectedTaskType === t.value }]"
                    @click="selectedTaskType = t.value as TaskType"
                  >
                    <div class="type-card-radio"></div>
                    <span>{{ t.label }}</span>
                  </div>
                </div>
              </div>

              <div class="form-group">
                <label class="form-label">选择数据源 <span class="required">*</span></label>
                <div class="source-grid">
                  <template v-if="selectedTaskType === 'vuln_crawl'">
                    <button
                      v-for="s in availableSources" :key="s"
                      :class="['source-btn', { active: isSourceSelected(s) }]"
                      @click="toggleSource(s)"
                    >{{ s }}</button>
                  </template>
                  <template v-else>
                    <button
                      v-for="s in articleSources" :key="s.value"
                      :class="['source-btn', 'source-btn-lg', { active: isSourceSelected(s.value) }]"
                      @click="toggleSource(s.value)"
                    >{{ s.label }}</button>
                  </template>
                </div>
              </div>

              <template v-if="selectedTaskType === 'vuln_crawl'">
                <div class="form-group">
                  <label class="form-label">爬取页数</label>
                  <input v-model.number="vulnTask.page_limit" type="number" min="1" max="10" class="form-input w-24" />
                </div>
                <div class="form-group">
                  <label class="form-checkbox">
                    <input v-model="vulnTask.enable_github" type="checkbox" />
                    <span>启用 GitHub POC 搜索</span>
                  </label>
                </div>
                <div class="form-group">
                  <label class="form-label">代理（可选）</label>
                  <input v-model="vulnTask.proxy" class="form-input" placeholder="http://127.0.0.1:7890" />
                </div>
              </template>

              <template v-else>
                <div class="form-group">
                  <label class="form-label">爬取数量</label>
                  <input v-model.number="articleTask.limit" type="number" min="1" max="50" class="form-input w-24" />
                </div>
              </template>
            </div>
            <div class="modal-footer">
              <button class="btn-secondary" @click="showCreateModal = false">取消</button>
              <button class="btn-primary" :disabled="creating || !getCurrentSources().length" @click="createTask">
                {{ creating ? '创建中…' : '创建任务' }}
              </button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- 任务详情弹窗 -->
    <Teleport to="body">
      <Transition name="fade">
        <div v-if="showDetailModal && selectedTask" class="modal-overlay" @click="showDetailModal = false">
          <div class="modal-content detail-modal" @click.stop>
            <div class="modal-header">
              <h2 class="modal-title">任务详情</h2>
              <button class="modal-close" @click="showDetailModal = false">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18"/>
                  <line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
              </button>
            </div>
            <div class="modal-body">
              <!-- 基本信息 -->
              <div class="detail-section">
                <h3 class="section-title">基本信息</h3>
                <div class="detail-grid">
                  <div class="detail-item">
                    <span class="detail-label">任务 ID</span>
                    <code class="detail-value mono">{{ selectedTask.task_id }}</code>
                  </div>
                  <div class="detail-item">
                    <span class="detail-label">任务类型</span>
                    <span class="type-tag" :class="selectedTask.type === 'vuln_crawl' ? 'type-vuln' : 'type-article'">
                      {{ getTaskTypeLabel(selectedTask.type) }}
                    </span>
                  </div>
                  <div class="detail-item">
                    <span class="detail-label">状态</span>
                    <span class="status-badge" :class="statusClass[selectedTask.status]">{{ statusLabel[selectedTask.status] }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="detail-label">分配节点</span>
                    <span class="detail-value">{{ selectedTask.assigned_to || '—' }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="detail-label">创建时间</span>
                    <span class="detail-value">{{ new Date(selectedTask.created_at).toLocaleString('zh-CN') }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="detail-label">完成时间</span>
                    <span class="detail-value">{{ selectedTask.finished_at ? new Date(selectedTask.finished_at).toLocaleString('zh-CN') : '—' }}</span>
                  </div>
                </div>
              </div>

              <!-- 执行信息 -->
              <div class="detail-section">
                <h3 class="section-title">执行信息</h3>
                <div class="detail-grid">
                  <div class="detail-item">
                    <span class="detail-label">执行进度</span>
                    <div class="progress-inline">
                      <div class="progress-bar-mini">
                        <div class="progress-fill" :style="{ width: selectedTask.progress + '%' }"></div>
                      </div>
                      <span>{{ selectedTask.progress }}%</span>
                    </div>
                  </div>
                  <div class="detail-item">
                    <span class="detail-label">更新时间</span>
                    <span class="detail-value">{{ new Date(selectedTask.updated_at).toLocaleString('zh-CN') }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="detail-label">节点 ID</span>
                    <span class="detail-value">{{ selectedTask.assigned_to || '—' }}</span>
                  </div>
                </div>
              </div>

              <!-- 错误信息 -->
              <div v-if="selectedTask.error" class="error-box">
                <div class="error-header">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"/>
                    <line x1="12" y1="8" x2="12" y2="12"/>
                    <line x1="12" y1="16" x2="12.01" y2="16"/>
                  </svg>
                  <span>错误信息</span>
                </div>
                <p class="error-text">{{ selectedTask.error }}</p>
              </div>

              <!-- Payload 配置 -->
              <div v-if="selectedTask.payload" class="detail-section">
                <h3 class="section-title">任务配置</h3>
                <div class="payload-box">
                  <pre class="payload-content">{{ JSON.stringify(selectedTask.payload, null, 2) }}</pre>
                </div>
              </div>

              <!-- 执行日志 -->
              <div class="detail-section">
                <h3 class="section-title">执行日志</h3>
                <div v-if="logsLoading" class="log-loading">加载中...</div>
                <div v-else-if="taskLogs.length === 0" class="log-empty">暂无日志</div>
                <div v-else class="log-list">
                  <div v-for="(log, idx) in taskLogs" :key="idx" class="log-item">{{ log }}</div>
                </div>
              </div>
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
  font-size: 0.85rem;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.toolbar-filters {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex: 1;
}

.toolbar-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-shrink: 0;
}

.btn-icon {
  width: 34px;
  height: 34px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}
.btn-icon:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.filter-select {
  padding: 0.45rem 0.75rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 0.8rem;
  color: var(--text-primary);
  cursor: pointer;
}
.filter-select:focus {
  outline: none;
  border-color: var(--primary);
}

.btn-primary {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.6rem 1rem;
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
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

.btn-secondary {
  padding: 0.6rem 1rem;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  color: var(--color-text-secondary);
  font-size: 0.85rem;
  cursor: pointer;
  transition: all 0.2s ease;
}
.btn-secondary:hover { background: var(--color-bg-secondary); }

/* 表格容器 */
.table-container {
  background: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

/* 表格 */
.data-table {
  width: 100%;
  border-collapse: collapse;
}

.data-table th {
  padding: 0.75rem 1rem;
  text-align: left;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text-tertiary);
  background: var(--color-bg-secondary);
  border-bottom: 1px solid var(--color-border);
}

.data-table td {
  padding: 0.75rem 1rem;
  font-size: 0.8rem;
  color: var(--color-text-primary);
  border-bottom: 1px solid var(--color-border-light);
}

.data-table tr:hover td {
  background: var(--color-bg-secondary);
}

.data-table tr:last-child td {
  border-bottom: none;
}

/* 状态徽章 */
.status-badge {
  display: inline-block;
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
  font-size: 0.65rem;
  font-weight: 600;
}

.status-pending { background: rgba(107, 114, 128, 0.15); color: #6b7280; }
.status-dispatched { background: rgba(59, 130, 246, 0.15); color: #3b82f6; }
.status-running { background: rgba(234, 179, 8, 0.15); color: #eab308; }
.status-done { background: rgba(34, 197, 94, 0.15); color: #22c55e; }
.status-failed { background: rgba(239, 68, 68, 0.15); color: #ef4444; }

/* 类型标签 */
.type-tag {
  font-size: 0.7rem;
  font-weight: 500;
  padding: 0.2rem 0.4rem;
  border-radius: 4px;
}

.type-vuln { background: rgba(249, 115, 22, 0.15); color: #f97316; }
.type-article { background: rgba(6, 182, 212, 0.15); color: #06b6d4; }

/* 代码 */
.cve-code {
  font-family: monospace;
  font-size: 0.75rem;
  color: var(--color-primary);
}

/* 文本 */
.text-secondary {
  color: var(--color-text-secondary);
  font-size: 0.75rem;
}

/* 进度 */
.progress-cell {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.progress-bar {
  flex: 1;
  height: 4px;
  background: var(--color-bg-tertiary);
  border-radius: 2px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: var(--color-primary);
  border-radius: 2px;
  transition: width 0.5s ease;
}

.progress-text {
  font-size: 0.7rem;
  color: var(--color-text-tertiary);
  min-width: 32px;
  text-align: right;
}

/* 数据源标签 */
.source-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem;
}

.source-tag {
  font-size: 0.6rem;
  padding: 0.15rem 0.35rem;
  background: rgba(59, 130, 246, 0.1);
  color: var(--color-primary);
  border-radius: 3px;
}

.source-more {
  font-size: 0.6rem;
  color: var(--color-text-tertiary);
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 0.25rem;
}

.action-btn {
  width: 28px;
  height: 28px;
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

.action-btn:hover { background: var(--color-bg-tertiary); color: var(--color-text-primary); }
.action-btn.warning { color: #eab308; }
.action-btn.warning:hover { background: rgba(234, 179, 8, 0.15); }
.action-btn.danger { color: #ef4444; }
.action-btn.danger:hover { background: rgba(239, 68, 68, 0.15); }

/* 分页 */
.pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem;
  border-top: 1px solid var(--color-border-light);
}

.page-info {
  font-size: 0.8rem;
  color: var(--color-text-tertiary);
}

.page-controls {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.page-btn {
  padding: 0.5rem 1rem;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  color: var(--color-text-secondary);
  font-size: 0.8rem;
  cursor: pointer;
  transition: all 0.2s ease;
}
.page-btn:hover:not(:disabled) { background: var(--color-primary); border-color: var(--color-primary); color: white; }
.page-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.page-num {
  font-size: 0.85rem;
  color: var(--color-text-secondary);
}

/* 骨架屏 */
.loading-state {
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.skeleton-row {
  height: 40px;
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
  max-width: 520px;
  max-height: 90vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.create-modal {
  max-width: 560px;
}

.detail-modal {
  max-width: 640px;
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

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  padding: 1rem 1.25rem;
  border-top: 1px solid var(--color-border);
}

/* 表单 */
.form-group {
  margin-bottom: 1rem;
}

.form-label {
  display: block;
  font-size: 0.8rem;
  color: var(--color-text-secondary);
  margin-bottom: 0.5rem;
}

.required {
  color: #ef4444;
}

.form-input {
  padding: 0.6rem 0.75rem;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: 0.85rem;
  color: var(--color-text-primary);
  width: 100%;
}

.form-input:focus {
  outline: none;
  border-color: var(--color-primary);
}

.w-24 { width: 96px; }

.form-checkbox {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.85rem;
  color: var(--color-text-secondary);
  cursor: pointer;
}

/* 类型卡片 */
.type-cards {
  display: flex;
  gap: 0.75rem;
}

.type-card {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  font-size: 0.85rem;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}

.type-card:hover {
  border-color: var(--color-primary);
}

.type-card.active {
  background: rgba(59, 130, 246, 0.1);
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.type-card-radio {
  width: 16px;
  height: 16px;
  border: 2px solid currentColor;
  border-radius: 50%;
  position: relative;
}

.type-card.active .type-card-radio::after {
  content: '';
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 8px;
  height: 8px;
  background: var(--color-primary);
  border-radius: 50%;
}

/* 数据源网格 */
.source-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  max-height: 160px;
  overflow-y: auto;
  padding: 0.5rem;
  background: var(--color-bg-secondary);
  border-radius: var(--radius-md);
}

.source-btn {
  padding: 0.35rem 0.6rem;
  background: var(--color-bg-tertiary);
  border: 1px solid var(--color-border-light);
  border-radius: 4px;
  font-size: 0.7rem;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}

.source-btn-lg {
  padding: 0.5rem 0.75rem;
  font-size: 0.8rem;
}

.source-btn:hover {
  border-color: var(--color-primary);
}

.source-btn.active {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: white;
}

/* 详情 */
.detail-section {
  margin-bottom: 1.25rem;
}

.detail-section:last-child {
  margin-bottom: 0;
}

.section-title {
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  margin: 0 0 0.75rem;
  padding-bottom: 0.5rem;
  border-bottom: 1px solid var(--color-border-light);
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
}

.detail-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.detail-label {
  font-size: 0.7rem;
  color: var(--color-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.detail-value {
  font-size: 0.85rem;
  color: var(--color-text-primary);
}

.detail-value.mono {
  font-family: monospace;
  font-size: 0.7rem;
  word-break: break-all;
  color: var(--color-primary);
}

/* 进度 */
.progress-inline {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.progress-bar-mini {
  flex: 1;
  height: 4px;
  background: var(--color-bg-tertiary);
  border-radius: 2px;
  overflow: hidden;
}

/* 错误框 */
.error-box {
  padding: 0.75rem;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.2);
  border-radius: var(--radius-md);
  margin-bottom: 1rem;
}

.error-header {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  color: #ef4444;
  font-size: 0.75rem;
  font-weight: 500;
  margin-bottom: 0.5rem;
}

.error-text {
  font-size: 0.8rem;
  color: #fca5a5;
  margin: 0;
  line-height: 1.5;
}

/* Payload */
.payload-box {
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  padding: 0.75rem;
  max-height: 200px;
  overflow: auto;
}

.payload-content {
  margin: 0;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 0.7rem;
  color: var(--color-text-secondary);
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
}

/* 日志 */
.log-section {
  margin-top: 1rem;
}

.log-label {
  display: block;
  font-size: 0.75rem;
  color: var(--color-text-tertiary);
  margin-bottom: 0.5rem;
}

.log-loading, .log-empty {
  text-align: center;
  padding: 1rem;
  color: var(--color-text-tertiary);
  font-size: 0.85rem;
}

.log-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.log-item {
  padding: 0.5rem 0.75rem;
  background: var(--color-bg-secondary);
  border-radius: var(--radius-sm);
  font-family: monospace;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

/* 过渡 */
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from, .fade-leave-to { opacity: 0; }

/* 响应式 */
@media (max-width: 768px) {
  .filter-row { flex-wrap: wrap; }
  .data-table { display: block; overflow-x: auto; }
  .detail-grid { grid-template-columns: 1fr; }
  .type-cards { flex-direction: column; }
}
</style>
