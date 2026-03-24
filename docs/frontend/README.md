# SecFlow 前端开发文档

## 目录

- [项目结构](#项目结构)
- [技术栈](#技术栈)
- [代码规范](#代码规范)
- [组件开发](#组件开发)
- [API 调用](#api-调用)
- [状态管理](#状态管理)
- [路由配置](#路由配置)
- [样式规范](#样式规范)

---

## 项目结构

```
secflow-web/
├── public/
│   └── favicon.ico
│
├── src/
│   ├── api/                   # API 调用封装
│   │   ├── index.ts          # Axios 实例配置
│   │   ├── auth.ts           # 认证 API
│   │   ├── vuln.ts           # 漏洞 API
│   │   ├── article.ts        # 文章 API
│   │   ├── node.ts           # 节点 API
│   │   ├── task.ts           # 任务 API
│   │   ├── dashboard.ts      # 仪表盘 API
│   │   └── system.ts         # 系统 API
│   │
│   ├── assets/               # 静态资源
│   │   └── logo.svg
│   │
│   ├── components/           # 公共组件
│   │   ├── common/           # 通用组件
│   │   │   ├── Loading.vue
│   │   │   ├── Empty.vue
│   │   │   └── Error.vue
│   │   └── layout/           # 布局组件
│   │       ├── AppHeader.vue
│   │       ├── AppSidebar.vue
│   │       └── AppLayout.vue
│   │
│   ├── composables/           # 组合式函数
│   │   ├── useAuth.ts
│   │   ├── usePagination.ts
│   │   └── useWebSocket.ts
│   │
│   ├── pages/                # 页面组件
│   │   ├── Login.vue         # 登录页
│   │   ├── dashboard/        # 仪表盘
│   │   │   └── DashboardPage.vue
│   │   ├── vuln/             # 漏洞情报
│   │   │   ├── VulnListPage.vue
│   │   │   └── VulnDetailPage.vue
│   │   ├── article/          # 安全简讯
│   │   │   ├── ArticleListPage.vue
│   │   │   └── ArticleDetailPage.vue
│   │   ├── nodes/            # 节点管理
│   │   │   ├── NodeListPage.vue
│   │   │   └── NodeDetailPage.vue
│   │   ├── task/             # 任务管理
│   │   │   └── TaskListPage.vue
│   │   ├── system/           # 系统管理
│   │   │   ├── UserManagePage.vue
│   │   │   ├── PushChannelPage.vue
│   │   │   └── SettingsPage.vue
│   │   └── reports/          # 报告管理
│   │       └── ReportListPage.vue
│   │
│   ├── router/               # 路由配置
│   │   └── index.ts
│   │
│   ├── stores/               # Pinia 状态管理
│   │   ├── auth.ts
│   │   ├── app.ts
│   │   ├── vuln.ts
│   │   ├── node.ts
│   │   └── task.ts
│   │
│   ├── types/                # TypeScript 类型定义
│   │   └── index.ts
│   │
│   ├── utils/               # 工具函数
│   │   ├── http.ts           # Axios 封装
│   │   ├── format.ts        # 格式化工具
│   │   └── storage.ts       # 本地存储
│   │
│   ├── App.vue              # 根组件
│   └── main.ts              # 入口文件
│
├── index.html
├── package.json
├── tsconfig.json
├── vite.config.ts
└── tailwind.config.js
```

---

## 技术栈

| 技术 | 版本 | 说明 |
|------|------|------|
| Vue | 3.4+ | 渐进式 JavaScript 框架 |
| TypeScript | 5.x | 类型安全 |
| Vite | 5.x | 构建工具 |
| TailwindCSS | 3.x | 原子化 CSS |
| Pinia | 2.x | 状态管理 |
| Vue Router | 4.x | 路由管理 |
| Axios | 1.x | HTTP 客户端 |
| Headless UI | 2.x | 无样式组件库 |
| Heroicons | 2.x | 图标库 |

---

## 代码规范

### Vue 组件规范

```vue
<!-- 命名：使用 PascalCase -->
<!-- src/pages/vuln/VulnListPage.vue -->

<script setup lang="ts">
// 1. 导入
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { vulnApi } from '@/api/vuln'
import type { VulnRecord } from '@/types'

// 2. Props 和 Emits
interface Props {
  severity?: string
}
const props = withDefaults(defineProps<Props>(), {
  severity: 'all'
})

const emit = defineEmits<{
  (e: 'select', vuln: VulnRecord): void
}>()

// 3. 响应式状态
const loading = ref(false)
const vulns = ref<VulnRecord[]>([])
const pagination = ref({
  page: 1,
  pageSize: 20,
  total: 0
})

// 4. 计算属性
const hasMore = computed(() => vulns.value.length < pagination.value.total)

// 5. 方法
async function fetchVulns() {
  loading.value = true
  try {
    const res = await vulnApi.list({
      page: pagination.value.page,
      page_size: pagination.value.pageSize,
      severity: props.severity === 'all' ? undefined : props.severity
    })
    vulns.value = res.items
    pagination.value.total = res.data.total
  } finally {
    loading.value = false
  }
}

// 6. 生命周期
onMounted(() => {
  fetchVulns()
})
</script>

<template>
  <div class="vuln-list">
    <!-- ... -->
  </div>
</template>

<style scoped>
.vuln-list {
  /* 使用 Tailwind CSS */
}
</style>
```

### TypeScript 规范

```typescript
// 1. 类型定义优先于 any
interface VulnRecord {
  id: string
  title: string
  severity: SeverityLevel
  cve: string
  description: string
  source: string
  created_at: string
}

// 2. 使用 const 断言
const SEVERITY_OPTIONS = [
  { label: '全部', value: 'all' },
  { label: '严重', value: '严重' },
  { label: '高危', value: '高危' },
  { label: '中危', value: '中危' },
  { label: '低危', value: '低危' },
] as const

// 3. 泛型约束
async function fetchData<T extends object>(
  url: string,
  params: Record<string, string | number>
): Promise<PageData<T>> {
  // ...
}

// 4. 类型守卫
function isVulnRecord(obj: unknown): obj is VulnRecord {
  return typeof obj === 'object' && obj !== null && 'cve' in obj
}
```

---

## 组件开发

### 列表组件模板

```vue
<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import {
  DocumentArrowDownIcon,
  TrashIcon,
  EyeIcon
} from '@heroicons/vue/24/outline'

interface Props {
  items: VulnRecord[]
  loading?: boolean
  selectedIds?: string[]
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  selectedIds: () => []
})

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'export'): void
  (e: 'delete', id: string): void
}>()

const router = useRouter()

function handleView(item: VulnRecord) {
  router.push(`/vulns/${item.id}`)
}

function handleDelete(item: VulnRecord) {
  if (confirm(`确定删除漏洞 "${item.title}" 吗？`)) {
    emit('delete', item.id)
  }
}
</script>

<template>
  <div class="vuln-table">
    <!-- 工具栏 -->
    <div class="flex items-center justify-between mb-4">
      <div class="flex items-center gap-2">
        <slot name="toolbar" />
      </div>
      <div class="flex items-center gap-2">
        <button
          class="btn btn-primary"
          @click="emit('refresh')"
        >
          刷新
        </button>
        <button
          class="btn btn-secondary"
          @click="emit('export')"
        >
          <DocumentArrowDownIcon class="w-4 h-4 mr-1" />
          导出
        </button>
      </div>
    </div>

    <!-- 表格 -->
    <div class="overflow-x-auto">
      <table class="table">
        <thead>
          <tr>
            <th>漏洞名称</th>
            <th>CVE</th>
            <th>风险等级</th>
            <th>来源</th>
            <th>发现时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td colspan="6" class="text-center py-8">
              <div class="loading-spinner" />
            </td>
          </tr>
          <tr v-else-if="items.length === 0">
            <td colspan="6" class="text-center py-8 text-gray-500">
              暂无数据
            </td>
          </tr>
          <tr
            v-for="item in items"
            :key="item.id"
            class="hover:bg-gray-50"
          >
            <td>
              <router-link
                :to="`/vulns/${item.id}`"
                class="text-primary hover:underline"
              >
                {{ item.title }}
              </router-link>
            </td>
            <td>
              <code class="text-sm bg-gray-100 px-1 rounded">
                {{ item.cve || '-' }}
              </code>
            </td>
            <td>
              <span :class="getSeverityClass(item.severity)">
                {{ item.severity }}
              </span>
            </td>
            <td>{{ item.source }}</td>
            <td>{{ formatDate(item.created_at) }}</td>
            <td>
              <div class="flex items-center gap-2">
                <button
                  class="btn-icon"
                  title="查看"
                  @click="handleView(item)"
                >
                  <EyeIcon class="w-4 h-4" />
                </button>
                <button
                  class="btn-icon text-red-500"
                  title="删除"
                  @click="handleDelete(item)"
                >
                  <TrashIcon class="w-4 h-4" />
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 分页 -->
    <slot name="pagination" />
  </div>
</template>
```

---

## API 调用

### Axios 封装

```typescript
// src/utils/http.ts
import axios, { AxiosError, AxiosResponse } from 'axios'
import { ElMessage } from 'element-plus'
import router from '@/router'

const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器
http.interceptors.request.use(
  (config) => {
    // 添加 Token
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// 响应拦截器
http.interceptors.response.use(
  (response: AxiosResponse) => {
    const { code, msg, data } = response.data

    // 业务错误
    if (code !== 0 && code !== 200) {
      ElMessage.error(msg || '请求失败')
      return Promise.reject(new Error(msg))
    }

    return data
  },
  (error: AxiosError<{ code: number; msg: string }>) => {
    if (error.response) {
      const { status, data } = error.response

      switch (status) {
        case 401:
          // Token 过期，跳转登录
          ElMessage.error('登录已过期，请重新登录')
          localStorage.removeItem('token')
          router.push('/login')
          break
        case 403:
          ElMessage.error('没有权限')
          break
        case 404:
          ElMessage.error('资源不存在')
          break
        case 500:
          ElMessage.error('服务器错误')
          break
        default:
          ElMessage.error(data?.msg || '请求失败')
      }
    } else {
      ElMessage.error('网络错误')
    }

    return Promise.reject(error)
  }
)

export default http
```

### API 模块定义

```typescript
// src/api/vuln.ts
import http from '@/utils/http'
import type { VulnRecord, VulnStats } from '@/types'

export interface VulnListParams {
  page?: number
  page_size?: number
  severity?: string
  source?: string
  cve?: string
  keyword?: string
  pushed?: string | boolean
}

export interface PageData<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

export const vulnApi = {
  // 获取漏洞列表
  list: (params: VulnListParams = {}) => {
    const query = new URLSearchParams()
    Object.entries(params).forEach(([k, v]) => {
      if (v !== undefined && v !== '') {
        query.set(k, String(v))
      }
    })
    return http.get<PageData<VulnRecord>>(`/vulns?${query}`)
  },

  // 获取漏洞详情
  get: (id: string) => {
    return http.get<VulnRecord>(`/vulns/${id}`)
  },

  // 获取漏洞统计
  stats: () => {
    return http.get<VulnStats>('/vulns/stats')
  },

  // 删除漏洞
  delete: (id: string) => {
    return http.delete(`/vulns/${id}`)
  },

  // 导出漏洞
  exportCSV: (params: VulnListParams = {}) => {
    const query = new URLSearchParams()
    Object.entries(params).forEach(([k, v]) => {
      if (v !== undefined && v !== '') {
        query.set(k, String(v))
      }
    })
    query.set('format', 'csv')
    return http.getRaw(`/vulns/export?${query}`) as Promise<Blob>
  }
}
```

---

## 状态管理

### Store 定义

```typescript
// src/stores/vuln.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { vulnApi, type VulnListParams } from '@/api/vuln'
import type { VulnRecord, VulnStats } from '@/types'

export const useVulnStore = defineStore('vuln', () => {
  // 状态
  const items = ref<VulnRecord[]>([])
  const stats = ref<VulnStats | null>(null)
  const loading = ref(false)
  const pagination = ref({
    page: 1,
    pageSize: 20,
    total: 0
  })
  const filters = ref<VulnListParams>({})

  // 计算属性
  const hasMore = computed(() => items.value.length < pagination.value.total)
  const severityOptions = computed(() => {
    const sources = new Set(items.value.map(v => v.source))
    return Array.from(sources)
  })

  // 方法
  async function fetchList(params?: VulnListParams) {
    loading.value = true
    try {
      const queryParams = { ...filters.value, ...params }
      const res = await vulnApi.list(queryParams)
      items.value = res.items
      pagination.value.total = res.total
      if (params?.page) {
        pagination.value.page = params.page
      }
    } finally {
      loading.value = false
    }
  }

  async function fetchStats() {
    stats.value = await vulnApi.stats()
  }

  async function remove(id: string) {
    await vulnApi.delete(id)
    items.value = items.value.filter(v => v.id !== id)
    pagination.value.total--
  }

  function setFilters(newFilters: VulnListParams) {
    filters.value = newFilters
    pagination.value.page = 1
  }

  return {
    // 状态
    items,
    stats,
    loading,
    pagination,
    filters,
    // 计算属性
    hasMore,
    severityOptions,
    // 方法
    fetchList,
    fetchStats,
    remove,
    setFilters
  }
})
```

---

## 路由配置

```typescript
// src/router/index.ts
import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/pages/Login.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/',
    component: () => import('@/components/layout/AppLayout.vue'),
    redirect: '/dashboard',
    meta: { requiresAuth: true },
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/pages/dashboard/DashboardPage.vue'),
        meta: { title: '首页' }
      },
      {
        path: 'vulns',
        name: 'Vulns',
        component: () => import('@/pages/vuln/VulnListPage.vue'),
        meta: { title: '漏洞情报' }
      },
      {
        path: 'vulns/:id',
        name: 'VulnDetail',
        component: () => import('@/pages/vuln/VulnDetailPage.vue'),
        meta: { title: '漏洞详情', parent: 'Vulns' }
      },
      {
        path: 'articles',
        name: 'Articles',
        component: () => import('@/pages/article/ArticleListPage.vue'),
        meta: { title: '安全简讯' }
      },
      {
        path: 'nodes',
        name: 'Nodes',
        component: () => import('@/pages/nodes/NodeListPage.vue'),
        meta: { title: '节点管理' }
      },
      {
        path: 'system',
        name: 'System',
        redirect: '/system/users',
        meta: { title: '系统管理' },
        children: [
          {
            path: 'users',
            name: 'Users',
            component: () => import('@/pages/system/UserManagePage.vue'),
            meta: { title: '用户管理', icon: 'users' }
          },
          {
            path: 'push-channels',
            name: 'PushChannels',
            component: () => import('@/pages/system/PushChannelPage.vue'),
            meta: { title: '推送渠道', icon: 'bell' }
          }
        ]
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes
})

// 路由守卫
router.beforeEach((to, from, next) => {
  const authStore = useAuthStore()

  if (to.meta.requiresAuth !== false && !authStore.isLoggedIn) {
    next('/login')
  } else if (to.path === '/login' && authStore.isLoggedIn) {
    next('/dashboard')
  } else {
    next()
  }
})

export default router
```

---

## 样式规范

### Tailwind CSS 配置

```javascript
// tailwind.config.js
export default {
  content: [
    './index.html',
    './src/**/*.{vue,js,ts,jsx,tsx}'
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#eff6ff',
          100: '#dbeafe',
          200: '#bfdbfe',
          300: '#93c5fd',
          400: '#60a5fa',
          500: '#3b82f6',
          600: '#2563eb',
          700: '#1d4ed8',
          800: '#1e40af',
          900: '#1e3a8a'
        }
      },
      borderRadius: {
        DEFAULT: '8px',
        sm: '4px',
        md: '8px',
        lg: '12px',
        xl: '16px'
      }
    }
  },
  plugins: []
}
```

### 常用样式类

```vue
<template>
  <div class="p-4">
    <!-- 卡片 -->
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
      <!-- 标题 -->
      <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
        标题
      </h2>

      <!-- 按钮 -->
      <button class="btn btn-primary">主要按钮</button>
      <button class="btn btn-secondary">次要按钮</button>
      <button class="btn btn-danger">危险按钮</button>

      <!-- 输入框 -->
      <input class="form-input" type="text" placeholder="请输入..." />

      <!-- 表格 -->
      <table class="table">
        <thead>
          <tr>
            <th>列1</th>
            <th>列2</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>数据1</td>
            <td>数据2</td>
          </tr>
        </tbody>
      </table>

      <!-- 徽章 -->
      <span class="badge badge-success">成功</span>
      <span class="badge badge-warning">警告</span>
      <span class="badge badge-danger">危险</span>
    </div>
  </div>
</template>
```
