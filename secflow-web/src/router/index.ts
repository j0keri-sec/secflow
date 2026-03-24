import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/pages/Login.vue'),
    meta: { public: true },
  },
  {
    path: '/',
    component: () => import('@/components/layout/AppLayout.vue'),
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/pages/dashboard/DashboardPage.vue'),
        meta: { title: '仪表盘' },
      },
      {
        path: 'news',
        name: 'News',
        component: () => import('@/pages/events/EventsPage.vue'),
        meta: { title: '安全简讯' },
      },
      // ── Vulnerability ──────────────────────────────────────────────
      {
        path: 'vulns',
        name: 'VulnList',
        component: () => import('@/pages/vuln/VulnListPage.vue'),
        meta: { title: '漏洞信息' },
      },
      {
        path: 'vulns/:id',
        name: 'VulnDetail',
        component: () => import('@/pages/vuln/VulnDetailPage.vue'),
        meta: { title: '漏洞详情' },
      },
      // ── Article ──────────────────────────────────────────────────
      {
        path: 'articles/:id',
        name: 'ArticleDetail',
        component: () => import('@/pages/article/ArticleDetailPage.vue'),
        meta: { title: '文章详情' },
      },
      // ── System Management ──────────────────────────────────────────
      {
        path: 'system/users',
        name: 'SystemUsers',
        component: () => import('@/pages/system/UsersPage.vue'),
        meta: { title: '用户管理', roles: ['admin'] },
      },
      {
        path: 'system/nodes',
        name: 'SystemNodes',
        component: () => import('@/pages/nodes/NodesPage.vue'),
        meta: { title: '节点管理' },
      },
      {
        path: 'system/tasks',
        name: 'SystemTasks',
        component: () => import('@/pages/nodes/TasksPage.vue'),
        meta: { title: '任务管理' },
      },
      {
        path: 'system/push',
        name: 'SystemPush',
        component: () => import('@/pages/system/PushPage.vue'),
        meta: { title: '推送管理' },
      },
      {
        path: 'system/logs',
        name: 'SystemLogs',
        component: () => import('@/pages/system/LogsPage.vue'),
        meta: { title: '日志管理' },
      },
      {
        path: 'system/settings',
        name: 'SystemSettings',
        component: () => import('@/pages/system/SettingsPage.vue'),
        meta: { title: '任务调度', roles: ['admin', 'editor'] },
      },
      // ── Reports ────────────────────────────────────────────────────
      {
        path: 'reports',
        name: 'Reports',
        component: () => import('@/pages/reports/ReportsPage.vue'),
        meta: { title: '报告管理' },
      },
    ],
  },
  { path: '/:pathMatch(.*)*', redirect: '/' },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// Navigation guard — redirect to /login when unauthenticated.
router.beforeEach(async (to, from) => {
  const auth = useAuthStore()

  console.log('路由守卫:', {
    from: from.path,
    to: to.path,
    isPublic: to.meta.public,
    isLoggedIn: auth.isLoggedIn,
    hasUser: !!auth.user,
  })

  // 公开路由允许直接访问
  if (to.meta.public) {
    console.log('→ 公开路由，允许访问')
    return true
  }

  // 检查是否已登录
  if (!auth.isLoggedIn) {
    console.log('→ 未登录，重定向到登录页')
    return { name: 'Login' }
  }

  // 如果已登录但没有用户信息，尝试获取
  if (!auth.user) {
    console.log('→ 已登录但没有用户信息，尝试获取...')
    try {
      await auth.fetchMe()
      console.log('→ 用户信息获取成功:', auth.user)
    } catch (e) {
      console.error('→ 获取用户信息失败:', e)
      auth.logout()
      return { name: 'Login' }
    }
  }

  // 角色权限检查
  const requiredRoles = to.meta.roles as string[] | undefined
  if (requiredRoles && auth.user) {
    if (!requiredRoles.includes(auth.user.role)) {
      console.log('→ 权限不足，重定向到 Dashboard')
      return { name: 'Dashboard' }
    }
  }

  console.log('→ 允许访问')
  return true
})

export default router
