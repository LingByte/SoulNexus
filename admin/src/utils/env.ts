// 环境变量工具函数

/**
 * 安全地获取环境变量
 * 支持 Vite 和 Node.js 环境
 */
export const getEnvVar = (key: string, defaultValue: string = ''): string => {
  // 优先使用 Vite 的环境变量
  if (typeof window !== 'undefined' && (window as any).import?.meta?.env) {
    return (window as any).import.meta.env[key] || defaultValue
  }
  
  // 回退到 Node.js 环境变量
  if (typeof process !== 'undefined' && process.env) {
    return process.env[key] || defaultValue
  }
  
  return defaultValue
}

/**
 * 获取布尔类型的环境变量
 */
export const getEnvBoolean = (key: string, defaultValue: boolean = false): boolean => {
  const value = getEnvVar(key, defaultValue.toString())
  return value === 'true' || value === '1'
}

/**
 * 获取数字类型的环境变量
 */
export const getEnvNumber = (key: string, defaultValue: number = 0): number => {
  const value = getEnvVar(key, defaultValue.toString())
  const parsed = parseInt(value, 10)
  return isNaN(parsed) ? defaultValue : parsed
}

/**
 * 获取浮点数类型的环境变量
 */
export const getEnvFloat = (key: string, defaultValue: number = 0): number => {
  const value = getEnvVar(key, defaultValue.toString())
  const parsed = parseFloat(value)
  return isNaN(parsed) ? defaultValue : parsed
}

/**
 * 获取数组类型的环境变量（逗号分隔）
 */
export const getEnvArray = (key: string, defaultValue: string[] = []): string[] => {
  const value = getEnvVar(key, '')
  if (!value) return defaultValue
  return value.split(',').map(item => item.trim()).filter(Boolean)
}

/**
 * 获取当前环境
 */
export const getCurrentEnv = (): string => {
  return getEnvVar('NODE_ENV', 'development')
}

/**
 * 检查是否为开发环境
 */
export const isDevelopment = (): boolean => {
  return getCurrentEnv() === 'development'
}

/**
 * 检查是否为生产环境
 */
export const isProduction = (): boolean => {
  return getCurrentEnv() === 'production'
}

/**
 * 检查是否为测试环境
 */
export const isTest = (): boolean => {
  return getCurrentEnv() === 'test'
}

/**
 * 获取所有环境变量（用于调试）
 */
export const getAllEnvVars = (): Record<string, string> => {
  const envVars: Record<string, string> = {}
  
  // 获取 Vite 环境变量
  if (typeof window !== 'undefined' && (window as any).import?.meta?.env) {
    const viteEnv = (window as any).import.meta.env
    Object.keys(viteEnv).forEach(key => {
      if (key.startsWith('VITE_')) {
        envVars[key] = String(viteEnv[key])
      }
    })
  }
  
  // 获取 Node.js 环境变量
  if (typeof process !== 'undefined' && process.env) {
    Object.keys(process.env).forEach(key => {
      if (key.startsWith('VITE_')) {
        envVars[key] = String(process.env[key])
      }
    })
  }
  
  return envVars
}

/**
 * 环境变量配置接口
 */
export interface EnvConfig {
  [key: string]: string | number | boolean | string[]
}

/**
 * 创建环境变量配置对象
 */
export const createEnvConfig = (config: Record<string, { key: string; type: 'string' | 'number' | 'boolean' | 'array'; default: any }>): EnvConfig => {
  const result: EnvConfig = {}
  
  Object.entries(config).forEach(([name, { key, type, default: defaultValue }]) => {
    switch (type) {
      case 'string':
        result[name] = getEnvVar(key, defaultValue)
        break
      case 'number':
        result[name] = getEnvNumber(key, defaultValue)
        break
      case 'boolean':
        result[name] = getEnvBoolean(key, defaultValue)
        break
      case 'array':
        result[name] = getEnvArray(key, defaultValue)
        break
    }
  })
  
  return result
}

/**
 * 验证必需的环境变量
 */
export const validateRequiredEnvVars = (requiredVars: string[]): { valid: boolean; missing: string[] } => {
  const missing: string[] = []
  
  requiredVars.forEach(key => {
    const value = getEnvVar(key)
    if (!value) {
      missing.push(key)
    }
  })
  
  return {
    valid: missing.length === 0,
    missing
  }
}

/**
 * 环境变量调试工具
 */
export const debugEnvVars = (): void => {
  if (isDevelopment()) {
    console.group('🔧 Environment Variables')
    console.log('Current Environment:', getCurrentEnv())
    console.log('All VITE_ Variables:', getAllEnvVars())
    console.groupEnd()
  }
}
