import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { DatePicker } from '@arco-design/web-react'
import { TableEmpty } from '@/components/ui'
import { Loading } from '@/components/ui/loading'
import dayjs, { type Dayjs } from 'dayjs'
import BaseLayout from '@/components/Layout/BaseLayout'
import { useAuthStore } from '@/stores/authStore'
import { useTranslation } from '@/i18n'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  defaultOverviewRange,
  getOverviewDashboard,
  type OverviewDashboardData,
} from '@/api/overviewStats'
import { useVChartTheme, DashboardChart, EmptyChart } from '@/components/Dashboard/ChartWidgets'
import { Button } from '@/components/ui'
import { showAlert } from '@/utils/notification'
import { cn } from '@/utils/cn'

const { RangePicker } = DatePicker

const MAX_RANGE_DAYS = 90
const CHART_COLORS = ['#8B5CF6', '#A78BFA', '#22c55e', '#f59e0b', '#ec4899', '#7C3AED']

function HaloSectionHead({ eyebrow, title }: { eyebrow?: string; title: string }) {
  return (
    <div className="halo-section-head">
      {eyebrow ? <span className="halo-eyebrow">{eyebrow}</span> : null}
      <h2 className="halo-title-md">{title}</h2>
    </div>
  )
}

function HaloMetricTile({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="halo-metric-tile">
      <div className="halo-metric-tile-label" title={label}>{label}</div>
      <div className="halo-metric-tile-value">{value}</div>
    </div>
  )
}

function fmtNum(v?: number) {
  const n = v ?? 0
  if (!Number.isFinite(n)) return '0'
  return Math.round(n) === n ? String(n) : n.toFixed(1)
}

function fmtPct(rate?: number) {
  const n = (rate ?? 0) * 100
  return `${Math.round(n * 10) / 10}%`
}

function toDayjs(dateStr: string): Dayjs {
  return dayjs(dateStr, 'YYYY-MM-DD')
}

function validateRange(from: string, to: string, t: (k: string, params?: Record<string, string | number>) => string): string | null {
  const start = toDayjs(from)
  const end = toDayjs(to)
  if (!start.isValid() || !end.isValid()) return t('overviewPage.validation.selectValidDates')
  if (end.isBefore(start, 'day')) return t('overviewPage.validation.endNotBeforeStart')
  if (end.isAfter(dayjs(), 'day')) return t('overviewPage.validation.endNotAfterToday')
  if (end.diff(start, 'day') + 1 > MAX_RANGE_DAYS) return t('overviewPage.validation.maxRangeDays', { days: MAX_RANGE_DAYS })
  return null
}

function formatOverviewRemaining(
  meta: OverviewDashboardData['meta'] | undefined,
  t: (k: string) => string,
): string {
  if (!meta) return '—'
  if (meta.billingUnlimited || meta.remainingMinutesDisplay === 'unlimited') {
    return t('billing.unlimitedBalance')
  }
  if (meta.billingMode === 'postpaid') {
    return t('billing.modePostpaid')
  }
  return meta.remainingMinutesDisplay || String(meta.prepaidMinutesRemaining ?? 0)
}

