// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// LLM Token 管理：用于 /v1/* Relay Gateway 鉴权的专属凭证。
// 与 UserCredential 解耦：无 API Secret，按 model_meta 倍率计费，可设白名单/额度/有效期。
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
  InputNumber,
  Progress,
  Typography,
  DatePicker,
} from '@arco-design/web-react'
import { Search, RefreshCw, Plus, Edit, Trash2, Eye, KeyRound, RotateCcw, Copy } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import {
  listLLMTokens,
  getLLMToken,
  createLLMToken,
  updateLLMToken,
  regenerateLLMToken,
  resetLLMTokenUsage,
  deleteLLMToken,
  listUsers,
  type LLMToken,
  type User,
} from '@/services/adminApi'

const Option = Select.Option
const FormItem = Form.Item

interface FormShape {
  user_id: number
  name: string
  type: 'llm' | 'asr' | 'tts'
  group: string
  status: 'active' | 'disabled' | 'expired'
  model_whitelist?: string
  unlimited_quota: boolean
  token_quota: number
  request_quota: number
  expires_at?: string
}

const TYPE_LABEL: Record<string, string> = { llm: 'LLM', asr: 'ASR', tts: 'TTS' }
const TYPE_COLOR: Record<string, string> = { llm: 'arcoblue', asr: 'purple', tts: 'cyan' }

const STATUS_COLOR: Record<string, string> = {
  active: 'green',
  disabled: 'gray',
  expired: 'orange',
}

const STATUS_LABEL: Record<string, string> = {
  active: '启用',
  disabled: '停用',
  expired: '已过期',
}

