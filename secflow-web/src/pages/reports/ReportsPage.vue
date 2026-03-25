<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { systemApi, type DataSource, type AIModel, type Report } from '@/api/system'
import { ElMessage } from 'element-plus'

// 列表数据
const items = ref<Report[]>([])
const total = ref(0)
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(20)

// 模态框状态
const showConfigModal = ref(false)
const previewHtml = ref('')
const showPreview = ref(false)
const generating = ref(false)

// 可选数据
const dataSources = ref<DataSource[]>([])
const aiModels = ref<AIModel[]>([])

// 报告配置
const config = reactive({
  title: '',
  type: 'weekly',
  sources: [] as string[],
  dateFrom: '',
  dateTo: '',
  aiModel: '',
  formats: ['html'] as string[],
})

// 预设时间选项
const selectedTimeRange = ref('week')

// 计算日期范围
const computedDateRange = computed(() => {
  const now = new Date()
  const to = now.toISOString().split('T')[0]
  let from: string

  switch (selectedTimeRange.value) {
    case 'week':
      from = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000).toISOString().split('T')[0]
      break
    case 'month':
      from = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000).toISOString().split('T')[0]
      break
    default:
      from = config.dateFrom || new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000).toISOString().split('T')[0]
  }

  return { from, to }
})

// 状态映射
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

// 加载数据
async function fetchData() {
  loading.value = true
  try {
    const res = await systemApi.listReports({ page: currentPage.value, page_size: pageSize.value })
    items.value = res.items
    total.value = res.total
  } catch (e) {
    console.error('Failed to fetch reports:', e)
  } finally {
    loading.value = false
  }
}

// 加载可选配置
async function loadConfigOptions() {
  try {
    const [dsRes, aiRes] = await Promise.all([
      systemApi.getDataSources(),
      systemApi.getAIModels(),
    ])
    dataSources.value = dsRes.sources
    aiModels.value = aiRes.models
  } catch (e) {
    console.error('Failed to load config options:', e)
  }
}

// 切换数据源
function toggleSource(id: string) {
  const idx = config.sources.indexOf(id)
  if (idx >= 0) {
    config.sources.splice(idx, 1)
  } else {
    config.sources.push(id)
  }
}

// 打开生成模态框
function openConfigModal() {
  // 设置默认标题
  const now = new Date()
  const weekNum = getWeekNumber(now)
  config.title = `第${weekNum}期安全周报`

  // 默认选择最近一周
  selectedTimeRange.value = 'week'
  config.sources = ['nvd', 'cnvd']
  config.aiModel = ''
  config.formats = ['html']

  showConfigModal.value = true
}

// 预览报告
async function previewReport() {
  if (!config.title) {
    ElMessage.warning('请输入报告标题')
    return
  }

  generating.value = true
  showPreview.value = true
  try {
    const range = computedDateRange.value
    const html = await systemApi.previewReport({
      type: config.type,
      sources: config.sources,
      date_from: range.from,
      date_to: range.to,
    })
    previewHtml.value = html
  } catch (e) {
    console.error('Preview failed:', e)
    ElMessage.error('预览生成失败')
    showPreview.value = false
  } finally {
    generating.value = false
  }
}

// 下载报告
async function downloadReport() {
  if (!config.title) {
    ElMessage.warning('请输入报告标题')
    return
  }

  generating.value = true
  try {
    const range = computedDateRange.value
    const blob = await systemApi.generateReport({
      title: config.title,
      type: config.type,
      sources: config.sources,
      date_from: range.from,
      date_to: range.to,
      ai_model: config.aiModel,
      formats: config.formats,
    })

    // 创建下载链接
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    const ext = config.formats.includes('md') ? 'md' : 'html'
    a.download = `${config.title}.${ext}`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)

    ElMessage.success('报告下载成功')
    showConfigModal.value = false
  } catch (e) {
    console.error('Download failed:', e)
    ElMessage.error('报告下载失败')
  } finally {
    generating.value = false
  }
}

