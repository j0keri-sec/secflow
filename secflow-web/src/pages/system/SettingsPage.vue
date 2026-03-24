<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { systemApi, type TaskSchedule } from '@/api/system'
import { ElMessage } from 'element-plus'

const loading = ref(false)
const saving = ref(false)
const schedules = ref<TaskSchedule[]>([])

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

const editForm = reactive({
  vuln_crawl: {
    enabled: false,
    interval: 30,
    sources: [] as string[],
  },
  article_crawl: {
    enabled: false,
    interval: 60,
    sources: [] as string[],
  },
})

async function fetchSchedules() {
  loading.value = true
  try {
    const data = await systemApi.getTaskSchedules()
    schedules.value = data
    // Populate edit form
    for (const s of data) {
      if (s.type === 'vuln_crawl') {
        editForm.vuln_crawl.enabled = s.enabled
        editForm.vuln_crawl.interval = s.interval
        editForm.vuln_crawl.sources = s.sources || []
      } else if (s.type === 'article_crawl') {
        editForm.article_crawl.enabled = s.enabled
        editForm.article_crawl.interval = s.interval
        editForm.article_crawl.sources = s.sources || []
      }
    }
  } catch (error) {
    console.error('Failed to load schedules:', error)
    ElMessage.error('加载配置失败')
  } finally {
    loading.value = false
  }
}