const LlmTokens = () => {
  const [rows, setRows] = useState<LLMToken[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)

  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [groupFilter, setGroupFilter] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<LLMToken | null>(null)
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm<FormShape>()

  const [detail, setDetail] = useState<LLMToken | null>(null)
  const [revealKey, setRevealKey] = useState<{ id: number; key: string } | null>(null)

  const fetchRows = async () => {
    setLoading(true)
    try {
      const r = await listLLMTokens({
        page,
        pageSize,
        search: search || undefined,
        status: statusFilter || undefined,
        group: groupFilter || undefined,
      })
      setRows(r.tokens || [])
      setTotal(r.total || 0)
    } catch (e: any) {
      Message.error(`加载失败：${e?.msg || e?.message || e}`)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchRows()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, search, statusFilter, groupFilter])

  const onSearch = () => { setPage(1); setSearch(searchInput) }

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({
      type: 'llm',
      group: 'default',
      status: 'active',
      unlimited_quota: false,
      token_quota: 0,
      request_quota: 0,
    } as any)
    setModalOpen(true)
  }

  const openEdit = (row: LLMToken) => {
    setEditing(row)
    form.setFieldsValue({
      user_id: row.user_id,
      name: row.name,
      type: row.type || 'llm',
      group: row.group,
      status: row.status,
      model_whitelist: row.model_whitelist || '',
      unlimited_quota: row.unlimited_quota,
      token_quota: row.token_quota,
      request_quota: row.request_quota,
      expires_at: row.expires_at || undefined,
    } as any)
    setModalOpen(true)
  }

  const save = async () => {
    try {
      const v = await form.validate()
      setSaving(true)
      const body = {
        user_id: v.user_id,
        name: v.name,
        type: v.type || 'llm',
        group: v.group || 'default',
        status: v.status,
        model_whitelist: v.model_whitelist || '',
        unlimited_quota: v.unlimited_quota,
        token_quota: v.token_quota,
        request_quota: v.request_quota,
        expires_at: v.expires_at || '',
      }
      if (editing) {
        await updateLLMToken(editing.id, body)
        Message.success('已更新')
        setModalOpen(false)
        fetchRows()
      } else {
        const r = await createLLMToken(body)
        Message.success('已创建')
        setModalOpen(false)
        fetchRows()
        // 创建成功后弹出 raw_api_key 让用户立即复制（仅此一次明文）
        if (r?.raw_api_key) {
          setRevealKey({ id: r.token.id, key: r.raw_api_key })
        }
      }
    } catch (e: any) {
      if (e?.message || e?.msg) Message.error(`保存失败：${e?.msg || e?.message}`)
    } finally {
      setSaving(false)
    }
  }

  const onRegenerate = async (row: LLMToken) => {
    try {
      const r = await regenerateLLMToken(row.id)
      Message.success('已重置 API Key')
      fetchRows()
      if (r?.raw_api_key) {
        setRevealKey({ id: r.token.id, key: r.raw_api_key })
      }
    } catch (e: any) {
      Message.error(`重置失败：${e?.msg || e?.message || e}`)
    }
  }

  const onResetUsage = async (row: LLMToken) => {
    try {
      await resetLLMTokenUsage(row.id)
      Message.success('已清零用量')
      fetchRows()
    } catch (e: any) {
      Message.error(`清零失败：${e?.msg || e?.message || e}`)
    }
  }

  const onDelete = async (row: LLMToken) => {
    try {
      await deleteLLMToken(row.id)
      Message.success('已删除')
      fetchRows()
    } catch (e: any) {
      Message.error(`删除失败：${e?.msg || e?.message || e}`)
    }
  }

  const openDetail = async (row: LLMToken) => {
    try {
      const t = await getLLMToken(row.id)
      setDetail(t)
    } catch (e: any) {
      Message.error(`加载详情失败：${e?.msg || e?.message || e}`)
    }
  }

  const copyKey = async (key: string) => {
    try {
      await navigator.clipboard.writeText(key)
      Message.success('已复制到剪贴板')
    } catch {
      Message.error('复制失败，请手动选中复制')
    }
  }

  const tokenUsageRatio = (r: LLMToken) => {
    if (r.unlimited_quota || r.token_quota <= 0) return null
    return Math.min(100, Math.round((r.token_used / r.token_quota) * 100))
  }

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: '名称', dataIndex: 'name', width: 180, ellipsis: true, render: (v: string) => v || '-' },
    {
      title: '类型',
      dataIndex: 'type',
      width: 90,
      render: (v: string) => <Tag color={TYPE_COLOR[v] || 'arcoblue'}>{TYPE_LABEL[v] || (v || 'LLM').toUpperCase()}</Tag>,
    },
    { title: '用户', dataIndex: 'user_id', width: 90 },
    {
      title: 'API Key',
      dataIndex: 'api_key',
      width: 220,
      render: (v: string) => <Typography.Text copyable={false} className="font-mono text-xs">{v}</Typography.Text>,
    },
    { title: '分组', dataIndex: 'group', width: 110, render: (v: string) => <Tag>{v}</Tag> },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (v: string) => <Tag color={STATUS_COLOR[v] || 'gray'}>{STATUS_LABEL[v] || v}</Tag>,
    },
    {
      title: 'Token 用量',
      key: '__usage__',
      width: 200,
      render: (_: any, r: LLMToken) => {
        if (r.unlimited_quota) return <Tag color="purple">不限额</Tag>
        if (r.token_quota <= 0) return <Tag color="gray">未授额</Tag>
        const ratio = tokenUsageRatio(r) || 0
        return (
          <div className="space-y-1">
            <div className="text-xs text-[var(--color-text-3)]">
              {r.token_used.toLocaleString()} / {r.token_quota.toLocaleString()}
            </div>
            <Progress percent={ratio} size="small" status={ratio >= 100 ? 'error' : ratio >= 80 ? 'warning' : 'normal'} showText={false} />
          </div>
        )
      },
    },
    {
      title: '请求数',
      dataIndex: 'request_used',
      width: 130,
      render: (_: any, r: LLMToken) =>
        r.request_quota > 0 ? `${r.request_used.toLocaleString()} / ${r.request_quota.toLocaleString()}` : r.request_used.toLocaleString(),
    },
    {
      title: '过期时间',
      dataIndex: 'expires_at',
      width: 170,
      render: (v: string) => (v ? new Date(v).toLocaleString('zh-CN') : '永久'),
    },
    {
      title: '操作',
      key: '__actions__',
      width: 360,
      fixed: 'right' as const,
      render: (_: any, row: LLMToken) => (
        <Space size="mini">
          <Button size="mini" type="text" onClick={() => openDetail(row)}>
            <span className="inline-flex items-center gap-1"><Eye size={14} /> 详情</span>
          </Button>
          <Button size="mini" type="text" onClick={() => openEdit(row)}>
            <span className="inline-flex items-center gap-1"><Edit size={14} /> 编辑</span>
          </Button>
          <Popconfirm
            title="重置 API Key？老 Key 将立即失效。"
            okText="重置"
            cancelText="取消"
            onOk={() => onRegenerate(row)}
          >
            <Button size="mini" type="text">
              <span className="inline-flex items-center gap-1"><KeyRound size={14} /> 重置 Key</span>
            </Button>
          </Popconfirm>
          <Popconfirm
            title="清零所有用量计数？"
            okText="清零"
            cancelText="取消"
            onOk={() => onResetUsage(row)}
          >
            <Button size="mini" type="text">
              <span className="inline-flex items-center gap-1"><RotateCcw size={14} /> 清零</span>
            </Button>
          </Popconfirm>
          <Popconfirm
            title={`确定删除「${row.name || row.id}」？`}
            okText="删除"
            cancelText="取消"
            onOk={() => onDelete(row)}
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
        title="LLM API Token"
        description="对外 /v1/* Relay Gateway 鉴权专用。每个 Token 可独立设额度、有效期、模型白名单、分组路由；用量按 LLMModelMeta 倍率自动扣减。"
        actions={
          <Space wrap>
            <Input
              prefix={<Search size={14} />}
              placeholder="搜索 名称 / api_key"
              value={searchInput}
              onChange={(v) => setSearchInput(v)}
              onPressEnter={onSearch}
              allowClear
              onClear={() => { setPage(1); setSearch('') }}
              style={{ width: 240 }}
            />
            <Select value={statusFilter} onChange={(v) => { setPage(1); setStatusFilter(v) }} placeholder="状态" style={{ width: 130 }}>
              <Option value="">全部</Option>
              <Option value="active">启用</Option>
              <Option value="disabled">停用</Option>
              <Option value="expired">过期</Option>
            </Select>
            <Input
              placeholder="分组 group"
              value={groupFilter}
              onChange={(v) => { setPage(1); setGroupFilter(v) }}
              allowClear
              style={{ width: 140 }}
            />
            <Button type="primary" onClick={onSearch}>搜索</Button>
            <Button onClick={fetchRows}>
              <span className="inline-flex items-center gap-1"><RefreshCw size={14} className={loading ? 'animate-spin' : ''} /> 刷新</span>
            </Button>
            <Button type="primary" onClick={openCreate}>
              <span className="inline-flex items-center gap-1"><Plus size={14} /> 新建 Token</span>
            </Button>
          </Space>
        }
      />

      <Table
        rowKey={(r: LLMToken) => String(r.id)}
        loading={loading}
        columns={columns}
        data={rows}
        scroll={{ x: 1700 }}
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
        title={editing ? '编辑 Token' : '新建 Token'}
        visible={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={save}
        confirmLoading={saving}
        okText="保存"
        cancelText="取消"
        autoFocus={false}
        style={{ width: 720 }}
      >
        <Form form={form} layout="vertical" autoComplete="off">
          <div className="grid grid-cols-2 gap-3">
            <FormItem label="所属用户" field="user_id" rules={[{ required: true, type: 'number', min: 1, message: '请选择用户' }]}>
              <UserSelect disabled={!!editing} />
            </FormItem>
            <FormItem label="名称" field="name">
              <Input placeholder="备注用途，便于识别" />
            </FormItem>
          </div>
          <div className="grid grid-cols-3 gap-3">
            <FormItem label="类型" field="type" rules={[{ required: true }]} extra="决定该 Token 可调用的 relay 协议族">
              <Select>
                <Option value="llm">LLM · 聊天补全</Option>
                <Option value="asr">ASR · 语音识别</Option>
                <Option value="tts">TTS · 语音合成</Option>
              </Select>
            </FormItem>
            <FormItem label="分组 group" field="group" rules={[{ required: true }]} extra="路由分组，与 abilities/channels 一致">
              <Input placeholder="default" />
            </FormItem>
            <FormItem label="状态" field="status" rules={[{ required: true }]}>
              <Select>
                <Option value="active">启用</Option>
                <Option value="disabled">停用</Option>
                <Option value="expired">已过期</Option>
              </Select>
            </FormItem>
          </div>
          <FormItem label="模型白名单" field="model_whitelist" extra="逗号分隔，留空 = 不限制；例如 gpt-4o-mini,gpt-4o">
            <Input.TextArea rows={2} />
          </FormItem>
          <div className="grid grid-cols-3 gap-3">
            <FormItem label="不限额度" field="unlimited_quota" triggerPropName="checked"><Switch /></FormItem>
            <FormItem label="Token 配额" field="token_quota" extra="0 + 不限额=否 视为禁用">
              <InputNumber min={0} step={10000} />
            </FormItem>
            <FormItem label="请求数配额" field="request_quota">
              <InputNumber min={0} step={100} />
            </FormItem>
          </div>
          <FormItem label="过期时间" field="expires_at" extra="留空表示永久；前端转 RFC3339 提交">
            <DatePicker showTime style={{ width: '100%' }} format="YYYY-MM-DDTHH:mm:ssZ" />
          </FormItem>
        </Form>
      </Modal>

      {/* 创建 / 重置后展示完整 Key 的弹窗（仅此一次明文） */}
      <Modal
        title="API Key 已生成"
        visible={!!revealKey}
        onCancel={() => setRevealKey(null)}
        onOk={() => setRevealKey(null)}
        okText="我已复制"
        cancelText="取消"
        autoFocus={false}
      >
        {revealKey && (
          <div className="space-y-3">
            <div className="text-sm text-[var(--color-text-3)]">
              这是你的完整 API Key。<strong>只展示这一次</strong>，请立即复制保存；离开后将无法再次查看明文。
            </div>
            <div className="rounded border border-[var(--color-border-2)] bg-[var(--color-fill-2)] p-3 break-all font-mono text-sm">
              {revealKey.key}
            </div>
            <Button type="primary" onClick={() => copyKey(revealKey.key)}>
              <span className="inline-flex items-center gap-1"><Copy size={14} /> 复制 API Key</span>
            </Button>
          </div>
        )}
      </Modal>

      <Drawer
        title="Token 详情"
        visible={!!detail}
        width={700}
        onCancel={() => setDetail(null)}
        footer={null}
        autoFocus={false}
      >
        {detail && (
          <div className="space-y-3 text-sm">
            <Row label="ID" value={String(detail.id)} />
            <Row label="名称" value={detail.name || '-'} />
            <Row label="用户" value={String(detail.user_id)} />
            <Row label="API Key" value={<span className="font-mono break-all">{detail.api_key}</span>} />
            <Row label="状态" value={<Tag color={STATUS_COLOR[detail.status]}>{STATUS_LABEL[detail.status]}</Tag>} />
            <Row label="分组" value={detail.group} />
            <Row label="模型白名单" value={detail.model_whitelist || '不限制'} />
            <Row label="不限额度" value={detail.unlimited_quota ? '是' : '否'} />
            <Row label="Token 配额" value={`${detail.token_used.toLocaleString()} / ${detail.unlimited_quota ? '∞' : detail.token_quota.toLocaleString()}`} />
            <Row label="额度单位" value={detail.quota_used.toLocaleString()} />
            <Row label="请求次数" value={`${detail.request_used.toLocaleString()} / ${detail.request_quota > 0 ? detail.request_quota.toLocaleString() : '∞'}`} />
            <Row label="过期时间" value={detail.expires_at ? new Date(detail.expires_at).toLocaleString('zh-CN') : '永久'} />
            <Row label="最近使用" value={detail.last_used_at ? new Date(detail.last_used_at).toLocaleString('zh-CN') : '从未'} />
            <Row label="创建时间" value={new Date(detail.created_at).toLocaleString('zh-CN')} />
            <Row label="更新时间" value={new Date(detail.updated_at).toLocaleString('zh-CN')} />
          </div>
        )}
      </Drawer>
    </div>
  )
}

const Row = ({ label, value }: { label: string; value: React.ReactNode }) => (
  <div className="flex items-start gap-3">
    <div className="w-28 shrink-0 text-[var(--color-text-3)]">{label}</div>
    <div className="flex-1 break-all text-[var(--color-text-1)]">{value}</div>
  </div>
)

// UserSelect 远程搜索用户：支持邮箱 / displayName / 手机号关键字检索；
// 受控组件 onChange/value 与 Arco Form.Item 默认协议一致（直接传 user_id 数字）。
interface UserSelectProps {
  value?: number
  onChange?: (v: number) => void
  disabled?: boolean
}
const UserSelect = ({ value, onChange, disabled }: UserSelectProps) => {
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(false)
  const [keyword, setKeyword] = useState('')

  const fetchList = async (kw: string) => {
    setLoading(true)
    try {
      const r = await listUsers({ page: 1, pageSize: 50, search: kw || undefined })
      const list = r.users || []
      // 把当前选中的 user_id 拼回去，避免编辑态选项缺失
      if (value && !list.find((u) => u.id === value)) {
        try {
          const single = await listUsers({ page: 1, pageSize: 1, search: String(value) })
          const matched = (single.users || []).find((u) => u.id === value)
          if (matched) list.unshift(matched)
        } catch { /* 忽略 */ }
      }
      setUsers(list)
    } catch (e: any) {
      Message.error(`加载用户失败：${e?.msg || e?.message || e}`)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchList(keyword)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [keyword])

  return (
    <Select
      showSearch
      placeholder="搜索邮箱 / 用户名 / 手机号"
      value={value}
      onChange={(v: number) => onChange?.(v)}
      disabled={disabled}
      loading={loading}
      filterOption={false}
      onSearch={(kw: string) => setKeyword(kw)}
      notFoundContent={loading ? '加载中…' : '无匹配用户'}
    >
      {users.map((u) => (
        <Option key={u.id} value={u.id}>
          <span className="font-mono text-xs text-[var(--color-text-3)]">#{u.id}</span>
          <span className="ml-2">{u.displayName || u.email || `User ${u.id}`}</span>
          {u.email && <span className="ml-2 text-[var(--color-text-3)] text-xs">{u.email}</span>}
        </Option>
      ))}
    </Select>
  )
}

export default LlmTokens
