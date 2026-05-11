// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 角色管理：Arco Table（含权限标签列）+ Form Modal + 权限分配 Drawer。
import { useCallback, useEffect, useState } from 'react'
import {
  Table,
  Input,
  Button,
  Modal,
  Form,
  Tag,
  Drawer,
  Checkbox,
  Popconfirm,
  Message,
  Space,
} from '@arco-design/web-react'
import { Search, RefreshCw, Plus, Edit, Trash2, KeyRound } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import {
  listRoles,
  createRole,
  updateRole,
  deleteRole,
  setRolePermissions,
  type Role,
} from '@/services/roleApi'
import { listPermissions, type PermissionListItem } from '@/services/permissionApi'

const FormItem = Form.Item

const Roles = () => {
  const [rows, setRows] = useState<Role[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(false)

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Role | null>(null)
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm()

  const [permModalRole, setPermModalRole] = useState<Role | null>(null)
  const [allPerms, setAllPerms] = useState<PermissionListItem[]>([])
  const [permPick, setPermPick] = useState<number[]>([])
  const [permSaving, setPermSaving] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const data = await listRoles({ page, pageSize, search: search.trim() || undefined })
      setRows(data.items || [])
      setTotal(data.total || 0)
    } catch (e: any) {
      Message.error(`加载失败：${e?.msg || e?.message || e}`)
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, search])

  useEffect(() => { void load() }, [load])

  const loadAllPerms = async () => {
    const data = await listPermissions({ page: 1, pageSize: 500 })
    setAllPerms(data.items || [])
  }

  const openPermEditor = async (r: Role) => {
    setPermModalRole(r)
    setPermPick((r.permissions || []).map((x) => x.id))
    if (allPerms.length === 0) await loadAllPerms()
  }

  const savePermPick = async () => {
    if (!permModalRole) return
    setPermSaving(true)
    try {
      const updated = await setRolePermissions(permModalRole.id, permPick)
      Message.success('角色权限已保存')
      setPermModalRole(null)
      setRows((prev) => prev.map((x) => (x.id === updated.id ? updated : x)))
    } catch (e: any) {
      Message.error(`保存失败：${e?.msg || e?.message || e}`)
    } finally {
      setPermSaving(false)
    }
  }

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    setModalOpen(true)
  }

  const openEdit = (r: Role) => {
    setEditing(r)
    form.setFieldsValue({ name: r.name, slug: r.slug, description: r.description || '' })
    setModalOpen(true)
  }

  const save = async () => {
    try {
      const values = await form.validate()
      setSaving(true)
      if (editing) {
        await updateRole(editing.id, values)
        Message.success('已更新')
      } else {
        await createRole(values)
        Message.success('已创建')
      }
      setModalOpen(false)
      void load()
    } catch (e: any) {
      if (e?.message || e?.msg) Message.error(`保存失败：${e?.msg || e?.message}`)
    } finally {
      setSaving(false)
    }
  }

  const remove = async (r: Role) => {
    if (r.isSystem) {
      Message.error('系统角色不可删除')
      return
    }
    try {
      await deleteRole(r.id)
      Message.success('已删除')
      void load()
    } catch (e: any) {
      Message.error(`删除失败：${e?.msg || e?.message || e}`)
    }
  }

  const columns = [
    {
      title: 'Slug',
      dataIndex: 'slug',
      width: 180,
      ellipsis: true,
      render: (v: string, r: Role) => (
        <span className="inline-flex items-center gap-1">
          <span style={{ fontFamily: 'ui-monospace, monospace', fontSize: 12 }} className="text-[var(--color-primary-light-4)]">{v}</span>
          {r.isSystem && <Tag color="orange">系统</Tag>}
        </span>
      ),
    },
    { title: '名称', dataIndex: 'name', width: 160, ellipsis: true },
    { title: '描述', dataIndex: 'description', ellipsis: true, render: (v: string) => v || '-' },
    {
      title: '权限',
      dataIndex: 'permissions',
      render: (_: any, r: Role) => {
        const perms = r.permissions || []
        if (perms.length === 0) return <span className="text-[var(--color-text-3)] text-xs">暂无</span>
        const shown = perms.slice(0, 6)
        const rest = perms.length - shown.length
        return (
          <div className="flex flex-wrap gap-1">
            {shown.map((p) => (
              <Tag key={p.id} size="small">{p.key}</Tag>
            ))}
            {rest > 0 && <Tag size="small" color="gray">+{rest}</Tag>}
          </div>
        )
      },
    },
    {
      title: '操作',
      key: '__actions__',
      width: 240,
      fixed: 'right' as const,
      render: (_: any, r: Role) => (
        <Space size="mini">
          <Button size="mini" type="text" onClick={() => void openPermEditor(r)}>
            <span className="inline-flex items-center gap-1"><KeyRound size={14} /> 权限</span>
          </Button>
          <Button size="mini" type="text" onClick={() => openEdit(r)}>
            <span className="inline-flex items-center gap-1"><Edit size={14} /> 编辑</span>
          </Button>
          <Popconfirm
            title={`删除角色「${r.slug}」？`}
            okText="删除"
            cancelText="取消"
            disabled={r.isSystem}
            onOk={() => remove(r)}
          >
            <Button size="mini" type="text" status="danger" disabled={r.isSystem}>
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
        title="角色管理"
        description="角色与权限多对多关联；列表已包含每个角色下的权限。"
        actions={
          <Space>
            <Input
              prefix={<Search size={14} />}
              placeholder="搜索 slug / 名称"
              value={searchInput}
              onChange={(v) => setSearchInput(v)}
              onPressEnter={() => { setPage(1); setSearch(searchInput) }}
              allowClear
              onClear={() => { setPage(1); setSearch('') }}
              style={{ width: 220 }}
            />
            <Button type="primary" onClick={() => { setPage(1); setSearch(searchInput) }}>搜索</Button>
            <Button onClick={() => void load()}>
              <span className="inline-flex items-center gap-1"><RefreshCw size={14} className={loading ? 'animate-spin' : ''} /> 刷新</span>
            </Button>
            <Button type="primary" onClick={openCreate}>
              <span className="inline-flex items-center gap-1"><Plus size={14} /> 新建角色</span>
            </Button>
          </Space>
        }
      />

      <Table
        rowKey={(r: Role) => String(r.id)}
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
        title={editing ? '编辑角色' : '新建角色'}
        visible={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={save}
        confirmLoading={saving}
        okText="保存"
        cancelText="取消"
        autoFocus={false}
      >
        <Form form={form} layout="vertical" autoComplete="off">
          <FormItem label="Slug" field="slug" rules={[{ required: true, message: '请填写 slug' }]}>
            <Input disabled={!!editing?.isSystem} placeholder="例如 admin / member" />
          </FormItem>
          <FormItem label="名称" field="name" rules={[{ required: true, message: '请填写名称' }]}>
            <Input />
          </FormItem>
          <FormItem label="描述" field="description">
            <Input />
          </FormItem>
        </Form>
      </Modal>

      <Drawer
        title={permModalRole ? `分配权限 — ${permModalRole.slug}` : ''}
        visible={!!permModalRole}
        onCancel={() => setPermModalRole(null)}
        onOk={savePermPick}
        okText="保存"
        cancelText="取消"
        confirmLoading={permSaving}
        width={520}
        autoFocus={false}
      >
        <Checkbox.Group
          value={permPick}
          onChange={(v) => setPermPick(v as number[])}
          direction="vertical"
          style={{ width: '100%' }}
        >
          {allPerms.map((p) => (
            <Checkbox key={p.id} value={p.id}>
              <span style={{ fontFamily: 'ui-monospace, monospace', fontSize: 12 }} className="text-[var(--color-primary-light-4)]">{p.key}</span>
              <span className="ml-2 text-sm">{p.name}</span>
            </Checkbox>
          ))}
        </Checkbox.Group>
        {allPerms.length === 0 && <div className="text-[var(--color-text-3)] text-sm">暂无可选权限</div>}
      </Drawer>
    </div>
  )
}

export default Roles
