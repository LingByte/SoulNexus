// 缓存系统类型定义

// 缓存策略枚举
export enum CacheStrategy {
  MEMORY = 'memory',           // 内存缓存
  LOCAL_STORAGE = 'localStorage', // 本地存储
  SESSION_STORAGE = 'sessionStorage', // 会话存储
  INDEXED_DB = 'indexedDB',    // IndexedDB
  HTTP = 'http',              // HTTP缓存
  HYBRID = 'hybrid'           // 混合策略
}

// 缓存过期策略
export enum ExpirationStrategy {
  ABSOLUTE = 'absolute',       // 绝对时间过期
  RELATIVE = 'relative',       // 相对时间过期
  SLIDING = 'sliding',         // 滑动过期
  NEVER = 'never'             // 永不过期
}

// 缓存优先级
export enum CachePriority {
  LOW = 1,
  NORMAL = 2,
  HIGH = 3,
  CRITICAL = 4
}

// 缓存项接口
export interface CacheItem<T = any> {
  key: string
  value: T
  timestamp: number
  expiresAt?: number
  accessCount: number
  lastAccessed: number
  size: number
  priority: CachePriority
  tags: string[]
  metadata?: Record<string, any>
}

// 缓存配置接口
export interface CacheConfig {
  strategy: CacheStrategy
  maxSize: number
  maxAge: number
  expirationStrategy: ExpirationStrategy
  priority: CachePriority
  tags: string[]
  persist: boolean
  compress: boolean
  encrypt: boolean
  version: string
  namespace: string
}

// 缓存统计信息
export interface CacheStats {
  totalItems: number
  totalSize: number
  hitCount: number
  missCount: number
  hitRate: number
  evictionCount: number
  lastCleanup: number
  memoryUsage: number
  storageUsage: number
}

// 缓存事件类型
export enum CacheEventType {
  SET = 'set',
  GET = 'get',
  DELETE = 'delete',
  CLEAR = 'clear',
  EXPIRE = 'expire',
  EVICT = 'evict',
  HIT = 'hit',
  MISS = 'miss'
}

// 缓存事件接口
export interface CacheEvent<T = any> {
  type: CacheEventType
  key: string
  value?: T
  timestamp: number
  metadata?: Record<string, any>
}

// 缓存监听器接口
export interface CacheListener<T = any> {
  (event: CacheEvent<T>): void
}

// 缓存序列化器接口
export interface CacheSerializer<T = any> {
  serialize(value: T): string | Promise<string>
  deserialize(data: string): T | Promise<T>
}

// 缓存压缩器接口
export interface CacheCompressor {
  compress(data: string): string | Promise<string>
  decompress(data: string): string | Promise<string>
}

// 缓存加密器接口
export interface CacheEncryptor {
  encrypt(data: string): string
  decrypt(data: string): string
}

// 缓存存储接口
export interface CacheStorage {
  get(key: string): Promise<string | null>
  set(key: string, value: string): Promise<void>
  delete(key: string): Promise<void>
  clear(): Promise<void>
  keys(): Promise<string[]>
  size(): Promise<number>
}

// 缓存管理器接口
export interface CacheManager<T = any> {
  // 基本操作
  get(key: string): Promise<T | null>
  set(key: string, value: T, config?: Partial<CacheConfig>): Promise<void>
  delete(key: string): Promise<boolean>
  clear(): Promise<void>
  has(key: string): Promise<boolean>
  
  // 批量操作
  getMany(keys: string[]): Promise<Record<string, T | null>>
  setMany(items: Record<string, T>, config?: Partial<CacheConfig>): Promise<void>
  deleteMany(keys: string[]): Promise<number>
  
  // 标签操作
  getByTag(tag: string): Promise<Record<string, T>>
  deleteByTag(tag: string): Promise<number>
  
  // 统计和监控
  getStats(): Promise<CacheStats>
  getKeys(): Promise<string[]>
  
