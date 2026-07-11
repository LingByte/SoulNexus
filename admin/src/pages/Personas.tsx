import { useEffect, useState, useCallback } from 'react'
import {
  Table,
  Card,
  Input,
  Button,
  Tag,
  Space,
  Modal,
  Form,
  Drawer,
  Switch,
  Popconfirm,
  Message,
  Descriptions,
  Avatar,
} from '@arco-design/web-react'
import { RefreshCw, Eye, Trash2, Edit3, UserCircle } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import {
  listAdminPersonas,
  getAdminPersona,
  updateAdminPersona,
  deleteAdminPersona,
  type AdminUserPersona,
} from '@/services/adminApi'

const { TextArea } = Input

const Personas = () => {
  const [loading, setLoading] = useState(false)
  const [rows, setRows] = useState<AdminUserPersona[]>([])
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [search, setSearch] = useState('')

  // 编辑抽屉
  const [editOpen, setEditOpen] = useState(false)
  const [editRow, setEditRow] = useState<AdminUserPersona | null>(null)
  const [editSubmitting, setEditSubmitting] = useState(false)
  const [editForm] = Form.useForm()

  // 详情抽屉
  const [detailOpen, setDetailOpen] = useState(false)
  const [detailRow, setDetailRow] = useState<AdminUserPersona | null>(null)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listAdminPersonas({ page, pageSize, search: search || undefined })
      setRows(res.items || [])
      setTotal(res.total || 0)
    } catch (e: any) {
      Message.error(`加载失败：${e?.msg || e?.message || e}`)
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, search])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  const handleSearch = () => {
    setPage(1)
    fetchData()
  }

  const handleEdit = async (row: AdminUserPersona) => {
    try {
      const detail = await getAdminPersona(row.id)
      setEditRow(detail)
      editForm.setFieldsValue({
        name: detail.name,
        description: detail.description || '',
        personality: detail.personality || '',
        scenario: detail.scenario || '',
        avatarUrl: detail.avatarUrl || '',
        isDefault: detail.isDefault,
      })
      setEditOpen(true)
    } catch (e: any) {
      Message.error(`获取详情失败：${e?.msg || e?.message || e}`)
    }
  }

  const handleEditSubmit = async () => {
    if (!editRow) return
    setEditSubmitting(true)
    try {
      const values = await editForm.validate()
      await updateAdminPersona(editRow.id, values)
      Message.success('更新成功')
      setEditOpen(false)
      fetchData()
    } catch (e: any) {
      if (e?.msg) {
        Message.error(`保存失败：${e.msg}`)
      }
    } finally {
      setEditSubmitting(false)
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await deleteAdminPersona(id)
      Message.success('删除成功')
      fetchData()
    } catch (e: any) {
      Message.error(`删除失败：${e?.msg || e?.message || e}`)
    }
  }

  const handleViewDetail = async (row: AdminUserPersona) => {
    try {
      const detail = await getAdminPersona(row.id)
      setDetailRow(detail)
      setDetailOpen(true)
    } catch (e: any) {
      Message.error(`获取详情失败：${e?.msg || e?.message || e}`)
    }
  }

  const columns = [
    {
      title: '头像',
      dataIndex: 'avatarUrl',
      width: 60,
      render: (v: string) => (
        <Avatar size={32}>
          {v ? <img src={v} alt="" /> : <UserCircle size={20} />}
        </Avatar>
      ),
    },
    {
      title: '名称',
      dataIndex: 'name',
      width: 160,
      ellipsis: true,
    },
    {
      title: '描述',
      dataIndex: 'description',
      ellipsis: true,
      render: (v: string) => (
        <span className="text-xs text-[var(--color-text-3)] line-clamp-1">{v || '-'}</span>
      ),
    },
    {
      title: '性格',
      dataIndex: 'personality',
      width: 160,
      ellipsis: true,
      render: (v: string) => (
        <span className="text-xs text-[var(--color-text-3)] line-clamp-1">{v || '-'}</span>
      ),
    },
    {
      title: '默认',
      dataIndex: 'isDefault',
      width: 60,
      render: (v: boolean) => (
        <Tag size="small" color={v ? 'green' : 'gray'}>{v ? '是' : '否'}</Tag>
      ),
    },
    {
      title: '用户',
      dataIndex: 'userId',
      width: 120,
      ellipsis: true,
      render: (v: string) => v || '-',
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      width: 160,
      render: (v: string) => v ? new Date(v).toLocaleString() : '-',
    },
    {
      title: '操作',
      width: 150,
      fixed: 'right' as const,
      render: (_: any, r: AdminUserPersona) => (
        <Space size="small">
          <Button
            type="text"
            size="small"
            icon={<Eye size={14} />}
            onClick={() => handleViewDetail(r)}
          >
            查看
          </Button>
          <Button
            type="text"
            size="small"
            icon={<Edit3 size={14} />}
            onClick={() => handleEdit(r)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定删除该 Persona？"
            onOk={() => handleDelete(r.id)}
          >
            <Button type="text" size="small" status="danger" icon={<Trash2 size={14} />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <PageHeader
        title="User Persona 管理"
        description="全局管理所有用户的 Persona 定义"
      />

      <Card style={{ marginBottom: 16 }}>
        <div className="flex flex-col sm:flex-row items-start sm:items-center gap-3">
          <Input.Search
            placeholder="搜索名称、描述或性格…"
            value={search}
            onChange={setSearch}
            onSearch={handleSearch}
            onClear={() => { setSearch(''); setPage(1) }}
            allowClear
            style={{ width: 320 }}
          />
          <Button icon={<RefreshCw size={14} />} onClick={() => { setPage(1); fetchData() }}>
            刷新
          </Button>
        </div>
      </Card>

      <Card>
        <Table
          rowKey="id"
          columns={columns}
          data={rows}
          loading={loading}
          scroll={{ x: 1000 }}
          pagination={{
            current: page,
            pageSize,
            total,
            showTotal: (t) => `共 ${t} 条`,
            onChange: (p, ps) => { setPage(p); setPageSize(ps) },
          }}
        />
      </Card>

      {/* 编辑抽屉 */}
      <Drawer
        width={560}
        title="编辑 User Persona"
        visible={editOpen}
        onCancel={() => setEditOpen(false)}
        footer={
          <Space>
            <Button onClick={() => setEditOpen(false)}>取消</Button>
            <Button type="primary" loading={editSubmitting} onClick={handleEditSubmit}>
              保存
            </Button>
          </Space>
        }
      >
        <Form form={editForm} layout="vertical" style={{ paddingRight: 16 }}>
          <Form.Item field="name" label="名称" rules={[{ required: true, message: '名称不能为空' }]}>
            <Input placeholder="Persona 名称" />
          </Form.Item>
          <Form.Item field="description" label="描述">
            <TextArea rows={3} placeholder="外貌、背景等" />
          </Form.Item>
          <Form.Item field="personality" label="性格">
            <TextArea rows={3} placeholder="性格特征、气质、说话方式" />
          </Form.Item>
          <Form.Item field="scenario" label="场景">
            <TextArea rows={3} placeholder="当前情境/上下文" />
          </Form.Item>
          <Form.Item field="avatarUrl" label="头像 URL">
            <Input placeholder="https://..." />
          </Form.Item>
          <Form.Item field="isDefault" label="默认 Persona" triggerPropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Drawer>

      {/* 详情抽屉 */}
      <Drawer
        width={500}
        title="Persona 详情"
        visible={detailOpen}
        onCancel={() => setDetailOpen(false)}
        footer={null}
      >
        {detailRow && (
          <Descriptions column={1} border size="small" style={{ marginBottom: 16 }}>
            <Descriptions.Item label="ID">{detailRow.id}</Descriptions.Item>
            <Descriptions.Item label="头像">
              <Avatar size={48}>
                {detailRow.avatarUrl ? <img src={detailRow.avatarUrl} alt="" /> : <UserCircle size={32} />}
              </Avatar>
            </Descriptions.Item>
            <Descriptions.Item label="名称">{detailRow.name}</Descriptions.Item>
            <Descriptions.Item label="用户">{detailRow.userId || '-'}</Descriptions.Item>
            <Descriptions.Item label="默认">
              <Tag color={detailRow.isDefault ? 'green' : 'gray'}>{detailRow.isDefault ? '是' : '否'}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="创建时间">{detailRow.createdAt ? new Date(detailRow.createdAt).toLocaleString() : '-'}</Descriptions.Item>
            <Descriptions.Item label="更新时间">{detailRow.updatedAt ? new Date(detailRow.updatedAt).toLocaleString() : '-'}</Descriptions.Item>
            <Descriptions.Item label="描述" span={1}>
              <div className="whitespace-pre-wrap text-xs max-h-40 overflow-y-auto">{detailRow.description || '-'}</div>
            </Descriptions.Item>
            <Descriptions.Item label="性格" span={1}>
              <div className="whitespace-pre-wrap text-xs max-h-40 overflow-y-auto">{detailRow.personality || '-'}</div>
            </Descriptions.Item>
            <Descriptions.Item label="场景" span={1}>
              <div className="whitespace-pre-wrap text-xs max-h-40 overflow-y-auto">{detailRow.scenario || '-'}</div>
            </Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>
    </div>
  )
}

export default Personas
