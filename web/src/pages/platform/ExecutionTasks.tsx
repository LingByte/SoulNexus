import { useCallback, useEffect, useState } from 'react'
import { Drawer, Modal, Tag, Progress } from '@arco-design/web-react'
import { RefreshCw, Eye, Search, Ban, RotateCcw, Trash2 } from 'lucide-react'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Button, DataList, Input, Select } from '@/components/ui'
import type { DataListColumn } from '@/components/ui'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useTranslation } from '@/i18n'
import { formatDateTime } from '@/utils/formatDateTime'
import {
  listExecutionTasks, getExecutionTaskStats, getExecutionTask,
  cancelExecutionTask, retryExecutionTask, deleteExecutionTask,
  type ExecutionTask, type ExecutionTaskStats,
} from '@/api/executionTasks'

const STATUS_OPTIONS = [
  { value: '', labelKey: 'executionTasks.statusAll' },
  { value: 'pending', labelKey: 'executionTasks.statusPending' },
  { value: 'running', labelKey: 'executionTasks.statusRunning' },
  { value: 'success', labelKey: 'executionTasks.statusSuccess' },
  { value: 'failed', labelKey: 'executionTasks.statusFailed' },
  { value: 'canceled', labelKey: 'executionTasks.statusCanceled' },
]

const statusColor = (s?: string) => {
  const v = (s || '').toLowerCase()
  if (v === 'pending') return 'arcoblue'
  if (v === 'running') return 'orange'
  if (v === 'success') return 'green'
  if (v === 'failed') return 'red'
  return 'gray'
}

