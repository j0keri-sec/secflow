<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { articleApi } from '@/api/article'
import type { Article } from '@/types'
import { Search, Timer, ArrowRight } from '@element-plus/icons-vue'

const router = useRouter()

const loading = ref(false)

const query = reactive({
  page: 1,
  page_size: 12,
  keyword: '',
  source: '',
})

// 分类
const categories = ['全部', '漏洞披露', '数据泄露', '勒索软件', 'APT攻击', '行业动态']
const activeCategory = ref('全部')

// 文章列表
const articles = ref<Article[]>([])
const total = ref(0)

function resetQuery() {
  Object.assign(query, { page: 1, keyword: '', source: '' })
  activeCategory.value = '全部'
  fetchArticles()
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

// 将后端文章数据映射为前端需要的格式
const newsList = computed(() => {
  return articles.value.map(article => ({
    ...article,
    title: article.title,
    description: article.summary || '暂无摘要',
    category: article.tags?.[0] || '行业动态',
    source: article.source || '安全资讯',
    time: article.published_at ? new Date(article.published_at).toLocaleDateString('zh-CN') : '未知',
    timeAgo: formatTimeAgo(article.published_at),
    image: article.image || 'https://images.unsplash.com/photo-1550751827-4bd374c3f58b?w=400&h=250&fit=crop',
    url: article.url
  }))
})

const filteredNews = computed(() => {
  if (activeCategory.value === '全部') return newsList.value
  return newsList.value.filter(n => n.category === activeCategory.value)
})

// 获取文章数据
async function fetchArticles() {
  loading.value = true
  try {
    const params: Record<string, any> = { page: query.page, page_size: 50 }
    if (query.keyword) params.keyword = query.keyword
    if (query.source) params.source = query.source
    const res = await articleApi.list(params)
    if (res?.items) {
      articles.value = res.items
      total.value = res.total || 0
    }
  } catch (error) {
    console.error('Failed to fetch articles:', error)
  } finally {
    loading.value = false
  }
}

// 打开文章详情
function openArticleDetail(article: Article) {
  router.push({ name: 'ArticleDetail', params: { id: article.id } })
}

onMounted(() => {
  fetchArticles()
})
</script>

<template>
  <div class="news-page">
    <!-- 搜索筛选区 -->
    <div class="filter-section">
      <div class="filter-row">
        <div class="search-box">
          <el-icon class="search-icon"><Search /></el-icon>
          <input 
            v-model="query.keyword" 
            type="text" 
            placeholder="搜索标题、摘要..." 
            class="search-input"
            @keyup.enter="() => { query.page = 1; fetchArticles() }"
          />
        </div>
        
        <div class="filter-group">
          <input 
            v-model="query.source" 
            type="text" 
            placeholder="数据源" 
            class="filter-input"
          />
          
          <button class="btn-search" @click="() => { query.page = 1; fetchArticles() }">
            <el-icon><Search /></el-icon>
            搜索
          </button>
          <button class="btn-reset" @click="resetQuery">重置</button>
        </div>
      </div>
    </div>

    <!-- 文章列表 -->
    <div class="article-list">
      <div class="list-header">
        <h3 class="list-title">
          <span class="list-title-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V9a2 2 0 012-2h2a2 2 0 012 2v9a2 2 0 01-2 2h-2z"/>
            </svg>
          </span>
          安全资讯
        </h3>
        <span class="list-count">共 {{ total }} 条记录</span>
      </div>

      <!-- 分类筛选标签 -->
      <div class="quick-filters">
        <button 
          v-for="cat in categories" 
          :key="cat"
          :class="['filter-tag', { active: activeCategory === cat }]"
          @click="activeCategory = cat"
        >
          {{ cat }}
        </button>
      </div>

      <!-- 加载状态 -->
      <div v-if="loading" class="loading-grid">
        <div v-for="i in 6" :key="i" class="skeleton-card">
          <div class="skeleton-image"></div>
          <div class="skeleton-content">
            <div class="skeleton-line w-2/3"></div>
            <div class="skeleton-line w-full"></div>
            <div class="skeleton-line w-1/2"></div>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else-if="filteredNews.length === 0" class="empty-state">
        <div class="empty-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V9a2 2 0 012-2h2a2 2 0 012 2v9a2 2 0 01-2 2h-2z"/>
          </svg>
        </div>
        <p class="empty-text">暂无相关资讯</p>
        <p class="empty-hint">尝试选择其他分类或调整搜索条件</p>
      </div>

      <!-- 文章网格 -->
      <div v-else class="article-grid">
        <article
          v-for="item in filteredNews"
          :key="item.id"
          class="article-card"
          @click="openArticleDetail(item)"
        >
          <div class="article-image">
            <img :src="item.image" :alt="item.title" />
            <span class="article-tag">{{ item.category }}</span>
          </div>
          <div class="article-content">
            <h4 class="article-title">{{ item.title }}</h4>
            <p class="article-desc">{{ item.description }}</p>
            <div class="article-footer">
              <span class="article-source">{{ item.source }}</span>
              <span class="article-time">
                <el-icon :size="12"><Timer /></el-icon>
                {{ item.timeAgo }}
              </span>
            </div>
          </div>
          <div class="article-arrow">
            <el-icon><ArrowRight /></el-icon>
          </div>
        </article>
      </div>
    </div>
  </div>
</template>

<style scoped>
.news-page {
  padding: 0 0 2rem;
  animation: fadeIn 0.4s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

/* 搜索筛选区 */
.filter-section {
  background: var(--color-bg-primary);
  border: 1px solid var(--color-border);
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
  color: var(--color-text-tertiary);
}

.search-input {
  width: 100%;
  padding: 0.6rem 1rem 0.6rem 2.25rem;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: 0.85rem;
  color: var(--color-text-primary);
  transition: all 0.2s ease;
}

.search-input:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.search-input::placeholder {
  color: var(--color-text-tertiary);
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.filter-input {
  padding: 0.6rem 0.75rem;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: 0.8rem;
  color: var(--color-text-primary);
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
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  color: var(--color-text-secondary);
  font-size: 0.85rem;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-reset:hover {
  background: var(--color-bg-secondary);
}

/* 文章列表容器 */
.article-list {
  background: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: 1.5rem;
}

.list-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--color-border-light);
}

.list-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 1rem;
  font-weight: 600;
  color: var(--color-text-primary);
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
  color: var(--color-text-tertiary);
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
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-full);
  font-size: 0.8rem;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}

