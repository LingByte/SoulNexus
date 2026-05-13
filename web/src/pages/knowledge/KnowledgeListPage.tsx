// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import React, { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { BookOpen, ChevronRight, Layers, Plus, RefreshCw, Search } from 'lucide-react'
import { PageSEO } from '@/components/SEO/PageSEO'
import PageContainer from '@/components/Layout/PageContainer'
import Button from '@/components/UI/Button'
import Card from '@/components/UI/Card'
import Badge from '@/components/UI/Badge'
import Input from '@/components/UI/Input'
import EmptyState from '@/components/UI/EmptyState'
import { useToast } from '@/components/UI/ToastContainer'
import { useI18nStore } from '@/stores/i18nStore'
import { cn } from '@/utils/cn'
import { listKnowledgeNamespaces, type KnowledgeNamespaceRow } from '@/api/knowledge'
import KnowledgeCreateDrawer from '@/pages/knowledge/KnowledgeCreateDrawer'

type StatusFilter = 'all' | 'active' | 'processing' | 'failed' | 'deleted'

function statusVariant(s: string): 'success' | 'warning' | 'error' | 'muted' | 'default' {
  const v = (s || '').toLowerCase()
  if (v === 'active') return 'success'
  if (v === 'processing') return 'warning'
  if (v === 'failed') return 'error'
  if (v === 'deleted') return 'muted'
  return 'default'
}

function KnowledgeCardSkeleton() {
  return (
    <div className="flex min-h-[152px] flex-col rounded-xl border border-border/70 bg-card/80 p-5 shadow-sm">
      <div className="h-5 w-2/3 max-w-[12rem] animate-pulse rounded-md bg-muted" />
      <div className="mt-3 h-3 w-full max-w-[14rem] animate-pulse rounded-md bg-muted/70" />
      <div className="mt-auto flex justify-between gap-2 pt-6">
        <div className="flex gap-2">
          <div className="h-6 w-10 animate-pulse rounded-md bg-muted/60" />
          <div className="h-6 w-20 animate-pulse rounded-md bg-muted/50" />
        </div>
        <div className="h-6 w-16 animate-pulse rounded-full bg-muted/60" />
      </div>
    </div>
  )
}

