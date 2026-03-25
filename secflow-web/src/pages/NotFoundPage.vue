<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<template>
  <div class="not-found-page">
    <div class="not-found-content">
      <!-- Animated 404 -->
      <div class="error-code">
        <span class="code-4">4</span>
        <span class="code-0">0</span>
        <span class="code-4">4</span>
      </div>
      
      <!-- Error message -->
      <h1 class="error-title">页面不存在</h1>
      <p class="error-desc">抱歉，您访问的页面已不存在或已被移除</p>
      
      <!-- Actions -->
      <div class="error-actions">
        <button class="btn-primary" @click="router.push('/')">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M13 5l6 6H8"/>
          </svg>
          返回首页
        </button>
        <button class="btn-secondary" @click="router.back()">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
          </svg>
          返回上页
        </button>
      </div>
      
      <!-- Decorative elements -->
      <div class="decoration">
        <div class="deco-circle circle-1"></div>
        <div class="deco-circle circle-2"></div>
        <div class="deco-line line-1"></div>
        <div class="deco-line line-2"></div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.not-found-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
  background: var(--bg-primary);
}

.not-found-content {
  text-align: center;
  position: relative;
  max-width: 500px;
}

.error-code {
  display: flex;
  justify-content: center;
  margin-bottom: 1.5rem;
}

.error-code span {
  font-size: 8rem;
  font-weight: 800;
  line-height: 1;
  letter-spacing: -0.05em;
  animation: float 3s ease-in-out infinite;
}

.code-4 {
  color: var(--primary);
  animation-delay: 0s;
}

.code-0 {
  color: var(--accent);
  animation-delay: 0.2s;
  animation-duration: 2.5s;
}

.code-4:last-child {
  animation-delay: 0.4s;
}

@keyframes float {
  0%, 100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-10px);
  }
}

.error-title {
  font-size: 1.75rem;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 0.75rem;
}

.error-desc {
  font-size: 1rem;
  color: var(--text-secondary);
  margin: 0 0 2rem;
  line-height: 1.6;
}

.error-actions {
  display: flex;
  gap: 1rem;
  justify-content: center;
  flex-wrap: wrap;
}

.error-actions button {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
}

/* Decorations */
.decoration {
  position: absolute;
  inset: 0;
  pointer-events: none;
  overflow: hidden;
  z-index: -1;
}

.deco-circle {
  position: absolute;
  border-radius: 50%;
  border: 1px solid var(--border);
  opacity: 0.3;
}

.circle-1 {
  width: 200px;
  height: 200px;
  top: -50px;
  right: -50px;
  animation: pulse-slow 4s ease-in-out infinite;
}

.circle-2 {
  width: 150px;
  height: 150px;
  bottom: -30px;
  left: -30px;
  animation: pulse-slow 4s ease-in-out infinite 1s;
}

@keyframes pulse-slow {
  0%, 100% {
    opacity: 0.3;
    transform: scale(1);
  }
  50% {
    opacity: 0.1;
    transform: scale(1.05);
  }
}

.deco-line {
  position: absolute;
  height: 1px;
  background: linear-gradient(90deg, transparent, var(--primary), transparent);
  opacity: 0.2;
}

.line-1 {
  width: 100px;
  top: 30%;
  left: -30px;
  transform: rotate(-45deg);
}

.line-2 {
  width: 80px;
  bottom: 20%;
  right: -20px;
  transform: rotate(-45deg);
}

@media (max-width: 640px) {
  .error-code span {
    font-size: 5rem;
  }
  
  .error-title {
    font-size: 1.5rem;
  }
  
  .error-desc {
    font-size: 0.9rem;
  }
}
</style>
