/** 智能体 icon 字段存的是可访问的图片 URL（含本站相对路径） */
export function isAssistantIconURL(icon?: string | null): boolean {
  const s = (icon || '').trim()
  if (!s) return false
  return (
    s.startsWith('http://') ||
    s.startsWith('https://') ||
    s.startsWith('//') ||
    s.startsWith('data:') ||
    s.startsWith('/')
  )
}

/**
 * 仅用于自定义 URL 图：<img /> 上追加。
 * 上传图常为浅色/白底，亮色界面不要「白花花一块」；暗色下保持原图色彩。
 */
export const ASSISTANT_URL_ICON_IMG_CLASS_FULL =
  'h-full w-full object-contain p-1.5 brightness-0 dark:brightness-100'

export const ASSISTANT_URL_ICON_IMG_CLASS_SM =
  'h-7 w-7 object-contain brightness-0 dark:brightness-100'
