// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 权限管理：Arco Table + Form Modal。
import { useCallback, useEffect, useState } from 'react'
import {
  Table,
  Input,
  Button,
  Modal,
  Form,
  Switch,
  Tag,
  Popconfirm,
  Message,
  Space,
  Drawer,
} from '@arco-design/web-react'
import { Search, RefreshCw, Plus, Edit, Trash2, Eye } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import {
  listPermissions,
  createPermission,
  updatePermission,
  deletePermission,
  type PermissionListItem,
} from '@/services/permissionApi'

const FormItem = Form.Item

const Permissions = () => {
  const [rows, setRows] = useState<PermissionListItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [search, setSearch] = useState('')
  const [searchInput, setSearchInput] = useState('')
  const [withRoles, setWithRoles] = useState(false)
  const [loading, setLoading] = useState(false)

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<PermissionListItem | null>(null)
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm()

  const [viewRoles, setViewRoles] = useState<PermissionListItem | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const data = await listPermissions({
        page,
        pageSize,
        search: search.trim() || undefined,
        withRoles,
      })
      setRows(data.items || [])
      setTotal(data.total || 0)
    } catch (e: any) {
      Message.error(`加载失败：${e?.message || e}`)
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, search, withRoles])

  useEffect(() => {
    void load()
  }, [load])

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    setModalOpen(true)
  }

  const openEdit = (row: PermissionListItem) => {
    setEditing(row)
    form.setFieldsValue({
      key: row.key,
      name: row.name,
      description: row.description || '',
      resource: row.resource || '',
    })
    setModalOpen(true)
  }

  const save = async () => {
    try {
      const values = await form.validate()
      setSaving(true)
      if (editing) {
        await updatePermission(editing.id, values)
        Message.success('已更新')
      } else {
        await createPermission(values)
        Message.success('已创建')
      }
      setModalOpen(false)
      void load()
    } catch (e: any) {
      if (e?.message || e?.msg) {
        Message.error(`保存失败：${e?.msg || e?.message}`)
      }
    } finally {
      setSaving(false)
    }
  }

  const remove = async (row: PermissionListItem) => {
    try {
      await deletePermission(row.id)
      Message.success('已删除')
      void load()
    } catch (e: any) {
      Message.error(`删除失败：${e?.msg || e?.message || e}`)
    }
  }

  const columns = [
    {
      title: 'Key',
      dataIndex: 'key',
      width: 220,
      ellipsis: true,
      render: (v: string) => <span style={{ fontFamily: 'ui-monospace, monospace', fontSize: 12 }}>{v}</span>,
    },
    { title: '名称', dataIndex: 'name', ellipsis: true },
    { title: '资源', dataIndex: 'resource', width: 140, render: (v: string) => v || '—' },
    { title: '关联角色数', dataIndex: 'roleCount', width: 110, render: (v: number) => v ?? 0 },
    { title: '直连用户授权', dataIndex: 'directUserGrantCount', width: 130, render: (v: number) => v ?? 0 },
    {
      title: '操作',
      key: '__actions__',
      width: 240,
      fixed: 'right' as const,
      render: (_: any, row: PermissionListItem) => (
        <Space size="mini">
          {withRoles && row.roles && row.roles.length > 0 && (
            <Button size="mini" type="text" onClick={() => setViewRoles(row)}>
              <span className="inline-flex items-center gap-1"><Eye size={14} /> 角色</span>
            </Button>
          )}
          <Button size="mini" type="text" onClick={() => openEdit(row)}>
            <span className="inline-flex items-center gap-1"><Edit size={14} /> 编辑</span>
          </Button>
          <Popconfirm title={`删除权限「${row.key}」？`} okText="删除" cancelText="取消" onOk={() => remove(row)}>
            <Button size="mini" type="text" status="danger">
              <span className="inline-flex items-center gap-1"><Trash2 size={14} /> 删除</span>
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <PageHeader
        title="权限管理"
        description="维护权限点；列表展示关联角色数与直连用户数，可选加载关联角色。"
        actions={
          <Space>
            <Input
              prefix={<Search size={14} />}
              placeholder="搜索 key / 名称"
              value={searchInput}
              onChange={(v) => setSearchInput(v)}
              onPressEnter={() => { setPage(1); setSearch(searchInput) }}
              allowClear
              onClear={() => { setPage(1); setSearch('') }}
              style={{ width: 220 }}
            />
            <Switch checked={withRoles} onChange={(v) => setWithRoles(v)} /> <span className="text-sm text-[var(--color-text-2)]">加载角色</span>
            <Button type="primary" onClick={() => { setPage(1); setSearch(searchInput) }}>搜索</Button>
            <Button onClick={() => void load()}>
              <span className="inline-flex items-center gap-1"><RefreshCw size={14} className={loading ? 'animate-spin' : ''} /> 刷新</span>
            </Button>
            <Button type="primary" onClick={openCreate}>
              <span className="inline-flex items-center gap-1"><Plus size={14} /> 新建权限</span>
            </Button>
          </Space>
        }
      />

      <Table
        rowKey={(r: PermissionListItem) => String(r.id)}
        loading={loading}
        columns={columns}
        data={rows}
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

      <Modal
        title={editing ? '编辑权限' : '新建权限'}
        visible={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={save}
        confirmLoading={saving}
        okText="保存"
        cancelText="取消"
        autoFocus={false}
      >
        <Form form={form} layout="vertical" autoComplete="off">
          <FormItem
            label="Key"
            field="key"
            rules={[{ required: true, message: '请填写权限 key' }]}
          >
            <Input placeholder="例如：user.read" disabled={!!editing} />
          </FormItem>
          <FormItem label="名称" field="name" rules={[{ required: true, message: '请填写名称' }]}>
            <Input placeholder="展示名称" />
          </FormItem>
          <FormItem label="描述" field="description">
            <Input placeholder="可选" />
          </FormItem>
          <FormItem label="资源分组" field="resource">
            <Input placeholder="可选，例如 user / billing" />
          </FormItem>
        </Form>
      </Modal>

      <Drawer
        title={viewRoles ? `关联角色 — ${viewRoles.key}` : ''}
        visible={!!viewRoles}
        onCancel={() => setViewRoles(null)}
        footer={null}
        width={420}
        autoFocus={false}
      >
        <div className="space-y-2">
          {(viewRoles?.roles || []).map((r) => (
            <div key={r.id} className="flex items-center gap-2">
              <Tag color="arcoblue">{r.slug}</Tag>
              <span className="text-sm">{r.name}</span>
            </div>
          ))}
          {!viewRoles?.roles?.length && <div className="text-[var(--color-text-3)] text-sm">暂无</div>}
        </div>
      </Drawer>
    </div>
  )
}

export default Permissions
