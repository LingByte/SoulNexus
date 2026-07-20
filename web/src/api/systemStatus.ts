import { get, type ApiResponse } from '@/utils/request'

export type PreflightLevel = 'ok' | 'warn' | 'error'

export interface PreflightCheck {
  id: string
  category: string
  level: PreflightLevel
  message: string
  detail?: string
  latencyMs?: number
}

export interface PreflightSnapshot {
  checkedAt?: string
  checks: PreflightCheck[]
}

export interface RuntimeMemorySnapshot {
  alloc: number
  totalAlloc: number
  sys: number
  lookups: number
  mallocs: number
  frees: number
  heapAlloc: number
  heapSys: number
  heapIdle: number
  heapInuse: number
  heapReleased: number
  heapObjects: number
  stackInuse: number
  stackSys: number
  mSpanInuse: number
  mSpanSys: number
  mCacheInuse: number
  mCacheSys: number
  buckHashSys: number
  gcSys: number
  otherSys: number
  nextGC: number
  lastGC: number
  pauseTotalNs: number
  numGC: number
  numForcedGC: number
  gcCpuFraction: number
  enableGC: boolean
}

export interface GCSnapshot {
  numGC: number
  numForcedGC: number
  pauseTotalNs: number
  pauseTotalMs: number
  recentPauseAvgMs: number
  recentPauseMaxMs: number
  recentPauseSamples: number
  gcCpuFraction: number
  nextGC: number
  lastGCUnixNano: number
  gcPerMinute: number
}

export interface GoroutineSnapshot {
  numGoroutine: number
  numGoroutineMax: number
  numThread: number
  numCgoCall: number
  byState?: Record<string, number>
  leakSuspect: boolean
}

export interface HostSnapshot {
  cpu?: {
    usagePercent?: number
    userPercent?: number
    systemPercent?: number
    idlePercent?: number
    iowaitPercent?: number
    numCpu?: number
  }
  load?: { load1?: number; load5?: number; load15?: number }
  memory?: {
    total?: number
    available?: number
    used?: number
    usedPercent?: number
    swapTotal?: number
    swapUsed?: number
    swapPercent?: number
  }
  disks?: Array<{
    path: string
    total: number
    used: number
    free: number
    usedPercent: number
    inodesTotal?: number
    inodesFree?: number
    inodesUsed?: number
  }>
  network?: {
    bytesSent?: number
    bytesRecv?: number
    packetsSent?: number
    packetsRecv?: number
    dropIn?: number
    dropOut?: number
  }
  process?: {
    pid?: number
    rss?: number
    vms?: number
    cpuPercent?: number
    openFds?: number
    maxFds?: number
    numThreads?: number
  }
}

export interface OpsCounters {
  panicTotal?: number
  http5xxTotal?: number
  http4xxTotal?: number
  jwtVerifyFailTotal?: number
  dbReconnectTotal?: number
  preflightFailTotal?: number
  dependencyFailTotal?: number
  fileIoErrorTotal?: number
  sslCertLoadFailTotal?: number
}

export interface Quantiles {
  p50?: number
  p90?: number
  p99?: number
  n?: number
}

export interface BusinessMetrics {
  http?: {
    requestsTotal?: number
    errors4xx?: number
    errors5xx?: number
    latencyMs?: Quantiles
  }
  database?: {
    openConnections?: number
    inUse?: number
    idle?: number
    waitCount?: number
    waitDurationMs?: number
    maxOpen?: number
  }
  knowledgeWorker?: {
    queued?: number
    running?: number
    unfinished?: number
  }
  voice?: PrometheusSnapshot
}

export interface PrometheusSnapshot {
  counters?: Record<string, number>
  gauges?: Record<string, number>
  summaries?: Record<string, Record<string, number>>
}

export interface SystemStatusPayload {
  uptimeSeconds: number
  startedAt?: string
  goroutines: number
  go: string
  server?: {
    name?: string
    addr?: string
    mode?: string
    sslEnabled?: boolean
    apiPrefix?: string
  }
  preflight?: PreflightSnapshot
  dependencies?: PreflightCheck[]
  runtimeMemory?: RuntimeMemorySnapshot
  gc?: GCSnapshot
  goroutineDetail?: GoroutineSnapshot
  host?: HostSnapshot
  ops?: OpsCounters
  business?: BusinessMetrics
  prometheus?: PrometheusSnapshot
  resource: { cpuUsage: number; memoryUsage: number; diskUsage: number }
  disk: { total: number; free: number; used: number; usedPercent: number }
  performanceMonitor: {
    enabled: boolean
    cpuThreshold: number
    memoryThreshold: number
    diskThreshold: number
  }
  diskCache: Record<string, number>
}

export async function fetchSystemStatus(refresh = false): Promise<ApiResponse<SystemStatusPayload>> {
  const q = refresh ? '?refresh=1' : ''
  return get(`/system/status${q}`)
}
