import { useCallback } from 'react'
import type { WorkflowNode } from '../types/workflow'

interface ValidationResult {
  valid: boolean
  message: string
}

interface UseWorkflowValidationProps {
  nodes: WorkflowNode[]
}

/**
 * 工作流验证 hooks
 * 包含工作流结构验证等操作
 */
export const useWorkflowValidation = ({ nodes }: UseWorkflowValidationProps) => {
  // 验证工作流
  const validateWorkflow = useCallback((): ValidationResult => {
    const startNodes = nodes.filter(n => n.type === 'start')
    const endNodes = nodes.filter(n => n.type === 'end')
    
    if (startNodes.length === 0) {
      return { valid: false, message: '工作流必须有一个开始节点' }
    }
    if (startNodes.length > 1) {
      return { valid: false, message: '工作流只能有一个开始节点' }
    }
    if (endNodes.length === 0) {
      return { valid: false, message: '工作流必须有一个结束节点' }
    }
    
    return { valid: true, message: '工作流验证通过' }
  }, [nodes])

  return {
    validateWorkflow
  }
}
