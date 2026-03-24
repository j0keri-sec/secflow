<script setup lang="ts">
import { ref, reactive, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import type { VulnRecord, SeverityLevel } from '@/types'
import { vulnApi } from '@/api/vuln'
import { Search, Timer, ArrowRight } from '@element-plus/icons-vue'

const router = useRouter()

// 漏洞数据
const items = ref<VulnRecord[]>([])
const total = ref(0)
const loading = ref(false)

// 查询条件
const query = reactive({
  page: 1,
  page_size: 12,
  keyword: '',
  severity: '' as SeverityLevel | '',
  source: '',
  cve: '',
  pushed: '' as '' | 'true' | 'false',
})

const severities: Array<SeverityLevel | ''> = ['', '严重', '高危', '中危', '低危']

// 最新高危漏洞（顶部横幅，只显示3个）
const criticalVulns = ref<VulnRecord[]>([])

// 获取最新高危漏洞
async function fetchCriticalVulns() {
  try {
    // 获取严重和高危漏洞
    const criticalRes = await vulnApi.list({ severity: '严重', page_size: 2 })
    const highRes = await vulnApi.list({ severity: '高危', page_size: 1 })
    
    const criticalItems = criticalRes?.items || []
    const highItems = highRes?.items || []
    
    // 合并并限制为3个
    criticalVulns.value = [...criticalItems, ...highItems].slice(0, 3)
  } catch (error) {
    console.error('Failed to load critical vulns:', error)
  }
}

// 格式化时间
function formatTimeAgo(dateStr?: string): string {
  if (!dateStr) return '未知'
  const date = new Date(dateStr)
  const now = new Date()
  const diff = Math.floor((now.getTime() - date.getTime()) / 1000)
  
  if (diff < 60) return '刚刚'
  if (diff < 3600) return `${Math.floor(diff / 60)}分钟前`
  if (diff < 86400) return `${Math.floor(diff / 3600)}小时前`
  if (diff < 604800) return `${Math.floor(diff / 86400)}天前`
  return `${Math.floor(diff / 604800)}周前`
}

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, any> = {
      page: query.page,
      page_size: query.page_size,
    }
    if (query.keyword) params.keyword = query.keyword
    if (query.severity) params.severity = query.severity
    if (query.source) params.source = query.source
    if (query.cve) params.cve = query.cve
    if (query.pushed !== '') params.pushed = query.pushed
    const res = await vulnApi.list(params)
    items.value = res?.items || []
    total.value = res?.total || 0
  } catch (error) {
    console.error('Failed to load vulnerability data:', error)
    items.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function resetQuery() {
  Object.assign(query, { page: 1, keyword: '', severity: '', source: '', cve: '', pushed: '' })
}

watch(() => [query.severity, query.source, query.pushed], () => { query.page = 1; fetchData() })

onMounted(async () => {
  await Promise.all([fetchCriticalVulns(), fetchData()])
})

const totalPages = () => Math.ceil(total.value / query.page_size)

// 跳转到详情
function goToDetail(id: string) {
  router.push(`/vulns/${id}`)
}

// 获取严重程度样式
function getSeverityStyle(severity: string) {
  const map: Record<string, { bg: string, text: string, border: string }> = {
    '严重': { bg: 'rgba(239, 68, 68, 0.15)', text: '#ef4444', border: 'rgba(239, 68, 68, 0.3)' },
    '高危': { bg: 'rgba(249, 115, 22, 0.15)', text: '#f97316', border: 'rgba(249, 115, 22, 0.3)' },
    '中危': { bg: 'rgba(234, 179, 8, 0.15)', text: '#eab308', border: 'rgba(234, 179, 8, 0.3)' },
    '低危': { bg: 'rgba(34, 197, 94, 0.15)', text: '#22c55e', border: 'rgba(34, 197, 94, 0.3)' }
  }
  return map[severity] || map['低危']
}
</script>

<template>
  <div class="vuln-page">
    <!-- 搜索筛选区 -->
    <div class="filter-section">
      <div class="filter-row">
        <div class="search-box">
          <el-icon class="search-icon"><Search /></el-icon>
          <input 
            v-model="query.keyword" 
            type="text" 
            placeholder="搜索 CVE编号、漏洞标题、描述..." 
            class="search-input"
            @keyup.enter="() => { query.page = 1; fetchData() }"
          />
        </div>
        
        <div class="filter-group">
          <select v-model="query.severity" class="filter-select">
            <option v-for="s in severities" :key="s" :value="s">{{ s || '全部严重程度' }}</option>
          </select>
          
          <input 
            v-model="query.cve" 
            type="text" 
            placeholder="CVE编号" 
            class="filter-input"
            @keyup.enter="() => { query.page = 1; fetchData() }"
          />
          
          <input 
            v-model="query.source" 
            type="text" 
            placeholder="数据源" 
            class="filter-input"
          />
          
          <button class="btn-search" @click="() => { query.page = 1; fetchData() }">
            <el-icon><Search /></el-icon>
            搜索
          </button>
          <button class="btn-reset" @click="resetQuery">重置</button>
        </div>
      </div>
    </div>

    <!-- 紧急漏洞横幅 -->
    <div v-if="criticalVulns.length > 0" class="critical-banner">
      <div class="banner-header">
        <div class="banner-title">
          <span class="critical-icon">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
              <line x1="12" y1="9" x2="12" y2="13"/>
              <line x1="12" y1="17" x2="12.01" y2="17"/>
            </svg>
          </span>
          <span>紧急关注</span>
        </div>
        <span class="banner-hint">需要优先处理的安全漏洞</span>
      </div>
      <div class="critical-list">
        <div 
          v-for="vuln in criticalVulns" 
          :key="vuln.id" 
          class="critical-card"
          @click="goToDetail(vuln.id)"
        >
          <div class="critical-header">
            <span class="cve-id">{{ vuln.cve || 'N/A' }}</span>
            <span 
              class="severity-tag"
              :style="{ 
                background: getSeverityStyle(vuln.severity).bg,
                color: getSeverityStyle(vuln.severity).text,
                border: `1px solid ${getSeverityStyle(vuln.severity).border}`
              }"
            >
              {{ vuln.severity || '未知' }}
            </span>
          </div>
          <h4 class="critical-title">{{ vuln.title }}</h4>
          <div class="critical-meta">
            <span class="source-tag">{{ vuln.source || '未知' }}</span>
            <span class="time-tag">
              <el-icon :size="12"><Timer /></el-icon>
              {{ formatTimeAgo(vuln.created_at) }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- 漏洞列表 -->
    <div class="vuln-list">
      <div class="list-header">
        <h3 class="list-title">
          <span class="list-title-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <path d="M12 6v6l4 2"/>
            </svg>
          </span>
          最新漏洞情报
        </h3>
        <span class="list-count">共 {{ total }} 条记录</span>
      </div>

      <!-- 快速筛选标签 -->
      <div class="quick-filters">
        <button 
          class="filter-tag" 
          :class="{ active: query.severity === '严重' }"
          @click="query.severity = query.severity === '严重' ? '' as any : '严重'; query.page = 1; fetchData()"
        >
          <span class="filter-dot critical"></span>
          严重
        </button>
        <button 
          class="filter-tag"
          :class="{ active: query.severity === '高危' }"
          @click="query.severity = query.severity === '高危' ? '' as any : '高危'; query.page = 1; fetchData()"
        >
          <span class="filter-dot high"></span>
          高危
        </button>
        <button 
          class="filter-tag"
          :class="{ active: query.severity === '中危' }"
          @click="query.severity = query.severity === '中危' ? '' as any : '中危'; query.page = 1; fetchData()"
        >
          <span class="filter-dot medium"></span>
          中危
        </button>
        <button 
          class="filter-tag"
          :class="{ active: query.severity === '低危' }"
          @click="query.severity = query.severity === '低危' ? '' as any : '低危'; query.page = 1; fetchData()"
        >
          <span class="filter-dot low"></span>
          低危
        </button>
      </div>

      <div v-if="loading" class="loading-grid">
        <div v-for="i in 6" :key="i" class="skeleton-card">
          <div class="skeleton-line w-1/3"></div>
          <div class="skeleton-line w-2/3"></div>
          <div class="skeleton-line w-1/2"></div>
        </div>
      </div>

      <div v-else-if="items.length === 0" class="empty-state">
        <div class="empty-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
          </svg>
        </div>
        <p class="empty-text">未找到匹配的漏洞情报</p>
        <p class="empty-hint">尝试调整筛选条件或搜索关键词</p>
      </div>

      <div v-else class="vuln-grid">
        <div 
          v-for="vuln in items" 
          :key="vuln.id" 
          class="vuln-card"
          @click="goToDetail(vuln.id)"
        >
          <div class="vuln-card-header">
            <span class="vuln-cve">{{ vuln.cve || 'N/A' }}</span>
            <span 
              class="vuln-severity"
              :style="{ 
                background: getSeverityStyle(vuln.severity).bg,
                color: getSeverityStyle(vuln.severity).text
              }"
            >
              {{ vuln.severity || '未知' }}
            </span>
          </div>
          <h4 class="vuln-title">{{ vuln.title }}</h4>
          <p class="vuln-desc">{{ vuln.description || '暂无描述' }}</p>
          <div class="vuln-footer">
            <span class="vuln-source">{{ vuln.source || vuln.from }}</span>
            <span class="vuln-date">{{ vuln.disclosure || vuln.created_at }}</span>
          </div>
          <div v-if="vuln.tags?.length" class="vuln-tags">
            <span v-for="tag in vuln.tags.slice(0, 2)" :key="tag" class="vuln-tag">{{ tag }}</span>
          </div>
          <div class="vuln-arrow">
            <el-icon><ArrowRight /></el-icon>
          </div>
        </div>
      </div>

      <!-- 分页 -->
      <div v-if="total > query.page_size" class="pagination">
        <span class="page-info">
          第 {{ (query.page - 1) * query.page_size + 1 }} - {{ Math.min(query.page * query.page_size, total) }} 条 / 共 {{ total }} 条
        </span>
        <div class="page-controls">
          <button 
            class="page-btn" 
            :disabled="query.page <= 1"
            @click="() => { query.page--; fetchData() }"
          >
            上一页
          </button>
          <span class="page-num">{{ query.page }} / {{ totalPages() }}</span>
          <button 
            class="page-btn"
            :disabled="query.page >= totalPages()"
            @click="() => { query.page++; fetchData() }"
          >
            下一页
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.vuln-page {
  padding: 0 0 2rem;
  animation: fadeIn 0.4s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

/* 搜索筛选区 */
.filter-section {
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 1rem 1.5rem;
  margin-bottom: 1.5rem;
}

.filter-row {
  display: flex;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
}

.search-box {
  position: relative;
  flex: 1;
  min-width: 280px;
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
  padding: 0.6rem 1rem 0.6rem 2.25rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 0.85rem;
  color: var(--text-primary);
  transition: all 0.2s ease;
}

.search-input:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.search-input::placeholder {
  color: var(--text-tertiary);
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.filter-select, .filter-input {
  padding: 0.6rem 0.75rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 0.8rem;
  color: var(--text-primary);
}

.filter-input {
  width: 120px;
}

.btn-search {
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

.btn-search:hover {
  opacity: 0.9;
  transform: translateY(-1px);
}

.btn-reset {
  padding: 0.6rem 1rem;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  font-size: 0.85rem;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-reset:hover {
  background: var(--bg-secondary);
}

/* 紧急漏洞横幅 */
.critical-banner {
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 1.25rem 1.5rem;
  margin-bottom: 1.5rem;
  overflow: hidden;
}

.banner-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--border-light);
}

.banner-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
}

.critical-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: linear-gradient(135deg, #ef4444 0%, #f97316 100%);
  border-radius: var(--radius-sm);
  color: white;
}

.banner-hint {
  font-size: 0.8rem;
  color: var(--text-tertiary);
}

.critical-list {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
  max-height: 180px;
  overflow: hidden;
}

.critical-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-light);
  border-radius: var(--radius-md);
  padding: 1rem;
  cursor: pointer;
  transition: all 0.3s ease;
  overflow: hidden;
}

