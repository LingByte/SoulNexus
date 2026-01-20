import { create } from 'zustand'
import { 
  getUnreadNotificationCount, 
  getNotifications, 
  markAllNotificationsAsRead, 
  markNotificationAsRead, 
  deleteNotification,
  batchDeleteNotifications,
  getAllNotificationIds,
  type Notification,
} from '../api/notification'

interface NotificationState {
  unreadCount: number
  notifications: Notification[]
  isLoading: boolean
  isUnreadCountLoading: boolean
  total: number
  currentPage: number
  pageSize: number
  totalPages: number
  totalUnread: number
  totalRead: number
  
  // Actions
  fetchUnreadCount: () => Promise<void>
  fetchNotifications: (params?: { 
    page?: number; 
    size?: number; 
    filter?: 'all' | 'read' | 'unread';
    title?: string;
    content?: string;
    start_time?: string;
    end_time?: string;
  }) => Promise<void>
  markAllAsRead: () => Promise<void>
  markAsRead: (id: string) => Promise<void>
  deleteNotification: (id: string) => Promise<void>
  batchDeleteNotifications: (ids: number[]) => Promise<void>
  getAllNotificationIds: (params?: {
    filter?: 'all' | 'read' | 'unread'
    title?: string
    content?: string
    start_time?: string
    end_time?: string
  }) => Promise<number[]>
  setUnreadCount: (count: number) => void
  clearNotifications: () => void
  addNotification: (notification: { type: 'success' | 'error' | 'warning' | 'info'; title: string; message: string }) => void
}

export const useNotificationStore = create<NotificationState>((set, get) => ({
  unreadCount: 0,
  notifications: [],
  isLoading: false,
  isUnreadCountLoading: false,
  total: 0,
  currentPage: 1,
  pageSize: 10,
  totalPages: 0,
  totalUnread: 0,
  totalRead: 0,

  fetchUnreadCount: async () => {
    set({ isUnreadCountLoading: true })
    try {
      const response = await getUnreadNotificationCount()
      if (response.code === 200) {
        set({ unreadCount: response.data })
      }
    } catch (error) {
      console.error('Failed to fetch unread count:', error)
    } finally {
      set({ isUnreadCountLoading: false })
    }
  },

  fetchNotifications: async (params = {}) => {
    set({ isLoading: true })
    try {
      const response = await getNotifications(params)
      if (response.code === 200) {
        const { list, total, totalUnread, totalRead, page, size } = response.data
        set({ 
          notifications: list || [],
          total: total || 0,
          totalUnread: totalUnread || 0,
          totalRead: totalRead || 0,
          currentPage: page,
          pageSize: size,
          totalPages: Math.ceil((total || 0) / (size || 10))
        })
      }
    } catch (error) {
      console.error('Failed to fetch notifications:', error)
      set({ notifications: [], total: 0, totalUnread: 0, totalRead: 0, totalPages: 0 })
    } finally {
      set({ isLoading: false })
    }
  },

  markAllAsRead: async () => {
    try {
      const response = await markAllNotificationsAsRead()
      if (response.code === 200) {
        set({ 
          unreadCount: 0,
          notifications: get().notifications.map(n => ({ ...n, read: true }))
        })
      }
    } catch (error) {
      console.error('Failed to mark all as read:', error)
    }
  },

  markAsRead: async (id: string | number) => {
    try {
      const response = await markNotificationAsRead(id)
      if (response.code === 200) {
        const { notifications, unreadCount } = get()
        const updatedNotifications = notifications.map(n => 
          n.id === id ? { ...n, read: true } : n
        )
        const newUnreadCount = Math.max(0, unreadCount - 1)
        
        set({ 
          notifications: updatedNotifications,
          unreadCount: newUnreadCount
        })
      }
    } catch (error) {
      console.error('Failed to mark notification as read:', error)
    }
  },

  deleteNotification: async (id: string | number) => {
    try {
      const response = await deleteNotification(id)
      if (response.code === 200) {
        const { notifications, unreadCount } = get()
        const notification = notifications.find(n => n.id === id)
        const updatedNotifications = notifications.filter(n => n.id !== id)
        const newUnreadCount = notification && !notification.read 
          ? Math.max(0, unreadCount - 1) 
          : unreadCount
        
        set({ 
          notifications: updatedNotifications,
          unreadCount: newUnreadCount
        })
      }
    } catch (error) {
      console.error('Failed to delete notification:', error)
    }
  },

  batchDeleteNotifications: async (ids: number[]) => {
    try {
      const response = await batchDeleteNotifications(ids)
      if (response.code === 200) {
        const { notifications, unreadCount } = get()
        const deletedNotifications = notifications.filter(n => ids.includes(n.id))
        const unreadDeletedCount = deletedNotifications.filter(n => !n.read).length
        const updatedNotifications = notifications.filter(n => !ids.includes(n.id))
        const newUnreadCount = Math.max(0, unreadCount - unreadDeletedCount)
        
        set({ 
          notifications: updatedNotifications,
          unreadCount: newUnreadCount
        })
      }
    } catch (error) {
      console.error('Failed to batch delete notifications:', error)
    }
  },

  getAllNotificationIds: async (params = {}) => {
    try {
      const response = await getAllNotificationIds(params)
      if (response.code === 200) {
        return response.data.ids
      }
      return []
    } catch (error) {
      console.error('Failed to get all notification IDs:', error)
      return []
    }
  },

  setUnreadCount: (count: number) => {
    set({ unreadCount: count })
  },

  clearNotifications: () => {
    set({ notifications: [], unreadCount: 0 })
  },

  addNotification: (notification) => {
    // 这里可以集成toast通知库，比如react-hot-toast
    console.log('Notification:', notification)
    // 实际项目中可以调用toast库
    // toast[notification.type](notification.message, { title: notification.title })
  }
}))
