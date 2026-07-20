import { del, get, post, put, type ApiResponse } from '@/utils/request'

export interface KnowledgeChunk {
  id: string
  docId: string
  chunkIndex: number
  recordId: string
  title: string
  content: string
  sourceType: string
  status: string
}

export interface KnowledgeUnansweredQuestion {
  id: string
  question: string
  occurrenceCount: number
  status: string
  callId?: string
  createdAt: string
}

export interface KnowledgeTypicalQuestion {
  id: string
  question: string
  totalCount: number
  quotedCount: number
}

export interface QuoteRateOverview {
  totalCalls: number
  callsWithKnowledge: number
  callsQuoted: number
  totalRetrievals: number
  retrievalsWithHit: number
  quoteRate: number
  hitRate: number
  kbAttachRate: number
}

export interface KnowledgeSyncSource {
  id: string
  name: string
  sourceType: string
  sourceUrl: string
  intervalMinutes: number
  status: string
  lastSyncAt?: string
  lastSyncError?: string
}

export async function listKnowledgeChunks(nsId: string, docId?: string) {
  const q = docId ? `?docId=${docId}` : ''
  return get<KnowledgeChunk[]>(`/knowledge-namespaces/${nsId}/chunks${q}`)
}

export async function createKnowledgeChunk(nsId: string, body: { docId?: string; title?: string; content: string }) {
  return post<KnowledgeChunk>(`/knowledge-namespaces/${nsId}/chunks`, body)
}

export async function updateKnowledgeChunk(nsId: string, chunkId: string, body: { title?: string; content: string }) {
  return put<KnowledgeChunk>(`/knowledge-namespaces/${nsId}/chunks/${chunkId}`, body)
}

export async function deleteKnowledgeChunk(nsId: string, chunkId: string) {
  return del<null>(`/knowledge-namespaces/${nsId}/chunks/${chunkId}`)
}

export function exportKnowledgeChunksUrl(nsId: string) {
  return `/knowledge-namespaces/${nsId}/chunks/export`
}

