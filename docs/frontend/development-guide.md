# 前端开发规范

本文档定义 SecFlow 前端项目的开发规范和最佳实践。

## 技术栈

| 技术 | 版本 | 说明 |
|------|------|------|
| Vue | 3.4+ | 渐进式框架 |
| TypeScript | 5.x | 类型安全 |
| Vite | 5.x | 构建工具 |
| TailwindCSS | 3.x | CSS 框架 |
| Pinia | 2.x | 状态管理 |
| Vue Router | 4.x | 路由管理 |
| Axios | 1.x | HTTP 客户端 |

---

## 项目结构

```
secflow-web/
├── src/
│   ├── api/                    # API 调用封装
│   │   ├── index.ts           # API 入口
│   │   ├── auth.ts            # 认证 API
│   │   ├── vuln.ts            # 漏洞 API
│   │   ├── article.ts         # 文章 API
│   │   ├── task.ts            # 任务 API
│   │   └── node.ts            # 节点 API
│   │
│   ├── components/             # 公共组件
│   │   ├── common/            # 通用组件
│   │   │   ├── Button.vue
│   │   │   ├── Input.vue
│   │   │   ├── Modal.vue
│   │   │   └── Table.vue
│   │   ├── layout/            # 布局组件
│   │   │   ├── Header.vue
│   │   │   ├── Sidebar.vue
│   │   │   └── Footer.vue
│   │   └── charts/            # 图表组件
│   │
│   ├── pages/                  # 页面组件
│   │   ├── Login.vue
│   │   ├── Dashboard.vue
│   │   ├── VulnList.vue
│   │   ├── ArticleList.vue
│   │   ├── TaskList.vue
│   │   ├── NodeList.vue
│   │   └── Settings.vue
│   │
│   ├── stores/                 # Pinia Store
│   │   ├── auth.ts            # 认证状态
│   │   ├── vuln.ts            # 漏洞状态
│   │   ├── task.ts            # 任务状态
│   │   └── node.ts            # 节点状态
│   │
│   ├── router/                 # 路由配置
│   │   └── index.ts
│   │
│   ├── types/                  # TypeScript 类型
│   │   ├── api.ts             # API 类型
│   │   ├── model.ts           # 数据模型
│   │   └── index.ts
│   │
│   ├── utils/                  # 工具函数
│   │   ├── request.ts         # Axios 封装
│   │   ├── storage.ts         # 本地存储
│   │   ├── format.ts          # 格式化
│   │   └── index.ts
│   │
│   ├── composables/            # Composable 函数
│   │   ├── usePagination.ts
│   │   ├── useForm.ts
│   │   └── useWebSocket.ts
│   │
│   ├── App.vue
│   ├── main.ts
│   └── env.d.ts
│
├── public/                     # 静态资源
├── package.json
├── vite.config.ts
├── tailwind.config.js
├── tsconfig.json
└── .eslintrc.cjs
```

---

## 组件规范

### 组件命名

```typescript
// PascalCase 用于组件名
// kebab-case 用于文件名

// ✅ 好的命名
VulnCard.vue       // 组件名
vuln-card.ts        // 工具文件
useVulnList.ts      // Composable

// ❌ 避免的命名
vulnCard.vue
VulnCardComponent.vue
```

### 组件结构

```vue
<!-- VulnCard.vue -->
<template>
  <div class="vuln-card" :class="{ 'vuln-card--critical': isCritical }">
    <h3 class="vuln-card__title">{{ title }}</h3>
    <p class="vuln-card__description">{{ description }}</p>
    <div class="vuln-card__footer">
      <span class="vuln-card__severity" :class="severityClass">
        {{ severity }}
      </span>
      <span class="vuln-card__date">{{ formattedDate }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
// Props 定义
interface Props {
  id: string
  title: string
  description: string
  severity: '低危' | '中危' | '高危' | '严重'
  date: string
}

const props = withDefaults(defineProps<Props>(), {
  severity: '中危'
})

// Emits
const emit = defineEmits<{
  (e: 'click', id: string): void
}>()

// 计算属性
const isCritical = computed(() => props.severity === '严重')
const severityClass = computed(() => `severity--${props.severity}`)
const formattedDate = computed(() => formatDate(props.date))

// 方法
function handleClick() {
  emit('click', props.id)
}
</script>

<style scoped>
.vuln-card {
  padding: 1rem;
  border-radius: 8px;
  background: var(--bg-card);
}

.vuln-card__title {
  font-size: 1rem;
  font-weight: 600;
}

.vuln-card--critical {
  border-left: 4px solid var(--color-critical);
}
</style>
```

