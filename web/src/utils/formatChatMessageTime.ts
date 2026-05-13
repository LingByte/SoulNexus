/**
 * 将 ISO / 后端时间字符串格式化为聊天区展示的本地时间（与站内其它列表日期风格一致）。
 */
export function formatChatMessageTime(iso: string): string {
  if (!iso || typeof iso !== 'string') return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) {
    return iso.length > 16 ? iso.slice(0, 16) : iso
  }
  return d.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  })
}
