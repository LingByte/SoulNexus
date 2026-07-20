import { get, post, put, del, ApiResponse } from '@/utils/request'

// 工作流状态类型
export type WorkflowStatus = 'draft' | 'active' | 'archived'

// 工作流边类型
export type WorkflowEdgeType = 'default' | 'true' | 'false' | 'error' | 'branch'

// 工作流节点类型
export type WorkflowNodeType = 'start' | 'end' | 'task' | 'gateway' | 'event' | 'subflow' | 'parallel' | 'wait' | 'timer' | 'script' | 'plugin' | 'workflow_plugin' | 'condition' | 'ai_chat' | 'knowledge_base'

// 工作流节点 Schema
export interface WorkflowNodeSchema {
  id: string
  name: string
  type: WorkflowNodeType
  description?: string
  inputMap?: Record<string, string>
  outputMap?: Record<string, string>
  properties?: Record<string, string>
  lanes?: string[]
  position?: { x: number; y: number }
}

// 工作流边 Schema
export interface WorkflowEdgeSchema {
  id: string
  source: string
  target: string
  type?: WorkflowEdgeType
  condition?: string
  description?: string
  metadata?: Record<string, any>
}

// 工作流图定义
export interface WorkflowGraph {
  nodes: WorkflowNodeSchema[]
  edges: WorkflowEdgeSchema[]
  metadata?: Record<string, any>
}

// 触发器配置类型
export interface WorkflowTriggerConfig {
  api?: {
    enabled: boolean
    public: boolean
    apiKey?: string
    description?: string
  }
  event?: {
    enabled: boolean
    events: string[]
    condition?: string
  }
  schedule?: {
    enabled: boolean
    cronExpr: string
    timezone?: string
  }
  webhook?: {
    enabled: boolean
    url?: string
    secret?: string
    method?: string
  }
  agent?: {
    enabled: boolean
    agentIds?: number[]
    intents?: string[]
    description?: string
  }
}

// 工作流定义
export interface WorkflowDefinition {
  id: number
  name: string
  slug: string
  description: string
  version: number
  status: WorkflowStatus
  definition: WorkflowGraph
  settings?: Record<string, any>
  triggers?: WorkflowTriggerConfig
  inputParameters?: WorkflowParameter[]
  outputParameters?: WorkflowParameter[]
  tags?: string[]
  createdBy: string
  updatedBy: string
  createdAt: string
  updatedAt: string
}

// 工作流参数定义
export interface WorkflowParameter {
  name: string
  type: 'string' | 'number' | 'boolean' | 'object' | 'array'
  required: boolean
  description?: string
  default?: any
}

// 工作流版本
export interface WorkflowVersion {
  id: number
  definitionId: number
  version: number
  name: string
  slug: string
  description: string
  status: WorkflowStatus
  definition: WorkflowGraph
  settings?: Record<string, any>
  triggers?: WorkflowTriggerConfig
  tags?: string[]
  createdBy: string
  updatedBy: string
  changeNote?: string
  createdAt: string
}

// 工作流版本对比结果
export interface WorkflowVersionDiff {
  name?: { old: string; new: string }
  description?: { old: string; new: string }
  status?: { old: string; new: string }
  nodes?: {
    added?: WorkflowNodeSchema[]
    removed?: WorkflowNodeSchema[]
    modified?: Array<{ id: string; old: WorkflowNodeSchema; new: WorkflowNodeSchema }>
  }
  edges?: {
    added?: WorkflowEdgeSchema[]
    removed?: WorkflowEdgeSchema[]
    modified?: Array<{ id: string; old: WorkflowEdgeSchema; new: WorkflowEdgeSchema }>
  }
  settings?: { old: Record<string, any>; new: Record<string, any> }
  triggers?: { old: Record<string, any>; new: Record<string, any> }
}

// 工作流版本对比响应
export interface WorkflowVersionCompareResponse {
  version1: WorkflowVersion
  version2: WorkflowVersion
  diff: WorkflowVersionDiff
}

