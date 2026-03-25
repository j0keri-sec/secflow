<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  current: number
  total: number
  pageSize?: number
}>()

const emit = defineEmits<{
  (e: 'update', page: number): void
  (e: 'change', page: number): void
}>()

const pageSize = computed(() => props.pageSize || 20)

const totalPages = computed(() => Math.ceil(props.total / pageSize.value))

const visiblePages = computed(() => {
  const pages: (number | string)[] = []
  const total = totalPages.value
  const current = props.current
  
  if (total <= 7) {
    // Show all pages
    for (let i = 1; i <= total; i++) pages.push(i)
  } else {
    // Always show first page
    pages.push(1)
    
    if (current > 3) {
      pages.push('...')
    }
    
    // Show pages around current
    const start = Math.max(2, current - 1)
    const end = Math.min(total - 1, current + 1)
    
    for (let i = start; i <= end; i++) {
      if (!pages.includes(i)) pages.push(i)
    }
    
    if (current < total - 2) {
      pages.push('...')
    }
    
    // Always show last page
    if (!pages.includes(total)) {
      pages.push(total)
    }
  }
  
  return pages
})

function goTo(page: number) {
  if (page < 1 || page > totalPages.value || page === props.current) return
  emit('update', page)
  emit('change', page)
}
</script>

<template>
  <div class="pagination" v-if="totalPages > 1">
    <!-- First/Prev -->
    <button
      class="page-btn"
      :disabled="current === 1"
      @click="goTo(1)"
    >
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 19l-7-7 7-7m8 14l-7-7 7-7"/>
      </svg>
    </button>
    <button
      class="page-btn"
      :disabled="current === 1"
      @click="goTo(current - 1)"
    >
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
      </svg>
    </button>
    
    <!-- Page numbers -->
    <template v-for="page in visiblePages" :key="page">
      <span v-if="page === '...'" class="page-ellipsis">...</span>
      <button
        v-else
        :class="['page-btn', { active: page === current }]"
        @click="goTo(page as number)"
      >
        {{ page }}
      </button>
    </template>
    
    <!-- Next/Last -->
    <button
      class="page-btn"
      :disabled="current === totalPages"
      @click="goTo(current + 1)"
    >
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
      </svg>
    </button>
    <button
      class="page-btn"
      :disabled="current === totalPages"
      @click="goTo(totalPages)"
    >
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 5l7 7-7 7M5 5l7 7-7 7"/>
      </svg>
    </button>
    
    <!-- Info -->
    <span class="page-info">
      第 {{ current }} / {{ totalPages }} 页，共 {{ total }} 条
    </span>
  </div>
</template>

<style scoped>
.pagination {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.page-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 2rem;
  height: 2rem;
  padding: 0 0.5rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--text-secondary);
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 0.375rem;
  cursor: pointer;
  transition: all 0.2s;
}

.page-btn:hover:not(:disabled) {
  color: var(--primary);
  border-color: var(--primary);
}

.page-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.page-btn.active {
  color: white;
  background: var(--primary);
  border-color: var(--primary);
}

.page-ellipsis {
  padding: 0 0.25rem;
  color: var(--text-tertiary);
}

.page-info {
  margin-left: 0.75rem;
  font-size: 0.75rem;
  color: var(--text-tertiary);
  white-space: nowrap;
}

@media (max-width: 640px) {
  .page-info {
    display: none;
  }
}
</style>
