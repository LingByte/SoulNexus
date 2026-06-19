import type { ApiResponse } from '@/utils/request'
import { showAlert } from '@/utils/alert'

export function formatApiError(
  res: Pick<ApiResponse<unknown>, 'msg' | 'data'> | { msg?: string; data?: unknown },
  fallback = '请求失败',
): string {
  const violations = (res.data as { violations?: string[] } | null | undefined)?.violations
  const base = res.msg?.trim() || fallback
  if (violations?.length) return `${base}：${violations.join('；')}`
  return base
}

export interface NotifyApiOptions {
  /** Override success message; defaults to res.msg */
  successMessage?: string
  /** When true, do not toast on success */
  silentSuccess?: boolean
  /** When true, do not toast on failure */
  silentError?: boolean
}

/** Show Arco toast for API result. Returns true when code === 200. */
export function notifyApiResult<T>(
  res: ApiResponse<T>,
  opts?: NotifyApiOptions,
): res is ApiResponse<T> & { code: 200 } {
  if (res.code === 200) {
    if (!opts?.silentSuccess) {
      showAlert(opts?.successMessage || res.msg || '操作成功', 'success', undefined, { duration: 4000 })
    }
    return true
  }
  if (!opts?.silentError) {
    showAlert(formatApiError(res), 'error', undefined, { duration: 6000 })
  }
  return false
}

export function notifyApiError(err: unknown, fallback = '请求失败'): void {
  const e = err as { msg?: string; data?: unknown }
  showAlert(formatApiError(e, e.msg || fallback), 'error', undefined, { duration: 6000 })
}