---

## TypeScript 规范

### 类型定义

```typescript
// types/model.ts

// 基础类型
export interface User {
  id: string
  username: string
  email: string
  role: 'admin' | 'editor' | 'viewer'
  avatar?: string
}

export interface VulnRecord {
  id: string
  key: string
  title: string
  description: string
  severity: SeverityLevel
  cve: string
  disclosure: string
  solutions: string
  references: string[]
  tags: string[]
  source: string
  url: string
  pushed: boolean
  reported_by: string
  created_at: string
  updated_at: string
}

export type SeverityLevel = '低危' | '中危' | '高危' | '严重'

// API 类型
export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

export interface PageResponse<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

// 请求类型
export interface VulnListParams {
  page?: number
  page_size?: number
  severity?: string
  source?: string
  cve?: string
  keyword?: string
  pushed?: boolean
}

export interface CreateTaskParams {
  sources: string[]
  page_limit?: number
  enable_github?: boolean
  proxy?: string
  priority?: number
}
```

### 使用类型

```typescript
// ✅ 好的写法
import type { VulnRecord, SeverityLevel } from '@/types'

function formatSeverity(level: SeverityLevel): string {
  const map: Record<SeverityLevel, string> = {
    '低危': 'Low',
    '中危': 'Medium',
    '高危': 'High',
    '严重': 'Critical'
  }
  return map[level]
}

// ❌ 避免 any
function handleData(data: any) { }

// ❌ 避免类型断言过度
const data = someFunction() as any
```

---

## API 调用规范

### 封装 Axios

```typescript
// utils/request.ts
import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios'
import type { ApiResponse } from '@/types'

const request: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  timeout: 30000,
})

// 请求拦截器
request.interceptors.request.use(
  (config) => {
    // 添加 Token
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
request.interceptors.response.use(
  (response: AxiosResponse<ApiResponse<any>>) => {
    const { code, message, data } = response.data

    if (code !== 0) {
      // 处理业务错误
      if (code === 401) {
        // Token 过期，跳转登录
        localStorage.removeItem('token')
        window.location.href = '/login'
      }
      return Promise.reject(new Error(message))
    }

    return response
  },
  (error) => {
    // 处理网络错误
    const message = error.response?.data?.message || error.message
    ElMessage.error(message)
    return Promise.reject(error)
  }
)

export default request
```

### API 模块化

```typescript
// api/vuln.ts
import request from '@/utils/request'
import type { ApiResponse, PageResponse, VulnRecord, VulnListParams } from '@/types'

export const vulnApi = {
  // 获取列表
  list(params: VulnListParams): Promise<ApiResponse<PageResponse<VulnRecord>>> {
    return request.get('/vulns', { params })
  },

  // 获取详情
  get(id: string): Promise<ApiResponse<VulnRecord>> {
    return request.get(`/vulns/${id}`)
  },

  // 删除
  delete(id: string): Promise<ApiResponse<null>> {
    return request.delete(`/vulns/${id}`)
  },

  // 获取统计
  stats(): Promise<ApiResponse<{
    total: number
    by_severity: Record<string, number>
  }>> {
    return request.get('/vulns/stats')
  },

  // 导出
  export(params: VulnListParams): Promise<Blob> {
    return request.get('/vulns/export', {
      params,
      responseType: 'blob'
    })
  }
}
```

---

## 状态管理规范

### Store 定义

