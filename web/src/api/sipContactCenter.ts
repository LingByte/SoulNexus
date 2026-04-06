import { get, post, put, del, ApiResponse } from '@/utils/request'
import { getApiBaseURL } from '@/config/apiConfig'

export interface SIPUserRow {
  id: number
  username: string
  domain: string
  contactUri?: string
  remoteIp?: string
  remotePort?: number
  online?: boolean
  expiresAt?: string
  lastSeenAt?: string
  userAgent?: string
  createdAt?: string
  updatedAt?: string
}

/** One ASR→LLM turn stored in `sip_calls.turns` JSON. */
export interface SIPCallDialogTurn {
  asrText?: string
  llmText?: string
  asrProvider?: string
  ttsProvider?: string
  llmModel?: string
  at?: string
}

export interface SIPCallRow {
  id: number
  callId: string
  fromHeader?: string
  toHeader?: string
  cseqInvite?: string
  direction?: string
  state?: string
  codec?: string
  payloadType?: number
  clockRate?: number
  remoteAddr?: string
  remoteRtpAddr?: string
  localRtpAddr?: string
  recordingUrl?: string
  durationSec?: number
  endStatus?: string
  failureReason?: string
  inviteAt?: string
  ackAt?: string
  byeAt?: string
  endedAt?: string
  turnCount?: number
  firstTurnAt?: string
  lastTurnAt?: string
  hadSipTransfer?: boolean
  hadWebSeat?: boolean
  turns?: SIPCallDialogTurn[]
  createdAt?: string
  updatedAt?: string
}

export interface Paginated<T> {
  list: T[]
  total: number
  page: number
  size: number
}

export interface ACDPoolTargetRow {
  id: number
  name?: string
  routeType: string
  /** SIP: `internal` = registered extension; `trunk` = external PSTN dial string. Web: omit/empty. */
  sipSource?: string
  targetValue?: string
  weight: number
  workState: string
  workStateAt?: string
  /** SIP trunk: gateway host (IP or domain) for Request-URI / signaling. */
  sipTrunkHost?: string
  /** SIP trunk: SIP port on gateway; 0 in DB means unset (UI shows 5060). */
  sipTrunkPort?: number
  /** SIP trunk: optional `host:port` override for where to send INVITE. */
  sipTrunkSignalingAddr?: string
  /** SIP: outbound From user part (CLI); empty → process `SIP_CALLER_ID` default. */
  sipCallerId?: string
  /** SIP: optional From display-name. */
  sipCallerDisplayName?: string
  /** SIP internal only: matched `sip_users.username` currently registered (`online`). */
  liveLineOnline?: boolean
  createdAt?: string
  updatedAt?: string
}

export type ACDSipSource = 'internal' | 'trunk'

export const ACD_SIP_SOURCES: ACDSipSource[] = ['internal', 'trunk']

export type ACDRouteType = 'sip' | 'web'

export const ACD_ROUTE_TYPES: ACDRouteType[] = ['sip', 'web']

export const ACD_WORK_STATES = [
  'offline',
  'available',
  'ringing',
  'busy',
  'acw',
  'break',
] as const

export type ACDWorkState = (typeof ACD_WORK_STATES)[number]

export interface OutboundCampaignRow {
  id: number
  name: string
  status: string
  scenario?: string
  mediaProfile?: string
  scriptId?: string
  createdAt?: string
  updatedAt?: string
}

export interface OutboundCampaignMetrics {
  invited_total: number
  answered_total: number
  failed_total: number
  retrying_total: number
  suppressed_total: number
}

export interface OutboundCampaignLogRow {
  id: number
  at: string
  type: string
  contactId?: number
  attemptId?: number
  attemptNo?: number
  phone?: string
  callId?: string
  correlationId?: string
  level?: string
  message: string
}

export interface OutboundCampaignContactRow {
  id: number
  campaignId: number
  phone: string
  status: string
  attemptCount?: number
  maxAttempts?: number
  failureReason?: string
  nextRunAt?: string
  lastDialAt?: string
  createdAt?: string
  updatedAt?: string
}

