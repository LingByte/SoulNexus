import { useCallback, useEffect, useMemo, useState } from 'react'
import { Card, Divider, Drawer, Space, Table, Tag, Typography } from '@arco-design/web-react'
import dayjs from 'dayjs'
import { Button, TableEmpty } from '@/components/ui'
import { useTranslation } from '@/i18n'
import { getAIReport, listAIReports, type AIReport, type CallAnalyticsDashboard } from '@/api/aiReports'
import AIReportsDashboard from '@/components/profile/AIReportsDashboard'

function summaryToDashboard(summary: Record<string, unknown> | undefined): CallAnalyticsDashboard | null {
  if (!summary || typeof summary !== 'object') return null
  const s = summary as CallAnalyticsDashboard['summary'] & CallAnalyticsDashboard
  if (!s.callCount && !s.durationBuckets?.length && !s.transferOutcomeBuckets?.length) return null
  return {
    rangeStart: String(s.rangeStart || ''),
    rangeEnd: String(s.rangeEnd || ''),
    summary: s,
    callTrend: s.callTrend || [],
    durationBuckets: s.durationBuckets || [],
    turnBuckets: s.turnBuckets || [],
    provinceBuckets: s.provinceBuckets || [],
    endStatusBuckets: s.endStatusBuckets || [],
    turnTelemetry: s.turnTelemetry,
    transferOutcomeBuckets: s.transferOutcomeBuckets || [],
    transferReasonBuckets: s.transferReasonBuckets || [],
    transferTrend: s.transferTrend || [],
  }
}

export default function AIReportsPanel() {
  const { t } = useTranslation()
  const [rows, setRows] = useState<AIReport[]>([])
  const [filter, setFilter] = useState<'all' | 'daily' | 'weekly'>('all')
  const [detail, setDetail] = useState<AIReport | null>(null)

  const reload = useCallback(async () => {
    const res = await listAIReports({
      type: filter === 'all' ? undefined : filter,
      page: 1,
      size: 50,
    })
    if (res.code === 200 && res.data?.items) setRows(res.data.items)
  }, [filter])

  useEffect(() => { void reload() }, [reload])

  const openDetail = async (row: AIReport) => {
    const res = await getAIReport(row.id)
    if (res.code === 200 && res.data) setDetail(res.data)
    else setDetail(row)
  }

  const detailDashboard = useMemo(
    () => summaryToDashboard(detail?.summary as Record<string, unknown> | undefined),
    [detail],
  )

  return (
    <Card bordered={false} className="rounded-xl">
      <Typography.Title heading={6} style={{ marginTop: 0 }}>{t('profile.aiReportDashboardTitle')}</Typography.Title>
      <Typography.Paragraph type="secondary" style={{ marginTop: 0 }}>
        {t('profile.aiReportsDesc')}
      </Typography.Paragraph>

      <AIReportsDashboard />

      <Divider style={{ margin: '24px 0 16px' }} />

      <Typography.Title heading={6}>{t('profile.aiReportHistory')}</Typography.Title>
      <Space style={{ marginBottom: 12 }}>
        <Button type={filter === 'all' ? 'primary' : 'secondary'} size="small" onClick={() => setFilter('all')}>
          {t('common.all')}
        </Button>
        <Button type={filter === 'daily' ? 'primary' : 'secondary'} size="small" onClick={() => setFilter('daily')}>
          {t('profile.aiReportDaily')}
        </Button>
        <Button type={filter === 'weekly' ? 'primary' : 'secondary'} size="small" onClick={() => setFilter('weekly')}>
          {t('profile.aiReportWeekly')}
        </Button>
      </Space>
      <Table
        data={rows}
        rowKey="id"
        pagination={false}
        noDataElement={<TableEmpty description={t('profile.aiReportsEmpty')} />}
        columns={[
          { title: t('profile.aiReportTitle'), dataIndex: 'title', ellipsis: true },
          {
            title: t('profile.aiReportType'),
            dataIndex: 'reportType',
            width: 88,
            render: (v: string) => (
              <Tag color={v === 'weekly' ? 'arcoblue' : 'green'}>
                {v === 'weekly' ? t('profile.aiReportWeekly') : t('profile.aiReportDaily')}
              </Tag>
            ),
          },
          {
            title: t('profile.aiReportPushed'),
            width: 120,
            render: (_: unknown, row: AIReport) => (
              <Space size={4}>
                {row.pushedInbox ? <Tag size="small">{t('profile.aiReportInbox')}</Tag> : null}
                {row.pushedEmail ? <Tag size="small">{t('profile.aiReportEmail')}</Tag> : null}
                {row.pushedWebhook ? <Tag size="small">{t('profile.aiReportWebhook')}</Tag> : null}
                {row.pushedIm ? <Tag size="small">{t('profile.aiReportIM')}</Tag> : null}
                {!row.pushedInbox && !row.pushedEmail && !row.pushedWebhook && !row.pushedIm ? (
                  <Typography.Text type="secondary">—</Typography.Text>
                ) : null}
              </Space>
            ),
          },
          { title: t('profile.aiReportCreated'), dataIndex: 'createdAt', width: 180, render: (v: string) => v ? dayjs(v).format('YYYY-MM-DD HH:mm:ss') : '—' },
          {
            title: t('common.actions'),
            width: 80,
            render: (_: unknown, row: AIReport) => (
              <Button size="small" onClick={() => void openDetail(row)}>{t('profile.aiReportView')}</Button>
            ),
          },
        ]}
      />
      <Drawer width={900} title={detail?.title} visible={!!detail} onCancel={() => setDetail(null)} footer={null}>
        {detailDashboard ? (
          <AIReportsDashboard embedded={detailDashboard} />
        ) : detail?.bodyHtml ? (
          <div dangerouslySetInnerHTML={{ __html: detail.bodyHtml }} />
        ) : (
          <Typography.Text type="secondary">{t('common.noData')}</Typography.Text>
        )}
      </Drawer>
    </Card>
  )
}
