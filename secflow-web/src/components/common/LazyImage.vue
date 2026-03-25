<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

defineProps<{
  src: string
  alt?: string
  placeholder?: string
}>()

const isLoaded = ref(false)
const isError = ref(false)
const imgRef = ref<HTMLImageElement | null>(null)

// Intersection Observer for lazy loading
let observer: IntersectionObserver | null = null
const shouldLoad = ref(false)

function handleLoad() {
  isLoaded.value = true
}

function handleError() {
  isError.value = true
}

onMounted(() => {
  // Check if IntersectionObserver is supported
  if ('IntersectionObserver' in window) {
    observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            shouldLoad.value = true
            observer?.disconnect()
          }
        })
      },
      { rootMargin: '100px' }
    )
    
    if (imgRef.value) {
      observer.observe(imgRef.value)
    }
  } else {
    // Fallback: load immediately
    shouldLoad.value = true
  }
})

onUnmounted(() => {
  observer?.disconnect()
})
</script>

<template>
  <div ref="imgRef" class="lazy-image">
    <!-- Placeholder / Loading state -->
    <div v-if="!isLoaded && !isError" class="lazy-placeholder">
      <slot name="placeholder">
        <div class="placeholder-content">
          <svg class="placeholder-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/>
          </svg>
        </div>
      </slot>
    </div>
    
    <!-- Error state -->
    <div v-else-if="isError" class="lazy-error">
      <slot name="error">
        <div class="error-content">
          <svg class="error-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
          </svg>
          <span>图片加载失败</span>
        </div>
      </slot>
    </div>
    
    <!-- Actual image -->
    <img
      v-if="shouldLoad"
      :src="src"
      :alt="alt || ''"
      class="lazy-img"
      :class="{ loaded: isLoaded }"
      @load="handleLoad"
      @error="handleError"
    />
  </div>
</template>

<style scoped>
.lazy-image {
  position: relative;
  width: 100%;
  height: 100%;
  min-height: 100px;
}

.lazy-placeholder,
.lazy-error {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-secondary);
  border-radius: inherit;
}

.placeholder-content,
.error-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  color: var(--text-tertiary);
}

.placeholder-icon,
.error-icon {
  width: 2rem;
  height: 2rem;
  opacity: 0.5;
}

.error-content {
  color: var(--text-tertiary);
  font-size: 0.75rem;
}

.lazy-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  opacity: 0;
  transition: opacity 0.3s ease;
}

.lazy-img.loaded {
  opacity: 1;
}
</style>
