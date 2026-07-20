import type { ExecutionLog } from '@/api/workflow'
import type { TerminalLog } from '@/components/workflow/Terminal'

/** 后端 run 接口的简化响应格式 */
export interface RunWorkflowSimpleResponse {
  id: number
  status: 'pending' | 'running' | 'completed' | 'failed'
  logs?: ExecutionLog[]
  result?: Record<string, unknown>
  context?: Record<string, unknown>
}

export interface ParsedRunWorkflowResult {
  instanceId: number
  status: RunWorkflowSimpleResponse['status']
  logs: ExecutionLog[]
  result?: Record<string, unknown>
}

export function executionLogToTerminal(log: ExecutionLog): TerminalLog {
  return {
    timestamp: log.timestamp,
    level: log.level as TerminalLog['level'],
    message: log.message,
    nodeId: log.nodeId,
    nodeName: log.nodeName,
  }
}

export function parseRunWorkflowResponse(data: unknown): ParsedRunWorkflowResult | null {
  if (!data || typeof data !== 'object') return null
  const payload = data as Record<string, unknown>

  if ('instance' in payload && payload.instance && typeof payload.instance === 'object') {
    const instance = payload.instance as { id: number; status: ParsedRunWorkflowResult['status']; resultData?: Record<string, unknown> }
    const logs = Array.isArray(payload.logs)
      ? (payload.logs as ExecutionLog[])
      : Array.isArray((payload.instance as { logs?: ExecutionLog[] }).logs)
        ? ((payload.instance as { logs: ExecutionLog[] }).logs)
        : []
    return {
      instanceId: instance.id,
      status: instance.status,
      logs,
      result: instance.resultData,
    }
  }

  if ('id' in payload && 'status' in payload) {
    return {
      instanceId: Number(payload.id),
      status: payload.status as ParsedRunWorkflowResult['status'],
      logs: Array.isArray(payload.logs) ? (payload.logs as ExecutionLog[]) : [],
      result: payload.result as Record<string, unknown> | undefined,
    }
  }

  return null
}

export function formatRunResultMessage(result: Record<string, unknown> | undefined): string | null {
  if (!result || Object.keys(result).length === 0) return null
  try {
    return `执行结果: ${JSON.stringify(result, null, 2)}`
  } catch {
    return `执行结果: ${String(result)}`
  }
}
