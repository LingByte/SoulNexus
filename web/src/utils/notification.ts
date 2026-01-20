import { useUIStore } from '../stores/uiStore'
import { playSuccessSound, playErrorSound, playWarningSound, playInfoSound } from './audioEffects'
import type { ToastPosition } from '../stores/uiStore'

// 通知类型
export type NotificationType = 'success' | 'error' | 'warning' | 'info'

// 通知选项
export interface NotificationOptions {
  title: string
  message?: string
  duration?: number
  position?: ToastPosition
  scrollToPosition?: {
    x: number
    y: number
    behavior?: 'smooth' | 'instant' | 'auto'
  }
}

// 创建通知的工具函数
export const createNotification = () => {
  const { addNotification } = useUIStore.getState()
  
  return {
    success: (options: NotificationOptions) => {
      playSuccessSound()
      addNotification({
        type: 'success',
        title: options.title,
        message: options.message,
        duration: options.duration || 5000,
        position: options.position,
        scrollToPosition: options.scrollToPosition
      })
    },
    
    error: (options: NotificationOptions) => {
      playErrorSound()
      addNotification({
        type: 'error',
        title: options.title,
        message: options.message,
        duration: options.duration || 7000,
        position: options.position,
        scrollToPosition: options.scrollToPosition
      })
    },
    
    warning: (options: NotificationOptions) => {
      playWarningSound()
      addNotification({
        type: 'warning',
        title: options.title,
        message: options.message,
        duration: options.duration || 6000,
        position: options.position,
        scrollToPosition: options.scrollToPosition
      })
    },
    
    info: (options: NotificationOptions) => {
      playInfoSound()
      addNotification({
        type: 'info',
        title: options.title,
        message: options.message,
        duration: options.duration || 5000,
        position: options.position,
        scrollToPosition: options.scrollToPosition
      })
    }
  }
}

// 替换alert的便捷函数
export const showAlert = (
  message: string, 
  type: NotificationType = 'info', 
  title?: string,
  options?: Partial<NotificationOptions>
) => {
  const notification = createNotification()
  
  notification[type]({
    title: title || (type === 'error' ? '错误' : type === 'warning' ? '警告' : type === 'success' ? '成功' : '提示'),
    message: message,
    ...options
  })
}

// 带滚动功能的提示
export const showAlertWithScroll = (
  message: string, 
  type: NotificationType = 'info', 
  title?: string,
  scrollToPosition?: { x: number; y: number; behavior?: 'smooth' | 'instant' | 'auto' }
) => {
  const notification = createNotification()
  
  // 先滚动到指定位置
  if (scrollToPosition) {
    window.scrollTo({
      left: scrollToPosition.x,
      top: scrollToPosition.y,
      behavior: scrollToPosition.behavior || 'smooth'
    })
  }
  
  // 然后显示通知
  notification[type]({
    title: title || (type === 'error' ? '错误' : type === 'warning' ? '警告' : type === 'success' ? '成功' : '提示'),
    message: message,
    duration: type === 'error' ? 7000 : 5000
  })
}

// 确认对话框的替代函数
export const showConfirm = (
  message: string,
  title: string = '确认',
  onConfirm: () => void,
  onCancel?: () => void
) => {
  const confirmed = window.confirm(`${title}\n\n${message}`)
  if (confirmed) {
    onConfirm()
  } else if (onCancel) {
    onCancel()
  }
}

// 输入对话框的替代函数
export const showPrompt = (
  message: string,
  title: string = '输入',
  defaultValue: string = ''
): string | null => {
  return window.prompt(`${title}\n\n${message}`, defaultValue)
}

// 导出默认的通知实例
export const notification = createNotification()
