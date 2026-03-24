<script setup lang="ts">
import { ref, reactive, computed, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { authApi } from '@/api/auth'

const router = useRouter()
const auth = useAuthStore()

type Mode = 'login' | 'register'
const mode = ref<Mode>('login')

const form = reactive({ username: '', password: '', email: '', invite_code: '' })
const error = ref('')
const loading = ref(false)
const showPassword = ref(false)

const isSuccess = computed(() => error.value.includes('成功'))

async function submit() {
  error.value = ''
  loading.value = true
  try {
    console.log('开始登录...', { username: form.username, password: '***' })

    if (mode.value === 'login') {
      const result = await auth.login({ username: form.username, password: form.password })
      console.log('登录成功，准备跳转到 dashboard...', result)

      // 使用 nextTick 确保 auth store 已完全更新
      await nextTick()

      console.log('开始路由跳转...', {
        isLoggedIn: auth.isLoggedIn,
        user: auth.user,
      })

      await router.push('/dashboard')
      console.log('路由跳转完成')
    } else {
      await authApi.register({
        username: form.username,
        password: form.password,
        email: form.email,
        invite_code: form.invite_code,
      })
      mode.value = 'login'
      error.value = '注册成功，请登录'
    }
  } catch (e: any) {
    console.error('登录失败:', e)
    error.value = e?.response?.data?.message ?? e?.message ?? '操作失败'
  } finally {
    loading.value = false
  }
}

function switchMode(m: Mode) {
  error.value = ''
  mode.value = m
}
</script>

<template>
  <div class="login-page">
    <!-- Animated background orbs -->
    <div class="orb orb-1" />
    <div class="orb orb-2" />
    <div class="orb orb-3" />

    <!-- Grid overlay -->
    <div class="grid-overlay" />

    <!-- Floating particles -->
    <div class="particle p-1" />
    <div class="particle p-2" />
    <div class="particle p-3" />
    <div class="particle p-4" />
    <div class="particle p-5" />

    <!-- Main card -->
    <div class="login-card-wrapper">
      <!-- Logo section -->
      <div class="logo-section">
        <div class="logo-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" class="icon-shield">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
              d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
          </svg>
        </div>
        <div class="logo-glow" />
        <h1 class="logo-title">SecFlow</h1>
        <p class="logo-subtitle">安全情报实时监控平台</p>
      </div>

      <!-- Glass card -->
      <div class="glass-card">
        <!-- Mode switcher -->
        <div class="mode-tabs">
          <button
            class="mode-tab"
            :class="{ active: mode === 'login' }"
            @click="switchMode('login')"
          >
            <svg viewBox="0 0 20 20" fill="currentColor" class="tab-icon">
              <path fill-rule="evenodd" d="M3 3a1 1 0 011 1v12a1 1 0 11-2 0V4a1 1 0 011-1zm7.707 3.293a1 1 0 010 1.414L9.414 9H17a1 1 0 110 2H9.414l1.293 1.293a1 1 0 01-1.414 1.414l-3-3a1 1 0 010-1.414l3-3a1 1 0 011.414 0z" clip-rule="evenodd"/>
            </svg>
            登录
          </button>
          <button
            class="mode-tab"
            :class="{ active: mode === 'register' }"
            @click="switchMode('register')"
          >
            <svg viewBox="0 0 20 20" fill="currentColor" class="tab-icon">
              <path d="M8 9a3 3 0 100-6 3 3 0 000 6zM8 11a6 6 0 016 6H2a6 6 0 016-6zM16 7a1 1 0 10-2 0v1h-1a1 1 0 100 2h1v1a1 1 0 102 0v-1h1a1 1 0 100-2h-1V7z"/>
            </svg>
            注册
          </button>
          <div class="tab-indicator" :class="{ 'tab-right': mode === 'register' }" />
        </div>

        <!-- Form -->
        <form @submit.prevent="submit" class="login-form">
          <!-- Username -->
          <div class="field-group">
            <label class="field-label">
              <svg viewBox="0 0 20 20" fill="currentColor" class="label-icon">
                <path fill-rule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clip-rule="evenodd"/>
              </svg>
              用户名
            </label>
            <div class="input-wrapper">
              <input
                v-model="form.username"
                type="text"
                class="fancy-input"
                placeholder="输入用户名"
                required
                autocomplete="username"
              />
              <div class="input-highlight" />
            </div>
          </div>

          <!-- Email (register only) -->
          <Transition name="field-slide">
            <div v-if="mode === 'register'" class="field-group">
              <label class="field-label">
                <svg viewBox="0 0 20 20" fill="currentColor" class="label-icon">
                  <path d="M2.003 5.884L10 9.882l7.997-3.998A2 2 0 0016 4H4a2 2 0 00-1.997 1.884z"/>
                  <path d="M18 8.118l-8 4-8-4V14a2 2 0 002 2h12a2 2 0 002-2V8.118z"/>
                </svg>
                邮箱地址
              </label>
              <div class="input-wrapper">
                <input
                  v-model="form.email"
                  type="email"
                  class="fancy-input"
                  placeholder="输入邮箱地址"
                  required
                  autocomplete="email"
                />
                <div class="input-highlight" />
              </div>
            </div>
          </Transition>

          <!-- Password -->
          <div class="field-group">
            <label class="field-label">
              <svg viewBox="0 0 20 20" fill="currentColor" class="label-icon">
                <path fill-rule="evenodd" d="M5 9V7a5 5 0 0110 0v2a2 2 0 012 2v5a2 2 0 01-2 2H5a2 2 0 01-2-2v-5a2 2 0 012-2zm8-2v2H7V7a3 3 0 016 0z" clip-rule="evenodd"/>
              </svg>
              密码
            </label>
            <div class="input-wrapper">
              <input
                v-model="form.password"
                :type="showPassword ? 'text' : 'password'"
                class="fancy-input pr-10"
                placeholder="输入密码"
                required
                autocomplete="current-password"
              />
              <button
                type="button"
                class="password-toggle"
                @click="showPassword = !showPassword"
                tabindex="-1"
              >
                <svg v-if="!showPassword" viewBox="0 0 20 20" fill="currentColor" class="w-4 h-4">
                  <path d="M10 12a2 2 0 100-4 2 2 0 000 4z"/>
                  <path fill-rule="evenodd" d="M.458 10C1.732 5.943 5.522 3 10 3s8.268 2.943 9.542 7c-1.274 4.057-5.064 7-9.542 7S1.732 14.057.458 10zM14 10a4 4 0 11-8 0 4 4 0 018 0z" clip-rule="evenodd"/>
                </svg>
                <svg v-else viewBox="0 0 20 20" fill="currentColor" class="w-4 h-4">
                  <path fill-rule="evenodd" d="M3.707 2.293a1 1 0 00-1.414 1.414l14 14a1 1 0 001.414-1.414l-1.473-1.473A10.014 10.014 0 0019.542 10C18.268 5.943 14.478 3 10 3a9.958 9.958 0 00-4.512 1.074l-1.78-1.781zm4.261 4.26l1.514 1.515a2.003 2.003 0 012.45 2.45l1.514 1.514a4 4 0 00-5.478-5.478z" clip-rule="evenodd"/>
                  <path d="M12.454 16.697L9.75 13.992a4 4 0 01-3.742-3.741L2.335 6.578A9.98 9.98 0 00.458 10c1.274 4.057 5.064 7 9.542 7 .847 0 1.669-.105 2.454-.303z"/>
                </svg>
              </button>
              <div class="input-highlight" />
            </div>
          </div>

          <!-- Invite code (register only) -->
          <Transition name="field-slide">
            <div v-if="mode === 'register'" class="field-group">
              <label class="field-label">
                <svg viewBox="0 0 20 20" fill="currentColor" class="label-icon">
                  <path fill-rule="evenodd" d="M5 2a1 1 0 011 1v1h1a1 1 0 010 2H6v1a1 1 0 01-2 0V6H3a1 1 0 010-2h1V3a1 1 0 011-1zm0 10a1 1 0 011 1v1h1a1 1 0 110 2H6v1a1 1 0 11-2 0v-1H3a1 1 0 110-2h1v-1a1 1 0 011-1zM12 2a1 1 0 01.967.744L14.146 7.2 17.5 9.134a1 1 0 010 1.732l-3.354 1.935-1.18 4.455a1 1 0 01-1.933 0L9.854 12.8 6.5 10.866a1 1 0 010-1.732l3.354-1.935 1.18-4.455A1 1 0 0112 2z" clip-rule="evenodd"/>
                </svg>
                邀请码
                <span class="optional-tag">首次注册可选</span>
              </label>
              <div class="input-wrapper">
                <input
                  v-model="form.invite_code"
                  type="text"
                  class="fancy-input"
                  placeholder="输入邀请码（首个用户无需填写）"
                />
                <div class="input-highlight" />
              </div>
            </div>
          </Transition>

          <!-- Error / Success message -->
          <Transition name="fade">
            <div v-if="error" class="message-box" :class="isSuccess ? 'message-success' : 'message-error'">
              <svg v-if="isSuccess" viewBox="0 0 20 20" fill="currentColor" class="message-icon">
                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"/>
              </svg>
              <svg v-else viewBox="0 0 20 20" fill="currentColor" class="message-icon">
                <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd"/>
              </svg>
              {{ error }}
            </div>
          </Transition>

          <!-- Submit button -->
          <button type="submit" class="submit-btn" :disabled="loading">
            <span v-if="!loading" class="btn-content">
              <svg viewBox="0 0 20 20" fill="currentColor" class="btn-icon">
                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-8.707l-3-3a1 1 0 00-1.414 1.414L10.586 9H7a1 1 0 100 2h3.586l-1.293 1.293a1 1 0 101.414 1.414l3-3a1 1 0 000-1.414z" clip-rule="evenodd"/>
              </svg>
              {{ mode === 'login' ? '登录账号' : '创建账号' }}
            </span>
            <span v-else class="btn-loading">
              <svg class="spin-icon" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-30" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3"/>
                <path class="opacity-80" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z"/>
              </svg>
              处理中...
            </span>
            <div class="btn-shimmer" />
          </button>
        </form>
      </div>

      <!-- Footer -->
      <div class="login-footer">
        <span class="footer-dot" />
        <span>SecFlow v1.0</span>
        <span class="footer-sep">·</span>
        <span>安全情报监控</span>
        <span class="footer-dot" />
      </div>
    </div>
  </div>
</template>

<style scoped>
/* ─── Page container ─────────────────────────────────────────────────────── */
.login-page {
  min-height: 100vh;
  background: #080f1e;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.5rem;
  overflow: hidden;
  position: relative;
}

/* ─── Animated orbs ──────────────────────────────────────────────────────── */
.orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  pointer-events: none;
  opacity: 0;
  animation: orb-fade-in 1.5s ease forwards;
}

