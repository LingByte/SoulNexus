// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 密钥管理：Arco Table + 详情/额度编辑 Drawer。
import { useEffect, useState } from 'react'
import {
  Table,
  Input,
  InputNumber,
  Select,
  Button,
  Tag,
  Drawer,
  Switch,
  Popconfirm,
  Message,
  Space,
} from '@arco-design/web-react'
import { Search, RefreshCw, Download, Eye, Trash2, Save } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import {
  listAdminCredentials,
  updateAdminCredentialStatus,
  deleteAdminCredential,
  type AdminCredential,
} from '@/services/adminApi'

const Option = Select.Option

const STATUS_OPTIONS = [
  { label: '全部状态', value: '' },
  { label: 'active', value: 'active' },
  { label: 'banned', value: 'banned' },
  { label: 'suspended', value: 'suspended' },
]

const DATE_NEVER = '1970-01-01 07:59:59'

const formatDateTime = (value?: string): string => {
  if (!value) return ''
  const dt = new Date(value)
  if (Number.isNaN(dt.getTime())) return value
  const y = dt.getFullYear()
  const m = `${dt.getMonth() + 1}`.padStart(2, '0')
  const d = `${dt.getDate()}`.padStart(2, '0')
  const h = `${dt.getHours()}`.padStart(2, '0')
  const mi = `${dt.getMinutes()}`.padStart(2, '0')
  const s = `${dt.getSeconds()}`.padStart(2, '0')
  return `${y}-${m}-${d} ${h}:${mi}:${s}`
}

const addDays = (days: number): string => {
  const now = new Date()
  now.setDate(now.getDate() + days)
  return formatDateTime(now.toISOString())
}

const statusColor = (s?: string) => {
  switch (s) {
    case 'active': return 'green'
    case 'banned': return 'red'
    case 'suspended': return 'orange'
    default: return 'gray'
  }
}

interface EditForm {
  expiresAt: string
  tokenQuota: number
  requestQuota: number
  useNativeQuota: boolean
  unlimitedQuota: boolean
}

