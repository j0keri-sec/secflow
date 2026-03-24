<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const isMobileMenuOpen = ref(false)

const navItems = [
  { name: '仪表盘',  key: 'dashboard', path: '/dashboard', icon: 'M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z' },
  { name: '漏洞信息', key: 'vulns',     path: '/vulns',     icon: 'M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z' },
  { name: '文章热点', key: 'articles',  path: '/articles',  icon: 'M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z' },
  { name: '系统管理', key: 'system',    path: '/system',    icon: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z' },
  { name: '报告管理', key: 'reports',   path: '/reports',   icon: 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z' },
]

const activeTop = computed(() => {
  const seg = route.path.split('/')[1]
  if (seg === 'system') return 'system'
  return navItems.find(n => route.path.startsWith(n.path))?.key ?? 'dashboard'
})

function closeMobileMenu() {
  isMobileMenuOpen.value = false
}
</script>

<template>
  <!-- Mobile Menu Button -->
  <button
    @click="isMobileMenuOpen = !isMobileMenuOpen"
    class="lg:hidden p-2 rounded-lg text-slate-400 hover:text-white hover:bg-surface/50 transition-all"
    aria-label="Toggle menu"
  >
    <svg v-if="!isMobileMenuOpen" class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
    </svg>
    <svg v-else class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
    </svg>
  </button>

  <!-- Mobile Menu Overlay -->
  <transition
    enter-active-class="transition-opacity duration-200"
    enter-from-class="opacity-0"
    enter-to-class="opacity-100"
    leave-active-class="transition-opacity duration-200"
    leave-from-class="opacity-100"
    leave-to-class="opacity-0"
  >
    <div
      v-if="isMobileMenuOpen"
      @click="closeMobileMenu"
      class="fixed inset-0 bg-black/60 backdrop-blur-sm z-40 lg:hidden"
    />
  </transition>

  <!-- Mobile Menu -->
  <transition
    enter-active-class="transition-transform duration-300 ease-out"
    enter-from-class="transform -translate-x-full"
    enter-to-class="transform translate-x-0"
    leave-active-class="transition-transform duration-200 ease-in"
    leave-from-class="transform translate-x-0"
    leave-to-class="transform -translate-x-full"
  >
    <div
      v-if="isMobileMenuOpen"
      class="fixed inset-y-0 left-0 w-72 bg-surface-card border-r border-surface-border z-50 lg:hidden flex flex-col"
    >
      <!-- Menu Header -->
      <div class="p-4 border-b border-surface-border">
        <div class="flex items-center gap-2">
          <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-brand-500 to-brand-700 flex items-center justify-center">
            <svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
            </svg>
          </div>
          <span class="text-lg font-semibold text-white">SecFlow</span>
        </div>
      </div>

      <!-- Menu Items -->
      <nav class="flex-1 overflow-y-auto p-4 space-y-2">
        <router-link
          v-for="item in navItems"
          :key="item.key"
          :to="item.path"
          @click="closeMobileMenu"
          class="flex items-center gap-3 px-4 py-3 rounded-xl transition-all duration-200 group"
          :class="activeTop === item.key
            ? 'bg-gradient-to-r from-brand-600 to-brand-500 text-white shadow-lg shadow-brand-500/25'
            : 'text-slate-400 hover:text-slate-100 hover:bg-surface/50'"
        >
          <svg class="w-5 h-5 flex-shrink-0" :class="activeTop === item.key ? 'text-white' : 'text-slate-400 group-hover:text-slate-200'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" :d="item.icon" />
          </svg>
          <span class="font-medium">{{ item.name }}</span>
          <!-- Active indicator -->
          <div v-if="activeTop === item.key" class="ml-auto w-2 h-2 rounded-full bg-white shadow-lg shadow-white/50"></div>
        </router-link>
      </nav>

      <!-- Menu Footer -->
      <div class="p-4 border-t border-surface-border">
        <div class="flex items-center gap-3 px-2">
          <div class="w-8 h-8 rounded-full bg-gradient-to-br from-brand-500 to-brand-700 flex items-center justify-center text-white text-sm font-semibold">
            {{ $slots.default ? 'U' : 'A' }}
          </div>
          <div class="flex-1 min-w-0">
            <p class="text-sm font-medium text-slate-200 truncate">用户名</p>
            <p class="text-xs text-slate-400">在线</p>
          </div>
        </div>
      </div>
    </div>
  </transition>
</template>
