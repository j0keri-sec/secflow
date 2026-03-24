<script setup lang="ts">
import { ref, reactive, onMounted, watch } from 'vue'
import { systemApi } from '@/api/system'
import { Search } from '@element-plus/icons-vue'

interface AuditLog {
  id: string
  user_id: string
  username: string
  action: string
  resource: string
  detail: string
  ip: string
  created_at: string
}

const items = ref<AuditLog[]>([])
const total = ref(0)
const loading = ref(false)

const query = reactive({
  page: 1,
  page_size: 30,
  keyword: '',
  action: '',
})

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, any> = { page: query.page, page_size: query.page_size }
    if (query.keyword) params.keyword = query.keyword
    if (query.action) params.action = query.action
    const res = await systemApi.listAuditLogs(params)
    items.value = res.items
    total.value = res.total
  } finally {
    loading.value = false
  }
}

watch(() => query.action, () => { query.page = 1; fetchData() })
onMounted(fetchData)

const totalPages = () => Math.ceil(total.value / query.page_size)

const actionColors: Record<string, string> = {
  login: 'action-login',
  logout: 'action-logout',
  create: 'action-create',
  update: 'action-update',
  delete: 'action-delete',
  push: 'action-push',
}

const actionLabels: Record<string, string> = {
  login: '登录',
  logout: '退出',
  create: '创建',
  update: '更新',
  delete: '删除',
  push: '推送',
}
</script>

<template>
  <div class="ops-page">
    <!-- 工具栏 -->
    <div class="toolbar">
      <span class="toolbar-info">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
          <polyline points="14 2 14 8 20 8"/>
          <line x1="16" y1="13" x2="8" y2="13"/>
          <line x1="16" y1="17" x2="8" y2="17"/>
        </svg>
        共 {{ total }} 条日志
      </span>
      <span class="toolbar-hint">系统操作将记录在此</span>
    </div>

    <!-- 筛选区 -->
    <div class="filter-section">
      <div class="filter-row">
        <div class="search-box">
          <el-icon class="search-icon"><Search /></el-icon>
          <input 
            v-model="query.keyword" 
            type="text" 
            placeholder="搜索用户名 / IP / 资源..." 
            class="search-input"
            @keyup.enter="() => { query.page = 1; fetchData() }"
          />
        </div>
        <select v-model="query.action" class="filter-select">
          <option value="">全部操作</option>
          <option v-for="[v, l] in Object.entries(actionLabels)" :key="v" :value="v">{{ l }}</option>
        </select>
        <button class="btn-primary" @click="() => { query.page = 1; fetchData() }">查询</button>
      </div>
    </div>

    <!-- 日志表格 -->
    <div class="table-container">
      <div v-if="loading" class="loading-state">
        <div v-for="i in 8" :key="i" class="skeleton-row"></div>
      </div>

      <div v-else-if="items.length === 0" class="empty-state">
        <div class="empty-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
            <polyline points="14 2 14 8 20 8"/>
          </svg>
        </div>
        <p class="empty-text">暂无日志</p>
        <p class="empty-hint">系统操作日志将显示在这里</p>
      </div>

      <table v-else class="data-table">
        <thead>
          <tr>
            <th>时间</th>
            <th>用户</th>
            <th>操作</th>
            <th>资源</th>
            <th>详情</th>
            <th>IP 地址</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="log in items" :key="log.id">
            <td><span class="text-secondary">{{ new Date(log.created_at).toLocaleString('zh-CN') }}</span></td>
            <td><span class="username">{{ log.username || '—' }}</span></td>
            <td>
              <span class="action-badge" :class="actionColors[log.action]">
                {{ actionLabels[log.action] || log.action }}
              </span>
            </td>
            <td><span class="text-secondary">{{ log.resource || '—' }}</span></td>
            <td><span class="detail-text">{{ log.detail || '—' }}</span></td>
            <td><code class="ip-code">{{ log.ip || '—' }}</code></td>
          </tr>
        </tbody>
      </table>

      <!-- 分页 -->
      <div v-if="total > query.page_size" class="pagination">
        <span class="page-info">共 {{ total }} 条</span>
        <div class="page-controls">
          <button class="page-btn" :disabled="query.page <= 1" @click="() => { query.page--; fetchData() }">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="15 18 9 12 15 6"/>
            </svg>
          </button>
          <span class="page-num">{{ query.page }} / {{ totalPages() }}</span>
          <button class="page-btn" :disabled="query.page >= totalPages()" @click="() => { query.page++; fetchData() }">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="9 18 15 12 9 6"/>
            </svg>
          </button>
        </div>
      </div>
    </div>
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
  font-size: 0.8rem;
  color: var(--text-tertiary);
}

