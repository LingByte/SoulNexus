import { useEffect, useState } from 'react'
import { Drawer, Modal as ArcoModal, Tag } from '@arco-design/web-react'
import dayjs from 'dayjs'
import { Eye, RefreshCw, Search, Send } from 'lucide-react'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Button, DataList, Input, Select } from '@/components/ui'
import type { DataListColumn } from '@/components/ui'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  listAdminSMSLogs,
  getAdminSMSLog,
  adminSendSMS,
  type SMSLog,
} from '@/api/notifications'

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

export default function SMSLogsPage() {
  const [list, setList] = useState<SMSLog[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const pageSize = 20
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
      const res = await listAdminSMSLogs({ page, pageSize, status: status || undefined, to_phone: phone || undefined })
      setList(res.list || [])
      setTotal(res.total || 0)
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, '加载短信日志失败'), 'error')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { void fetchList() }, [page, status])

  const openDetail = async (id: number) => {
    setDetailLoading(true)
    try { setDetail(await getAdminSMSLog(id)) }
    catch (e: unknown) { showAlert(extractApiErrorMessage(e, '读取详情失败'), 'error') }
    finally { setDetailLoading(false) }
  }

  const submitSend = async () => {
    if (!sendTo.trim()) { showAlert('请填写手机号', 'error'); return }
    setSending(true)
    try {
      await adminSendSMS({ to: sendTo, content: sendContent || undefined, template: sendTemplate || undefined })
      showAlert('已发送', 'success')
      setShowSend(false); setSendTo(''); setSendContent(''); setSendTemplate('')
      void fetchList()
    } catch (e: unknown) { showAlert(extractApiErrorMessage(e, '发送失败'), 'error') }
    finally { setSending(false) }
  }

  const columns: DataListColumn<Record<string, unknown>>[] = [
    {
      key: 'time', title: '时间', width: 170,
      render: (_, r) => {
        const ts = String(r.sent_at || r.created_at || '')
        return <span className="text-sm text-neutral-700">{ts ? dayjs(ts).format('YYYY-MM-DD HH:mm:ss') : '—'}</span>
      },
    },
    { key: 'phone', title: '手机号', width: 160, render: (_, r) => <span className="font-mono text-sm text-neutral-700">{String(r.to_phone || '—')}</span> },
    {
      key: 'channel', title: '渠道', width: 200,
      render: (_, r) => (
        <div className="flex items-center gap-1.5">
          <Tag color="arcoblue" className="!rounded-full !text-xs">{String(r.provider)}</Tag>
          <span className="text-xs text-neutral-400">{String(r.channel_name || '')}</span>
        </div>
      ),
    },
    { key: 'template', title: '模板', width: 160, render: (_, r) => <span className="text-sm text-neutral-700">{String(r.template || '—')}</span> },
    { key: 'status', title: '状态', width: 110, render: (_, r) => <Tag color={statusColor(String(r.status))} className="!rounded-full">{String(r.status || '—')}</Tag> },
    {
      key: 'actions', title: '操作', width: 80, align: 'right',
      render: (_, r) => <Button size="mini" icon={<Eye size={12} />} onClick={() => void openDetail(r.id as number)}>详情</Button>,
    },
  ]

  return (
    <BaseLayout title="短信发送日志" description="查询全平台短信投递记录，并可通过启用渠道发送测试短信。">
      <DataList
        data={list as unknown as (SMSLog & Record<string, unknown>)[]}
        columns={columns}
        loading={loading}
        rowKey={(r) => String((r as Record<string, unknown>).id)}
        emptyText="暂无数据"
       
        pagination={{ current: page, pageSize, total, onChange: setPage }}
        header={
          <div className="flex flex-wrap items-end gap-3">
            <div>
              <div className="mb-1 text-xs text-neutral-400">手机号</div>
              <Input prefix={<Search size={14} />} placeholder="手机号精确匹配" value={phone} onChange={setPhone} onPressEnter={() => { setPage(1); void fetchList() }} allowClear style={{ width: 200 }} />
            </div>
            <div>
              <div className="mb-1 text-xs text-neutral-400">状态</div>
              <Select value={status || undefined} onChange={(v) => { setPage(1); setStatus(String(v || '')) }} style={{ width: 140 }} options={STATUS_OPTIONS} />
            </div>
            <Button type="outline" icon={<RefreshCw size={14} />} onClick={() => { setPage(1); void fetchList() }}>刷新</Button>
            <Button type="primary" icon={<Send size={14} />} onClick={() => setShowSend(true)}>发送测试短信</Button>
          </div>
        }
      />

      <Drawer title="短信日志详情" visible={!!detail} width={640} onCancel={() => setDetail(null)} footer={null}>
        {detailLoading ? <div className="p-8 text-center text-neutral-400">加载中...</div> : detail ? (
          <div className="space-y-3 text-sm">
            <InfoRow label="ID" value={String(detail.id)} />
            <InfoRow label="状态" value={<Tag color={statusColor(detail.status)}>{detail.status}</Tag>} />
            <InfoRow label="provider" value={detail.provider} />
            <InfoRow label="channel" value={detail.channel_name || '—'} />
            <InfoRow label="手机号" value={detail.to_phone} />
            <InfoRow label="模板" value={detail.template || '—'} />
            <InfoRow label="messageId" value={detail.message_id || '—'} />
            <InfoRow label="sent_at" value={detail.sent_at ? dayjs(detail.sent_at).format('YYYY-MM-DD HH:mm:ss') : '—'} />
            <InfoRow label="created_at" value={detail.created_at ? dayjs(detail.created_at).format('YYYY-MM-DD HH:mm:ss') : '—'} />
            {detail.error_msg && (
              <div className="rounded-md border border-red-200 bg-red-50 p-3 text-red-600 break-all">
                <div className="mb-1 text-xs font-semibold">错误信息</div>{detail.error_msg}
              </div>
            )}
            {detail.content && (
              <div><div className="mb-1 text-neutral-500">内容</div><pre className="m-0 max-h-48 overflow-auto rounded bg-neutral-50 p-3 text-xs">{detail.content}</pre></div>
            )}
          </div>
        ) : null}
      </Drawer>

      <ArcoModal title="发送测试短信" visible={showSend} onCancel={() => setShowSend(false)} onOk={() => void submitSend()} confirmLoading={sending} okText="发送" cancelText="取消">
        <div className="space-y-3">
          <Field label="手机号（E.164 或国内号）" required><Input value={sendTo} onChange={setSendTo} placeholder="13800138000 / +14155550100" /></Field>
          <Field label="内容（如无模板则用 content）"><Input.TextArea value={sendContent} onChange={setSendContent} rows={3} placeholder="测试短信内容" /></Field>
          <Field label="模板 ID（可选，按当前渠道格式）"><Input value={sendTemplate} onChange={setSendTemplate} /></Field>
        </div>
      </ArcoModal>
    </BaseLayout>
  )
}

function Field({ label, required, children }: { label: string; required?: boolean; children: React.ReactNode }) {
  return <div><label className="mb-1 block text-sm font-medium text-neutral-500">{label}{required && <span className="ml-0.5 text-red-500">*</span>}</label>{children}</div>
}

function InfoRow({ label, value }: { label: string; value: React.ReactNode }) {
  return <div className="flex items-start gap-3"><div className="w-24 shrink-0 text-neutral-500">{label}</div><div className="flex-1 break-all">{value}</div></div>
}
