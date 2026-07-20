import { useCallback, useEffect, useState } from 'react'
import dayjs from 'dayjs'
import { Drawer, Tag, DatePicker } from '@arco-design/web-react'
import { Eye, Download, CheckCircle, Search, RefreshCw } from 'lucide-react'
import { Button, DataList, Empty, Select } from '@/components/ui'
import type { DataListColumn } from '@/components/ui'
import { showAlert } from '@/utils/notification'
import BaseLayout from '@/components/Layout/BaseLayout'
import { useTranslation } from '@/i18n'
import { useAuthStore } from '@/stores/authStore'
import {
  downloadTenantBillExport,
  finalizeTenantBill,
  getTenantBillingAccount,
  listTenantBills,
  type TenantBillRow,
  type TenantBillingAccount,
} from '@/api/tenantBills'
import { markTenantBillPaid } from '@/api/billingUsage'

const { MonthPicker } = DatePicker

function hasPermission(codes: readonly string[] | undefined, code: string): boolean {
  const list = codes ?? []; return list.includes('*') || list.includes(code)
}

function formatPeriod(row: TenantBillRow): string {
  return row.periodStart ? dayjs(row.periodStart).format('YYYY-MM') : '—'
}

function formatRemainingAccount(acct: TenantBillingAccount | null, t: (k: string) => string): string {
  if (!acct) return '—'
  if (acct.billingUnlimited || acct.remainingMinutesDisplay === 'unlimited') return t('billing.unlimitedBalance')
  if (acct.billingMode === 'postpaid') return t('billing.modePostpaid')
  return acct.remainingMinutesDisplay || String(acct.prepaidMinutesRemaining ?? 0)
}

