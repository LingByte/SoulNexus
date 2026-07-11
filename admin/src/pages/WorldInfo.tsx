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
  InputNumber,
  Popconfirm,
  Message,
  Descriptions,
} from '@arco-design/web-react'
import { RefreshCw, Eye, Trash2, Edit3, Globe } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import {
  listAdminWorldInfo,
  getAdminWorldInfo,
  updateAdminWorldInfo,
  deleteAdminWorldInfo,
  type AdminWorldInfoEntry,
} from '@/services/adminApi'

const { TextArea } = Input

const Assistants = () => {
  const [loading, setLoading] = useState(false)
  const [rows, setRows] = useState<AdminWorldInfoEntry[]>([])
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [search, setSearch] = useState('')

  // 编辑抽屉
  const [editOpen, setEditOpen] = useState(false)
  const [editRow, setEditRow] = useState<AdminWorldInfoEntry | null>(null)
  const [editSubmitting, setEditSubmitting] = useState(false)
  const [editForm] = Form.useForm()

  // 详情抽屉
  const [detailOpen, setDetailOpen] = useState(false)
  const [detailRow, setDetailRow] = useState<AdminWorldInfoEntry | null>(null)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listAdminWorldInfo({ page, pageSize, search: search || undefined })
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

  // 打开编辑
  const handleEdit = async (row: AdminWorldInfoEntry) => {
    try {
      const detail = await getAdminWorldInfo(row.id)
      setEditRow(detail)
      editForm.setFieldsValue({
        name: detail.name,
        content: detail.content,
        keys: detail.keys,
        regex: detail.regex || '',
        selective: detail.selective,
        constant: detail.constant,
        depth: detail.depth,
        order: detail.order,
        enabled: detail.enabled,
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
      await updateAdminWorldInfo(editRow.id, values)
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
      await deleteAdminWorldInfo(id)
      Message.success('删除成功')
      fetchData()
    } catch (e: any) {
      Message.error(`删除失败：${e?.msg || e?.message || e}`)
    }
  }

  const handleViewDetail = async (row: AdminWorldInfoEntry) => {
    try {
      const detail = await getAdminWorldInfo(row.id)
      setDetailRow(detail)
      setDetailOpen(true)
    } catch (e: any) {
      Message.error(`获取详情失败：${e?.msg || e?.message || e}`)
    }
  }

  const columns = [
    {
      title: '名称',
      dataIndex: 'name',
      width: 180,
      ellipsis: true,
    },
    {
      title: '内容',
      dataIndex: 'content',
      ellipsis: true,
      render: (v: string) => (
        <span className="text-xs text-[var(--color-text-3)] line-clamp-1">{v || '-'}</span>
      ),
    },
    {
      title: '触发词',
      dataIndex: 'keys',
      width: 150,
      ellipsis: true,
      render: (v: string) => {
        if (!v) return <span className="text-xs text-[var(--color-text-4)]">-</span>
        return (
          <Space size={4} wrap>
            {v.split(',').filter(Boolean).slice(0, 3).map((k: string) => (
              <Tag key={k} size="small" color="arcoblue">{k.trim()}</Tag>
            ))}
            {v.split(',').filter(Boolean).length > 3 && (
              <Tag size="small" color="gray">+{v.split(',').filter(Boolean).length - 3}</Tag>
            )}
          </Space>
        )
      },
    },
    {
      title: '激活模式',
      width: 130,
      render: (_: any, r: AdminWorldInfoEntry) => (
        <Space size={4} wrap>
          {r.constant && <Tag size="small" color="green">常驻</Tag>}
          {r.selective && <Tag size="small" color="purple">链式</Tag>}
          {!r.constant && !r.selective && r.keys && <Tag size="small" color="arcoblue">关键词</Tag>}
          {r.regex && <Tag size="small" color="orange">正则</Tag>}
          {!r.constant && !r.selective && !r.keys && !r.regex && (
            <span className="text-xs text-[var(--color-text-4)]">-</span>
          )}
        </Space>
      ),
    },
    {
      title: '深度',
      dataIndex: 'depth',
      width: 60,
    },
    {
      title: '排序',
      dataIndex: 'order',
      width: 60,
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 70,
      render: (v: boolean) => (
        <Tag size="small" color={v ? 'green' : 'gray'}>{v ? '启用' : '禁用'}</Tag>
      ),
    },
    {
      title: 'Agent ID',
      dataIndex: 'agentId',
      width: 80,
      render: (v: number) => (v ? v : <span className="text-xs text-[var(--color-text-4)]">全局</span>),
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
      render: (_: any, r: AdminWorldInfoEntry) => (
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
            title="确定删除该条目？"
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
        title="World Info 管理"
        description="全局管理所有用户的 Lorebook / World Info 条目"
      />

      <Card style={{ marginBottom: 16 }}>
        <div className="flex flex-col sm:flex-row items-start sm:items-center gap-3">
          <Input.Search
            placeholder="搜索名称、内容或触发词…"
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
          scroll={{ x: 1300 }}
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
        title="编辑 World Info 条目"
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
            <Input placeholder="条目名称" />
          </Form.Item>
          <Form.Item field="content" label="内容" rules={[{ required: true, message: '内容不能为空' }]}>
            <TextArea rows={8} placeholder="World Info 内容，将在激活时注入 prompt" />
          </Form.Item>
          <Form.Item field="keys" label="触发关键词">
            <Input placeholder="逗号分隔，如：森林,夜晚,魔法" />
          </Form.Item>
          <Form.Item field="regex" label="正则表达式">
            <Input placeholder="可选的正则匹配" />
          </Form.Item>
          <Form.Item field="selective" label="链式激活" triggerPropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item field="constant" label="常驻激活" triggerPropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item field="enabled" label="启用" triggerPropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item field="depth" label="递归深度">
            <InputNumber min={1} max={10} />
          </Form.Item>
          <Form.Item field="order" label="排序优先级">
            <InputNumber min={0} max={1000} />
          </Form.Item>
        </Form>
      </Drawer>

      {/* 详情抽屉 */}
      <Drawer
        width={500}
        title="World Info 详情"
        visible={detailOpen}
        onCancel={() => setDetailOpen(false)}
        footer={null}
      >
        {detailRow && (
          <Descriptions column={1} border size="small" style={{ marginBottom: 16 }}>
            <Descriptions.Item label="ID">{detailRow.id}</Descriptions.Item>
            <Descriptions.Item label="名称">{detailRow.name}</Descriptions.Item>
            <Descriptions.Item label="Agent ID">{detailRow.agentId || '全局'}</Descriptions.Item>
            <Descriptions.Item label="用户">{detailRow.userId || '-'}</Descriptions.Item>
            <Descriptions.Item label="组织">{detailRow.groupId || '-'}</Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={detailRow.enabled ? 'green' : 'gray'}>{detailRow.enabled ? '启用' : '禁用'}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="常驻">
              <Tag color={detailRow.constant ? 'green' : 'gray'}>{detailRow.constant ? '是' : '否'}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="链式">
              <Tag color={detailRow.selective ? 'purple' : 'gray'}>{detailRow.selective ? '是' : '否'}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="递归深度">{detailRow.depth}</Descriptions.Item>
            <Descriptions.Item label="排序">{detailRow.order}</Descriptions.Item>
            <Descriptions.Item label="关键词">{detailRow.keys || '-'}</Descriptions.Item>
            <Descriptions.Item label="正则">{detailRow.regex || '-'}</Descriptions.Item>
            <Descriptions.Item label="创建时间">{detailRow.createdAt ? new Date(detailRow.createdAt).toLocaleString() : '-'}</Descriptions.Item>
            <Descriptions.Item label="更新时间">{detailRow.updatedAt ? new Date(detailRow.updatedAt).toLocaleString() : '-'}</Descriptions.Item>
            <Descriptions.Item label="内容" span={1}>
              <div className="whitespace-pre-wrap text-xs max-h-60 overflow-y-auto">{detailRow.content}</div>
            </Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>
    </div>
  )
}

export default Assistants