async function saveVulnSchedule() {
  saving.value = true
  try {
    await systemApi.updateTaskSchedule({
      type: 'vuln_crawl',
      enabled: editForm.vuln_crawl.enabled,
      interval: editForm.vuln_crawl.interval,
      sources: editForm.vuln_crawl.sources,
    })
    ElMessage.success('漏洞爬取调度已更新')
    await fetchSchedules()
  } catch (error) {
    console.error('Failed to save vuln schedule:', error)
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

async function saveArticleSchedule() {
  saving.value = true
  try {
    await systemApi.updateTaskSchedule({
      type: 'article_crawl',
      enabled: editForm.article_crawl.enabled,
      interval: editForm.article_crawl.interval,
      sources: editForm.article_crawl.sources,
    })
    ElMessage.success('文章爬取调度已更新')
    await fetchSchedules()
  } catch (error) {
    console.error('Failed to save article schedule:', error)
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

function toggleVulnSource(s: string) {
  const idx = editForm.vuln_crawl.sources.indexOf(s)
  if (idx === -1) {
    editForm.vuln_crawl.sources.push(s)
  } else {
    editForm.vuln_crawl.sources.splice(idx, 1)
  }
}

function toggleArticleSource(s: string) {
  const idx = editForm.article_crawl.sources.indexOf(s)
  if (idx === -1) {
    editForm.article_crawl.sources.push(s)
  } else {
    editForm.article_crawl.sources.splice(idx, 1)
  }
}

onMounted(fetchSchedules)
</script>

<template>
  <div class="settings-page">
    <div class="page-header">
      <h1 class="page-title">任务调度设置</h1>
      <p class="page-desc">配置定时任务执行的周期和数据源</p>
    </div>

    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <p>加载中...</p>
    </div>

    <div v-else class="settings-grid">
      <!-- Vuln Crawl Schedule -->
      <div class="settings-card">
        <div class="card-header">
          <h2 class="card-title">漏洞情报收集</h2>
          <label class="toggle-switch">
            <input type="checkbox" v-model="editForm.vuln_crawl.enabled" />
            <span class="toggle-slider"></span>
          </label>
        </div>

        <div class="card-body">
          <div class="form-group">
            <label class="form-label">执行周期（分钟）</label>
            <input
              v-model.number="editForm.vuln_crawl.interval"
              type="number"
              min="1"
              max="1440"
              class="form-input w-32"
              :disabled="!editForm.vuln_crawl.enabled"
            />
            <span class="form-hint">1-1440 分钟</span>
          </div>

          <div class="form-group">
            <label class="form-label">数据源</label>
            <div class="source-grid">
              <button
                v-for="s in vulnSources"
                :key="s"
                :class="['source-btn', { active: editForm.vuln_crawl.sources.includes(s) }]"
                :disabled="!editForm.vuln_crawl.enabled"
                @click="toggleVulnSource(s)"
              >{{ s }}</button>
            </div>
          </div>
        </div>

        <div class="card-footer">
          <button
            class="btn-primary"
            :disabled="saving || !editForm.vuln_crawl.enabled"
            @click="saveVulnSchedule"
          >
            {{ saving ? '保存中...' : '保存配置' }}
          </button>
        </div>
      </div>

      <!-- Article Crawl Schedule -->
      <div class="settings-card">
        <div class="card-header">
          <h2 class="card-title">安全资讯收集</h2>
          <label class="toggle-switch">
            <input type="checkbox" v-model="editForm.article_crawl.enabled" />
            <span class="toggle-slider"></span>
          </label>
        </div>

        <div class="card-body">
          <div class="form-group">
            <label class="form-label">执行周期（分钟）</label>
            <input
              v-model.number="editForm.article_crawl.interval"
              type="number"
              min="1"
              max="1440"
              class="form-input w-32"
              :disabled="!editForm.article_crawl.enabled"
            />
            <span class="form-hint">1-1440 分钟</span>
          </div>

          <div class="form-group">
            <label class="form-label">数据源</label>
            <div class="source-grid">
              <button
                v-for="s in articleSources"
                :key="s.value"
                :class="['source-btn', 'source-btn-lg', { active: editForm.article_crawl.sources.includes(s.value) }]"
                :disabled="!editForm.article_crawl.enabled"
                @click="toggleArticleSource(s.value)"
              >{{ s.label }}</button>
            </div>
          </div>
        </div>

        <div class="card-footer">
          <button
            class="btn-primary"
            :disabled="saving || !editForm.article_crawl.enabled"
            @click="saveArticleSchedule"
          >
            {{ saving ? '保存中...' : '保存配置' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-page {
  padding: 1.5rem;
  animation: fadeIn 0.4s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

.page-header {
  margin-bottom: 1.5rem;
}

.page-title {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 0.5rem;
}

.page-desc {
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin: 0;
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 3rem;
  color: var(--text-tertiary);
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--border);
  border-top-color: var(--primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 1rem;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.settings-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 1.5rem;
}

.settings-card {
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.25rem;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
}

.card-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.card-body {
  padding: 1.25rem;
}

.card-footer {
  padding: 1rem 1.25rem;
  border-top: 1px solid var(--border);
  background: var(--bg-secondary);
}

.form-group {
  margin-bottom: 1.25rem;
}

.form-group:last-child {
  margin-bottom: 0;
}

.form-label {
  display: block;
  font-size: 0.8rem;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: 0.5rem;
}

.form-input {
  padding: 0.6rem 0.75rem;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 0.85rem;
  color: var(--text-primary);
}

.form-input:focus {
  outline: none;
  border-color: var(--primary);
}

.form-input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.w-32 { width: 96px; }

.form-hint {
  display: inline-block;
  margin-left: 0.5rem;
  font-size: 0.75rem;
  color: var(--text-tertiary);
}

.source-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.source-btn {
  padding: 0.35rem 0.6rem;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: 4px;
  font-size: 0.7rem;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}

.source-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.source-btn-lg {
  padding: 0.5rem 0.75rem;
  font-size: 0.8rem;
}

.source-btn:hover:not(:disabled) {
  border-color: var(--primary);
}

.source-btn.active:not(:disabled) {
  background: var(--primary);
  border-color: var(--primary);
  color: white;
}

.btn-primary {
  padding: 0.6rem 1.25rem;
  background: var(--gradient-primary);
  border: none;
  border-radius: var(--radius-sm);
  color: white;
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.2s ease;
}

.btn-primary:hover:not(:disabled) {
  opacity: 0.9;
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Toggle Switch */
.toggle-switch {
  position: relative;
  display: inline-block;
  width: 44px;
  height: 24px;
}

.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  position: absolute;
  cursor: pointer;
  inset: 0;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: 24px;
  transition: all 0.3s ease;
}

.toggle-slider::before {
  content: '';
  position: absolute;
  height: 18px;
  width: 18px;
  left: 2px;
  bottom: 2px;
  background: white;
  border-radius: 50%;
  transition: transform 0.3s ease;
}

.toggle-switch input:checked + .toggle-slider {
  background: var(--primary);
  border-color: var(--primary);
}

.toggle-switch input:checked + .toggle-slider::before {
  transform: translateX(20px);
}

@media (max-width: 768px) {
  .settings-grid {
    grid-template-columns: 1fr;
  }
}
</style>
