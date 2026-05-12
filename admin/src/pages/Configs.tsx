// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 系统配置管理：使用 Arco Design 的 Table + Modal + Form，对齐 SpeechChannels / LlmChannels 的交互风格。
// 字段：key / desc / value / format(json|yaml|int|float|bool|text) / autoload / public

import { useEffect, useMemo, useState } from 'react'
import {
  Table,
  Input,
  Select,
  Button,
  Tag,
  Modal,
  Form,
  Switch,
  Popconfirm,
  Message,
  Space,
} from '@arco-design/web-react'
import {
  Search,
  RefreshCw,
  Plus,
  Edit,
  Trash2,
  Eye,
  EyeOff,
  CheckCircle2,
  XCircle,
} from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import {
  listConfigs,
  createConfig,
  updateConfig,
  deleteConfig,
  type Config,
  type ListConfigsParams,
} from '@/services/adminApi'

const Option = Select.Option
const FormItem = Form.Item
const TextArea = Input.TextArea

const FORMATS: Array<{ value: Config['format']; label: string; color: string }> = [
  { value: 'text', label: 'Text', color: 'gray' },
  { value: 'json', label: 'JSON', color: 'arcoblue' },
  { value: 'yaml', label: 'YAML', color: 'purple' },
  { value: 'int', label: 'Integer', color: 'green' },
  { value: 'float', label: 'Float', color: 'lime' },
  { value: 'bool', label: 'Boolean', color: 'orange' },
]

const STORAGE_TYPES = [
  { value: 'qiniu', label: '七牛云' },
  { value: 'cos', label: '腾讯云 COS' },
  { value: 'oss', label: '阿里云 OSS' },
  { value: 'minio', label: 'MinIO' },
  { value: 'local', label: '本地存储' },
]

const BOOL_OPTIONS = [
  { label: '全部', value: '' },
  { label: '是', value: 'true' },
  { label: '否', value: 'false' },
]

interface FormShape {
  key: string
  desc?: string
  value: string
  format: Config['format']
  autoload: boolean
  public: boolean
}

