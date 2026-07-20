import { useCallback, useEffect, useMemo, useState } from 'react'
import { Drawer, DatePicker } from '@arco-design/web-react'
import dayjs from 'dayjs'
import { Button, Input, Select, Card, TableEmpty } from '@/components/ui'
import { Loading } from '@/components/ui/loading'
import BaseLayout from '@/components/Layout/BaseLayout'
import {
  listMyOperationLogs,
  listTenantOperationLogs,
  listOperationLogsPlatform,
  type OperationLogRow,
} from '@/api/operationLogs'
import { useAuthStore } from '@/stores/authStore'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useTranslation } from '@/i18n'
import { EllipsisHoverCell } from '@/pages/assistants/EllipsisHoverCell'
import { downloadSpreadsheetFile, type SpreadsheetExportFormat } from '@/utils/csvExport'

const ACTION_OPTIONS = [
  '',
  'create',
  'update',
  'delete',
  'restore',
  'enable',
  'disable',
  'reorder',
  'start',
  'pause',
  'resume',
  'stop',
]

const actionLabelMap: Record<string, string> = {
  create: 'operationLogs.actionCreate',
  update: 'operationLogs.actionUpdate',
  delete: 'operationLogs.actionDelete',
  restore: 'operationLogs.actionRestore',
  enable: 'operationLogs.actionEnable',
  disable: 'operationLogs.actionDisable',
  reorder: 'operationLogs.actionReorder',
  start: 'operationLogs.actionStart',
  pause: 'operationLogs.actionPause',
  resume: 'operationLogs.actionResume',
  stop: 'operationLogs.actionStop',
  api_call: 'operationLogs.actionApiCall',
}