@keyframes orb-fade-in {
  to { opacity: 1; }
}

.orb-1 {
  width: 500px; height: 500px;
  background: radial-gradient(circle, rgba(14, 165, 233, 0.18) 0%, transparent 70%);
  top: -150px; left: -100px;
  animation: orb-float-1 12s ease-in-out infinite, orb-fade-in 1.5s ease forwards;
}
.orb-2 {
  width: 600px; height: 600px;
  background: radial-gradient(circle, rgba(139, 92, 246, 0.12) 0%, transparent 70%);
  bottom: -200px; right: -150px;
  animation: orb-float-2 15s ease-in-out infinite, orb-fade-in 1.5s ease 0.3s forwards;
}
.orb-3 {
  width: 300px; height: 300px;
  background: radial-gradient(circle, rgba(16, 185, 129, 0.1) 0%, transparent 70%);
  top: 50%; left: 50%;
  transform: translate(-50%, -50%);
  animation: orb-pulse 8s ease-in-out infinite, orb-fade-in 1.5s ease 0.6s forwards;
}

@keyframes orb-float-1 {
  0%, 100% { transform: translate(0, 0) scale(1); }
  33% { transform: translate(30px, 20px) scale(1.05); }
  66% { transform: translate(-20px, 30px) scale(0.95); }
}
@keyframes orb-float-2 {
  0%, 100% { transform: translate(0, 0) scale(1); }
  50% { transform: translate(-40px, -30px) scale(1.08); }
}
@keyframes orb-pulse {
  0%, 100% { transform: translate(-50%, -50%) scale(1); opacity: 0.4; }
  50% { transform: translate(-50%, -50%) scale(1.3); opacity: 0.15; }
}