// 删除报告
async function deleteReport(id: string) {
  if (!confirm('确认删除该报告？')) return
  try {
    await systemApi.deleteReport(id)
    ElMessage.success('删除成功')
    await fetchData()
  } catch (e) {
    ElMessage.error('删除失败')
  }
}

// 获取周数
function getWeekNumber(date: Date): number {
  const firstDayOfYear = new Date(date.getFullYear(), 0, 1)
  const pastDaysOfYear = (date.getTime() - firstDayOfYear.getTime()) / 86400000
  return Math.ceil((pastDaysOfYear + firstDayOfYear.getDay() + 1) / 7)
}

// 分页变化
function onPageChange(page: number) {
  currentPage.value = page
  fetchData()
}

function onSizeChange(size: number) {
  pageSize.value = size
  fetchData()
}

// 初始化
onMounted(() => {
  fetchData()
  loadConfigOptions()
})
</script>

<template>
  <div class="reports-page">
    <!-- 页面标题 -->
    <div class="page-header">
      <h2>报告管理</h2>
      <div class="header-actions">
        <el-button type="primary" @click="openConfigModal">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
            <polyline points="14 2 14 8 20 8"/>
            <line x1="12" y1="18" x2="12" y2="12"/>
            <line x1="9" y1="15" x2="15" y2="15"/>
          </svg>
          生成报告
        </el-button>
      </div>
    </div>

    <!-- 报告列表 -->
    <div class="card">
      <el-table :data="items" v-loading="loading" stripe>
        <el-table-column prop="title" label="报告标题" min-width="200"/>
        <el-table-column prop="type" label="类型" width="100">
          <template #default="{ row }">
            <span class="type-badge">{{ typeLabel[row.type] || row.type }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <span :class="['status-badge', statusClass[row.status]]">
              {{ statusLabel[row.status] || row.status }}
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180"/>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="downloadReport">下载</el-button>
            <el-button link type="danger" size="small" @click="deleteReport(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next"
          @current-change="onPageChange"
          @size-change="onSizeChange"
        />
      </div>
    </div>

    <!-- 生成报告配置模态框 -->
    <el-dialog v-model="showConfigModal" title="生成报告" width="700px" :close-on-click-modal="false">
      <div class="config-form">
        <!-- 基本信息 -->
        <div class="form-section">
          <h4>基本信息</h4>
          <el-form label-width="100px">
            <el-form-item label="报告标题">
              <el-input v-model="config.title" placeholder="例如：第45期安全周报"/>
            </el-form-item>
            <el-form-item label="报告类型">
              <el-select v-model="config.type" style="width: 100%">
                <el-option value="daily" label="日报"/>
                <el-option value="weekly" label="周报"/>
                <el-option value="monthly" label="月报"/>
              </el-select>
            </el-form-item>
          </el-form>
        </div>

        <!-- 时间范围 -->
        <div class="form-section">
          <h4>时间范围</h4>
          <el-form label-width="100px">
            <el-form-item label="时间范围">
              <el-radio-group v-model="selectedTimeRange">
                <el-radio value="week">最近一周</el-radio>
                <el-radio value="month">最近一月</el-radio>
                <el-radio value="custom">自定义</el-radio>
              </el-radio-group>
            </el-form-item>
            <el-form-item v-if="selectedTimeRange === 'custom'" label="自定义日期">
              <div style="display: flex; gap: 10px;">
                <el-date-picker
                  v-model="config.dateFrom"
                  type="date"
                  placeholder="开始日期"
                  value-format="YYYY-MM-DD"
                  style="width: 150px"
                />
                <span style="line-height: 32px;">至</span>
                <el-date-picker
                  v-model="config.dateTo"
                  type="date"
                  placeholder="结束日期"
                  value-format="YYYY-MM-DD"
                  style="width: 150px"
                />
              </div>
            </el-form-item>
            <el-form-item label="实际范围" v-if="selectedTimeRange !== 'custom'">
              <span class="date-range-display">
                {{ computedDateRange.from }} 至 {{ computedDateRange.to }}
              </span>
            </el-form-item>
          </el-form>
        </div>

        <!-- 数据来源 -->
        <div class="form-section">
          <h4>数据来源</h4>
          <div class="source-grid">
            <div
              v-for="source in dataSources"
              :key="source.id"
              :class="['source-card', { selected: config.sources.includes(source.id) }]"
              @click="toggleSource(source.id)"
            >
              <div class="source-name">{{ source.name }}</div>
              <div class="source-desc">{{ source.description }}</div>
            </div>
          </div>
        </div>

        <!-- AI 模型 -->
        <div class="form-section">
          <h4>AI 智能摘要</h4>
          <div class="ai-models">
            <div
              v-for="model in aiModels"
              :key="model.id"
              :class="['model-card', { selected: config.aiModel === model.id }]"
              @click="config.aiModel = model.id"
            >
              <div class="model-name">{{ model.name }}</div>
              <div class="model-desc">{{ model.description }}</div>
            </div>
          </div>
        </div>

        <!-- 导出格式 -->
        <div class="form-section">
          <h4>导出格式</h4>
          <el-checkbox-group v-model="config.formats">
            <el-checkbox value="html">HTML (推荐)</el-checkbox>
            <el-checkbox value="md">Markdown</el-checkbox>
          </el-checkbox-group>
        </div>
      </div>

      <template #footer>
        <el-button @click="showConfigModal = false">取消</el-button>
        <el-button @click="previewReport" :loading="generating" :disabled="!config.title">
          预览
        </el-button>
        <el-button type="primary" @click="downloadReport" :loading="generating" :disabled="!config.title">
          下载报告
        </el-button>
      </template>
    </el-dialog>

    <!-- 预览模态框 -->
    <el-dialog v-model="showPreview" title="报告预览" width="900px" fullscreen>
      <div class="preview-container" v-loading="generating">
        <iframe v-if="previewHtml" :srcdoc="previewHtml" class="preview-frame"/>
      </div>
    </el-dialog>
  </div>
</template>

<style scoped>
.reports-page {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #1a1a2e;
}

.header-actions {
  display: flex;
  gap: 10px;
}

.header-actions .el-button svg {
  margin-right: 6px;
}

.card {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.type-badge, .status-badge {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
}

.type-badge {
  background: #e8f4ff;
  color: #0066cc;
}

.status-draft { background: #f0f0f0; color: #666; }
.status-generating { background: #fff7e6; color: #fa8c16; }
.status-done { background: #f6ffed; color: #52c41a; }
.status-failed { background: #fff2f0; color: #ff4d4f; }

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

/* 配置表单样式 */
.config-form {
  max-height: 60vh;
  overflow-y: auto;
}

.form-section {
  margin-bottom: 24px;
  padding-bottom: 20px;
  border-bottom: 1px solid #f0f0f0;
}

.form-section:last-child {
  border-bottom: none;
  margin-bottom: 0;
}

.form-section h4 {
  margin: 0 0 16px 0;
  font-size: 14px;
  font-weight: 600;
  color: #1a1a2e;
}

.date-range-display {
  color: #666;
  font-size: 14px;
}

/* 数据源网格 */
.source-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}

.source-card {
  padding: 12px;
  border: 1px solid #d9d9d9;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.source-card:hover {
  border-color: #0066cc;
}

.source-card.selected {
  border-color: #0066cc;
  background: #f0f7ff;
}

.source-name {
  font-weight: 500;
  color: #1a1a2e;
  margin-bottom: 4px;
}

.source-desc {
  font-size: 12px;
  color: #999;
}

/* AI 模型 */
.ai-models {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}

.model-card {
  padding: 12px;
  border: 1px solid #d9d9d9;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.model-card:hover {
  border-color: #722ed1;
}

.model-card.selected {
  border-color: #722ed1;
  background: #f9f0ff;
}

.model-name {
  font-weight: 500;
  color: #1a1a2e;
  margin-bottom: 4px;
}

.model-desc {
  font-size: 12px;
  color: #999;
}

/* 预览容器 */
.preview-container {
  height: calc(100vh - 120px);
}

.preview-frame {
  width: 100%;
  height: 100%;
  border: none;
}
</style>
