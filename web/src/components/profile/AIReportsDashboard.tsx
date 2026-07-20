import { useCallback, useEffect, useMemo, useState } from 'react'
import { Card, Grid, Space, Spin, Statistic, Table, Typography } from '@arco-design/web-react'
import dayjs from 'dayjs'
import { Button } from '@/components/ui'
import { DashboardChart, EmptyChart, useVChartTheme } from '@/components/Dashboard/ChartWidgets'
import {
  downloadCallerAttributesExport,
  getAICallAnalytics,
  getAICallerAttributes,
  type CallAnalyticsDashboard,
  type CallerAttributeRow,
} from '@/api/aiReports'
import { useTranslation } from '@/i18n'
import { formatLatencyKpi, coerceLatencyMs } from '@/utils/formatLatency'

const CHART_COLORS = ['#165dff', '#14c9c9', '#f7ba1e', '#722ed1', '#f5319d', '#00b42a']

type Props = {
  /** When set, render from stored report summary instead of live API. */
  embedded?: CallAnalyticsDashboard | null
}

export default function AIReportsDashboard({ embedded }: Props) {
  const { t } = useTranslation()
  useVChartTheme()
  const [days, setDays] = useState(14)
  const [loading, setLoading] = useState(!embedded)
  const [data, setData] = useState<CallAnalyticsDashboard | null>(embedded ?? null)
  const [provinceFilter, setProvinceFilter] = useState<string | undefined>()
  const [callerRows, setCallerRows] = useState<CallerAttributeRow[]>([])
  const [callerTotal, setCallerTotal] = useState(0)
  const [callerLoading, setCallerLoading] = useState(false)
  const [exporting, setExporting] = useState(false)

  const reload = useCallback(async () => {
    if (embedded) {
      setData(embedded)
      return
    }
    setLoading(true)
    try {
      const res = await getAICallAnalytics(days)
      if (res.code === 200 && res.data) setData(res.data)
    } finally {
      setLoading(false)
    }
  }, [days, embedded])

  const reloadCallers = useCallback(async () => {
    if (embedded) return
    setCallerLoading(true)
    try {
      const res = await getAICallerAttributes({ days, province: provinceFilter, page: 1, size: 20 })
      if (res.code === 200 && res.data) {
        setCallerRows(res.data.items)
        setCallerTotal(res.data.total)
      }
    } finally {
      setCallerLoading(false)
    }
  }, [days, embedded, provinceFilter])

  useEffect(() => { void reload() }, [reload])
  useEffect(() => { void reloadCallers() }, [reloadCallers])

  const summary = data?.summary
  const telemetry = data?.turnTelemetry
  const pipelineKpi = formatLatencyKpi(summary?.avgPipelineMs)
  const llmFirstKpi = formatLatencyKpi(summary?.avgLlmFirstMs)

  const trendSpec = useMemo(() => {
    const days = data?.callTrend || []
    if (!days.length) return null
    const values: { day: string; series: string; value: number }[] = []
    for (const d of days) {
      const day = d.day.slice(5)
      values.push(
        { day, series: t('profile.aiReportChartCalls'), value: d.callCount },
        { day, series: t('profile.aiReportChartMinutes'), value: Math.round(d.totalMinutes) },
      )
    }
    return {
      type: 'common',
      padding: { left: 8, right: 8, top: 28, bottom: 8 },
      color: CHART_COLORS,
      data: [{ id: 'trend', values }],
      series: [{
        type: 'bar',
        dataIndex: 0,
        xField: 'day',
        yField: 'value',
        seriesField: 'series',
        bar: { style: { cornerRadius: 4 } },
      }],
      legends: { visible: true, orient: 'top' },
      tooltip: { visible: true },
      axes: [{ orient: 'bottom', type: 'band' }, { orient: 'left' }],
    }
  }, [data, t])

  const barSpec = (buckets: { label: string; value: number }[] | undefined, horizontal = false) => {
    if (!buckets?.length) return null
    return {
      type: 'bar',
      padding: { left: horizontal ? 72 : 8, right: 8, top: 8, bottom: 8 },
      color: CHART_COLORS,
      data: [{ id: 'b', values: buckets.map((b) => ({ name: b.label, value: b.value })) }],
      xField: horizontal ? 'value' : 'name',
      yField: horizontal ? 'name' : 'value',
      direction: horizontal ? 'horizontal' : 'vertical',
      bar: { style: { cornerRadius: 4 } },
      tooltip: { visible: true },
      axes: [
        { orient: horizontal ? 'left' : 'bottom', type: horizontal ? 'band' : 'band' },
        { orient: horizontal ? 'bottom' : 'left' },
      ],
    }
  }

  const pieSpec = (buckets: { label: string; value: number }[] | undefined) => {
    if (!buckets?.length) return null
    return {
      type: 'pie',
      padding: { top: 8, bottom: 8, left: 8, right: 8 },
      color: CHART_COLORS,
      data: [{ id: 'p', values: buckets.map((r) => ({ name: r.label, value: r.value })) }],
      valueField: 'value',
      categoryField: 'name',
      outerRadius: 0.78,
      innerRadius: 0.52,
      legends: { visible: true, orient: 'bottom' },
      label: { visible: true },
    }
  }

  const durationSpec = useMemo(() => barSpec(data?.durationBuckets), [data])
  const turnSpec = useMemo(() => barSpec(data?.turnBuckets, true), [data])
  const provinceSpec = useMemo(() => pieSpec(data?.provinceBuckets), [data])

  const telemetrySpec = useMemo(() => {
    if (!telemetry) return null
    const buckets = [
      { name: t('profile.aiReportTelemetryLlmFirst'), value: coerceLatencyMs(telemetry.avgLlmFirstMs) },
      { name: t('profile.aiReportTelemetryTts'), value: coerceLatencyMs(telemetry.avgTtsMs) },
      { name: t('profile.aiReportTelemetryPipeline'), value: coerceLatencyMs(telemetry.avgPipelineMs) },
      { name: t('profile.aiReportTelemetryRecall'), value: coerceLatencyMs(telemetry.avgRecallMs) },
    ].filter((b) => b.value > 0)
    if (!buckets.length) return null
    return barSpec(buckets.map((b) => ({ label: b.name, value: b.value })))
  }, [telemetry, t])

  const handleExport = async () => {
    setExporting(true)
    try {
      await downloadCallerAttributesExport(days, provinceFilter)
    } finally {
      setExporting(false)
    }
  }

  if (loading && !data) {
    return <div className="flex justify-center py-12"><Spin /></div>
  }

  return (
    <div>
      {!embedded ? (
        <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
          <Typography.Text type="secondary">{t('profile.aiReportDashboardHint')}</Typography.Text>
          <Space>
            {[7, 14, 30].map((d) => (
              <Button key={d} size="small" type={days === d ? 'primary' : 'secondary'} onClick={() => setDays(d)}>
                {t('profile.aiReportDays', { n: String(d) })}
              </Button>
            ))}
          </Space>
        </div>
      ) : null}

      <Grid cols={{ xs: 2, sm: 3, md: 4, lg: 6 }} colGap={12} rowGap={12} className="mb-4">
        <Grid.GridItem><Card size="small"><Statistic title={t('profile.aiReportKpiCalls')} value={summary?.callCount ?? 0} /></Card></Grid.GridItem>
        <Grid.GridItem><Card size="small"><Statistic title={t('profile.aiReportKpiConnectRate')} value={summary?.connectedRate ?? 0} suffix="%" precision={1} /></Card></Grid.GridItem>
        <Grid.GridItem><Card size="small"><Statistic title={t('profile.aiReportKpiAvgTurns')} value={summary?.avgTurns ?? 0} precision={1} /></Card></Grid.GridItem>
        <Grid.GridItem><Card size="small"><Statistic title={t('profile.aiReportKpiAvgDuration')} value={summary?.avgDurationSec ?? 0} suffix="s" precision={0} /></Card></Grid.GridItem>
        <Grid.GridItem><Card size="small"><Statistic title={t('profile.aiReportKpiLlmFirst')} value={llmFirstKpi.value} suffix={llmFirstKpi.suffix} precision={llmFirstKpi.precision} /></Card></Grid.GridItem>
        <Grid.GridItem><Card size="small"><Statistic title={t('profile.aiReportKpiOneTime')} value={summary?.oneTimeResolutionRate ?? 0} suffix="%" precision={1} /></Card></Grid.GridItem>
        <Grid.GridItem><Card size="small"><Statistic title={t('profile.aiReportKpiQuoteRate')} value={summary?.quoteRate ?? 0} suffix="%" precision={1} /></Card></Grid.GridItem>
        <Grid.GridItem><Card size="small"><Statistic title={t('profile.aiReportKpiRagHit')} value={telemetry?.ragHitRate ?? 0} suffix="%" precision={1} /></Card></Grid.GridItem>
        <Grid.GridItem><Card size="small"><Statistic title={t('profile.aiReportKpiInterrupts')} value={telemetry?.interruptCount ?? 0} /></Card></Grid.GridItem>
        <Grid.GridItem><Card size="small"><Statistic title={t('profile.aiReportKpiPipeline')} value={pipelineKpi.value} suffix={pipelineKpi.suffix} precision={pipelineKpi.precision} /></Card></Grid.GridItem>
      </Grid>

      <Grid cols={{ xs: 1, md: 2 }} colGap={16} rowGap={16}>
        <Grid.GridItem>
          <Card title={t('profile.aiReportChartTrend')} bordered={false} className="rounded-xl">
            {trendSpec ? <DashboardChart spec={trendSpec} height={240} /> : <EmptyChart tall />}
          </Card>
        </Grid.GridItem>
        <Grid.GridItem>
          <Card title={t('profile.aiReportChartTelemetry')} bordered={false} className="rounded-xl">
            {telemetrySpec ? <DashboardChart spec={telemetrySpec} height={240} /> : <EmptyChart tall />}
          </Card>
        </Grid.GridItem>
        <Grid.GridItem>
          <Card title={t('profile.aiReportChartDuration')} bordered={false} className="rounded-xl">
            {durationSpec ? <DashboardChart spec={durationSpec} height={240} /> : <EmptyChart tall />}
          </Card>
        </Grid.GridItem>
        <Grid.GridItem>
          <Card title={t('profile.aiReportChartTurns')} bordered={false} className="rounded-xl">
            {turnSpec ? <DashboardChart spec={turnSpec} height={240} /> : <EmptyChart tall />}
          </Card>
        </Grid.GridItem>
        <Grid.GridItem>
          <Card title={t('profile.aiReportChartProvince')} bordered={false} className="rounded-xl">
            {provinceSpec ? <DashboardChart spec={provinceSpec} height={240} /> : <EmptyChart tall />}
          </Card>
        </Grid.GridItem>
      </Grid>

      {!embedded ? (
        <Card
          className="mt-4 rounded-xl"
          title={t('profile.aiReportCallerDetailTitle')}
          extra={(
            <Space>
              <Button size="small" loading={exporting} onClick={() => void handleExport()}>
                {t('profile.aiReportExportExcel')}
              </Button>
            </Space>
          )}
        >
          <div className="mb-3 flex flex-wrap gap-2">
            <Button
              size="mini"
              type={!provinceFilter ? 'primary' : 'secondary'}
              onClick={() => setProvinceFilter(undefined)}
            >
              {t('profile.aiReportAllProvinces')}
            </Button>
            {(data?.provinceBuckets || []).map((p) => (
              <Button
                key={p.label}
                size="mini"
                type={provinceFilter === p.label ? 'primary' : 'secondary'}
                onClick={() => setProvinceFilter(p.label)}
              >
                {p.label} ({p.value})
              </Button>
            ))}
          </div>
          <Table
            loading={callerLoading}
            rowKey="id"
            pagination={{ total: callerTotal, pageSize: 20, showTotal: true }}
            data={callerRows}
            columns={[
              { title: t('profile.aiReportColNumber'), dataIndex: 'fromNumber' },
              { title: t('profile.aiReportColProvince'), dataIndex: 'callerProvince' },
              { title: t('profile.aiReportColCity'), dataIndex: 'callerCity' },
              { title: t('profile.aiReportColCarrier'), dataIndex: 'callerCarrier' },
              { title: t('profile.aiReportColDuration'), dataIndex: 'durationSec', render: (v: number) => `${v}s` },
              { title: t('profile.aiReportColTurns'), dataIndex: 'turnCount' },
              { title: t('profile.aiReportColEnded'), dataIndex: 'endedAt', render: (v: string) => v ? dayjs(v).format('YYYY-MM-DD HH:mm:ss') : '—' },
            ]}
          />
        </Card>
      ) : null}
    </div>
  )
}
