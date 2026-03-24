<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { VulnRecord } from '@/types'
import { vulnApi } from '@/api/vuln'
import SeverityBadge from '@/components/common/SeverityBadge.vue'

const route = useRoute()
const router = useRouter()
const vuln = ref<VulnRecord | null>(null)
const loading = ref(true)
const error = ref('')
const copied = ref('')

// 获取CVSS评分（从描述中提取）
const cvssScore = computed(() => {
  if (!vuln.value?.description) return null
  const match = vuln.value.description.match(/CVSS[:\s]*([0-9.]+)/i)
  return match ? parseFloat(match[1]) : null
})

// CVSS样式
const cvssStyle = computed(() => {
  const score = cvssScore.value
  if (!score) return { bg: '#6b7280', text: '#fff' }
  if (score >= 9.0) return { bg: '#dc2626', text: '#fff' }
  if (score >= 7.0) return { bg: '#ea580c', text: '#fff' }
  if (score >= 4.0) return { bg: '#ca8a04', text: '#fff' }
  return { bg: '#16a34a', text: '#fff' }
})

// 复制到剪贴板
async function copyToClipboard(text: string, type: string) {
  try {
    await navigator.clipboard.writeText(text)
    copied.value = type
    setTimeout(() => { copied.value = '' }, 2000)
  } catch (e) {
    console.error('Copy failed:', e)
  }
}

// 格式化日期
function formatDate(dateStr?: string): string {
  if (!dateStr) return '—'
  return dateStr
}

