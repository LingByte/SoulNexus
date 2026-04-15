import { create } from 'zustand'

// 通知类型定义（可以根据实际需求修改）
export interface Notification {
  id: string | number
  title: string
  content?: string
  type?: 'success' | 'error' | 'warning' | 'info'
  read: boolean
  createdAt?: string
  [key: string]: any
}

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
  markAsRead: (id: string | number) => Promise<void>
  deleteNotification: (id: string | number) => Promise<void>
  batchDeleteNotifications: (ids: number[]) => Promise<void>
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
      // 可以在这里调用API获取未读数量
      // const response = await getUnreadNotificationCount()
      // if (response.code === 200) {
      //   set({ unreadCount: response.data })
      // }
      set({ unreadCount: 0 })
    } catch (error) {
      console.error('Failed to fetch unread count:', error)
    } finally {
      set({ isUnreadCountLoading: false })
    }
  },

  fetchNotifications: async (params = {}) => {
    set({ isLoading: true })
    try {
      // 可以在这里调用API获取通知列表
      // const response = await getNotifications(params)
      // if (response.code === 200) {
      //   const { list, total, totalUnread, totalRead, page, size } = response.data
      //   set({ 
      //     notifications: list || [],
      //     total: total || 0,
      //     totalUnread: totalUnread || 0,
      //     totalRead: totalRead || 0,
      //     currentPage: page,
      //     pageSize: size,
      //     totalPages: Math.ceil((total || 0) / (size || 10))
      //   })
      // }
      set({ notifications: [], total: 0, totalUnread: 0, totalRead: 0, totalPages: 0 })
    } catch (error) {
      console.error('Failed to fetch notifications:', error)
      set({ notifications: [], total: 0, totalUnread: 0, totalRead: 0, totalPages: 0 })
    } finally {
      set({ isLoading: false })
    }
  },

  markAllAsRead: async () => {
    try {
      // 可以在这里调用API标记所有为已读
      // const response = await markAllNotificationsAsRead()
      // if (response.code === 200) {
      //   set({ 
      //     unreadCount: 0,
      //     notifications: get().notifications.map(n => ({ ...n, read: true }))
      //   })
      // }
      set({ 
        unreadCount: 0,
        notifications: get().notifications.map(n => ({ ...n, read: true }))
      })
    } catch (error) {
      console.error('Failed to mark all as read:', error)
    }
  },

  markAsRead: async (id: string | number) => {
    try {
      // 可以在这里调用API标记为已读
      // const response = await markNotificationAsRead(id)
      // if (response.code === 200) {
      //   const { notifications, unreadCount } = get()
      //   const updatedNotifications = notifications.map(n => 
      //     n.id === id ? { ...n, read: true } : n
      //   )
      //   const newUnreadCount = Math.max(0, unreadCount - 1)
      //   
      //   set({ 
      //     notifications: updatedNotifications,
      //     unreadCount: newUnreadCount
      //   })
      // }
      const { notifications, unreadCount } = get()
      const updatedNotifications = notifications.map(n => 
        n.id === id ? { ...n, read: true } : n
      )
      const newUnreadCount = Math.max(0, unreadCount - 1)
      
      set({ 
        notifications: updatedNotifications,
        unreadCount: newUnreadCount
      })
    } catch (error) {
      console.error('Failed to mark notification as read:', error)
    }
  },

  deleteNotification: async (id: string | number) => {
    try {
      // 可以在这里调用API删除通知
      // const response = await deleteNotification(id)
      // if (response.code === 200) {
      //   const { notifications, unreadCount } = get()
      //   const notification = notifications.find(n => n.id === id)
      //   const updatedNotifications = notifications.filter(n => n.id !== id)
      //   const newUnreadCount = notification && !notification.read 
      //     ? Math.max(0, unreadCount - 1) 
      //     : unreadCount
      //   
      //   set({ 
      //     notifications: updatedNotifications,
      //     unreadCount: newUnreadCount
      //   })
      // }
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
    } catch (error) {
      console.error('Failed to delete notification:', error)
    }
  },

  batchDeleteNotifications: async (ids: number[]) => {
    try {
      // 可以在这里调用API批量删除通知
      // const response = await batchDeleteNotifications(ids)
      // if (response.code === 200) {
      //   const { notifications, unreadCount } = get()
      //   const deletedNotifications = notifications.filter(n => ids.includes(n.id))
      //   const unreadDeletedCount = deletedNotifications.filter(n => !n.read).length
      //   const updatedNotifications = notifications.filter(n => !ids.includes(n.id))
      //   const newUnreadCount = Math.max(0, unreadCount - unreadDeletedCount)
      //   
      //   set({ 
      //     notifications: updatedNotifications,
      //     unreadCount: newUnreadCount
      //   })
      // }
      const { notifications, unreadCount } = get()
      const deletedNotifications = notifications.filter(n => ids.includes(n.id))
      const unreadDeletedCount = deletedNotifications.filter(n => !n.read).length
      const updatedNotifications = notifications.filter(n => !ids.includes(n.id))
      const newUnreadCount = Math.max(0, unreadCount - unreadDeletedCount)
      
      set({ 
        notifications: updatedNotifications,
        unreadCount: newUnreadCount
      })
    } catch (error) {
      console.error('Failed to batch delete notifications:', error)
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
