import { useCallback, useRef } from 'react'
import workflowService from '@/api/workflow'
import { buildWebSocketURL } from '@/config/apiConfig'
import type { TerminalLog } from '@/components/workflow/Terminal'
import {
  executionLogToTerminal,
  formatRunResultMessage,
  parseRunWorkflowResponse,
} from '@/utils/workflowRun'
import { showAlert } from '@/utils/notification'

function nowTimestamp() {
  const now = new Date()
  return `${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}:${now.getSeconds().toString().padStart(2, '0')}.${now.getMilliseconds().toString().padStart(3, '0')}`
}

interface UseWorkflowRunOptions {
  onLogsChange: React.Dispatch<React.SetStateAction<TerminalLog[]>>
  onVisibleChange: (visible: boolean) => void
  onRunningChange: (running: boolean) => void
  onInstanceIdChange: (id: number | null) => void
}

export function useWorkflowRun({
  onLogsChange,
  onVisibleChange,
  onRunningChange,
  onInstanceIdChange,
}: UseWorkflowRunOptions) {
  const wsRef = useRef<WebSocket | null>(null)

  const appendLog = useCallback(
    (log: TerminalLog) => {
      onLogsChange((prev) => [...prev, log])
    },
    [onLogsChange],
  )

  const connectWebSocket = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }
    try {
      const token = localStorage.getItem('auth_token')
      const wsUrl = token
        ? `${buildWebSocketURL('/api/ws')}?token=${encodeURIComponent(token)}`
        : buildWebSocketURL('/api/ws')
      const ws = new WebSocket(wsUrl)
      wsRef.current = ws
      ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data.toString().trim())
          if (message.type === 'workflow_log' && message.data) {
            appendLog(executionLogToTerminal(message.data))
          }
        } catch {
          /* ignore malformed ws payloads */
        }
      }
    } catch (error) {
      console.error('Failed to establish WebSocket connection:', error)
    }
  }, [appendLog])

  const disconnectWebSocket = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }
  }, [])

  const runWorkflow = useCallback(
    async (definitionId: number, workflowName: string, parameters: Record<string, unknown> = {}) => {
      onLogsChange([])
      onVisibleChange(true)
      connectWebSocket()

      appendLog({
        timestamp: nowTimestamp(),
        level: 'info',
        message: `开始运行工作流: ${workflowName}`,
      })

      try {
        const response = await workflowService.runDefinition(definitionId, parameters)
        if (response.code !== 200 || !response.data) {
          appendLog({
            timestamp: nowTimestamp(),
            level: 'error',
            message: response.msg || '运行工作流时发生错误',
          })
          showAlert(response.msg || '运行工作流时发生错误', 'error', '运行失败')
          return
        }

        const parsed = parseRunWorkflowResponse(response.data)
        if (!parsed) {
          appendLog({
            timestamp: nowTimestamp(),
            level: 'error',
            message: '无法解析工作流执行结果',
          })
          showAlert('无法解析工作流执行结果', 'error', '运行失败')
          return
        }

        onInstanceIdChange(parsed.instanceId)
        onRunningChange(parsed.status === 'running')

        if (parsed.logs.length > 0) {
          onLogsChange((prev) => [...prev, ...parsed.logs.map(executionLogToTerminal)])
        }

        const resultMessage = formatRunResultMessage(parsed.result)
        if (resultMessage) {
          appendLog({
            timestamp: nowTimestamp(),
            level: 'info',
            message: resultMessage,
          })
        }

        if (parsed.status === 'completed') {
          appendLog({
            timestamp: nowTimestamp(),
            level: 'success',
            message: '工作流执行完成',
          })
          onRunningChange(false)
          onInstanceIdChange(null)
          showAlert('工作流执行完成', 'success', '运行成功')
        } else if (parsed.status === 'failed') {
          appendLog({
            timestamp: nowTimestamp(),
            level: 'error',
            message: '工作流执行失败',
          })
          onRunningChange(false)
          onInstanceIdChange(null)
          showAlert('工作流执行失败', 'error', '运行失败')
        }
      } catch (error: unknown) {
        const err = error as { msg?: string; message?: string }
        appendLog({
          timestamp: nowTimestamp(),
          level: 'error',
          message: err.msg || err.message || '运行工作流时发生错误',
        })
        showAlert(err.msg || err.message || '运行工作流时发生错误', 'error', '运行失败')
      } finally {
        disconnectWebSocket()
      }
    },
    [
      appendLog,
      connectWebSocket,
      disconnectWebSocket,
      onInstanceIdChange,
      onLogsChange,
      onRunningChange,
      onVisibleChange,
    ],
  )

  return { runWorkflow, disconnectWebSocket }
}