// 创建工作流定义请求
export interface CreateWorkflowDefinitionRequest {
  name: string
  slug?: string
  description?: string
  status?: WorkflowStatus
  definition: WorkflowGraph
  settings?: Record<string, any>
  triggers?: WorkflowTriggerConfig
  tags?: string[]
  version?: number
}

// 更新工作流定义请求
export interface UpdateWorkflowDefinitionRequest {
  name?: string
  description?: string
  status?: WorkflowStatus
  definition?: WorkflowGraph
  settings?: Record<string, any>
  triggers?: WorkflowTriggerConfig
  inputParameters?: WorkflowParameter[]
  outputParameters?: WorkflowParameter[]
  tags?: string[]
  version: number // 必须提供当前版本号，用于乐观锁
  changeNote?: string // 版本变更说明
}

// 工作流定义列表查询参数
export interface ListWorkflowDefinitionsParams {
  status?: WorkflowStatus
  keyword?: string
}

// 工作流 API 服务
export const workflowService = {
  /**
   * 创建工作流定义
   */
  async createDefinition(data: CreateWorkflowDefinitionRequest): Promise<ApiResponse<WorkflowDefinition>> {
    return post<WorkflowDefinition>('/workflows/definitions', data)
  },

  /**
   * 获取工作流定义列表
   */
  async listDefinitions(params?: ListWorkflowDefinitionsParams): Promise<ApiResponse<WorkflowDefinition[]>> {
    return get<WorkflowDefinition[]>('/workflows/definitions', { params })
  },

  /**
   * 获取单个工作流定义
   */
  async getDefinition(id: number): Promise<ApiResponse<WorkflowDefinition>> {
    return get<WorkflowDefinition>(`/workflows/definitions/${id}`)
  },

  /**
   * 更新工作流定义
   */
  async updateDefinition(id: number, data: UpdateWorkflowDefinitionRequest): Promise<ApiResponse<WorkflowDefinition>> {
    return put<WorkflowDefinition>(`/workflows/definitions/${id}`, data)
  },

  /** 发布当前草稿为不可变版本（与保存草稿分离） */
  async publishDefinition(id: number, changeNote?: string): Promise<ApiResponse<{ definition: WorkflowDefinition; version: WorkflowVersion }>> {
    return post(`/workflows/definitions/${id}/publish`, { changeNote: changeNote || '' })
  },

  /**
   * 删除工作流定义
   */
  async deleteDefinition(id: number): Promise<ApiResponse<{ message: string }>> {
    return del<{ message: string }>(`/workflows/definitions/${id}`)
  },

  /**
   * 运行工作流定义
   */
  async runDefinition(id: number, parameters?: Record<string, any>): Promise<ApiResponse<RunWorkflowResponse>> {
    return post<RunWorkflowResponse>(`/workflows/definitions/${id}/run`, {
      parameters: parameters || {}
    })
  },

  /**
   * 测试单个节点
   */
  async testNode(
    definitionId: number,
    nodeId: string,
    parameters?: Record<string, any>,
    opts?: { node?: WorkflowNodeSchema; edges?: WorkflowGraph['edges'] },
  ): Promise<ApiResponse<{
    nodeId: string
    nodeName: string
    status: string
    nextNodes: string[]
    context: Record<string, any>
    logs: ExecutionLog[]
    error?: string
  }>> {
    return post(`/workflows/definitions/${definitionId}/nodes/${nodeId}/test`, {
      parameters: parameters || {},
      ...(opts?.node ? { node: opts.node } : {}),
      ...(opts?.edges ? { edges: opts.edges } : {}),
    })
  },

  /**
   * 获取工作流定义的历史版本列表
   */
  async listVersions(definitionId: number): Promise<ApiResponse<WorkflowVersion[]>> {
    return get<WorkflowVersion[]>(`/workflows/definitions/${definitionId}/versions`)
  },

  /**
   * 获取工作流定义的特定版本
   */
  async getVersion(definitionId: number, versionId: number): Promise<ApiResponse<WorkflowVersion>> {
    return get<WorkflowVersion>(`/workflows/definitions/${definitionId}/versions/${versionId}`)
  },

  /**
   * 回滚工作流定义到特定版本
   */
  async rollbackVersion(definitionId: number, versionId: number): Promise<ApiResponse<WorkflowDefinition>> {
    return post<WorkflowDefinition>(`/workflows/definitions/${definitionId}/versions/${versionId}/rollback`)
  },

  /**
   * 对比两个工作流版本
   */
  async compareVersions(definitionId: number, version1Id: number, version2Id: number): Promise<ApiResponse<WorkflowVersionCompareResponse>> {
    return get<WorkflowVersionCompareResponse>(`/workflows/definitions/${definitionId}/versions/compare`, {
      params: {
        version1: version1Id,
        version2: version2Id
      }
    })
  },

  /**
   * 获取可用的事件类型
   */
  async getAvailableEventTypes(): Promise<ApiResponse<{
    event_types: Array<{
      type: string
      first_published: string | null
      source: string
    }>
    total: number
  }>> {
    return get<{
      event_types: Array<{
        type: string
        first_published: string | null
        source: string
      }>
      total: number
    }>('/workflows/events/types')
  },

  /**
   * 停止工作流实例
   */
  async stopInstance(instanceId: number): Promise<ApiResponse<{ instance_id: number; status: string }>> {
    return post<{ instance_id: number; status: string }>(`/workflows/instances/${instanceId}/stop`)
  },

  async listInstances(params: ListWorkflowInstancesParams): Promise<ApiResponse<{ list: WorkflowInstance[]; total: number; page: number; pageSize: number }>> {
    return get('/workflows/instances', { params })
  },

  async getInstance(id: number): Promise<ApiResponse<WorkflowInstance>> {
    return get<WorkflowInstance>(`/workflows/instances/${id}`)
  },

  exportInstancesUrl(params: ListWorkflowInstancesParams): string {
    const q = new URLSearchParams()
    q.set('definitionId', String(params.definitionId))
    if (params.source) q.set('source', params.source)
    if (params.keyword) q.set('keyword', params.keyword)
    if (params.from) q.set('from', params.from)
    if (params.to) q.set('to', params.to)
    return `/workflows/instances/export?${q.toString()}`
  },
}