```typescript
// stores/vuln.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { vulnApi } from '@/api'
import type { VulnRecord, VulnListParams } from '@/types'

export const useVulnStore = defineStore('vuln', () => {
  // State
  const list = ref<VulnRecord[]>([])
  const total = ref(0)
  const loading = ref(false)
  const params = ref<VulnListParams>({
    page: 1,
    page_size: 20
  })

  // Getters
  const hasMore = computed(() => list.value.length < total.value)
  const criticalVulns = computed(() =>
    list.value.filter(v => v.severity === '严重')
  )

  // Actions
  async function fetchList() {
    loading.value = true
    try {
      const res = await vulnApi.list(params.value)
      list.value = res.data.data.items
      total.value = res.data.data.total
    } finally {
      loading.value = false
    }
  }

  async function loadMore() {
    if (!hasMore.value || loading.value) return
    params.value.page++
    loading.value = true
    try {
      const res = await vulnApi.list(params.value)
      list.value.push(...res.data.data.items)
    } finally {
      loading.value = false
    }
  }

  async function remove(id: string) {
    await vulnApi.delete(id)
    list.value = list.value.filter(v => v.id !== id)
    total.value--
  }

  function reset() {
    list.value = []
    total.value = 0
    params.value.page = 1
  }

  return {
    // State
    list,
    total,
    loading,
    params,
    // Getters
    hasMore,
    criticalVulns,
    // Actions
    fetchList,
    loadMore,
    remove,
    reset
  }
})
```

---

## 样式规范

### Tailwind CSS

```vue
<template>
  <!-- 使用 Tailwind 类 -->
  <div class="p-4 bg-white rounded-lg shadow">
    <h2 class="text-xl font-bold text-gray-900">标题</h2>
    <p class="mt-2 text-gray-600">内容</p>
  </div>
</template>

<style scoped>
/* 自定义样式使用 CSS 变量 */
.custom-card {
  @apply p-4 rounded-lg shadow;
  background-color: var(--bg-card);
}
</style>
```

### 响应式设计

```vue
<template>
  <!-- 移动优先 -->
  <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
    <Card v-for="item in items" :key="item.id" :item="item" />
  </div>
</template>
```

---

## 路由规范

```typescript
// router/index.ts
import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
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
    component: () => import('@/components/layout/MainLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        name: 'Dashboard',
        component: () => import('@/pages/Dashboard.vue')
      },
      {
        path: 'vulns',
        name: 'VulnList',
        component: () => import('@/pages/VulnList.vue')
      },
      {
        path: 'articles',
        name: 'ArticleList',
        component: () => import('@/pages/ArticleList.vue')
      },
      {
        path: 'tasks',
        name: 'TaskList',
        component: () => import('@/pages/TaskList.vue')
      },
      {
        path: 'nodes',
        name: 'NodeList',
        component: () => import('@/pages/NodeList.vue')
      },
      {
        path: 'settings',
        name: 'Settings',
        component: () => import('@/pages/Settings.vue'),
        meta: { requiresAuth: true, roles: ['admin'] }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 导航守卫
router.beforeEach((to, from, next) => {
  const authStore = useAuthStore()

  if (to.meta.requiresAuth && !authStore.isLoggedIn) {
    next('/login')
    return
  }

  // 角色权限检查
  if (to.meta.roles && !to.meta.roles.includes(authStore.user?.role)) {
    next('/')
    return
  }

  next()
})

export default router
```

---

## 组件通信

### Props 和 Emits

```vue
<!-- Parent.vue -->
<template>
  <ChildComponent
    :title="title"
    :items="items"
    @select="handleSelect"
    @delete="handleDelete"
  />
</template>

<script setup lang="ts">
const title = ref('列表')
const items = ref(['a', 'b', 'c'])

function handleSelect(item: string) {
  console.log('Selected:', item)
}

function handleDelete(id: string) {
  items.value = items.value.filter(i => i !== id)
}
</script>
```

```vue
<!-- Child.vue -->
<script setup lang="ts">
interface Props {
  title: string
  items: string[]
}

const props = withDefaults(defineProps<Props>(), {
  items: () => []
})

const emit = defineEmits<{
  (e: 'select', item: string): void
  (e: 'delete', id: string): void
}>()
</script>
```

### Provide / Inject

```typescript
// Parent.vue
import { provide, ref } from 'vue'

const theme = ref('dark')
provide('theme', theme)

// Child.vue
import { inject } from 'vue'

const theme = inject('theme')
```

### Pinia Store

