// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 通知渠道列表页：仅负责列表 / 删除；新建与编辑跳转至独立编辑页。
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, Edit, Trash2, RefreshCw } from 'lucide-react'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Badge from '@/components/UI/Badge'
import EmptyState from '@/components/UI/EmptyState'
import ConfirmDialog from '@/components/UI/ConfirmDialog'
import { showAlert } from '@/utils/notification'
import {
  listNotificationChannels,
  deleteNotificationChannel,
  type NotificationChannel,
} from '@/services/notificationsApi'

type ChannelType = 'email' | 'sms'

const NotificationChannels = () => {
  const navigate = useNavigate()
  const [tab, setTab] = useState<ChannelType>('email')
  const [list, setList] = useState<NotificationChannel[]>([])
  const [loading, setLoading] = useState(false)
  const [showDelete, setShowDelete] = useState(false)
  const [deleting, setDeleting] = useState<NotificationChannel | null>(null)

  const fetchList = async () => {
    setLoading(true)
    try {
      const res = await listNotificationChannels({ type: tab, page: 1, pageSize: 200 })
      setList(res.list || [])
    } catch (e: any) {
      showAlert('加载渠道失败', 'error', e?.msg || e?.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchList()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab])

  const submitDelete = async () => {
    if (!deleting) return
    try {
      await deleteNotificationChannel(deleting.id)
      showAlert('删除成功', 'success')
      setShowDelete(false)
      setDeleting(null)
      fetchList()
    } catch (e: any) {
      showAlert('删除失败', 'error', e?.msg || e?.message)
    }
  }

  return (
    <div className="p-4 md:p-6 space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">通知渠道供应商</h1>
          <p className="text-sm text-muted-foreground">
            管理邮件 / 短信发送供应商，启用后系统消息会按权重负载与故障切换发送。
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={fetchList}>
            <RefreshCw className="w-4 h-4 mr-1" />刷新
          </Button>
          <Button onClick={() => navigate(`/notification-channels/new?type=${tab}`)}>
            <Plus className="w-4 h-4 mr-1" />新建{tab === 'email' ? '邮件' : '短信'}渠道
          </Button>
        </div>
      </div>

      <div className="flex gap-2 border-b">
        <button
          className={`px-4 py-2 -mb-px border-b-2 ${tab === 'email' ? 'border-primary text-primary' : 'border-transparent'}`}
          onClick={() => setTab('email')}
        >
          邮件渠道
        </button>
        <button
          className={`px-4 py-2 -mb-px border-b-2 ${tab === 'sms' ? 'border-primary text-primary' : 'border-transparent'}`}
          onClick={() => setTab('sms')}
        >
          短信渠道
        </button>
      </div>

      <Card>
        {loading ? (
          <div className="p-8 text-center text-muted-foreground">加载中...</div>
        ) : list.length === 0 ? (
          <EmptyState title="暂无渠道" description="点击右上角新建一个渠道" />
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full text-sm">
              <thead className="bg-muted/50">
                <tr>
                  <th className="text-left px-4 py-2">名称</th>
                  <th className="text-left px-4 py-2">类型</th>
                  <th className="text-left px-4 py-2">排序</th>
                  <th className="text-left px-4 py-2">状态</th>
                  <th className="text-left px-4 py-2">备注</th>
                  <th className="text-right px-4 py-2">操作</th>
                </tr>
              </thead>
              <tbody>
                {list.map((c) => (
                  <tr key={c.id} className="border-t">
                    <td className="px-4 py-2">{c.name}</td>
                    <td className="px-4 py-2"><Badge variant="outline">{c.type}</Badge></td>
                    <td className="px-4 py-2">{c.sortOrder}</td>
                    <td className="px-4 py-2">
                      {c.enabled ? <Badge>启用</Badge> : <Badge variant="outline">禁用</Badge>}
                    </td>
                    <td className="px-4 py-2 text-muted-foreground">{c.remark}</td>
                    <td className="px-4 py-2 text-right space-x-1">
                      <Button size="sm" variant="ghost" onClick={() => navigate(`/notification-channels/${c.id}/edit`)}>
                        <Edit className="w-4 h-4" />
                      </Button>
                      <Button size="sm" variant="ghost" onClick={() => { setDeleting(c); setShowDelete(true) }}>
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      <ConfirmDialog
        isOpen={showDelete}
        title="删除渠道"
        message={`确定要删除渠道 “${deleting?.name}” 吗？`}
        onClose={() => setShowDelete(false)}
        onConfirm={submitDelete}
        variant="danger"
      />
    </div>
  )
}

export default NotificationChannels