onMounted(async () => {
  try {
    vuln.value = await vulnApi.get(route.params.id as string)
  } catch (e: any) {
    error.value = e?.message ?? '加载失败'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="max-w-5xl mx-auto space-y-5">
    <!-- Back button -->
    <button 
      class="flex items-center gap-2 text-sm text-slate-400 hover:text-slate-200 transition-colors group"
      @click="router.back()"
    >
      <svg class="w-4 h-4 transition-transform group-hover:-translate-x-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
      </svg>
      返回漏洞列表
    </button>

    <!-- Loading -->
    <div v-if="loading" class="card space-y-4 p-6">
      <div class="h-8 bg-surface rounded animate-pulse w-2/3" />
      <div class="h-4 bg-surface rounded animate-pulse w-1/3" />
      <div class="h-40 bg-surface rounded animate-pulse" />
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
    <template v-else-if="vuln">
      <!-- Header Card -->
      <div class="card p-6 relative overflow-hidden">
        <div class="absolute inset-0 bg-gradient-to-br from-brand-600/5 to-transparent" />
        
        <div class="relative">
          <!-- Top badges -->
          <div class="flex flex-wrap items-center gap-3 mb-4">
            <SeverityBadge :severity="vuln.severity" />
            <span 
              v-if="cvssScore" 
              class="px-2.5 py-1 rounded-md text-xs font-bold"
              :style="{ background: cvssStyle.bg, color: cvssStyle.text }"
            >
              CVSS {{ cvssScore.toFixed(1) }}
            </span>
            <span 
              :class="[
                'px-2.5 py-1 rounded-md text-xs font-medium',
                vuln.pushed 
                  ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20' 
                  : 'bg-slate-500/10 text-slate-400 border border-slate-500/20'
              ]"
            >
              {{ vuln.pushed ? '已推送' : '未推送' }}
            </span>
          </div>

          <!-- Title -->
          <h1 class="text-xl md:text-2xl font-bold text-white mb-4 leading-snug">
            {{ vuln.title }}
          </h1>

          <!-- Meta info grid -->
          <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
            <!-- CVE -->
            <div class="space-y-1">
              <div class="text-xs text-slate-500 uppercase tracking-wide">CVE编号</div>
              <div class="flex items-center gap-2">
                <span class="text-sm font-mono text-brand-400">{{ vuln.cve || '—' }}</span>
                <button 
                  v-if="vuln.cve"
                  @click="copyToClipboard(vuln.cve!, 'cve')"
                  class="p-1 text-slate-500 hover:text-brand-400 transition-colors"
                  :title="copied === 'cve' ? '已复制' : '复制'"
                >
                  <svg v-if="copied !== 'cve'" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                  </svg>
                  <svg v-else class="w-4 h-4 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
                  </svg>
                </button>
              </div>
            </div>

            <!-- Source -->
            <div class="space-y-1">
              <div class="text-xs text-slate-500 uppercase tracking-wide">数据来源</div>
              <div class="text-sm text-slate-300">{{ vuln.source || '未知' }}</div>
            </div>

            <!-- Disclosure -->
            <div class="space-y-1">
              <div class="text-xs text-slate-500 uppercase tracking-wide">披露时间</div>
              <div class="text-sm text-slate-300">{{ formatDate(vuln.disclosure) }}</div>
            </div>

            <!-- Created -->
            <div class="space-y-1">
              <div class="text-xs text-slate-500 uppercase tracking-wide">收录时间</div>
              <div class="text-sm text-slate-300">
                {{ vuln.created_at ? new Date(vuln.created_at).toLocaleDateString('zh-CN') : '—' }}
              </div>
            </div>
          </div>

          <!-- Action buttons -->
          <div class="flex flex-wrap gap-3 mt-5 pt-5 border-t border-slate-700/50">
            <a 
              v-if="vuln.cve"
              :href="`https://nvd.nist.gov/vuln/detail/${vuln.cve}`"
              target="_blank"
              rel="noopener noreferrer"
              class="flex items-center gap-2 px-4 py-2 bg-brand-600/10 hover:bg-brand-600/20 text-brand-400 rounded-lg text-sm transition-colors border border-brand-600/20"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/>
              </svg>
              NVD详情
            </a>
            
            <a 
              v-if="vuln.url || vuln.from"
              :href="vuln.url || vuln.from"
              target="_blank"
              rel="noopener noreferrer"
              class="flex items-center gap-2 px-4 py-2 bg-slate-600/10 hover:bg-slate-600/20 text-slate-300 rounded-lg text-sm transition-colors border border-slate-600/20"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3.055 11H5a2 2 0 012 2v1a2 2 0 002 2 2 2 0 012 2v2.945M8 3.935V5.5A2.5 2.5 0 0010.5 8h.5a2 2 0 012 2 2 2 0 104 0 2 2 0 012-2h1.064M15 20.488V18a2 2 0 012-2h3.064"/>
              </svg>
              来源链接
            </a>
          </div>
        </div>
      </div>

      <!-- Tags -->
      <div v-if="vuln.tags?.length" class="flex flex-wrap gap-2">
        <span 
          v-for="tag in vuln.tags" 
          :key="tag"
          class="px-3 py-1 bg-brand-500/10 text-brand-400 text-xs rounded-full border border-brand-500/20 hover:bg-brand-500/20 transition-colors cursor-default"
        >
          {{ tag }}
        </span>
      </div>

      <!-- Description Card -->
      <div class="card p-6">
        <h2 class="text-sm font-semibold text-slate-300 mb-4 flex items-center gap-2">
          <svg class="w-4 h-4 text-brand-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
          </svg>
          漏洞描述
        </h2>
        <p class="text-sm text-slate-300 leading-relaxed whitespace-pre-wrap">
          {{ vuln.description || '暂无详细描述' }}
        </p>
      </div>

      <!-- Affected Versions Card -->
      <div class="card p-6">
        <h2 class="text-sm font-semibold text-slate-300 mb-4 flex items-center gap-2">
          <svg class="w-4 h-4 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/>
          </svg>
          受影响版本
        </h2>
        <div class="bg-slate-800/50 rounded-lg p-4 text-sm text-slate-400">
          <div v-if="vuln.tags?.length" class="space-y-2">
            <div class="flex items-center gap-2 text-slate-300">
              <span class="w-2 h-2 rounded-full bg-amber-400"></span>
              根据漏洞标签，相关组件可能受影响
            </div>
            <div class="flex flex-wrap gap-2 mt-2">
              <span 
                v-for="tag in vuln.tags" 
                :key="tag"
                class="px-2 py-1 bg-slate-700/50 text-slate-300 text-xs rounded border border-slate-600/50"
              >
                {{ tag }}
              </span>
            </div>
          </div>
          <div v-else>
            暂无受影响版本信息
          </div>
        </div>
      </div>

      <!-- Solutions Card -->
      <div class="card p-6">
        <h2 class="text-sm font-semibold text-slate-300 mb-4 flex items-center gap-2">
          <svg class="w-4 h-4 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"/>
          </svg>
          修复建议
        </h2>
        <div v-if="vuln.solutions" class="text-sm text-slate-300 leading-relaxed whitespace-pre-wrap">
          {{ vuln.solutions }}
        </div>
        <div v-else class="text-sm text-slate-500 italic">
          暂无修复建议，请关注官方安全公告获取最新修复方案
        </div>
      </div>

      <!-- Bottom Grid -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-5">
        <!-- References -->
        <div class="card p-5">
          <h2 class="text-sm font-semibold text-slate-300 mb-4 flex items-center gap-2">
            <svg class="w-4 h-4 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"/>
            </svg>
            参考链接
            <span class="text-xs text-slate-500 font-normal ml-auto">{{ vuln.references?.length || 0 }}条</span>
          </h2>
          <div v-if="vuln.references?.length" class="max-h-64 overflow-y-auto space-y-2">
            <a 
              v-for="(ref, index) in vuln.references" 
              :key="index"
              :href="ref" 
              target="_blank" 
              rel="noopener noreferrer"
              class="flex items-start gap-2 text-xs text-brand-400 hover:text-brand-300 hover:underline break-all line-clamp-2 p-2 rounded-md hover:bg-brand-500/5 transition-colors"
            >
              <svg class="w-3 h-3 mt-0.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/>
              </svg>
              {{ ref }}
            </a>
          </div>
          <div v-else class="text-sm text-slate-500 italic py-4 text-center">
            暂无参考链接
          </div>
        </div>

        <!-- GitHub POC -->
        <div class="card p-5">
          <h2 class="text-sm font-semibold text-slate-300 mb-4 flex items-center gap-2">
            <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
              <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
            </svg>
            GitHub POC / EXP
            <span class="text-xs text-slate-500 font-normal ml-auto">{{ vuln.github_search?.length || 0 }}条</span>
          </h2>
          <div v-if="vuln.github_search?.length" class="max-h-64 overflow-y-auto space-y-2">
            <a 
              v-for="(poc, index) in vuln.github_search" 
              :key="index"
              :href="poc" 
              target="_blank" 
              rel="noopener noreferrer"
              class="flex items-start gap-2 text-xs text-emerald-400 hover:text-emerald-300 hover:underline break-all line-clamp-2 p-2 rounded-md hover:bg-emerald-500/5 transition-colors"
            >
              <svg class="w-3 h-3 mt-0.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/>
              </svg>
              {{ poc }}
            </a>
          </div>
          <div v-else class="text-sm text-slate-500 italic py-4 text-center">
            暂无相关POC/EXP
          </div>
        </div>
      </div>

      <!-- Footer -->
      <div class="text-center text-xs text-slate-600 py-4">
        <span>报告者节点: {{ vuln.reported_by || '系统' }}</span>
        <span class="mx-2">·</span>
        <span>更新时间: {{ vuln.updated_at ? new Date(vuln.updated_at).toLocaleString('zh-CN') : '—' }}</span>
      </div>
    </template>
  </div>
</template>

<style scoped>
.max-h-64::-webkit-scrollbar {
  width: 4px;
}

.max-h-64::-webkit-scrollbar-track {
  background: transparent;
}

.max-h-64::-webkit-scrollbar-thumb {
  background: rgba(100, 116, 139, 0.3);
  border-radius: 2px;
}

.max-h-64::-webkit-scrollbar-thumb:hover {
  background: rgba(100, 116, 139, 0.5);
}
</style>
