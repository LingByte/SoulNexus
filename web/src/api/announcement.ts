import { get, ApiResponse } from '@/utils/request'

export interface Announcement {
  id: number
  title: string
  summary?: string
  content: string
  status: 'draft' | 'published' | 'offline'
  pinned?: boolean
  publishAt?: string
  expireAt?: string
  createdAt: string
  updatedAt?: string
}

export interface AnnouncementListResponse {
  items: Announcement[]
  total: number
  page: number
  pageSize: number
}

export const listAnnouncements = async (params?: {
  page?: number
  pageSize?: number
}): Promise<ApiResponse<AnnouncementListResponse>> => {
  return get('/announcements', { params })
}

export const getAnnouncement = async (id: number): Promise<ApiResponse<Announcement>> => {
  return get(`/announcements/${id}`)
}
