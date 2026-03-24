<script setup lang="ts">
import type { SeverityLevel } from '@/types'

const props = defineProps<{ severity: SeverityLevel }>()

const classMap: Record<SeverityLevel, string> = {
  '严重': 'badge-critical',
  '高危': 'badge-high',
  '中危': 'badge-medium',
  '低危': 'badge-low',
}
</script>

<template>
  <span :class="classMap[props.severity]">{{ props.severity }}</span>
</template>
