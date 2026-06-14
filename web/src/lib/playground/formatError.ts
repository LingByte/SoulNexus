/** 将 API 返回的 JSON / 纯文本错误整理为可读摘要 */
export function formatApiError(raw: string): {
  title?: string
  detail: string
  raw: string
} {
  const trimmed = raw.trim()
  if (!trimmed) return { detail: '未知错误', raw: trimmed }

  try {
    const j = JSON.parse(trimmed) as {
      error?: { message?: string; type?: string; code?: string | number }
      message?: string
    }
    const err = j.error ?? j
    const message =
      (typeof err === 'object' && err && 'message' in err && typeof err.message === 'string'
        ? err.message
        : undefined) ?? (typeof j.message === 'string' ? j.message : undefined)

    if (message) {
      const type =
        typeof err === 'object' && err && 'type' in err && typeof err.type === 'string'
          ? err.type
          : undefined
      const code =
        typeof err === 'object' && err && 'code' in err ? String(err.code) : undefined
      const title = [type, code].filter(Boolean).join(' · ')
      return { title: title || undefined, detail: message, raw: trimmed }
    }
  } catch {
    /* 非 JSON */
  }

  return { detail: trimmed, raw: trimmed }
}
