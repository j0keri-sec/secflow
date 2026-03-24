<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { systemApi } from '@/api/system'

interface PushChannel {
  id: string
  name: string
  type: string
  enabled: boolean
  config: Record<string, string>
  created_at: string
}

const channels = ref<PushChannel[]>([])
const loading = ref(true)
const showModal = ref(false)
const saving = ref(false)
const editTarget = ref<PushChannel | null>(null)

const blankForm = (): Partial<PushChannel> & { config: Record<string, string> } => ({
  name: '',
  type: 'dingtalk',
  enabled: true,
  config: {},
})

const form = ref(blankForm())

const pushTypes = [
  { value: 'dingtalk', label: '钉钉', icon: 'D' },
  { value: 'feishu', label: '飞书', icon: 'F' },
  { value: 'wecom', label: '企业微信', icon: 'W' },
  { value: 'telegram', label: 'Telegram', icon: 'T' },
  { value: 'slack', label: 'Slack', icon: 'S' },
  { value: 'webhook', label: 'Webhook', icon: 'H' },
  { value: 'bark', label: 'Bark', icon: 'B' },
  { value: 'serverchan', label: 'Server酱', icon: 'S' },
  { value: 'pushplus', label: 'PushPlus', icon: 'P' },
]

const typeFields: Record<string, { key: string; label: string; placeholder: string }[]> = {
  dingtalk: [
    { key: 'webhook', label: 'Webhook URL', placeholder: 'https://oapi.dingtalk.com/robot/send?access_token=…' },
    { key: 'secret', label: '签名密钥（可选）', placeholder: 'SEC…' },
  ],
  feishu: [{ key: 'webhook', label: 'Webhook URL', placeholder: 'https://open.feishu.cn/open-apis/bot/v2/hook/…' }],
  wecom: [{ key: 'webhook', label: 'Webhook URL', placeholder: 'https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=…' }],
  telegram: [
    { key: 'token', label: 'Bot Token', placeholder: '123456789:AAE…' },
    { key: 'chat_id', label: 'Chat ID', placeholder: '-100…' },
  ],
  slack: [{ key: 'webhook', label: 'Webhook URL', placeholder: 'https://hooks.slack.com/services/…' }],
  webhook: [{ key: 'url', label: 'URL', placeholder: 'https://…' }],
  bark: [{ key: 'url', label: 'Bark URL', placeholder: 'https://api.day.app/your_key/' }],
  serverchan: [{ key: 'send_key', label: 'SendKey', placeholder: 'SCT…' }],
  pushplus: [{ key: 'token', label: 'Token', placeholder: '…' }],
}

const currentFields = () => typeFields[form.value.type ?? 'dingtalk'] ?? []

function openCreate() {
  editTarget.value = null
  form.value = blankForm()
  showModal.value = true
}

function openEdit(ch: PushChannel) {
  editTarget.value = ch
  form.value = { ...ch, config: { ...ch.config } }
  showModal.value = true
}

async function fetchChannels() {
  const res = await systemApi.listPushChannels()
  channels.value = res
}

onMounted(async () => {
  try { await fetchChannels() } finally { loading.value = false }
})

async function save() {
  saving.value = true
  try {
    if (editTarget.value) {
      await systemApi.updatePushChannel(editTarget.value.id, form.value)
    } else {
      await systemApi.createPushChannel(form.value)
    }
    showModal.value = false
    await fetchChannels()
  } finally {
    saving.value = false
  }
}

async function deleteChannel(id: string) {
  if (!confirm('确认删除该推送渠道？')) return
  await systemApi.deletePushChannel(id)
  await fetchChannels()
}

async function toggle(ch: PushChannel) {
  await systemApi.updatePushChannel(ch.id, { enabled: !ch.enabled })
  await fetchChannels()
}

function getTypeInfo(type: string) {
  return pushTypes.find(t => t.value === type) || { label: type, icon: '?' }
}
</script>

