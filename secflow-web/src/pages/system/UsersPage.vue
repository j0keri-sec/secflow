<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { UserProfile, RoleType } from '@/types'
import { authApi } from '@/api/auth'

const users = ref<UserProfile[]>([])
const loading = ref(true)
const inviteCodes = ref<{ id: string; code: string; used: boolean; created_at: string }[]>([])
const showInviteModal = ref(false)
const generatingCode = ref(false)
const actionLoading = ref<string | null>(null)

async function fetchUsers() {
  const res = await authApi.adminListUsers()
  users.value = res
}

async function fetchInviteCodes() {
  const res = await authApi.listInviteCodes()
  inviteCodes.value = res
}

onMounted(async () => {
  try {
    await Promise.all([fetchUsers(), fetchInviteCodes()])
  } finally {
    loading.value = false
  }
})

async function generateCode() {
  generatingCode.value = true
  try {
    await authApi.generateInviteCode()
    await fetchInviteCodes()
  } finally {
    generatingCode.value = false
  }
}

async function toggleActive(user: UserProfile) {
  actionLoading.value = user.id
  try {
    await authApi.adminUpdateUser(user.id, { active: !user.active })
    await fetchUsers()
  } finally {
    actionLoading.value = null
  }
}

async function changeRole(user: UserProfile, role: RoleType) {
  actionLoading.value = user.id
  try {
    await authApi.adminUpdateUser(user.id, { role })
    await fetchUsers()
  } finally {
    actionLoading.value = null
  }
}

function copyCode(code: string) {
  navigator.clipboard.writeText(code)
}
</script>

