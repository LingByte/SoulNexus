import { useCallback, useEffect, useState } from 'react'
import { Form, Modal, Popconfirm, Space, Table, Tag, Typography } from '@arco-design/web-react'
import { IconPlus, IconDelete, IconEdit, IconBook, IconFile, IconList, IconApps } from '@arco-design/web-react/icon'
import { useNavigate } from 'react-router-dom'
import BaseLayout from '@/components/Layout/BaseLayout'
import { useTranslation } from '@/i18n'
import {
  listKnowledgeNamespaces,
  createKnowledgeNamespace,
  updateKnowledgeNamespace,
  deleteKnowledgeNamespace,
  type KnowledgeNamespace,
  type CreateNamespaceReq,
} from '@/api/knowledgeNamespaces'
import { showAlert } from '@/utils/notification'
import { Button, Input, Card, Empty, TableEmpty } from '@/components/ui'
import { Loading } from '@/components/ui/loading'
import { cn } from '@/utils/cn'

const FormItem = Form.Item

type ViewMode = 'card' | 'list'

const STATUS_MAP: Record<string, { color: string; label: string }> = {
  active: { color: 'green', label: 'Active' },
  inactive: { color: 'gray', label: 'Inactive' },
}

export default function KnowledgeNamespaces() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [data, setData] = useState<KnowledgeNamespace[]>([])
  const [visible, setVisible] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [viewMode, setViewMode] = useState<ViewMode>('card')
  const [form] = Form.useForm()

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listKnowledgeNamespaces()
      if (res.code === 200 && res.data) {
        setData(res.data)
      }
    } catch { /* ignore */ }
    setLoading(false)
  }, [])

  useEffect(() => { load() }, [load])

  const handleCreate = () => {
    setEditingId(null)
    form.resetFields()
    setVisible(true)
  }

  const handleEdit = (record: KnowledgeNamespace) => {
    setEditingId(record.id)
    form.setFieldsValue({
      name: record.name,
      description: record.description,
    })
    setVisible(true)
  }

  const handleSubmit = async () => {
    const values = await form.validate()
    try {
      if (editingId) {
        await updateKnowledgeNamespace(editingId, {
          name: values.name,
          description: values.description,
        })
        showAlert(t('common.success'), 'success')
      } else {
        const payload: CreateNamespaceReq = {
          name: values.name,
          description: values.description,
        }
        await createKnowledgeNamespace(payload)
        showAlert(t('common.success'), 'success')
      }
      setVisible(false)
      load()
    } catch (err: any) {
      showAlert(err?.message || t('common.failed'), 'error')
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await deleteKnowledgeNamespace(id)
      showAlert(t('common.success'), 'success')
      load()
    } catch (err: any) {
      showAlert(err?.message || t('common.failed'), 'error')
    }
  }

  const openDocuments = (record: KnowledgeNamespace) => {
    navigate(`/knowledge-base/${record.id}`)
  }

  const NamespaceCard = ({ record }: { record: KnowledgeNamespace }) => {
    const status = STATUS_MAP[record.status] || { color: 'gray', label: record.status }
    return (
      <Card variant="elevated" icon={<IconBook style={{ fontSize: 28 }} />} className="h-full">
        <div className="flex w-full flex-col gap-3">
          <div className="flex flex-wrap gap-2">
            <span className="inline-flex rounded-md bg-white/20 px-2 py-0.5 text-xs text-white">{status.label}</span>
            <span className="inline-flex rounded-md bg-white/15 px-2 py-0.5 text-xs text-white">
              {record.vectorProvider?.toUpperCase() || '—'}
            </span>
          </div>
          <h3 className="mb-0 truncate text-base font-semibold text-white">{record.name}</h3>
          {record.description ? (
            <p className="mb-0 line-clamp-2 text-xs text-white/85">{record.description}</p>
          ) : (
            <p className="mb-0 text-xs italic text-white/60">{t('knowledgeBase.noDescription')}</p>
          )}
          <p className="mb-0 truncate font-mono text-[11px] text-white/75">{record.namespace}</p>
          <p className="mb-0 text-xs text-white/80">
            {record.embedModel} · {record.vectorDim}d
          </p>
          <div className="mt-1 flex flex-wrap gap-2 border-t border-white/20 pt-3">
            <Button
              size="small"
              type="text"
              className="!text-white hover:!bg-white/10"
              icon={<IconFile />}
              onClick={() => openDocuments(record)}
            >
              {t('knowledgeBase.openDocuments')}
            </Button>
            <Button
              size="small"
              type="text"
              className="!text-white hover:!bg-white/10"
              icon={<IconEdit />}
              onClick={() => handleEdit(record)}
            />
            <Popconfirm title={t('common.confirm')} onOk={() => handleDelete(record.id)}>
              <Button
                size="small"
                type="text"
                status="danger"
                className="!text-white/90 hover:!bg-white/10"
                icon={<IconDelete />}
              />
            </Popconfirm>
          </div>
        </div>
      </Card>
    )
  }

  const columns = [
    {
      title: t('knowledgeBase.colName'),
      dataIndex: 'name',
      ellipsis: true,
      width: 280,
      render: (text: string) => (
        <Space>
          <IconBook style={{ color: 'var(--color-primary-6)' }} />
          <Typography.Text bold>{text}</Typography.Text>
        </Space>
      ),
    },
    {
      title: t('knowledgeBase.colNamespace'),
      dataIndex: 'namespace',
      ellipsis: true,
      width: 280,
      render: (text: string) => (
        <Typography.Text code copyable>{text}</Typography.Text>
      ),
    },
    {
      title: t('knowledgeBase.colProvider'),
      dataIndex: 'vectorProvider',
      render: (text: string) => <Tag>{text?.toUpperCase()}</Tag>,
    },
    {
      title: t('knowledgeBase.colModel'),
      dataIndex: 'embedModel',
      ellipsis: true,
    },
    {
      title: t('knowledgeBase.colDim'),
      dataIndex: 'vectorDim',
    },
    {
      title: t('knowledgeBase.colStatus'),
      dataIndex: 'status',
      render: (text: string) => {
        const s = STATUS_MAP[text] || { color: 'gray', label: text }
        return <Tag color={s.color}>{s.label}</Tag>
      },
    },
    {
      title: t('knowledgeBase.colActions'),
      dataIndex: 'actions',
      width: 160,
      render: (_: unknown, record: KnowledgeNamespace) => (
        <Space>
          <Button icon={<IconFile />} onClick={() => openDocuments(record)} title={t('knowledgeBase.openDocuments')} />
          <Button icon={<IconEdit />} onClick={() => handleEdit(record)} />
          <Popconfirm
            title={t('common.confirm')}
            onOk={() => handleDelete(record.id)}
          >
            <Button status="danger" icon={<IconDelete />} />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <BaseLayout title={t('knowledgeBase.title')} description={t('knowledgeBase.description')}>
      <Card bordered={false} style={{ borderRadius: 12 }}>
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
          <Typography.Text style={{ color: 'var(--color-text-3)' }}>
            {t('knowledgeBase.total')}: {data.length}
          </Typography.Text>
          <div className="flex flex-wrap items-center gap-3">
            <div className="flex items-center rounded-md border border-border bg-muted/40 p-0.5">
              <button
                type="button"
                onClick={() => setViewMode('card')}
                className={cn(
                  'rounded px-2 py-1.5 transition-colors',
                  viewMode === 'card' ? 'bg-card shadow-sm text-foreground' : 'text-muted-foreground hover:text-foreground',
                )}
                title="卡片视图"
              >
                <IconApps style={{ fontSize: 16 }} />
              </button>
              <button
                type="button"
                onClick={() => setViewMode('list')}
                className={cn(
                  'rounded px-2 py-1.5 transition-colors',
                  viewMode === 'list' ? 'bg-card shadow-sm text-foreground' : 'text-muted-foreground hover:text-foreground',
                )}
                title="列表视图"
              >
                <IconList style={{ fontSize: 16 }} />
              </button>
            </div>
            <Button icon={<IconPlus />} onClick={handleCreate}>
              {t('knowledgeBase.create')}
            </Button>
          </div>
        </div>

        {loading ? (
          <Loading block tip={t('common.loading')} />
        ) : data.length === 0 ? (
          <Empty preset="no-data" description={t('knowledgeBase.emptyNamespaces')}>
            <Button type="primary" icon={<IconPlus />} onClick={handleCreate}>
              {t('knowledgeBase.create')}
            </Button>
          </Empty>
        ) : viewMode === 'card' ? (
          <div className="grid grid-cols-1 gap-4 overflow-visible sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {data.map((record) => (
              <NamespaceCard key={record.id} record={record} />
            ))}
          </div>
        ) : (
          <div className="overflow-x-auto">
            <Table
              columns={columns}
              data={data}
              rowKey="id"
              pagination={false}
              scroll={{ x: 1200 }}
              noDataElement={<TableEmpty description={t('common.noData')} />}
            />
          </div>
        )}
      </Card>

      <Modal
        title={editingId ? t('knowledgeBase.edit') : t('knowledgeBase.create')}
        visible={visible}
        onOk={handleSubmit}
        onCancel={() => setVisible(false)}
        unmountOnExit
        style={{ width: 520 }}
      >
        <Form form={form} layout="vertical">
          <FormItem label={t('knowledgeBase.form.name')} field="name" rules={[{ required: true }]}>
            <Input placeholder={t('knowledgeBase.form.namePlaceholder')} />
          </FormItem>
          <FormItem label={t('knowledgeBase.form.description')} field="description">
            <Input.TextArea placeholder={t('knowledgeBase.form.descriptionPlaceholder')} />
          </FormItem>
        </Form>
      </Modal>
    </BaseLayout>
  )
}
