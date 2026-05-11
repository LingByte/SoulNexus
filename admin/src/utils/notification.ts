// Bridge to Arco's Notification so existing call sites (`showAlert`, `notification.x`, etc.) keep working
// without the deleted PositionedToast / framer-motion stack.
import { Notification } from '@arco-design/web-react'

export type NotificationType = 'success' | 'error' | 'warning' | 'info'

export interface NotificationOptions {
  title: string
  message?: string
  duration?: number
}

const open = (type: NotificationType, opts: NotificationOptions) => {
  Notification[type]({
    title: opts.title,
    content: opts.message ?? '',
    duration: opts.duration ?? (type === 'error' ? 7000 : 5000),
  })
}

export const createNotification = () => ({
  success: (o: NotificationOptions) => open('success', o),
  error: (o: NotificationOptions) => open('error', o),
  warning: (o: NotificationOptions) => open('warning', o),
  info: (o: NotificationOptions) => open('info', o),
})

const defaultTitle: Record<NotificationType, string> = {
  success: '成功',
  error: '错误',
  warning: '警告',
  info: '提示',
}

export const showAlert = (
  message: string,
  type: NotificationType = 'info',
  title?: string,
  options?: Partial<NotificationOptions>,
) => {
  open(type, {
    title: title || defaultTitle[type],
    message,
    ...options,
  })
}

export const showAlertWithScroll = (
  message: string,
  type: NotificationType = 'info',
  title?: string,
  scrollToPosition?: { x: number; y: number; behavior?: ScrollBehavior },
) => {
  if (scrollToPosition) {
    window.scrollTo({
      left: scrollToPosition.x,
      top: scrollToPosition.y,
      behavior: scrollToPosition.behavior || 'smooth',
    })
  }
  open(type, {
    title: title || defaultTitle[type],
    message,
  })
}

export const showConfirm = (
  message: string,
  title: string = '确认',
  onConfirm: () => void,
  onCancel?: () => void,
) => {
  const ok = window.confirm(`${title}\n\n${message}`)
  if (ok) onConfirm()
  else onCancel?.()
}

export const showPrompt = (
  message: string,
  title: string = '输入',
  defaultValue: string = '',
): string | null => window.prompt(`${title}\n\n${message}`, defaultValue)

export const notification = createNotification()
