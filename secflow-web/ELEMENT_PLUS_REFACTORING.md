# SecFlow Web - Element Plus 重构文档

## 📋 重构概述

本次重构将前端 UI 框架从纯 Tailwind CSS 迁移到 **Element Plus**,提供更稳定、美观、易用的组件库。

---

## 🎯 重构目标

1. ✅ **引入 Element Plus**: 使用成熟的企业级 UI 组件库
2. ✅ **优化布局结构**: 顶部一级菜单 + 底部二级菜单
3. ✅ **提升开发效率**: 减少自定义 CSS 编写
4. ✅ **保持设计风格**: 保留 projectdiscovery 深色主题风格

---

## 📦 安装的依赖

```bash
npm install element-plus @element-plus/icons-vue
```

### 依赖说明
- **element-plus**: Element Plus 核心库
- **@element-plus/icons-vue**: Element Plus 图标库

---

## 🔧 配置更改

### 1. main.ts - 全局引入

```typescript
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'

// 注册所有图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

app.use(ElementPlus)
```

### 2. tailwind.config.js - 简化配置

移除了自定义动画、阴影等,让 Element Plus 组件承担这些功能:

```javascript
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        brand: {
          // ... 保留品牌色
        },
      },
    },
  },
  plugins: [],
}
```

---

## 🎨 布局结构

### 主布局 (AppLayout.vue)

#### 结构概览
```
┌─────────────────────────────────────────┐
│  Logo │ 一级菜单 │ 用户信息 │ 退出   │ ← 顶部导航栏
├─────────────────────────────────────────┤
│  二级菜单 (仅系统管理显示)           │ ← 二级导航栏
├─────────────────────────────────────────┤
│                                     │
│           主内容区                    │ ← 路由视图
│           <router-view />             │
│                                     │
└─────────────────────────────────────────┘
```

#### 使用的 Element Plus 组件

**布局组件**:
- `el-container`: 根容器
- `el-header`: 顶部导航栏
- `el-main`: 主内容区

**导航组件**:
- `el-menu`: 菜单容器
- `el-menu-item`: 菜单项

**其他组件**:
- `el-button`: 按钮
- `el-tag`: 标签
- `el-icon`: 图标

#### 菜单配置

**一级菜单**:
```typescript
const topMenuItems = [
  { id: 'dashboard', label: '仪表盘', path: '/dashboard', icon: Monitor },
  { id: 'vulns',     label: '漏洞信息', path: '/vulns',     icon: ShieldCheck },
  { id: 'articles',  label: '文章热点', path: '/articles',  icon: DocumentText },
  { id: 'system',    label: '系统管理', path: '/system',    icon: Setting },
  { id: 'reports',   label: '报告管理', path: '/reports',   icon: Files },
]
```

**二级菜单** (系统管理):
```typescript
const subMenuItems = computed(() => [
  { id: 'users',  label: '用户管理', path: '/system/users' },  // 管理员可见
  { id: 'nodes',  label: '节点管理', path: '/system/nodes' },
  { id: 'tasks',  label: '任务管理', path: '/system/tasks' },
  { id: 'push',   label: '推送管理', path: '/system/push' },
  { id: 'logs',   label: '日志管理', path: '/system/logs' },
])
```

---

## 🖼️ 页面组件

### 仪表盘 (DashboardPage.vue)

#### 使用的 Element Plus 组件

**布局**:
- `el-row`: 行布局
- `el-col`: 列布局
- `el-space`: 间距布局

**数据展示**:
- `el-card`: 卡片容器
- `el-skeleton`: 骨架屏
- `el-empty`: 空状态
- `el-progress`: 进度条
- `el-tag`: 标签

**交互**:
- `el-button`: 按钮
- `el-icon`: 图标

#### 组件特性

**统计卡片**:
```vue
<el-card shadow="hover" class="stat-card">
  <div class="flex items-center gap-4">
    <div class="icon-wrapper">
      <el-icon :size="28" :color="card.color">
        <component :is="card.icon" />
      </el-icon>
    </div>
    <div class="text-3xl font-bold">{{ card.value }}</div>
  </div>
</el-card>
```

**漏洞列表**:
```vue
<div class="vuln-item">
  <el-tag :type="getSeverityType(v.severity)" size="small">
    {{ v.severity }}
  </el-tag>
  <router-link :to="`/vulns/${v.id}`">{{ v.title }}</router-link>
</div>
```

**严重程度分布**:
```vue
<el-progress
  :percentage="bar.pct"
  :type="bar.type"
  :show-text="false"
  :stroke-width="8"
/>
```

---

## 🎨 样式定制

### 深色主题

Element Plus 默认支持深色主题,通过 CSS 变量控制:

```css
/* main.ts 中引入 */
import 'element-plus/theme-chalk/dark/css-vars.css'
```

### 自定义样式覆盖

```vue
<style scoped>
/* 覆盖 Element Plus 组件样式 */
:deep(.el-card) {
  background-color: #1f2937;
  border-color: #374151;
}

:deep(.el-menu--horizontal) {
  border-bottom: none !important;
}

:deep(.el-menu-item) {
  color: #9ca3af !important;
}

:deep(.el-menu-item:hover) {
  background-color: rgba(64, 158, 255, 0.1) !important;
  color: #409eff !important;
}
</style>
```

---

## 📱 响应式设计

### 断点设置

Element Plus 使用默认的响应式断点:

- `xs`: <768px (移动端)
- `sm`: ≥768px (平板)
- `md`: ≥992px (小屏桌面)
- `lg`: ≥1200px (桌面)
- `xl`: ≥1920px (大屏桌面)

### 响应式列布局

