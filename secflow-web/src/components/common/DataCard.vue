<script setup lang="ts">
defineProps<{
  title?: string
  loading?: boolean
}>()
</script>

<template>
  <div class="card">
    <div v-if="title" class="text-xs font-semibold text-surface-muted uppercase tracking-wider mb-4">
      {{ title }}
    </div>
    <div v-if="loading" class="flex items-center justify-center py-10">
      <svg class="animate-spin w-6 h-6 text-brand-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
      </svg>
    </div>
    <slot v-else />
  </div>
</template>
