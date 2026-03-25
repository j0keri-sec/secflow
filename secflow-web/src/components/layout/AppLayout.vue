<template>
  <div class="app-layout" :data-theme="theme">
    <!-- ===== 顶部主 Header ===== -->
    <header class="header">
      <div class="header-container">
        <!-- Logo + Brand -->
        <div class="brand" @click="navigateTo('/dashboard')">
          <div class="logo-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M12 2L2 7V12C2 18.6274 12 22 12 22C12 22 22 18.6274 22 12V7L12 2Z"
                stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" fill="none"/>
              <circle cx="12" cy="12" r="3" stroke="currentColor" stroke-width="1.5" fill="currentColor"/>
            </svg>
          </div>
          <h1 class="brand-name">SecFlow</h1>
          <span class="brand-tag">BETA</span>
        </div>

        <!-- 主菜单导航 -->
        <nav class="nav-menu">
          <button
            v-for="item in topMenuItems"
            :key="item.id"
            :class="['nav-item', { active: activeTopMenu === item.id }]"
            @click="navigateToMenu(item)"
          >
            <component :is="item.icon" class="nav-icon" />
            <span>{{ item.label }}</span>
            <span v-if="item.children?.length" class="nav-arrow">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none">
                <path d="M6 9L12 15L18 9" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </span>
          </button>
        </nav>

        <!-- 右侧区域 -->
        <div class="user-area">
          <!-- 通知铃 -->
          <button class="icon-btn" title="通知">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
              <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
              <path d="M13.73 21a2 2 0 0 1-3.46 0" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            </svg>
            <span class="notify-dot"></span>
          </button>

          <!-- 主题切换 -->
          <button class="icon-btn theme-toggle" @click="toggleTheme" :title="`切换到${theme === 'light' ? '深色' : '浅色'}主题`">
            <svg v-if="theme === 'light'" width="18" height="18" viewBox="0 0 24 24" fill="none">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
            <svg v-else width="18" height="18" viewBox="0 0 24 24" fill="none">
              <circle cx="12" cy="12" r="5" stroke="currentColor" stroke-width="2"/>
              <path d="M12 1V3M12 21V23M4.22 4.22L5.64 5.64M18.36 18.36L19.78 19.78M1 12H3M21 12H23M4.22 19.78L5.64 18.36M18.36 5.64L19.78 4.22" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            </svg>
          </button>

          <!-- 分隔线 -->
          <div class="divider-v"></div>

          <!-- 用户头像+名称 -->
          <div class="user-card">
            <div class="user-avatar">
              {{ currentUser.charAt(0).toUpperCase() }}
            </div>
            <span class="user-name">{{ currentUser }}</span>
          </div>

          <!-- 登出 -->
          <button class="icon-btn logout-btn" @click="handleLogout" title="登出">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
              <path d="M9 21H5C3.89543 21 3 20.1046 3 19V5C3 3.89543 3.89543 3 5 3H9M16 17L21 12M21 12L16 7M21 12H9" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
        </div>
      </div>
    </header>

    <!-- ===== 二级菜单栏（动态显示，居中）===== -->
    <div
      class="subnav-bar"
      :class="{ visible: currentSubMenu.length > 0 }"
    >
      <div class="subnav-container">
        <!-- 子菜单列表 - 居中显示 -->
        <nav class="subnav-center">
          <button
            v-for="sub in currentSubMenu"
            :key="sub.path"
            :class="['subnav-item', { active: route.path === sub.path || route.path.startsWith(sub.path + '/') }]"
            @click="navigateTo(sub.path)"
          >
            <component :is="sub.icon" class="subnav-icon" />
            <span>{{ sub.label }}</span>
            <span v-if="sub.badge" class="subnav-badge">{{ sub.badge }}</span>
          </button>
        </nav>

        <!-- 右侧路径面包屑 -->
        <div class="breadcrumb" v-if="currentPage">
          <span class="breadcrumb-parent">{{ activeMenuLabel }}</span>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" class="breadcrumb-sep">
            <path d="M9 18L15 12L9 6" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          </svg>
          <span class="breadcrumb-current">{{ currentPage }}</span>
        </div>
      </div>
    </div>

    <!-- ===== 主内容区 ===== -->
    <main class="main-content" :class="{ 'has-subnav': currentSubMenu.length > 0 }">
      <router-view v-slot="{ Component }">
        <transition name="page-fade" mode="out-in">
          <component :is="Component" :key="route.path" />
        </transition>
      </router-view>
    </main>

    <!-- ===== Footer ===== -->
    <footer class="footer">
      <div class="footer-inner">
        <div class="footer-left">
          <span class="footer-brand">SecFlow</span>
          <span class="footer-sep">·</span>
          <span>漏洞情报与事件管理平台</span>
        </div>
        <div class="footer-right">
          <span>实时安全态势</span>
          <span class="footer-sep">·</span>
          <span>智能漏洞分析</span>
          <span class="footer-sep">·</span>
          <span>© 2024 SecFlow</span>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, markRaw } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import logger from '@/utils/logger'