```vue
<el-row :gutter="20">
  <el-col :xs="24" :sm="12" :lg="6">
    <!-- 移动端: 1列 -->
    <!-- 平板: 2列 -->
    <!-- 桌面: 4列 -->
  </el-col>
</el-row>
```

---

## 🔄 迁移指南

### 从 Tailwind CSS 到 Element Plus

| 功能 | Tailwind CSS | Element Plus |
|------|-------------|-------------|
| 布局 | `flex`, `grid` | `el-row`, `el-col`, `el-container` |
| 卡片 | 自定义 `div` + `bg-surface-card` | `el-card` |
| 按钮 | `class="btn btn-primary"` | `el-button type="primary"` |
| 标签 | 自定义 `span` + `badge` | `el-tag` |
| 菜单 | 自定义 `nav` + `router-link` | `el-menu` + `el-menu-item` |
| 加载 | 自定义 `div` + `animate-pulse` | `el-skeleton` |
| 空状态 | 自定义 `div` | `el-empty` |
| 进度条 | 自定义 `div` + `width: %` | `el-progress` |

### 保留的自定义样式

虽然引入了 Element Plus,但仍保留了一些 Tailwind CSS 工具类:

- **布局**: `flex`, `grid`, `gap`, `p-`, `m-`
- **颜色**: `text-white`, `text-gray-400`, `bg-gray-900`
- **尺寸**: `w-full`, `h-screen`, `text-2xl`
- **边框**: `border`, `rounded-`
- **过渡**: `transition-all`

---

## 🎯 图标使用

### 导入图标

```typescript
import {
  Monitor,         // 仪表盘
  ShieldCheck,     // 漏洞
  DocumentText,    // 文章
  Setting,         // 设置
  Files,           // 报告
  User,            // 用户
  ArrowRight,      // 箭头
} from '@element-plus/icons-vue'
```

### 使用图标

```vue
<!-- 静态图标 -->
<el-icon :size="24" color="#409eff">
  <ShieldCheck />
</el-icon>

<!-- 动态图标 -->
<el-icon>
  <component :is="card.icon" />
</el-icon>

<!-- 菜单图标 -->
<el-menu-item>
  <el-icon class="mr-2">
    <Monitor />
  </el-icon>
  <span>仪表盘</span>
</el-menu-item>
```

---

## 🎨 设计规范

### 颜色系统

**Element Plus 默认颜色**:
- `primary`: #409eff (蓝色)
- `success`: #67c23a (绿色)
- `warning`: #e6a23c (橙色)
- `danger`: #f56c6c (红色)
- `info`: #909399 (灰色)

**自定义颜色** (品牌色):
```css
.brand-500: #0ea5e9
```

### 间距规范

使用 Element Plus 默认间距:
- `small`: 8px
- `default`: 16px
- `large`: 24px

或使用 `el-space` 组件:
```vue
<el-space direction="vertical" :size="20">
  <div>内容 1</div>
  <div>内容 2</div>
</el-space>
```

---

## 🚀 性能优化

### 1. 按需引入 (可选)

如果需要减小打包体积,可以按需引入:

```typescript
import { ElButton, ElCard } from 'element-plus'
import 'element-plus/theme-chalk/el-button.css'
import 'element-plus/theme-chalk/el-card.css'

app.use(ElButton)
app.use(ElCard)
```

### 2. 图标按需引入

```typescript
import { Monitor, ShieldCheck } from '@element-plus/icons-vue'
app.component('Monitor', Monitor)
app.component('ShieldCheck', ShieldCheck)
```

---

## 📝 代码示例

### 完整的仪表盘页面

```vue
<script setup lang="ts">
import { ref, computed } from 'vue'
import { Monitor, ShieldCheck } from '@element-plus/icons-vue'

const stats = ref({
  vuln_total: 128,
  vuln_today: 12,
  article_total: 256,
  node_online: 8,
})
</script>

<template>
  <el-space direction="vertical" :size="20" fill>
    <!-- 统计卡片 -->
    <el-row :gutter="20">
      <el-col :xs="12" :lg="6">
        <el-card shadow="hover">
          <div class="flex items-center gap-4">
            <el-icon :size="28" color="#409eff">
              <ShieldCheck />
            </el-icon>
            <div>
              <div class="text-3xl font-bold">{{ stats.vuln_total }}</div>
              <div class="text-sm text-gray-400">漏洞总数</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </el-space>
</template>
```

---

## 🐛 常见问题

### 1. 样式不生效

**问题**: Element Plus 组件样式显示不正确

**解决**:
```typescript
// 确保引入了 CSS
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
```

### 2. 图标不显示

**问题**: 使用图标时显示空白

**解决**:
```typescript
// 确保注册了图标
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}
```

### 3. 深色主题不生效

**问题**: 组件显示为浅色

**解决**:
```typescript
// 引入深色主题 CSS
import 'element-plus/theme-chalk/dark/css-vars.css'

// 或在父元素添加 class="dark"
<div class="dark">
  <app />
</div>
```

---

## 📚 参考资料

- [Element Plus 官方文档](https://element-plus.org/zh-CN/)
- [Element Plus 图标](https://element-plus.org/zh-CN/component/icon.html)
- [Vue 3 文档](https://cn.vuejs.org/)
- [TypeScript 文档](https://www.typescriptlang.org/)

---

## 🎯 后续计划

### 短期目标
- [ ] 完成所有页面的 Element Plus 迁移
- [ ] 优化移动端适配
- [ ] 添加国际化支持

### 长期目标
- [ ] 实现主题切换功能
- [ ] 优化性能和包体积
- [ ] 添加单元测试

---

**文档版本**: 2.0.0
**最后更新**: 2026-03-18
**维护者**: SecFlow 开发团队