/* ─── Grid overlay ───────────────────────────────────────────────────────── */
.grid-overlay {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background-image:
    linear-gradient(rgba(148, 163, 184, 0.04) 1px, transparent 1px),
    linear-gradient(90deg, rgba(148, 163, 184, 0.04) 1px, transparent 1px);
  background-size: 40px 40px;
}

/* ─── Floating particles ─────────────────────────────────────────────────── */
.particle {
  position: absolute;
  border-radius: 50%;
  pointer-events: none;
  background: rgba(14, 165, 233, 0.6);
}
.p-1 { width: 3px; height: 3px; top: 20%; left: 15%; animation: float-particle 6s 0s ease-in-out infinite; }
.p-2 { width: 2px; height: 2px; top: 70%; left: 80%; animation: float-particle 8s 1s ease-in-out infinite; }
.p-3 { width: 4px; height: 4px; top: 40%; right: 20%; background: rgba(139, 92, 246, 0.5); animation: float-particle 7s 2s ease-in-out infinite; }
.p-4 { width: 2px; height: 2px; bottom: 30%; left: 30%; animation: float-particle 9s 0.5s ease-in-out infinite; }
.p-5 { width: 3px; height: 3px; top: 15%; right: 35%; background: rgba(16, 185, 129, 0.5); animation: float-particle 5s 3s ease-in-out infinite; }

