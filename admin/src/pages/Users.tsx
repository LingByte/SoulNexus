// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 用户管理：Arco Table + Form Modal（创建/编辑）+ 详情 Drawer。
import { useEffect, useState } from 'react'
import {
  Table,
  Input,
  Select,
  Button,
  Tag,
  Modal,
  Form,
  Drawer,
  Switch,
  Popconfirm,
  Message,
  Space,
} from '@arco-design/web-react'
import { Search, RefreshCw, Plus, Edit, Trash2, Eye } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import {
  listUsers,
  getUser,
  createUser,
  updateUser,
  deleteUser,
  type User,
  type ListUsersParams,
  type CreateUserRequest,
  type UpdateUserRequest,
} from '@/services/adminApi'

const Option = Select.Option
const FormItem = Form.Item

const ROLES = [
  { value: 'user', label: '普通用户' },
  { value: 'admin', label: '管理员' },
  { value: 'superadmin', label: '超级管理员' },
]

const ACCOUNT_STATUSES = [
  { value: 'active', label: '正常' },
  { value: 'pending_verification', label: '待验证' },
  { value: 'suspended', label: '暂停' },
  { value: 'banned', label: '封禁' },
]

const accountStatusLabel = (s?: string) => {
  const v = s || 'active'
  return ACCOUNT_STATUSES.find((x) => x.value === v)?.label || v
}

const statusColor = (s?: string) => {
  switch (s) {
    case 'active': return 'green'
    case 'banned': return 'red'
    case 'suspended': return 'orange'
    case 'pending_verification': return 'arcoblue'
    default: return 'gray'
  }
}