// Element Plus Icons
import {
  Odometer,
  Warning,
  Notification,
  Setting,
  User,
  Bell,
  Connection,
  Document,
  Cpu,
  DataLine,
  Timer,
} from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()

const currentUser = ref('Admin')
const theme = ref<'light' | 'dark'>('light')

// ── 菜单数据定义 ──────────────────────────────
interface SubMenuItem {
  label: string
  path: string
  icon?: any
  badge?: string | number
}

interface MenuItem {
  id: string
  label: string
  path: string
  icon?: any
  children?: SubMenuItem[]
}

const topMenuItems: MenuItem[] = [
  {
    id: 'dashboard',
    label: '首页',
    path: '/dashboard',
    icon: markRaw(Odometer),
  },
  {
    id: 'vulns',
    label: '漏洞情报',
    path: '/vulns',
    icon: markRaw(Warning),
  },
  {
    id: 'news',
    label: '安全简讯',
    path: '/news',
    icon: markRaw(Notification),
  },
  {
    id: 'articles',
    label: '文章列表',
    path: '/articles',
    icon: markRaw(Document),
  },
  {
    id: 'reports',
    label: '报表中心',
    path: '/reports',
    icon: markRaw(DataLine),
  },
  {
    id: 'ops',
    label: '运维中心',
    path: '/system/nodes',
    icon: markRaw(Setting),
    children: [
      { label: '节点管理', path: '/system/nodes', icon: markRaw(Cpu) },
      { label: '任务管理', path: '/system/tasks', icon: markRaw(Connection) },
      { label: '任务调度', path: '/system/settings', icon: markRaw(Timer) },
      { label: '推送管理', path: '/system/push', icon: markRaw(Bell) },
      { label: '用户管理', path: '/system/users', icon: markRaw(User) },
      { label: '系统日志', path: '/system/logs', icon: markRaw(Document) },
    ],
  },
]

// ── 计算当前激活菜单 ──────────────────────────
const activeTopMenu = computed(() => {
  for (const item of topMenuItems) {
    // 优先检查子菜单精确匹配
    if (item.children) {
      for (const child of item.children) {
        if (route.path === child.path || route.path.startsWith(child.path + '/')) {
          return item.id
        }
      }
    }
    // 再检查主菜单路径前缀匹配
    if (route.path === item.path || route.path.startsWith(item.path + '/')) {
      return item.id
    }
  }
  return 'dashboard'
})

const currentSubMenu = computed<SubMenuItem[]>(() => {
  const active = topMenuItems.find(i => i.id === activeTopMenu.value)
  return active?.children ?? []
})

const activeMenuLabel = computed(() => {
  return topMenuItems.find(i => i.id === activeTopMenu.value)?.label ?? ''
})

const currentPage = computed(() => {
  const subs = currentSubMenu.value
  const matched = subs.find(s => route.path === s.path || route.path.startsWith(s.path + '/'))
  return matched?.label ?? ''
})

// ── 导航函数 ──────────────────────────────────
function navigateToMenu(item: MenuItem) {
  // 有子菜单则默认进入第一个子菜单
  if (item.children?.length) {
    router.push(item.children[0].path)
  } else {
    router.push(item.path)
  }
}

function navigateTo(path: string) {
  router.push(path)
}

// ── 主题切换 ──────────────────────────────────
const toggleTheme = () => {
  theme.value = theme.value === 'light' ? 'dark' : 'light'
  document.documentElement.setAttribute('data-theme', theme.value)
  localStorage.setItem('theme', theme.value)
}

const handleLogout = () => {
  logger.info('User logged out')
}

