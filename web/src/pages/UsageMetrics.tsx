import { useCallback, useEffect, useState } from 'react'
import { Tag } from '@arco-design/web-react'
import dayjs from 'dayjs'
import { RefreshCw, Phone, BarChart3 } from 'lucide-react'
import { Button, DataList } from '@/components/ui'
import type { DataListColumn } from '@/components/ui'
import BaseLayout from '@/components/Layout/BaseLayout'
import { useTranslation } from '@/i18n'
import {
  getBillingBusinessMetrics,
  getBillingUsageSummary,
  listBillingUsageEvents,
  type BillingUsageSummary,
  type BusinessMetrics,
  type TenantUsageEventRow,
} from '@/api/billingUsage'
import { showAlert } from '@/utils/notification'

function pct(v?: number) {
  return `${Math.round((v ?? 0) * 1000) / 10}%`
}

export default function UsageMetrics() {
  const { t } = useTranslation()
  const [summary, setSummary] = useState<BillingUsageSummary | null>(null)
  const [metrics, setMetrics] = useState<BusinessMetrics | null>(null)
  const [events, setEvents] = useState<TenantUsageEventRow[]>([])
  const [loading, setLoading] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [sRes, mRes, eRes] = await Promise.all([
        getBillingUsageSummary(),
        getBillingBusinessMetrics({ days: 30 }),
        listBillingUsageEvents(1, 10),
      ])
      if (sRes.code === 200 && sRes.data) setSummary(sRes.data)
      if (mRes.code === 200 && mRes.data) setMetrics(mRes.data)
      if (eRes.code === 200 && eRes.data) {
        setEvents(eRes.data.list || [])
      }
    } catch {
      showAlert(t('usageMetrics.loadFailed'), 'error')
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => { void load() }, [load])

  const acct = summary?.account
  const quotas = summary?.quotas

  const eventColumns: DataListColumn<Record<string, unknown>>[] = [
    {
      key: 'callId', title: t('usageMetrics.colCallId'),
      render: (_, r) => <span className="truncate font-mono text-sm text-neutral-700">{String(r.callId || '—')}</span>,
    },
    {
      key: 'duration', title: t('usageMetrics.colDuration'), width: 100,
      render: (_, r) => <span className="text-sm text-neutral-700">{String(r.durationSec ?? '—')}</span>,
    },
    {
      key: 'billed', title: t('usageMetrics.colBilledMin'), width: 100,
      render: (_, r) => <span className="text-sm text-neutral-700">{String(r.billedMinutes ?? '—')}</span>,
    },
    {
      key: 'deducted', title: t('usageMetrics.colDeducted'), width: 100,
      render: (_, r) => <span className="text-sm text-neutral-700">{String(r.minutesDeducted ?? '—')}</span>,
    },
    {
      key: 'time', title: t('usageMetrics.colTime'), width: 180,
      render: (_, r) => {
        const ts = String(r.createdAt || '')
        return <span className="text-sm text-neutral-700">{ts ? dayjs(ts).format('YYYY-MM-DD HH:mm:ss') : '—'}</span>
      },
    },
  ]

  return (
    <BaseLayout title={t('usageMetrics.title')} description={t('usageMetrics.subtitle')}>
      <div className="space-y-4">
        <div className="flex items-center justify-end">
          <Button type="outline" icon={<RefreshCw size={14} />} loading={loading} onClick={() => void load()}>{t('usageMetrics.refresh')}</Button>
        </div>

        <div className="rounded-xl border border-border bg-card p-5">
          <div className="mb-3 text-sm font-medium text-neutral-900">{t('usageMetrics.title')}</div>
          <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
            <div className="rounded-lg border border-neutral-100 bg-neutral-50 p-3">
              <div className="text-xs text-neutral-500">{t('billing.remainingMinutes')}</div>
              <div className="mt-1 text-lg font-semibold text-neutral-900">{acct?.remainingMinutesDisplay ?? '—'}</div>
            </div>
            <div className="rounded-lg border border-neutral-100 bg-neutral-50 p-3">
              <div className="text-xs text-neutral-500">{t('usageMetrics.dailyUsed')}</div>
              <div className="mt-1 text-lg font-semibold text-neutral-900">
                {quotas?.dailyMinutesUsed ?? 0}{quotas?.dailyMinuteLimit ? ` / ${quotas.dailyMinuteLimit}` : ''}
              </div>
            </div>
            <div className="rounded-lg border border-neutral-100 bg-neutral-50 p-3">
              <div className="text-xs text-neutral-500">{t('usageMetrics.monthlyUsed')}</div>
              <div className="mt-1 text-lg font-semibold text-neutral-900">
                {quotas?.monthlyMinutesUsed ?? 0}{quotas?.monthlyMinuteLimit ? ` / ${quotas.monthlyMinuteLimit}` : ''}
              </div>
            </div>
            <div className="rounded-lg border border-neutral-100 bg-neutral-50 p-3">
              <div className="text-xs text-neutral-500">{t('usageMetrics.concurrent')}</div>
              <div className="mt-1 text-lg font-semibold text-neutral-900">
                {quotas?.heldMinutes ?? 0}{quotas?.maxConcurrentCalls ? ` / ${quotas.maxConcurrentCalls}` : ''}
              </div>
            </div>
          </div>
          <div className="mt-3 flex gap-2">
            {quotas?.quotaSuspended ? <Tag color="red">{t('usageMetrics.quotaSuspended')}</Tag> : null}
            {!quotas?.licenseValid && quotas?.licenseExpiresAt ? <Tag color="orangered">{t('usageMetrics.licenseExpired')}</Tag> : null}
          </div>
        </div>

        <div className="rounded-xl border border-border bg-card p-5">
          <div className="mb-3 flex items-center gap-2 text-sm font-medium text-neutral-900">
            <BarChart3 size={16} className="text-neutral-500" />
            {t('usageMetrics.businessKpi')}
          </div>
          <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
            <div className="rounded-lg border border-neutral-100 bg-neutral-50 p-3 text-center">
              <div className="text-xs text-neutral-500">{t('usageMetrics.connectRate')}</div>
              <div className="mt-1 text-xl font-semibold text-neutral-900">{pct(metrics?.connectRate)}</div>
            </div>
            <div className="rounded-lg border border-neutral-100 bg-neutral-50 p-3 text-center">
              <div className="text-xs text-neutral-500">{t('usageMetrics.totalCalls')}</div>
              <div className="mt-1 text-xl font-semibold text-neutral-900">{metrics?.totalCalls ?? 0}</div>
            </div>
            <div className="rounded-lg border border-neutral-100 bg-neutral-50 p-3 text-center">
              <div className="text-xs text-neutral-500">{t('usageMetrics.connectedCalls')}</div>
              <div className="mt-1 text-xl font-semibold text-neutral-900">{metrics?.connectedCalls ?? 0}</div>
            </div>
            <div className="rounded-lg border border-neutral-100 bg-neutral-50 p-3 text-center">
              <div className="text-xs text-neutral-500">{t('usageMetrics.billedMinutes')}</div>
              <div className="mt-1 text-xl font-semibold text-neutral-900">{metrics?.billedMinutes ?? 0}</div>
            </div>
          </div>
        </div>

        <DataList
          data={events as unknown as (TenantUsageEventRow & Record<string, unknown>)[]}
          columns={eventColumns}
          loading={loading}
          rowKey="id"
          emptyText={t('common.noData')}
          header={
            <div className="flex items-center gap-2">
              <Phone size={14} className="text-neutral-500" />
              <span className="text-sm font-medium text-neutral-900">{t('usageMetrics.recentUsage')}</span>
            </div>
          }
        />
      </div>
    </BaseLayout>
  )
}
