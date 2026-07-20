import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, Pencil, Trash2, RefreshCw } from 'lucide-react'
import { Modal as ArcoModal, Tag } from '@arco-design/web-react'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Button, DataList } from '@/components/ui'
import type { DataListColumn } from '@/components/ui'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'
import { listMailTemplates, deleteMailTemplate, type MailTemplate } from '@/api/notifications'

export default function MailTemplatesPage() {
  const navigate = useNavigate()
  const [list, setList] = useState<MailTemplate[]>([])
  const [loading, setLoading] = useState(false)

  const fetchList = async () => {
    setLoading(true)
    try {
      const res = await listMailTemplates({ page: 1, pageSize: 200 })
      setList(res.list || [])
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, '加载模板失败'), 'error')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void fetchList()
  }, [])

  const confirmDelete = (row: MailTemplate) => {
    ArcoModal.confirm({
      title: '删除模板',
      content: `确定要删除模板「${row.name}」吗？`,
      onOk: async () => {
        await deleteMailTemplate(row.id)
        showAlert('删除成功', 'success')
        await fetchList()
      },
    })
  }

  const columns: DataListColumn<Record<string, unknown>>[] = [
    {
      key: 'code',
      title: 'code',
      width: 180,
      render: (_, r) => (
        <span className="font-mono text-xs text-neutral-700">{String(r.code || '—')}</span>
      ),
    },
    {
      key: 'info',
      title: '名称',
      render: (_, r) => (
        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-medium text-neutral-900">{String(r.name || '—')}</div>
          {r.description ? (
            <div className="mt-0.5 truncate text-xs text-neutral-400">{String(r.description)}</div>
          ) : null}
        </div>
      ),
    },
    {
      key: 'locale',
      title: '语言',
      width: 80,
      render: (_, r) => <span className="text-xs text-neutral-500">{String(r.locale || '—')}</span>,
    },
    {
      key: 'description',
      title: '描述',
      render: (_, r) => (
        <span className="truncate text-sm text-neutral-500">{String(r.description || '—')}</span>
      ),
    },
    {
      key: 'status',
      title: '状态',
      width: 100,
      render: (_, r) => (
        <Tag color={r.enabled ? 'green' : 'gray'} className="!rounded-full">
          {r.enabled ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      key: 'actions',
      width: 80,
     
      align: 'right',
      render: (_, r) => (
        <div className="flex items-center justify-end gap-1">
          <Button size="mini" icon={<Pencil size={12} />} onClick={() => navigate(`/platform/mail-templates/${String(r.id)}/edit`)} />
          <Button size="mini" status="danger" icon={<Trash2 size={12} />} onClick={() => confirmDelete(r as unknown as MailTemplate)} />
        </div>
      ),
    },
  ]

  return (
    <BaseLayout title="邮件模板" description="业务方按 code 引用模板，运行期渲染占位符。">
      <DataList
        data={list as unknown as (MailTemplate & Record<string, unknown>)[]}
        columns={columns}
        loading={loading}
        rowKey="id"
        emptyText="暂无模板"
        header={
          <div className="flex items-center justify-end gap-2">
            <Button type="outline" icon={<RefreshCw size={14} />} onClick={() => void fetchList()}>刷新</Button>
            <Button type="primary" icon={<Plus size={14} />} onClick={() => navigate('/platform/mail-templates/new')}>新建模板</Button>
          </div>
        }
      />
    </BaseLayout>
  )
}
