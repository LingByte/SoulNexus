import { get, type ApiResponse } from '@/utils/request'
import { getApiBaseURL } from '@/config/apiConfig'
import { readAuthToken } from '@/utils/authToken'

export interface ChartBucket {
  label: string
  value: number
}

export interface TurnTelemetrySummary {
  turnCount?: number
  avgLlmFirstMs?: number
  avgLlmWallMs?: number
  avgTtsMs?: number
  avgPipelineMs?: number
  avgRecallMs?: number
  ragHitTurns?: number
  ragMissTurns?: number
  ragHitRate?: number
  interruptCount?: number
}

export interface TransferTrendPoint {
  day: string
  transferRate: number
  transferAnswerRate: number
  aiToHumanCount: number
  connectedCount: number
}

export interface CallerAttributeRow {
  id: number
  callId: string
  fromNumber: string
  callerProvince: string
  callerCity: string
  callerCarrier: string
  direction: string
  durationSec: number
  turnCount: number
  endStatus: string
  endedAt?: string
}

export interface CallAnalyticsSummary {
  callCount?: number
  connectedRate?: number
  totalMinutes?: number
  avgMinutesPerCall?: number
  transferRate?: number
  pureAIRate?: number
  quoteRate?: number
  hitRate?: number
  kbAttachRate?: number
  unansweredNew?: number
  unansweredOpen?: number
  avgTurns?: number
  avgDurationSec?: number
  avgLlmFirstMs?: number
  avgPipelineMs?: number
  oneTimeResolutionRate?: number
  callsWithKb?: number
  callsQuoted?: number
  transferAnswerRate?: number
  durationBuckets?: ChartBucket[]
  turnBuckets?: ChartBucket[]
  provinceBuckets?: ChartBucket[]
  callTrend?: { day: string; callCount: number; totalMinutes: number; transferRate?: number }[]
}

export interface CallAnalyticsDashboard {
  rangeStart: string
  rangeEnd: string
  summary: CallAnalyticsSummary
  callTrend: { day: string; callCount: number; totalMinutes: number; transferRate?: number }[]
  durationBuckets: ChartBucket[]
  turnBuckets: ChartBucket[]
  provinceBuckets: ChartBucket[]
  endStatusBuckets?: ChartBucket[]
  turnTelemetry?: TurnTelemetrySummary
  transferOutcomeBuckets?: ChartBucket[]
  transferReasonBuckets?: ChartBucket[]
  transferTrend?: TransferTrendPoint[]
}

export interface AIReport {
  id: string
  reportType: 'daily' | 'weekly'
  periodStart: string
  periodEnd: string
  title: string
  summary?: Record<string, unknown>
  bodyHtml?: string
  pushedInbox: boolean
  pushedEmail: boolean
  pushedWebhook?: boolean
  pushedIm?: boolean
  createdAt: string
}

export async function listAIReports(params?: { type?: string; page?: number; size?: number }) {
  const q = new URLSearchParams()
  if (params?.type) q.set('type', params.type)
  if (params?.page) q.set('page', String(params.page))
  if (params?.size) q.set('size', String(params.size))
  const suffix = q.toString() ? `?${q.toString()}` : ''
  return get<{ items: AIReport[]; total: number; page: number; size: number }>(`/reports/ai${suffix}`)
}

export async function getAIReport(id: string) {
  return get<AIReport>(`/reports/ai/${id}`)
}

export async function getAICallAnalytics(days = 14) {
  return get<CallAnalyticsDashboard>(`/reports/ai/analytics?days=${days}`)
}

export async function getAICalloutAnalysis(days = 14) {
  return get<CalloutAnalysisData>(`/reports/ai/callout-analysis?days=${days}`)
}

export interface CalloutAnalysisData {
  summary: {
    totalAttempts: number
    totalAnswered: number
    totalUnanswered: number
    answeredPercent: number
    highIntentCount: number
    mediumIntentCount: number
    lowIntentCount: number
  }
  byCaller: Array<{
    callerNumber: string
    totalAttempts: number
    totalAnswered: number
    answeredPercent: number
    highIntentCount: number
    mediumIntentCount: number
    lowIntentCount: number
  }>
  trend: Array<{
    day: string
    attempts: number
    answered: number
    answeredPercent: number
  }>
}

export async function getAICallerAttributes(params: { days?: number; province?: string; page?: number; size?: number }) {
  const q = new URLSearchParams()
  if (params.days) q.set('days', String(params.days))
  if (params.province) q.set('province', params.province)
  if (params.page) q.set('page', String(params.page))
  if (params.size) q.set('size', String(params.size))
  return get<{ items: CallerAttributeRow[]; total: number; page: number; size: number }>(
    `/reports/ai/analytics/caller-attributes?${q.toString()}`,
  )
}

export function callerAttributesExportUrl(days = 14, province?: string) {
  const q = new URLSearchParams({ days: String(days) })
  if (province) q.set('province', province)
  return `/api/reports/ai/analytics/caller-export?${q.toString()}`
}

export async function downloadCallerAttributesExport(days = 14, province?: string): Promise<void> {
  const base = getApiBaseURL().replace(/\/$/, '')
  const token = readAuthToken()
  const q = new URLSearchParams({ days: String(days), _t: String(Date.now()) })
  if (province) q.set('province', province)
  const url = `${base}/reports/ai/analytics/caller-export?${q.toString()}`
  const res = await fetch(url, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!res.ok) {
    throw new Error(`export failed (${res.status})`)
  }
  const blob = await res.blob()
  const cd = res.headers.get('Content-Disposition')
  const m = cd ? /filename="([^"]+)"/i.exec(cd) : null
  const filename = m?.[1]?.trim() || `caller-attributes-${days}d.xlsx`
  const objectUrl = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = objectUrl
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(objectUrl)
}

export type { ApiResponse }
