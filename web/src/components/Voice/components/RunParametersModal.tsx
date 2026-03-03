import React from 'react'
import Modal from '@/components/UI/Modal'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import type { WorkflowNode } from '../types'

interface RunParametersModalProps {
  isOpen: boolean
  onClose: () => void
  nodes: WorkflowNode[]
  parameters: Record<string, string>
  onParameterChange: (key: string, value: string) => void
  onRun: (parameters: Record<string, any>) => void
  isRunning: boolean
}

/**
 * 运行参数模态框组件
 * 用于输入工作流运行时的参数
 */
export const RunParametersModal: React.FC<RunParametersModalProps> = ({
  isOpen,
  onClose,
  nodes,
  parameters,
  onParameterChange,
  onRun,
  isRunning
}) => {
  const startNode = nodes.find(n => n.type === 'start')
  const hasInputs = startNode && startNode.inputs && startNode.inputs.length > 0

  const handleRun = () => {
    // 转换参数值（尝试解析 JSON，如果失败则作为字符串）
    const parsedParams: Record<string, any> = {}
    Object.entries(parameters).forEach(([key, value]) => {
      if (value.trim() === '') {
        parsedParams[key] = null
      } else {
        try {
          // 尝试解析为 JSON
          parsedParams[key] = JSON.parse(value)
        } catch {
          // 解析失败，作为字符串
          parsedParams[key] = value
        }
      }
    })
    onRun(parsedParams)
  }

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="填写工作流参数"
      size="md"
    >
      <div className="space-y-4">
        <p className="text-sm text-gray-600 dark:text-gray-400">
          请填写开始节点的输入参数值：
        </p>

        {!hasInputs ? (
          <p className="text-sm text-gray-500 dark:text-gray-400 italic">
            开始节点没有定义输入参数
          </p>
        ) : (
          <div className="space-y-3">
            {startNode!.inputs.map((input, idx) => (
              <div key={idx}>
                <Input
                  label={input || `参数 ${idx + 1}`}
                  size="sm"
                  value={parameters[input] || ''}
                  onChange={(e) => onParameterChange(input, e.target.value)}
                  placeholder={`请输入 ${input || `参数 ${idx + 1}`} 的值`}
                />
              </div>
            ))}
          </div>
        )}

        {/* 参数提示 */}
        <div className="p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
          <p className="text-xs text-blue-800 dark:text-blue-200">
            <strong>提示：</strong>支持 JSON 格式的参数值，如 <code className="bg-blue-100 dark:bg-blue-900 px-1 rounded">{"{"}"key": "value"{"}"}</code>
          </p>
        </div>

        {/* 操作按钮 */}
        <div className="flex items-center justify-end gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
          <Button
            variant="outline"
            size="sm"
            onClick={onClose}
            disabled={isRunning}
          >
            取消
          </Button>
          <Button
            variant="primary"
            size="sm"
            onClick={handleRun}
            disabled={isRunning}
            loading={isRunning}
          >
            {isRunning ? '运行中...' : '运行'}
          </Button>
        </div>
      </div>
    </Modal>
  )
}
