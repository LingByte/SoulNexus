import { get, post, del } from '@/utils/request'

export type EntityId = string

function idPath(id: EntityId | number): string {
  const s = String(id).trim()
  if (!s || s === '0') throw new Error('invalid id')
  return s
}

export type ExecutionTaskStatus = 'pending' | 'running' | 'success' | 'failed' | 'canceled'

export interface ExecutionTask {
  id: EntityId
  taskId: string
  queueName: string
  title?: string
  kind?: string
  source?: string
  tenantId?: string
  priority: number
  paramsJson: string
  resultJson?: string
  status: ExecutionTaskStatus | string
  progress: number
  retryCount: number
  maxRetries: number
  errorMsg?: string
  submitTime: string
  startedAt?: string
  finishedAt?: string
  workerId?: string
  remark?: string
  createBy?: string
  updateBy?: string
  createdAt?: string
  updatedAt?: string
}

export interface ExecutionTaskStats {
  total: number
  byStatus?: Record<string, number>
}

export interface PageResp<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
  totalPage: number
}

export interface ListExecutionTasksParams {
  page?: number
  pageSize?: number
  queueName?: string
  kind?: string
  source?: string
  status?: string
  taskId?: string
  search?: string
  tenantId?: string
}

export const listExecutionTasks = async (
  params?: ListExecutionTasksParams,
): Promise<PageResp<ExecutionTask>> => {
  const res = await get<PageResp<ExecutionTask>>('/admin/execution-tasks', { params })
  return res.data!
}

export const getExecutionTaskStats = async (queueName?: string): Promise<ExecutionTaskStats> => {
  const res = await get<ExecutionTaskStats>('/admin/execution-tasks/stats/summary', {
    params: queueName ? { queueName } : undefined,
  })
  return res.data!
}

export const getExecutionTask = async (id: EntityId | number): Promise<ExecutionTask> => {
  const res = await get<ExecutionTask>(`/admin/execution-tasks/${idPath(id)}`)
  return res.data!
}

export const cancelExecutionTask = async (id: EntityId | number): Promise<ExecutionTask> => {
  const res = await post<ExecutionTask>(`/admin/execution-tasks/${idPath(id)}/cancel`)
  return res.data!
}

export const retryExecutionTask = async (id: EntityId | number): Promise<ExecutionTask> => {
  const res = await post<ExecutionTask>(`/admin/execution-tasks/${idPath(id)}/retry`)
  return res.data!
}

export const deleteExecutionTask = async (id: EntityId | number): Promise<void> => {
  await del(`/admin/execution-tasks/${idPath(id)}`)
}