.filter-tag:hover {
  background: var(--color-bg-tertiary);
}

.filter-tag.active {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: white;
}

/* 加载骨架屏 */
.loading-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
}

.skeleton-card {
  background: var(--color-bg-secondary);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.skeleton-image {
  height: 160px;
  background: linear-gradient(90deg, var(--color-bg-tertiary) 25%, var(--color-border) 50%, var(--color-bg-tertiary) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}

.skeleton-content {
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.skeleton-line {
  height: 12px;
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

/* 文章卡片网格 */
.article-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
}

.article-card {
  position: relative;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  overflow: hidden;
  cursor: pointer;
  transition: all 0.3s ease;
}

.article-card:hover {
  border-color: var(--color-primary);
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.article-card:hover .article-arrow {
  opacity: 1;
  transform: translateX(0);
}

.article-image {
  position: relative;
  height: 160px;
  overflow: hidden;
}

.article-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.3s;
}

.article-card:hover .article-image img {
  transform: scale(1.05);
}

.article-tag {
  position: absolute;
  top: 0.75rem;
  left: 0.75rem;
  background: rgba(59, 130, 246, 0.9);
  color: white;
  padding: 0.25rem 0.6rem;
  border-radius: 4px;
  font-size: 0.7rem;
  font-weight: 600;
}

.article-content {
  padding: 1rem;
}

.article-title {
  font-size: 0.9rem;
  font-weight: 500;
  color: var(--color-text-primary);
  margin: 0 0 0.5rem;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.article-desc {
  font-size: 0.75rem;
  color: var(--color-text-tertiary);
  margin: 0 0 0.75rem;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.article-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.7rem;
  color: var(--color-text-tertiary);
}

.article-source {
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.article-time {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.article-arrow {
  position: absolute;
  top: 50%;
  right: 0.75rem;
  transform: translateY(-50%) translateX(-4px);
  opacity: 0;
  transition: all 0.2s ease;
  color: var(--color-primary);
}

/* Drawer Transition */
.slide-enter-from .relative,
.slide-leave-to .relative {
  transform: translateX(100%);
}

.slide-enter-to .relative,
.slide-leave-from .relative {
  transform: translateX(0);
}

.slide-enter-active .relative,
.slide-leave-active .relative {
  transition: transform 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

/* Prose Content */
.prose-content {
  line-height: 1.8;
}

/* 响应式 */
@media (max-width: 1200px) {
  .article-grid, .loading-grid {
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
  
  .article-grid, .loading-grid {
    grid-template-columns: 1fr;
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
