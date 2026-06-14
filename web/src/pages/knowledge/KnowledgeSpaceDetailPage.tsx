// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { BookOpen, ChevronRight, FlaskConical, Loader2, RefreshCw, Trash2, Upload } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'
import { PageSEO } from '@/components/SEO/PageSEO'
import PageHeader from '@/components/Layout/PageHeader'
import Button from '@/components/UI/Button'
import { Input as ArcoInput, Tabs as ArcoTabs, Tag } from '@arco-design/web-react'
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
import { KnowledgeDocStatusTag } from '@/components/knowledge/KnowledgeDocStatusTag'
import { cn } from '@/utils/cn'

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
      setDocs((res.data?.list || []).filter((d) => (d.status || '').toLowerCase() !== 'deleted'))
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

  const refreshAll = useCallback(async () => {
    await Promise.all([loadNsRow(), loadDocs()])
  }, [loadNsRow, loadDocs])

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
        <div className="flex flex-col items-center gap-3">
          <Loader2 className="h-7 w-7 animate-spin text-purple-500" />
          <span className="text-xs text-gray-400">{t('knowledge.loading')}</span>
        </div>
      </div>
    )
  }

  if (!ns) {
    return (
      <div className="flex h-full items-center justify-center">
        <EmptyState icon={BookOpen} title={t('knowledge.notFound')} />
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
              <Button variant="outline" size="sm" onClick={() => void refreshAll()} leftIcon={<RefreshCw className={cn('h-4 w-4', docsLoading && 'animate-spin')} />}>
                {t('knowledge.refresh')}
              </Button>
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
          <div className="border-b border-gray-100 dark:border-neutral-800 bg-gray-50/50 dark:bg-neutral-900/30 px-6 py-3">
            <p className="text-sm text-gray-500 dark:text-gray-400 leading-relaxed">{ns.description}</p>
          </div>
        )}

        <div className="flex-1 overflow-auto">
          <div className="mx-auto max-w-6xl w-full px-4 sm:px-6 lg:px-8 py-6">
            <ArcoTabs
              activeTab={tab}
              onChange={setTab}
              size="large"
              className="knowledge-tabs"
            >
              <ArcoTabs.TabPane key="docs" title={
                <span className="inline-flex items-center gap-1.5">
                  <BookOpen className="h-4 w-4" />
                  {t('knowledge.docs')}
                  {docs.length > 0 && (
                    <Tag size="small" color="gray" className="!rounded-full !text-[10px] !ml-0.5">
                      {docs.length}
                    </Tag>
                  )}
                </span>
              }>
                <div className="pt-5">
                  {/* Search bar */}
                  <div className="mb-4">
                    <ArcoInput
                      size="large"
                      className="!h-10 !rounded-lg !bg-gray-50 dark:!bg-neutral-800/60"
                      value={docQInput}
                      onChange={(val) => setDocQInput(val)}
                      onKeyDown={(e) => e.key === 'Enter' && setDocQ(docQInput.trim())}
                      placeholder={t('knowledge.docSearchPlaceholder')}
                      prefix={<BookOpen className="h-3.5 w-3.5 text-gray-400" />}
                      allowClear
                    />
                  </div>

                  <AnimatePresence mode="wait">
                    {docsLoading ? (
                      <motion.div
                        key="loading"
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        className="flex justify-center py-16"
                      >
                        <Loader2 className="h-6 w-6 animate-spin text-purple-500" />
                      </motion.div>
                    ) : docs.length === 0 ? (
                      <motion.div
                        key="empty"
                        initial={{ opacity: 0, y: 12 }}
                        animate={{ opacity: 1, y: 0 }}
                        exit={{ opacity: 0 }}
                        className="py-12"
                      >
                        <EmptyState icon={BookOpen} title={t('knowledge.empty')} description={t('knowledge.emptyHint')} />
                      </motion.div>
                    ) : (
                      <motion.div
                        key="list"
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        className="rounded-xl border border-gray-200/70 dark:border-neutral-700/60 bg-white dark:bg-neutral-800/80 overflow-hidden"
                      >
                        <ul className="divide-y divide-gray-100 dark:divide-neutral-700/50">
                          {docs.map((d, idx) => (
                            <motion.li
                              key={d.id}
                              initial={{ opacity: 0, x: -8 }}
                              animate={{ opacity: 1, x: 0 }}
                              transition={{ delay: idx * 0.02 }}
                            >
                              <Link
                                to={`/knowledge/documents/${d.id}?ns=${encodeURIComponent(ns.id)}`}
                                className="flex items-center justify-between gap-3 px-4 py-3.5 hover:bg-gray-50 dark:hover:bg-neutral-700/30 transition-colors group"
                              >
                                <div className="min-w-0 flex-1">
                                  <div className="flex items-center gap-2">
                                    <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-purple-50 dark:bg-purple-900/20">
                                      <BookOpen className="h-3.5 w-3.5 text-purple-500 dark:text-purple-400" />
                                    </div>
                                    <div className="min-w-0">
                                      <div className="flex items-center gap-2 min-w-0">
                                        <div className="truncate font-medium text-sm text-gray-900 dark:text-gray-100 group-hover:text-purple-600 dark:group-hover:text-purple-400 transition-colors">
                                          {d.title}
                                        </div>
                                        <KnowledgeDocStatusTag status={d.status} className="shrink-0" />
                                      </div>
                                      <div className="mt-0.5 font-mono text-[11px] text-gray-400 dark:text-gray-500">
                                        {(d.fileHash?.length ?? 0) > 12 ? `${d.fileHash.slice(0, 12)}…` : d.fileHash}
                                      </div>
                                      {d.status === 'failed' && d.processError && (
                                        <p className="mt-1 text-[11px] text-red-500 dark:text-red-400 line-clamp-2">
                                          {d.processError}
                                        </p>
                                      )}
                                    </div>
                                  </div>
                                </div>
                                <ChevronRight className="h-4 w-4 shrink-0 text-gray-300 dark:text-gray-600 transition-transform group-hover:translate-x-0.5 group-hover:text-purple-400" />
                              </Link>
                            </motion.li>
                          ))}
                        </ul>
                      </motion.div>
                    )}
                  </AnimatePresence>
                </div>
              </ArcoTabs.TabPane>

              <ArcoTabs.TabPane key="recall" title={
                <span className="inline-flex items-center gap-1.5">
                  <FlaskConical className="h-4 w-4" />
                  {t('knowledge.recall')}
                </span>
              }>
                <div className="pt-5">
                  <div className="rounded-xl border border-gray-200/70 dark:border-neutral-700/60 bg-white dark:bg-neutral-800/80 p-6 space-y-5">
                    <div>
                      <label className="mb-1.5 block text-xs font-medium text-gray-500 dark:text-gray-400">
                        {t('knowledge.query')}
                      </label>
                      <ArcoInput.TextArea
                        value={recallQ}
                        onChange={(val: string) => setRecallQ(val)}
                        rows={3}
                        placeholder={t('knowledge.query')}
                        className="!rounded-lg"
                      />
                    </div>

                    <div className="flex flex-wrap gap-4">
                      <div className="w-32">
                        <label className="mb-1.5 block text-xs font-medium text-gray-500 dark:text-gray-400">{t('knowledge.topK')}</label>
                        <ArcoInput
                          size="large"
                          className="!h-10 !rounded-lg"
                          type="number"
                          value={String(recallTopK)}
                          onChange={(val) => setRecallTopK(parseInt(val, 10) || 5)}
                        />
                      </div>
                      <div className="w-36">
                        <label className="mb-1.5 block text-xs font-medium text-gray-500 dark:text-gray-400">{t('knowledge.minScore')}</label>
                        <ArcoInput
                          size="large"
                          className="!h-10 !rounded-lg"
                          type="number"
                          step={0.05}
                          value={String(recallMin)}
                          onChange={(val) => setRecallMin(parseFloat(val) || 0)}
                        />
                      </div>
                    </div>

                    <Button
                      variant="primary"
                      size="sm"
                      onClick={() => void runRecall()}
                      loading={recallBusy}
                      leftIcon={<FlaskConical className="h-3.5 w-3.5" />}
                    >
                      {t('knowledge.runRecall')}
                    </Button>

                    <AnimatePresence>
                      {recallHits.length > 0 && (
                        <motion.div
                          initial={{ opacity: 0, height: 0 }}
                          animate={{ opacity: 1, height: 'auto' }}
                          exit={{ opacity: 0, height: 0 }}
                          className="space-y-2.5 pt-1"
                        >
                          <div className="flex items-center gap-2 text-xs font-medium text-gray-500 dark:text-gray-400">
                            <span>{t('knowledge.recallResults')}</span>
                            <Tag size="small" color="purple" className="!rounded-full !text-[10px]">
                              {recallHits.length}
                            </Tag>
                          </div>
                          {recallHits.map((hit, i) => (
                            <motion.div
                              key={hit.record?.id || i}
                              initial={{ opacity: 0, y: 8 }}
                              animate={{ opacity: 1, y: 0 }}
                              transition={{ delay: i * 0.05 }}
                              className="rounded-lg border border-gray-100 dark:border-neutral-700/50 bg-gray-50/50 dark:bg-neutral-900/30 p-3.5"
                            >
                              <div className="flex justify-between items-start gap-2 mb-1.5">
                                <span className="truncate font-medium text-sm text-gray-900 dark:text-gray-100">{hit.record?.title}</span>
                                <span className="shrink-0 rounded-md bg-purple-50 dark:bg-purple-900/20 px-2 py-0.5 font-mono text-[11px] font-medium text-purple-600 dark:text-purple-400">
                                  {typeof hit.score === 'number' ? hit.score.toFixed(4) : ''}
                                </span>
                              </div>
                              <p className="line-clamp-3 text-xs leading-relaxed text-gray-500 dark:text-gray-400">{hit.record?.content}</p>
                            </motion.div>
                          ))}
                        </motion.div>
                      )}
                    </AnimatePresence>
                  </div>
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