<template>
  <div class="ops-page">
    <!-- 工具栏 -->
    <div class="toolbar">
      <span class="toolbar-info">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M22 17H2a3 3 0 000 6h20a3 3 0 000-6zM6 17V7a4 4 0 018 0v10"/>
        </svg>
        {{ channels.length }} 个推送渠道
      </span>
      <span class="toolbar-hint">点击卡片可编辑渠道配置</span>
      <button class="btn-primary" @click="openCreate">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="5" x2="12" y2="19"/>
          <line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        添加渠道
      </button>
    </div>

    <!-- 推送渠道网格 -->
    <div class="channel-container">
      <div v-if="loading" class="channel-grid">
        <div v-for="i in 3" :key="i" class="channel-card skeleton">
          <div class="skeleton-header"></div>
          <div class="skeleton-body"></div>
        </div>
      </div>

      <div v-else-if="channels.length === 0" class="empty-state">
        <div class="empty-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M22 17H2a3 3 0 000 6h20a3 3 0 000-6zM6 17V7a4 4 0 018 0v10"/>
          </svg>
        </div>
        <p class="empty-text">尚未配置推送渠道</p>
        <p class="empty-hint">点击右上角按钮添加第一个推送渠道</p>
      </div>

      <div v-else class="channel-grid">
        <div v-for="ch in channels" :key="ch.id" class="channel-card">
          <div class="channel-header">
            <div class="channel-icon" :class="'icon-' + ch.type">
              {{ getTypeInfo(ch.type).icon }}
            </div>
            <div class="channel-info">
              <span class="channel-name">{{ ch.name }}</span>
              <span class="channel-type">{{ getTypeInfo(ch.type).label }}</span>
            </div>
            <div class="toggle-switch" :class="{ active: ch.enabled }" @click="toggle(ch)">
              <div class="toggle-dot"></div>
            </div>
          </div>
          <div class="channel-actions">
            <button class="btn-edit" @click="openEdit(ch)">编辑</button>
            <button class="btn-delete" @click="deleteChannel(ch.id)">删除</button>
          </div>
        </div>
      </div>
    </div>

    <!-- 添加/编辑弹窗 -->
    <Teleport to="body">
      <Transition name="fade">
        <div v-if="showModal" class="modal-overlay" @click="showModal = false">
          <div class="modal-content" @click.stop>
            <div class="modal-header">
              <h2 class="modal-title">{{ editTarget ? '编辑渠道' : '添加推送渠道' }}</h2>
              <button class="modal-close" @click="showModal = false">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18"/>
                  <line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
              </button>
            </div>
            <div class="modal-body">
              <div class="form-group">
                <label class="form-label">渠道名称</label>
                <input v-model="form.name" class="form-input" placeholder="我的钉钉机器人" />
              </div>
              <div class="form-group">
                <label class="form-label">推送类型</label>
                <select v-model="form.type" class="form-input" @change="form.config = {}">
                  <option v-for="t in pushTypes" :key="t.value" :value="t.value">{{ t.label }}</option>
                </select>
              </div>
              <div v-for="field in currentFields()" :key="field.key" class="form-group">
                <label class="form-label">{{ field.label }}</label>
                <input v-model="form.config![field.key]" class="form-input" :placeholder="field.placeholder" />
              </div>
              <div class="form-checkbox">
                <input id="enabled" v-model="form.enabled" type="checkbox" />
                <label for="enabled">立即启用</label>
              </div>
            </div>
            <div class="modal-footer">
              <button class="btn-secondary" @click="showModal = false">取消</button>
              <button class="btn-primary" :disabled="saving" @click="save">保存</button>
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

/* 工具栏 */
.toolbar {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.75rem 1rem;
  margin-bottom: 1rem;
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.toolbar-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.toolbar-hint {
  flex: 1;
  font-size: 0.8rem;
  color: var(--text-tertiary);
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
  padding: 0.6rem 1rem;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  font-size: 0.85rem;
  cursor: pointer;
  transition: all 0.2s ease;
}
.btn-secondary:hover { background: var(--bg-secondary); }

/* 渠道容器 */
.channel-container {
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 1.5rem;
}

.channel-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
}