export interface SIPScriptTemplateRow {
  id: number
  name: string
  scriptId: string
  version?: string
  description?: string
  enabled: boolean
  scriptSpec: unknown
  createdAt?: string
  updatedAt?: string
}

export async function listACDPoolTargets(
  page = 1,
  size = 20,
  opts?: { routeType?: string }
): Promise<ApiResponse<Paginated<ACDPoolTargetRow>>> {
  const q = new URLSearchParams({ page: String(page), size: String(size) })
  if (opts?.routeType) q.set('routeType', opts.routeType)
  return get(`/sip-center/acd-pool?${q.toString()}`)
}

export async function getACDPoolTarget(id: number): Promise<ApiResponse<ACDPoolTargetRow>> {
  return get(`/sip-center/acd-pool/${id}`)
}

export async function createACDPoolTarget(body: {
  name?: string
  routeType: string
  sipSource?: string
  targetValue?: string
  sipTrunkHost?: string
  sipTrunkPort?: number
  sipTrunkSignalingAddr?: string
  sipCallerId?: string
  sipCallerDisplayName?: string
  weight?: number
  workState?: string
}): Promise<ApiResponse<ACDPoolTargetRow>> {
  return post('/sip-center/acd-pool', body)
}

export async function updateACDPoolTarget(
  id: number,
  body: {
    name?: string
    routeType: string
    sipSource?: string
    targetValue?: string
    sipTrunkHost?: string
    sipTrunkPort?: number
    sipTrunkSignalingAddr?: string
    sipCallerId?: string
    sipCallerDisplayName?: string
    weight?: number
    workState?: string
  }
): Promise<ApiResponse<ACDPoolTargetRow>> {
  return put(`/sip-center/acd-pool/${id}`, body)
}

export async function deleteACDPoolTarget(id: number): Promise<ApiResponse<{ id: number }>> {
  return del(`/sip-center/acd-pool/${id}`)
}

export async function listSIPUsers(page = 1, size = 20): Promise<ApiResponse<Paginated<SIPUserRow>>> {
  return get(`/sip-center/users?page=${page}&size=${size}`)
}

/** Load registered SIP users for ACD internal-extension picker (paginates until empty). */
export async function fetchSIPUsersForSelect(maxTotal = 500): Promise<SIPUserRow[]> {
  const out: SIPUserRow[] = []
  const size = 100
  let page = 1
  while (out.length < maxTotal) {
    const res = await listSIPUsers(page, size)
    if (res.code !== 200 || !res.data?.list?.length) break
    out.push(...res.data.list)
    if (res.data.list.length < size) break
    page += 1
  }
  return out.slice(0, maxTotal)
}

export async function getSIPUser(id: number): Promise<ApiResponse<SIPUserRow>> {
  return get(`/sip-center/users/${id}`)
}

export async function deleteSIPUser(id: number): Promise<ApiResponse<{ id: number }>> {
  return del(`/sip-center/users/${id}`)
}

export async function listSIPCalls(
  page = 1,
  size = 20,
  opts?: { callId?: string; state?: string }
): Promise<ApiResponse<Paginated<SIPCallRow>>> {
  const q = new URLSearchParams({ page: String(page), size: String(size) })
  if (opts?.callId) q.set('callId', opts.callId)
  if (opts?.state) q.set('state', opts.state)
  return get(`/sip-center/calls?${q.toString()}`)
}

export async function getSIPCall(id: number): Promise<ApiResponse<SIPCallRow>> {
  return get(`/sip-center/calls/${id}`)
}

