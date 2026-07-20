import { useCallback, useEffect, useMemo, useState } from 'react'
import { DatePicker, Drawer, Tag, Typography } from '@arco-design/web-react'
import dayjs from 'dayjs'
import { Button, Input, Select, TableEmpty } from '@/components/ui'
import { Loading } from '@/components/ui/loading'
import workflowService, { type ExecutionLog, type WorkflowInstance } from '@/api/workflow'
import { showAlert } from '@/utils/notification'
import { getApiMountPath } from '@/config/apiConfig'
import { getAuthToken } from '@/api/voiceSession'

const SOURCE_OPTIONS = [
  { value: 'all', label: '全部来源' },
  { value: 'manual', label: '手动运行' },
  { value: 'api:public', label: '公开 API' },
  { value: 'webhook', label: 'Webhook' },
  { value: 'schedule', label: '定时任务' },
  { value: 'event', label: '事件触发' },
]

function sourceLabel(source?: string) {
  if (!source) return '—'
  if (source === 'manual') return '手动运行'
  if (source.startsWith('api:')) return 'API'
  if (source.startsWith('webhook')) return 'Webhook'
  if (source.startsWith('schedule')) return '定时任务'
  if (source.startsWith('event')) return '事件'
  return source
}

function statusColor(status: string) {
  if (status === 'completed') return 'green'
  if (status === 'failed') return 'red'
  if (status === 'running') return 'arcoblue'
  return 'gray'
}

function fmtTime(iso?: string) {
  if (!iso) return '—'
  try {
    return new Date(iso).toLocaleString()
  } catch {
    return iso
  }
}

function metaText(meta: Record<string, unknown> | undefined, key: string) {
  const v = meta?.[key]
  if (typeof v === 'string' && v.trim()) return v.trim()
  return '—'
}

export type WorkflowInvocationLogsPanelProps = {
  definitionId: number
  definitionName: string
}