/** Download chunks Excel with auth (do not use window.open on /api). */
export async function downloadKnowledgeChunksExport(nsId: string): Promise<void> {
  const { getApiBaseURL } = await import('@/config/apiConfig')
  const { readAuthToken } = await import('@/utils/authToken')
  const base = getApiBaseURL().replace(/\/$/, '')
  const token = readAuthToken()
  const res = await fetch(`${base}/knowledge-namespaces/${nsId}/chunks/export?_t=${Date.now()}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!res.ok) throw new Error('export failed')
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `knowledge-chunks-${nsId}.xlsx`
  a.click()
  URL.revokeObjectURL(url)
}

export async function listUnansweredQuestions(nsId: string, page = 1, size = 20, status = 'open') {
  return get<{ items: KnowledgeUnansweredQuestion[]; total: number }>(
    `/knowledge-namespaces/${nsId}/unanswered-questions?page=${page}&size=${size}&status=${status}`,
  )
}

export async function countUnansweredQuestions(nsId: string) {
  return get<{ open: number; resolved: number }>(`/knowledge-namespaces/${nsId}/unanswered-questions/count`)
}

export async function resolveUnansweredQuestion(nsId: string, questionId: string, body: { title?: string; content: string }) {
  return post<{ queued?: boolean }>(`/knowledge-namespaces/${nsId}/unanswered-questions/${questionId}/resolve`, body, {
    timeout: 60_000,
  })
}

export async function draftUnansweredAnswer(nsId: string, questionId: string) {
  return post<{ title: string; content: string; source?: string }>(
    `/knowledge-namespaces/${nsId}/unanswered-questions/${questionId}/draft-answer`,
    {},
    { timeout: 90_000 },
  )
}

export async function deleteUnansweredQuestion(nsId: string, questionId: string) {
  return del<null>(`/knowledge-namespaces/${nsId}/unanswered-questions/${questionId}`)
}

export async function listHFQuestions(nsId: string, page = 1, size = 20) {
  return get<{ items: KnowledgeTypicalQuestion[]; total: number }>(
    `/knowledge-namespaces/${nsId}/hf-questions?page=${page}&size=${size}`,
  )
}

export interface HFDailyStat {
  statDate: string
  count: number
  quotedCount: number
}

export interface HFAnsweredRecord {
  id: string
  question: string
  callId?: string
  knowledgeQuoted: boolean
  retrievalHitCount: number
  createdAt: string
}

export async function getHFDailySummary(nsId: string, from?: string, to?: string) {
  const q = new URLSearchParams()
  if (from) q.set('from', from)
  if (to) q.set('to', to)
  const qs = q.toString()
  return get<{ items: HFDailyStat[] }>(`/knowledge-namespaces/${nsId}/hf-questions/daily-summary${qs ? `?${qs}` : ''}`)
}

export async function getHFQuestionStats(nsId: string, typicalId: string, from?: string, to?: string) {
  const q = new URLSearchParams()
  if (from) q.set('from', from)
  if (to) q.set('to', to)
  const qs = q.toString()
  return get<{ items: HFDailyStat[] }>(`/knowledge-namespaces/${nsId}/hf-questions/${typicalId}/stats${qs ? `?${qs}` : ''}`)
}

export async function listHFQuestionAnswers(nsId: string, typicalId: string, day?: string, page = 1, size = 20) {
  const q = new URLSearchParams({ page: String(page), size: String(size) })
  if (day) q.set('day', day)
  return get<{ items: HFAnsweredRecord[]; total: number }>(
    `/knowledge-namespaces/${nsId}/hf-questions/${typicalId}/answers?${q}`,
  )
}

export async function getQuoteRateReport(nsId: string, body: { from?: string; to?: string }) {
  return post<{ overview: QuoteRateOverview; from: string; to: string }>(
    `/knowledge-namespaces/${nsId}/analytics/quote-rate`,
    body,
  )
}

export async function listSyncSources(nsId: string) {
  return get<KnowledgeSyncSource[]>(`/knowledge-namespaces/${nsId}/sync-sources`)
}

export async function createSyncSource(nsId: string, body: {
  name: string
  sourceType?: string
  sourceUrl: string
  intervalMinutes?: number
  chunkConfig?: Record<string, unknown>
}) {
  return post<KnowledgeSyncSource>(`/knowledge-namespaces/${nsId}/sync-sources`, body)
}

export async function updateSyncSource(nsId: string, sourceId: string, body: {
  name: string
  sourceType?: string
  sourceUrl: string
  intervalMinutes?: number
  chunkConfig?: Record<string, unknown>
}) {
  return put<KnowledgeSyncSource>(`/knowledge-namespaces/${nsId}/sync-sources/${sourceId}`, body)
}

export async function deleteSyncSource(nsId: string, sourceId: string) {
  return del<null>(`/knowledge-namespaces/${nsId}/sync-sources/${sourceId}`)
}

export async function triggerSyncSource(nsId: string, sourceId: string) {
  return post<{ queued: boolean }>(`/knowledge-namespaces/${nsId}/sync-sources/${sourceId}/trigger`, {})
}

export async function getWorkerStats(nsId: string) {
  return get<KnowledgeWorkerSnapshot>(`/knowledge-namespaces/${nsId}/worker/stats`)
}

export interface KnowledgeWorkerJob {
  taskId: string
  kind: 'ingest' | 'purge' | 'sync'
  docId?: number
  syncSourceId?: number
  taskStatus: string
  queueAhead: number
  queuedTotal: number
  runningWorkers: number
  unfinishedEstimate: number
  submittedAt: string
}

export interface KnowledgeWorkerSnapshot {
  queued: number
  running: number
  unfinished: number
  jobs: KnowledgeWorkerJob[]
}

export interface KnowledgeDocumentProgress {
  docId: number
  documentStatus: string
  inWorker: boolean
  taskId?: string
  taskStatus?: string
  queueAhead?: number
  queuedTotal?: number
  runningWorkers?: number
  unfinishedEstimate?: number
  submittedAt?: string
}

export async function getKnowledgeDocumentProgress(nsId: string, docId: string) {
  return get<KnowledgeDocumentProgress>(`/knowledge-namespaces/${nsId}/documents/${docId}/progress`)
}

const evalStartTimeoutMs = 15_000

export interface EvalJobStatus {
  jobId: string
  status: 'pending' | 'running' | 'done' | 'failed'
  result?: Record<string, unknown>
  error?: string
  createdAt?: string
  finishedAt?: string
}

export async function runKnowledgeEval(nsId: string, body: { datasetId: string; strategy?: string; topK?: number; minScore?: number }) {
  return post<{ jobId: string; status: string }>(`/knowledge-namespaces/${nsId}/eval/run`, body, { timeout: evalStartTimeoutMs })
}

export async function compareKnowledgeEval(nsId: string, body: { datasetId: string; topK?: number; minScore?: number }) {
  return post<{ jobId: string; status: string }>(`/knowledge-namespaces/${nsId}/eval/compare`, body, { timeout: evalStartTimeoutMs })
}

export async function getKnowledgeEvalJob(nsId: string, jobId: string) {
  return get<EvalJobStatus>(`/knowledge-namespaces/${nsId}/eval/jobs/${jobId}`, { timeout: evalStartTimeoutMs })
}

export interface KnowledgeEvalDataset {
  id: string
  name: string
  sampleCount: number
  createdAt: string
}

export async function listEvalDatasets(nsId: string) {
  return get<KnowledgeEvalDataset[]>(`/knowledge-namespaces/${nsId}/eval/datasets`)
}

export async function createEvalDataset(nsId: string, body: {
  name: string
  items?: { query: string; relevantIds: string[] }[]
  samples?: string
}) {
  return post<KnowledgeEvalDataset>(`/knowledge-namespaces/${nsId}/eval/datasets`, body)
}

export async function deleteEvalDataset(nsId: string, datasetId: string) {
  return del<null>(`/knowledge-namespaces/${nsId}/eval/datasets/${datasetId}`)
}

export type { ApiResponse }