onMounted(() => {
  const savedTheme = localStorage.getItem('theme') as 'light' | 'dark' | null
  if (savedTheme) {
    theme.value = savedTheme
    document.documentElement.setAttribute('data-theme', theme.value)
  }
})
</script>

<style scoped>
/* ═══════════════════════════════════════
   Layout
   CSS 变量在 main.css 中全局定义
═══════════════════════════════════════ */
.app-layout {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background: var(--bg-secondary);
  color: var(--text-primary);
  transition: background-color 0.35s ease, color 0.35s ease;
}

/* ═══════════════════════════════════════
   Header
═══════════════════════════════════════ */
.header {
  position: sticky;
  top: 0;
  z-index: 200;
  height: 60px;
  background: var(--bg-primary);
  border-bottom: 1px solid var(--border);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  box-shadow: var(--shadow-sm);
  transition: all 0.35s ease;
}

.header-container {
  max-width: 1440px;
  margin: 0 auto;
  padding: 0 1.75rem;
  height: 100%;
  display: flex;
  align-items: center;
  gap: 2rem;
}

/* ── Brand ─── */
.brand {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  flex-shrink: 0;
  cursor: pointer;
  transition: opacity 0.2s ease;
  user-select: none;
}
.brand:hover { opacity: 0.85; }

.logo-icon {
  width: 34px;
  height: 34px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--gradient-primary);
  color: white;
  box-shadow: 0 2px 8px rgba(59,130,246,0.3);
  flex-shrink: 0;
}

.brand-name {
  font-size: 1.2rem;
  font-weight: 800;
  letter-spacing: -0.5px;
  background: var(--gradient-primary);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  margin: 0;
  line-height: 1;
}

.brand-tag {
  font-size: 0.58rem;
  font-weight: 700;
  letter-spacing: 0.05em;
  color: var(--color-accent);
  border: 1.5px solid var(--color-accent);
  border-radius: 4px;
  padding: 1px 5px;
  line-height: 1.4;
  opacity: 0.7;
}

/* ── Nav Menu ─── */
.nav-menu {
  display: flex;
  align-items: center;
  gap: 2px;
  flex: 1;
  justify-content: center;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.45rem 1rem;
  background: transparent;
  border: none;
  border-radius: 8px;
  color: var(--text-secondary);
  font-size: 0.9rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  white-space: nowrap;
  position: relative;
}

.nav-icon {
  width: 15px;
  height: 15px;
  flex-shrink: 0;
  opacity: 0.8;
}

.nav-arrow {
  display: flex;
  align-items: center;
  opacity: 0.5;
  transition: transform 0.2s ease;
}

.nav-item:hover {
  color: var(--color-primary);
  background: rgba(59,130,246,0.07);
}

.nav-item:hover .nav-arrow {
  transform: rotate(180deg);
}

.nav-item.active {
  color: var(--color-primary);
  background: rgba(59,130,246,0.1);
  font-weight: 600;
}

.nav-item.active::before {
  content: '';
  position: absolute;
  bottom: -1px;
  left: 50%;
  transform: translateX(-50%);
  width: 24px;
  height: 2px;
  background: var(--gradient-primary);
  border-radius: 2px 2px 0 0;
}

/* ── User Area ─── */
.user-area {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-shrink: 0;
}

