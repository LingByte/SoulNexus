import { get, type ApiResponse } from '@/utils/request'

export interface OverviewDailyPoint {
  day: string
  callCount: number
  totalMinutes: number
  avgMinutesPerCall: number
  transferRate: number
  knowledgeRecallTotal?: number
  knowledgeRecallHitRate?: number
  knowledgeUnrecognizedRate?: number
}

export interface OverviewProvinceStat {
  province: string
  callCount: number
  totalMinutes: number
}

export interface OverviewCityStat {
  province: string
  city: string
  callCount: number
  totalMinutes: number
}

export type OverviewRegionCitiesData = {
  rangeStart: string
  rangeEnd: string
  province: string
  cities: OverviewCityStat[]
}

export interface OverviewDirectionStat {
  direction: string
  callCount: number
}

export interface OverviewTransferStat {
  pureAI: number
  aiToHuman: number
  pureAIRate: number
  aiToHumanRate: number
}

export interface OverviewTransferPoolStat {
  poolId: number
  poolName: string
  attemptCount: number
  answeredCount: number
  unansweredCount: number
  answerRate: number
}

export interface OverviewRangeSummary {
  callCount: number
  connectedCount: number
  totalMinutes: number
  avgMinutesPerCall: number
  transferRate: number
  knowledgeRecallTotal?: number
  knowledgeRecallHitRate?: number
  knowledgeUnrecognizedRate?: number
}

export interface OverviewAccountMeta {
  tenantCreatedAt?: string
  usageDays: number
  allTimeCallCount: number
  allTimeBilledMinutes: number
  tenantMemberCount?: number
  billingMode?: 'prepaid' | 'postpaid'
  billingUnlimited?: boolean
  prepaidMinutesRemaining?: number
  remainingMinutesDisplay?: string
  billingRatePerMinute?: number
}

export interface OverviewDashboardData {
  rangeStart: string
  rangeEnd: string
  meta: OverviewAccountMeta
  summary: OverviewRangeSummary
  days: OverviewDailyPoint[]
  provinceDistribution: OverviewProvinceStat[]
  directionDistribution: OverviewDirectionStat[]
  transferStat: OverviewTransferStat
  transferPoolStats: OverviewTransferPoolStat[]
}

export type OverviewDashboardOptions = {
  from?: string
  to?: string
  days?: number
}

export async function getOverviewDashboard(
  opts?: OverviewDashboardOptions,
): Promise<ApiResponse<OverviewDashboardData>> {
  const q = new URLSearchParams()
  if (opts?.from) q.set('from', opts.from)
  if (opts?.to) q.set('to', opts.to)
  if (!opts?.from && !opts?.to) {
    q.set('days', String(opts?.days ?? 14))
  }
  return get(`/overview/stats?${q.toString()}`)
}

export function defaultOverviewRange(days = 14): { from: string; to: string } {
  const to = new Date()
  const from = new Date(to)
  from.setDate(to.getDate() - (days - 1))
  const fmt = (d: Date) => {
    const pad = (n: number) => `${n}`.padStart(2, '0')
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
  }
  return { from: fmt(from), to: fmt(to) }
}

/** 号段属地 · 地市分布（可按省份过滤） */
export async function getOverviewRegionCities(opts: {
  from: string
  to: string
  province?: string
}): Promise<ApiResponse<OverviewRegionCitiesData>> {
  const q = new URLSearchParams()
  q.set('from', opts.from)
  q.set('to', opts.to)
  if (opts.province) q.set('province', opts.province)
  return get(`/overview/region-cities?${q.toString()}`)
}
