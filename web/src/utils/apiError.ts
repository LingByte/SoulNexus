/** Shape thrown by web/src/utils/request.ts on HTTP / business errors. */
export type ApiErrorPayload = {
  code?: number
  msg?: string
  error?: string
  message?: string
}

/** Extract a user-visible message from API / validation errors. */
export function extractApiErrorMessage(err: unknown, fallback = 'Request failed'): string {
  if (typeof err === 'string' && err.trim()) return err.trim()
  if (err instanceof Error && err.message.trim()) return err.message.trim()
  if (err && typeof err === 'object') {
    const o = err as ApiErrorPayload
    if (typeof o.msg === 'string' && o.msg.trim()) return o.msg.trim()
    if (typeof o.error === 'string' && o.error.trim()) return o.error.trim()
    if (typeof o.message === 'string' && o.message.trim()) return o.message.trim()
  }
  return fallback
}
