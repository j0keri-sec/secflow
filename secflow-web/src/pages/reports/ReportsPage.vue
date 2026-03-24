<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { systemApi } from '@/api/system'

interface Report {
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

const items = ref<Report[]>([])
const total = ref(0)
const loading = ref(true)
const showCreateModal = ref(false)
const creating = ref(false)

const query = reactive({ page: 1, page_size: 20 })
const newReport = reactive({
  title: '',
  description: '',
  type: 'weekly',
  date_from: '',
  date_to: '',
})

async function fetchData() {
  loading.value = true
  try {
    const res = await systemApi.listReports(query)
    items.value = res.items
    total.value = res.total
  } finally {
    loading.value = false
  }
}

onMounted(fetchData)

async function createReport() {
  if (!newReport.title) return
  creating.value = true
  try {
    await systemApi.createReport(newReport)
    showCreateModal.value = false
    await fetchData()
  } finally {
    creating.value = false
  }
}

async function deleteReport(id: string) {
  if (!confirm('确认删除该报告？')) return
  await systemApi.deleteReport(id)
  await fetchData()
}

function downloadReport(report: Report) {
  if (!report.file_url) return
  window.open(report.file_url, '_blank')
}

const totalPages = () => Math.ceil(total.value / query.page_size)

const statusLabel: Record<string, string> = {
  draft: '草稿',
  generating: '生成中',
  done: '完成',
  failed: '失败',
}

const statusClass: Record<string, string> = {
  draft: 'status-draft',
  generating: 'status-generating',
  done: 'status-done',
  failed: 'status-failed',
}

const typeLabel: Record<string, string> = {
  daily: '日报',
  weekly: '周报',
  monthly: '月报',
  custom: '自定义',
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
        共 {{ total }} 份报告
      </span>
      <span class="toolbar-hint">点击报告行可查看详情</span>
      <button class="btn-primary" @click="showCreateModal = true">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="5" x2="12" y2="19"/>
          <line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        生成报告
      </button>
    </div>

    <!-- 报告表格 -->
    <div class="table-container">
      <div v-if="loading" class="loading-state">
        <div v-for="i in 5" :key="i" class="skeleton-row"></div>
      </div>

      <div v-else-if="items.length === 0" class="empty-state">
        <div class="empty-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
            <polyline points="14 2 14 8 20 8"/>
          </svg>
        </div>
        <p class="empty-text">暂无报告</p>
        <p class="empty-hint">点击右上角按钮生成第一份报告</p>
      </div>

      <table v-else class="data-table">
        <thead>
          <tr>
            <th>报告标题</th>
            <th>类型</th>
            <th>状态</th>
            <th>创建人</th>
            <th>创建时间</th>
            <th>完成时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="report in items" :key="report.id">
            <td>
              <div class="report-title-cell">
                <span class="report-name">{{ report.title }}</span>
                <span v-if="report.description" class="report-desc">{{ report.description }}</span>
              </div>
            </td>
            <td><span class="type-tag">{{ typeLabel[report.type] || report.type }}</span></td>
            <td>
              <span class="status-badge" :class="statusClass[report.status]">
                {{ statusLabel[report.status] }}
              </span>
            </td>
            <td><span class="text-secondary">{{ report.created_by }}</span></td>
            <td><span class="text-secondary">{{ new Date(report.created_at).toLocaleString('zh-CN') }}</span></td>
            <td><span class="text-secondary">{{ report.finished_at ? new Date(report.finished_at).toLocaleString('zh-CN') : '—' }}</span></td>
            <td>
              <div class="action-buttons">
                <button v-if="report.status === 'done'" class="btn-download" @click="downloadReport(report)">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/>
                    <polyline points="7 10 12 15 17 10"/>
                    <line x1="12" y1="15" x2="12" y2="3"/>
                  </svg>
                  下载
                </button>
                <button class="btn-delete" @click="deleteReport(report.id)">删除</button>
              </div>
            </td>
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

    <!-- 创建报告弹窗 -->
    <Teleport to="body">
      <Transition name="fade">
        <div v-if="showCreateModal" class="modal-overlay" @click="showCreateModal = false">
          <div class="modal-content" @click.stop>
            <div class="modal-header">
              <h2 class="modal-title">生成报告</h2>
              <button class="modal-close" @click="showCreateModal = false">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18"/>
                  <line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
              </button>
            </div>
            <div class="modal-body">
              <div class="form-group">
                <label class="form-label">报告标题 <span class="required">*</span></label>
                <input v-model="newReport.title" class="form-input" placeholder="2026年3月漏洞周报" />
              </div>
              <div class="form-group">
                <label class="form-label">报告类型</label>
                <select v-model="newReport.type" class="form-input">
                  <option v-for="[v, l] in Object.entries(typeLabel)" :key="v" :value="v">{{ l }}</option>
                </select>
              </div>
              <div class="form-row">
                <div class="form-group">
                  <label class="form-label">开始日期</label>
                  <input v-model="newReport.date_from" type="date" class="form-input" />
                </div>
                <div class="form-group">
                  <label class="form-label">结束日期</label>
                  <input v-model="newReport.date_to" type="date" class="form-input" />
                </div>
              </div>
              <div class="form-group">
                <label class="form-label">描述（可选）</label>
                <textarea v-model="newReport.description" class="form-textarea" placeholder="报告描述..."></textarea>
              </div>
            </div>
            <div class="modal-footer">
              <button class="btn-secondary" @click="showCreateModal = false">取消</button>
              <button class="btn-primary" :disabled="creating || !newReport.title" @click="createReport">
                {{ creating ? '生成中...' : '生成' }}
              </button>
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
  font-size: 0.8rem;
  color: var(--text-tertiary);
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
  font-size: 0.8rem;
}

/* 报告标题单元格 */
.report-title-cell {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.report-name {
  font-weight: 500;
  color: var(--text-primary);
}

.report-desc {
  font-size: 0.75rem;
  color: var(--text-tertiary);
  max-width: 300px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 类型标签 */
.type-tag {
  font-size: 0.7rem;
  padding: 0.2rem 0.5rem;
  background: rgba(59, 130, 246, 0.1);
  color: var(--color-primary);
  border-radius: 4px;
}

/* 状态徽章 */
.status-badge {
  display: inline-block;
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
  font-size: 0.65rem;
  font-weight: 600;
}

.status-draft { background: rgba(107, 114, 128, 0.15); color: #6b7280; }
.status-generating { background: rgba(234, 179, 8, 0.15); color: #eab308; }
.status-done { background: rgba(34, 197, 94, 0.15); color: #22c55e; }
.status-failed { background: rgba(239, 68, 68, 0.15); color: #ef4444; }

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 0.5rem;
}

.btn-download, .btn-delete {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.4rem 0.75rem;
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-download {
  background: rgba(59, 130, 246, 0.1);
  border: 1px solid rgba(59, 130, 246, 0.2);
  color: var(--color-primary);
}
.btn-download:hover {
  background: rgba(59, 130, 246, 0.2);
}

.btn-delete {
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.2);
  color: #ef4444;
}
.btn-delete:hover {
  background: rgba(239, 68, 68, 0.2);
}

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
  height: 48px;
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
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  width: 100%;
  max-width: 480px;
  max-height: 90vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--border);
}

.modal-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
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
  color: var(--text-tertiary);
  cursor: pointer;
  transition: all 0.2s ease;
}
.modal-close:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
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
  border-top: 1px solid var(--border);
}

/* 表单 */
.form-group {
  margin-bottom: 1rem;
}

.form-label {
  display: block;
  font-size: 0.8rem;
  color: var(--text-secondary);
  margin-bottom: 0.5rem;
}

.required {
  color: #ef4444;
}

.form-input, .form-textarea {
  padding: 0.6rem 0.75rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 0.85rem;
  color: var(--text-primary);
  width: 100%;
  transition: border-color 0.2s ease;
}

.form-input:focus, .form-textarea:focus {
  outline: none;
  border-color: var(--primary);
}

.form-input[type="date"] {
  color-scheme: light;
}

[data-theme="dark"] .form-input[type="date"] {
  color-scheme: dark;
}

.form-textarea {
  resize: none;
  height: 80px;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

/* 过渡 */
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from, .fade-leave-to { opacity: 0; }

/* 响应式 */
@media (max-width: 768px) {
  .data-table { display: block; overflow-x: auto; }
  .form-row { grid-template-columns: 1fr; }
}
</style>
