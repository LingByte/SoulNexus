import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, Pencil, Trash2, RefreshCw, Mail, MessageSquare } from 'lucide-react'
import { Modal, Tag } from '@arco-design/web-react'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Button, DataList } from '@/components/ui'
import type { DataListColumn } from '@/components/ui'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'
import { cn } from '@/utils/cn'
import {
  listNotificationChannels,
  deleteNotificationChannel,
  type NotificationChannel,
} from '@/api/notifications'

type ChannelType = 'email' | 'sms'

export default function NotificationChannelsPage() {
  const navigate = useNavigate()
  const [tab, setTab] = useState<ChannelType>('email')
  const [list, setList] = useState<NotificationChannel[]>([])
  const [loading, setLoading] = useState(false)

  const fetchList = async () => {
    setLoading(true)
    try {
      const res = await listNotificationChannels({ type: tab, page: 1, pageSize: 200 })
      setList(res.list || [])
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, '加载渠道失败'), 'error')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void fetchList()
  }, [tab])

  const confirmDelete = (row: NotificationChannel) => {
    Modal.confirm({
      title: '删除渠道',
      content: `确定要删除渠道「${row.name}」吗？`,
      onOk: async () => {
        await deleteNotificationChannel(row.id)
        showAlert('删除成功', 'success')
        await fetchList()
      },
    })
  }

  const columns: DataListColumn<Record<string, unknown>>[] = [
    {
      key: 'icon',
      width: 40,
      render: (_, r) => (
        <div className={cn(
          'flex h-9 w-9 shrink-0 items-center justify-center rounded-full',
          r.type === 'email' ? 'bg-blue-50 text-blue-600' : 'bg-emerald-50 text-emerald-600',
        )}>
          {r.type === 'email' ? <Mail size={16} /> : <MessageSquare size={16} />}
        </div>
      ),
    },
    {
      key: 'info',
      title: '名称',
      render: (_, r) => (
        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-medium text-neutral-900">{String(r.name || '—')}</div>
          <div className="mt-0.5 text-xs text-neutral-500">{String(r.remark || '')}</div>
        </div>
      ),
    },
    {
      key: 'meta',
      width: 200,
      render: (_, r) => (
        <div className="flex items-center gap-3 text-xs text-neutral-500">
          <Tag className="!rounded-full !text-xs">{String(r.type)}</Tag>
          <span>排序: {String(r.sortOrder ?? 0)}</span>
          <Tag color={r.enabled ? 'green' : 'gray'} className="!rounded-full !text-xs">
            {r.enabled ? '启用' : '禁用'}
          </Tag>
        </div>
      ),
    },
    {
      key: 'actions',
      width: 80,
     
      align: 'right',
      render: (_, r) => (
        <div className="flex items-center justify-end gap-1">
          <Button size="mini" icon={<Pencil size={12} />} onClick={() => navigate(`/platform/notification-channels/${String(r.id)}/edit`)} />
          <Button size="mini" status="danger" icon={<Trash2 size={12} />} onClick={() => confirmDelete(r as unknown as NotificationChannel)} />
        </div>
      ),
    },
  ]

  return (
    <BaseLayout title="通知渠道" description="管理邮件 / 短信发送供应商，启用后按排序轮询与故障切换发送。"
      actions={
        <div className="flex gap-2">
          <Button type="outline" icon={<RefreshCw size={14} />} onClick={() => void fetchList()}>刷新</Button>
          <Button type="primary" icon={<Plus size={14} />} onClick={() => navigate(`/platform/notification-channels/new?type=${tab}`)}>
            新建{tab === 'email' ? '邮件' : '短信'}渠道
          </Button>
        </div>
      }
    >
      <div className="mx-auto max-w-6xl space-y-4">
        <div className="flex gap-1 rounded-lg bg-neutral-100 p-1">
          {(['email', 'sms'] as const).map((t) => (
            <button
              key={t}
              type="button"
              className={cn(
                'flex-1 rounded-md px-4 py-2 text-sm font-medium transition',
                tab === t ? 'bg-white text-neutral-900 shadow-sm' : 'text-neutral-500 hover:text-neutral-700',
              )}
              onClick={() => setTab(t)}
            >
              {t === 'email' ? '邮件渠道' : '短信渠道'}
            </button>
          ))}
        </div>

        <DataList
          data={list as unknown as (NotificationChannel & Record<string, unknown>)[]}
          columns={columns}
          loading={loading}
          rowKey="id"
          emptyText="暂无渠道"
        />
      </div>
    </BaseLayout>
  )
}