// 执行日志
export interface ExecutionLog {
  timestamp: string
  level: 'info' | 'success' | 'warning' | 'error' | 'debug'
  message: string
  nodeId?: string
  nodeName?: string
}

// 工作流实例类型
export interface WorkflowInstance {
  id: number
  definitionId: number
  definitionName: string
  groupId?: number
  userId?: number
  triggerUser?: string
  triggerSource?: string
  clientMeta?: Record<string, unknown>
  status: 'pending' | 'running' | 'completed' | 'failed'
  currentNodeId?: string
  inputParameters?: Record<string, any>
  executionLogs?: ExecutionLog[]
  logCount?: number
  durationMs?: number
  errorMessage?: string
  contextData?: Record<string, any>
  resultData?: Record<string, any>
  startedAt?: string
  completedAt?: string
  createdAt: string
  updatedAt: string
}

export interface ListWorkflowInstancesParams {
  definitionId: number
  source?: string
  keyword?: string
  from?: string
  to?: string
  page?: number
  pageSize?: number
}

// 工作流运行响应（包含日志）
export interface RunWorkflowResponse {
  /** 简化响应：id + status + logs + result */
  id?: number
  status?: 'pending' | 'running' | 'completed' | 'failed'
  logs?: ExecutionLog[]
  result?: Record<string, unknown>
  /** 兼容旧格式 */
  instance?: WorkflowInstance
}

export default workflowService

