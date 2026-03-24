import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Node } from '@/types'
import { nodeApi } from '@/api/node'

export const useNodeStore = defineStore('node', () => {
  const nodes = ref<Node[]>([])
  const loading = ref(false)

  async function fetchNodes() {
    loading.value = true
    try {
      const result = await nodeApi.list()
      nodes.value = result || []
    } catch (error) {
      console.error('Failed to fetch nodes:', error)
      nodes.value = []
    } finally {
      loading.value = false
    }
  }

  const onlineCount = () => (nodes.value || []).filter(n => n.online).length

  return { nodes, loading, fetchNodes, onlineCount }
})
