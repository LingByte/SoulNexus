// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import React, { useCallback, useEffect, useRef, useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { Loader2, Pencil, Trash2, Upload } from 'lucide-react'
import { PageSEO } from '@/components/SEO/PageSEO'
import PageHeader from '@/components/Layout/PageHeader'
import Button from '@/components/UI/Button'
import { Card as ArcoCard, Input as ArcoInput } from '@arco-design/web-react'
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
  type KnowledgeDocumentRow,
} from '@/api/knowledge'


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
        <Loader2 className="h-8 w-8 animate-spin text-purple-500" />
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
              <Button variant="outline" size="sm" onClick={() => setEditMode((e) => !e)} leftIcon={<Pencil className="h-4 w-4" />}>
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
            <ArcoCard bordered className="!rounded-xl !p-6">
              {editMode ? (
                <div className="space-y-4">
                  <ArcoInput.TextArea
                    value={md}
                    onChange={(val: string) => setMd(val)}
                    rows={24}
                    placeholder={t('knowledge.docTextEmpty')}
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
                <p className="text-sm text-gray-500 dark:text-gray-400">{t('knowledge.docTextEmpty')}</p>
              )}
            </ArcoCard>
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
