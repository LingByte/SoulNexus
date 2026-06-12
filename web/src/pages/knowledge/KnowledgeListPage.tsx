// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import React, { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { BookOpen, ChevronRight, Layers, Plus, RefreshCw, Search } from 'lucide-react'
import { PageSEO } from '@/components/SEO/PageSEO'
import PageHeader from '@/components/Layout/PageHeader'
import Button from '@/components/UI/Button'
import { Input as ArcoInput, Card as ArcoCard, Pagination } from '@arco-design/web-react'
import EmptyState from '@/components/UI/EmptyState'
import { showAlert } from '@/utils/alert'
import { useI18nStore } from '@/stores/i18nStore'
import { listKnowledgeNamespaces, type KnowledgeNamespaceRow } from '@/api/knowledge'
import KnowledgeCreateDrawer from '@/pages/knowledge/KnowledgeCreateDrawer'
import { cn } from '@/utils/cn'

function KnowledgeCardSkeleton() {
  return (
    <ArcoCard bordered hoverable className="!rounded-xl !p-5">
      <div className="h-5 w-2/3 max-w-[12rem] animate-pulse rounded-md bg-gray-200 dark:bg-neutral-700" />
      <div className="mt-3 h-3 w-full max-w-[14rem] animate-pulse rounded-md bg-gray-200 dark:bg-neutral-700" />
      <div className="mt-4 flex gap-2">
        <div className="h-6 w-10 animate-pulse rounded bg-gray-200 dark:bg-neutral-700" />
        <div className="h-6 w-20 animate-pulse rounded bg-gray-200 dark:bg-neutral-700" />
      </div>
    </ArcoCard>
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
          <div className="mx-auto max-w-7xl w-full px-4 sm:px-6 lg:px-8 py-6 pb-16">
            {/* Search bar */}
            <div className="mb-6 flex items-center gap-3">
              <ArcoInput
                size="large"
                className="flex-1 !h-10"
                value={qInput}
                onChange={(val) => setQInput(val)}
                onKeyDown={(e) => e.key === 'Enter' && (setPage(1), setQ(qInput.trim()))}
                placeholder={t('knowledge.searchPlaceholder')}
                prefix={<Search className="h-4 w-4 text-gray-400" />}
                allowClear
              />
              <Button variant="secondary" size="sm" onClick={() => { setPage(1); setQ(qInput.trim()) }}>
                {t('knowledge.search')}
              </Button>
            </div>

            {/* Content */}
            {loading ? (
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
                {Array.from({ length: 6 }).map((_, i) => (
                  <KnowledgeCardSkeleton key={i} />
                ))}
              </div>
            ) : rows.length === 0 ? (
              <div className="py-16">
                <EmptyState icon={BookOpen} title={t('knowledge.empty')} description={t('knowledge.emptyHint')} />
              </div>
            ) : (
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
                {rows.map((r) => (
                  <Link
                    key={r.id}
                    to={`/knowledge/ns/${r.id}`}
                    className="block h-full"
                  >
                    <ArcoCard
                      bordered
                      hoverable
                      className="!rounded-xl !p-5 !h-full min-h-[152px] transition-shadow hover:shadow-md cursor-pointer"
                    >
                      <div className="flex h-full min-h-[120px] flex-col">
                        <div className="flex items-start gap-3">
                          <span className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-purple-50 text-purple-600 dark:bg-purple-900/30 dark:text-purple-400">
                            <Layers className="h-4 w-4" />
                          </span>
                          <div className="min-w-0 flex-1">
                            <h2 className="line-clamp-2 text-base font-semibold leading-snug text-gray-900 dark:text-gray-100">
                              {r.name}
                            </h2>
                            <p className="mt-1 line-clamp-1 font-mono text-xs text-gray-500 dark:text-gray-400">
                              {r.namespace}
                            </p>
                            {r.description && (
                              <p className="mt-1.5 line-clamp-2 text-xs text-gray-500 dark:text-gray-400">
                                {r.description}
                              </p>
                            )}
                          </div>
                          <ChevronRight className="mt-1 h-4 w-4 shrink-0 text-gray-400 transition-transform group-hover:translate-x-0.5" />
                        </div>
                      </div>
                    </ArcoCard>
                  </Link>
                ))}
              </div>
            )}

            {/* Pagination */}
            {total > pageSize && !loading && (
              <div className="mt-6 flex justify-center">
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
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        onCreated={(row) => navigate(`/knowledge/ns/${row.id}`)}
      />
    </>
  )
}

export default KnowledgeListPage