export default function TenantBills() {
  const { t } = useTranslation()
  const user = useAuthStore((s) => s.user)
  const isPlatformAdmin = Boolean(user?.isPlatformAdmin || user?.principal === 'platform')
  const permissionCodes = (user?.permissionCodes as string[] | undefined) ?? []
  const canRead = isPlatformAdmin || hasPermission(permissionCodes, 'api.billing.read')
  const canWrite = isPlatformAdmin || hasPermission(permissionCodes, 'api.billing.write')

  const [rows, setRows] = useState<TenantBillRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [periodQ, setPeriodQ] = useState('')
  const [statusQ, setStatusQ] = useState('')
  const [detailRow, setDetailRow] = useState<TenantBillRow | null>(null)
  const [finalizingId, setFinalizingId] = useState<string | null>(null)
  const [markingPaidId, setMarkingPaidId] = useState<string | null>(null)
  const [account, setAccount] = useState<TenantBillingAccount | null>(null)
  const [exportingId, setExportingId] = useState<string | null>(null)
  const pageSize = 20

  const loadAccount = useCallback(async () => {
    if (!canRead) return
    try { const res = await getTenantBillingAccount(); if (res.code === 200 && res.data) setAccount(res.data) } catch { /* ignore */ }
  }, [canRead])

  const load = useCallback(async () => {
    if (!canRead) return
    setLoading(true)
    try {
      const res = await listTenantBills(page, pageSize, { period: periodQ.trim() || undefined, status: statusQ.trim() || undefined })
      if (res.code === 200 && res.data) { setRows(res.data.list || []); setTotal(res.data.total || 0) }
    } finally { setLoading(false) }
  }, [canRead, page, periodQ, statusQ])

  useEffect(() => { void load() }, [load])
  useEffect(() => { void loadAccount() }, [loadAccount])

  const handleExport = async (row: TenantBillRow, format: 'csv' | 'xlsx') => {
    if (!row.id) return
    setExportingId(row.id)
    try { await downloadTenantBillExport(row.id, format); showAlert(t('billing.exportSuccess'), 'success') }
    catch { showAlert(t('billing.exportFailed'), 'error') }
    finally { setExportingId(null) }
  }

  const statusTag = (status?: string) => {
    const map: Record<string, { color: string; label: string }> = {
      draft: { color: 'gray', label: t('billing.status.draft') },
      finalized: { color: 'arcoblue', label: t('billing.status.finalized') },
      paid: { color: 'green', label: t('billing.status.paid') },
    }
    const item = map[status || ''] || { color: 'gray', label: status || '—' }
    return <Tag color={item.color} className="!rounded-full">{item.label}</Tag>
  }

  const handleFinalize = async (row: TenantBillRow) => {
    if (!canWrite || !row.id) return
    setFinalizingId(row.id)
    try {
      const res = await finalizeTenantBill(row.id)
      if (res.code === 200) { showAlert(t('billing.finalizeSuccess'), 'success'); await load(); if (detailRow?.id === row.id && res.data) setDetailRow(res.data) }
      else showAlert(res.msg || t('common.failed'), 'error')
    } catch (err: any) { showAlert(err?.msg || t('common.failed'), 'error') }
    finally { setFinalizingId(null) }
  }

  const handleMarkPaid = async (row: TenantBillRow) => {
    if (!isPlatformAdmin || !row.id) return
    setMarkingPaidId(row.id)
    try {
      const res = await markTenantBillPaid(row.id)
      if (res.code === 200) { showAlert(t('billing.markPaidSuccess'), 'success'); await load() }
      else showAlert(res.msg || t('common.failed'), 'error')
    } catch (err: unknown) { showAlert((err as { msg?: string })?.msg || t('common.failed'), 'error') }
    finally { setMarkingPaidId(null) }
  }

  const columns: DataListColumn<Record<string, unknown>>[] = [
    {
      key: 'info', title: t('billing.colBillNo'), width: 200,
      render: (_, r) => (
        <div className="min-w-0">
          <div className="truncate font-mono text-sm font-medium text-neutral-900">{String(r.billNo || '—')}</div>
          <div className="mt-0.5 text-xs text-neutral-500">{formatPeriod(r as unknown as TenantBillRow)}</div>
        </div>
      ),
    },
    {
      key: 'status', title: t('common.status'), width: 100,
      render: (_, r) => statusTag(String(r.status)),
    },
    {
      key: 'metrics', title: '用量', width: 180,
      render: (_, r) => (
        <div className="flex items-center gap-3 text-xs text-neutral-500">
          <span>{String(r.billedMinutes ?? 0)} 分钟</span>
          <span>{String(r.callCount ?? 0)} 会话</span>
          {r.analysisCount != null ? <span>{String(r.analysisCount)} 分析</span> : null}
        </div>
      ),
    },
    {
      key: 'time', title: t('billing.colUpdatedAt'), width: 170,
      render: (_, r) => {
        const ts = String(r.updatedAt || r.createdAt || '')
        return <span className="text-sm text-neutral-700">{ts ? dayjs(ts).format('YYYY-MM-DD HH:mm:ss') : '—'}</span>
      },
    },
    {
      key: 'actions', title: t('common.actions'), width: 260, align: 'right',
      render: (_, r) => (
        <div className="flex items-center justify-end gap-1">
          <Button size="mini" icon={<Eye size={12} />} onClick={() => setDetailRow(r as unknown as TenantBillRow)}>{t('billing.detailTitle')}</Button>
          <Button size="mini" icon={<Download size={12} />} loading={exportingId === r.id} onClick={() => void handleExport(r as unknown as TenantBillRow, 'csv')}>{t('billing.exportCsv')}</Button>
          <Button size="mini" icon={<Download size={12} />} loading={exportingId === r.id} onClick={() => void handleExport(r as unknown as TenantBillRow, 'xlsx')}>{t('billing.exportExcel')}</Button>
          {canWrite && r.status === 'draft' ? (
            <Button size="mini" icon={<CheckCircle size={12} />} loading={finalizingId === r.id} onClick={() => void handleFinalize(r as unknown as TenantBillRow)}>{t('billing.finalize')}</Button>
          ) : null}
          {isPlatformAdmin && r.status === 'finalized' ? (
            <Button size="mini" icon={<CheckCircle size={12} />} loading={markingPaidId === r.id} onClick={() => void handleMarkPaid(r as unknown as TenantBillRow)}>{t('billing.markPaid')}</Button>
          ) : null}
        </div>
      ),
    },
  ]

  if (!canRead) {
    return <BaseLayout title={t('nav.billing')} description="账单管理"><Empty preset="no-permission" description={t('billing.noPermission')} /></BaseLayout>
  }

  return (
    <BaseLayout title={t('nav.billing')} description="账单管理">
      <div className="space-y-4">
        <div className="rounded-xl border border-border bg-card p-5">
          <div className="mb-3 text-sm font-medium text-neutral-900">账户概览</div>
          <div className="grid grid-cols-2 gap-4 md:grid-cols-5">
            <div className="rounded-lg border border-neutral-100 bg-neutral-50 p-3">
              <div className="text-xs text-neutral-500">{t('billing.accountMode')}</div>
              <div className="mt-1 text-sm font-medium text-neutral-900">
                {account?.billingUnlimited ? t('billing.unlimitedBalance') : account?.billingMode === 'postpaid' ? t('billing.modePostpaid') : t('billing.modePrepaid')}
              </div>
            </div>
            <div className="rounded-lg border border-neutral-100 bg-neutral-50 p-3">
              <div className="text-xs text-neutral-500">{t('billing.remainingMinutes')}</div>
              <div className="mt-1 text-sm font-medium text-neutral-900">{formatRemainingAccount(account, t)}</div>
            </div>
            <div className="rounded-lg border border-neutral-100 bg-neutral-50 p-3">
              <div className="text-xs text-neutral-500">{t('billing.accountMeteredMinutes')}</div>
              <div className="mt-1 text-sm font-medium text-neutral-900">{account?.meteredBilledMinutes ?? 0}</div>
            </div>
            <div className="rounded-lg border border-neutral-100 bg-neutral-50 p-3">
              <div className="text-xs text-neutral-500">{t('billing.accountMeteredCalls')}</div>
              <div className="mt-1 text-sm font-medium text-neutral-900">{account?.meteredCallCount ?? 0}</div>
            </div>
            <div className="rounded-lg border border-neutral-100 bg-neutral-50 p-3">
              <div className="text-xs text-neutral-500">{t('billing.accountRate')}</div>
              <div className="mt-1 text-sm font-medium text-neutral-900">
                {account?.billingUnlimited ? t('billing.unlimitedBalance') : `${account?.billingRatePerMinute ?? 0} ${account?.billingCurrency || 'CNY'}`}
              </div>
            </div>
          </div>
        </div>

        <DataList
          data={rows as unknown as (TenantBillRow & Record<string, unknown>)[]}
          columns={columns}
          loading={loading}
          rowKey="id"
          emptyText={t('common.noData')}
          pagination={{ current: page, pageSize, total, onChange: (p) => setPage(p) }}
          header={
            <div className="flex flex-wrap items-end gap-3">
              <div>
                <div className="mb-1 text-xs text-neutral-400">{t('billing.filterPeriod')}</div>
                <MonthPicker allowClear placeholder="YYYY-MM" format="YYYY-MM" style={{ width: 140 }}
                  value={periodQ ? dayjs(periodQ, 'YYYY-MM') : undefined}
                  onChange={(_dateString, date) => setPeriodQ(date ? dayjs(date).format('YYYY-MM') : '')}
                />
              </div>
              <div>
                <div className="mb-1 text-xs text-neutral-400">{t('common.status')}</div>
                <Select allowClear placeholder={t('billing.allStatus')} value={statusQ || undefined}
                  onChange={(v) => setStatusQ(String(v || ''))} style={{ width: 140 }}
                  options={[
                    { label: t('billing.status.draft'), value: 'draft' },
                    { label: t('billing.status.finalized'), value: 'finalized' },
                    { label: t('billing.status.paid'), value: 'paid' },
                  ]}
                />
              </div>
              <Button type="primary" icon={<Search size={14} />} onClick={() => { setPage(1); void load() }}>{t('common.search')}</Button>
              <Button type="outline" icon={<RefreshCw size={14} />} onClick={() => void load()}>{t('common.refresh')}</Button>
            </div>
          }
          footer={<p className="text-xs text-neutral-400">{t('billing.autoBillHint')}</p>}
        />

        <Drawer width={720} title={t('billing.detailTitle')} visible={detailRow != null} onCancel={() => setDetailRow(null)}
          footer={detailRow ? (
            <div className="flex flex-wrap gap-2">
              <Button icon={<Download size={14} />} loading={exportingId === detailRow.id} onClick={() => void handleExport(detailRow, 'csv')}>{t('billing.exportCsv')}</Button>
              <Button icon={<Download size={14} />} loading={exportingId === detailRow.id} onClick={() => void handleExport(detailRow, 'xlsx')}>{t('billing.exportExcel')}</Button>
              {detailRow.status === 'draft' && canWrite ? (
                <Button type="primary" loading={finalizingId === detailRow.id} onClick={() => void handleFinalize(detailRow)}>{t('billing.finalize')}</Button>
              ) : null}
            </div>
          ) : null}
        >
          {detailRow ? (
            <div className="space-y-4">
              <div className="flex flex-wrap items-center gap-2">
                {statusTag(detailRow.status)}
                <span className="text-sm text-neutral-500">{detailRow.billNo}</span>
              </div>
              <div className="grid grid-cols-2 gap-3 text-sm md:grid-cols-3">
                <div><span className="text-neutral-500">{t('billing.colPeriod')}: </span>{formatPeriod(detailRow)}</div>
                <div><span className="text-neutral-500">{t('billing.colBilledMinutes')}: </span>{detailRow.billedMinutes ?? 0}</div>
                <div><span className="text-neutral-500">{t('billing.colCallCount')}: </span>{detailRow.callCount ?? 0}</div>
                <div><span className="text-neutral-500">{t('billing.colConnectedCalls')}: </span>{detailRow.connectedCallCount ?? 0}</div>
                <div><span className="text-neutral-500">{t('billing.colAnalysisCount')}: </span>{detailRow.analysisCount ?? 0}</div>
                <div><span className="text-neutral-500">{t('billing.colAmount')}: </span>{detailRow.totalAmount ?? 0} {detailRow.currency || 'CNY'}</div>
              </div>
              {detailRow.usageDetail?.daily?.length ? (
                <div className="rounded-xl border border-neutral-100 p-4">
                  <div className="mb-2 text-sm font-medium text-neutral-900">{t('billing.sectionDaily')}</div>
                  <div className="space-y-2">
                    {detailRow.usageDetail.daily.map((d, i) => (
                      <div key={i} className="flex items-center justify-between rounded-lg border border-neutral-100 px-3 py-2 text-sm">
                        <span className="text-neutral-700">{d.day ? dayjs(d.day).format('YYYY-MM-DD') : '—'}</span>
                        <span className="text-neutral-500">{d.callCount} 会话 · {d.billedMinutes} 分钟</span>
                      </div>
                    ))}
                  </div>
                </div>
              ) : null}
              {detailRow.usageDetail?.direction?.length ? (
                <div className="rounded-xl border border-neutral-100 p-4">
                  <div className="mb-2 text-sm font-medium text-neutral-900">{t('billing.sectionDirection')}</div>
                  <div className="space-y-2">
                    {detailRow.usageDetail.direction.map((d, i) => (
                      <div key={i} className="flex items-center justify-between rounded-lg border border-neutral-100 px-3 py-2 text-sm">
                        <span className="text-neutral-700">{d.direction}</span>
                        <span className="text-neutral-500">{d.callCount} 会话 · {d.billedMinutes} 分钟</span>
                      </div>
                    ))}
                  </div>
                </div>
              ) : null}
            </div>
          ) : null}
        </Drawer>
      </div>
    </BaseLayout>
  )
}