export default function ExecutionTasksPage() {
  const { t } = useTranslation()
  const statusLabel = (s: string) => {
    const map: Record<string, string> = {
      pending: t('executionTasks.statusPending'), running: t('executionTasks.statusRunning'),
      success: t('executionTasks.statusSuccess'), failed: t('executionTasks.statusFailed'),
      canceled: t('executionTasks.statusCanceled'),
    }
    return map[s] || s || '-'
  }

  const [list, setList] = useState<ExecutionTask[]>([])
  const [total, setTotal] = useState(0)
  const [stats, setStats] = useState<ExecutionTaskStats | null>(null)
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [status, setStatus] = useState('')
  const [queueName, setQueueName] = useState('')
  const [search, setSearch] = useState('')
  const [detail, setDetail] = useState<ExecutionTask | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [actionId, setActionId] = useState<string | null>(null)

  const fetchList = useCallback(async () => {
    setLoading(true)
    try {
      const [res, st] = await Promise.all([
        listExecutionTasks({ page, pageSize: 20, status: status || undefined, queueName: queueName.trim() || undefined, search: search.trim() || undefined }),
        getExecutionTaskStats(queueName.trim() || undefined),
      ])
      setList(res.list || []); setTotal(res.total || 0); setStats(st)
    } catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('executionTasks.loadFailed')), 'error') }
    finally { setLoading(false) }
  }, [page, status, queueName, search, t])

  useEffect(() => { void fetchList() }, [fetchList])

  const openDetail = async (id: string) => {
    setDetailLoading(true)
    try { setDetail(await getExecutionTask(id)) }
    catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('executionTasks.detailFailed')), 'error') }
    finally { setDetailLoading(false) }
  }

  const handleCancel = (row: ExecutionTask) => {
    Modal.confirm({
      title: t('executionTasks.cancelConfirmTitle'), content: t('executionTasks.cancelConfirmBody', { taskId: row.taskId }),
      onOk: async () => {
        setActionId(row.id)
        try { await cancelExecutionTask(row.id); showAlert(t('executionTasks.cancelOk'), 'success'); fetchList(); if (detail?.id === row.id) setDetail(null) }
        catch (e: unknown) { showAlert(extractApiErrorMessage(e), 'error') }
        finally { setActionId(null) }
      },
    })
  }

  const handleRetry = (row: ExecutionTask) => {
    Modal.confirm({
      title: t('executionTasks.retryConfirmTitle'), content: t('executionTasks.retryConfirmBody', { taskId: row.taskId }),
      onOk: async () => {
        setActionId(row.id)
        try { await retryExecutionTask(row.id); showAlert(t('executionTasks.retryOk'), 'success'); fetchList() }
        catch (e: unknown) { showAlert(extractApiErrorMessage(e), 'error') }
        finally { setActionId(null) }
      },
    })
  }

  const handleDelete = (row: ExecutionTask) => {
    Modal.confirm({
      title: t('executionTasks.deleteConfirmTitle'), content: t('executionTasks.deleteConfirmBody', { taskId: row.taskId }),
      onOk: async () => {
        setActionId(row.id)
        try { await deleteExecutionTask(row.id); showAlert(t('executionTasks.deleteOk'), 'success'); fetchList(); if (detail?.id === row.id) setDetail(null) }
        catch (e: unknown) { showAlert(extractApiErrorMessage(e), 'error') }
        finally { setActionId(null) }
      },
    })
  }

  const canCancel = (s: string) => s === 'pending' || s === 'running'
  const canRetry = (s: string) => s === 'failed' || s === 'canceled'
  const canDelete = (s: string) => s !== 'running'

  const columns: DataListColumn<Record<string, unknown>>[] = [
    {
      key: 'time', title: t('executionTasks.colSubmitTime'), width: 170,
      render: (_, r) => <span className="text-sm text-neutral-700">{formatDateTime(String(r.submitTime || r.createdAt))}</span>,
    },
    {
      key: 'title', title: t('executionTasks.colTitle'), width: 200,
      render: (_, r) => <span className="truncate text-sm font-medium text-neutral-900">{String(r.title || r.taskId || '—')}</span>,
    },
    { key: 'queue', title: t('executionTasks.colQueue'), width: 180, render: (_, r) => <span className="truncate text-sm text-neutral-700">{String(r.queueName || '—')}</span> },
    { key: 'kind', title: t('executionTasks.colKind'), width: 120, render: (_, r) => <span className="truncate text-sm text-neutral-500">{String(r.kind || '—')}</span> },
    { key: 'status', title: t('executionTasks.colStatus'), width: 100, render: (_, r) => <Tag color={statusColor(String(r.status))} className="!rounded-full">{statusLabel(String(r.status))}</Tag> },
    {
      key: 'progress', title: t('executionTasks.colProgress'), width: 120,
      render: (_, r) => <Progress percent={Math.min(100, Math.max(0, Number(r.progress) || 0))} size="small" status={r.status === 'failed' ? 'error' : r.status === 'success' ? 'success' : 'normal'} />,
    },
    { key: 'priority', title: t('executionTasks.colPriority'), width: 80, render: (_, r) => <span className="text-sm text-neutral-700">{String(r.priority)}</span> },
    {
      key: 'actions', title: t('executionTasks.colActions'), width: 220, align: 'right',
      render: (_, r) => {
        const s = String(r.status)
        return (
          <div className="flex items-center justify-end gap-1">
            <Button size="mini" icon={<Eye size={12} />} onClick={() => openDetail(String(r.id))}>{t('executionTasks.actionDetail')}</Button>
            {canCancel(s) && <Button size="mini" status="warning" icon={<Ban size={12} />} loading={actionId === r.id} onClick={() => handleCancel(r as unknown as ExecutionTask)}>{t('executionTasks.actionCancel')}</Button>}
            {canRetry(s) && <Button size="mini" icon={<RotateCcw size={12} />} loading={actionId === r.id} onClick={() => handleRetry(r as unknown as ExecutionTask)}>{t('executionTasks.actionRetry')}</Button>}
            {canDelete(s) && <Button size="mini" status="danger" icon={<Trash2 size={12} />} loading={actionId === r.id} onClick={() => handleDelete(r as unknown as ExecutionTask)}>{t('executionTasks.actionDelete')}</Button>}
          </div>
        )
      },
    },
  ]

  return (
    <BaseLayout title={t('executionTasks.title')} description={t('executionTasks.subtitle')}>
      {stats && (
        <div className="mb-3 flex flex-wrap gap-2">
          <Tag>{t('executionTasks.statsTotal', { count: stats.total })}</Tag>
          {Object.entries(stats.byStatus || {}).map(([k, v]) => (
            <Tag key={k} color={statusColor(k)}>{statusLabel(k)}: {v}</Tag>
          ))}
        </div>
      )}
      <DataList
        data={list as unknown as Record<string, unknown>[]}
        columns={columns}
        loading={loading}
        rowKey={(r) => String((r as Record<string, unknown>).id)}
        emptyText={t('executionTasks.empty')}
       
        pagination={{ current: page, pageSize: 20, total, onChange: (p: number) => setPage(p) }}
        header={
          <div className="flex flex-wrap items-end gap-3">
            <Input prefix={<Search size={14} />} placeholder={t('executionTasks.searchPlaceholder')} value={search} onChange={setSearch} onPressEnter={() => { setPage(1); void fetchList() }} allowClear style={{ width: 200 }} />
            <Input placeholder={t('executionTasks.queuePlaceholder')} value={queueName} onChange={setQueueName} onPressEnter={() => { setPage(1); void fetchList() }} allowClear style={{ width: 160 }} />
            <Select placeholder={t('executionTasks.statusFilter')} value={status} onChange={(v) => { setPage(1); setStatus(v) }} style={{ width: 130 }} options={STATUS_OPTIONS.map((o) => ({ value: o.value, label: t(o.labelKey) }))} />
            <Button type="outline" icon={<RefreshCw size={14} />} onClick={() => { setPage(1); void fetchList() }}>{t('executionTasks.refresh')}</Button>
          </div>
        }
      />
      <Drawer title={t('executionTasks.detailTitle')} visible={!!detail} width={720} onCancel={() => setDetail(null)} footer={null}>
        {detailLoading ? <div className="p-8 text-center text-neutral-400">{t('executionTasks.loading')}</div> : detail ? (
          <div className="space-y-3 text-sm">
            <InfoRow label={t('executionTasks.fieldId')} value={detail.id} />
            <InfoRow label={t('executionTasks.fieldTaskId')} value={detail.taskId} />
            <InfoRow label={t('executionTasks.colStatus')} value={<Tag color={statusColor(detail.status)}>{statusLabel(detail.status)}</Tag>} />
            <InfoRow label={t('executionTasks.colTitle')} value={detail.title || '-'} />
            <InfoRow label={t('executionTasks.colQueue')} value={detail.queueName} />
            <InfoRow label={t('executionTasks.colKind')} value={detail.kind || '-'} />
            <InfoRow label={t('executionTasks.colPriority')} value={String(detail.priority)} />
            <InfoRow label={t('executionTasks.fieldRetry')} value={`${detail.retryCount}/${detail.maxRetries}`} />
            <InfoRow label={t('executionTasks.fieldWorker')} value={detail.workerId || '-'} />
            <InfoRow label={t('executionTasks.colSubmitTime')} value={formatDateTime(detail.submitTime)} />
            <InfoRow label={t('executionTasks.fieldStartedAt')} value={formatDateTime(detail.startedAt)} />
            <InfoRow label={t('executionTasks.fieldFinishedAt')} value={formatDateTime(detail.finishedAt)} />
            <InfoRow label={t('executionTasks.fieldRemark')} value={detail.remark || '-'} />
            {detail.errorMsg && (
              <div className="rounded-md border border-red-200 bg-red-50 p-3 text-red-600 break-all">
                <div className="mb-1 text-xs font-semibold">{t('executionTasks.fieldError')}</div>{detail.errorMsg}
              </div>
            )}
            {detail.paramsJson && <div><div className="mb-1 text-xs text-neutral-500">{t('executionTasks.fieldParams')}</div><pre className="m-0 rounded bg-neutral-50 p-3 text-xs overflow-auto max-h-64">{formatJSON(detail.paramsJson)}</pre></div>}
            {detail.resultJson && <div><div className="mb-1 text-xs text-neutral-500">{t('executionTasks.fieldResult')}</div><pre className="m-0 rounded bg-neutral-50 p-3 text-xs overflow-auto max-h-48">{formatJSON(detail.resultJson)}</pre></div>}
          </div>
        ) : null}
      </Drawer>
    </BaseLayout>
  )
}

function formatJSON(raw: string): string { try { return JSON.stringify(JSON.parse(raw), null, 2) } catch { return raw } }
function InfoRow({ label, value }: { label: string; value: React.ReactNode }) { return <div className="flex items-start gap-3"><div className="w-28 shrink-0 text-neutral-500">{label}</div><div className="flex-1 break-all">{value}</div></div> }
