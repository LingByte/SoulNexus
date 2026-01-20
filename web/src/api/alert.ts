import { get, post, put, del, ApiResponse } from '@/utils/request'

// 告警类型
export type AlertType = 'system_error' | 'quota_exceeded' | 'service_error' | 'custom'

// 告警严重程度
export type AlertSeverity = 'critical' | 'high' | 'medium' | 'low'

// 告警状态
export type AlertStatus = 'active' | 'resolved' | 'muted'

// 通知渠道
export type NotificationChannel = 'email' | 'internal' | 'webhook' | 'sms'

// 告警条件
export interface AlertCondition {
  quotaType?: string
  quotaThreshold?: number
  errorCount?: number
  errorWindow?: number
  serviceName?: string
  failureRate?: number
  responseTime?: number
  customExpression?: string
}

// 告警规则
export interface AlertRule {
  id: number
  userId: number
  name: string
  description?: string
  alertType: AlertType
  severity: AlertSeverity
  conditions: string // JSON string
  channels: string // JSON array string
  webhookUrl?: string
  webhookMethod?: string
  cooldown: number
  enabled: boolean
  triggerCount: number
  lastTriggerAt?: string
  createdAt: string
  updatedAt: string
}

// 告警记录
export interface Alert {
  id: number
  userId: number
  ruleId: number
  rule?: AlertRule
  alertType: AlertType
  severity: AlertSeverity
  title: string
  message: string
  data?: string // JSON string
  status: AlertStatus
  resolvedAt?: string
  resolvedBy?: number
  notified: boolean
  notifiedAt?: string
  createdAt: string
  updatedAt: string
}

// 告警通知记录
export interface AlertNotification {
  id: number
  alertId: number
  alert?: Alert
  channel: NotificationChannel
  status: string
  message?: string
  sentAt?: string
  createdAt: string
}

// 创建告警规则请求
export interface CreateAlertRuleRequest {
  name: string
  description?: string
  alertType: AlertType
  severity: AlertSeverity
  conditions: AlertCondition
  channels: NotificationChannel[]
  webhookUrl?: string
  webhookMethod?: string
  cooldown?: number
  enabled?: boolean
}

// 更新告警规则请求
export interface UpdateAlertRuleRequest {
  name?: string
  description?: string
  severity?: AlertSeverity
  conditions?: AlertCondition
  channels?: NotificationChannel[]
  webhookUrl?: string
  webhookMethod?: string
  cooldown?: number
  enabled?: boolean
}

// 告警列表响应
export interface AlertListResponse {
  list: Alert[]
  total: number
  page: number
  pageSize: number
}

// 创建告警规则
export const createAlertRule = async (data: CreateAlertRuleRequest): Promise<ApiResponse<AlertRule>> => {
  return post('/alert/rules', data)
}

// 获取告警规则列表
export const getAlertRules = async (params?: {
  alertType?: AlertType
  enabled?: boolean
}): Promise<ApiResponse<AlertRule[]>> => {
  return get('/alert/rules', { params })
}

// 获取告警规则详情
export const getAlertRule = async (id: number): Promise<ApiResponse<AlertRule>> => {
  return get(`/alert/rules/${id}`)
}

// 更新告警规则
export const updateAlertRule = async (id: number, data: UpdateAlertRuleRequest): Promise<ApiResponse<AlertRule>> => {
  return put(`/alert/rules/${id}`, data)
}

// 删除告警规则
export const deleteAlertRule = async (id: number): Promise<ApiResponse<null>> => {
  return del(`/alert/rules/${id}`)
}

// 获取告警列表
export const getAlerts = async (params?: {
  status?: AlertStatus
  alertType?: AlertType
  page?: number
  pageSize?: number
}): Promise<ApiResponse<AlertListResponse>> => {
  return get('/alert', { params })
}

// 获取告警详情
export const getAlert = async (id: number): Promise<ApiResponse<{
  alert: Alert
  notifications: AlertNotification[]
}>> => {
  return get(`/alert/${id}`)
}

// 解决告警
export const resolveAlert = async (id: number): Promise<ApiResponse<Alert>> => {
  return post(`/alert/${id}/resolve`)
}

// 静音告警
export const muteAlert = async (id: number): Promise<ApiResponse<Alert>> => {
  return post(`/alert/${id}/mute`)
}