.critical-card:hover {
  border-color: var(--color-primary);
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.critical-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.5rem;
}

.cve-id {
  font-family: monospace;
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--color-primary);
}

.severity-tag {
  font-size: 0.7rem;
  font-weight: 600;
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
}

.critical-title {
  font-size: 0.9rem;
  font-weight: 500;
  color: var(--text-primary);
  margin: 0 0 0.5rem;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.critical-meta {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  font-size: 0.7rem;
  color: var(--text-tertiary);
}

.source-tag, .time-tag {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

/* 漏洞列表 */
.vuln-list {
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 1.5rem;
}

.list-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border-light);
}

.list-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.list-title-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-primary);
}

.list-count {
  font-size: 0.8rem;
  color: var(--text-tertiary);
}

/* 快速筛选标签 */
.quick-filters {
  display: flex;
  gap: 0.75rem;
  margin-bottom: 1.25rem;
  flex-wrap: wrap;
}

.filter-tag {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.5rem 1rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-light);
  border-radius: var(--radius-full);
  font-size: 0.8rem;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}

.filter-tag:hover {
  background: var(--bg-tertiary);
}

.filter-tag.active {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: white;
}

.filter-tag.active .filter-dot {
  background: white !important;
}

.filter-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.filter-dot.critical { background: #ef4444; }
.filter-dot.high { background: #f97316; }
.filter-dot.medium { background: #eab308; }
.filter-dot.low { background: #22c55e; }

/* 加载骨架屏 */
.loading-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
}

.skeleton-card {
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.skeleton-line {
  height: 12px;
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

/* 漏洞卡片网格 */
.vuln-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
}

.vuln-card {
  position: relative;
  background: var(--bg-secondary);
  border: 1px solid var(--border-light);
  border-radius: var(--radius-md);
  padding: 1rem;
  cursor: pointer;
  transition: all 0.3s ease;
}

.vuln-card:hover {
  border-color: var(--color-primary);
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.vuln-card:hover .vuln-arrow {
  opacity: 1;
  transform: translateX(0);
}

.vuln-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.5rem;
}

.vuln-cve {
  font-family: monospace;
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--color-primary);
}

.vuln-severity {
  font-size: 0.65rem;
  font-weight: 600;
  padding: 0.15rem 0.4rem;
  border-radius: 4px;
}

.vuln-title {
  font-size: 0.9rem;
  font-weight: 500;
  color: var(--text-primary);
  margin: 0 0 0.5rem;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  padding-right: 1.5rem;
}

.vuln-desc {
  font-size: 0.75rem;
  color: var(--text-tertiary);
  margin: 0 0 0.75rem;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.vuln-footer {
  display: flex;
  justify-content: space-between;
  font-size: 0.7rem;
  color: var(--text-tertiary);
}

.vuln-tags {
  display: flex;
  gap: 0.4rem;
  margin-top: 0.5rem;
}

.vuln-tag {
  font-size: 0.6rem;
  color: var(--text-tertiary);
  background: var(--bg-tertiary);
  padding: 0.15rem 0.35rem;
  border-radius: 3px;
}

.vuln-arrow {
  position: absolute;
  top: 50%;
  right: 0.75rem;
  transform: translateY(-50%) translateX(-4px);
  opacity: 0;
  transition: all 0.2s ease;
  color: var(--color-primary);
}

/* 分页 */
.pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 1.5rem;
  padding-top: 1rem;
  border-top: 1px solid var(--border-light);
}

.page-info {
  font-size: 0.8rem;
  color: var(--text-tertiary);
}

.page-controls {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.page-btn {
  padding: 0.5rem 1rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  font-size: 0.8rem;
  cursor: pointer;
  transition: all 0.2s ease;
}

.page-btn:hover:not(:disabled) {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: white;
}

.page-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.page-num {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

/* 响应式 */
@media (max-width: 1200px) {
  .vuln-grid, .loading-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  
  .critical-list {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .filter-row {
    flex-direction: column;
  }
  
  .search-box {
    width: 100%;
  }
  
  .filter-group {
    width: 100%;
    justify-content: flex-start;
  }
  
  .vuln-grid, .loading-grid {
    grid-template-columns: 1fr;
  }
  
  .critical-list {
    grid-template-columns: 1fr;
    max-height: none;
  }
  
  .quick-filters {
    overflow-x: auto;
    flex-wrap: nowrap;
    padding-bottom: 0.5rem;
  }
  
  .filter-tag {
    flex-shrink: 0;
  }
}
</style>