export default function OverviewDashboard() {
  const { t } = useTranslation()
  useVChartTheme()

  const user = useAuthStore((s) => s.user)
  const isPlatform = Boolean(user?.isPlatformAdmin || user?.principal === 'platform')
  const tenantName = String(user?.tenantName || t('common.yourOrg'))
  const displayName = String(user?.displayName || user?.username || user?.email || t('common.adminFallback'))

  const initialRange = defaultOverviewRange(14)
  const [draftRange, setDraftRange] = useState<[string, string]>([initialRange.from, initialRange.to])
  const [queryRange, setQueryRange] = useState<[string, string]>([initialRange.from, initialRange.to])
  const [rangeSelecting, setRangeSelecting] = useState<Dayjs | null>(null)

  const [loading, setLoading] = useState(true)
  const [err, setErr] = useState<string | null>(null)
  const [dashboard, setDashboard] = useState<OverviewDashboardData | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setErr(null)
    try {
      const res = await getOverviewDashboard({ from: queryRange[0], to: queryRange[1] })
      if (res.code === 200 && res.data) {
        setDashboard(res.data)
      } else {
        setErr(res.msg || t('common.loadFailed'))
        setDashboard(null)
      }
    } catch (e: unknown) {
      setErr(extractApiErrorMessage(e, t('common.loadFailed')))
      setDashboard(null)
    } finally {
      setLoading(false)
    }
  }, [queryRange, t])

  useEffect(() => { void load() }, [load])

  const handleSearch = () => {
    const validationErr = validateRange(draftRange[0], draftRange[1], t)
    if (validationErr) { showAlert(validationErr, 'warning'); return }
    setQueryRange(draftRange)
  }

  const applyQuickRange = (days: number) => {
    const { from, to } = defaultOverviewRange(days)
    const validationErr = validateRange(from, to, t)
    if (validationErr) { showAlert(validationErr, 'warning'); return }
    setDraftRange([from, to])
    setQueryRange([from, to])
  }

  const summary = dashboard?.summary
  const meta = dashboard?.meta

  const peakDay = useMemo(() => {
    const days = dashboard?.days || []
    if (!days.length) return null
    return days.reduce((best, d) => (d.callCount > best.callCount ? d : best), days[0])
  }, [dashboard])

  const metricTiles = useMemo(() => {
    const tiles: { label: string; value: ReactNode }[] = []
    if (!isPlatform) {
      tiles.push({
        label: t('overviewPage.remainingMinutes'),
        value: formatOverviewRemaining(meta, t),
      })
    }
    tiles.push(
      { label: t('overviewPage.totalCallsAllTime'), value: meta?.allTimeCallCount ?? 0 },
      { label: t('overviewPage.totalBilledMinutesAllTime'), value: meta?.allTimeBilledMinutes ?? 0 },
    )
    if (peakDay) {
      tiles.push({ label: t('overviewPage.peakDayCalls'), value: `${peakDay.day.slice(5)} · ${peakDay.callCount}` })
    }
    return tiles
  }, [isPlatform, meta, peakDay, t])

  const trendValues = useMemo(() => {
    if (!dashboard?.days?.length) return { metrics: [], rate: [] }
    const metrics: { day: string; series: string; value: number }[] = []
    const rate: { day: string; series: string; value: number }[] = []
    for (const d of dashboard.days) {
      const day = d.day.slice(5)
      metrics.push(
        { day, series: t('overviewPage.chart.callCount'), value: d.callCount },
        { day, series: t('overviewPage.chart.billedMinutes'), value: d.totalMinutes },
        { day, series: t('overviewPage.chart.avgMinutes'), value: d.avgMinutesPerCall },
      )
      if ((d.knowledgeRecallTotal ?? 0) > 0) {
        rate.push({ day, series: t('overviewPage.chart.knowledgeRecallHitRate'), value: Math.round((d.knowledgeRecallHitRate ?? 0) * 1000) / 10 })
        rate.push({ day, series: t('overviewPage.chart.knowledgeUnrecognizedRate'), value: Math.round((d.knowledgeUnrecognizedRate ?? 0) * 1000) / 10 })
      }
    }
    return { metrics, rate }
  }, [dashboard, t])

  const trendSpec = useMemo(
    () => ({
      type: 'common',
      padding: { left: 12, right: 48, top: 32, bottom: 24 },
      color: CHART_COLORS,
      data: [
        { id: 'metrics', values: trendValues.metrics },
        { id: 'rate', values: trendValues.rate },
      ],
      series: [
        {
          type: 'line', dataIndex: 0, xField: 'day', yField: 'value', seriesField: 'series',
          point: { visible: true, style: { size: 3 } },
          line: { style: { lineWidth: 2 } },
        },
        {
          type: 'line', dataIndex: 1, id: 'knowledge-rate', xField: 'day', yField: 'value', seriesField: 'series',
          point: { visible: true, style: { size: 3 } },
          line: { style: { lineWidth: 2, lineDash: [5, 3] } },
        },
      ],
      legends: { visible: true, orient: 'top', position: 'start' },
      tooltip: { visible: true },
      axes: [
        { orient: 'bottom', type: 'band' },
        { orient: 'left', title: { visible: true, text: t('overviewPage.chart.callsOrMinutes') } },
        { orient: 'right', seriesId: ['knowledge-rate'], title: { visible: true, text: '%' }, min: 0, max: 100, grid: { visible: false } },
      ],
    }),
    [trendValues, t],
  )

  const dayRows = dashboard?.days || []
  const draftPickerValue: [Dayjs, Dayjs] | undefined = useMemo(() => {
    const start = toDayjs(draftRange[0])
    const end = toDayjs(draftRange[1])
    if (!start.isValid() || !end.isValid()) return undefined
    return [start, end]
  }, [draftRange])

  const disabledDate = (current?: Dayjs) => {
    if (!current) return false
    if (current.isAfter(dayjs(), 'day')) return true
    if (rangeSelecting) {
      return Math.abs(current.diff(rangeSelecting, 'day')) + 1 > MAX_RANGE_DAYS
    }
    return false
  }

  const hasTrend = trendValues.metrics.length > 0

  const activeQuickDays = useMemo(() => {
    const span = toDayjs(queryRange[1]).diff(toDayjs(queryRange[0]), 'day') + 1
    if (queryRange[1] === dayjs().format('YYYY-MM-DD')) {
      if (span === 7) return 7
      if (span === 14) return 14
      if (span === 30) return 30
    }
    return null
  }, [queryRange])

  const heroMetrics = !isPlatform ? [
    {
      label: t('overviewPage.remainingMinutes'),
      value: loading && !meta ? '—' : formatOverviewRemaining(meta, t),
      foot: !meta?.billingUnlimited && meta?.billingMode === 'prepaid' ? t('overviewPage.remainingMinutesUnit') : undefined,
    },
    {
      label: t('overviewPage.usageDays'),
      value: loading && !meta ? '—' : (
        <>
          {meta?.usageDays ?? 0}
          <span style={{ fontSize: 14, fontWeight: 500, marginLeft: 4 }}>{t('overviewPage.days')}</span>
        </>
      ),
      foot: meta?.tenantMemberCount != null
        ? `${t('overviewPage.teamMembers')} ${meta.tenantMemberCount} ${t('overviewPage.membersUnit')}`
        : undefined,
    },
  ] : []

  const knowledgeHeadlineStats = useMemo(() => {
    if (!summary) return []
    const total = summary.knowledgeRecallTotal ?? 0
    return [
      {
        tone: 'info' as const,
        label: t('overviewPage.knowledgeRecallHitRate'),
        value: total > 0 ? fmtPct(summary.knowledgeRecallHitRate) : '—',
      },
      {
        tone: 'warning' as const,
        label: t('overviewPage.knowledgeUnrecognizedRate'),
        value: total > 0 ? fmtPct(summary.knowledgeUnrecognizedRate) : '—',
      },
    ]
  }, [summary, t])

  const headlineStats = useMemo(() => {
    if (!summary) return []
    return [
      { tone: 'info' as const, label: t('overviewPage.rangeCallCount'), value: summary.callCount ?? 0 },
      { tone: 'success' as const, label: t('overviewPage.rangeBilledMinutes'), value: fmtNum(summary.totalMinutes) },
      { tone: undefined, label: t('overviewPage.rangeAvgMinutesPerCall'), value: fmtNum(summary.avgMinutesPerCall) },
    ]
  }, [summary, t])

  return (
    <BaseLayout
      title={t('nav.overview')}
      description={isPlatform ? t('overview.allPlatform') : tenantName}
    >
      <div className="halo-page-shell">
        <div className="halo-page overview-halo">
          <section className="halo-hero">
            <div className="halo-hero-inner">
              <div className="halo-hero-main">
                <span className="halo-hero-pill">
                  <span className="halo-hero-pill-dot" aria-hidden />
                  {isPlatform ? t('overview.platformConsole') : t('overview.tenantSpace')}
                </span>
                <h1 className="halo-hero-title">{t('overview.welcomeBack', { name: displayName })}</h1>
                <p className="halo-hero-sub">
                  {isPlatform ? t('overview.heroPlatform') : t('overview.heroTenant', { tenant: tenantName })}
                </p>
              </div>
              <div className="halo-hero-side">
                {heroMetrics.length > 0 ? (
                  <div className="halo-hero-metrics">
                    {heroMetrics.map((m) => (
                      <div key={m.label} className="halo-hero-metric">
                        <div className="halo-hero-metric-label">{m.label}</div>
                        <div className="halo-hero-metric-value">{m.value}</div>
                        {m.foot ? <div className="halo-hero-metric-foot">{m.foot}</div> : null}
                      </div>
                    ))}
                  </div>
                ) : null}
                <div className="halo-hero-org-label">{t('overview.organization')}</div>
                <div className="halo-hero-org-value">{isPlatform ? t('overview.allPlatform') : tenantName}</div>
              </div>
            </div>
          </section>

          <div className="halo-dashboard-body">
            {loading ? (
              <div className="halo-loading-overlay">
                <Loading block />
              </div>
            ) : null}

            {err ? (
              <p className="text-sm text-destructive mb-0">{err}</p>
            ) : null}

            <div className="halo-main-head">
              <div>
                <span className="halo-eyebrow">Overview</span>
                <h2 className="halo-title-md">{t('overviewPage.sectionMetrics')}</h2>
                {dashboard ? (
                  <p className="halo-body-sm" style={{ marginTop: 4, marginBottom: 0 }}>
                    {t('overviewPage.currentStats')}：{dashboard.rangeStart} ~ {dashboard.rangeEnd}
                    <span style={{ marginLeft: 8, opacity: 0.75 }}>（{t('overviewPage.maxDays', { days: MAX_RANGE_DAYS })}）</span>
                  </p>
                ) : null}
              </div>
              <div className="halo-main-actions">
                <div className="halo-tabs" role="tablist" aria-label={t('overviewPage.sectionMetrics')}>
                  {[7, 14, 30].map((days) => (
                    <button
                      key={days}
                      type="button"
                      className={cn('halo-tab', activeQuickDays === days && 'is-active')}
                      onClick={() => applyQuickRange(days)}
                    >
                      {days === 7 ? t('overviewPage.last7Days') : days === 14 ? t('overviewPage.last14Days') : t('overviewPage.last30Days')}
                    </button>
                  ))}
                </div>
                <RangePicker
                  style={{ width: 260 }}
                  value={draftPickerValue}
                  allowClear={false}
                  disabledDate={disabledDate}
                  onSelect={(_dateString, date) => { if (Array.isArray(date)) setRangeSelecting(date[date.length - 1] ?? null); else setRangeSelecting(date ?? null) }}
                  onVisibleChange={(visible) => { if (!visible) setRangeSelecting(null) }}
                  onChange={(dates) => {
                    setRangeSelecting(null)
                    if (!dates || !dates[0] || !dates[1]) return
                    setDraftRange([dayjs(dates[0]).format('YYYY-MM-DD'), dayjs(dates[1]).format('YYYY-MM-DD')])
                  }}
                />
                <Button type="primary" size="small" onClick={handleSearch}>{t('common.search')}</Button>
              </div>
            </div>

            {headlineStats.length > 0 ? (
              <div className="halo-stat-strip halo-stat-strip--4">
                {headlineStats.map((s) => (
                  <article key={s.label} className="halo-stat-tile" data-tone={s.tone}>
                    <div className="halo-stat-eyebrow">{s.label}</div>
                    <div className="halo-stat-value">{s.value}</div>
                  </article>
                ))}
              </div>
            ) : null}

            {knowledgeHeadlineStats.length > 0 ? (
              <div className="halo-stat-strip halo-stat-strip--2">
                {knowledgeHeadlineStats.map((s) => (
                  <article key={s.label} className="halo-stat-tile" data-tone={s.tone}>
                    <div className="halo-stat-eyebrow">{s.label}</div>
                    <div className="halo-stat-value">{s.value}</div>
                  </article>
                ))}
              </div>
            ) : null}

            <section className="halo-surface halo-panel">
              <HaloSectionHead eyebrow="Metrics" title={t('overview.metricsPanel')} />
              <div className="halo-stat-strip--auto">
                {metricTiles.map((m) => (
                  <HaloMetricTile key={m.label} label={m.label} value={m.value} />
                ))}
              </div>
            </section>

            <div className="halo-chart-grid-2">
              <article className="halo-work-panel halo-work-panel--chart">
                <h3 className="halo-chart-panel-title">{t('overviewPage.dailyTrend')}</h3>
                {hasTrend ? <DashboardChart spec={trendSpec} height={220} /> : <EmptyChart />}
              </article>
            </div>

            <div className="halo-work-row halo-work-row--tables">
              <article className="halo-work-panel">
                <HaloSectionHead eyebrow="Details" title={t('overviewPage.dailyDetail')} />
                <div className="halo-table-wrap halo-table-wrap--scroll">
                  <table className="halo-table" style={{ minWidth: 480 }}>
                    <thead>
                      <tr>
                        <th>{t('overviewPage.colDate')}</th>
                        <th>{t('overviewPage.colCallCount')}</th>
                        <th>{t('overviewPage.colBilledMinutes')}</th>
                        <th>{t('overviewPage.colAvgMinutes')}</th>
                        <th>{t('overviewPage.colKnowledgeRecallHitRate')}</th>
                        <th>{t('overviewPage.colKnowledgeUnrecognizedRate')}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {dayRows.length > 0 ? dayRows.map((d) => (
                        <tr key={d.day}>
                          <td>{d.day}</td>
                          <td>{d.callCount}</td>
                          <td>{fmtNum(d.totalMinutes)}</td>
                          <td>{fmtNum(d.avgMinutesPerCall)}</td>
                          <td>{(d.knowledgeRecallTotal ?? 0) > 0 ? fmtPct(d.knowledgeRecallHitRate) : '—'}</td>
                          <td>{(d.knowledgeRecallTotal ?? 0) > 0 ? fmtPct(d.knowledgeUnrecognizedRate) : '—'}</td>
                        </tr>
                      )) : (
                        <tr>
                          <td colSpan={6} style={{ padding: 0 }}>
                            <TableEmpty description={t('common.noData')} />
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
              </article>
            </div>
          </div>
        </div>
      </div>
    </BaseLayout>
  )
}
