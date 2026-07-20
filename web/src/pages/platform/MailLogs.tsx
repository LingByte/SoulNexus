import { useEffect, useState } from 'react'
import { Drawer, Tag } from '@arco-design/web-react'
import dayjs from 'dayjs'
import { Eye, RefreshCw, Search } from 'lucide-react'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Button, DataList, Input, Select } from '@/components/ui'
import type { DataListColumn } from '@/components/ui'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  listAdminMailLogs,
  getAdminMailLog,
  type MailLog,
} from '@/api/notifications'

const STATUS_OPTIONS = [
  { label: '全部', value: '' },
  { label: 'sent', value: 'sent' },
  { label: 'delivered', value: 'delivered' },
  { label: 'failed', value: 'failed' },
  { label: 'soft_bounce', value: 'soft_bounce' },
  { label: 'invalid', value: 'invalid' },
  { label: 'spam', value: 'spam' },
  { label: 'opened', value: 'opened' },
  { label: 'clicked', value: 'clicked' },
  { label: 'unsubscribed', value: 'unsubscribed' },
  { label: 'unknown', value: 'unknown' },
]

const PROVIDER_OPTIONS = [
  { label: '全部', value: '' },
  { label: 'smtp', value: 'smtp' },
  { label: 'sendcloud', value: 'sendcloud' },
  { label: 'multi（多渠道汇总失败）', value: 'multi' },
  { label: 'none（预检失败）', value: 'none' },
]

const statusColor = (s?: string) => {
  const v = (s || '').toLowerCase()
  if (v === 'sent' || v === 'delivered' || v === 'opened' || v === 'clicked') return 'green'
  if (v === 'failed' || v === 'invalid' || v === 'spam' || v === 'soft_bounce') return 'red'
  if (v === 'unsubscribed') return 'orange'
  return 'gray'
}

