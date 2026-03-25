// Simple logger utility that can be disabled in production
const isDev = import.meta.env.DEV

export const logger = {
  log: (...args: unknown[]) => {
    if (isDev) console.log('[LOG]', ...args)
  },
  info: (...args: unknown[]) => {
    if (isDev) console.info('[INFO]', ...args)
  },
  warn: (...args: unknown[]) => {
    if (isDev) console.warn('[WARN]', ...args)
  },
  error: (...args: unknown[]) => {
    // Errors are always logged
    console.error('[ERROR]', ...args)
  },
  debug: (...args: unknown[]) => {
    if (isDev) console.debug('[DEBUG]', ...args)
  },
}

export default logger