```typescript
// stores/counter.ts
import { defineStore } from 'pinia'

export const useCounterStore = defineStore('counter', () => {
  const count = ref(0)
  const doubleCount = computed(() => count.value * 2)

  function increment() {
    count.value++
  }

  return { count, doubleCount, increment }
})

// AnyComponent.vue
import { useCounterStore } from '@/stores/counter'

const counterStore = useCounterStore()
counterStore.increment()
```

---

## 常用 Composables

### 分页

```typescript
// composables/usePagination.ts
import { ref, computed } from 'vue'

export function usePagination<T>(api: (params: any) => Promise<{ data: { items: T[], total: number } }>) {
  const list = ref<T[]>([])
  const total = ref(0)
  const page = ref(1)
  const pageSize = ref(20)
  const loading = ref(false)

  const totalPages = computed(() => Math.ceil(total.value / pageSize.value))
  const hasMore = computed(() => page.value < totalPages.value)

  async function fetch() {
    loading.value = true
    try {
      const res = await api({ page: page.value, page_size: pageSize.value })
      list.value = res.data.items
      total.value = res.data.total
    } finally {
      loading.value = false
    }
  }

  async function loadMore() {
    if (!hasMore.value || loading.value) return
    page.value++
    loading.value = true
    try {
      const res = await api({ page: page.value, page_size: pageSize.value })
      list.value.push(...res.data.items)
    } finally {
      loading.value = false
    }
  }

  function reset() {
    page.value = 1
    list.value = []
    total.value = 0
  }

  return {
    list,
    total,
    page,
    pageSize,
    loading,
    totalPages,
    hasMore,
    fetch,
    loadMore,
    reset
  }
}
```

### 表单处理

```typescript
// composables/useForm.ts
import { ref, reactive } from 'vue'

interface FormRules {
  [key: string]: Array<{
    required?: boolean
    message: string
    trigger?: string | string[]
    validator?: (rule: any, value: any, callback: any) => void
  }>
}

export function useForm<T extends Record<string, any>>(initial: T, rules?: FormRules) {
  const form = reactive<T>({ ...initial })
  const errors = ref<Partial<Record<keyof T, string>>>({})
  const loading = ref(false)

  function reset() {
    Object.assign(form, initial)
    errors.value = {}
  }

  function setErrors(field: keyof T, message: string) {
    errors.value[field] = message
  }

  function clearErrors() {
    errors.value = {}
  }

  function validate(): boolean {
    if (!rules) return true
    // 实现验证逻辑...
    return true
  }

  return {
    form,
    errors,
    loading,
    reset,
    setErrors,
    clearErrors,
    validate
  }
}
```

---

## 测试规范

### 组件测试

```typescript
// components/__tests__/VulnCard.spec.ts
import { mount } from '@vue/test-utils'
import { describe, it, expect } from 'vitest'
import VulnCard from '../VulnCard.vue'

describe('VulnCard', () => {
  it('renders title correctly', () => {
    const wrapper = mount(VulnCard, {
      props: {
        id: '1',
        title: 'Test Vulnerability',
        description: 'Test description',
        severity: '高危',
        date: '2026-03-21'
      }
    })

    expect(wrapper.find('.vuln-card__title').text()).toBe('Test Vulnerability')
  })

  it('emits click event', async () => {
    const wrapper = mount(VulnCard, {
      props: {
        id: '1',
        title: 'Test',
        description: 'Test',
        severity: '中危',
        date: '2026-03-21'
      }
    })

    await wrapper.find('.vuln-card').trigger('click')
    expect(wrapper.emitted('click')).toBeTruthy()
  })
})
```

---

## 最佳实践

1. **组件拆分** - 保持组件小巧，一个组件做一件事
2. **类型安全** - 使用 TypeScript，避免 any
3. **组合优于继承** - 使用 Composable 和 Pinia
4. **懒加载** - 使用 `defineAsyncComponent` 懒加载路由组件
5. **错误处理** - 在 API 调用和用户交互中处理错误
6. **可访问性** - 添加适当的 ARIA 属性和键盘支持
7. **性能优化** - 使用 `v-memo`、`shallowRef` 避免不必要的重渲染
8. **代码分割** - 按路由分割代码，减少初始加载体积
