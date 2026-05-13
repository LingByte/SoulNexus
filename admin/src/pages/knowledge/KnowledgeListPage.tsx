// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import {
  Table,
  Button,
  Drawer,
  Form,
  Input,
  Select,
  Message,
  Space,
  Tag,
} from '@arco-design/web-react'
import { RefreshCw, Plus } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import { useMediaQuery } from '@/hooks/useMediaQuery'
import {
  listKnowledgeNamespaces,
  listMyGroupsForKnowledge,
  createKnowledgeNamespace,
  type KnowledgeNamespaceRow,
  type MyGroupRow,
} from '@/services/adminApi'

const FormItem = Form.Item

const statusTag = (s: string) => {
  const v = (s || '').toLowerCase()
  if (v === 'active') return <Tag color="green">active</Tag>
  if (v === 'processing') return <Tag color="arcoblue">processing</Tag>
  if (v === 'failed') return <Tag color="red">failed</Tag>
  if (v === 'deleted') return <Tag color="gray">deleted</Tag>
  return <Tag>{s || '-'}</Tag>
}

const KnowledgeListPage = () => {
  const navigate = useNavigate()
  const isSmallScreen = useMediaQuery('(max-width: 639px)')
  const [loading, setLoading] = useState(false)
  const [rows, setRows] = useState<KnowledgeNamespaceRow[]>([])
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [statusFilter, setStatusFilter] = useState('active')
  const [qInput, setQInput] = useState('')
  const [q, setQ] = useState('')

  const [groups, setGroups] = useState<MyGroupRow[]>([])
  const [createOpen, setCreateOpen] = useState(false)
  const [createSubmitting, setCreateSubmitting] = useState(false)
  const [form] = Form.useForm()

  const loadNamespaces = useCallback(async () => {
    setLoading(true)
    try {
      const out = await listKnowledgeNamespaces({
        page,
        pageSize,
        status: statusFilter === 'all' ? 'all' : statusFilter,
        q: q.trim() || undefined,
      })
      setRows(out.list || [])
      setTotal(out.total || 0)
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      Message.error(`加载知识库失败：${msg}`)
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, statusFilter, q])

  useEffect(() => {
    void loadNamespaces()
  }, [loadNamespaces])

  const openCreate = async () => {
    try {
      const g = await listMyGroupsForKnowledge()
      setGroups(g)
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      Message.error(`加载组织失败：${msg}`)
      setGroups([])
    }
    form.resetFields()
    form.setFieldsValue({
      vectorProvider: 'qdrant',
      embedModel: '',
    })
    setCreateOpen(true)
  }

  const submitCreate = async () => {
    try {
      const v = await form.validate()
      setCreateSubmitting(true)
      const row = await createKnowledgeNamespace({
        namespace: String(v.namespace || '').trim(),
        name: String(v.name || '').trim(),
        description: v.description ? String(v.description) : undefined,
        vectorProvider: v.vectorProvider || 'qdrant',
        embedModel: String(v.embedModel || '').trim(),
        groupId: v.groupId ? Number(v.groupId) : undefined,
      })
      Message.success('创建成功')
      setCreateOpen(false)
      void loadNamespaces()
      navigate(`/knowledge-bases/${row.id}`)
    } catch (e: unknown) {
      if ((e as { errors?: unknown })?.errors) return
      const msg = e instanceof Error ? e.message : String(e)
      Message.error(msg)
    } finally {
      setCreateSubmitting(false)
    }
  }

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 100 },
    { title: '组织', dataIndex: 'groupId', width: 80 },
    {
      title: 'Collection',
      dataIndex: 'namespace',
      ellipsis: true,
      render: (_: unknown, r: KnowledgeNamespaceRow) => (
        <span className="font-mono text-xs">{r.namespace}</span>
      ),
    },
    {
      title: '名称',
      dataIndex: 'name',
      ellipsis: true,
      render: (_: unknown, r: KnowledgeNamespaceRow) => (
        <Link to={`/knowledge-bases/${r.id}`} className="text-[rgb(var(--primary-6))] hover:underline">
          {r.name}
        </Link>
      ),
    },
    {
      title: '后端',
      dataIndex: 'vectorProvider',
      width: 90,
      render: (v: string) => <Tag>{v || 'qdrant'}</Tag>,
    },
    { title: '维度', dataIndex: 'vectorDim', width: 72 },
    { title: '嵌入模型', dataIndex: 'embedModel', width: 120, ellipsis: true },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (s: string) => statusTag(s),
    },
    {
      title: '操作',
      key: 'a',
      width: 100,
      fixed: 'right' as const,
      render: (_: unknown, r: KnowledgeNamespaceRow) => (
        <Link to={`/knowledge-bases/${r.id}`}>
          <Button size="mini" type="outline">
            详情
          </Button>
        </Link>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <PageHeader
        title="知识库"
        description="命名空间与文档；点名称进入详情页。"
        actions={
          <div className="flex w-full max-w-full flex-col gap-2 sm:w-auto sm:flex-row sm:flex-wrap sm:items-center">
            <Input.Search
              allowClear
              placeholder="名称 / Collection"
              className="w-full min-w-0 sm:w-[220px]"
              value={qInput}
              onChange={setQInput}
              onSearch={(v) => {
                setPage(1)
                setQ(String(v || '').trim())
              }}
            />
            <Select
              placeholder="状态"
              style={{ width: 120 }}
              value={statusFilter}
              onChange={(v) => {
                setPage(1)
                setStatusFilter(String(v))
              }}
              options={[
                { label: 'active', value: 'active' },
                { label: 'processing', value: 'processing' },
                { label: 'failed', value: 'failed' },
                { label: 'deleted', value: 'deleted' },
                { label: '全部', value: 'all' },
              ]}
            />
            <Button type="primary" onClick={() => void openCreate()}>
              <span className="inline-flex items-center gap-1">
                <Plus size={16} /> 新建
              </span>
            </Button>
            <Button onClick={() => void loadNamespaces()}>
              <span className="inline-flex items-center gap-1">
                <RefreshCw size={14} className={loading ? 'animate-spin' : ''} /> 刷新
              </span>
            </Button>
          </div>
        }
      />

      <Table
        rowKey={(r) => String(r.id)}
        loading={loading}
        columns={columns}
        data={rows}
        scroll={{ x: 960 }}
        pagination={{
          current: page,
          pageSize,
          total,
          showTotal: (t: number) => `共 ${t} 条`,
          sizeCanChange: true,
          pageSizeChangeResetCurrent: true,
          onChange: (p: number, ps: number) => {
            setPage(p)
            setPageSize(ps)
          },
        }}
      />

      <Drawer
        title="新建知识库"
        visible={createOpen}
        placement="right"
        width={isSmallScreen ? '100%' : 480}
        onCancel={() => setCreateOpen(false)}
        footer={
          <Space>
            <Button onClick={() => setCreateOpen(false)}>取消</Button>
            <Button type="primary" loading={createSubmitting} onClick={() => void submitCreate()}>
              创建
            </Button>
          </Space>
        }
      >
        <Form form={form} layout="vertical">
          <FormItem label="所属组织" field="groupId" extra="不选则使用个人组织">
            <Select allowClear placeholder="默认个人组织" options={groups.map((g) => ({ label: `${g.name} (#${g.id})`, value: g.id }))} />
          </FormItem>
          <FormItem label="Collection 名（namespace）" field="namespace" rules={[{ required: true, message: '必填' }]}>
            <Input placeholder="英文标识，如 product_kb_v1" />
          </FormItem>
          <FormItem label="显示名称" field="name" rules={[{ required: true, message: '必填' }]}>
            <Input placeholder="对内展示名称" />
          </FormItem>
          <FormItem label="描述" field="description">
            <Input.TextArea placeholder="可选" autoSize={{ minRows: 2, maxRows: 4 }} />
          </FormItem>
          <FormItem label="向量后端" field="vectorProvider" initialValue="qdrant">
            <Select options={[{ label: 'Qdrant', value: 'qdrant' }, { label: 'Milvus', value: 'milvus' }]} />
          </FormItem>
          <FormItem label="嵌入模型名" field="embedModel" rules={[{ required: true, message: '必填' }]} extra="与 EMBED_MODEL 对齐">
            <Input placeholder="例如 bge-m3" />
          </FormItem>
        </Form>
      </Drawer>
    </div>
  )
}

export default KnowledgeListPage
