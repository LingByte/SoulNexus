import { get, post, put, del, type ApiResponse } from '@/utils/request'

export type AssistantToolKind = 'http' | 'mcp_stdio' | 'mcp_sse'

export interface DiscoveredMCPTool {
  name: string
  description?: string
  inputSchema?: Record<string, unknown> | null
}

export interface AssistantToolRow {
  id: string
  tenantId?: string
  name: string
  displayName?: string
  description?: string
  kind: AssistantToolKind | string
  enabled: boolean
  source?: 'custom' | 'market' | string
  marketItemId?: string
  marketPublished?: boolean
  publishedMarketItemId?: string
  method?: string
  url?: string
  headers?: Record<string, string> | null
  bodyTemplate?: string
  timeoutMs?: number
  parameters?: Record<string, unknown> | null
  mcpCommand?: string
  mcpArgs?: string[] | null
  mcpEnvs?: Record<string, string> | null
  mcpSseUrl?: string
  discoveredTools?: DiscoveredMCPTool[] | null
  createdAt?: string
  updatedAt?: string
}

export type AssistantToolWriteBody = {
  name: string
  displayName?: string
  description?: string
  kind?: AssistantToolKind | string
  enabled?: boolean
  method?: string
  url?: string
  headers?: Record<string, string>
  bodyTemplate?: string
  timeoutMs?: number
  parameters?: Record<string, unknown>
  mcpCommand?: string
  mcpArgs?: string[]
  mcpEnvs?: Record<string, string>
  mcpSseUrl?: string
}

export async function listAssistantTools(params?: {
  source?: 'custom' | 'market' | string
  enabled?: boolean
}): Promise<ApiResponse<AssistantToolRow[]>> {
  const q = new URLSearchParams()
  if (params?.source) q.set('source', params.source)
  if (params?.enabled) q.set('enabled', '1')
  const suffix = q.toString() ? `?${q}` : ''
  return get(`/assistant-tools${suffix}`)
}

export async function getAssistantTool(id: string): Promise<ApiResponse<AssistantToolRow>> {
  return get(`/assistant-tools/${id}`)
}

export async function createAssistantTool(body: AssistantToolWriteBody): Promise<ApiResponse<AssistantToolRow>> {
  return post('/assistant-tools', body)
}

export async function updateAssistantTool(
  id: string,
  body: Partial<AssistantToolWriteBody>,
): Promise<ApiResponse<AssistantToolRow>> {
  return put(`/assistant-tools/${id}`, body)
}

export async function deleteAssistantTool(id: string): Promise<ApiResponse<{ id: string }>> {
  return del(`/assistant-tools/${id}`)
}

export async function discoverAssistantTool(
  id: string,
): Promise<ApiResponse<{ tool: AssistantToolRow; discoveredTools: DiscoveredMCPTool[] }>> {
  return post(`/assistant-tools/${id}/discover`, {})
}

/** Bind key for one MCP tool under a catalog MCP server row. */
export function mcpToolBindKey(catalogId: string, toolName: string): string {
  return `${catalogId}:${toolName}`
}