@keyframes float-particle {
  0%, 100% { transform: translateY(0) scale(1); opacity: 0.6; }
  50% { transform: translateY(-20px) scale(1.5); opacity: 0.2; }
}

/* ─── Wrapper ────────────────────────────────────────────────────────────── */
.login-card-wrapper {
  position: relative;
  width: 100%;
  max-width: 420px;
  animation: wrapper-enter 0.7s cubic-bezier(0.16, 1, 0.3, 1) forwards;
}

@keyframes wrapper-enter {
  from { opacity: 0; transform: translateY(24px) scale(0.97); }
  to   { opacity: 1; transform: translateY(0) scale(1); }
}

/* ─── Logo section ───────────────────────────────────────────────────────── */
.logo-section {
  text-align: center;
  margin-bottom: 2rem;
  position: relative;
}
.logo-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 56px; height: 56px;
  border-radius: 18px;
  background: linear-gradient(135deg, rgba(14, 165, 233, 0.2), rgba(139, 92, 246, 0.15));
  border: 1px solid rgba(14, 165, 233, 0.3);
  margin-bottom: 1rem;
  position: relative;
  backdrop-filter: blur(10px);
  box-shadow: 0 0 30px rgba(14, 165, 233, 0.15), inset 0 1px 0 rgba(255,255,255,0.08);
}
.icon-shield {
  width: 28px; height: 28px;
  color: #38bdf8;
  filter: drop-shadow(0 0 8px rgba(56, 189, 248, 0.5));
}
.logo-glow {
  position: absolute;
  top: -20px; left: 50%;
  transform: translateX(-50%);
  width: 100px; height: 100px;
  background: radial-gradient(circle, rgba(14, 165, 233, 0.2), transparent 70%);
  pointer-events: none;
}
.logo-title {
  font-size: 1.75rem;
  font-weight: 800;
  letter-spacing: -0.03em;
  background: linear-gradient(135deg, #f8fafc 30%, #38bdf8 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  margin: 0 0 0.25rem;
}
.logo-subtitle {
  font-size: 0.8125rem;
  color: #64748b;
  letter-spacing: 0.02em;
}

/* ─── Glass card ─────────────────────────────────────────────────────────── */
.glass-card {
  background: rgba(15, 23, 42, 0.75);
  backdrop-filter: blur(24px) saturate(180%);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 20px;
  padding: 2rem;
  box-shadow:
    0 25px 50px rgba(0, 0, 0, 0.5),
    0 0 0 1px rgba(255, 255, 255, 0.04),
    inset 0 1px 0 rgba(255, 255, 255, 0.06);
  position: relative;
  overflow: hidden;
}
/* Top edge glow */
.glass-card::before {
  content: '';
  position: absolute;
  top: 0; left: 20%; right: 20%;
  height: 1px;
  background: linear-gradient(90deg, transparent, rgba(14, 165, 233, 0.4), transparent);
}

/* ─── Mode tabs ──────────────────────────────────────────────────────────── */
.mode-tabs {
  display: flex;
  background: rgba(0, 0, 0, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 12px;
  padding: 4px;
  margin-bottom: 1.75rem;
  position: relative;
}
.tab-indicator {
  position: absolute;
  top: 4px; bottom: 4px;
  left: 4px;
  width: calc(50% - 4px);
  background: linear-gradient(135deg, rgba(14, 165, 233, 0.25), rgba(14, 165, 233, 0.1));
  border: 1px solid rgba(14, 165, 233, 0.3);
  border-radius: 9px;
  transition: transform 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
  box-shadow: 0 0 12px rgba(14, 165, 233, 0.1);
}
.tab-indicator.tab-right {
  transform: translateX(calc(100% + 0px));
}
.mode-tab {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
  padding: 0.5rem 1rem;
  font-size: 0.875rem;
  font-weight: 500;
  border-radius: 9px;
  color: #64748b;
  transition: color 0.2s ease;
  position: relative;
  z-index: 1;
  border: none;
  background: transparent;
  cursor: pointer;
}
.mode-tab.active {
  color: #e2e8f0;
}
.tab-icon {
  width: 15px; height: 15px;
}

/* ─── Field ──────────────────────────────────────────────────────────────── */
.login-form {
  display: flex;
  flex-direction: column;
  gap: 1.1rem;
}
.field-group {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}
.field-label {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.75rem;
  font-weight: 500;
  color: #94a3b8;
  letter-spacing: 0.02em;
}
.label-icon {
  width: 13px; height: 13px;
  color: #38bdf8;
  opacity: 0.7;
}
.optional-tag {
  margin-left: auto;
  font-size: 0.65rem;
  color: #475569;
  background: rgba(71, 85, 105, 0.2);
  padding: 1px 6px;
  border-radius: 4px;
  border: 1px solid rgba(71, 85, 105, 0.3);
}

/* ─── Input ──────────────────────────────────────────────────────────────── */
.input-wrapper {
  position: relative;
}
.fancy-input {
  width: 100%;
  background: rgba(0, 0, 0, 0.35);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 10px;
  padding: 0.65rem 0.875rem;
  font-size: 0.875rem;
  color: #e2e8f0;
  outline: none;
  transition: border-color 0.2s ease, box-shadow 0.2s ease, background 0.2s ease;
  box-sizing: border-box;
  font-family: inherit;
}
.fancy-input::placeholder { color: #475569; }
.fancy-input:focus {
  border-color: rgba(14, 165, 233, 0.5);
  background: rgba(14, 165, 233, 0.04);
  box-shadow: 0 0 0 3px rgba(14, 165, 233, 0.08), 0 0 15px rgba(14, 165, 233, 0.06);
}
.fancy-input.pr-10 { padding-right: 2.5rem; }
.input-highlight {
  position: absolute;
  bottom: 0; left: 10%; right: 10%;
  height: 1px;
  background: linear-gradient(90deg, transparent, rgba(14, 165, 233, 0.4), transparent);
  opacity: 0;
  transition: opacity 0.2s ease;
  pointer-events: none;
}
.fancy-input:focus ~ .input-highlight {
  opacity: 1;
}
.password-toggle {
  position: absolute;
  right: 0.75rem;
  top: 50%;
  transform: translateY(-50%);
  color: #475569;
  background: transparent;
  border: none;
  cursor: pointer;
  padding: 2px;
  transition: color 0.2s;
  display: flex;
  align-items: center;
}
.password-toggle:hover { color: #94a3b8; }

/* ─── Messages ───────────────────────────────────────────────────────────── */
.message-box {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8125rem;
  padding: 0.6rem 0.875rem;
  border-radius: 9px;
  font-weight: 450;
}
.message-success {
  color: #34d399;
  background: rgba(16, 185, 129, 0.08);
  border: 1px solid rgba(16, 185, 129, 0.2);
}
.message-error {
  color: #f87171;
  background: rgba(239, 68, 68, 0.08);
  border: 1px solid rgba(239, 68, 68, 0.2);
}
.message-icon { width: 15px; height: 15px; flex-shrink: 0; }

/* ─── Submit button ──────────────────────────────────────────────────────── */
.submit-btn {
  position: relative;
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.75rem 1.5rem;
  font-size: 0.9375rem;
  font-weight: 600;
  border: none;
  border-radius: 12px;
  cursor: pointer;
  overflow: hidden;
  margin-top: 0.25rem;
  color: #fff;
  background: linear-gradient(135deg, #0284c7, #0369a1);
  box-shadow: 0 4px 15px rgba(2, 132, 199, 0.3), 0 0 0 1px rgba(14, 165, 233, 0.2);
  transition: transform 0.15s ease, box-shadow 0.15s ease;
  font-family: inherit;
}
.submit-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 8px 25px rgba(2, 132, 199, 0.4), 0 0 0 1px rgba(14, 165, 233, 0.3);
}
.submit-btn:active:not(:disabled) {
  transform: translateY(0);
  box-shadow: 0 2px 8px rgba(2, 132, 199, 0.2);
}
.submit-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.btn-shimmer {
  position: absolute;
  top: 0; left: -100%; width: 60%; height: 100%;
  background: linear-gradient(90deg, transparent, rgba(255,255,255,0.08), transparent);
  transition: left 0.5s ease;
  pointer-events: none;
}
.submit-btn:hover .btn-shimmer { left: 150%; }
.btn-content {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  position: relative;
  z-index: 1;
}
.btn-icon { width: 17px; height: 17px; }
.btn-loading {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  position: relative;
  z-index: 1;
}
.spin-icon {
  width: 16px; height: 16px;
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}

/* ─── Footer ─────────────────────────────────────────────────────────────── */
.login-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  margin-top: 1.5rem;
  font-size: 0.7rem;
  color: #334155;
  letter-spacing: 0.04em;
}
.footer-dot {
  width: 3px; height: 3px;
  border-radius: 50%;
  background: #334155;
}
.footer-sep { color: #2d3748; }

/* ─── Transitions ────────────────────────────────────────────────────────── */
.field-slide-enter-active,
.field-slide-leave-active {
  transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
  overflow: hidden;
}
.field-slide-enter-from {
  opacity: 0;
  max-height: 0;
  transform: translateY(-8px);
}
.field-slide-enter-to {
  opacity: 1;
  max-height: 100px;
  transform: translateY(0);
}
.field-slide-leave-from {
  opacity: 1;
  max-height: 100px;
  transform: translateY(0);
}
.field-slide-leave-to {
  opacity: 0;
  max-height: 0;
  transform: translateY(-8px);
}

.fade-enter-active, .fade-leave-active {
  transition: all 0.25s ease;
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
