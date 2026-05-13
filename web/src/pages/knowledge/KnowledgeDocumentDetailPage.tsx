// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import { ArrowLeft, Loader2, Trash2, Upload } from 'lucide-react'
import { PageSEO } from '@/components/SEO/PageSEO'
import PageContainer from '@/components/Layout/PageContainer'
import Button from '@/components/UI/Button'
import Card from '@/components/UI/Card'
import Badge from '@/components/UI/Badge'
import MarkdownPreview from '@/components/UI/MarkdownPreview'
import { useToast } from '@/components/UI/ToastContainer'
import { useI18nStore } from '@/stores/i18nStore'
import {
  getKnowledgeDocument,
  getKnowledgeDocumentText,
  putKnowledgeDocumentText,
  deleteKnowledgeDocument,
  reuploadKnowledgeDocument,
  type KnowledgeDocumentRow,
} from '@/api/knowledge'

function statusVariant(s: string): 'success' | 'warning' | 'error' | 'muted' | 'default' {
  const v = (s || '').toLowerCase()
  if (v === 'active') return 'success'
  if (v === 'processing') return 'warning'
  if (v === 'failed') return 'error'
  if (v === 'deleted') return 'muted'
  return 'default'
}

const KnowledgeDocumentDetailPage: React.FC = () => {
  const { docId } = useParams<{ docId: string }>()
  const [searchParams] = useSearchParams()
  const nsId = searchParams.get('ns') || ''
  const { t } = useI18nStore()
  const { success: toastSuccess, error: toastError } = useToast()
  const uploadRef = useRef<HTMLInputElement>(null)

  const [doc, setDoc] = useState<KnowledgeDocumentRow | null>(null)
  const [loading, setLoading] = useState(true)
  const [editMode, setEditMode] = useState(false)
  const [md, setMd] = useState('')
  const [saving, setSaving] = useState(false)

  const backHref = nsId ? `/knowledge/ns/${nsId}` : '/knowledge'

  const loadAll = useCallback(async () => {
    if (!docId) return
    setLoading(true)
    try {
      const dRes = await getKnowledgeDocument(docId)
      if (dRes.code !== 200 || !dRes.data?.document) {
        toastError(t('knowledge.docs'), dRes.msg || 'not found')
        setDoc(null)
        return
      }
      const row = dRes.data.document
      setDoc(row)

      const tRes = await getKnowledgeDocumentText(docId)
      const fromApi = tRes.code === 200 ? (tRes.data?.markdown || '').trim() : ''
      const fromRow = (row.storedMarkdown || '').trim()
      setMd(fromApi || fromRow)
    } catch (e: unknown) {
      toastError(t('knowledge.docs'), (e as { msg?: string })?.msg || String(e))
      setDoc(null)
    } finally {
      setLoading(false)
    }
  }, [docId, toastError, t])

  useEffect(() => {
    void loadAll()
  }, [loadAll])

  const onSave = async () => {
    if (!docId) return
    setSaving(true)
    try {
      const res = await putKnowledgeDocumentText(docId, md)
      if (res.code !== 200) {
        toastError(t('knowledge.saveMarkdown'), res.msg || 'failed')
        return
      }
      toastSuccess(t('knowledge.saveMarkdown'), res.msg || 'ok')
      setEditMode(false)
      void loadAll()
    } catch (e: unknown) {
      toastError(t('knowledge.saveMarkdown'), (e as { msg?: string })?.msg || String(e))
    } finally {
      setSaving(false)
    }
  }

  const onDelete = async () => {
    if (!doc || !window.confirm(`${t('knowledge.deleteDoc')}?`)) return
    const res = await deleteKnowledgeDocument(doc.id)
    if (res.code !== 200) {
      toastError(t('knowledge.deleteDoc'), res.msg || 'failed')
      return
    }
    toastSuccess(t('knowledge.deleteDoc'), res.msg || 'ok')
    window.location.href = backHref
  }

  const onReupload = async (f: File) => {
    if (!doc) return
    const res = await reuploadKnowledgeDocument(doc.id, f)
    if (res.code !== 200) {
      toastError(t('knowledge.reupload'), res.msg || 'failed')
      return
    }
    toastSuccess(t('knowledge.reupload'), res.msg || 'ok')
    void loadAll()
  }

  const site = t('brand.name')

  if (loading) {
    return (
      <PageContainer maxWidth="full" padding="md" className="flex justify-center py-24">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </PageContainer>
    )
  }

  if (!doc) {
    return (
      <PageContainer maxWidth="md" padding="md" className="py-16 text-center text-sm text-muted-foreground">
        {t('knowledge.notFound')}
      </PageContainer>
    )
  }

  return (
    <>
      <PageSEO title={`${doc.title} · ${t('knowledge.pageTitle')} · ${site}`} description={doc.namespace} />
      <PageContainer maxWidth="full" padding="md" className="pb-16">
        <div className="mb-6 flex flex-wrap items-center gap-3">
          <Link to={backHref} className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
            <ArrowLeft className="h-4 w-4" />
            {t('knowledge.back')}
          </Link>
        </div>

        <div className="mb-6 flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="min-w-0">
            <h1 className="text-2xl font-bold tracking-tight">{doc.title}</h1>
            <p className="mt-1 font-mono text-xs text-muted-foreground">{doc.namespace}</p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={statusVariant(doc.status)}>{doc.status}</Badge>
            <input
              ref={uploadRef}
              type="file"
              className="hidden"
              onChange={(e) => {
                const f = e.target.files?.[0]
                e.target.value = ''
                if (f) void onReupload(f)
              }}
            />
            <Button variant="outline" size="sm" type="button" onClick={() => uploadRef.current?.click()} leftIcon={<Upload className="h-4 w-4" />}>
              {t('knowledge.reupload')}
            </Button>
            <Button variant="outline" size="sm" type="button" onClick={() => setEditMode((e) => !e)}>
              {t('knowledge.editMarkdown')}
            </Button>
            <Button variant="destructive" size="sm" type="button" onClick={() => void onDelete()} leftIcon={<Trash2 className="h-4 w-4" />}>
              {t('knowledge.deleteDoc')}
            </Button>
          </div>
        </div>

        <Card className="border-border/80 p-4 md:p-6">
          {editMode ? (
            <div className="space-y-3">
              <textarea
                value={md}
                onChange={(e) => setMd(e.target.value)}
                rows={24}
                className="w-full rounded-lg border border-input bg-background px-3 py-2 font-mono text-sm"
              />
              <div className="flex gap-2">
                <Button variant="primary" size="sm" loading={saving} onClick={() => void onSave()}>
                  {t('knowledge.saveMarkdown')}
                </Button>
                <Button variant="ghost" size="sm" disabled={saving} onClick={() => { setEditMode(false); void loadAll() }}>
                  {t('knowledge.cancel')}
                </Button>
              </div>
            </div>
          ) : md ? (
            <div className="prose prose-sm max-w-none dark:prose-invert">
              <MarkdownPreview content={md} />
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">{t('knowledge.docTextEmpty')}</p>
          )}
        </Card>
      </PageContainer>
    </>
  )
}

export default KnowledgeDocumentDetailPage
