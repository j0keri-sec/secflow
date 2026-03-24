<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { marked } from 'marked'
import type { Article } from '@/types'
import { articleApi } from '@/api/article'

const route = useRoute()
const router = useRouter()
const article = ref<Article | null>(null)
const loading = ref(true)
const error = ref('')

// 目录相关
const tableOfContents = ref<{ id: string; text: string; level: number }[]>([])
const activeHeading = ref('')
const showToc = ref(false)

// 配置 marked 选项
marked.setOptions({
  breaks: true,    // GFM line breaks
  gfm: true,       // GitHub Flavored Markdown
})

// 渲染 Markdown 内容
const renderedContent = computed(() => {
  if (!article.value?.content) return ''
  const html = marked(article.value.content) as string
  // 为 h2, h3 添加 id 用于锚点跳转
  return addHeadingIds(html)
})

// 为 heading 添加 id
function addHeadingIds(html: string): string {
  return html.replace(/<(h[23])[^>]*>([^<]*)<\/h[23]>/gi, (_match, tag, text) => {
    const id = text.trim().toLowerCase().replace(/\s+/g, '-').replace(/[^\u4e00-\u9fa5a-z0-9-]/g, '')
    return `<${tag} id="${id}">${text}</${tag}>`
  })
}

// 提取目录
function extractToc() {
  if (!article.value?.content) {
    tableOfContents.value = []
    return
  }

  const headings: { id: string; text: string; level: number }[] = []
  const headingRegex = /^(#{2,3})\s+(.+)$/gm
  let match

  while ((match = headingRegex.exec(article.value.content)) !== null) {
    const level = match[1].length
    const text = match[2].trim()
    const id = text.toLowerCase().replace(/\s+/g, '-').replace(/[^\u4e00-\u9fa5a-z0-9-]/g, '')
    headings.push({ id, text, level })
  }

  tableOfContents.value = headings
  showToc.value = headings.length >= 2 // 至少2个标题才显示目录
}

// 滚动监听
function handleScroll() {
  const headings = document.querySelectorAll('.article-markdown-content h2, .article-markdown-content h3')
  if (!headings.length) return

  let current = ''
  for (const heading of headings) {
    const rect = heading.getBoundingClientRect()
    if (rect.top <= 100) {
      current = heading.id
    }
  }
  activeHeading.value = current
}

// 滚动到指定标题
function scrollToHeading(id: string) {
  const element = document.getElementById(id)
  if (element) {
    element.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
}

// 格式化日期时间
function formatDateTime(dateStr?: string): string {
  if (!dateStr) return '—'
  return new Date(dateStr).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

onMounted(async () => {
  try {
    // 获取文章详情的 ID
    const id = route.params.id as string
    if (id) {
      // 如果有 ID，直接获取详情
      const res = await articleApi.get(id)
      article.value = res
      extractToc()
    }
  } catch (e: any) {
    error.value = e?.message ?? '加载失败'
  } finally {
    loading.value = false
  }

  window.addEventListener('scroll', handleScroll)
})

onUnmounted(() => {
  window.removeEventListener('scroll', handleScroll)
})

// 设置文章数据（从列表页传入）
function setArticle(a: Article) {
  article.value = a
  extractToc()
}

// 暴露方法给父组件调用
defineExpose({ setArticle })
</script>

<template>
  <div class="article-page">
    <div class="max-w-4xl mx-auto space-y-6">
      <!-- Back button -->
      <button
        class="flex items-center gap-2 text-sm text-slate-400 hover:text-slate-200 transition-colors group"
        @click="router.back()"
      >
        <svg class="w-4 h-4 transition-transform group-hover:-translate-x-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
        返回列表
      </button>

      <!-- Loading -->
      <div v-if="loading" class="card space-y-4 p-6">
        <div class="h-8 bg-surface rounded animate-pulse w-2/3" />
        <div class="h-4 bg-surface rounded animate-pulse w-1/3" />
        <div class="h-48 bg-surface rounded animate-pulse" />
        <div class="h-32 bg-surface rounded animate-pulse" />
      </div>

      <!-- Error -->
      <div v-else-if="error" class="card p-6 text-red-400">
        <div class="flex items-center gap-2">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
          </svg>
          {{ error }}
        </div>
      </div>

      <!-- Content -->
      <template v-else-if="article">
        <!-- Header Card -->
        <div class="card p-6 relative overflow-hidden">
          <div class="absolute inset-0 bg-gradient-to-br from-brand-600/5 to-transparent" />

          <div class="relative">
            <!-- Cover Image -->
            <div v-if="article.image" class="w-full h-56 -mx-6 -mt-6 mb-6 overflow-hidden rounded-t-lg">
              <img :src="article.image" :alt="article.title" class="w-full h-full object-cover" />
            </div>

            <!-- Category Tag -->
            <div class="flex items-center gap-3 mb-4">
              <span
                v-if="article.tags?.[0]"
                class="px-3 py-1 bg-brand-500/10 text-brand-400 text-xs font-medium rounded-full border border-brand-500/20"
              >
                {{ article.tags[0] }}
              </span>
              <span
                :class="[
                  'px-3 py-1 rounded-full text-xs font-medium',
                  article.pushed
                    ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                    : 'bg-slate-500/10 text-slate-400 border border-slate-500/20'
                ]"
              >
                {{ article.pushed ? '已推送' : '未推送' }}
              </span>
            </div>

            <!-- Title -->
            <h1 class="text-xl md:text-2xl font-bold mb-4 leading-snug" style="color: var(--text-primary)">
              {{ article.title }}
            </h1>

            <!-- Meta info -->
            <div class="grid grid-cols-2 md:grid-cols-3 gap-4 mb-5">
              <!-- Author -->
              <div class="space-y-1">
                <div class="text-xs text-slate-500 uppercase tracking-wide">作者</div>
                <div class="flex items-center gap-2">
                  <svg class="w-4 h-4 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
                  </svg>
                  <span class="text-sm text-slate-300">{{ article.author || '—' }}</span>
                </div>
              </div>

              <!-- Source -->
              <div class="space-y-1">
                <div class="text-xs text-slate-500 uppercase tracking-wide">来源</div>
                <div class="flex items-center gap-2">
                  <svg class="w-4 h-4 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"/>
                  </svg>
                  <span class="text-sm text-slate-300">{{ article.source || '未知' }}</span>
                </div>
              </div>

              <!-- Published Time -->
              <div class="space-y-1">
                <div class="text-xs text-slate-500 uppercase tracking-wide">发布时间</div>
                <div class="flex items-center gap-2">
                  <svg class="w-4 h-4 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/>
                  </svg>
                  <span class="text-sm text-slate-300">{{ formatDateTime(article.published_at) }}</span>
                </div>
              </div>
            </div>

            <!-- Action buttons -->
            <div class="flex flex-wrap gap-3 pt-5 border-t border-slate-700/50">
              <a
                v-if="article.url"
                :href="article.url"
                target="_blank"
                rel="noopener noreferrer"
                class="flex items-center gap-2 px-4 py-2 bg-brand-600/10 hover:bg-brand-600/20 text-brand-400 rounded-lg text-sm transition-colors border border-brand-600/20"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/>
                </svg>
                查看原文
              </a>
            </div>
          </div>
        </div>

        <!-- Tags -->
        <div v-if="article.tags?.length" class="flex flex-wrap gap-2">
          <span
            v-for="tag in article.tags"
            :key="tag"
            class="px-3 py-1 bg-brand-500/10 text-brand-400 text-xs rounded-full border border-brand-500/20 hover:bg-brand-500/20 transition-colors cursor-default"
          >
            {{ tag }}
          </span>
        </div>

        <!-- Summary Card -->
        <div v-if="article.summary" class="card p-6">
          <h2 class="text-sm font-semibold text-slate-300 mb-4 flex items-center gap-2">
            <svg class="w-4 h-4 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
            </svg>
            摘要
          </h2>
          <div class="bg-slate-800/50 rounded-lg p-4 border-l-4 border-brand-500">
            <p class="text-sm text-slate-300 leading-relaxed whitespace-pre-wrap">
              {{ article.summary }}
            </p>
          </div>
        </div>

        <!-- Content Card -->
        <div class="card p-6">
          <h2 class="text-sm font-semibold text-slate-300 mb-4 flex items-center gap-2">
            <svg class="w-4 h-4 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
            </svg>
            正文内容
          </h2>
          <div v-if="renderedContent" class="article-markdown-content" v-html="renderedContent"></div>
          <div v-else class="text-sm text-slate-500 italic text-center py-8">
            暂无正文内容
          </div>
        </div>

        <!-- Footer -->
        <div class="text-center text-xs text-slate-600 py-4">
          <span>发布时间: {{ formatDateTime(article.published_at) }}</span>
          <span v-if="article.source" class="mx-2">·</span>
          <span v-if="article.source">来源: {{ article.source }}</span>
        </div>
      </template>
    </div>

    <!-- Table of Contents (右侧浮动目录) -->
    <nav v-if="showToc" class="toc-sidebar">
      <div class="toc-header">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h7"/>
        </svg>
        <span>目录</span>
      </div>
      <ul class="toc-list">
        <li
          v-for="heading in tableOfContents"
          :key="heading.id"
          :class="['toc-item', `level-${heading.level}`, { active: activeHeading === heading.id }]"
        >
          <a :href="`#${heading.id}`" @click.prevent="scrollToHeading(heading.id)">
            {{ heading.text }}
          </a>
        </li>
      </ul>
    </nav>
  </div>
</template>

<style scoped>
/* Markdown content rendering styles */
.article-markdown-content {
  font-size: 0.875rem;
  line-height: 1.8;
  color: var(--text-primary);
}

.article-markdown-content :deep(h1),
.article-markdown-content :deep(h2),
.article-markdown-content :deep(h3),
.article-markdown-content :deep(h4),
.article-markdown-content :deep(h5),
.article-markdown-content :deep(h6) {
  font-weight: 600;
  color: var(--text-primary);
  margin-top: 1.5em;
  margin-bottom: 0.75em;
  line-height: 1.4;
}

.article-markdown-content :deep(h1) { font-size: 1.5rem; }
.article-markdown-content :deep(h2) { font-size: 1.25rem; }
.article-markdown-content :deep(h3) { font-size: 1.125rem; }
.article-markdown-content :deep(h4) { font-size: 1rem; }

.article-markdown-content :deep(p) {
  margin-bottom: 1em;
  line-height: 1.8;
}

.article-markdown-content :deep(ul),
.article-markdown-content :deep(ol) {
  margin: 1em 0;
  padding-left: 1.5em;
}

.article-markdown-content :deep(li) {
  margin-bottom: 0.5em;
  line-height: 1.7;
}

.article-markdown-content :deep(ul) {
  list-style-type: disc;
}

.article-markdown-content :deep(ol) {
  list-style-type: decimal;
}

.article-markdown-content :deep(a) {
  color: var(--color-primary);
  text-decoration: none;
}

.article-markdown-content :deep(a:hover) {
  text-decoration: underline;
}

.article-markdown-content :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 0.375rem;
  margin: 1em 0;
}

.article-markdown-content :deep(pre) {
  background-color: var(--bg-tertiary);
  border-radius: 0.375rem;
  padding: 1em;
  overflow-x: auto;
  margin: 1em 0;
}

.article-markdown-content :deep(code) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.875em;
  background-color: var(--bg-tertiary);
  padding: 0.125em 0.25em;
  border-radius: 0.25em;
  color: var(--text-primary);
}

