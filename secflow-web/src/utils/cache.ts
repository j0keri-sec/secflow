// Simple cache utility with TTL support
interface CacheEntry<T> {
  data: T
  timestamp: number
  ttl: number // milliseconds
}

export class Cache {
  private store = new Map<string, CacheEntry<unknown>>()
  private cleanupInterval: ReturnType<typeof setInterval> | null = null

  constructor(private defaultTTL = 5 * 60 * 1000) { // 5 minutes default
    // Start cleanup interval
    this.cleanupInterval = setInterval(() => {
      this.cleanup()
    }, 60000) // Clean up every minute
  }

  set<T>(key: string, data: T, ttl?: number): void {
    this.store.set(key, {
      data,
      timestamp: Date.now(),
      ttl: ttl ?? this.defaultTTL,
    })
  }

  get<T>(key: string): T | null {
    const entry = this.store.get(key) as CacheEntry<T> | undefined
    if (!entry) return null

    // Check if expired
    if (Date.now() - entry.timestamp > entry.ttl) {
      this.store.delete(key)
      return null
    }

    return entry.data
  }

  has(key: string): boolean {
    return this.get(key) !== null
  }

  delete(key: string): void {
    this.store.delete(key)
  }

  clear(): void {
    this.store.clear()
  }

  // Clean up expired entries
  private cleanup(): void {
    const now = Date.now()
    for (const [key, entry] of this.store.entries()) {
      if (now - entry.timestamp > entry.ttl) {
        this.store.delete(key)
      }
    }
  }

  // Destroy the cache and stop cleanup interval
  destroy(): void {
    if (this.cleanupInterval) {
      clearInterval(this.cleanupInterval)
      this.cleanupInterval = null
    }
    this.store.clear()
  }

  // Get cache stats
  getStats() {
    return {
      size: this.store.size,
      keys: Array.from(this.store.keys()),
    }
  }
}

// Global cache instance
export const globalCache = new Cache()

// LocalStorage cache with TTL support
export class LocalStorageCache {
  private prefix: string

  constructor(prefix = 'secflow_cache') {
    this.prefix = prefix
  }

  private getKey(key: string): string {
    return `${this.prefix}:${key}`
  }

  set<T>(key: string, data: T, ttlSeconds = 300): void {
    const entry = {
      data,
      expires: Date.now() + ttlSeconds * 1000,
    }
    try {
      localStorage.setItem(this.getKey(key), JSON.stringify(entry))
    } catch (e) {
      console.warn('LocalStorageCache set failed:', e)
    }
  }

  get<T>(key: string): T | null {
    try {
      const raw = localStorage.getItem(this.getKey(key))
      if (!raw) return null

      const entry = JSON.parse(raw) as { data: T; expires: number }
      if (Date.now() > entry.expires) {
        this.delete(key)
        return null
      }
      return entry.data
    } catch {
      return null
    }
  }

  has(key: string): boolean {
    return this.get(key) !== null
  }

  delete(key: string): void {
    localStorage.removeItem(this.getKey(key))
  }

  clear(): void {
    const prefix = this.prefix
    for (let i = localStorage.length - 1; i >= 0; i--) {
      const key = localStorage.key(i)
      if (key?.startsWith(`${prefix}:`)) {
        localStorage.removeItem(key)
      }
    }
  }
}

export const localCache = new LocalStorageCache()
