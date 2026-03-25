<script setup lang="ts">
defineProps<{
  rows?: number
  height?: string
}>()
</script>

<template>
  <div class="skeleton-list" :style="{ '--skeleton-height': height || '1rem' }">
    <div v-for="i in (rows || 5)" :key="i" class="skeleton-item">
      <div class="skeleton-line w-3/4"></div>
      <div class="skeleton-line w-full"></div>
      <div class="skeleton-line w-1/2"></div>
    </div>
  </div>
</template>

<style scoped>
.skeleton-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.skeleton-item {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 1rem;
  background: var(--bg-secondary);
  border-radius: 0.5rem;
}

.skeleton-line {
  height: var(--skeleton-height);
  background: linear-gradient(
    90deg,
    var(--bg-tertiary) 25%,
    var(--bg-secondary) 50%,
    var(--bg-tertiary) 75%
  );
  background-size: 200% 100%;
  border-radius: 0.25rem;
  animation: shimmer 1.5s infinite;
}

@keyframes shimmer {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}
</style>