const Users = () => {
  const [users, setUsers] = useState<User[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)

  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [roleFilter, setRoleFilter] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [hasPhoneFilter, setHasPhoneFilter] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<User | null>(null)
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm<CreateUserRequest>()

  const [detail, setDetail] = useState<User | null>(null)

  const fetchUsers = async () => {
    setLoading(true)
    try {
      const params: ListUsersParams = { page, pageSize }
      if (search) params.search = search
      if (roleFilter) params.role = roleFilter
      if (statusFilter === '__inactive__') params.enabled = 'false'
      else if (statusFilter) params.status = statusFilter
      if (hasPhoneFilter) params.hasPhone = hasPhoneFilter
      const response = await listUsers(params)
      setUsers(response.users || [])
      setTotal(response.total || 0)
    } catch (e: any) {
      Message.error(`获取用户列表失败：${e?.msg || e?.message || e}`)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchUsers()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, search, roleFilter, statusFilter, hasPhoneFilter])

  const handleSearch = () => {
    setPage(1)
    setSearch(searchInput)
  }

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({
      email: '',
      displayName: '',
      firstName: '',
      lastName: '',
      role: 'user',
      status: 'active',
      phone: '',
      emailNotifications: true,
      pushNotifications: true,
    })
    setModalOpen(true)
  }

  const openEdit = async (user: User) => {
    try {
      const full = await getUser(user.id)
      setEditing(full)
      form.setFieldsValue({
        email: full.email,
        displayName: full.displayName || '',
        firstName: full.firstName || '',
        lastName: full.lastName || '',
        role: full.role || 'user',
        status: full.status || 'active',
        phone: full.phone || '',
        locale: full.locale || '',
        timezone: full.timezone || '',
        city: (full as any).city || '',
        region: (full as any).region || '',
        gender: (full as any).gender || '',
        emailNotifications: (full as any).emailNotifications ?? true,
        pushNotifications: (full as any).pushNotifications ?? true,
      } as any)
      setModalOpen(true)
    } catch (e: any) {
      Message.error(`获取用户详情失败：${e?.msg || e?.message || e}`)
    }
  }

  const save = async () => {
    try {
      const values = await form.validate()
      setSaving(true)
      if (editing) {
        const updateData: UpdateUserRequest = { ...values }
        if (!updateData.password) delete (updateData as any).password
        await updateUser(editing.id, updateData)
        Message.success('更新用户成功')
      } else {
        await createUser(values as CreateUserRequest)
        Message.success('创建用户成功')
      }
      setModalOpen(false)
      fetchUsers()
    } catch (e: any) {
      if (e?.message || e?.msg) Message.error(`保存失败：${e?.msg || e?.message}`)
    } finally {
      setSaving(false)
    }
  }

  const onDelete = async (u: User) => {
    try {
      await deleteUser(u.id)
      Message.success('已删除')
      fetchUsers()
    } catch (e: any) {
      Message.error(`删除失败：${e?.msg || e?.message || e}`)
    }
  }

  const openDetail = async (u: User) => {
    try {
      const full = await getUser(u.id)
      setDetail(full)
    } catch (e: any) {
      Message.error(`获取详情失败：${e?.msg || e?.message || e}`)
    }
  }

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 100 },
    { title: '邮箱', dataIndex: 'email', width: 220, ellipsis: true },
    {
      title: '姓名',
      dataIndex: 'displayName',
      ellipsis: true,
      render: (_: any, u: User) => u.displayName || `${u.firstName || ''} ${u.lastName || ''}`.trim() || '-',
    },
    {
      title: '角色',
      dataIndex: 'role',
      width: 110,
      render: (v: string) => (
        <Tag color={v === 'admin' || v === 'superadmin' ? 'arcoblue' : 'gray'}>{v || 'user'}</Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 110,
      render: (v: string) => <Tag color={statusColor(v)}>{accountStatusLabel(v)}</Tag>,
    },
    { title: '手机号', dataIndex: 'phone', width: 130, render: (v: string) => v || '-' },
    {
      title: '最近登录',
      dataIndex: 'lastLogin',
      width: 170,
      render: (v: string) => (v ? new Date(v).toLocaleString('zh-CN') : '-'),
    },
    { title: '登录次数', dataIndex: 'loginCount', width: 100, render: (v: number) => v || 0 },
    {
      title: '操作',
      key: '__actions__',
      width: 240,
      fixed: 'right' as const,
      render: (_: any, u: User) => (
        <Space size="mini">
          <Button size="mini" type="text" onClick={() => openDetail(u)}>
            <span className="inline-flex items-center gap-1"><Eye size={14} /> 详情</span>
          </Button>
          <Button size="mini" type="text" onClick={() => openEdit(u)}>
            <span className="inline-flex items-center gap-1"><Edit size={14} /> 编辑</span>
          </Button>
          <Popconfirm
            title={`确定删除用户「${u.email}」？此操作不可恢复。`}
            okText="删除"
            cancelText="取消"
            onOk={() => onDelete(u)}
          >
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
        title="用户管理"
        description="维护后台用户、角色、状态与基础资料。"
        actions={
          <Space wrap>
            <Input
              prefix={<Search size={14} />}
              placeholder="搜索邮箱 / 姓名"
              value={searchInput}
              onChange={(v) => setSearchInput(v)}
              onPressEnter={handleSearch}
              allowClear
              onClear={() => { setPage(1); setSearch('') }}
              style={{ width: 220 }}
            />
            <Select value={roleFilter} onChange={(v) => { setPage(1); setRoleFilter(v) }} placeholder="角色" style={{ width: 130 }}>
              <Option value="">全部角色</Option>
              {ROLES.map((r) => <Option key={r.value} value={r.value}>{r.label}</Option>)}
            </Select>
            <Select value={statusFilter} onChange={(v) => { setPage(1); setStatusFilter(v) }} placeholder="状态" style={{ width: 130 }}>
              <Option value="">全部状态</Option>
              {ACCOUNT_STATUSES.map((s) => <Option key={s.value} value={s.value}>{s.label}</Option>)}
              <Option value="__inactive__">非活跃</Option>
            </Select>
            <Select value={hasPhoneFilter} onChange={(v) => { setPage(1); setHasPhoneFilter(v) }} placeholder="手机号" style={{ width: 130 }}>
              <Option value="">全部</Option>
              <Option value="true">有手机号</Option>
              <Option value="false">无手机号</Option>
            </Select>
            <Button type="primary" onClick={handleSearch}>搜索</Button>
            <Button onClick={fetchUsers}>
              <span className="inline-flex items-center gap-1"><RefreshCw size={14} className={loading ? 'animate-spin' : ''} /> 刷新</span>
            </Button>
            <Button type="primary" onClick={openCreate}>
              <span className="inline-flex items-center gap-1"><Plus size={14} /> 新建用户</span>
            </Button>
          </Space>
        }
      />

      <Table
        rowKey={(r: User) => String(r.id)}
        loading={loading}
        columns={columns}
        data={users}
        scroll={{ x: 1500 }}
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
        title={editing ? '编辑用户' : '新建用户'}
        visible={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={save}
        confirmLoading={saving}
        okText="保存"
        cancelText="取消"
        autoFocus={false}
        style={{ width: 680 }}
      >
        <Form form={form} layout="vertical" autoComplete="off">
          <FormItem label="邮箱" field="email" rules={[{ required: true, type: 'email', message: '请填写有效邮箱' }]}>
            <Input disabled={!!editing} placeholder="user@example.com" />
          </FormItem>
          <div className="grid grid-cols-2 gap-3">
            <FormItem label="显示名称" field="displayName"><Input /></FormItem>
            <FormItem label="手机号" field="phone"><Input /></FormItem>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <FormItem label="名" field="firstName"><Input /></FormItem>
            <FormItem label="姓" field="lastName"><Input /></FormItem>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <FormItem label="角色" field="role" rules={[{ required: true }]}>
              <Select>
                {ROLES.map((r) => <Option key={r.value} value={r.value}>{r.label}</Option>)}
              </Select>
            </FormItem>
            <FormItem label="账号状态" field="status" rules={[{ required: true }]}>
              <Select>
                {ACCOUNT_STATUSES.map((s) => <Option key={s.value} value={s.value}>{s.label}</Option>)}
              </Select>
            </FormItem>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <FormItem label="语言" field="locale"><Input placeholder="zh-CN / en-US" /></FormItem>
            <FormItem label="时区" field="timezone"><Input placeholder="Asia/Shanghai" /></FormItem>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <FormItem label="城市" field="city"><Input /></FormItem>
            <FormItem label="地区" field="region"><Input /></FormItem>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <FormItem label="邮件通知" field="emailNotifications" triggerPropName="checked"><Switch /></FormItem>
            <FormItem label="推送通知" field="pushNotifications" triggerPropName="checked"><Switch /></FormItem>
          </div>
          {editing && (
            <FormItem label="新密码（留空则不改）" field="password">
              <Input.Password placeholder="留空保持原密码" />
            </FormItem>
          )}
        </Form>
      </Modal>

      <Drawer
        title="用户详情"
        visible={!!detail}
        width={640}
        onCancel={() => setDetail(null)}
        footer={null}
        autoFocus={false}
      >
        {detail && (
          <div className="space-y-3 text-sm">
            <InfoRow label="ID" value={String(detail.id)} />
            <InfoRow label="邮箱" value={detail.email} />
            <InfoRow label="显示名" value={detail.displayName || '-'} />
            <InfoRow label="角色" value={<Tag color={detail.role === 'admin' || detail.role === 'superadmin' ? 'arcoblue' : 'gray'}>{detail.role || 'user'}</Tag>} />
            <InfoRow label="状态" value={<Tag color={statusColor(detail.status)}>{accountStatusLabel(detail.status)}</Tag>} />
            <InfoRow label="手机号" value={detail.phone || '-'} />
            <InfoRow label="时区" value={detail.timezone || '-'} />
            <InfoRow label="城市" value={(detail as any).city || '-'} />
            <InfoRow label="地区" value={(detail as any).region || '-'} />
            <InfoRow label="性别" value={(detail as any).gender || '-'} />
            <InfoRow label="最近登录" value={detail.lastLogin ? new Date(detail.lastLogin).toLocaleString('zh-CN') : '-'} />
            <InfoRow label="登录次数" value={String(detail.loginCount || 0)} />
            <InfoRow label="来源" value={detail.source || '-'} />
            <InfoRow label="邮件通知" value={(detail as any).emailNotifications ? '开启' : '关闭'} />
            <InfoRow label="推送通知" value={(detail as any).pushNotifications ? '开启' : '关闭'} />
          </div>
        )}
      </Drawer>
    </div>
  )
}

const InfoRow = ({ label, value }: { label: string; value: React.ReactNode }) => (
  <div className="flex items-start gap-3">
    <div className="w-24 shrink-0 text-[var(--color-text-3)]">{label}</div>
    <div className="flex-1 break-all text-[var(--color-text-1)]">{value}</div>
  </div>
)

export default Users