const OperationLogs = ({ embedded = false }: { embedded?: boolean }) => {
  const { t } = useTranslation()

  const actionLabel = useMemo(() => {
    const map: Record<string, string> = {}
    for (const [key, i18nKey] of Object.entries(actionLabelMap)) {
      map[key] = t(i18nKey)
    }
    return map
  }, [t])
  const user = useAuthStore((s) => s.user)
  const refreshUserInfo = useAuthStore((s) => s.refreshUserInfo)
  const isPlatformAdmin = Boolean(user?.isPlatformAdmin || user?.principal === 'platform')
  const permissionCodes = (user?.permissionCodes as string[] | undefined) ?? []
  const canTenantAudit =
    permissionCodes.includes('*') || permissionCodes.includes('api.operation_logs.read')

  useEffect(() => {
    void refreshUserInfo()
  }, [refreshUserInfo])

  const [rows, setRows] = useState<OperationLogRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [searchNonce, setSearchNonce] = useState(0)

  const [actionQ, setActionQ] = useState('')
  const [resourceQ, setResourceQ] = useState('')
  const [operatorQ, setOperatorQ] = useState('')
  const [tenantIdQ, setTenantIdQ] = useState('')
  const [successQ, setSuccessQ] = useState<'' | 'true' | 'false'>('')
  const [fromQ, setFromQ] = useState('')
  const [toQ, setToQ] = useState('')

  const [detailRow, setDetailRow] = useState<OperationLogRow | null>(null)
  const [exporting, setExporting] = useState(false)

  const pageSize = 20

  const listOpts = useMemo(() => {
    const opts: Parameters<typeof listMyOperationLogs>[2] = {}
    if (actionQ) opts.action = actionQ
    if (resourceQ.trim()) opts.resource = resourceQ.trim()
    if (successQ === 'true') opts.success = true
    if (successQ === 'false') opts.success = false
    if (fromQ) opts.from = fromQ
    if (toQ) opts.to = toQ
    if (isPlatformAdmin) {
      if (operatorQ.trim()) opts.operator = operatorQ.trim()
      if (tenantIdQ.trim()) {
        const n = Number(tenantIdQ.trim())
        if (Number.isFinite(n) && n > 0) opts.tenantId = n
      }
    }
    return opts
  }, [actionQ, resourceQ, successQ, fromQ, toQ, operatorQ, tenantIdQ, isPlatformAdmin])

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = isPlatformAdmin
        ? await listOperationLogsPlatform(page, pageSize, listOpts)
        : canTenantAudit
          ? await listTenantOperationLogs(page, pageSize, listOpts)
          : await listMyOperationLogs(page, pageSize, listOpts)
      if (res.code === 200 && res.data) {
        setRows(res.data.list || [])
        setTotal(res.data.total || 0)
      } else {
        showAlert(res.msg || t('common.loadFailed'), 'error')
      }
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('common.loadFailed')), 'error')
    } finally {
      setLoading(false)
    }
  }, [canTenantAudit, isPlatformAdmin, listOpts, page, pageSize, t])

  useEffect(() => {
    void load()
  }, [load, searchNonce])

  const fmt = (s?: string) => (s ? new Date(s).toLocaleString() : '—')

  const formatDetailJson = (raw?: string) => {
    if (!raw?.trim()) return '—'
    try {
      return JSON.stringify(JSON.parse(raw), null, 2)
    } catch {
      return raw
    }
  }

  const applyQuickRange = (kind: '3d' | '7d' | '1m') => {
    const now = new Date()
    const start = new Date(now)
    if (kind === '3d') start.setDate(now.getDate() - 3)
    else if (kind === '7d') start.setDate(now.getDate() - 7)
    else start.setMonth(now.getMonth() - 1)
    const toDay = (d: Date) => {
      const pad = (n: number) => `${n}`.padStart(2, '0')
      return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
    }
    setFromQ(toDay(start))
    setToQ(toDay(now))
    setPage(1)
    setSearchNonce((n) => n + 1)
  }

  const handleExport = async (format: SpreadsheetExportFormat) => {
    setExporting(true)
    try {
      const headers = [
        t('operationLogs.colTime'),
        t('operationLogs.colOperator'),
        t('operationLogs.colAction'),
        t('operationLogs.colResource'),
        t('operationLogs.colResourceName'),
        t('operationLogs.colSuccess'),
        t('operationLogs.colSummary'),
      ]
      const data = rows.map((r) => [
        r.createdAt ? new Date(r.createdAt).toLocaleString() : '',
        r.operator || '',
        r.action || '',
        r.resource || '',
        r.resourceName || '',
        r.success ? t('operationLogs.success') : t('operationLogs.failed'),
        r.summary || '',
      ])
      await downloadSpreadsheetFile(format, 'operation-logs', headers, data)
    } finally {
      setExporting(false)
    }
  }

  const body = (
    <>
      <div className="mb-3 flex flex-wrap gap-2 items-center">
        {isPlatformAdmin ? (
          <>
            <Input
              placeholder={t('operationLogs.filterOperator')}
              value={operatorQ}
              onChange={(v) => setOperatorQ(v)}
              style={{ width: 140 }}
              size="small"
              block={false}
              allowClear
            />
            <Input
              placeholder={t('operationLogs.filterTenantId')}
              value={tenantIdQ}
              onChange={(v) => setTenantIdQ(v)}
              style={{ width: 120 }}
              size="small"
              block={false}
              allowClear
            />
          </>
        ) : null}
        <Select
          value={actionQ}
          onChange={(v) => setActionQ(v as string)}
          style={{ width: 160 }}
          size="small"
          options={[
            { label: t('operationLogs.allActions'), value: '' },
            ...ACTION_OPTIONS.filter(Boolean).map((a) => ({ label: actionLabel[a] || a, value: a })),
          ]}
        />
        <Input
          placeholder={t('operationLogs.filterResource')}
          value={resourceQ}
          onChange={(v) => setResourceQ(v)}
          style={{ width: 160 }}
          size="small"
          block={false}
          allowClear
        />
        <Select
          value={successQ}
          onChange={(v) => setSuccessQ(v as '' | 'true' | 'false')}
          style={{ width: 140 }}
          size="small"
          options={[
            { label: t('operationLogs.allResults'), value: '' },
            { label: t('operationLogs.success'), value: 'true' },
            { label: t('operationLogs.failed'), value: 'false' },
          ]}
        />
        <DatePicker
          style={{ width: 160 }}
          size="small"
          allowClear
          placeholder={t('operationLogs.fromDate')}
          value={fromQ ? dayjs(fromQ, 'YYYY-MM-DD') : undefined}
          onChange={(_dateString, date) => setFromQ(date ? dayjs(date).format('YYYY-MM-DD') : '')}
        />
        <DatePicker
          style={{ width: 160 }}
          size="small"
          allowClear
          placeholder={t('operationLogs.toDate')}
          value={toQ ? dayjs(toQ, 'YYYY-MM-DD') : undefined}
          onChange={(_dateString, date) => setToQ(date ? dayjs(date).format('YYYY-MM-DD') : '')}
        />
        <Button
          type="primary"
          size="small"
          onClick={() => {
            setPage(1)
            setSearchNonce((n) => n + 1)
          }}
        >
          {t('common.search')}
        </Button>
        <Button type="outline" size="small" onClick={() => applyQuickRange('3d')}>
          {t('operationLogs.last3Days')}
        </Button>
        <Button type="outline" size="small" onClick={() => applyQuickRange('7d')}>
          {t('operationLogs.last7Days')}
        </Button>
        <Button
          type="outline"
          size="small"
          onClick={() => {
            setActionQ('')
            setResourceQ('')
            setOperatorQ('')
            setTenantIdQ('')
            setSuccessQ('')
            setFromQ('')
            setToQ('')
            setPage(1)
            setSearchNonce((n) => n + 1)
          }}
        >
          {t('operationLogs.reset')}
        </Button>
        <Button size="small" loading={exporting} onClick={() => void handleExport('csv')}>
          {t('billing.exportCsv')}
        </Button>
        <Button size="small" loading={exporting} onClick={() => void handleExport('xlsx')}>
          {t('billing.exportExcel')}
        </Button>
      </div>

      <Card bordered={false} bodyStyle={{ padding: 0 }}>
        <div className="overflow-x-auto">
          <table className="min-w-[1100px] w-full text-sm">
            <thead className="bg-muted/50">
              <tr>
                <th className="text-left p-3 whitespace-nowrap">{t('operationLogs.colTime')}</th>
                {isPlatformAdmin ? (
                  <th className="text-left p-3 whitespace-nowrap">{t('operationLogs.colTenant')}</th>
                ) : null}
                <th className="text-left p-3 whitespace-nowrap">{t('operationLogs.colOperator')}</th>
                <th className="text-left p-3 whitespace-nowrap">{t('operationLogs.colResource')}</th>
                <th className="text-left p-3 min-w-[180px]">{t('operationLogs.colSummary')}</th>
                <th className="text-left p-3 whitespace-nowrap">{t('common.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td className="p-6 text-center" colSpan={isPlatformAdmin ? 8 : 7}>
                    <Loading block tip={t('common.loading')} />
                  </td>
                </tr>
              ) : rows.length === 0 ? (
                <tr>
                  <td className="p-2" colSpan={isPlatformAdmin ? 8 : 7}>
                    <TableEmpty description={t('common.noData')} />
                  </td>
                </tr>
              ) : (
                rows.map((row) => (
                  <tr key={row.id} className="border-t border-border align-top">
                    <td className="p-3 whitespace-nowrap text-xs">{fmt(row.createdAt)}</td>
                    {isPlatformAdmin ? (
                      <td className="p-3 whitespace-nowrap">{row.tenantId ?? '—'}</td>
                    ) : null}
                    <td className="p-3 max-w-[140px]">
                      <EllipsisHoverCell text={row.operator?.trim() || '—'} lines={2} />
                    </td>
                    <td className="p-3 text-xs max-w-[140px]">
                      <div>{row.resource || '—'}</div>
                      {row.resourceName ? (
                        <div className="text-muted-foreground mt-0.5">
                          <EllipsisHoverCell text={row.resourceName} lines={1} />
                        </div>
                      ) : null}
                    </td>
                    <td className="p-3 text-xs max-w-[200px]">
                      <EllipsisHoverCell text={row.summary?.trim() || '—'} lines={2} />
                    </td>
                    <td className="p-3 whitespace-nowrap">
                      <Button type="outline" size="small" onClick={() => setDetailRow(row)}>
                        {t('common.detail')}
                      </Button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
        <div className="flex items-center justify-between p-3 border-t border-border text-sm">
          <span className="text-muted-foreground">
            {t('common.total')}: {total}
          </span>
          <div className="flex gap-2">
            <Button
              type="outline"
              size="small"
              disabled={page <= 1}
              onClick={() => setPage((p) => Math.max(1, p - 1))}
            >
              {t('common.previous')}
            </Button>
            <Button
              type="outline"
              size="small"
              disabled={page * pageSize >= total}
              onClick={() => setPage((p) => p + 1)}
            >
              {t('common.next')}
            </Button>
          </div>
        </div>
      </Card>

      <Drawer
        width={560}
        title={t('operationLogs.detailTitle')}
        visible={detailRow != null}
        onCancel={() => setDetailRow(null)}
        footer={null}
      >
        {detailRow ? (
          <div className="space-y-3 text-sm">
            <div className="grid grid-cols-2 gap-2">
              <div>
                <div className="text-xs text-muted-foreground">{t('operationLogs.colTime')}</div>
                <div>{fmt(detailRow.createdAt)}</div>
              </div>
              {isPlatformAdmin ? (
                <div>
                  <div className="text-xs text-muted-foreground">{t('operationLogs.colTenant')}</div>
                  <div>{detailRow.tenantId ?? '—'}</div>
                </div>
              ) : null}
              <div>
                <div className="text-xs text-muted-foreground">{t('operationLogs.colOperator')}</div>
                <div>{detailRow.operator || '—'}</div>
              </div>
              <div>
                <div className="text-xs text-muted-foreground">{t('operationLogs.colAction')}</div>
                <div>{actionLabel[detailRow.action || ''] || detailRow.action || '—'}</div>
              </div>
              <div>
                <div className="text-xs text-muted-foreground">{t('operationLogs.colResource')}</div>
                <div>
                  {detailRow.resource || '—'}
                  {detailRow.resourceId ? ` #${detailRow.resourceId}` : ''}
                </div>
              </div>
              <div>
                <div className="text-xs text-muted-foreground">{t('operationLogs.colResult')}</div>
                <div>{detailRow.success ? t('operationLogs.success') : t('operationLogs.failed')}</div>
              </div>
              {detailRow.httpMethod || detailRow.httpPath ? (
                <div className="col-span-2">
                  <div className="text-xs text-muted-foreground">HTTP</div>
                  <div className="font-mono text-xs break-all">
                    {[detailRow.httpMethod, detailRow.httpPath].filter(Boolean).join(' ')}
                  </div>
                </div>
              ) : null}
              {detailRow.errorMsg ? (
                <div className="col-span-2">
                  <div className="text-xs text-muted-foreground">{t('operationLogs.errorMsg')}</div>
                  <div className="text-destructive">{detailRow.errorMsg}</div>
                </div>
              ) : null}
              {detailRow.summary ? (
                <div className="col-span-2">
                  <div className="text-xs text-muted-foreground">{t('operationLogs.colSummary')}</div>
                  <div>{detailRow.summary}</div>
                </div>
              ) : null}
            </div>
            <div>
              <div className="mb-1 text-xs text-muted-foreground">{t('operationLogs.detailJson')}</div>
              <pre className="max-h-[420px] overflow-auto rounded-md border border-border bg-muted/30 p-3 text-xs whitespace-pre-wrap break-all">
                {formatDetailJson(detailRow.detailJson)}
              </pre>
            </div>
          </div>
        ) : null}
      </Drawer>
    </>
  )

  if (embedded) {
    return (
      <div>
        <div className="mb-5">
          <h2 className="text-base font-medium text-foreground">{t('pages.operationLogs.title')}</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            {isPlatformAdmin ? t('pages.operationLogs.descriptionPlatform') : t('pages.operationLogs.descriptionTenant')}
          </p>
        </div>
        {body}
      </div>
    )
  }

  return (
    <BaseLayout
      title={t('pages.operationLogs.title')}
      description={
        isPlatformAdmin
          ? t('pages.operationLogs.descriptionPlatform')
          : t('pages.operationLogs.descriptionTenant')
      }
    >
      {body}
    </BaseLayout>
  )
}

export default OperationLogs
export { OperationLogs as OperationLogsPanel }