.channel-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-light);
  border-radius: var(--radius-md);
  padding: 1rem;
  transition: all 0.3s ease;
}

.channel-card:hover {
  border-color: var(--color-primary);
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.channel-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.channel-icon {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--gradient-primary);
  border-radius: var(--radius-md);
  color: white;
  font-weight: 700;
  font-size: 1rem;
  flex-shrink: 0;
}

.channel-info {
  flex: 1;
  min-width: 0;
}

.channel-name {
  display: block;
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.channel-type {
  font-size: 0.75rem;
  color: var(--text-tertiary);
}

/* 开关 */
.toggle-switch {
  width: 44px;
  height: 24px;
  background: var(--bg-tertiary);
  border-radius: 12px;
  position: relative;
  cursor: pointer;
  transition: background 0.2s ease;
}

.toggle-switch.active {
  background: var(--color-primary);
}

.toggle-dot {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 20px;
  height: 20px;
  background: white;
  border-radius: 50%;
  transition: transform 0.2s ease;
  box-shadow: 0 1px 3px rgba(0,0,0,0.2);
}

.toggle-switch.active .toggle-dot {
  transform: translateX(20px);
}

/* 操作按钮 */
.channel-actions {
  display: flex;
  gap: 0.5rem;
}

.btn-edit, .btn-delete {
  flex: 1;
  padding: 0.5rem;
  border-radius: var(--radius-sm);
  font-size: 0.8rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-edit {
  background: var(--bg-tertiary);
  border: 1px solid var(--border-light);
  color: var(--text-secondary);
}
.btn-edit:hover {
  background: var(--border);
  color: var(--text-primary);
}

.btn-delete {
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.2);
  color: #ef4444;
}
.btn-delete:hover {
  background: rgba(239, 68, 68, 0.2);
}

/* 骨架屏 */
.skeleton .skeleton-header,
.skeleton .skeleton-body {
  background: linear-gradient(90deg, var(--bg-tertiary) 25%, var(--border) 50%, var(--bg-tertiary) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: 4px;
}

.skeleton-header {
  height: 40px;
  margin-bottom: 1rem;
}

.skeleton-body {
  height: 32px;
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
  color: var(--text-tertiary);
}

.empty-text {
  font-size: 1rem;
  color: var(--text-secondary);
  margin: 0 0 0.5rem;
}

.empty-hint {
  font-size: 0.85rem;
  color: var(--text-tertiary);
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
  background: var(--bg-primary);
  border: 1px solid var(--border);
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
  border-bottom: 1px solid var(--border);
}

.modal-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
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
  color: var(--text-tertiary);
  cursor: pointer;
  transition: all 0.2s ease;
}
.modal-close:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.modal-body {
  padding: 1.25rem;
  overflow-y: auto;
  flex: 1;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  padding: 1rem 1.25rem;
  border-top: 1px solid var(--border);
}

/* 表单 */
.form-group {
  margin-bottom: 1rem;
}

.form-label {
  display: block;
  font-size: 0.8rem;
  color: var(--text-secondary);
  margin-bottom: 0.5rem;
}

.form-input {
  padding: 0.6rem 0.75rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 0.85rem;
  color: var(--text-primary);
  width: 100%;
}

.form-input:focus {
  outline: none;
  border-color: var(--color-primary);
}

.form-checkbox {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.85rem;
  color: var(--text-secondary);
  cursor: pointer;
}

/* 过渡 */
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from, .fade-leave-to { opacity: 0; }

/* 响应式 */
@media (max-width: 1200px) {
  .channel-grid { grid-template-columns: repeat(2, 1fr); }
}

@media (max-width: 768px) {
  .channel-grid { grid-template-columns: 1fr; }
}
</style>