/** Full recording URL for <audio> / download (handles relative paths from API origin). */
export function resolveSipRecordingUrl(url?: string | null): string {
  if (!url) return ''
  const u = url.trim()
  if (/^https?:\/\//i.test(u)) return u
  const base = getApiBaseURL().replace(/\/$/, '')
  return u.startsWith('/') ? `${base}${u}` : `${base}/${u}`
}

/** i18n key under `contactCenter.ai.endStatus.*` for backend `endStatus` string. */
export function sipAiEndStatusI18nKey(status?: string | null): string {
  const s = (status || '').trim()
  const map: Record<string, string> = {
    completed_remote: 'contactCenter.ai.endStatus.completed_remote',
    completed_local: 'contactCenter.ai.endStatus.completed_local',
    after_transfer_remote: 'contactCenter.ai.endStatus.after_transfer_remote',
    after_transfer_local: 'contactCenter.ai.endStatus.after_transfer_local',
  }
  return map[s] || 'contactCenter.ai.endStatus.unknown'
}

export async function createOutboundCampaign(body: {
  name: string
  scenario: string
  media_profile: string
  script_id?: string
  script_version?: string
  script_spec?: string
  system_prompt?: string
  opening_message?: string
  closing_message?: string
  outbound_host?: string
  outbound_port?: number
  signaling_addr?: string
}): Promise<ApiResponse<OutboundCampaignRow>> {
  return post('/sip-center/campaigns', body)
}

export async function listOutboundCampaigns(
  page = 1,
  size = 20,
  opts?: { status?: string; name?: string }
): Promise<ApiResponse<Paginated<OutboundCampaignRow>>> {
  const q = new URLSearchParams({ page: String(page), size: String(size) })
  if (opts?.status) q.set('status', opts.status)
  if (opts?.name) q.set('name', opts.name)
  return get(`/sip-center/campaigns?${q.toString()}`)
}

export async function enqueueOutboundCampaignContacts(
  campaignId: number,
  contacts: Array<{ phone: string; display?: string; priority?: number }>
): Promise<ApiResponse<{ accepted: number }>> {
  return post(`/sip-center/campaigns/${campaignId}/contacts`, contacts)
}

export async function listOutboundCampaignContacts(
  campaignId: number,
  page = 1,
  size = 50
): Promise<ApiResponse<Paginated<OutboundCampaignContactRow>>> {
  const q = new URLSearchParams({ page: String(page), size: String(size) })
  return get(`/sip-center/campaigns/${campaignId}/contacts?${q.toString()}`)
}

export async function resetOutboundCampaignSuppressedContacts(
  campaignId: number
): Promise<ApiResponse<{ updated: number }>> {
  return post(`/sip-center/campaigns/${campaignId}/contacts/reset-suppressed`, {})
}

export async function startOutboundCampaign(campaignId: number): Promise<ApiResponse<null>> {
  return post(`/sip-center/campaigns/${campaignId}/start`, {})
}

export async function pauseOutboundCampaign(campaignId: number): Promise<ApiResponse<null>> {
  return post(`/sip-center/campaigns/${campaignId}/pause`, {})
}

export async function resumeOutboundCampaign(campaignId: number): Promise<ApiResponse<null>> {
  return post(`/sip-center/campaigns/${campaignId}/resume`, {})
}

export async function stopOutboundCampaign(campaignId: number): Promise<ApiResponse<null>> {
  return post(`/sip-center/campaigns/${campaignId}/stop`, {})
}

export async function deleteOutboundCampaign(campaignId: number): Promise<ApiResponse<{ id: number }>> {
  return del(`/sip-center/campaigns/${campaignId}`)
}

export async function getOutboundCampaignMetrics(): Promise<ApiResponse<OutboundCampaignMetrics>> {
  return get('/sip-center/campaigns/metrics')
}

export async function getOutboundCampaignLogs(
  campaignId: number,
  limit = 100
): Promise<ApiResponse<{ list: OutboundCampaignLogRow[]; total: number }>> {
  const q = new URLSearchParams({ limit: String(limit) })
  return get(`/sip-center/campaigns/${campaignId}/logs?${q.toString()}`)
}

export async function listSIPScriptTemplates(
  page = 1,
  size = 20,
  opts?: { scriptId?: string; name?: string }
): Promise<ApiResponse<Paginated<SIPScriptTemplateRow>>> {
  const q = new URLSearchParams({ page: String(page), size: String(size) })
  if (opts?.scriptId) q.set('scriptId', opts.scriptId)
  if (opts?.name) q.set('name', opts.name)
  return get(`/sip-center/scripts?${q.toString()}`)
}

export async function createSIPScriptTemplate(body: {
  name: string
  scriptId?: string
  version?: string
  description?: string
  enabled?: boolean
  scriptSpec: string
}): Promise<ApiResponse<SIPScriptTemplateRow>> {
  return post('/sip-center/scripts', body)
}

export async function updateSIPScriptTemplate(
  id: number,
  body: {
    name: string
    scriptId?: string
    version?: string
    description?: string
    enabled?: boolean
    scriptSpec?: string
  }
): Promise<ApiResponse<SIPScriptTemplateRow>> {
  return put(`/sip-center/scripts/${id}`, body)
}

export async function deleteSIPScriptTemplate(id: number): Promise<ApiResponse<{ id: number }>> {
  return del(`/sip-center/scripts/${id}`)
}

const WEBSEAT_ACD_POOL_ROW_SESSION_KEY = 'soulnexus.webseat.acdPoolTargetId'

function readAnchoredWebSeatAcdPoolId(): number | null {
  if (typeof sessionStorage === 'undefined') return null
  try {
    const s = sessionStorage.getItem(WEBSEAT_ACD_POOL_ROW_SESSION_KEY)
    if (!s) return null
    const id = parseInt(s, 10)
    return Number.isFinite(id) && id > 0 ? id : null
  } catch {
    return null
  }
}

function writeAnchoredWebSeatAcdPoolId(id: number): void {
  if (typeof sessionStorage === 'undefined') return
  sessionStorage.setItem(WEBSEAT_ACD_POOL_ROW_SESSION_KEY, String(id))
}

export function clearWebSeatAcdPoolAnchor(): void {
  if (typeof sessionStorage === 'undefined') return
  sessionStorage.removeItem(WEBSEAT_ACD_POOL_ROW_SESSION_KEY)
}

/** After Go Online: first time in this tab creates a `web` row; later visits update the same row to `available`. */
export async function ensureWebSeatAcdPoolRowOnline(opts: {
  /** Shown as row name, e.g. `张三-Web` (current user + suffix). */
  displayLabel: string
}): Promise<void> {
  const label = opts.displayLabel.trim() || 'Web'
  let anchor = readAnchoredWebSeatAcdPoolId()
  if (anchor != null) {
    const cur = await getACDPoolTarget(anchor)
    if (cur.code === 200 && cur.data?.routeType === 'web') {
      const r = cur.data
      const wt = r.weight != null && r.weight > 0 ? r.weight : 10
      const u = await updateACDPoolTarget(anchor, {
        name: label,
        routeType: 'web',
        sipSource: '',
        targetValue: '',
        weight: wt,
        workState: 'available',
      })
      if (u.code !== 200) throw new Error(u.msg || 'update web seat acd failed')
      return
    }
    clearWebSeatAcdPoolAnchor()
  }
  const c = await createACDPoolTarget({
    name: label,
    routeType: 'web',
    sipSource: '',
    targetValue: '',
    weight: 10,
    workState: 'available',
  })
  if (c.code !== 200 || !c.data?.id) throw new Error(c.msg || 'create web seat acd failed')
  writeAnchoredWebSeatAcdPoolId(c.data.id)
}

/** After Go Offline: anchored `web` row → `offline`. */
export async function setWebSeatAcdPoolRowOffline(): Promise<void> {
  const anchor = readAnchoredWebSeatAcdPoolId()
  if (anchor == null) return
  const cur = await getACDPoolTarget(anchor)
  if (cur.code !== 200 || !cur.data || cur.data.routeType !== 'web') {
    clearWebSeatAcdPoolAnchor()
    return
  }
  const r = cur.data
  const u = await updateACDPoolTarget(anchor, {
    name: r.name || '',
    routeType: 'web',
    sipSource: '',
    targetValue: '',
    weight: r.weight ?? 10,
    workState: 'offline',
  })
  if (u.code !== 200) throw new Error(u.msg || 'set web seat acd offline failed')
}