  // 生命周期管理
  cleanup(): Promise<void>
  preload(keys: string[]): Promise<void>
  
  // 事件监听
  on(event: CacheEventType, listener: CacheListener<T>): void
  off(event: CacheEventType, listener: CacheListener<T>): void
  
  // 配置管理
  getConfig(): CacheConfig
  updateConfig(config: Partial<CacheConfig>): void
}

// 缓存工厂接口
export interface CacheFactory {
  createManager<T = any>(config: Partial<CacheConfig>): CacheManager<T>
  createStorage(strategy: CacheStrategy): CacheStorage
  createSerializer<T = any>(): CacheSerializer<T>
  createCompressor(): CacheCompressor
  createEncryptor(): CacheEncryptor
}

// 缓存策略配置
export interface CacheStrategyConfig {
  memory: {
    maxItems: number
    maxSize: number
    cleanupInterval: number
  }
  localStorage: {
    maxSize: number
    prefix: string
  }
  sessionStorage: {
    maxSize: number
    prefix: string
  }
  indexedDB: {
    dbName: string
    version: number
    storeName: string
  }
  http: {
    maxAge: number
    staleWhileRevalidate: number
    cacheControl: string
  }
}

// 缓存性能指标
export interface CachePerformance {
  averageGetTime: number
  averageSetTime: number
  averageDeleteTime: number
  memoryEfficiency: number
  storageEfficiency: number
  compressionRatio: number
}

// 缓存错误类型
export enum CacheErrorType {
  STORAGE_FULL = 'storage_full',
  SERIALIZATION_ERROR = 'serialization_error',
  DESERIALIZATION_ERROR = 'deserialization_error',
  COMPRESSION_ERROR = 'compression_error',
  DECOMPRESSION_ERROR = 'decompression_error',
  ENCRYPTION_ERROR = 'encryption_error',
  DECRYPTION_ERROR = 'decryption_error',
  EXPIRED = 'expired',
  NOT_FOUND = 'not_found',
  INVALID_KEY = 'invalid_key',
  INVALID_VALUE = 'invalid_value',
  QUOTA_EXCEEDED = 'quota_exceeded'
}

// 缓存错误接口
export interface CacheError extends Error {
  type: CacheErrorType
  key?: string
  code?: string
  details?: any
}

// 缓存配置选项
export interface CacheOptions {
  // 基本配置
  strategy?: CacheStrategy
  maxSize?: number
  maxAge?: number
  expirationStrategy?: ExpirationStrategy
  priority?: CachePriority
  
  // 高级配置
  tags?: string[]
  persist?: boolean
  compress?: boolean
  encrypt?: boolean
  version?: string
  namespace?: string
  
  // 性能配置
  cleanupInterval?: number
  preloadKeys?: string[]
  enableStats?: boolean
  enableEvents?: boolean
  
  // 存储配置
  storageConfig?: Partial<CacheStrategyConfig>
}

// 缓存查询接口
export interface CacheQuery {
  keys?: string[]
  tags?: string[]
  pattern?: string
  minAge?: number
  maxAge?: number
  minSize?: number
  maxSize?: number
  priority?: CachePriority
  limit?: number
  offset?: number
}

// 缓存结果接口
export interface CacheResult<T = any> {
  items: Record<string, CacheItem<T>>
  total: number
  hasMore: boolean
}

// 缓存预热配置
export interface CacheWarmupConfig {
  keys: string[]
  loader: (key: string) => Promise<any>
  batchSize?: number
  concurrency?: number
  retryCount?: number
  retryDelay?: number
}

// 缓存同步配置
export interface CacheSyncConfig {
  enabled: boolean
  interval: number
  strategy: 'push' | 'pull' | 'bidirectional'
  endpoint?: string
  headers?: Record<string, string>
  transform?: {
    serialize: (data: any) => any
    deserialize: (data: any) => any
  }
}

