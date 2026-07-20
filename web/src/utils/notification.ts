import { Message, Modal } from '@arco-design/web-react'

let _toastFn: ((msg: string, type?: 'success' | 'error' | 'warning' | 'info', duration?: number) => void) | null = null

export function setToastFn(fn: typeof _toastFn) {
  _toastFn = fn
}

export type NotificationType = 'success' | 'error' | 'warning' | 'info'

export interface NotificationOptions {
  title?: string
  message?: string
  duration?: number
}

export const showAlert = (
  message: string,
  type: NotificationType = 'info',
  title?: string,
  options?: Partial<NotificationOptions>
) => {
  const duration = options?.duration
  if (_toastFn) {
    _toastFn(message, type, duration)
    return
  }
  const head =
    title ||
    (type === 'error' ? '错误' : type === 'warning' ? '警告' : type === 'success' ? '成功' : '提示')
  const content = `${head}: ${message}`
  switch (type) {
    case 'success':
      Message.success({ content, duration })
      break
    case 'error':
      Message.error({ content, duration })
      break
    case 'warning':
      Message.warning({ content, duration })
      break
    default:
      Message.info({ content, duration })
  }
}

/** In-app confirm via Arco Modal (not browser window.confirm). */
export const showConfirm = (
  message: string,
  title: string = '确认',
  onConfirm: () => void,
  onCancel?: () => void,
) => {
  Modal.confirm({
    title,
    content: message,
    okText: '确定',
    cancelText: '取消',
    onOk: () => {
      onConfirm()
    },
    onCancel: () => {
      onCancel?.()
    },
  })
}

/** Promise wrapper for async flows (route blockers, leave guards). */
export function confirmAsync(message: string, title = '确认'): Promise<boolean> {
  return new Promise((resolve) => {
    Modal.confirm({
      title,
      content: message,
      okText: '确定',
      cancelText: '取消',
      onOk: () => resolve(true),
      onCancel: () => resolve(false),
    })
  })
}

export const showPrompt = (
  message: string,
  title: string = '输入',
  defaultValue: string = ''
): string | null => window.prompt(`${title}\n\n${message}`, defaultValue)