.icon-btn {
  width: 34px;
  height: 34px;
  border: none;
  background: transparent;
  border-radius: 8px;
  color: var(--text-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
  position: relative;
}

.icon-btn:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.theme-toggle:hover {
  background: rgba(59,130,246,0.1);
  color: var(--color-primary);
}

.logout-btn:hover {
  background: rgba(239,68,68,0.1);
  color: #ef4444;
}

.notify-dot {
  position: absolute;
  top: 6px;
  right: 6px;
  width: 7px;
  height: 7px;
  background: #ef4444;
  border-radius: 50%;
  border: 1.5px solid var(--bg-primary);
}

.divider-v {
  width: 1px;
  height: 20px;
  background: var(--border);
  margin: 0 0.25rem;
}

.user-card {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.3rem 0.6rem;
  border-radius: 8px;
  cursor: default;
  transition: background 0.2s ease;
}
.user-card:hover {
  background: var(--bg-tertiary);
}

.user-avatar {
  width: 28px;
  height: 28px;
  border-radius: 8px;
  background: var(--gradient-primary);
  color: white;
  font-size: 0.8rem;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.user-name {
  font-size: 0.875rem;
  color: var(--text-primary);
  font-weight: 500;
}

/* ═══════════════════════════════════════
   二级菜单栏
═══════════════════════════════════════ */
.subnav-bar {
  position: sticky;
  top: 60px;
  z-index: 190;
  background: var(--bg-primary);
  border-bottom: 1px solid var(--border-light);
  height: 0;
  overflow: hidden;
  transition: height 0.3s cubic-bezier(0.4, 0, 0.2, 1),
              opacity 0.2s ease,
              box-shadow 0.25s ease;
  opacity: 0;
}

.subnav-bar.visible {
  height: 44px;
  opacity: 1;
  box-shadow: 0 4px 12px rgba(0,0,0,0.06);
}

.subnav-container {
  max-width: 1440px;
  margin: 0 auto;
  padding: 0 1.75rem;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

/* 子菜单容器 - 居中 */
.subnav-center {
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 4px;
  height: 100%;
}

.subnav-item {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.4rem 1rem;
  background: transparent;
  border: none;
  border-radius: 8px;
  color: var(--text-secondary);
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  white-space: nowrap;
  height: 34px;
  position: relative;
}

.subnav-icon {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  opacity: 0.75;
}

.subnav-item:hover {
  color: var(--primary);
  background: var(--bg-tertiary);
}

.subnav-item.active {
  color: var(--primary);
  background: rgba(59, 130, 246, 0.12);
  font-weight: 600;
}

.subnav-item.active::after {
  content: '';
  position: absolute;
  bottom: -1px;
  left: 50%;
  transform: translateX(-50%);
  width: 22px;
  height: 2px;
  background: var(--gradient-primary);
  border-radius: 2px;
}

.subnav-badge {
  background: #ef4444;
  color: white;
  font-size: 0.65rem;
  font-weight: 700;
  padding: 1px 5px;
  border-radius: 10px;
  min-width: 16px;
  text-align: center;
  line-height: 1.4;
  margin-left: 2px;
}

/* 面包屑 */
.breadcrumb {
  display: flex;
  align-items: center;
  gap: 0.3rem;
  font-size: 0.78rem;
  color: var(--text-tertiary);
  flex-shrink: 0;
}

.breadcrumb-sep {
  opacity: 0.5;
  flex-shrink: 0;
}

.breadcrumb-parent {
  color: var(--text-tertiary);
}

.breadcrumb-current {
  color: var(--text-secondary);
  font-weight: 500;
}

/* ═══════════════════════════════════════
   Main Content
═══════════════════════════════════════ */
.main-content {
  flex: 1;
  max-width: 1440px;
  width: 100%;
  margin: 0 auto;
  padding: 1.75rem;
}

/* 页面过渡动画 */
.page-fade-enter-active,
.page-fade-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.page-fade-enter-from {
  opacity: 0;
  transform: translateY(6px);
}
.page-fade-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}

/* ═══════════════════════════════════════
   Footer
═══════════════════════════════════════ */
.footer {
  background: var(--bg-primary);
  border-top: 1px solid var(--border);
  padding: 1rem 1.75rem;
  margin-top: 1rem;
}

.footer-inner {
  max-width: 1440px;
  margin: 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 0.78rem;
  color: var(--text-tertiary);
}

.footer-left,
.footer-right {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.footer-brand {
  font-weight: 700;
  background: var(--gradient-primary);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.footer-sep {
  opacity: 0.4;
}

/* ═══════════════════════════════════════
   响应式
═══════════════════════════════════════ */
@media (max-width: 1100px) {
  .header-container {
    padding: 0 1rem;
  }
  .nav-item {
    padding: 0.4rem 0.7rem;
    font-size: 0.85rem;
  }
}

@media (max-width: 768px) {
  .header-container {
    gap: 0.75rem;
  }
  .brand-tag { display: none; }
  .user-name { display: none; }
  .breadcrumb { display: none; }

  .nav-item span:not(.nav-arrow) {
    display: none;
  }
  .nav-icon {
    width: 18px;
    height: 18px;
    opacity: 1;
  }
  .nav-item {
    padding: 0.5rem;
    width: 40px;
    justify-content: center;
  }
}
</style>
