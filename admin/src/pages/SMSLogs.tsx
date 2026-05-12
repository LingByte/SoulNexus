// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 短信发送日志：Arco Table + 详情抽屉 + 测试发送弹窗。
import { useEffect, useState } from 'react'
import {
  Table,
  Input,
  Select,
  Button,
  Tag,
  Drawer,
  Modal,
  Message,
  Space,
} from '@arco-design/web-react'
import { RefreshCw, Eye, Send, Search } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import {
  listAdminSMSLogs,
  getAdminSMSLog,
  adminSendSMS,
  type SMSLog,
} from '@/services/notificationsApi'

const Option = Select.Option

const STATUS_OPTIONS = [
  { label: '全部', value: '' },
  { label: 'accepted', value: 'accepted' },
  { label: 'sent', value: 'sent' },
  { label: 'success', value: 'success' },
  { label: 'failed', value: 'failed' },
  { label: 'error', value: 'error' },
]

const statusColor = (s?: string) => {
  const v = (s || '').toLowerCase()
  if (v === 'accepted' || v === 'sent' || v === 'success') return 'green'
  if (v === 'failed' || v === 'error') return 'red'
  return 'gray'
}

const SMSLogsPage = () => {
  const [list, setList] = useState<SMSLog[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [status, setStatus] = useState('')
  const [phone, setPhone] = useState('')

  const [detail, setDetail] = useState<SMSLog | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)

  const [showSend, setShowSend] = useState(false)
  const [sendTo, setSendTo] = useState('')
  const [sendContent, setSendContent] = useState('')
  const [sendTemplate, setSendTemplate] = useState('')
  const [sending, setSending] = useState(false)

  const fetchList = async () => {
    setLoading(true)
    try {
      const res = await listAdminSMSLogs({
        page,
        pageSize,
        status: status || undefined,
        to_phone: phone || undefined,
      })
      setList(res.list || [])
      setTotal(res.total || 0)
    } catch (e: any) {
      Message.error(`加载短信日志失败：${e?.msg || e?.message || e}`)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchList()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, status])

  const openDetail = async (id: number) => {
    setDetailLoading(true)
    try {
      setDetail(await getAdminSMSLog(id))
    } catch (e: any) {
      Message.error(`读取详情失败：${e?.msg || e?.message || e}`)
    } finally {
      setDetailLoading(false)
    }
  }

  const submitSend = async () => {
    if (!sendTo.trim()) {
      Message.error('请填写手机号')
      return
    }
    setSending(true)
    try {
      await adminSendSMS({
        to: sendTo,
        content: sendContent || undefined,
        template: sendTemplate || undefined,
      })
      Message.success('已发送')
      setShowSend(false)
      setSendTo('')
      setSendContent('')
      setSendTemplate('')
      fetchList()
    } catch (e: any) {
      Message.error(`发送失败：${e?.msg || e?.message || e}`)
    } finally {
      setSending(false)
    }
  }

  const columns = [
    {
      title: '时间',
      dataIndex: 'sent_at',
      width: 170,
      render: (_: any, r: SMSLog) => r.sent_at || r.created_at || '-',
    },
    {
      title: '手机号',
      dataIndex: 'to_phone',
      width: 160,
      render: (v: string) => <span style={{ fontFamily: 'ui-monospace, monospace' }}>{v}</span>,
    },
    {
      title: '渠道',
      dataIndex: 'provider',
      width: 200,
      render: (_: any, r: SMSLog) => (
        <span>
          <Tag color="arcoblue">{r.provider}</Tag>
          <span className="ml-1 text-xs text-[var(--color-text-3)]">{r.channel_name}</span>
        </span>
      ),
    },
    { title: '模板', dataIndex: 'template', width: 160, render: (v: string) => v || '-' },
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
      render: (_: any, r: SMSLog) => (
        <Button size="mini" type="text" onClick={() => openDetail(r.id)}>
          <span className="inline-flex items-center gap-1"><Eye size={14} /> 详情</span>
        </Button>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <PageHeader
        title="短信发送日志"
        description="查询全平台短信投递记录，并可通过当前启用渠道发送测试短信。"
        actions={
          <Space>
            <Input
              prefix={<Search size={14} />}
              placeholder="手机号精确匹配"
              value={phone}
              onChange={(v) => setPhone(v)}
              onPressEnter={() => { setPage(1); fetchList() }}
              allowClear
              style={{ width: 200 }}
            />
            <Select
              value={status}
              onChange={(v) => { setPage(1); setStatus(v) }}
              placeholder="状态"
              style={{ width: 140 }}
            >
              {STATUS_OPTIONS.map((o) => (
                <Option key={o.value} value={o.value}>{o.label}</Option>
              ))}
            </Select>
            <Button onClick={() => { setPage(1); fetchList() }}>
              <span className="inline-flex items-center gap-1"><RefreshCw size={14} /> 刷新</span>
            </Button>
            <Button type="primary" onClick={() => setShowSend(true)}>
              <span className="inline-flex items-center gap-1"><Send size={14} /> 发送测试短信</span>
            </Button>
          </Space>
        }
      />

      <Table
        rowKey={(r: SMSLog) => String(r.id)}
        loading={loading}
        columns={columns}
        data={list}
        scroll={{ x: 1100 }}
        pagination={{
          current: page,
          pageSize,
          total,
          showTotal: (t: number) => `共 ${t} 条`,
          sizeCanChange: true,
          sizeOptions: [10, 20, 50, 100],
          onChange: (p: number, ps: number) => { setPage(p); setPageSize(ps) },
        }}
      />

      <Drawer
        title="短信日志详情"
        visible={!!detail}
        width={640}
        onCancel={() => setDetail(null)}
        footer={null}
        autoFocus={false}
      >
        {detailLoading ? (
          <div className="p-8 text-center text-[var(--color-text-3)]">加载中...</div>
        ) : detail ? (
          <div className="space-y-3 text-sm">
            <InfoRow label="ID" value={String(detail.id)} />
            <InfoRow label="状态" value={<Tag color={statusColor(detail.status)}>{detail.status}</Tag>} />
            <InfoRow label="provider" value={detail.provider} />
            <InfoRow label="channel" value={detail.channel_name || '-'} />
            <InfoRow label="手机号" value={detail.to_phone} />
            <InfoRow label="模板" value={detail.template || '-'} />
            <InfoRow label="messageId" value={detail.message_id || '-'} />
            <InfoRow label="IP" value={detail.ip_address || '-'} />
            <InfoRow label="sent_at" value={detail.sent_at || '-'} />
            <InfoRow label="created_at" value={detail.created_at || '-'} />
            {detail.error_msg && (
              <div className="rounded-md bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-3 text-red-600 dark:text-red-300 break-all">
                <div className="text-xs font-semibold mb-1">错误信息</div>
                {detail.error_msg}
              </div>
            )}
            {detail.content && (
              <div>
                <div className="text-[var(--color-text-3)] mb-1">内容</div>
                <pre className="m-0 p-3 bg-[var(--color-fill-1)] rounded text-xs overflow-auto max-h-48">
                  {detail.content}
                </pre>
              </div>
            )}
            {detail.raw && (
              <div>
                <div className="text-[var(--color-text-3)] mb-1">raw</div>
                <pre className="m-0 p-3 bg-[var(--color-fill-1)] rounded text-xs overflow-auto max-h-60">
                  {detail.raw}
                </pre>
              </div>
            )}
          </div>
        ) : null}
      </Drawer>

      <Modal
        title="发送测试短信"
        visible={showSend}
        onCancel={() => setShowSend(false)}
        onOk={submitSend}
        confirmLoading={sending}
        okText="发送"
        cancelText="取消"
        autoFocus={false}
      >
        <div className="space-y-3">
          <Field label="手机号（E.164 或国内号）" required>
            <Input value={sendTo} onChange={(v) => setSendTo(v)} placeholder="13800138000 / +14155550100" />
          </Field>
          <Field label="内容（如无模板则用 content）">
            <Input.TextArea
              value={sendContent}
              onChange={(v) => setSendContent(v)}
              rows={3}
              placeholder="测试短信内容"
            />
          </Field>
          <Field label="模板 ID（可选，按当前渠道格式）">
            <Input value={sendTemplate} onChange={(v) => setSendTemplate(v)} />
          </Field>
        </div>
      </Modal>
    </div>
  )
}

const Field = ({
  label,
  required,
  children,
}: {
  label: string
  required?: boolean
  children: React.ReactNode
}) => (
  <div>
    <label className="block text-sm font-medium text-[var(--color-text-2)] mb-1">
      {label}
      {required && <span className="text-red-500 ml-0.5">*</span>}
    </label>
    {children}
  </div>
)

const InfoRow = ({ label, value }: { label: string; value: React.ReactNode }) => (
  <div className="flex items-start gap-3">
    <div className="w-24 shrink-0 text-[var(--color-text-3)]">{label}</div>
    <div className="flex-1 break-all text-[var(--color-text-1)]">{value}</div>
  </div>
)

export default SMSLogsPage
