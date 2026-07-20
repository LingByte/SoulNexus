import { getApiBaseURL } from '@/config/apiConfig'
import type { WorkflowDefinition } from '@/api/workflow'

/** 构建公开 API 执行的完整 URL（供 curl 示例使用） */
export function buildPublicWorkflowExecuteUrl(slug: string): string {
  const apiBase = getApiBaseURL().replace(/\/$/, '')
  const path = `${apiBase.startsWith('/') ? apiBase : `/${apiBase}`}/public/workflows/${slug}/execute`

  if (apiBase.startsWith('http://') || apiBase.startsWith('https://')) {
    return `${apiBase}/public/workflows/${slug}/execute`
  }

  const origin = typeof window !== 'undefined' ? window.location.origin : 'http://localhost:9003'
  return `${origin}${path.startsWith('/') ? path : `/${path}`}`
}

/** 生成可复制执行的 curl 命令 */
export function buildWorkflowExecuteCurl(opts: {
  slug: string
  apiKey?: string
  body?: string
}): string {
  const url = buildPublicWorkflowExecuteUrl(opts.slug)
  const body = (opts.body || '{"parameters":{}}').trim()
  const lines = [
    `curl -X POST '${url}' \\`,
    `  -H 'Content-Type: application/json' \\`,
  ]
  if (opts.apiKey?.trim()) {
    lines.push(`  -H 'X-API-Key: ${opts.apiKey.trim()}' \\`)
  }
  lines.push(`  -d '${body.replace(/'/g, `'\\''`)}'`)
  return lines.join('\n')
}

/** 从工作流定义提取开始节点入参名 */
export function getWorkflowStartParameterNames(workflow: WorkflowDefinition | null): string[] {
  if (!workflow) return []

  const startNode = workflow.definition?.nodes?.find((n) => n.type === 'start')
  if (startNode) {
    const fromInputMap = Object.keys(startNode.inputMap || {})
    if (fromInputMap.length > 0) return fromInputMap

    const inputs = startNode.properties?.inputs
    if (typeof inputs === 'string' && inputs.trim()) {
      return inputs.split(',').map((s) => s.trim()).filter(Boolean)
    }
  }

  if (workflow.inputParameters?.length) {
    return workflow.inputParameters.map((p) => p.name)
  }

  return []
}

/** 根据开始节点入参生成 API 测试默认请求体 */
export function buildDefaultApiTestBody(workflow: WorkflowDefinition | null): string {
  const keys = getWorkflowStartParameterNames(workflow)
  const parameters: Record<string, string> = {}
  for (const key of keys) {
    parameters[key] = key === 'city' ? '成都' : ''
  }
  return JSON.stringify({ parameters }, null, 2)
}

/**
 * 规范化 API 测试请求体。
 * 支持 {"parameters": {...}} 或简写 {"city": "成都"}（自动包一层 parameters）。
 */
export function normalizeApiTestBody(raw: string): { body: string; error?: string } {
  const trimmed = (raw || '').trim()
  if (!trimmed) {
    return { body: JSON.stringify({ parameters: {} }) }
  }

  let parsed: unknown
  try {
    parsed = JSON.parse(trimmed)
  } catch {
    return { body: raw, error: 'invalid_json' }
  }

  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    return { body: JSON.stringify({ parameters: {} }) }
  }

  const obj = parsed as Record<string, unknown>
  if (obj.parameters && typeof obj.parameters === 'object' && !Array.isArray(obj.parameters)) {
    return { body: JSON.stringify({ parameters: obj.parameters }) }
  }

  // 顶层直接写字段时自动包一层 parameters
  if (Object.keys(obj).length > 0) {
    return { body: JSON.stringify({ parameters: obj }) }
  }

  return { body: JSON.stringify({ parameters: {} }) }
}