/* 筛选区 */
.filter-section {
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 1rem 1.5rem;
  margin-bottom: 1rem;
}

.filter-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.search-box {
  position: relative;
  flex: 1;
  min-width: 200px;
}

.search-icon {
  position: absolute;
  left: 0.75rem;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-tertiary);
}

.search-input {
  width: 100%;
  padding: 0.6rem 0.75rem 0.6rem 2.25rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 0.85rem;
  color: var(--text-primary);
}

.search-input:focus {
  outline: none;
  border-color: var(--color-primary);
}

.filter-select {
  padding: 0.6rem 0.75rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 0.85rem;
  color: var(--text-primary);
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

/* 表格容器 */
.table-container {
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
}

.data-table th {
  padding: 0.75rem 1rem;
  text-align: left;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-tertiary);
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
}

.data-table td {
  padding: 0.75rem 1rem;
  font-size: 0.8rem;
  color: var(--text-primary);
  border-bottom: 1px solid var(--border-light);
}

.data-table tr:hover td {
  background: var(--bg-secondary);
}

.data-table tr:last-child td {
  border-bottom: none;
}

.text-secondary {
  color: var(--text-secondary);
  font-size: 0.75rem;
}

.username {
  font-weight: 500;
  color: var(--text-primary);
}

.detail-text {
  max-width: 300px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: block;
  color: var(--text-secondary);
  font-size: 0.75rem;
}

.ip-code {
  font-family: monospace;
  font-size: 0.75rem;
  color: var(--text-tertiary);
  background: var(--bg-tertiary);
  padding: 0.15rem 0.4rem;
  border-radius: 4px;
}

/* 操作徽章 */
.action-badge {
  display: inline-block;
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
  font-size: 0.65rem;
  font-weight: 600;
}

.action-login { background: rgba(34, 197, 94, 0.15); color: #22c55e; }
.action-logout { background: rgba(107, 114, 128, 0.15); color: #6b7280; }
.action-create { background: rgba(59, 130, 246, 0.15); color: #3b82f6; }
.action-update { background: rgba(234, 179, 8, 0.15); color: #eab308; }
.action-delete { background: rgba(239, 68, 68, 0.15); color: #ef4444; }
.action-push { background: rgba(168, 85, 247, 0.15); color: #a855f7; }

/* 分页 */
.pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem;
  border-top: 1px solid var(--border-light);
}

.page-info {
  font-size: 0.8rem;
  color: var(--text-tertiary);
}

.page-controls {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.page-btn {
  width: 32px;
  height: 32px;
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
.page-btn:hover:not(:disabled) { background: var(--color-primary); border-color: var(--color-primary); color: white; }
.page-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.page-num {
  font-size: 0.85rem;
  color: var(--text-secondary);
  padding: 0 0.5rem;
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
  background: linear-gradient(90deg, var(--bg-tertiary) 25%, var(--border) 50%, var(--bg-tertiary) 75%);
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
  color: var(--text-tertiary);
}

.empty-text {
  font-size: 1rem;
  color: var(--text-secondary);
  margin: 0 0 0.5rem;
}

.empty-hint {
  font-size: 0.85rem;
  color: var(--text-tertiary);
  margin: 0;
}

/* 响应式 */
@media (max-width: 768px) {
  .filter-row { flex-wrap: wrap; }
  .data-table { display: block; overflow-x: auto; }
}
</style>
