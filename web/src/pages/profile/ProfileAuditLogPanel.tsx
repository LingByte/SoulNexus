import { useCallback, useEffect, useMemo, useState } from 'react'
import { FileText, Search, RotateCcw, ChevronLeft, ChevronRight } from 'lucide-react'
import { getUserActivity, type ActivityLog } from '@/api/profile'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/UI/Select'
import LoadingAnimation from '@/components/Animations/LoadingAnimation'

function formatDateInput(d: Date) {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

const SERVICE_OPTIONS: { value: string; label: string }[] = [
  { value: 'all', label: '全部服务' },
  { value: 'account', label: 'Account（认证与账号）' },
  { value: 'credential', label: 'Credential（API 密钥）' },
  { value: 'assistant', label: 'Assistant' },
  { value: 'group', label: 'Group' },
  { value: 'chat', label: 'Chat' },
  { value: 'voice', label: 'Voice' },
  { value: 'notification', label: 'Notification' },
  { value: 'upload', label: 'Upload' },
  { value: 'billing', label: 'Billing / Quota' },
  { value: 'knowledge', label: 'Knowledge' },
  { value: 'workflow', label: 'Workflow' },
  { value: 'other', label: 'Other' },
]

type FilterState = {
  start: string
  end: string
  service: string
  eventKeyword: string
  operatorId: string
  credentialId: string
  resource: string
  method: string
}

type ProfileAuditLogPanelProps = {
  userId?: number
}

const ProfileAuditLogPanel = ({ userId }: ProfileAuditLogPanelProps) => {
  const defaultEnd = useMemo(() => new Date(), [])
  const defaultStart = useMemo(() => {
    const d = new Date()
    d.setDate(d.getDate() - 180)
    return d
  }, [])

  const initialFilters = useMemo(
    (): FilterState => ({
      start: formatDateInput(defaultStart),
      end: formatDateInput(defaultEnd),
      service: 'all',
      eventKeyword: '',
      operatorId: userId != null ? String(userId) : '',
      credentialId: '',
      resource: '',
      method: '',
    }),
    [defaultStart, defaultEnd, userId],
  )

  const [draft, setDraft] = useState<FilterState>(initialFilters)
  const [applied, setApplied] = useState<FilterState>(initialFilters)
  const [page, setPage] = useState(1)
  const [limit] = useState(20)
  const [total, setTotal] = useState(0)
  const [totalPages, setTotalPages] = useState(1)
  const [retentionDays, setRetentionDays] = useState(180)
  const [rows, setRows] = useState<ActivityLog[]>([])
  const [loading, setLoading] = useState(false)
  const [detail, setDetail] = useState<ActivityLog | null>(null)

  useEffect(() => {
    if (userId != null) {
      setDraft((d) => ({ ...d, operatorId: String(userId) }))
      setApplied((a) => ({ ...a, operatorId: String(userId) }))
    }
  }, [userId])

  const fetchList = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getUserActivity({
        page,
        limit,
        start: applied.start || undefined,
        end: applied.end || undefined,
        service: applied.service === 'all' ? undefined : applied.service,
        event: applied.eventKeyword.trim() || undefined,
        operatorId: applied.operatorId.trim() || undefined,
        credentialId: applied.credentialId.trim() || undefined,
        resource: applied.resource.trim() || undefined,
        action: applied.method || undefined,
      })
      if (res.code === 200 && res.data) {
        setRows(res.data.activities || [])
        setTotal(res.data.pagination?.total ?? 0)
        setTotalPages(Math.max(1, res.data.pagination?.totalPages ?? 1))
        if (typeof (res.data as { retentionDays?: number }).retentionDays === 'number') {
          setRetentionDays((res.data as { retentionDays: number }).retentionDays)
        }
      } else {
        setRows([])
        setTotal(0)
        setTotalPages(1)
      }
    } finally {
      setLoading(false)
    }
  }, [page, limit, applied])

  useEffect(() => {
    void fetchList()
  }, [fetchList])

  const resetFilters = () => {
    setDraft(initialFilters)
    setApplied(initialFilters)
    setPage(1)
  }

  const submitQuery = () => {
    setApplied({ ...draft })
    setPage(1)
  }

  const displayEventName = (row: ActivityLog) =>
    (row.details && row.details.trim()) || row.eventCode || row.target || '—'

  return (
    <div className="space-y-6">
      <div>
        <div className="flex flex-wrap items-baseline gap-3">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-gray-100">审计日志</h2>
          <span className="text-sm text-slate-500 dark:text-gray-400">日志查询</span>
        </div>
        <p className="mt-2 text-sm leading-relaxed text-slate-600 dark:text-gray-400">
          以下列表包括了近 {retentionDays} 天您在本账号中进行的操作，详情见{' '}
          <a href="/terms" className="text-sky-600 hover:underline dark:text-sky-400">
            说明文档
          </a>
          。
        </p>
      </div>

      <div className="rounded-xl border border-slate-200 bg-slate-50/80 p-4 dark:border-neutral-800 dark:bg-neutral-900/50">
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <div>
            <label className="mb-1 block text-xs font-medium text-slate-600 dark:text-gray-400">事件时间（起）</label>
            <Input type="date" value={draft.start} onChange={(e) => setDraft((d) => ({ ...d, start: e.target.value }))} className="h-9" />
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-slate-600 dark:text-gray-400">事件时间（止）</label>
            <Input type="date" value={draft.end} onChange={(e) => setDraft((d) => ({ ...d, end: e.target.value }))} className="h-9" />
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-slate-600 dark:text-gray-400">服务名称</label>
            <Select value={draft.service} onValueChange={(v) => setDraft((d) => ({ ...d, service: v }))} className="w-full">
              <SelectTrigger className="h-9">
                <SelectValue placeholder="请选择服务" />
              </SelectTrigger>
              <SelectContent searchable searchPlaceholder="搜索服务...">
                {SERVICE_OPTIONS.map((o) => (
                  <SelectItem key={o.value} value={o.value}>
                    {o.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
        <div className="mt-4 flex flex-wrap gap-2">
          <Button
            variant="primary"
            size="sm"
            onClick={submitQuery}
            disabled={loading}
            leftIcon={<Search className="h-4 w-4" />}
          >
            查询
          </Button>
          <Button variant="outline" size="sm" onClick={resetFilters} leftIcon={<RotateCcw className="h-4 w-4" />}>
            重置
          </Button>
          <span className="ml-auto text-xs text-slate-500 dark:text-gray-500 self-center">共 {total} 条</span>
        </div>
      </div>

      <div className="overflow-hidden rounded-xl border border-slate-200 bg-white dark:border-neutral-800 dark:bg-neutral-950">
        {loading ? (
          <div className="flex items-center justify-center gap-2 py-20">
            <LoadingAnimation type="spinner" size="md" />
            <span className="text-sm text-slate-600 dark:text-gray-400">加载中…</span>
          </div>
        ) : rows.length === 0 ? (
          <div className="py-16 text-center text-sm text-slate-500 dark:text-gray-400">暂无记录</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[880px] text-left text-sm">
              <thead className="border-b border-slate-200 bg-slate-50 text-xs font-semibold uppercase tracking-wide text-slate-600 dark:border-neutral-800 dark:bg-neutral-900 dark:text-gray-400">
                <tr>
                  <th className="px-4 py-3">事件时间</th>
                  <th className="px-4 py-3">操作者 ID</th>
                  <th className="px-4 py-3">源 IP</th>
                  <th className="px-4 py-3">服务名称</th>
                  <th className="px-4 py-3">事件名称</th>
                  <th className="px-4 py-3">资源名称</th>
                  <th className="px-4 py-3 text-right">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
                {rows.map((row) => (
                  <tr key={row.id} className="hover:bg-slate-50/80 dark:hover:bg-neutral-900/60">
                    <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-slate-800 dark:text-gray-200">
                      {row.createdAt ? new Date(row.createdAt).toLocaleString('zh-CN') : '—'}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-slate-700 dark:text-gray-300">{row.userId ?? '—'}</td>
                    <td className="px-4 py-3 font-mono text-xs text-slate-700 dark:text-gray-300">{row.ipAddress || '—'}</td>
                    <td className="px-4 py-3 text-slate-800 dark:text-gray-200">{row.serviceName || '—'}</td>
                    <td className="max-w-[220px] truncate px-4 py-3 text-slate-800 dark:text-gray-200" title={displayEventName(row)}>
                      {displayEventName(row)}
                    </td>
                    <td className="max-w-[260px] truncate px-4 py-3 text-xs text-slate-600 dark:text-gray-400" title={row.resourceSummary || row.target}>
                      {row.resourceSummary || row.target || '—'}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <button
                        type="button"
                        className="text-sky-600 text-xs font-medium hover:underline dark:text-sky-400"
                        onClick={() => setDetail(row)}
                      >
                        查看详情
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {totalPages > 1 && (
          <div className="flex items-center justify-center gap-2 border-t border-slate-100 px-4 py-3 dark:border-neutral-800">
            <Button
              variant="outline"
              size="sm"
              disabled={page <= 1 || loading}
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              leftIcon={<ChevronLeft className="h-4 w-4" />}
            >
              上一页
            </Button>
            <span className="text-xs text-slate-500 dark:text-gray-400">
              第 {page} / {totalPages} 页
            </span>
            <Button
              variant="outline"
              size="sm"
              disabled={page >= totalPages || loading}
              onClick={() => setPage((p) => p + 1)}
              rightIcon={<ChevronRight className="h-4 w-4" />}
            >
              下一页
            </Button>
          </div>
        )}
      </div>

      {detail && (
        <div className="fixed inset-0 z-[200] flex items-center justify-center p-4">
          <button
            type="button"
            className="absolute inset-0 bg-black/50"
            aria-label="关闭"
            onClick={() => setDetail(null)}
          />
          <div className="relative max-h-[85vh] w-full max-w-2xl overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xl dark:border-neutral-700 dark:bg-neutral-900">
            <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-neutral-800">
              <div className="flex items-center gap-2 text-sm font-semibold text-slate-900 dark:text-gray-100">
                <FileText className="h-4 w-4" />
                事件详情
              </div>
              <button
                type="button"
                className="rounded p-1 text-slate-500 hover:bg-slate-100 dark:hover:bg-neutral-800"
                onClick={() => setDetail(null)}
              >
                ✕
              </button>
            </div>
            <pre className="max-h-[calc(85vh-52px)] overflow-auto p-4 text-xs leading-relaxed text-slate-800 dark:text-gray-200">
              {JSON.stringify(detail, null, 2)}
            </pre>
          </div>
        </div>
      )}
    </div>
  )
}

export default ProfileAuditLogPanel