export default function WorkflowInvocationLogsPanel({
  definitionId,
  definitionName,
}: WorkflowInvocationLogsPanelProps) {
  const [rows, setRows] = useState<WorkflowInstance[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [source, setSource] = useState('all')
  const [keyword, setKeyword] = useState('')
  const [dateRange, setDateRange] = useState<[string, string]>(() => {
    const end = dayjs()
    const start = end.subtract(6, 'day')
    return [start.startOf('day').toISOString(), end.endOf('day').toISOString()]
  })
  const [detail, setDetail] = useState<WorkflowInstance | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const pageSize = 20

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await workflowService.listInstances({
        definitionId,
        source: source === 'all' ? undefined : source,
        keyword: keyword.trim() || undefined,
        from: dateRange[0],
        to: dateRange[1],
        page,
        pageSize,
      })
      if (res.code === 200 && res.data) {
        setRows(res.data.list || [])
        setTotal(res.data.total || 0)
      } else {
        showAlert(res.msg || '加载调用日志失败', 'error')
      }
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || '加载调用日志失败', 'error')
    } finally {
      setLoading(false)
    }
  }, [definitionId, source, keyword, dateRange, page])

  useEffect(() => {
    void load()
  }, [load])

  const openDetail = async (row: WorkflowInstance) => {
    setDetail(row)
    if (row.executionLogs && row.executionLogs.length > 0) return
    setDetailLoading(true)
    try {
      const res = await workflowService.getInstance(row.id)
      if (res.code === 200 && res.data) setDetail(res.data)
    } finally {
      setDetailLoading(false)
    }
  }

  const exportCsv = () => {
    const mount = getApiMountPath().replace(/\/$/, '')
    const url = `${mount}${workflowService.exportInstancesUrl({
      definitionId,
      source: source === 'all' ? undefined : source,
      keyword: keyword.trim() || undefined,
      from: dateRange[0],
      to: dateRange[1],
    })}`
    const token = getAuthToken()
    const a = document.createElement('a')
    a.href = token ? `${url}${url.includes('?') ? '&' : '?'}token=${encodeURIComponent(token)}` : url
    a.download = `workflow-${definitionId}-logs.csv`
    a.click()
  }

  const columns = useMemo(
    () => [
      { key: 'source', title: '来源', width: 120, render: (_: unknown, row: WorkflowInstance) => sourceLabel(row.triggerSource) },
      { key: 'ip', title: 'IP', width: 130, render: (_: unknown, row: WorkflowInstance) => metaText(row.clientMeta, 'ip') },
      { key: 'ua', title: 'UA', width: 200, render: (_: unknown, row: WorkflowInstance) => (
        <span className="line-clamp-2 text-xs" title={metaText(row.clientMeta, 'userAgent')}>
          {metaText(row.clientMeta, 'userAgent')}
        </span>
      ) },
      { key: 'title', title: '标题', render: (_: unknown, row: WorkflowInstance) => row.definitionName || definitionName },
      { key: 'time', title: '上次对话时间', width: 180, render: (_: unknown, row: WorkflowInstance) => fmtTime(row.completedAt || row.startedAt || row.createdAt) },
      { key: 'logs', title: '日志条数', width: 100, render: (_: unknown, row: WorkflowInstance) => row.logCount ?? 0 },
      {
        key: 'status',
        title: '状态',
        width: 100,
        render: (_: unknown, row: WorkflowInstance) => <Tag color={statusColor(row.status)}>{row.status}</Tag>,
      },
      {
        key: 'action',
        title: '操作',
        width: 80,
        render: (_: unknown, row: WorkflowInstance) => (
          <Button type="text" size="mini" onClick={() => void openDetail(row)}>
            详情
          </Button>
        ),
      },
    ],
    [definitionName],
  )

  return (
    <div className="flex h-full min-h-0 w-full flex-1 flex-col bg-white dark:bg-gray-900">
      <div className="border-b border-gray-200 px-4 py-3 dark:border-gray-800">
        <Typography.Title heading={6} style={{ margin: 0 }}>
          调用日志
        </Typography.Title>
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          查看工作流运行记录、执行日志与导出
        </Typography.Text>
      </div>

      <div className="flex flex-wrap items-center gap-3 border-b border-gray-200 px-4 py-3 dark:border-gray-800">
        <Select
          value={source}
          onChange={setSource}
          options={SOURCE_OPTIONS}
          style={{ width: 140 }}
          size="small"
        />
        <DatePicker.RangePicker
          size="small"
          style={{ width: 260 }}
          value={[dayjs(dateRange[0]), dayjs(dateRange[1])]}
          onChange={(_, ds) => {
            if (!ds || ds.length < 2 || !ds[0] || !ds[1]) return
            setPage(1)
            setDateRange([dayjs(ds[0]).startOf('day').toISOString(), dayjs(ds[1]).endOf('day').toISOString()])
          }}
        />
        <Input
          placeholder="搜索标题或实例 ID"
          value={keyword}
          onChange={setKeyword}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              setPage(1)
              void load()
            }
          }}
          style={{ width: 220 }}
          size="small"
        />
        <div className="ml-auto flex gap-2">
          <Button size="small" onClick={() => { setPage(1); void load() }}>
            查询
          </Button>
          <Button type="primary" size="small" onClick={exportCsv}>
            导出
          </Button>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-auto px-4 py-3">
        {loading ? (
          <Loading block tip="加载中…" />
        ) : rows.length === 0 ? (
          <TableEmpty description="暂无调用记录" />
        ) : (
          <table className="w-full min-w-full border-collapse text-sm">
            <thead>
              <tr className="border-b border-gray-200 text-left text-muted-foreground dark:border-gray-700">
                {columns.map((col) => (
                  <th key={col.key} className="px-2 py-2 font-medium" style={{ width: col.width }}>
                    {col.title}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.id} className="border-b border-gray-100 hover:bg-gray-50 dark:border-gray-800 dark:hover:bg-gray-800/50">
                  {columns.map((col) => (
                    <td key={col.key} className="px-2 py-2.5 align-middle">
                      {col.render(null, row)}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {total > pageSize && (
        <div className="flex items-center justify-end gap-2 border-t border-gray-200 px-4 py-2 dark:border-gray-800">
          <Button size="mini" disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>
            上一页
          </Button>
          <span className="text-xs text-muted-foreground">
            {page} / {Math.ceil(total / pageSize)}
          </span>
          <Button size="mini" disabled={page * pageSize >= total} onClick={() => setPage((p) => p + 1)}>
            下一页
          </Button>
        </div>
      )}

      <Drawer
        width={520}
        title={`调用详情 #${detail?.id ?? ''}`}
        visible={!!detail}
        onCancel={() => setDetail(null)}
        footer={null}
      >
        {detailLoading ? (
          <Loading block tip="加载详情…" />
        ) : detail ? (
          <div className="space-y-4 text-sm">
            <div><span className="text-muted-foreground">来源：</span>{sourceLabel(detail.triggerSource)}</div>
            <div className="rounded-lg border border-border bg-muted/20 p-3 space-y-1.5 text-xs">
              <Typography.Text bold className="!text-xs !block mb-1">客户端信息</Typography.Text>
              <div><span className="text-muted-foreground">IP：</span>{metaText(detail.clientMeta, 'ip')}</div>
              <div className="break-all"><span className="text-muted-foreground">UA：</span>{metaText(detail.clientMeta, 'userAgent')}</div>
              <div><span className="text-muted-foreground">Referer：</span>{metaText(detail.clientMeta, 'referer')}</div>
              <div><span className="text-muted-foreground">Origin：</span>{metaText(detail.clientMeta, 'origin')}</div>
              <div><span className="text-muted-foreground">语言：</span>{metaText(detail.clientMeta, 'acceptLanguage')}</div>
              <div><span className="text-muted-foreground">X-Forwarded-For：</span>{metaText(detail.clientMeta, 'forwardedFor')}</div>
              <div><span className="text-muted-foreground">请求：</span>{metaText(detail.clientMeta, 'method')} {metaText(detail.clientMeta, 'path')}</div>
            </div>
            <div><span className="text-muted-foreground">状态：</span><Tag color={statusColor(detail.status)}>{detail.status}</Tag></div>
            <div><span className="text-muted-foreground">耗时：</span>{detail.durationMs ? `${detail.durationMs} ms` : '—'}</div>
            {detail.errorMessage && (
              <div className="rounded-lg bg-red-50 p-3 text-red-700 dark:bg-red-950/30 dark:text-red-300">{detail.errorMessage}</div>
            )}
            <div>
              <Typography.Text bold>执行日志</Typography.Text>
              <div className="mt-2 max-h-[420px] space-y-2 overflow-y-auto rounded-lg border border-border p-3 font-mono text-xs">
                {(detail.executionLogs as ExecutionLog[] | undefined)?.length ? (
                  (detail.executionLogs as ExecutionLog[]).map((log, i) => (
                    <div key={`${log.timestamp}-${i}`} className="whitespace-pre-wrap">
                      <span className="text-muted-foreground">[{log.timestamp}]</span>{' '}
                      <span className={log.level === 'error' ? 'text-red-500' : log.level === 'success' ? 'text-green-600' : ''}>
                        {log.level}
                      </span>{' '}
                      {log.nodeName ? `[${log.nodeName}] ` : ''}{log.message}
                    </div>
                  ))
                ) : (
                  <span className="text-muted-foreground">无日志</span>
                )}
              </div>
            </div>
          </div>
        ) : null}
      </Drawer>
    </div>
  )
}
