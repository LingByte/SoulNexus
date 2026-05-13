// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  Table,
  Button,
  Message,
  Space,
  Tag,
  Popconfirm,
  Input,
  InputNumber,
  Card,
  Typography,
} from '@arco-design/web-react'
import { ArrowLeft, Upload, FlaskConical, Trash2 } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import {
  getKnowledgeNamespace,
  listKnowledgeDocuments,
  deleteKnowledgeNamespace,
  uploadKnowledgeToNamespace,
  reuploadKnowledgeDocument,
  deleteKnowledgeDocument,
  runKnowledgeRecallTest,
  type KnowledgeNamespaceRow,
  type KnowledgeDocumentRow,
} from '@/services/adminApi'

const statusTag = (s: string) => {
  const v = (s || '').toLowerCase()
  if (v === 'active') return <Tag color="green">active</Tag>
  if (v === 'processing') return <Tag color="arcoblue">processing</Tag>
  if (v === 'failed') return <Tag color="red">failed</Tag>
  if (v === 'deleted') return <Tag color="gray">deleted</Tag>
  return <Tag>{s || '-'}</Tag>
}

const KnowledgeSpaceDetailPage = () => {
  const { id } = useParams<{ id: string }>()
  const [ns, setNs] = useState<KnowledgeNamespaceRow | null>(null)
  const [loadErr, setLoadErr] = useState<string | null>(null)

  const [docsLoading, setDocsLoading] = useState(false)
  const [docs, setDocs] = useState<KnowledgeDocumentRow[]>([])
  const [docQ, setDocQ] = useState('')

  const [recallQuery, setRecallQuery] = useState('')
  const [recallTopK, setRecallTopK] = useState(5)
  const [recallMin, setRecallMin] = useState(0)
  const [recallResult, setRecallResult] = useState<Record<string, unknown> | null>(null)
  const [recallLoading, setRecallLoading] = useState(false)

  const uploadInputRef = useRef<HTMLInputElement>(null)
  const reuploadInputRef = useRef<HTMLInputElement>(null)
  const reuploadDocIdRef = useRef<string | null>(null)

  const loadNs = useCallback(async () => {
    if (!id) return
    setLoadErr(null)
    try {
      const row = await getKnowledgeNamespace(id)
      setNs(row)
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      setLoadErr(msg)
      setNs(null)
    }
  }, [id])

  const loadDocs = useCallback(async () => {
    if (!ns) return
    setDocsLoading(true)
    try {
      const out = await listKnowledgeDocuments({
        namespace: ns.namespace,
        page: 1,
        pageSize: 200,
        status: 'all',
        q: docQ.trim() || undefined,
      })
      setDocs(out.list || [])
    } catch (e: unknown) {
      Message.error(e instanceof Error ? e.message : String(e))
    } finally {
      setDocsLoading(false)
    }
  }, [ns, docQ])

  useEffect(() => {
    void loadNs()
  }, [loadNs])

  useEffect(() => {
    if (ns) void loadDocs()
  }, [ns, loadDocs])

  const onUpload = async (file: File) => {
    if (!ns) return
    try {
      await uploadKnowledgeToNamespace(ns.id, file)
      Message.success('已提交后台处理')
      void loadDocs()
      void loadNs()
    } catch (e: unknown) {
      Message.error(e instanceof Error ? e.message : String(e))
    }
  }

  const onDeleteNs = async () => {
    if (!ns) return
    try {
      await deleteKnowledgeNamespace(ns.id)
      Message.success('已删除')
      window.location.href = '/knowledge-bases'
    } catch (e: unknown) {
      Message.error(e instanceof Error ? e.message : String(e))
    }
  }

  if (loadErr || !ns) {
    return (
      <div className="space-y-4">
        <Link to="/knowledge-bases" className="inline-flex items-center gap-1 text-sm text-[var(--color-text-2)] hover:text-[rgb(var(--primary-6))]">
          <ArrowLeft size={16} /> 返回列表
        </Link>
        <Typography.Text type="secondary">{loadErr || '未找到'}</Typography.Text>
      </div>
    )
  }

  const docColumns = [
    { title: 'ID', dataIndex: 'id', width: 100 },
    {
      title: '标题',
      dataIndex: 'title',
      ellipsis: true,
      render: (_: unknown, d: KnowledgeDocumentRow) => (
        <Link to={`/knowledge-bases/documents/${d.id}?ns=${encodeURIComponent(ns.id)}`} className="text-[rgb(var(--primary-6))] hover:underline">
          {d.title}
        </Link>
      ),
    },
    { title: '状态', dataIndex: 'status', width: 100, render: (s: string) => statusTag(s) },
    {
      title: '操作',
      key: 'da',
      width: 200,
      render: (_: unknown, d: KnowledgeDocumentRow) => (
        <Space size="mini" wrap>
          <Button
            size="mini"
            type="outline"
            onClick={() => {
              reuploadDocIdRef.current = String(d.id)
              reuploadInputRef.current?.click()
            }}
          >
            <span className="inline-flex items-center gap-1">
              <Upload size={12} /> 重传
            </span>
          </Button>
          <Link to={`/knowledge-bases/documents/${d.id}?ns=${encodeURIComponent(ns.id)}`}>
            <Button size="mini" type="outline">
              正文
            </Button>
          </Link>
          <Popconfirm
            title="删除文档并清理向量点？"
            onOk={async () => {
              try {
                await deleteKnowledgeDocument(d.id)
                Message.success('已删除')
                void loadDocs()
                void loadNs()
              } catch (e: unknown) {
                Message.error(e instanceof Error ? e.message : String(e))
              }
            }}
          >
            <Button size="mini" type="text" status="danger">
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <div>
        <Link to="/knowledge-bases" className="inline-flex items-center gap-1 text-sm text-[var(--color-text-2)] hover:text-[rgb(var(--primary-6))]">
          <ArrowLeft size={16} /> 返回列表
        </Link>
      </div>

      <PageHeader
        title={ns.name}
        description={
          <span className="font-mono text-xs">{ns.namespace}</span>
        }
        actions={
          <div className="flex w-full max-w-full flex-col gap-2 sm:w-auto sm:flex-row sm:flex-wrap sm:items-center">
            <input
              ref={uploadInputRef}
              type="file"
              className="hidden"
              onChange={(e) => {
                const f = e.target.files?.[0]
                e.target.value = ''
                if (f) void onUpload(f)
              }}
            />
            <Button type="primary" onClick={() => uploadInputRef.current?.click()}>
              <span className="inline-flex items-center gap-1">
                <Upload size={14} /> 上传文件
              </span>
            </Button>
            <Popconfirm title="将删除向量库 collection 并软删除记录，确认？" onOk={() => void onDeleteNs()}>
              <Button status="danger">
                <span className="inline-flex items-center gap-1">
                  <Trash2 size={14} /> 删除知识库
                </span>
              </Button>
            </Popconfirm>
          </div>
        }
      />

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Card size="small" title="维度">
          {ns.vectorDim}
        </Card>
        <Card size="small" title="嵌入模型">
          <span className="text-xs">{ns.embedModel}</span>
        </Card>
        <Card size="small" title="后端">
          <Tag>{ns.vectorProvider}</Tag>
        </Card>
        <Card size="small" title="状态">
          {statusTag(ns.status)}
        </Card>
      </div>

      <Card title="文档">
        <div className="mb-3 flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
          <Input.Search
            allowClear
            placeholder="标题 / hash"
            className="w-full min-w-0 sm:max-w-[280px]"
            value={docQ}
            onChange={setDocQ}
            onSearch={() => void loadDocs()}
          />
          <Button size="small" onClick={() => void loadDocs()}>
            筛选
          </Button>
        </div>
        <Table
          rowKey={(d) => String(d.id)}
          loading={docsLoading}
          columns={docColumns}
          data={docs}
          pagination={false}
          size="small"
          scroll={{ x: 'max-content' }}
        />
      </Card>

      <Card
        title={
          <span className="inline-flex items-center gap-2">
            <FlaskConical size={16} /> 召回测试
          </span>
        }
      >
        <Space direction="vertical" style={{ width: '100%' }} size="medium">
          <Input.TextArea
            placeholder="检索语句"
            value={recallQuery}
            onChange={setRecallQuery}
            autoSize={{ minRows: 2, maxRows: 6 }}
          />
          <Space wrap className="w-full sm:w-auto">
            <span className="text-sm text-[var(--color-text-2)]">topK</span>
            <InputNumber min={1} max={50} value={recallTopK} onChange={(v) => setRecallTopK(Number(v) || 5)} />
            <span className="text-sm text-[var(--color-text-2)]">minScore</span>
            <InputNumber min={0} max={1} step={0.05} value={recallMin} onChange={(v) => setRecallMin(Number(v) || 0)} />
            <Button
              type="primary"
              loading={recallLoading}
              onClick={async () => {
                if (!recallQuery.trim()) {
                  Message.warning('请输入 query')
                  return
                }
                setRecallLoading(true)
                setRecallResult(null)
                try {
                  const data = await runKnowledgeRecallTest(ns.id, {
                    query: recallQuery.trim(),
                    topK: recallTopK,
                    minScore: recallMin,
                  })
                  setRecallResult(data)
                } catch (e: unknown) {
                  Message.error(e instanceof Error ? e.message : String(e))
                } finally {
                  setRecallLoading(false)
                }
              }}
            >
              运行
            </Button>
          </Space>
          {recallResult && (
            <pre className="max-h-[360px] overflow-auto rounded border border-[var(--color-border-2)] bg-[var(--color-fill-2)] p-3 text-xs">
              {JSON.stringify(recallResult, null, 2)}
            </pre>
          )}
        </Space>
      </Card>

      <input
        ref={reuploadInputRef}
        type="file"
        className="hidden"
        onChange={async (ev) => {
          const docId = reuploadDocIdRef.current
          reuploadDocIdRef.current = null
          const f = ev.target.files?.[0]
          ev.target.value = ''
          if (!f || !docId) return
          try {
            await reuploadKnowledgeDocument(docId, f)
            Message.success('已提交后台处理')
            void loadDocs()
            void loadNs()
          } catch (e: unknown) {
            Message.error(e instanceof Error ? e.message : String(e))
          }
        }}
      />
    </div>
  )
}

export default KnowledgeSpaceDetailPage
