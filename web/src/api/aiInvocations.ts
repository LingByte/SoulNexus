import { get } from '@/utils/request'

export type AIInvocationLog = {
  id: string
  tenant_id?: string
  user_id?: string
  component: string
  provider?: string
  model?: string
  status: string
  call_id?: string
  source?: string
  latency_ms?: number
  first_token_ms?: number
  prompt_tokens?: number
  completion_tokens?: number
  total_tokens?: number
  input_chars?: number
  output_chars?: number
  audio_ms?: number
  audio_bytes?: number
  intent_name?: string
  confidence?: number
  request_text?: string
  response_text?: string
  error_msg?: string
  meta_json?: string
  created_at?: string
}

type PageResp = {
  list: AIInvocationLog[]
  total: number
  page: number
  pageSize: number
}

function buildQuery(page: number, pageSize: number, opts?: Record<string, string | undefined>) {
  const q = new URLSearchParams({ page: String(page), pageSize: String(pageSize) })
  if (opts) {
    for (const [k, v] of Object.entries(opts)) {
      if (v) q.set(k, v)
    }
  }
  return q
}

export async function listTenantAIInvocations(
  page = 1,
  pageSize = 20,
  opts?: {
    component?: string
    status?: string
    provider?: string
    call_id?: string
    source?: string
  },
): Promise<PageResp> {
  const res = await get<PageResp>(`/ai-invocations/tenant?${buildQuery(page, pageSize, opts).toString()}`)
  return res.data!
}

export async function getTenantAIInvocation(id: string): Promise<AIInvocationLog> {
  const res = await get<AIInvocationLog>(`/ai-invocations/tenant/${id}`)
  return res.data!
}

export async function listAdminAIInvocations(
  page = 1,
  pageSize = 20,
  opts?: {
    tenant_id?: string
    component?: string
    status?: string
    provider?: string
    call_id?: string
    source?: string
  },
): Promise<PageResp> {
  const res = await get<PageResp>(`/admin/ai-invocations?${buildQuery(page, pageSize, opts).toString()}`)
  return res.data!
}

export async function getAdminAIInvocation(id: string): Promise<AIInvocationLog> {
  const res = await get<AIInvocationLog>(`/admin/ai-invocations/${id}`)
  return res.data!
}
