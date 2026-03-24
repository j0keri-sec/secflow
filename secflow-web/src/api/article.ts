import http from '@/utils/http'
import type { Article, PageData } from '@/types'

export interface ArticleListParams {
  page?: number
  page_size?: number
  keyword?: string
  source?: string
  pushed?: string
}

export const articleApi = {
  list: (params: ArticleListParams = {}) => {
    const q = new URLSearchParams()
    Object.entries(params).forEach(([k, v]) => {
      if (v !== undefined && v !== '') q.set(k, String(v))
    })
    return http.get<PageData<Article>>(`/articles?${q}`)
  },
  get: (id: string) => http.get<Article>(`/articles/${id}`),
  delete: (id: string) => http.del<null>(`/articles/${id}`),
}
