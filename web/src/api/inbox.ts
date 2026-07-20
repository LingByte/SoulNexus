import { get, post, put, del, type ApiResponse } from '@/utils/request'

export interface InboxMessage {
  id: string
  user_id: number | string
  title: string
  content: string
  action_url?: string
  action_label?: string
  read: boolean
  created_at: string
}

export async function fetchInboxUnreadCount(): Promise<ApiResponse<number>> {
  return get<number>('/notification/unread-count')
}

export async function listInboxMessages(params?: {
  page?: number
  pageSize?: number
  filter?: 'all' | 'read' | 'unread'
}): Promise<
  ApiResponse<{
    list: InboxMessage[]
    total: number
    totalUnread: number
    totalRead: number
    page: number
    size: number
  }>
> {
  return get('/notification', params)
}

export async function markInboxRead(id: string): Promise<ApiResponse<unknown>> {
  return put(`/notification/read/${id}`)
}

export async function markAllInboxRead(): Promise<ApiResponse<unknown>> {
  return post('/notification/readAll')
}

export async function deleteInboxMessage(id: string): Promise<ApiResponse<unknown>> {
  return del(`/notification/${id}`)
}
