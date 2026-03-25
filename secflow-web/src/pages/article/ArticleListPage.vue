<script setup lang="ts">
import logger from '@/utils/logger'
import { ref, reactive, onMounted } from 'vue'
import type { Article } from '@/types'
import { articleApi } from '@/api/article'

const items = ref<Article[]>([])
const total = ref(0)
const loading = ref(false)
const selectedArticle = ref<Article | null>(null)

// 可用的来源列表
const sources = ['先知社区', '嘶吼', 'FreeBuf', '安全客', '奇安信', '启明星辰']
const selectedSource = ref('')

const query = reactive({
  page: 1,
  page_size: 20,
  keyword: '',
  source: '',
})

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, any> = { page: query.page, page_size: query.page_size }
    if (query.keyword) params.keyword = query.keyword
    if (query.source) params.source = query.source
    const res = await articleApi.list(params)
    items.value = res?.items || []
    total.value = res?.total || 0
  } catch (error) {
    logger.error('Failed to load article data:', error)
    items.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function selectSource(source: string) {
  if (selectedSource.value === source) {
    selectedSource.value = ''
    query.source = ''
  } else {
    selectedSource.value = source
    query.source = source
  }
  query.page = 1
  fetchData()
}

function search() {
  query.page = 1
  fetchData()
}

function clearFilters() {
  query.keyword = ''
  query.source = ''
  selectedSource.value = ''
  query.page = 1
  fetchData()
}

function openDetail(a: Article) {
  selectedArticle.value = a
}

onMounted(fetchData)

const totalPages = () => Math.ceil(total.value / query.page_size)
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-semibold text-white">文章热点</h1>
        <p class="text-sm text-slate-400 mt-0.5">共 {{ total }} 篇技术文章</p>
      </div>
      <button class="btn-secondary flex items-center gap-2" @click="fetchData">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.582 0a2.003 2.003 0 004.418 0L20 10M4 4l5 5m5-5l-5 5m5-5H9.582"/>
        </svg>
        刷新
      </button>
    </div>

    <!-- Source Filter Chips -->
    <div class="card !p-4">
      <div class="flex items-center gap-2 mb-3">
        <span class="text-xs text-slate-400">来源筛选:</span>
        <button
          v-for="source in sources"
          :key="source"
          :class="['filter-chip', { active: selectedSource === source }]"
          @click="selectSource(source)"
        >
          {{ source }}
        </button>
        <button
          v-if="selectedSource"
          class="text-xs text-slate-500 hover:text-slate-300 ml-auto"
          @click="clearFilters"
        >
          清除筛选
        </button>
      </div>
      
      <!-- Search -->
      <div class="flex gap-3">
        <div class="flex-1">
          <input 
            v-model="query.keyword" 
            class="input" 
            placeholder="搜索文章标题..." 
            @keyup.enter="search"
          />
        </div>
        <button class="btn-primary" @click="search">搜索</button>
      </div>
    </div>

    <!-- Article Grid -->
    <div v-if="loading" class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
      <div v-for="i in 9" :key="i" class="card space-y-3">
        <div class="h-4 bg-surface rounded animate-pulse w-4/5" />
        <div class="h-3 bg-surface rounded animate-pulse w-full" />
        <div class="h-3 bg-surface rounded animate-pulse w-2/3" />
        <div class="flex gap-2 mt-2">
          <div class="h-5 w-12 bg-surface rounded animate-pulse" />
          <div class="h-5 w-16 bg-surface rounded animate-pulse" />
        </div>
      </div>
    </div>

    <div v-else class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
      <div v-if="!items.length" class="col-span-3 text-center text-slate-500 py-12">暂无数据</div>
      <div
        v-for="article in items" :key="article.id"
        class="card cursor-pointer hover:border-brand-500/50 transition-colors group"
        @click="openDetail(article)"
      >
        <div class="flex items-start justify-between gap-2 mb-2">
          <h3 class="text-sm font-medium text-slate-200 group-hover:text-white line-clamp-2 leading-snug">
            {{ article.title }}
          </h3>
          <span :class="article.pushed ? 'badge-online' : 'badge-offline'" class="badge shrink-0 mt-0.5">
            {{ article.pushed ? '已推送' : '未推送' }}
          </span>
        </div>
        <p class="text-xs text-slate-400 line-clamp-2 mb-3">{{ article.summary || '暂无摘要' }}</p>
        <div class="flex flex-wrap gap-1 mb-3">
          <span v-for="tag in (article.tags ?? []).slice(0, 3)" :key="tag"
            class="px-1.5 py-0.5 bg-brand-600/10 text-brand-400 text-xs rounded">{{ tag }}</span>
        </div>
        <div class="flex items-center justify-between text-xs text-slate-500">
          <span>{{ article.source }}</span>
          <span>{{ article.published_at ? new Date(article.published_at).toLocaleDateString('zh-CN') : '—' }}</span>
        </div>
      </div>
    </div>

    <!-- Pagination -->
    <div v-if="total > query.page_size" class="flex items-center justify-between">
      <span class="text-xs text-slate-500">共 {{ total }} 条</span>
      <div class="flex items-center gap-1">
        <button class="btn-secondary !px-2 !py-1" :disabled="query.page <= 1"
          @click="() => { query.page--; fetchData() }">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
          </svg>
        </button>
        <span class="px-3 py-1 text-sm text-slate-300">{{ query.page }} / {{ totalPages() }}</span>
        <button class="btn-secondary !px-2 !py-1" :disabled="query.page >= totalPages()"
          @click="() => { query.page++; fetchData() }">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
          </svg>
        </button>
      </div>
    </div>

    <!-- Article Detail Drawer -->
    <Teleport to="body">
      <Transition name="slide">
        <div v-if="selectedArticle" class="fixed inset-0 z-50 flex justify-end">
          <div class="absolute inset-0 bg-black/50" @click="selectedArticle = null" />
          <div class="relative w-full max-w-2xl bg-[#0f172a] border-l border-surface-border overflow-y-auto shadow-2xl">
            <div class="sticky top-0 bg-[#0f172a] border-b border-surface-border p-4 flex items-center justify-between">
              <h2 class="text-sm font-semibold text-white">文章详情</h2>
              <button class="text-slate-400 hover:text-white" @click="selectedArticle = null">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                </svg>
              </button>
            </div>
            <div class="p-6 space-y-4">
              <h3 class="text-base font-semibold text-white">{{ selectedArticle.title }}</h3>
              <div class="flex items-center gap-4 text-xs text-slate-400">
                <span>来源：{{ selectedArticle.source }}</span>
                <span>作者：{{ selectedArticle.author || '—' }}</span>
                <span>发布：{{ selectedArticle.published_at ? new Date(selectedArticle.published_at).toLocaleDateString('zh-CN') : '—' }}</span>
              </div>
              <div class="flex flex-wrap gap-1">
                <span v-for="tag in selectedArticle.tags" :key="tag"
                  class="px-2 py-0.5 bg-brand-600/10 text-brand-400 text-xs rounded border border-brand-600/20">{{ tag }}</span>
              </div>
              <p class="text-sm text-slate-300 leading-relaxed whitespace-pre-wrap">
                {{ selectedArticle.content || selectedArticle.summary || '暂无内容' }}
              </p>
              <a v-if="selectedArticle.url" :href="selectedArticle.url" target="_blank" rel="noopener"
                class="inline-flex items-center gap-1.5 text-brand-400 text-sm hover:underline">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/>
                </svg>
                查看原文
              </a>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<style scoped>
.slide-enter-from .relative { transform: translateX(100%); }
.slide-enter-to .relative { transform: translateX(0); }
.slide-leave-from .relative { transform: translateX(0); }
.slide-leave-to .relative { transform: translateX(100%); }
.slide-enter-active .relative,
.slide-leave-active .relative { transition: transform 0.3s cubic-bezier(0.16,1,0.3,1); }

/* Filter Chips */
.filter-chip {
  padding: 0.375rem 0.75rem;
  font-size: 0.75rem;
  font-weight: 500;
  border-radius: 9999px;
  border: 1px solid var(--border);
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s;
}

.filter-chip:hover {
  border-color: var(--primary);
  color: var(--primary);
}

.filter-chip.active {
  background: var(--primary);
  border-color: var(--primary);
  color: white;
}
</style>