const KnowledgeListPage: React.FC = () => {
  const { t } = useI18nStore()
  const navigate = useNavigate()
  const { error: toastError } = useToast()
  const [rows, setRows] = useState<KnowledgeNamespaceRow[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(30)
  const [total, setTotal] = useState(0)
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('active')
  const [qInput, setQInput] = useState('')
  const [q, setQ] = useState('')
  const [drawerOpen, setDrawerOpen] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listKnowledgeNamespaces({
        page,
        pageSize,
        status: statusFilter === 'all' ? 'all' : statusFilter,
        q: q.trim() || undefined,
      })
      if (res.code !== 200) {
        toastError(t('knowledge.pageTitle'), res.msg || 'load failed')
        return
      }
      const d = res.data
      setRows(d?.list || [])
      setTotal(d?.total || 0)
    } catch (e: unknown) {
      toastError(t('knowledge.pageTitle'), (e as { msg?: string })?.msg || String(e))
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, statusFilter, q, toastError, t])

  useEffect(() => {
    void load()
  }, [load])

  const site = t('brand.name')
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <>
      <PageSEO title={`${t('knowledge.pageTitle')} · ${site}`} description={t('knowledge.listSubtitle')} />
      <PageContainer maxWidth="full" padding="md" className="pb-16">
        <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-xl font-semibold tracking-tight">{t('knowledge.pageTitle')}</h1>
            <p className="mt-0.5 text-sm text-muted-foreground">{t('knowledge.listSubtitle')}</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button variant="outline" size="sm" onClick={() => void load()} leftIcon={<RefreshCw className={cn('h-4 w-4', loading && 'animate-spin')} />}>
              {t('knowledge.refresh')}
            </Button>
            <Button variant="primary" size="sm" onClick={() => setDrawerOpen(true)} leftIcon={<Plus className="h-4 w-4" />}>
              {t('knowledge.newBase')}
            </Button>
          </div>
        </div>

        <Card variant="outlined" padding="md" className="mb-6 border-border/80 shadow-sm">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
            <div className="relative min-w-0 flex-1">
              <Search className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={qInput}
                onChange={(e) => setQInput(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && (setPage(1), setQ(qInput.trim()))}
                placeholder={t('knowledge.searchPlaceholder')}
                className="pl-8"
              />
            </div>
            <div className="flex flex-shrink-0 flex-wrap items-center gap-2">
              <Button variant="secondary" size="sm" onClick={() => { setPage(1); setQ(qInput.trim()) }}>
                {t('knowledge.search')}
              </Button>
              <div className="flex flex-wrap gap-1.5">
                {(['all', 'active', 'processing', 'failed', 'deleted'] as StatusFilter[]).map((s) => (
                  <button
                    key={s}
                    type="button"
                    onClick={() => {
                      setStatusFilter(s)
                      setPage(1)
                    }}
                    className={cn(
                      'rounded-md px-2.5 py-1 text-xs font-medium transition-colors',
                      statusFilter === s ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground hover:bg-muted/80',
                    )}
                  >
                    {s === 'all'
                      ? t('knowledge.statusAll')
                      : s === 'active'
                        ? t('knowledge.statusActive')
                        : s === 'processing'
                          ? t('knowledge.statusProcessing')
                          : s === 'failed'
                            ? t('knowledge.statusFailed')
                            : t('knowledge.statusDeleted')}
                  </button>
                ))}
              </div>
            </div>
          </div>
        </Card>

        {loading ? (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {Array.from({ length: 6 }).map((_, i) => (
              <KnowledgeCardSkeleton key={i} />
            ))}
          </div>
        ) : rows.length === 0 ? (
          <Card variant="outlined" padding="lg" className="border-dashed border-border/80">
            <EmptyState icon={BookOpen} title={t('knowledge.empty')} description={t('knowledge.emptyHint')} />
          </Card>
        ) : (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {rows.map((r) => (
              <Link
                key={r.id}
                to={`/knowledge/ns/${r.id}`}
                className="block h-full rounded-xl outline-none ring-offset-background focus-visible:ring-2 focus-visible:ring-ring"
              >
                <Card
                  variant="elevated"
                  hover
                  padding="md"
                  animation="none"
                  className="relative h-full min-h-[152px] overflow-hidden border-border/60 bg-card/95 shadow-md"
                >
                  <div className="relative z-10 flex h-full min-h-[120px] flex-col">
                    <div className="flex items-start justify-between gap-2">
                      <div className="flex min-w-0 flex-1 items-start gap-2">
                        <span className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                          <Layers className="h-4 w-4" />
                        </span>
                        <h2 className="line-clamp-2 text-base font-semibold leading-snug tracking-tight text-foreground">{r.name}</h2>
                      </div>
                      <ChevronRight className="mt-1 h-4 w-4 shrink-0 text-muted-foreground transition-transform duration-200 group-hover:translate-x-0.5" />
                    </div>
                    <p className="mt-2 line-clamp-2 pl-11 font-mono text-[11px] leading-relaxed text-muted-foreground">{r.namespace}</p>
                    {r.description ? (
                      <p className="mt-2 line-clamp-2 pl-11 text-xs text-muted-foreground/90">{r.description}</p>
                    ) : null}
                    <div className="mt-auto flex flex-wrap items-end justify-between gap-2 border-t border-border/50 pt-4">
                      <div className="flex flex-wrap items-center gap-1.5 text-[11px] text-muted-foreground">
                        <span className="rounded-md bg-muted/90 px-2 py-0.5 font-mono font-medium tabular-nums">{r.vectorDim}d</span>
                        <span
                          className="max-w-[10rem] truncate rounded-md bg-muted/60 px-2 py-0.5 font-mono"
                          title={r.embedModel}
                        >
                          {r.embedModel}
                        </span>
                        <span className="hidden rounded-md bg-muted/40 px-2 py-0.5 uppercase sm:inline">{r.vectorProvider}</span>
                      </div>
                      <Badge variant={statusVariant(r.status)} className="shrink-0">
                        {r.status}
                      </Badge>
                    </div>
                  </div>
                </Card>
              </Link>
            ))}
          </div>
        )}

        {total > pageSize && !loading && (
          <Card variant="outlined" padding="sm" className="mt-6 flex items-center justify-center gap-1 border-border/60">
            <Button variant="ghost" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
              ‹
            </Button>
            <span className="min-w-[5rem] px-2 text-center text-sm text-muted-foreground">
              {page} / {totalPages}
            </span>
            <Button variant="ghost" size="sm" disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)}>
              ›
            </Button>
          </Card>
        )}
      </PageContainer>

      <KnowledgeCreateDrawer
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        onCreated={(row) => navigate(`/knowledge/ns/${row.id}`)}
      />
    </>
  )
}

export default KnowledgeListPage
