// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import { useCallback, useEffect, useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import { Button, Card, Message, Popconfirm, Space, Typography, Input } from '@arco-design/web-react'
import { ArrowLeft } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import {
  getKnowledgeDocument,
  getKnowledgeDocumentText,
  putKnowledgeDocumentText,
  deleteKnowledgeDocument,
} from '@/services/adminApi'
import type { KnowledgeDocumentRow } from '@/services/adminApi'

const KnowledgeDocumentDetailPage = () => {
  const { docId } = useParams<{ docId: string }>()
  const [searchParams] = useSearchParams()
  const nsId = searchParams.get('ns') || ''
  const backHref = nsId ? `/knowledge-bases/${nsId}` : '/knowledge-bases'

  const [doc, setDoc] = useState<KnowledgeDocumentRow | null>(null)
  const [loading, setLoading] = useState(true)
  const [md, setMd] = useState('')
  const [saving, setSaving] = useState(false)

  const loadAll = useCallback(async () => {
    if (!docId) return
    setLoading(true)
    try {
      const { document: row } = await getKnowledgeDocument(docId)
      setDoc(row)
      const t = await getKnowledgeDocumentText(docId)
      const fromApi = (t.markdown || '').trim()
      const fromRow = (row.storedMarkdown || '').trim()
      setMd(fromApi || fromRow)
    } catch (e: unknown) {
      Message.error(e instanceof Error ? e.message : String(e))
      setDoc(null)
    } finally {
      setLoading(false)
    }
  }, [docId])

  useEffect(() => {
    void loadAll()
  }, [loadAll])

  const save = async () => {
    if (!docId) return
    setSaving(true)
    try {
      await putKnowledgeDocumentText(docId, md)
      Message.success('已提交后台向量化')
      void loadAll()
    } catch (e: unknown) {
      Message.error(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  const onDelete = async () => {
    if (!docId) return
    try {
      await deleteKnowledgeDocument(docId)
      Message.success('已删除')
      window.location.href = backHref
    } catch (e: unknown) {
      Message.error(e instanceof Error ? e.message : String(e))
    }
  }

  if (loading) {
    return <Typography.Text type="secondary">加载中…</Typography.Text>
  }

  if (!doc) {
    return (
      <div className="space-y-2">
        <Link to={backHref} className="text-sm text-[rgb(var(--primary-6))]">
          ← 返回
        </Link>
        <Typography.Text type="secondary">未找到文档</Typography.Text>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <Link to={backHref} className="inline-flex items-center gap-1 text-sm text-[var(--color-text-2)] hover:text-[rgb(var(--primary-6))]">
        <ArrowLeft size={16} /> 返回知识库
      </Link>

      <PageHeader
        title={doc.title}
        description={
          <span>
            <span className="font-mono text-xs">{doc.namespace}</span>
            <Typography.Text type="secondary" className="ml-2">
              {doc.status}
            </Typography.Text>
          </span>
        }
        actions={
          <Space>
            <Button type="primary" loading={saving} onClick={() => void save()}>
              保存并重新向量化
            </Button>
            <Popconfirm title="删除文档并清理向量点？" onOk={() => void onDelete()}>
              <Button status="danger">删除</Button>
            </Popconfirm>
          </Space>
        }
      />

      <Card title="Markdown 正文">
        {!md && <Typography.Text type="secondary">暂无正文（若刚上传请等待处理完成）。</Typography.Text>}
        <Input.TextArea value={md} onChange={setMd} autoSize={{ minRows: 20, maxRows: 40 }} placeholder="正文" />
      </Card>
    </div>
  )
}

export default KnowledgeDocumentDetailPage