export default function MailLogsPage() {
  const [list, setList] = useState<MailLog[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const pageSize = 20
  const [status, setStatus] = useState('')
  const [provider, setProvider] = useState('')
  const [keyword, setKeyword] = useState('')
  const [detail, setDetail] = useState<MailLog | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)

  const fetchList = async () => {
    setLoading(true)
    try {
      const res = await listAdminMailLogs({
        page,
        pageSize,
        status: status || undefined,
        provider: provider || undefined,
      })
      let items = res.list || []
      if (keyword.trim()) {
        const k = keyword.trim().toLowerCase()
        items = items.filter(
          (m) =>
            m.to_email?.toLowerCase().includes(k) ||
            m.subject?.toLowerCase().includes(k) ||
            m.channel_name?.toLowerCase().includes(k),
        )
      }
      setList(items)
      setTotal(res.total || 0)
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, '加载邮件日志失败'), 'error')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void fetchList()
  }, [page, pageSize, status, provider])

  const openDetail = async (id: number) => {
    setDetailLoading(true)
    try {
      const full = await getAdminMailLog(id)
      setDetail(full)
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, '读取详情失败'), 'error')
    } finally {
      setDetailLoading(false)
    }
  }

  const columns: DataListColumn<Record<string, unknown>>[] = [
    {
      key: 'time',
      title: '时间',
      width: 170,
      render: (_, r) => {
        const ts = String(r.sent_at || r.created_at || '')
        const formatted = ts ? dayjs(ts).format('YYYY-MM-DD HH:mm:ss') : '—'
        return <span className="text-sm text-neutral-700">{formatted}</span>
      },
    },
    {
      key: 'to',
      title: '收件人',
      width: 200,
      render: (_, r) => (
        <div className="truncate text-sm text-neutral-900">{String(r.to_email || '—')}</div>
      ),
    },
    {
      key: 'subject',
      title: '主题',
      render: (_, r) => (
        <div className="truncate text-sm text-neutral-900">{String(r.subject || '—')}</div>
      ),
    },
    {
      key: 'channel',
      title: '渠道',
      width: 180,
      render: (_, r) => (
        <div className="flex items-center gap-1.5">
          <Tag color="arcoblue" className="!rounded-full !text-xs">{String(r.provider)}</Tag>
          <span className="text-xs text-neutral-400">{String(r.channel_name || '')}</span>
        </div>
      ),
    },
    {
      key: 'status',
      title: '状态',
      width: 110,
      render: (_, r) => (
        <Tag color={statusColor(String(r.status))} className="!rounded-full">{String(r.status || '—')}</Tag>
      ),
    },
    {
      key: 'actions',
      title: '操作',
      width: 80,
     
      align: 'right',
      render: (_, r) => (
        <Button size="mini" icon={<Eye size={12} />} onClick={() => void openDetail(r.id as number)}>
          详情
        </Button>
      ),
    },
  ]

  return (
    <BaseLayout title="邮件发送日志" description="查询全平台邮件投递与回执；预检失败会以 provider=none 入库。">
      <DataList
        data={list as unknown as (MailLog & Record<string, unknown>)[]}
        columns={columns}
        loading={loading}
        rowKey={(r) => String((r as Record<string, unknown>).id)}
        emptyText="暂无数据"
       
        pagination={{ current: page, pageSize, total, onChange: setPage }}
        header={
          <div className="flex flex-wrap items-end gap-3">
            <div>
              <div className="mb-1 text-xs text-neutral-400">关键词</div>
              <Input
                prefix={<Search size={14} />}
                placeholder="按收件人 / 主题 / 渠道名过滤"
                value={keyword}
                onChange={setKeyword}
                onPressEnter={() => void fetchList()}
                allowClear
                style={{ width: 240 }}
              />
            </div>
            <div>
              <div className="mb-1 text-xs text-neutral-400">状态</div>
              <Select
                value={status || undefined}
                onChange={(v) => { setPage(1); setStatus(String(v || '')) }}
                style={{ width: 140 }}
                options={STATUS_OPTIONS}
              />
            </div>
            <div>
              <div className="mb-1 text-xs text-neutral-400">供应商</div>
              <Select
                value={provider || undefined}
                onChange={(v) => { setPage(1); setProvider(String(v || '')) }}
                style={{ width: 200 }}
                options={PROVIDER_OPTIONS}
              />
            </div>
            <Button type="outline" icon={<RefreshCw size={14} />} onClick={() => { setPage(1); void fetchList() }}>
              刷新
            </Button>
          </div>
        }
      />

      <Drawer
        title="邮件日志详情"
        visible={!!detail}
        width={760}
        onCancel={() => setDetail(null)}
        footer={null}
      >
        {detailLoading ? (
          <div className="p-8 text-center text-neutral-400">加载中...</div>
        ) : detail ? (
          <div className="space-y-3 text-sm">
            <InfoRow label="ID" value={String(detail.id)} />
            <InfoRow label="状态" value={<Tag color={statusColor(detail.status)}>{detail.status}</Tag>} />
            <InfoRow label="provider" value={detail.provider} />
            <InfoRow label="channel" value={detail.channel_name || '—'} />
            <InfoRow label="收件人" value={detail.to_email} />
            <InfoRow label="主题" value={detail.subject} />
            <InfoRow label="messageId" value={detail.message_id || '—'} />
            {detail.error_msg && (
              <div className="rounded-md border border-red-200 bg-red-50 p-3 text-red-600 break-all">
                <div className="mb-1 text-xs font-semibold">错误信息</div>
                {detail.error_msg}
              </div>
            )}
            {detail.html_body && (
              <div>
                <div className="mb-1 text-neutral-500">HTML 渲染</div>
                <iframe
                  title="mail-html"
                  sandbox=""
                  srcDoc={detail.html_body}
                  style={{ width: '100%', height: 360, border: '1px solid var(--color-border-2)', borderRadius: 4, background: '#fff' }}
                />
              </div>
            )}
          </div>
        ) : null}
      </Drawer>
    </BaseLayout>
  )
}

const InfoRow = ({ label, value }: { label: string; value: React.ReactNode }) => (
  <div className="flex items-start gap-3">
    <div className="w-24 shrink-0 text-neutral-500">{label}</div>
    <div className="flex-1 break-all">{value}</div>
  </div>
)
