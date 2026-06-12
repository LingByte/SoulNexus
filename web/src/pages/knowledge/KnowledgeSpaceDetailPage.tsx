// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { BookOpen, ChevronRight, FlaskConical, Loader2, Trash2, Upload } from 'lucide-react'
import { PageSEO } from '@/components/SEO/PageSEO'
import PageHeader from '@/components/Layout/PageHeader'
import Button from '@/components/UI/Button'
import { Input as ArcoInput, Card as ArcoCard, Tabs as ArcoTabs } from '@arco-design/web-react'
import ConfirmDialog from '@/components/UI/ConfirmDialog'
import EmptyState from '@/components/UI/EmptyState'
import { showAlert } from '@/utils/alert'
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

const KnowledgeSpaceDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>()
  const { t } = useI18nStore()
  const uploadRef = useRef<HTMLInputElement>(null)

  const [ns, setNs] = useState<KnowledgeNamespaceRow | null>(null)
  const [loadNs, setLoadNs] = useState(true)
  const [docs, setDocs] = useState<KnowledgeDocumentRow[]>([])
  const [docsLoading, setDocsLoading] = useState(false)
  const [docQInput, setDocQInput] = useState('')
  const [docQ, setDocQ] = useState('')
  const [tab, setTab] = useState<string>('docs')
  const [recallQ, setRecallQ] = useState('')
  const [recallTopK, setRecallTopK] = useState(5)
  const [recallMin, setRecallMin] = useState(0)
  const [recallBusy, setRecallBusy] = useState(false)
  const [recallPayload, setRecallPayload] = useState<Record<string, unknown> | null>(null)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [deleting, setDeleting] = useState(false)

  const loadNsRow = useCallback(async () => {
    if (!id) return
    setLoadNs(true)
    try {
      const res = await getKnowledgeNamespace(id)
      if (res.code !== 200) {
        showAlert(res.msg || 'not found', 'error', t('knowledge.pageTitle'))
        setNs(null)
        return
      }
      const row = (res.data as { namespace?: KnowledgeNamespaceRow })?.namespace
      setNs(row || null)
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || String(e), 'error', t('knowledge.pageTitle'))
      setNs(null)
    } finally {
      setLoadNs(false)
    }
  }, [id, t])

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
        showAlert(res.msg || 'failed', 'error', ns.name)
        return
      }
      setDocs(res.data?.list || [])
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || String(e), 'error', ns.name)
    } finally {
      setDocsLoading(false)
    }
  }, [ns, docQ])

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
      showAlert(res.msg || 'failed', 'error', t('knowledge.upload'))
      return
    }
    showAlert(res.msg || 'ok', 'success', t('knowledge.upload'))
    void loadDocs()
    void loadNsRow()
  }

  const onDeleteNs = async () => {
    if (!ns) return
    setDeleting(true)
    try {
      const res = await deleteKnowledgeNamespace(ns.id)
      if (res.code !== 200) {
        showAlert(res.msg || 'failed', 'error', t('knowledge.deleteBase'))
        return
      }
      showAlert(res.msg || 'ok', 'success', t('knowledge.deleteBase'))
      window.location.href = '/knowledge'
    } finally {
      setDeleting(false)
      setShowDeleteConfirm(false)
    }
  }

  const runRecall = async () => {
    if (!ns || !recallQ.trim()) return
    setRecallBusy(true)
    setRecallPayload(null)
    try {
      const res = await runKnowledgeRecallTest(ns.id, { query: recallQ.trim(), topK: recallTopK, minScore: recallMin })
      if (res.code !== 200) {
        showAlert(res.msg || 'failed', 'error', t('knowledge.runRecall'))
        return
      }
      setRecallPayload((res.data as Record<string, unknown>) || {})
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || String(e), 'error', t('knowledge.runRecall'))
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
      <div className="flex h-full items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-purple-500" />
      </div>
    )
  }

  if (!ns) {
    return (
      <div className="flex h-full items-center justify-center text-sm text-gray-500">
        {t('knowledge.notFound')}
      </div>
    )
  }

  return (
    <>
      <PageSEO title={`${ns.name} · ${t('knowledge.pageTitle')} · ${site}`} description={ns.namespace} />
      <div className="flex flex-col h-full">
        <PageHeader
          title={ns.name}
          subtitle={ns.namespace}
          backTo="/knowledge"
          actions={
            <>
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
              <Button variant="primary" size="sm" onClick={() => uploadRef.current?.click()} leftIcon={<Upload className="h-4 w-4" />}>
                {t('knowledge.upload')}
              </Button>
              <Button variant="destructive" size="sm" onClick={() => setShowDeleteConfirm(true)} leftIcon={<Trash2 className="h-4 w-4" />}>
                {t('knowledge.deleteBase')}
              </Button>
            </>
          }
        />

        {ns.description && (
          <div className="border-b border-border bg-muted/20 px-6 py-3">
            <p className="text-sm text-gray-600 dark:text-gray-400">{ns.description}</p>
          </div>
        )}

        <div className="flex-1 overflow-auto">
          <div className="mx-auto max-w-7xl w-full px-4 sm:px-6 lg:px-8 py-6">
            <ArcoTabs activeTab={tab} onChange={setTab} size="large" className="knowledge-tabs">
              <ArcoTabs.TabPane key="docs" title={t('knowledge.docs')}>
                <div className="pt-4">
                  {/* Search bar */}
                  <div className="mb-4 flex items-center gap-3">
                    <ArcoInput
                      size="large"
                      className="flex-1 !h-10"
                      value={docQInput}
                      onChange={(val) => setDocQInput(val)}
                      onKeyDown={(e) => e.key === 'Enter' && setDocQ(docQInput.trim())}
                      placeholder={t('knowledge.docSearchPlaceholder')}
                      allowClear
                    />
                    <Button variant="secondary" size="sm" onClick={() => setDocQ(docQInput.trim())}>
                      {t('knowledge.search')}
                    </Button>
                  </div>

                  {docsLoading ? (
                    <div className="flex justify-center py-16">
                      <Loader2 className="h-7 w-7 animate-spin text-purple-500" />
                    </div>
                  ) : docs.length === 0 ? (
                    <div className="py-12">
                      <EmptyState icon={BookOpen} title={t('knowledge.empty')} description={t('knowledge.emptyHint')} />
                    </div>
                  ) : (
                    <ArcoCard bordered className="!rounded-xl overflow-hidden">
                      <ul className="divide-y divide-gray-100 dark:divide-gray-800">
                        {docs.map((d) => (
                          <li key={d.id}>
                            <Link
                              to={`/knowledge/documents/${d.id}?ns=${encodeURIComponent(ns.id)}`}
                              className="flex items-center justify-between gap-3 px-4 py-3 hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors"
                            >
                              <div className="min-w-0">
                                <div className="truncate font-medium text-gray-900 dark:text-gray-100">{d.title}</div>
                                <div className="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                                  {(d.fileHash?.length ?? 0) > 12 ? `${d.fileHash.slice(0, 12)}...` : d.fileHash}
                                </div>
                              </div>
                              <ChevronRight className="h-4 w-4 shrink-0 text-gray-400" />
                            </Link>
                          </li>
                        ))}
                      </ul>
                    </ArcoCard>
                  )}
                </div>
              </ArcoTabs.TabPane>

              <ArcoTabs.TabPane key="recall" title={
                <span className="inline-flex items-center gap-1.5">
                  <FlaskConical className="h-4 w-4" />
                  {t('knowledge.recall')}
                </span>
              }>
                <div className="pt-4">
                  <ArcoCard bordered className="!rounded-xl !p-6 space-y-4">
                    <ArcoInput.TextArea
                      value={recallQ}
                      onChange={(val: string) => setRecallQ(val)}
                      rows={3}
                      placeholder={t('knowledge.query')}
                    />
                    <div className="flex flex-wrap gap-4">
                      <div className="w-32">
                        <label className="mb-1 block text-xs text-gray-500 dark:text-gray-400">{t('knowledge.topK')}</label>
                        <ArcoInput
                          size="large"
                          className="!h-10"
                          type="number"
                          value={String(recallTopK)}
                          onChange={(val) => setRecallTopK(parseInt(val, 10) || 5)}
                        />
                      </div>
                      <div className="w-32">
                        <label className="mb-1 block text-xs text-gray-500 dark:text-gray-400">{t('knowledge.minScore')}</label>
                        <ArcoInput
                          size="large"
                          className="!h-10"
                          type="number"
                          step={0.05}
                          value={String(recallMin)}
                          onChange={(val) => setRecallMin(parseFloat(val) || 0)}
                        />
                      </div>
                    </div>
                    <Button variant="primary" size="sm" onClick={() => void runRecall()} loading={recallBusy}>
                      {t('knowledge.runRecall')}
                    </Button>

                    {recallHits.length > 0 && (
                      <div className="space-y-2 pt-2">
                        {recallHits.map((hit, i) => (
                          <ArcoCard key={hit.record?.id || i} bordered size="small" className="!rounded-lg">
                            <div className="mb-1 flex justify-between gap-2">
                              <span className="truncate font-medium text-sm">{hit.record?.title}</span>
                              <span className="shrink-0 font-mono text-xs text-purple-600 dark:text-purple-400">
                                {typeof hit.score === 'number' ? hit.score.toFixed(4) : ''}
                              </span>
                            </div>
                            <p className="line-clamp-3 text-xs text-gray-500 dark:text-gray-400">{hit.record?.content}</p>
                          </ArcoCard>
                        ))}
                      </div>
                    )}
                  </ArcoCard>
                </div>
              </ArcoTabs.TabPane>
            </ArcoTabs>
          </div>
        </div>
      </div>

      <ConfirmDialog
        isOpen={showDeleteConfirm}
        onClose={() => setShowDeleteConfirm(false)}
        onConfirm={() => void onDeleteNs()}
        title={t('knowledge.deleteBase')}
        message={t('knowledge.deleteBaseConfirm')}
        confirmText={t('knowledge.deleteBase')}
        cancelText={t('knowledge.cancel')}
        type="danger"
        loading={deleting}
      />
    </>
  )
}

export default KnowledgeSpaceDetailPage