const Configs = () => {
  const [rows, setRows] = useState<Config[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)

  // 筛选
  const [search, setSearch] = useState('')
  const [autoloadFilter, setAutoloadFilter] = useState('')
  const [publicFilter, setPublicFilter] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)

  // 行内值显示/隐藏
  const [showValue, setShowValue] = useState<Record<string, boolean>>({})

  // 编辑 Modal
  const [modalOpen, setModalOpen] = useState(false)
  const [saving, setSaving] = useState(false)
  const [editing, setEditing] = useState<Config | null>(null)
  const [form] = Form.useForm<FormShape>()

  const fetchRows = async () => {
    setLoading(true)
    try {
      const params: ListConfigsParams = { page, page_size: pageSize }
      if (search.trim()) params.search = search.trim()
      if (autoloadFilter === 'true') params.autoload = true
      else if (autoloadFilter === 'false') params.autoload = false
      if (publicFilter === 'true') params.public = true
      else if (publicFilter === 'false') params.public = false
      const res = await listConfigs(params)
      const normalized = (res.configs || []).map((c) => ({
        ...c,
        value: c.value || (c as any).Value || '',
      }))
      setRows(normalized)
      setTotal(res.total || 0)
    } catch (e: any) {
      Message.error(`加载失败：${e?.msg || e?.message || e}`)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchRows()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, autoloadFilter, publicFilter])

  const handleSearch = () => {
    if (page !== 1) setPage(1)
    else fetchRows()
  }

  const toggleShow = (key: string) =>
    setShowValue((p) => ({ ...p, [key]: !p[key] }))

  const formatValue = (v: string, fmt: Config['format']) => {
    if (!v) return ''
    if (fmt === 'json') {
      try { return JSON.stringify(JSON.parse(v), null, 2) } catch { return v }
    }
    return v
  }

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({
      key: '',
      desc: '',
      value: '',
      format: 'text',
      autoload: false,
      public: false,
    })
    setModalOpen(true)
  }

  const openEdit = (row: Config) => {
    setEditing(row)
    form.setFieldsValue({
      key: row.key,
      desc: row.desc || '',
      value: row.value || (row as any).Value || '',
      format: row.format || 'text',
      autoload: !!row.autoload,
      public: !!row.public,
    })
    setModalOpen(true)
  }

  const save = async () => {
    try {
      const v = await form.validate()
      // JSON 格式校验
      if (v.format === 'json' && v.value && v.value.trim()) {
        try { JSON.parse(v.value) } catch { Message.error('value 不是合法 JSON'); return }
      }
      setSaving(true)
      if (editing) {
        await updateConfig(editing.key, {
          desc: v.desc,
          value: v.value,
          format: v.format,
          autoload: v.autoload,
          public: v.public,
        })
        Message.success('已保存')
      } else {
        await createConfig({
          key: v.key.trim(),
          desc: v.desc,
          value: v.value,
          format: v.format,
          autoload: v.autoload,
          public: v.public,
        })
        Message.success('已创建')
      }
      setModalOpen(false)
      fetchRows()
    } catch (e: any) {
      if (e?.errorFields) return // form validation
      Message.error(`保存失败：${e?.msg || e?.message || e}`)
    } finally {
      setSaving(false)
    }
  }

  const remove = async (row: Config) => {
    try {
      await deleteConfig(row.key)
      Message.success('已删除')
      fetchRows()
    } catch (e: any) {
      Message.error(`删除失败：${e?.msg || e?.message || e}`)
    }
  }

  // 当前 Modal 中 key 是否为 STORAGE_KIND，用以切换 value 的输入控件
  const currentKey = Form.useWatch?.('key', form) as string | undefined
  const isStorageKindKey = (currentKey || '').toUpperCase() === 'STORAGE_KIND'

  const columns = useMemo(
    () => [
      {
        title: '键',
        dataIndex: 'key',
        width: 220,
        render: (v: string) => (
          <code className="rounded bg-[var(--color-fill-2)] px-2 py-0.5 text-xs font-mono">
            {v}
          </code>
        ),
      },
      {
        title: '描述',
        dataIndex: 'desc',
        width: 200,
        ellipsis: true,
        render: (v: string) => v || <span className="text-[var(--color-text-3)]">-</span>,
      },
      {
        title: '值',
        dataIndex: 'value',
        ellipsis: true,
        render: (_: any, r: Config) => {
          const v = r.value || (r as any).Value || ''
          if (!r.public) {
            return <span className="text-[var(--color-text-3)] text-xs">-（非公开）</span>
          }
          const shown = !!showValue[r.key]
          return (
            <div className="flex items-center gap-2">
              <code className="flex-1 break-words rounded bg-[var(--color-fill-2)] px-2 py-0.5 text-xs font-mono">
                {shown ? formatValue(v, r.format) : '••••••••'}
              </code>
              <Button
                size="mini"
                type="text"
                onClick={() => toggleShow(r.key)}
                icon={shown ? <EyeOff size={14} /> : <Eye size={14} />}
              />
            </div>
          )
        },
      },
      {
        title: '格式',
        dataIndex: 'format',
        width: 100,
        render: (v: Config['format']) => {
          const f = FORMATS.find((x) => x.value === v)
          return <Tag color={f?.color || 'gray'}>{f?.label || v || 'text'}</Tag>
        },
      },
      {
        title: '自动加载',
        dataIndex: 'autoload',
        width: 100,
        align: 'center' as const,
        render: (v: boolean) =>
          v ? (
            <CheckCircle2 size={16} className="mx-auto text-green-500" />
          ) : (
            <XCircle size={16} className="mx-auto text-[var(--color-text-3)]" />
          ),
      },
      {
        title: '公开',
        dataIndex: 'public',
        width: 80,
        align: 'center' as const,
        render: (v: boolean) =>
          v ? (
            <CheckCircle2 size={16} className="mx-auto text-green-500" />
          ) : (
            <XCircle size={16} className="mx-auto text-[var(--color-text-3)]" />
          ),
      },
      {
        title: '操作',
        key: '__actions__',
        width: 170,
        fixed: 'right' as const,
        render: (_: any, r: Config) => (
          <Space size={4}>
            <Button size="mini" type="text" onClick={() => openEdit(r)}>
              <span className="inline-flex items-center gap-1"><Edit size={14} />编辑</span>
            </Button>
            <Popconfirm
              title={`确认删除配置 ${r.key}？`}
              onOk={() => remove(r)}
              okText="删除"
              cancelText="取消"
            >
              <Button size="mini" type="text" status="danger">
                <span className="inline-flex items-center gap-1"><Trash2 size={14} />删除</span>
              </Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [showValue],
  )

  return (
    <div className="space-y-4">
      <PageHeader
        title="系统配置"
        description="维护键值型系统配置（key/value/format）。自动加载项会在启动时被服务读入；公开项允许通过 /api/configs/:key 公开读取。"
        actions={
          <Space>
            <Input
              prefix={<Search size={14} />}
              placeholder="搜索 key 或描述"
              value={search}
              onChange={(v) => setSearch(v)}
              onPressEnter={handleSearch}
              allowClear
              style={{ width: 240 }}
            />
            <Select
              value={autoloadFilter}
              onChange={(v) => setAutoloadFilter(String(v ?? ''))}
              placeholder="自动加载"
              style={{ width: 140 }}
            >
              {BOOL_OPTIONS.map((o) => (
                <Option key={`a-${o.value}`} value={o.value}>{`自动加载：${o.label}`}</Option>
              ))}
            </Select>
            <Select
              value={publicFilter}
              onChange={(v) => setPublicFilter(String(v ?? ''))}
              placeholder="公开"
              style={{ width: 120 }}
            >
              {BOOL_OPTIONS.map((o) => (
                <Option key={`p-${o.value}`} value={o.value}>{`公开：${o.label}`}</Option>
              ))}
            </Select>
            <Button type="primary" onClick={handleSearch}>搜索</Button>
            <Button onClick={fetchRows}>
              <span className="inline-flex items-center gap-1">
                <RefreshCw size={14} className={loading ? 'animate-spin' : ''} /> 刷新
              </span>
            </Button>
            <Button type="primary" status="success" onClick={openCreate}>
              <span className="inline-flex items-center gap-1"><Plus size={14} />新建配置</span>
            </Button>
          </Space>
        }
      />

      <Table
        rowKey="key"
        loading={loading}
        columns={columns as any}
        data={rows}
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

      <Modal
        title={editing ? `编辑配置：${editing.key}` : '新建配置'}
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
          <div className="grid grid-cols-2 gap-3">
            <FormItem
              label="配置键 key"
              field="key"
              rules={[{ required: true, message: '请输入配置键（建议大写下划线风格）' }]}
              disabled={!!editing}
              extra={editing ? '配置键创建后不可修改' : '建议大写下划线，如 STORAGE_KIND'}
            >
              <Input
                placeholder="KEY_SITE_NAME"
                onChange={(v) => form.setFieldsValue({ key: (v || '').toUpperCase() })}
              />
            </FormItem>
            <FormItem label="格式 format" field="format" rules={[{ required: true }]}>
              <Select>
                {FORMATS.map((f) => (
                  <Option key={f.value} value={f.value}>{f.label}</Option>
                ))}
              </Select>
            </FormItem>
          </div>

          <FormItem label="描述 desc" field="desc">
            <Input placeholder="配置项描述（可空）" />
          </FormItem>

          <FormItem
            label="值 value"
            field="value"
            extra={
              isStorageKindKey
                ? 'STORAGE_KIND 取值仅限以下下拉项'
                : '格式为 JSON 时会做合法性校验'
            }
          >
            {isStorageKindKey ? (
              <Select placeholder="选择存储类型">
                {STORAGE_TYPES.map((t) => (
                  <Option key={t.value} value={t.value}>{t.label}</Option>
                ))}
              </Select>
            ) : (
              <TextArea
                rows={6}
                placeholder="配置值；JSON / YAML 直接粘贴，bool 填 true/false"
              />
            )}
          </FormItem>

          <div className="grid grid-cols-2 gap-3">
            <FormItem label="自动加载 autoload" field="autoload" triggerPropName="checked" extra="勾选后启动时由服务读入">
              <Switch />
            </FormItem>
            <FormItem label="公开 public" field="public" triggerPropName="checked" extra="勾选后允许公开端点读取此 key 的值">
              <Switch />
            </FormItem>
          </div>
        </Form>
      </Modal>
    </div>
  )
}

export default Configs
