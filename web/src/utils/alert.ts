/**
 * 统一通知系统 - showAlert
 * 用于替代 react-hot-toast / ToastContainer
 * 使用 ArcoDesign Message
 */
import { Message } from '@arco-design/web-react/es'

export type AlertType = 'success' | 'error' | 'warning' | 'info'

interface AlertOptions {
    duration?: number
    closable?: boolean
}

/**
 * 统一通知接口
 * @param message 消息文本
 * @param type 类型：success / error / warning / info
 * @param title 标题（可选）
 * @param options 选项
 */
export function showAlert(
    message: string,
    type: AlertType = 'info',
    _title?: string,
    options?: AlertOptions,
) {
    const duration = options?.duration ?? 3000

    const msgConfig = {
        content: message,
        duration,
        closable: options?.closable ?? true,
    }

    switch (type) {
        case 'success':
            Message.success(msgConfig)
            break
        case 'error':
            Message.error(msgConfig)
            break
        case 'warning':
            Message.warning(msgConfig)
            break
        case 'info':
        default:
            Message.info(msgConfig)
            break
    }
}

/** 便捷方法 */
export const alert = {
    success: (message: string, title?: string, duration?: number) =>
        showAlert(message, 'success', title, { duration }),
    error: (message: string, title?: string, duration?: number) =>
        showAlert(message, 'error', title, { duration }),
    warning: (message: string, title?: string, duration?: number) =>
        showAlert(message, 'warning', title, { duration }),
    info: (message: string, title?: string, duration?: number) =>
        showAlert(message, 'info', title, { duration }),
}
