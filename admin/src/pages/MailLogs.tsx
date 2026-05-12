// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 邮件发送日志：Arco Table + 渲染预览的详情抽屉。
// 后端 mail_logs 表既记录成功也记录预检失败（无渠道/加载失败等）。
import { useEffect, useState } from 'react'
import {
  Table,
  Input,
  Select,
  Button,
  Tag,
  Drawer,
  Message,
  Tabs,
  Space,
} from '@arco-design/web-react'
import { RefreshCw, Eye, Search } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import {
  listAdminMailLogs,
  getAdminMailLog,
  type MailLog,
} from '@/services/notificationsApi'

const Option = Select.Option
const TabPane = Tabs.TabPane

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

const MailLogsPage = () => {
  const [list, setList] = useState<MailLog[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
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
    } catch (e: any) {
      Message.error(`加载邮件日志失败：${e?.msg || e?.message || e}`)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchList()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, status, provider])

  const openDetail = async (id: number) => {
    setDetailLoading(true)
    try {
      const full = await getAdminMailLog(id)
      setDetail(full)
    } catch (e: any) {
      Message.error(`读取详情失败：${e?.msg || e?.message || e}`)
    } finally {
      setDetailLoading(false)
    }
  }

  const columns = [
    {
      title: '时间',
      dataIndex: 'sent_at',
      width: 170,
      render: (_: any, r: MailLog) => r.sent_at || r.created_at || '-',
    },
    { title: '收件人', dataIndex: 'to_email', width: 200, ellipsis: true },
    { title: '主题', dataIndex: 'subject', ellipsis: true },
    {
      title: '渠道',
      dataIndex: 'provider',
      width: 180,
      render: (_: any, r: MailLog) => (
        <span>
          <Tag color="arcoblue">{r.provider}</Tag>
          <span className="ml-1 text-xs text-[var(--color-text-3)]">{r.channel_name}</span>
        </span>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 110,
      render: (s: string) => <Tag color={statusColor(s)}>{s || '-'}</Tag>,
    },
    {
      title: '操作',
      key: '__actions__',
      width: 90,
      fixed: 'right' as const,
      render: (_: any, r: MailLog) => (
        <Button size="mini" type="text" onClick={() => openDetail(r.id)}>
          <span className="inline-flex items-center gap-1"><Eye size={14} /> 详情</span>
        </Button>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <PageHeader
        title="邮件发送日志"
        description="查询全平台邮件投递与回执；预检失败（无渠道/模板缺失）会以 provider=none 入库。"
        actions={
          <Space>
            <Input
              prefix={<Search size={14} />}
              placeholder="按收件人 / 主题 / 渠道名过滤当前页"
              value={keyword}
              onChange={(v) => setKeyword(v)}
              onPressEnter={() => fetchList()}
              allowClear
              style={{ width: 280 }}
            />
            <Select
              placeholder="状态"
              value={status}
              onChange={(v) => {
                setPage(1)
                setStatus(v)
              }}
              style={{ width: 140 }}
            >
              {STATUS_OPTIONS.map((o) => (
                <Option key={o.value} value={o.value}>{o.label}</Option>
              ))}
            </Select>
            <Select
              placeholder="供应商"
              value={provider}
              onChange={(v) => {
                setPage(1)
                setProvider(v)
              }}
              style={{ width: 200 }}
            >
              {PROVIDER_OPTIONS.map((o) => (
                <Option key={o.value} value={o.value}>{o.label}</Option>
              ))}
            </Select>
            <Button onClick={() => { setPage(1); fetchList() }}>
              <span className="inline-flex items-center gap-1"><RefreshCw size={14} /> 刷新</span>
            </Button>
          </Space>
        }
      />

      <Table
        rowKey={(r: MailLog) => String(r.id)}
        loading={loading}
        columns={columns}
        data={list}
        scroll={{ x: 1200 }}
        pagination={{
          current: page,
          pageSize,
          total,
          showTotal: (t: number) => `共 ${t} 条`,
          sizeCanChange: true,
          sizeOptions: [10, 20, 50, 100],
          onChange: (p: number, ps: number) => {
            setPage(p)
            setPageSize(ps)
          },
        }}
      />

      <Drawer
        title="邮件日志详情"
        visible={!!detail}
        width={760}
        onCancel={() => setDetail(null)}
        footer={null}
        autoFocus={false}
      >
        {detailLoading ? (
          <div className="p-8 text-center text-[var(--color-text-3)]">加载中...</div>
        ) : detail ? (
          <div className="space-y-3 text-sm">
            <InfoRow label="ID" value={String(detail.id)} />
            <InfoRow
              label="状态"
              value={<Tag color={statusColor(detail.status)}>{detail.status}</Tag>}
            />
            <InfoRow label="provider" value={detail.provider} />
            <InfoRow label="channel" value={detail.channel_name || '-'} />
            <InfoRow label="收件人" value={detail.to_email} />
            <InfoRow label="主题" value={detail.subject} />
            <InfoRow label="messageId" value={detail.message_id || '-'} />
            <InfoRow label="重试次数" value={String(detail.retry_count ?? 0)} />
            <InfoRow label="IP" value={detail.ip_address || '-'} />
            <InfoRow label="sent_at" value={detail.sent_at || '-'} />
            <InfoRow label="created_at" value={detail.created_at || '-'} />
            {detail.error_msg && (
              <div className="rounded-md bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-3 text-red-600 dark:text-red-300 break-all">
                <div className="text-xs font-semibold mb-1">错误信息</div>
                {detail.error_msg}
              </div>
            )}
            {(detail.html_body || detail.body || detail.raw) && (
              <Tabs defaultActiveTab={detail.html_body ? 'render' : detail.body ? 'text' : 'raw'}>
                {detail.html_body && (
                  <TabPane key="render" title="HTML 渲染">
                    <iframe
                      title="mail-html"
                      sandbox=""
                      srcDoc={detail.html_body}
                      style={{
                        width: '100%',
                        height: 360,
                        border: '1px solid var(--color-border-2)',
                        borderRadius: 4,
                        background: '#fff',
                      }}
                    />
                  </TabPane>
                )}
                {detail.html_body && (
                  <TabPane key="html-source" title="HTML 源码">
                    <pre className="m-0 p-3 bg-[var(--color-fill-1)] rounded text-xs overflow-auto max-h-80">
                      {detail.html_body}
                    </pre>
                  </TabPane>
                )}
                {detail.body && (
                  <TabPane key="text" title="文本正文">
                    <pre className="m-0 p-3 bg-[var(--color-fill-1)] rounded text-xs overflow-auto max-h-80">
                      {detail.body}
                    </pre>
                  </TabPane>
                )}
                {detail.raw && (
                  <TabPane key="raw" title="raw">
                    <pre className="m-0 p-3 bg-[var(--color-fill-1)] rounded text-xs overflow-auto max-h-80">
                      {detail.raw}
                    </pre>
                  </TabPane>
                )}
              </Tabs>
            )}
          </div>
        ) : null}
      </Drawer>
    </div>
  )
}

const InfoRow = ({ label, value }: { label: string; value: React.ReactNode }) => (
  <div className="flex items-start gap-3">
    <div className="w-24 shrink-0 text-[var(--color-text-3)]">{label}</div>
    <div className="flex-1 break-all text-[var(--color-text-1)]">{value}</div>
  </div>
)

export default MailLogsPage
