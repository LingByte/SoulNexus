import dayjs from 'dayjs'

/** Format API ISO timestamps for admin tables (locale-aware via dayjs locale). */
export function formatDateTime(value?: string | null): string {
  if (!value || !String(value).trim()) return '—'
  const d = dayjs(value)
  if (!d.isValid()) return String(value)
  return d.format('YYYY-MM-DD HH:mm:ss')
}