const Credentials = () => {
  const [list, setList] = useState<AdminCredential[]>([])
  const [loading, setLoading] = useState(false)
  const [status, setStatus] = useState('')
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)

  const [detail, setDetail] = useState<AdminCredential | null>(null)
  const [saving, setSaving] = useState(false)
  const [editForm, setEditForm] = useState<EditForm>({
    expiresAt: '',
    tokenQuota: 0,
    requestQuota: 0,
    useNativeQuota: false,
    unlimitedQuota: true,
  })

  const fetchData = async () => {
    setLoading(true)
    try {
      const res = await listAdminCredentials({
        status: status || undefined,
        search: search || undefined,
        page,
        pageSize,
      })
      setList(res.credentials || [])
      setTotal(res.total || 0)
    } catch (e: any) {
      Message.error(`加载密钥失败：${e?.msg || e?.message || e}`)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status, search, page, pageSize])

  const updateStatus = async (id: number, next: 'active' | 'banned' | 'suspended') => {
    try {
      await updateAdminCredentialStatus(id, { status: next })
      Message.success('状态已更新')
      fetchData()
    } catch (e: any) {
      Message.error(`更新失败：${e?.msg || e?.message || e}`)
    }
  }

  const onDelete = async (id: number) => {
    try {
      await deleteAdminCredential(id)
      Message.success('已删除')
      fetchData()
    } catch (e: any) {
      Message.error(`删除失败：${e?.msg || e?.message || e}`)
    }
  }

  const openDetail = (item: AdminCredential) => {
    setDetail(item)
    setEditForm({
      expiresAt: formatDateTime(item.expiresAt),
      tokenQuota: item.tokenQuota || 0,
      requestQuota: item.requestQuota || 0,
      useNativeQuota: !!item.useNativeQuota,
      unlimitedQuota: item.unlimitedQuota !== false,
    })
  }

  const saveSettings = async () => {
    if (!detail) return
    setSaving(true)
    try {
      await updateAdminCredentialStatus(detail.id, {
        status: detail.status,
        expiresAt: editForm.expiresAt.trim(),
        tokenQuota: Math.max(0, Number(editForm.tokenQuota || 0)),
        requestQuota: Math.max(0, Number(editForm.requestQuota || 0)),
        useNativeQuota: !!editForm.useNativeQuota,
        unlimitedQuota: !!editForm.unlimitedQuota,
      })
      Message.success('密钥设置已保存')
      setDetail(null)
      fetchData()
    } catch (e: any) {
      Message.error(`保存失败：${e?.msg || e?.message || e}`)
    } finally {
      setSaving(false)
    }
  }

  const exportCsv = () => {
    if (list.length === 0) {
      Message.info('当前页无数据可导出')
      return
    }
    const headers = ['id', 'name', 'userId', 'status', 'llmProvider', 'usageCount', 'createdAt']
    const rows = list.map((i) => [i.id, i.name, i.userId, i.status, i.llmProvider || '', i.usageCount, i.createdAt])
    const csv = [headers, ...rows]
      .map((row) => row.map((v) => `"${String(v ?? '').replace(/"/g, '""')}"`).join(','))
      .join('\n')
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `credentials_${Date.now()}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 100 },
    { title: '名称', dataIndex: 'name', ellipsis: true },
    { title: '所属用户', dataIndex: 'userId', width: 100 },
    { title: 'Provider', dataIndex: 'llmProvider', width: 130, render: (v: string) => v || '-' },
    { title: '使用次数', dataIndex: 'usageCount', width: 100 },
    {
      title: '状态',
      dataIndex: 'status',
      width: 110,
      render: (s: string) => <Tag color={statusColor(s)}>{s}</Tag>,
    },
    {
      title: '操作',
      key: '__actions__',
      width: 380,
      fixed: 'right' as const,
      render: (_: any, item: AdminCredential) => (
        <Space size="mini">
          <Button size="mini" type="text" onClick={() => openDetail(item)}>
            <span className="inline-flex items-center gap-1"><Eye size={14} /> 详情</span>
          </Button>
          <Button size="mini" type="text" disabled={item.status === 'active'} onClick={() => updateStatus(item.id, 'active')}>启用</Button>
          <Button size="mini" type="text" disabled={item.status === 'suspended'} onClick={() => updateStatus(item.id, 'suspended')}>暂停</Button>
          <Button size="mini" type="text" status="warning" disabled={item.status === 'banned'} onClick={() => updateStatus(item.id, 'banned')}>封禁</Button>
          <Popconfirm title="确认删除该密钥？" okText="删除" cancelText="取消" onOk={() => onDelete(item.id)}>
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
        title="密钥管理"
        description="管理用户 API 密钥状态、过期时间与令牌额度。"
        actions={
          <Space>
            <Input
              prefix={<Search size={14} />}
              placeholder="名称 / API Key / Provider"
              value={searchInput}
              onChange={(v) => setSearchInput(v)}
              onPressEnter={() => { setPage(1); setSearch(searchInput) }}
              allowClear
              onClear={() => { setPage(1); setSearch('') }}
              style={{ width: 240 }}
            />
            <Select value={status} onChange={(v) => { setPage(1); setStatus(v) }} style={{ width: 140 }}>
              {STATUS_OPTIONS.map((o) => <Option key={o.value} value={o.value}>{o.label}</Option>)}
            </Select>
            <Button type="primary" onClick={() => { setPage(1); setSearch(searchInput) }}>搜索</Button>
            <Button onClick={fetchData}>
              <span className="inline-flex items-center gap-1"><RefreshCw size={14} className={loading ? 'animate-spin' : ''} /> 刷新</span>
            </Button>
            <Button onClick={exportCsv}>
              <span className="inline-flex items-center gap-1"><Download size={14} /> 导出 CSV</span>
            </Button>
          </Space>
        }
      />

      <Table
        rowKey={(r: AdminCredential) => String(r.id)}
        loading={loading}
        columns={columns}
        data={list}
        scroll={{ x: 1200 }}
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

      <Drawer
        title="密钥详情"
        visible={!!detail}
        width={560}
        onCancel={() => setDetail(null)}
        onOk={saveSettings}
        confirmLoading={saving}
        okText={<span className="inline-flex items-center gap-1"><Save size={14} /> 保存设置</span>}
        cancelText="取消"
        autoFocus={false}
      >
        {detail && (
          <div className="space-y-3 text-sm">
            <InfoRow label="ID" value={String(detail.id)} />
            <InfoRow label="名称" value={detail.name} />
            <InfoRow label="所属用户" value={String(detail.userId)} />
            <InfoRow label="状态" value={<Tag color={statusColor(detail.status)}>{detail.status}</Tag>} />
            <InfoRow label="API Key" value={<span style={{ fontFamily: 'ui-monospace, monospace', fontSize: 12 }}>{detail.apiKey}</span>} />
            <InfoRow label="Provider" value={detail.llmProvider || '-'} />
            <InfoRow label="使用次数" value={String(detail.usageCount)} />
            <InfoRow label="最近使用" value={detail.lastUsedAt || '-'} />

            <div className="pt-3 border-t border-[var(--color-border-2)]">
              <div className="font-medium mb-2">过期时间</div>
              <Space wrap>
                <Button size="mini" onClick={() => setEditForm((p) => ({ ...p, expiresAt: addDays(1) }))}>+1 天</Button>
                <Button size="mini" onClick={() => setEditForm((p) => ({ ...p, expiresAt: addDays(7) }))}>+7 天</Button>
                <Button size="mini" onClick={() => setEditForm((p) => ({ ...p, expiresAt: addDays(30) }))}>+30 天</Button>
                <Button size="mini" onClick={() => setEditForm((p) => ({ ...p, expiresAt: DATE_NEVER }))}>永不</Button>
                <Button size="mini" type="text" onClick={() => setEditForm((p) => ({ ...p, expiresAt: '' }))}>清空</Button>
              </Space>
              <Input
                value={editForm.expiresAt}
                onChange={(v) => setEditForm((p) => ({ ...p, expiresAt: v }))}
                placeholder="YYYY-MM-DD HH:MM:SS"
                style={{ marginTop: 8 }}
              />
            </div>

            <div className="pt-3 border-t border-[var(--color-border-2)]">
              <div className="font-medium mb-2">令牌分组额度</div>
              <div className="space-y-2">
                <Field label="使用原生额度">
                  <Switch checked={editForm.useNativeQuota} onChange={(v) => setEditForm((p) => ({ ...p, useNativeQuota: v }))} />
                </Field>
                <Field label="无限额度">
                  <Switch checked={editForm.unlimitedQuota} onChange={(v) => setEditForm((p) => ({ ...p, unlimitedQuota: v }))} />
                </Field>
                <Field label="令牌可用额度">
                  <InputNumber
                    value={editForm.tokenQuota}
                    onChange={(v) => setEditForm((p) => ({ ...p, tokenQuota: Number(v ?? 0) }))}
                    min={0}
                    style={{ width: 200 }}
                    disabled={editForm.unlimitedQuota}
                  />
                </Field>
                <Field label="令牌可用数量">
                  <InputNumber
                    value={editForm.requestQuota}
                    onChange={(v) => setEditForm((p) => ({ ...p, requestQuota: Number(v ?? 0) }))}
                    min={0}
                    style={{ width: 200 }}
                    disabled={editForm.unlimitedQuota}
                  />
                </Field>
              </div>
            </div>
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

const Field = ({ label, children }: { label: string; children: React.ReactNode }) => (
  <div className="flex items-center gap-3">
    <div className="w-32 shrink-0 text-[var(--color-text-3)]">{label}</div>
    <div className="flex-1">{children}</div>
  </div>
)

export default Credentials
