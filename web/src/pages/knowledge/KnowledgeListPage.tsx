// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import React, { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { BookOpen, ChevronRight, Layers, Plus, RefreshCw, Search } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'
import { PageSEO } from '@/components/SEO/PageSEO'
import PageHeader from '@/components/Layout/PageHeader'
import Button from '@/components/UI/Button'
import { Input as ArcoInput, Pagination, Tag } from '@arco-design/web-react'
import EmptyState from '@/components/UI/EmptyState'
import { showAlert } from '@/utils/alert'
import { useI18nStore } from '@/stores/i18nStore'
import { listKnowledgeNamespaces, type KnowledgeNamespaceRow } from '@/api/knowledge'
import KnowledgeCreateDrawer from '@/pages/knowledge/KnowledgeCreateDrawer'
import { cn } from '@/utils/cn'

const CARD_GRADIENTS = [
  'from-violet-500 to-purple-600',
  'from-blue-500 to-indigo-600',
  'from-emerald-500 to-teal-600',
  'from-amber-500 to-orange-600',
  'from-rose-500 to-pink-600',
  'from-cyan-500 to-blue-600',
]

function getCardGradient(id: string) {
  let hash = 0
  for (let i = 0; i < id.length; i++) {
    hash = ((hash << 5) - hash + id.charCodeAt(i)) | 0
  }
  return CARD_GRADIENTS[Math.abs(hash) % CARD_GRADIENTS.length]
}

function KnowledgeCardSkeleton() {
  return (
    <div className="rounded-2xl border border-gray-200/70 dark:border-neutral-700/60 dark:bg-neutral-800/80 p-5">
      <div className="flex items-start gap-3">
        <div className="h-11 w-11 rounded-xl animate-pulse bg-gray-200 dark:bg-neutral-700" />
        <div className="flex-1 space-y-2.5">
          <div className="h-4 w-2/3 animate-pulse rounded-md bg-gray-200 dark:bg-neutral-700" />
          <div className="h-3 w-full animate-pulse rounded-md bg-gray-200 dark:bg-neutral-700" />
        </div>
      </div>
      <div className="mt-4 flex gap-2">
        <div className="h-5 w-14 animate-pulse rounded-md bg-gray-200 dark:bg-neutral-700" />
        <div className="h-5 w-20 animate-pulse rounded-md bg-gray-200 dark:bg-neutral-700" />
      </div>
    </div>
  )
}

