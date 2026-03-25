import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import logger from '@/utils/logger'

export type WebSocketStatus = 'connecting' | 'connected' | 'disconnected' | 'reconnecting'

interface WebSocketMessage {
  type: string
  payload?: unknown
}

export const useWebSocketStore = defineStore('websocket', () => {
  // State
  const status = ref<WebSocketStatus>('disconnected')
  const socket = ref<WebSocket | null>(null)
  const reconnectAttempts = ref(0)
  const lastMessage = ref<WebSocketMessage | null>(null)
  
  // Listeners for different message types
  const listeners = new Map<string, Set<(payload: unknown) => void>>()
  
  // Config
  const maxReconnectAttempts = 5
  const baseReconnectDelay = 1000 // 1 second
  const maxReconnectDelay = 30000 // 30 seconds
  
  let reconnectTimeout: ReturnType<typeof setTimeout> | null = null
  let pingInterval: ReturnType<typeof setInterval> | null = null
  
  // Getters
  const isConnected = computed(() => status.value === 'connected')
  const isReconnecting = computed(() => status.value === 'reconnecting')
  
  // Actions
  function connect(url: string) {
    if (socket.value?.readyState === WebSocket.OPEN) {
      logger.warn('WebSocket already connected')
      return
    }
    
    status.value = 'connecting'
    logger.info('WebSocket connecting...', { url })
    
    try {
      socket.value = new WebSocket(url)
      
      socket.value.onopen = () => {
        logger.info('WebSocket connected')
        status.value = 'connected'
        reconnectAttempts.value = 0
        
        // Start ping interval to keep connection alive
        startPing()
      }
      
      socket.value.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data)
          lastMessage.value = message
          
          // Notify listeners
          const typeListeners = listeners.get(message.type)
          if (typeListeners) {
            typeListeners.forEach((callback) => {
              try {
                callback(message.payload)
              } catch (e) {
                logger.error('WebSocket listener error', e)
              }
            })
          }
          
          // Notify 'all' listeners
          const allListeners = listeners.get('*')
          if (allListeners) {
            allListeners.forEach((callback) => {
              try {
                callback(message)
              } catch (e) {
                logger.error('WebSocket listener error', e)
              }
            })
          }
        } catch (e) {
          logger.error('Failed to parse WebSocket message', e)
        }
      }
      
      socket.value.onclose = (event) => {
        logger.info('WebSocket closed', { code: event.code, reason: event.reason })
        status.value = 'disconnected'
        stopPing()
        
        // Auto reconnect if not a clean close
        if (event.code !== 1000 && event.code !== 1001) {
          scheduleReconnect(url)
        }
      }
      
      socket.value.onerror = (error) => {
        logger.error('WebSocket error', error)
        status.value = 'disconnected'
      }
    } catch (e) {
      logger.error('Failed to create WebSocket', e)
      status.value = 'disconnected'
      scheduleReconnect(url)
    }
  }
  
  function disconnect() {
    stopPing()
    clearReconnect()
    
    if (socket.value) {
      socket.value.close(1000, 'Client disconnect')
      socket.value = null
    }
    
    status.value = 'disconnected'
    reconnectAttempts.value = 0
  }
  
  function send(type: string, payload?: unknown) {
    if (socket.value?.readyState !== WebSocket.OPEN) {
      logger.warn('WebSocket not connected, cannot send message')
      return false
    }
    
    try {
      socket.value.send(JSON.stringify({ type, payload }))
      return true
    } catch (e) {
      logger.error('Failed to send WebSocket message', e)
      return false
    }
  }
  
  function on(type: string, callback: (payload: unknown) => void) {
    if (!listeners.has(type)) {
      listeners.set(type, new Set())
    }
    listeners.get(type)!.add(callback)
    
    // Return unsubscribe function
    return () => {
      listeners.get(type)?.delete(callback)
    }
  }
  
  function off(type: string, callback: (payload: unknown) => void) {
    listeners.get(type)?.delete(callback)
  }
  
  function scheduleReconnect(url: string) {
    if (reconnectAttempts.value >= maxReconnectAttempts) {
      logger.warn('Max reconnection attempts reached')
      return
    }
    
    // Exponential backoff with jitter
    const delay = Math.min(
      baseReconnectDelay * Math.pow(2, reconnectAttempts.value) + Math.random() * 1000,
      maxReconnectDelay
    )
    
    status.value = 'reconnecting'
    reconnectAttempts.value++
    
    logger.info(`WebSocket reconnecting in ${Math.round(delay)}ms (attempt ${reconnectAttempts.value})`)
    
    reconnectTimeout = setTimeout(() => {
      connect(url)
    }, delay)
  }
  
  function clearReconnect() {
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout)
      reconnectTimeout = null
    }
    reconnectAttempts.value = 0
  }
  
  function startPing() {
    stopPing()
    pingInterval = setInterval(() => {
      if (socket.value?.readyState === WebSocket.OPEN) {
        socket.value.send(JSON.stringify({ type: 'ping' }))
      }
    }, 30000) // Ping every 30 seconds
  }
  
  function stopPing() {
    if (pingInterval) {
      clearInterval(pingInterval)
      pingInterval = null
    }
  }
  
  return {
    // State
    status,
    reconnectAttempts,
    lastMessage,
    
    // Getters
    isConnected,
    isReconnecting,
    
    // Actions
    connect,
    disconnect,
    send,
    on,
    off,
  }
})
