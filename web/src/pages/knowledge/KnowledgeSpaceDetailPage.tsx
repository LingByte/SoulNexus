// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ArrowLeft, FlaskConical, Loader2, Trash2, Upload } from 'lucide-react'
import { PageSEO } from '@/components/SEO/PageSEO'
import PageContainer from '@/components/Layout/PageContainer'
import Button from '@/components/UI/Button'
import Card from '@/components/UI/Card'
import Badge from '@/components/UI/Badge'
import Input from '@/components/UI/Input'
import { useToast } from '@/components/UI/ToastContainer'
import { useI18nStore } from '@/stores/i18nStore'
import {
  getKnowledgeNamespace,
  listKnowledgeDocuments,
  deleteKnowledgeNamespace,
  uploadKnowledgeToNamespace,
  runKnowledgeRecallTest,
  type KnowledgeNamespaceRow,
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

const KnowledgeSpaceDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>()
  const { t } = useI18nStore()
  const { success: toastSuccess, error: toastError } = useToast()
  const uploadRef = useRef<HTMLInputElement>(null)

  const [ns, setNs] = useState<KnowledgeNamespaceRow | null>(null)
  const [loadNs, setLoadNs] = useState(true)
  const [docs, setDocs] = useState<KnowledgeDocumentRow[]>([])
  const [docsLoading, setDocsLoading] = useState(false)
  const [docQInput, setDocQInput] = useState('')
  const [docQ, setDocQ] = useState('')
  const [tab, setTab] = useState<'docs' | 'recall'>('docs')
  const [recallQ, setRecallQ] = useState('')
  const [recallTopK, setRecallTopK] = useState(5)
  const [recallMin, setRecallMin] = useState(0)
  const [recallBusy, setRecallBusy] = useState(false)
  const [recallPayload, setRecallPayload] = useState<Record<string, unknown> | null>(null)

  const loadNsRow = useCallback(async () => {
    if (!id) return
    setLoadNs(true)
    try {
      const res = await getKnowledgeNamespace(id)
      if (res.code !== 200) {
        toastError(t('knowledge.pageTitle'), res.msg || 'not found')
        setNs(null)
        return
      }
      const row = (res.data as { namespace?: KnowledgeNamespaceRow })?.namespace
      setNs(row || null)
    } catch (e: unknown) {
      toastError(t('knowledge.pageTitle'), (e as { msg?: string })?.msg || String(e))
      setNs(null)
    } finally {
      setLoadNs(false)
    }
  }, [id, toastError, t])

  const loadDocs = useCallback(async () => {
    if (!ns) return
    setDocsLoading(true)
    try {
      const res = await listKnowledgeDocuments({
        namespace: ns.namespace,
        page: 1,
        pageSize: 200,
        status: 'all',
        q: docQ.trim() || undefined,
      })
      if (res.code !== 200) {
        toastError(ns.name, res.msg || 'failed')
        return
      }
      setDocs(res.data?.list || [])
    } catch (e: unknown) {
      toastError(ns.name, (e as { msg?: string })?.msg || String(e))
    } finally {
      setDocsLoading(false)
    }
  }, [ns, docQ, toastError])

  useEffect(() => {
    void loadNsRow()
  }, [loadNsRow])

  useEffect(() => {
    if (ns) void loadDocs()
  }, [ns, loadDocs])

  const onUpload = async (f: File) => {
    if (!ns) return
    const res = await uploadKnowledgeToNamespace(ns.id, f)
    if (res.code !== 200) {
      toastError(t('knowledge.upload'), res.msg || 'failed')
      return
    }
    toastSuccess(t('knowledge.upload'), res.msg || 'ok')
    void loadDocs()
    void loadNsRow()
  }

  const onDeleteNs = async () => {
    if (!ns || !window.confirm(t('knowledge.deleteBaseConfirm'))) return
    const res = await deleteKnowledgeNamespace(ns.id)
    if (res.code !== 200) {
      toastError(t('knowledge.deleteBase'), res.msg || 'failed')
      return
    }
    toastSuccess(t('knowledge.deleteBase'), res.msg || 'ok')
    window.location.href = '/knowledge'
  }

  const runRecall = async () => {
    if (!ns || !recallQ.trim()) return
    setRecallBusy(true)
    setRecallPayload(null)
    try {
      const res = await runKnowledgeRecallTest(ns.id, { query: recallQ.trim(), topK: recallTopK, minScore: recallMin })
      if (res.code !== 200) {
        toastError(t('knowledge.runRecall'), res.msg || 'failed')
        return
      }
      setRecallPayload((res.data as Record<string, unknown>) || {})
    } catch (e: unknown) {
      toastError(t('knowledge.runRecall'), (e as { msg?: string })?.msg || String(e))
    } finally {
      setRecallBusy(false)
    }
  }

  const recallHits = Array.isArray(recallPayload?.results)
    ? (recallPayload!.results as Array<{ record?: { title?: string; content?: string; id?: string }; score?: number }>)
    : []

  const site = t('brand.name')

  if (loadNs) {
    return (
      <PageContainer maxWidth="full" padding="md" className="flex justify-center py-24">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </PageContainer>
    )
  }

  if (!ns) {
    return (
      <PageContainer maxWidth="md" padding="md" className="py-16 text-center text-sm text-muted-foreground">
        {t('knowledge.notFound')}
      </PageContainer>
    )
  }

  return (
    <>
      <PageSEO title={`${ns.name} · ${t('knowledge.pageTitle')} · ${site}`} description={ns.namespace} />
      <PageContainer maxWidth="full" padding="md" className="pb-16">
        <div className="mb-6 flex flex-wrap items-center gap-3">
          <Link to="/knowledge" className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
            <ArrowLeft className="h-4 w-4" />
            {t('knowledge.back')}
          </Link>
        </div>

        <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0">
            <h1 className="text-2xl font-bold tracking-tight">{ns.name}</h1>
            <p className="mt-1 font-mono text-sm text-muted-foreground">{ns.namespace}</p>
            {ns.description ? <p className="mt-2 max-w-prose text-sm text-muted-foreground">{ns.description}</p> : null}
          </div>
          <div className="flex flex-wrap gap-2">
            <input
              ref={uploadRef}
              type="file"
              className="hidden"
              onChange={(e) => {
                const f = e.target.files?.[0]
                e.target.value = ''
                if (f) void onUpload(f)
              }}
            />
            <Button variant="primary" size="sm" type="button" onClick={() => uploadRef.current?.click()} leftIcon={<Upload className="h-4 w-4" />}>
              {t('knowledge.upload')}
            </Button>
            <Button variant="destructive" size="sm" type="button" onClick={() => void onDeleteNs()} leftIcon={<Trash2 className="h-4 w-4" />}>
              {t('knowledge.deleteBase')}
            </Button>
          </div>
        </div>

        <div className="mb-6 grid gap-3 sm:grid-cols-4">
          <Card className="border-border/60 p-3 text-sm">
            <div className="text-muted-foreground">{t('knowledge.dimension')}</div>
            <div className="mt-1 font-semibold">{ns.vectorDim}</div>
          </Card>
          <Card className="border-border/60 p-3 text-sm">
            <div className="text-muted-foreground">{t('knowledge.embedModel')}</div>
            <div className="mt-1 truncate">{ns.embedModel}</div>
          </Card>
          <Card className="border-border/60 p-3 text-sm">
            <div className="text-muted-foreground">Backend</div>
            <div className="mt-1 font-mono text-xs">{ns.vectorProvider}</div>
          </Card>
          <Card className="border-border/60 p-3 text-sm">
            <div className="text-muted-foreground">{t('knowledge.status')}</div>
            <div className="mt-1">
              <Badge variant={statusVariant(ns.status)}>{ns.status}</Badge>
            </div>
          </Card>
        </div>

        <div className="mb-4 flex gap-1 rounded-lg border border-border bg-muted/30 p-1">
          <button
            type="button"
            onClick={() => setTab('docs')}
            className={`flex-1 rounded-md px-3 py-2 text-sm font-medium ${tab === 'docs' ? 'bg-background shadow-sm' : 'text-muted-foreground'}`}
          >
            {t('knowledge.docs')}
          </button>
          <button
            type="button"
            onClick={() => setTab('recall')}
            className={`flex-1 rounded-md px-3 py-2 text-sm font-medium ${tab === 'recall' ? 'bg-background shadow-sm' : 'text-muted-foreground'}`}
          >
            <span className="inline-flex items-center justify-center gap-1.5">
              <FlaskConical className="h-4 w-4" />
              {t('knowledge.recall')}
            </span>
          </button>
        </div>

        {tab === 'docs' && (
          <Card className="overflow-hidden border-border/80">
            <div className="flex flex-wrap gap-2 border-b border-border p-3">
              <Input
                value={docQInput}
                onChange={(e) => setDocQInput(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && setDocQ(docQInput.trim())}
                placeholder={t('knowledge.docSearchPlaceholder')}
                className="max-w-md flex-1"
              />
              <Button variant="secondary" size="sm" type="button" onClick={() => setDocQ(docQInput.trim())}>
                {t('knowledge.search')}
              </Button>
            </div>
            {docsLoading ? (
              <div className="flex justify-center py-16">
                <Loader2 className="h-7 w-7 animate-spin text-primary" />
              </div>
            ) : (
              <ul className="divide-y divide-border">
                {docs.map((d) => (
                  <li key={d.id}>
                    <Link
                      to={`/knowledge/documents/${d.id}?ns=${encodeURIComponent(ns.id)}`}
                      className="flex items-center justify-between gap-3 px-4 py-3 hover:bg-muted/30"
                    >
                      <div className="min-w-0">
                        <div className="truncate font-medium">{d.title}</div>
                        <div className="mt-0.5 text-xs text-muted-foreground">
                          {(d.fileHash?.length ?? 0) > 12 ? `${d.fileHash.slice(0, 12)}…` : d.fileHash}
                        </div>
                      </div>
                      <Badge variant={statusVariant(d.status)}>{d.status}</Badge>
                    </Link>
                  </li>
                ))}
              </ul>
            )}
          </Card>
        )}

        {tab === 'recall' && (
          <Card className="space-y-4 border-border/80 p-4 md:p-6">
            <textarea
              value={recallQ}
              onChange={(e) => setRecallQ(e.target.value)}
              rows={3}
              className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm"
              placeholder={t('knowledge.query')}
            />
            <div className="flex flex-wrap gap-4">
              <div>
                <label className="mb-1 block text-xs text-muted-foreground">{t('knowledge.topK')}</label>
                <Input type="number" value={String(recallTopK)} onChange={(e) => setRecallTopK(parseInt(e.target.value, 10) || 5)} className="w-24" />
              </div>
              <div>
                <label className="mb-1 block text-xs text-muted-foreground">{t('knowledge.minScore')}</label>
                <Input type="number" step={0.05} value={String(recallMin)} onChange={(e) => setRecallMin(parseFloat(e.target.value) || 0)} className="w-24" />
              </div>
            </div>
            <Button variant="primary" size="sm" onClick={() => void runRecall()} loading={recallBusy}>
              {t('knowledge.runRecall')}
            </Button>
            {recallHits.length > 0 && (
              <ul className="space-y-2">
                {recallHits.map((hit, i) => (
                  <li key={hit.record?.id || i} className="rounded-lg border border-border/60 bg-muted/10 p-3 text-sm">
                    <div className="mb-1 flex justify-between gap-2 font-medium">
                      <span className="truncate">{hit.record?.title}</span>
                      <span className="shrink-0 font-mono text-xs text-primary">{typeof hit.score === 'number' ? hit.score.toFixed(4) : ''}</span>
                    </div>
                    <p className="line-clamp-3 text-muted-foreground">{hit.record?.content}</p>
                  </li>
                ))}
              </ul>
            )}
          </Card>
        )}
      </PageContainer>
    </>
  )
}

export default KnowledgeSpaceDetailPage