const KnowledgeListPage: React.FC = () => {
  const { t } = useI18nStore()
  const navigate = useNavigate()
  const [rows, setRows] = useState<KnowledgeNamespaceRow[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(30)
  const [total, setTotal] = useState(0)
  const [qInput, setQInput] = useState('')
  const [q, setQ] = useState('')
  const [drawerOpen, setDrawerOpen] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listKnowledgeNamespaces({
        page,
        pageSize,
        q: q.trim() || undefined,
      })
      if (res.code !== 200) {
        showAlert(res.msg || 'load failed', 'error', t('knowledge.pageTitle'))
        return
      }
      const d = res.data
      setRows(d?.list || [])
      setTotal(d?.total || 0)
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || String(e), 'error', t('knowledge.pageTitle'))
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, q, t])

  useEffect(() => {
    void load()
  }, [load])

  const site = t('brand.name')

  return (
    <>
      <PageSEO title={`${t('knowledge.pageTitle')} · ${site}`} description={t('knowledge.listSubtitle')} />
      <div className="flex flex-col h-full">
        <PageHeader
          title={t('knowledge.pageTitle')}
          actions={
            <>
              <Button variant="outline" size="sm" onClick={() => void load()} leftIcon={<RefreshCw className={cn('h-4 w-4', loading && 'animate-spin')} />}>
                {t('knowledge.refresh')}
              </Button>
              <Button variant="primary" size="sm" onClick={() => setDrawerOpen(true)} leftIcon={<Plus className="h-4 w-4" />}>
                {t('knowledge.newBase')}
              </Button>
            </>
          }
        />

        <div className="flex-1 overflow-auto">
          <div className="mx-auto max-w-6xl w-full px-4 sm:px-6 lg:px-8 py-6">
            {/* Search bar */}
            <div className="mb-6">
              <ArcoInput
                size="large"
                className="!h-11 !rounded-xl !bg-gray-50 dark:!bg-neutral-800/60 !border-gray-200/60 dark:!border-neutral-700/50"
                value={qInput}
                onChange={(val) => setQInput(val)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    setPage(1)
                    setQ(qInput.trim())
                  }
                }}
                placeholder={t('knowledge.searchPlaceholder')}
                prefix={<Search className="h-4 w-4 text-gray-400" />}
                allowClear
              />
            </div>

            {/* Content */}
            <AnimatePresence mode="wait">
              {loading ? (
                <motion.div
                  key="skeleton"
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  exit={{ opacity: 0 }}
                  className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3"
                >
                  {Array.from({ length: 6 }).map((_, i) => (
                    <KnowledgeCardSkeleton key={i} />
                  ))}
                </motion.div>
              ) : rows.length === 0 ? (
                <motion.div
                  key="empty"
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -20 }}
                  className="py-20"
                >
                  <EmptyState icon={BookOpen} title={t('knowledge.empty')} description={t('knowledge.emptyHint')} />
                </motion.div>
              ) : (
                <motion.div
                  key="grid"
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  exit={{ opacity: 0 }}
                  className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3"
                >
                  {rows.map((r, idx) => (
                    <motion.div
                      key={r.id}
                      initial={{ opacity: 0, y: 12 }}
                      animate={{ opacity: 1, y: 0 }}
                      transition={{ delay: idx * 0.03, duration: 0.25 }}
                      whileHover={{ y: -3 }}
                    >
                      <Link to={`/knowledge/ns/${r.id}`} className="block h-full">
                        <div className="group relative h-full rounded-2xl border border-gray-200/70 dark:border-neutral-700/60 dark:bg-neutral-800/80 p-5 transition-all duration-200 hover:border-purple-300/50 dark:hover:border-purple-500/30 hover:shadow-lg hover:shadow-purple-500/5 cursor-pointer">
                          <div className="flex items-start gap-3.5">
                            <div className={cn(
                              'flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br shadow-sm',
                              getCardGradient(r.id)
                            )}>
                              <Layers className="h-5 w-5" />
                            </div>
                            <div className="min-w-0 flex-1">
                              <h2 className="text-[15px] font-semibold leading-snug text-gray-900 dark:text-gray-100 group-hover:text-purple-600 dark:group-hover:text-purple-400 transition-colors truncate">
                                {r.name}
                              </h2>
                              <p className="mt-0.5 font-mono text-[11px] text-gray-400 dark:text-gray-500 truncate">
                                {r.namespace}
                              </p>
                            </div>
                            <ChevronRight className="mt-1 h-4 w-4 shrink-0 text-gray-300 dark:text-gray-600 transition-transform duration-200 group-hover:translate-x-0.5 group-hover:text-purple-400" />
                          </div>

                          {r.description && (
                            <p className="mt-3 line-clamp-2 text-xs leading-relaxed text-gray-500 dark:text-gray-400">
                              {r.description}
                            </p>
                          )}

                          <div className="mt-3.5 pt-3 border-t border-gray-100 dark:border-neutral-700/50 flex items-center gap-1.5">
                            <Tag size="small" color="purple" className="!rounded-md !text-[10px] !px-1.5">
                              {t('knowledge.collection')}
                            </Tag>
                            <span className="text-[11px] text-gray-400 dark:text-gray-500 truncate">{r.namespace}</span>
                          </div>
                        </div>
                      </Link>
                    </motion.div>
                  ))}
                </motion.div>
              )}
            </AnimatePresence>

            {/* Pagination */}
            {total > pageSize && !loading && (
              <div className="mt-8 flex justify-center">
                <Pagination
                  current={page}
                  total={total}
                  pageSize={pageSize}
                  onChange={(p) => setPage(p)}
                  size="small"
                  showTotal
                />
              </div>
            )}
          </div>
        </div>
      </div>

      <KnowledgeCreateDrawer
        visible={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        onCreated={(row) => navigate(`/knowledge/ns/${row.id}`)}
      />
    </>
  )
}

export default KnowledgeListPage
