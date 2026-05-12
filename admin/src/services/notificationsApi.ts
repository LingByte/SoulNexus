// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
import { get, post, put, del } from '@/utils/request'
import { getMainApiBaseURL } from '@/config/apiConfig'

const BASE = getMainApiBaseURL()

// ---------------- Types ----------------
export interface NotificationChannel {
  id: number
  orgId: number
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
  config: Record<string, any>
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
  smsConfig?: Record<string, any>
}

export interface MailTemplate {
  id: number
  orgId: number
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
  id: number
  org_id: number
  user_id: number
  provider: string
  channel_name: string
  to_email: string
  cc?: string
  bcc?: string
  subject: string
  body?: string
  html_body?: string
  status: string
  error_msg?: string
  message_id?: string
  raw?: string
  ip_address?: string
  retry_count?: number
  sent_at?: string
  created_at?: string
  updated_at?: string
}

export interface SMSLog {
  id: number
  org_id: number
  user_id: number
  provider: string
  channel_name: string
  to_phone: string
  template?: string
  content?: string
  status: string
  error_msg?: string
  message_id?: string
  raw?: string
  ip_address?: string
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

// ---------------- Notification Channels ----------------
export const listNotificationChannels = async (params?: {
  type?: string
  page?: number
  pageSize?: number
}): Promise<ChannelListResp> => {
  const res = await get<ChannelListResp>(`${BASE}/admin/notification-channels`, { params })
  return res.data
}

export const getNotificationChannel = async (id: number): Promise<ChannelDetail> => {
  const res = await get<ChannelDetail>(`${BASE}/admin/notification-channels/${id}`)
  return res.data
}

export const createNotificationChannel = async (data: UpsertChannelReq): Promise<NotificationChannel> => {
  const res = await post<NotificationChannel>(`${BASE}/admin/notification-channels`, data)
  return res.data
}

export const updateNotificationChannel = async (id: number, data: UpsertChannelReq): Promise<NotificationChannel> => {
  const res = await put<NotificationChannel>(`${BASE}/admin/notification-channels/${id}`, data)
  return res.data
}

export const deleteNotificationChannel = async (id: number): Promise<void> => {
  await del(`${BASE}/admin/notification-channels/${id}`)
}

// ---------------- Mail Templates ----------------
export const listMailTemplates = async (params?: {
  page?: number
  pageSize?: number
}): Promise<PageResp<MailTemplate>> => {
  const res = await get<PageResp<MailTemplate>>(`${BASE}/admin/mail-templates`, { params })
  return res.data
}

export const getMailTemplate = async (id: number): Promise<MailTemplate> => {
  const res = await get<MailTemplate>(`${BASE}/admin/mail-templates/${id}`)
  return res.data
}

export const createMailTemplate = async (data: MailTemplateUpsertReq): Promise<MailTemplate> => {
  const res = await post<MailTemplate>(`${BASE}/admin/mail-templates`, data)
  return res.data
}

export const updateMailTemplate = async (id: number, data: MailTemplateUpsertReq): Promise<MailTemplate> => {
  const res = await put<MailTemplate>(`${BASE}/admin/mail-templates/${id}`, data)
  return res.data
}

export const deleteMailTemplate = async (id: number): Promise<void> => {
  await del(`${BASE}/admin/mail-templates/${id}`)
}

// ---------------- Mail / SMS Logs ----------------
export const listAdminMailLogs = async (params?: {
  page?: number
  pageSize?: number
  status?: string
  provider?: string
  channel_name?: string
  user_id?: number
}): Promise<PageResp<MailLog>> => {
  const res = await get<PageResp<MailLog>>(`${BASE}/admin/mail-logs`, { params })
  return res.data
}

export const getAdminMailLog = async (id: number): Promise<MailLog> => {
  const res = await get<MailLog>(`${BASE}/admin/mail-logs/${id}`)
  return res.data
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
  const res = await get<PageResp<SMSLog>>(`${BASE}/admin/sms-logs`, { params })
  return res.data
}

export const getAdminSMSLog = async (id: number): Promise<SMSLog> => {
  const res = await get<SMSLog>(`${BASE}/admin/sms-logs/${id}`)
  return res.data
}

export const adminSendSMS = async (data: {
  to: string
  content?: string
  template?: string
  data?: Record<string, string>
}) => {
  const res = await post<{ to: string }>(`${BASE}/admin/sms/send`, data)
  return res.data
}
