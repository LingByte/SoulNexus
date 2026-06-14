// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import React, { useCallback, useEffect, useRef, useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { Loader2, Pencil, Save, Trash2, Upload, X } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'
import { PageSEO } from '@/components/SEO/PageSEO'
import PageHeader from '@/components/Layout/PageHeader'
import Button from '@/components/UI/Button'
import { Input as ArcoInput, Tag } from '@arco-design/web-react'
import MarkdownPreview from '@/components/UI/MarkdownPreview'
import ConfirmDialog from '@/components/UI/ConfirmDialog'
import { showAlert } from '@/utils/alert'
import { useI18nStore } from '@/stores/i18nStore'
import {
  getKnowledgeDocument,
  getKnowledgeDocumentText,
  putKnowledgeDocumentText,
  deleteKnowledgeDocument,
  reuploadKnowledgeDocument,
  isKnowledgeDocInProgress,
  type KnowledgeDocumentRow,
} from '@/api/knowledge'
import { KnowledgeDocStatusTag } from '@/components/knowledge/KnowledgeDocStatusTag'


const KnowledgeDocumentDetailPage: React.FC = () => {
  const { docId } = useParams<{ docId: string }>()
  const [searchParams] = useSearchParams()
  const nsId = searchParams.get('ns') || ''
  const { t } = useI18nStore()
  const uploadRef = useRef<HTMLInputElement>(null)

  const [doc, setDoc] = useState<KnowledgeDocumentRow | null>(null)
  const [loading, setLoading] = useState(true)
  const [editMode, setEditMode] = useState(false)
  const [md, setMd] = useState('')
  const [saving, setSaving] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [deleting, setDeleting] = useState(false)

  const backHref = nsId ? `/knowledge/ns/${nsId}` : '/knowledge'

  const loadAll = useCallback(async () => {
    if (!docId) return
    setLoading(true)
    try {
      const dRes = await getKnowledgeDocument(docId)
      if (dRes.code !== 200 || !dRes.data?.document) {
        showAlert(dRes.msg || 'not found', 'error', t('knowledge.docs'))
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
      showAlert((e as { msg?: string })?.msg || String(e), 'error', t('knowledge.docs'))
      setDoc(null)
    } finally {
      setLoading(false)
    }
  }, [docId, t])

  useEffect(() => {
    void loadAll()
  }, [loadAll])

  useEffect(() => {
    if (!doc || !isKnowledgeDocInProgress(doc.status)) return
    const timer = setInterval(() => {
      void loadAll()
    }, 3000)
    return () => clearInterval(timer)
  }, [doc, loadAll])

  const onSave = async () => {
    if (!docId) return
    setSaving(true)
    try {
      const res = await putKnowledgeDocumentText(docId, md)
      if (res.code !== 200) {
        showAlert(res.msg || 'failed', 'error', t('knowledge.saveMarkdown'))
        return
      }
      showAlert(res.msg || 'ok', 'success', t('knowledge.saveMarkdown'))
      setEditMode(false)
      void loadAll()
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || String(e), 'error', t('knowledge.saveMarkdown'))
    } finally {
      setSaving(false)
    }
  }

  const onDelete = async () => {
    if (!doc) return
    setDeleting(true)
    try {
      const res = await deleteKnowledgeDocument(doc.id)
      if (res.code !== 200) {
        showAlert(res.msg || 'failed', 'error', t('knowledge.deleteDoc'))
        return
      }
      showAlert(res.msg || 'ok', 'success', t('knowledge.deleteDoc'))
      window.location.href = backHref
    } finally {
      setDeleting(false)
      setShowDeleteConfirm(false)
    }
  }

  const onReupload = async (f: File) => {
    if (!doc) return
    const res = await reuploadKnowledgeDocument(doc.id, f)
    if (res.code !== 200) {
      showAlert(res.msg || 'failed', 'error', t('knowledge.reupload'))
      return
    }
    showAlert(res.msg || 'ok', 'success', t('knowledge.reupload'))
    void loadAll()
  }

  const site = t('brand.name')

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <Loader2 className="h-7 w-7 animate-spin text-purple-500" />
          <span className="text-xs text-gray-400">{t('knowledge.loading')}</span>
        </div>
      </div>
    )
  }

  if (!doc) {
    return (
      <div className="flex h-full items-center justify-center text-sm text-gray-500">
        {t('knowledge.notFound')}
      </div>
    )
  }

  return (
    <>
      <PageSEO title={`${doc.title} · ${t('knowledge.pageTitle')} · ${site}`} description={doc.namespace} />
      <div className="flex flex-col h-full">
        <PageHeader
          title={doc.title}
          subtitle={doc.namespace}
          backTo={backHref}
          actions={
            <>
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
              <Button variant="outline" size="sm" onClick={() => uploadRef.current?.click()} leftIcon={<Upload className="h-4 w-4" />}>
                {t('knowledge.reupload')}
              </Button>
              <Button
                variant={editMode ? 'ghost' : 'outline'}
                size="sm"
                onClick={() => setEditMode((e) => !e)}
                leftIcon={editMode ? <X className="h-4 w-4" /> : <Pencil className="h-4 w-4" />}
              >
                {editMode ? t('knowledge.cancel') : t('knowledge.editMarkdown')}
              </Button>
              <Button variant="destructive" size="sm" onClick={() => setShowDeleteConfirm(true)} leftIcon={<Trash2 className="h-4 w-4" />}>
                {t('knowledge.deleteDoc')}
              </Button>
            </>
          }
        />

        <div className="flex-1 overflow-auto">
          <div className="mx-auto max-w-4xl w-full px-4 sm:px-6 lg:px-8 py-6 pb-16">
            {(doc.status === 'failed' || isKnowledgeDocInProgress(doc.status)) && (
              <div
                className={`mb-4 rounded-xl border px-4 py-3 ${
                  doc.status === 'failed'
                    ? 'border-red-200 bg-red-50 dark:border-red-900/40 dark:bg-red-950/30'
                    : 'border-blue-200 bg-blue-50 dark:border-blue-900/40 dark:bg-blue-950/20'
                }`}
              >
                <div className="flex flex-wrap items-center gap-2">
                  <KnowledgeDocStatusTag status={doc.status} />
                  {isKnowledgeDocInProgress(doc.status) && (
                    <span className="text-xs text-gray-600 dark:text-gray-400">{t('knowledge.statusProcessing')}</span>
                  )}
                </div>
                {doc.status === 'failed' && doc.processError && (
                  <p className="mt-2 text-sm text-red-600 dark:text-red-400 whitespace-pre-wrap break-words">
                    <span className="font-medium">{t('knowledge.processError')}: </span>
                    {doc.processError}
                  </p>
                )}
              </div>
            )}
            <AnimatePresence mode="wait">
              {editMode ? (
                <motion.div
                  key="editor"
                  initial={{ opacity: 0, y: 8 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -8 }}
                  transition={{ duration: 0.2 }}
                  className="space-y-4"
                >
                  <div className="rounded-xl border border-gray-200/70 dark:border-neutral-700/60 bg-white dark:bg-neutral-800/80 p-6">
                    <div className="flex items-center gap-2 mb-4">
                      <Pencil className="h-4 w-4 text-purple-500" />
                      <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{t('knowledge.editMarkdown')}</span>
                    </div>
                    <ArcoInput.TextArea
                      value={md}
                      onChange={(val: string) => setMd(val)}
                      rows={28}
                      placeholder={t('knowledge.docTextEmpty')}
                      className="!rounded-lg !font-mono !text-sm"
                    />
                    <div className="flex items-center gap-2 mt-4 pt-4 border-t border-gray-100 dark:border-neutral-700/50">
                      <Button variant="primary" size="sm" loading={saving} onClick={() => void onSave()} leftIcon={<Save className="h-3.5 w-3.5" />}>
                        {t('knowledge.saveMarkdown')}
                      </Button>
                      <Button variant="ghost" size="sm" disabled={saving} onClick={() => { setEditMode(false); void loadAll() }}>
                        {t('knowledge.cancel')}
                      </Button>
                    </div>
                  </div>
                </motion.div>
              ) : md ? (
                <motion.div
                  key="preview"
                  initial={{ opacity: 0, y: 8 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -8 }}
                  transition={{ duration: 0.2 }}
                >
                  <div className="rounded-xl border border-gray-200/70 dark:border-neutral-700/60 bg-white dark:bg-neutral-800/80 p-6 sm:p-8">
                    <div className="flex items-center gap-2 mb-5 pb-4 border-b border-gray-100 dark:border-neutral-700/50">
                      <Tag size="small" color="purple" className="!rounded-md">
                        Markdown
                      </Tag>
                      <span className="text-xs text-gray-400 dark:text-gray-500">
                        {md.split('\n').length} lines
                      </span>
                    </div>
                    <div className="prose prose-sm max-w-none dark:prose-invert">
                      <MarkdownPreview content={md} />
                    </div>
                  </div>
                </motion.div>
              ) : (
                <motion.div
                  key="empty"
                  initial={{ opacity: 0, y: 8 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0 }}
                >
                  <div className="rounded-xl border border-dashed border-gray-300 dark:border-neutral-700 bg-gray-50/50 dark:bg-neutral-900/30 p-12 text-center">
                    <p className="text-sm text-gray-500 dark:text-gray-400">{t('knowledge.docTextEmpty')}</p>
                  </div>
                </motion.div>
              )}
            </AnimatePresence>
          </div>
        </div>
      </div>

      <ConfirmDialog
        isOpen={showDeleteConfirm}
        onClose={() => setShowDeleteConfirm(false)}
        onConfirm={() => void onDelete()}
        title={t('knowledge.deleteDoc')}
        message={`${t('knowledge.deleteDoc')}?`}
        confirmText={t('common.delete')}
        cancelText={t('common.cancel')}
        type="danger"
        loading={deleting}
      />
    </>
  )
}

export default KnowledgeDocumentDetailPage
