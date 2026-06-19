/**
 * 统一通知 — 使用 Arco Design Message（全局可见）
 */
import { showAlert as showArcoAlert, type AlertType } from './alert'

export type NotificationType = AlertType

export interface NotificationOptions {
  title?: string
  message?: string
  duration?: number
}

/** @deprecated use title as prefix; renders via Arco Message */
export const showAlert = (
  message: string,
  type: NotificationType = 'info',
  title?: string,
  options?: { duration?: number },
) => {
  const text = title && title !== message ? `${title}：${message}` : message
  showArcoAlert(text, type, title, options)
}

// Legacy exports — prefer @/utils/alert directly
export { showAlert as alertShow } from './alert'