.article-markdown-content :deep(pre code) {
  background-color: transparent;
  padding: 0;
  color: var(--text-primary);
}

.article-markdown-content :deep(blockquote) {
  border-left: 4px solid var(--color-primary);
  padding-left: 1em;
  margin: 1em 0;
  color: var(--text-secondary);
  font-style: italic;
}

.article-markdown-content :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 1em 0;
}

.article-markdown-content :deep(th),
.article-markdown-content :deep(td) {
  border: 1px solid var(--border);
  padding: 0.5em 0.75em;
  text-align: left;
  color: var(--text-primary);
}

.article-markdown-content :deep(th) {
  background-color: var(--bg-tertiary);
  font-weight: 600;
}

.article-markdown-content :deep(hr) {
  border: none;
  border-top: 1px solid var(--border);
  margin: 2em 0;
}

.article-markdown-content :deep(strong) {
  font-weight: 600;
  color: var(--text-primary);
}

.article-markdown-content :deep(em) {
  font-style: italic;
  color: var(--text-secondary);
}

/* ═══════════════════════════════════════
   Table of Contents (右侧浮动目录)
═══════════════════════════════════════ */
.article-page {
  position: relative;
}

.toc-sidebar {
  position: fixed;
  top: 50%;
  right: 1.5rem;
  transform: translateY(-50%);
  width: 220px;
  max-height: 60vh;
  background: var(--component-bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 1rem;
  box-shadow: var(--shadow-lg);
  z-index: 100;
  overflow-y: auto;
  transition: all 0.3s ease;
}

.toc-sidebar:hover {
  box-shadow: var(--shadow-lg), 0 0 0 1px var(--color-primary);
}

.toc-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 0.75rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--border);
}

.toc-header svg {
  color: var(--color-primary);
}

.toc-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.toc-item {
  margin-bottom: 0.25rem;
}

.toc-item.level-2 {
  padding-left: 0;
}

.toc-item.level-3 {
  padding-left: 0.75rem;
}

.toc-item a {
  display: block;
  font-size: 0.8rem;
  color: var(--text-secondary);
  text-decoration: none;
  padding: 0.3rem 0.5rem;
  border-radius: 4px;
  transition: all 0.2s ease;
  line-height: 1.4;
}

.toc-item a:hover {
  color: var(--color-primary);
  background: var(--bg-tertiary);
}

.toc-item.active a {
  color: var(--color-primary);
  background: rgba(59, 130, 246, 0.1);
  font-weight: 500;
}

/* 响应式 - 小屏幕隐藏目录 */
@media (max-width: 1280px) {
  .toc-sidebar {
    display: none;
  }
}
</style>
