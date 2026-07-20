import { get, post, put, del } from '@/utils/request'

/** Snowflake entity id — keep as string in JS to avoid precision loss. */
export type EntityId = string

function idPath(id: EntityId | number): string {
  const s = String(id).trim()
  if (!s || s === '0') {
    throw new Error('invalid id')
  }
  return s
}

export interface NotificationChannel {
  id: EntityId
  type: 'email' | 'sms'
  code?: string
  name: string
  sortOrder: number
  enabled: boolean
  remark?: string
  configJson?: string
  createdAt?: string
  updatedAt?: string
}

export interface EmailChannelForm {
  driver: 'smtp' | 'sendcloud'
  smtpHost?: string
  smtpPort?: number
  smtpUsername?: string
  smtpFrom?: string
  fromDisplayName?: string
  smtpPasswordSet?: boolean
  sendcloudApiUser?: string
  sendcloudApiKeySet?: boolean
  sendcloudFrom?: string
}

export interface SMSChannelForm {
  provider: string
  config: Record<string, unknown>
  secretKeys?: string[]
}

export interface ChannelDetail {
  channel: NotificationChannel
  emailForm?: EmailChannelForm
  smsForm?: SMSChannelForm
}

export interface ChannelListResp {
  list: NotificationChannel[]
  total: number
  page: number
  pageSize: number
  totalPage: number
}

export interface UpsertChannelReq {
  channelType: 'email' | 'sms'
  name: string
  sortOrder?: number
  enabled?: boolean
  remark?: string
  driver?: 'smtp' | 'sendcloud'
  smtpHost?: string
  smtpPort?: number
  smtpUsername?: string
  smtpPassword?: string
  smtpFrom?: string
  sendcloudApiUser?: string
  sendcloudApiKey?: string
  sendcloudFrom?: string
  fromDisplayName?: string
  smsProvider?: string
  smsConfig?: Record<string, unknown>
}

export interface MailTemplate {
  id: EntityId
  code: string
  name: string
  subject: string
  htmlBody: string
  textBody?: string
  description?: string
  variables?: string
  locale?: string
  enabled: boolean
  createdAt?: string
  updatedAt?: string
}

export interface MailTemplateUpsertReq {
  code?: string
  name: string
  subject?: string
  htmlBody: string
  description?: string
  variables?: string
  locale?: string
  enabled?: boolean
}

export interface MailLog {
  id: EntityId
  user_id: EntityId
  provider: string
  channel_name: string
  to_email: string
  subject: string
  html_body?: string
  status: string
  error_msg?: string
  message_id?: string
  sent_at?: string
  created_at?: string
  updated_at?: string
}

export interface SMSLog {
  id: EntityId
  user_id: EntityId
  provider: string
  channel_name: string
  to_phone: string
  template?: string
  content?: string
  status: string
  error_msg?: string
  message_id?: string
  sent_at?: string
  created_at?: string
}

export interface PageResp<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
  totalPage: number
}

export const listNotificationChannels = async (params?: {
  type?: string
  page?: number
  pageSize?: number
}): Promise<ChannelListResp> => {
  const res = await get<ChannelListResp>('/admin/notification-channels', { params })
  return res.data!
}

export const getNotificationChannel = async (id: EntityId | number): Promise<ChannelDetail> => {
  const res = await get<ChannelDetail>(`/admin/notification-channels/${idPath(id)}`)
  return res.data!
}

export const createNotificationChannel = async (data: UpsertChannelReq): Promise<NotificationChannel> => {
  const res = await post<NotificationChannel>('/admin/notification-channels', data)
  return res.data!
}

export const updateNotificationChannel = async (id: EntityId | number, data: UpsertChannelReq): Promise<NotificationChannel> => {
  const res = await put<NotificationChannel>(`/admin/notification-channels/${idPath(id)}`, data)
  return res.data!
}

export const deleteNotificationChannel = async (id: EntityId | number): Promise<void> => {
  await del(`/admin/notification-channels/${idPath(id)}`)
}

export const listMailTemplates = async (params?: { page?: number; pageSize?: number }): Promise<PageResp<MailTemplate>> => {
  const res = await get<PageResp<MailTemplate>>('/admin/mail-templates', { params })
  return res.data!
}

export const getMailTemplate = async (id: EntityId | number): Promise<MailTemplate> => {
  const res = await get<MailTemplate>(`/admin/mail-templates/${idPath(id)}`)
  return res.data!
}

export const createMailTemplate = async (data: MailTemplateUpsertReq): Promise<MailTemplate> => {
  const res = await post<MailTemplate>('/admin/mail-templates', data)
  return res.data!
}

export const updateMailTemplate = async (id: EntityId | number, data: MailTemplateUpsertReq): Promise<MailTemplate> => {
  const res = await put<MailTemplate>(`/admin/mail-templates/${idPath(id)}`, data)
  return res.data!
}

export const deleteMailTemplate = async (id: EntityId | number): Promise<void> => {
  await del(`/admin/mail-templates/${idPath(id)}`)
}

export const listAdminMailLogs = async (params?: {
  page?: number
  pageSize?: number
  status?: string
  provider?: string
  channel_name?: string
  user_id?: number
}): Promise<PageResp<MailLog>> => {
  const res = await get<PageResp<MailLog>>('/admin/mail-logs', { params })
  return res.data!
}

export const getAdminMailLog = async (id: EntityId | number): Promise<MailLog> => {
  const res = await get<MailLog>(`/admin/mail-logs/${idPath(id)}`)
  return res.data!
}

export const listAdminSMSLogs = async (params?: {
  page?: number
  pageSize?: number
  status?: string
  provider?: string
  channel_name?: string
  to_phone?: string
  user_id?: number
}): Promise<PageResp<SMSLog>> => {
  const res = await get<PageResp<SMSLog>>('/admin/sms-logs', { params })
  return res.data!
}

export const getAdminSMSLog = async (id: EntityId | number): Promise<SMSLog> => {
  const res = await get<SMSLog>(`/admin/sms-logs/${idPath(id)}`)
  return res.data!
}

export const adminSendSMS = async (data: {
  to: string
  content?: string
  template?: string
  data?: Record<string, string>
}) => {
  const res = await post<{ to: string }>('/admin/sms/send', data)
  return res.data!
}
