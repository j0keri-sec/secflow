import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Article } from '@/types'
import { articleApi, type ArticleListParams } from '@/api/article'

export const useArticleStore = defineStore('article', () => {
  // State
  const articles = ref<Article[]>([])
  const currentArticle = ref<Article | null>(null)
  const total = ref(0)
  const loading = ref(false)
  const error = ref<string | null>(null)
  
  // Cache
  const cache = new Map<string, { data: Article; timestamp: number }>()
  const CACHE_TTL = 5 * 60 * 1000 // 5 minutes

  // Getters
  const hasArticles = computed(() => articles.value.length > 0)
  
  const articlesBySource = computed(() => {
    const grouped: Record<string, Article[]> = {}
    for (const article of articles.value) {
      const source = article.source || '未知来源'
      if (!grouped[source]) grouped[source] = []
      grouped[source].push(article)
    }
    return grouped
  })

  const uniqueSources = computed(() => {
    const sources = new Set<string>()
    for (const article of articles.value) {
      if (article.source) sources.add(article.source)
    }
    return Array.from(sources).sort()
  })

  // Actions
  async function fetchArticles(params: ArticleListParams = {}) {
    loading.value = true
    error.value = null
    
    // Generate cache key
    const cacheKey = JSON.stringify(params)
    
    // Check cache first (only for first page without filters)
    if (!params.page || params.page === 1) {
      const cached = cache.get(cacheKey)
      if (cached && Date.now() - cached.timestamp < CACHE_TTL) {
        articles.value = [cached.data]
        total.value = 1
        loading.value = false
        return cached.data
      }
    }
    
    try {
      const res = await articleApi.list({
        page: params.page || 1,
        page_size: params.page_size || 20,
        keyword: params.keyword,
        source: params.source,
      })
      
      articles.value = res?.items || []
      total.value = res?.total || 0
      
      // Update cache
      if (!params.page || params.page === 1) {
        cache.set(cacheKey, {
          data: articles.value[0] || articles.value,
          timestamp: Date.now()
        })
      }
      
      return articles.value
    } catch (e: any) {
      error.value = e?.message || '获取文章失败'
      articles.value = []
      return []
    } finally {
      loading.value = false
    }
  }

  async function fetchArticle(id: string) {
    loading.value = true
    error.value = null
    
    // Check cache
    const cacheKey = `article:${id}`
    const cached = cache.get(cacheKey)
    if (cached && Date.now() - cached.timestamp < CACHE_TTL) {
      currentArticle.value = cached.data
      loading.value = false
      return cached.data
    }
    
    try {
      const res = await articleApi.get(id)
      currentArticle.value = res
      
      // Update cache
      cache.set(cacheKey, {
        data: res,
        timestamp: Date.now()
      })
      
      return res
    } catch (e: any) {
      error.value = e?.message || '获取文章详情失败'
      currentArticle.value = null
      return null
    } finally {
      loading.value = false
    }
  }

  async function deleteArticle(id: string) {
    try {
      await articleApi.delete(id)
      // Remove from list
      articles.value = articles.value.filter(a => a.id !== id)
      total.value = Math.max(0, total.value - 1)
      // Clear cache
      cache.delete(`article:${id}`)
      return true
    } catch (e: any) {
      error.value = e?.message || '删除文章失败'
      return false
    }
  }

  function clearCache() {
    cache.clear()
  }

  function clearError() {
    error.value = null
  }

  return {
    // State
    articles,
    currentArticle,
    total,
    loading,
    error,
    
    // Getters
    hasArticles,
    articlesBySource,
    uniqueSources,
    
    // Actions
    fetchArticles,
    fetchArticle,
    deleteArticle,
    clearCache,
    clearError,
  }
})
