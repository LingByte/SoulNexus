/**
 * 工作流节点定义
 */
export interface WorkflowNode {
  id: string
  type: 'start' | 'end' | 'task' | 'gateway' | 'event' | 'subflow' | 'parallel' | 'wait' | 'timer' | 'script' | 'workflow_plugin' | 'ai_chat'
  position: { x: number; y: number }
  data: {
    label: string
    config?: any
    pluginId?: number // 插件ID，用于插件节点
    [key: string]: any
  }
  inputs: string[]
  outputs: string[]
}

/**
 * 工作流连接线定义
 */
export interface WorkflowConnection {
  id: string
  source: string
  target: string
  sourceHandle: string
  targetHandle: string
  type?: 'default' | 'true' | 'false' | 'error' | 'branch'
  condition?: string
}

/**
 * 工作流定义
 */
export interface Workflow {
  id: string
  name: string
  description: string
  nodes: WorkflowNode[]
  connections: WorkflowConnection[]
  createdAt: string
  updatedAt: string
}
