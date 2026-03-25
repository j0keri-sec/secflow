<script setup lang="ts">
defineProps<{
  title?: string
  description?: string
  icon?: 'empty' | 'search' | 'error'
}>()
</script>

<template>
  <div class="empty-state">
    <!-- Icon -->
    <div class="empty-icon">
      <!-- Empty icon (default) -->
      <svg v-if="!icon || icon === 'empty'" class="w-16 h-16" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"/>
      </svg>
      
      <!-- Search icon -->
      <svg v-else-if="icon === 'search'" class="w-16 h-16" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
      </svg>
      
      <!-- Error icon -->
      <svg v-else-if="icon === 'error'" class="w-16 h-16" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
      </svg>
    </div>
    
    <!-- Title -->
    <h3 class="empty-title">
      {{ title || '暂无数据' }}
    </h3>
    
    <!-- Description -->
    <p v-if="description" class="empty-desc">
      {{ description }}
    </p>
    
    <!-- Slot for custom actions -->
    <slot />
  </div>
</template>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 3rem 1.5rem;
  text-align: center;
}

.empty-icon {
  color: var(--text-tertiary);
  opacity: 0.5;
  margin-bottom: 1rem;
}

.empty-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-secondary);
  margin: 0 0 0.5rem;
}

.empty-desc {
  font-size: 0.875rem;
  color: var(--text-tertiary);
  margin: 0;
  max-width: 20rem;
  line-height: 1.5;
}
</style>