<template>
  <div class="ops-page">
    <!-- 工具栏 -->
    <div class="toolbar">
      <span class="toolbar-info">共 {{ users.length }} 名用户</span>
      <div class="toolbar-actions">
        <button class="btn-secondary" @click="showInviteModal = true">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"/>
          </svg>
          邀请码管理
        </button>
      </div>
    </div>

    <!-- 用户表格 -->
    <div class="table-container">
      <div v-if="loading" class="loading-state">
        <div v-for="i in 5" :key="i" class="skeleton-row"></div>
      </div>

      <div v-else-if="users.length === 0" class="empty-state">
        <div class="empty-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2"/>
            <circle cx="9" cy="7" r="4"/>
          </svg>
        </div>
        <p class="empty-text">暂无用户</p>
      </div>

      <table v-else class="data-table">
        <thead>
          <tr>
            <th>用户名</th>
            <th>邮箱</th>
            <th>角色</th>
            <th>状态</th>
            <th>注册时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="user in users" :key="user.id">
            <td>
              <div class="user-cell">
                <div class="user-avatar">{{ user.username.charAt(0).toUpperCase() }}</div>
                <span class="user-name">{{ user.username }}</span>
              </div>
            </td>
            <td><span class="text-secondary">{{ user.email || '—' }}</span></td>
            <td>
              <select
                :value="user.role"
                class="role-select"
                :disabled="actionLoading === user.id"
                @change="(e) => changeRole(user, (e.target as HTMLSelectElement).value as RoleType)"
              >
                <option value="admin">admin</option>
                <option value="editor">editor</option>
                <option value="viewer">viewer</option>
              </select>
            </td>
            <td>
              <span class="status-badge" :class="user.active ? 'status-active' : 'status-inactive'">
                {{ user.active ? '正常' : '禁用' }}
              </span>
            </td>
            <td><span class="text-secondary">{{ new Date(user.created_at).toLocaleDateString('zh-CN') }}</span></td>
            <td>
              <button
                class="btn-toggle"
                :disabled="actionLoading === user.id"
                @click="toggleActive(user)"
              >
                {{ user.active ? '禁用' : '启用' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 邀请码弹窗 -->
    <Teleport to="body">
      <Transition name="fade">
        <div v-if="showInviteModal" class="modal-overlay" @click="showInviteModal = false">
          <div class="modal-content" @click.stop>
            <div class="modal-header">
              <h2 class="modal-title">邀请码管理</h2>
              <button class="modal-close" @click="showInviteModal = false">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18"/>
                  <line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
              </button>
            </div>
            <div class="modal-body">
              <div class="code-header">
                <p class="code-hint">管理员可无限生成，普通用户最多 5 个</p>
                <button class="btn-primary" :disabled="generatingCode" @click="generateCode">
                  {{ generatingCode ? '生成中...' : '生成邀请码' }}
                </button>
              </div>
              <div class="code-list">
                <div v-if="!inviteCodes.length" class="code-empty">暂无邀请码</div>
                <div v-for="code in inviteCodes" :key="code.id" class="code-item" :class="{ used: code.used }">
                  <div class="code-info">
                    <code class="code-text">{{ code.code }}</code>
                    <span class="code-time">{{ new Date(code.created_at).toLocaleString('zh-CN') }}</span>
                  </div>
                  <div class="code-actions">
                    <span class="status-badge" :class="code.used ? 'status-inactive' : 'status-active'">
                      {{ code.used ? '已使用' : '未使用' }}
                    </span>
                    <button v-if="!code.used" class="btn-copy" @click="copyCode(code.code)">
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
                        <path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/>
                      </svg>
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<style scoped>
.ops-page {
  padding: 0 0 2rem;
  animation: fadeIn 0.4s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1rem;
  margin-bottom: 1rem;
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.toolbar-info {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.toolbar-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.btn-primary {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.6rem 1rem;
  background: var(--gradient-primary);
  border: none;
  border-radius: var(--radius-sm);
  color: white;
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}
.btn-primary:hover { opacity: 0.9; }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

.btn-secondary {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.6rem 1rem;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  color: var(--color-text-secondary);
  font-size: 0.85rem;
  cursor: pointer;
  transition: all 0.2s ease;
}
.btn-secondary:hover { background: var(--color-bg-secondary); }

/* 表格容器 */
.table-container {
  background: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
}

.data-table th {
  padding: 0.75rem 1rem;
  text-align: left;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text-tertiary);
  background: var(--color-bg-secondary);
  border-bottom: 1px solid var(--color-border);
}

.data-table td {
  padding: 0.75rem 1rem;
  font-size: 0.8rem;
  color: var(--color-text-primary);
  border-bottom: 1px solid var(--color-border-light);
}

.data-table tr:hover td {
  background: var(--color-bg-secondary);
}

.data-table tr:last-child td {
  border-bottom: none;
}

.text-secondary {
  color: var(--color-text-secondary);
  font-size: 0.8rem;
}

/* 用户单元格 */
.user-cell {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.user-avatar {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(59, 130, 246, 0.15);
  color: var(--color-primary);
  border-radius: 50%;
  font-size: 0.75rem;
  font-weight: 600;
}

.user-name {
  font-weight: 500;
}

/* 角色选择 */
.role-select {
  padding: 0.35rem 0.5rem;
  background: var(--color-bg-tertiary);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
  color: var(--color-text-primary);
  cursor: pointer;
}
.role-select:focus {
  outline: none;
  border-color: var(--color-primary);
}

/* 状态徽章 */
.status-badge {
  display: inline-block;
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
  font-size: 0.65rem;
  font-weight: 600;
}

.status-active { background: rgba(34, 197, 94, 0.15); color: #22c55e; }
.status-inactive { background: rgba(107, 114, 128, 0.15); color: #6b7280; }

/* 操作按钮 */
.btn-toggle {
  padding: 0.4rem 0.75rem;
  background: var(--color-bg-tertiary);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}
.btn-toggle:hover {
  background: var(--color-border);
  color: var(--color-text-primary);
}
.btn-toggle:disabled { opacity: 0.5; cursor: not-allowed; }

/* 骨架屏 */
.loading-state {
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.skeleton-row {
  height: 48px;
  background: linear-gradient(90deg, var(--color-bg-tertiary) 25%, var(--color-border) 50%, var(--color-bg-tertiary) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: 4px;
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* 空状态 */
.empty-state {
  text-align: center;
  padding: 3rem 1rem;
}

.empty-icon {
  display: flex;
  justify-content: center;
  margin-bottom: 1rem;
  color: var(--color-text-tertiary);
}

.empty-text {
  font-size: 1rem;
  color: var(--color-text-secondary);
  margin: 0;
}

/* 弹窗 */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.modal-content {
  background: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  width: 100%;
  max-width: 480px;
  max-height: 90vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--color-border);
}

.modal-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}

.modal-close {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  border-radius: 6px;
  color: var(--color-text-tertiary);
  cursor: pointer;
  transition: all 0.2s ease;
}
.modal-close:hover {
  background: var(--color-bg-tertiary);
  color: var(--color-text-primary);
}

.modal-body {
  padding: 1.25rem;
  overflow-y: auto;
  flex: 1;
}

.code-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
}

.code-hint {
  font-size: 0.8rem;
  color: var(--color-text-tertiary);
  margin: 0;
}

.code-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  max-height: 300px;
  overflow-y: auto;
}

.code-empty {
  text-align: center;
  padding: 2rem;
  color: var(--color-text-tertiary);
  font-size: 0.85rem;
}

.code-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1rem;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  transition: all 0.2s ease;
}

.code-item:hover {
  border-color: var(--color-border);
}

.code-item.used {
  opacity: 0.5;
}

.code-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.code-text {
  font-family: monospace;
  font-size: 0.9rem;
  color: var(--color-primary);
  font-weight: 500;
}

.code-item.used .code-text {
  color: var(--color-text-tertiary);
  text-decoration: line-through;
}

.code-time {
  font-size: 0.7rem;
  color: var(--color-text-tertiary);
}

.code-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.btn-copy {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg-tertiary);
  border: none;
  border-radius: 6px;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}
.btn-copy:hover {
  background: var(--color-border);
  color: var(--color-text-primary);
}

/* 过渡 */
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from, .fade-leave-to { opacity: 0; }

/* 响应式 */
@media (max-width: 768px) {
  .page-header { flex-direction: column; gap: 1rem; align-items: flex-start; }
  .data-table { display: block; overflow-x: auto; }
  .code-header { flex-direction: column; gap: 0.75rem; align-items: flex-start; }
}
</style>
