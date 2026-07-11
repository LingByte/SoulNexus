import { useEffect, useMemo, useRef, useState } from 'react'
import { Input as ArcoInput, Select as ArcoSelect, Modal, Tag, Tabs } from '@arco-design/web-react'
import { Sparkles, Trash2, Code2, ExternalLink, Cat, Upload, Download, Copy, Store } from 'lucide-react'
import Button from '@/components/UI/Button'
import PageHeader from '@/components/Layout/PageHeader'
import EmbedPanel from '@/components/PetStudio/EmbedPanel'
import MarketStarRating from '@/components/PetStudio/MarketStarRating'
import { useI18nStore } from '@/stores/i18nStore'
import { jsTemplateService, JSTemplate } from '@/api/jsTemplate'
import { petPackageService } from '@/api/petPackage'
import { petMarketListingService, type PetMarketListing } from '@/api/petMarketListing'
import { notifyApiError, notifyApiResult } from '@/utils/apiFeedback'
import { showAlert } from '@/utils/notification'
import {
  descriptionFromReadme,
  parseProjectContent,
  previewLabelFromManifest,
} from './projectUtils'

type EmbedTarget = { jsSourceId: string; name: string }

export default function PetMarketPage() {
  const { t } = useI18nStore()
  const [templates, setTemplates] = useState<JSTemplate[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [filterType, setFilterType] = useState<'all' | 'default' | 'custom'>('all')
  const [importing, setImporting] = useState(false)
  const [tab, setTab] = useState<'mine' | 'market'>('mine')
  const [publicListings, setPublicListings] = useState<PetMarketListing[]>([])
  const [publicLoading, setPublicLoading] = useState(false)
  const [userRatings, setUserRatings] = useState<Record<string, number>>({})
  const [ratingBusy, setRatingBusy] = useState<string | null>(null)
  const [embedTarget, setEmbedTarget] = useState<EmbedTarget | null>(null)
  const importInputRef = useRef<HTMLInputElement>(null)
  const publishInputRef = useRef<HTMLInputElement>(null)

  const openEmbed = (tpl: Pick<JSTemplate, 'jsSourceId' | 'name'>) => {
    if (!tpl.jsSourceId) return
    setEmbedTarget({ jsSourceId: tpl.jsSourceId, name: tpl.name })
  }

  const fetchList = async () => {
    setLoading(true)
    try {
      const res = await jsTemplateService.getTemplates({ page: 1, limit: 200 })
      if (notifyApiResult(res, { silentSuccess: true })) {
        setTemplates(res.data.data ?? [])
      }
    } catch (e) {
      notifyApiError(e, t('jsTemplate.messages.fetchFailed'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { void fetchList() }, [])

  const fetchPublic = async () => {
    setPublicLoading(true)
    try {
      const res = await petMarketListingService.list({ page: 1, limit: 100, keyword: search.trim() || undefined })
      if (notifyApiResult(res, { silentSuccess: true })) {
        setPublicListings(res.data?.data ?? [])
      }
    } catch (e) {
      notifyApiError(e, t('petMarket.public.fetchFailed'))
    } finally {
      setPublicLoading(false)
    }
  }

  useEffect(() => {
    if (tab === 'market') void fetchPublic()
  }, [tab, search])

  const cards = useMemo(() => {
    return templates
      .map((tpl) => {
        const project = parseProjectContent(tpl.content, tpl.name)
        return {
          tpl,
          badge: previewLabelFromManifest(project),
          desc: descriptionFromReadme(project) || tpl.usage?.slice(0, 100) || t('petMarket.card.noDesc'),
        }
      })
      .filter(({ tpl }) => {
        const q = search.trim().toLowerCase()
        const matchQ = !q || tpl.name.toLowerCase().includes(q) || tpl.content.toLowerCase().includes(q)
        const matchType = filterType === 'all' || tpl.type === filterType
        return matchQ && matchType
      })
  }, [templates, search, filterType, t])

  const handleDelete = (tpl: JSTemplate) => {
    if (tpl.type === 'default') return
    Modal.confirm({
      title: t('jsTemplate.messages.deleteConfirm'),
      onOk: async () => {
        try {
          const res = await jsTemplateService.deleteTemplate(tpl.id)
          if (notifyApiResult(res, { successMessage: t('jsTemplate.messages.deleteSuccess') })) {
            void fetchList()
          }
        } catch (e) {
          notifyApiError(e, t('jsTemplate.messages.deleteFailed'))
        }
      },
    })
  }

  const handleImportPick = () => importInputRef.current?.click()

  const handleImportFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return
    setImporting(true)
    try {
      const res = await petPackageService.importZip(file)
      if (notifyApiResult(res, { successMessage: t('petMarket.importSuccess') })) {
        void fetchList()
        const tpl = res.data?.template
        const jsSourceId = res.data?.jsSourceId || tpl?.jsSourceId
        if (jsSourceId && tpl?.name) {
          setEmbedTarget({ jsSourceId, name: tpl.name })
        }
      }
    } catch (err) {
      notifyApiError(err, t('petMarket.importFailed'))
    } finally {
      setImporting(false)
    }
  }

  const handleExport = async (tpl: JSTemplate, e: React.MouseEvent) => {
    e.stopPropagation()
    try {
      await petPackageService.downloadExport(tpl.id, tpl.name)
      showAlert(t('petMarket.exportSuccess'), 'success')
    } catch (err) {
      notifyApiError(err, t('petMarket.exportFailed'))
    }
  }

  const handlePublishPick = () => publishInputRef.current?.click()

  const handlePublishFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return
    setImporting(true)
    try {
      const res = await petMarketListingService.publishZip(file)
      if (notifyApiResult(res, { successMessage: t('petMarket.public.publishSuccess') })) {
        if (tab === 'market') void fetchPublic()
      }
    } catch (err) {
      notifyApiError(err, t('petMarket.public.publishFailed'))
    } finally {
      setImporting(false)
    }
  }

  const handleFork = async (listing: PetMarketListing, e: React.MouseEvent) => {
    e.stopPropagation()
    try {
      const res = await petMarketListingService.fork(listing.marketId)
      if (notifyApiResult(res, { successMessage: t('petMarket.public.forkSuccess') })) {
        const tpl = res.data?.template
        if (tpl?.jsSourceId) {
          setEmbedTarget({ jsSourceId: tpl.jsSourceId, name: tpl.name })
        }
        void fetchList()
      }
    } catch (err) {
      notifyApiError(err, t('petMarket.public.forkFailed'))
    }
  }

  const handlePublicDownload = async (listing: PetMarketListing, e: React.MouseEvent) => {
    e.stopPropagation()
    try {
      await petMarketListingService.downloadZip(listing.marketId, listing.name)
    } catch (err) {
      notifyApiError(err, t('petMarket.exportFailed'))
    }
  }

  const handleRate = async (listing: PetMarketListing, score: number) => {
    if (ratingBusy === listing.marketId) return
    setRatingBusy(listing.marketId)
    try {
      const res = await petMarketListingService.rate(listing.marketId, score)
      if (notifyApiResult(res, { successMessage: t('petMarket.public.rateSuccess') })) {
        setUserRatings((prev) => ({ ...prev, [listing.marketId]: score }))
        setPublicListings((rows) =>
          rows.map((r) =>
            r.marketId === listing.marketId
              ? { ...r, rating: res.data?.rating ?? r.rating, ratingCount: res.data?.ratingCount ?? r.ratingCount }
              : r,
          ),
        )
      }
    } catch (err) {
      notifyApiError(err, t('petMarket.public.rateFailed'))
    } finally {
      setRatingBusy(null)
    }
  }

  return (
    <div className="flex flex-col h-full bg-gray-50 dark:bg-gray-950">
      <PageHeader
        title={t('petMarket.title')}
        subtitle={t('petMarket.subtitle')}
        actions={
          <div className="flex items-center gap-2">
            <input
              ref={importInputRef}
              type="file"
              accept=".zip,application/zip"
              className="hidden"
              onChange={(e) => void handleImportFile(e)}
            />
            <input
              ref={publishInputRef}
              type="file"
              accept=".zip,application/zip"
              className="hidden"
              onChange={(e) => void handlePublishFile(e)}
            />
            <Button
              size="sm"
              variant="outline"
              leftIcon={<Upload className="w-4 h-4" />}
              onClick={handleImportPick}
              disabled={importing}
            >
              {importing ? t('petMarket.importing') : t('petMarket.import')}
            </Button>
            <Button
              size="sm"
              variant="outline"
              leftIcon={<Store className="w-4 h-4" />}
              onClick={handlePublishPick}
              disabled={importing}
            >
              {t('petMarket.public.publish')}
            </Button>
          </div>
        }
      />

      <div className="flex-1 overflow-auto">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="mb-8 rounded-2xl border border-indigo-200/60 dark:border-indigo-900/50 bg-gradient-to-br from-indigo-50 via-white to-violet-50 dark:from-indigo-950/40 dark:via-gray-900 dark:to-violet-950/30 p-6 sm:p-8">
            <div className="flex flex-col sm:flex-row sm:items-center gap-4">
              <div className="w-14 h-14 rounded-2xl bg-indigo-600 flex items-center justify-center shrink-0 shadow-lg shadow-indigo-600/25">
                <Cat className="w-8 h-8 text-white" />
              </div>
              <div className="flex-1">
                <h2 className="text-lg font-semibold text-gray-900 dark:text-white flex items-center gap-2">
                  {t('petMarket.hero.title')}
                  <Sparkles className="w-4 h-4 text-amber-500" />
                </h2>
                <p className="text-sm text-gray-600 dark:text-gray-400 mt-1 max-w-2xl">{t('petMarket.hero.desc')}</p>
              </div>
              <Button variant="outline" size="sm" leftIcon={<Upload className="w-4 h-4" />} onClick={handleImportPick}>
                {t('petMarket.import')}
              </Button>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-3 mb-6">
            <ArcoInput
              placeholder={t('petMarket.search')}
              value={search}
              onChange={setSearch}
              className="w-full sm:w-72"
              allowClear
            />
            <ArcoSelect
              value={filterType}
              onChange={(v) => setFilterType(v as typeof filterType)}
              className="w-32"
              options={[
                { label: t('jsTemplate.filter.all'), value: 'all' },
                { label: t('jsTemplate.filter.default'), value: 'default' },
                { label: t('jsTemplate.filter.custom'), value: 'custom' },
              ]}
            />
            <span className="text-xs text-gray-500 ml-auto">{t('petMarket.count', { count: cards.length })}</span>
          </div>

          <Tabs activeTab={tab} onChange={(k) => setTab(k as 'mine' | 'market')} className="mb-6">
            <Tabs.TabPane key="mine" title={t('petMarket.tabs.mine')} />
            <Tabs.TabPane key="market" title={t('petMarket.tabs.public')} />
          </Tabs>

          {tab === 'market' ? (
            publicLoading ? (
              <div className="text-center py-20 text-gray-400">{t('jsTemplate.loading')}</div>
            ) : publicListings.length === 0 ? (
              <div className="text-center py-20 border border-dashed border-gray-300 dark:border-gray-700 rounded-2xl">
                <Store className="w-12 h-12 text-gray-300 mx-auto mb-3" />
                <p className="text-gray-600 dark:text-gray-400">{t('petMarket.public.empty')}</p>
              </div>
            ) : (
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                {publicListings.map((listing) => (
                  <article
                    key={listing.marketId}
                    className="group relative flex flex-col rounded-xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 overflow-hidden hover:border-violet-400 transition-all"
                  >
                    <div className="h-32 bg-gradient-to-br from-violet-100 to-fuchsia-100 dark:from-violet-950 dark:to-fuchsia-950 flex items-center justify-center text-4xl">
                      {listing.previewEmoji || '🎭'}
                    </div>
                    <div className="p-4 flex-1 flex flex-col">
                      <div className="flex items-start justify-between gap-2 mb-2">
                        <h3 className="font-semibold text-gray-900 dark:text-white truncate">{listing.name}</h3>
                        <Tag size="small" color="purple">{listing.kind}</Tag>
                      </div>
                      <p className="text-xs text-gray-500 dark:text-gray-400 line-clamp-2 flex-1">{listing.description || '—'}</p>
                      <div className="mt-2">
                        <MarketStarRating
                          rating={listing.rating ?? 0}
                          ratingCount={listing.ratingCount}
                          userRating={userRatings[listing.marketId]}
                          onRate={(score) => void handleRate(listing, score)}
                          disabled={ratingBusy === listing.marketId}
                        />
                      </div>
                      <div className="mt-3 pt-3 border-t border-gray-100 dark:border-gray-800 text-[11px] text-gray-400 flex justify-between">
                        <span>{t('petMarket.public.forks', { count: listing.forkCount })}</span>
                        <code className="text-[10px] opacity-60">{listing.marketId.slice(0, 12)}…</code>
                      </div>
                    </div>
                    <div className="px-4 pb-4 flex gap-2">
                      <Button size="sm" variant="primary" className="flex-1" leftIcon={<Copy className="w-3.5 h-3.5" />} onClick={(e) => void handleFork(listing, e)}>
                        {t('petMarket.public.fork')}
                      </Button>
                      <Button size="sm" variant="outline" onClick={(e) => void handlePublicDownload(listing, e)} title={t('petMarket.export')}>
                        <Download className="w-3.5 h-3.5" />
                      </Button>
                    </div>
                  </article>
                ))}
              </div>
            )
          ) : loading ? (
            <div className="text-center py-20 text-gray-400">
              <div className="animate-spin w-8 h-8 border-2 border-gray-300 border-t-indigo-600 rounded-full mx-auto mb-3" />
              {t('jsTemplate.loading')}
            </div>
          ) : cards.length === 0 ? (
            <div className="text-center py-20 border border-dashed border-gray-300 dark:border-gray-700 rounded-2xl">
              <Cat className="w-12 h-12 text-gray-300 mx-auto mb-3" />
              <p className="text-gray-600 dark:text-gray-400 mb-4">{t('petMarket.empty')}</p>
              <Button leftIcon={<Upload className="w-4 h-4" />} onClick={handleImportPick}>
                {t('petMarket.importFirst')}
              </Button>
            </div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
              {cards.map(({ tpl, badge, desc }) => (
                <article
                  key={tpl.id}
                  className="group relative flex flex-col rounded-xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 overflow-hidden hover:border-indigo-400 dark:hover:border-indigo-600 hover:shadow-lg hover:shadow-indigo-500/10 transition-all cursor-pointer"
                  onClick={() => openEmbed(tpl)}
                  onKeyDown={(e) => e.key === 'Enter' && openEmbed(tpl)}
                  role="button"
                  tabIndex={0}
                >
                  <div className="h-32 bg-gradient-to-br from-slate-100 to-indigo-100 dark:from-gray-800 dark:to-indigo-950 flex items-center justify-center">
                    <span className="text-sm font-semibold tracking-wide text-indigo-700 dark:text-indigo-300 uppercase px-3 py-1.5 rounded-lg bg-white/70 dark:bg-gray-900/50 border border-indigo-200/60 dark:border-indigo-800/60">
                      {badge}
                    </span>
                  </div>
                  <div className="p-4 flex-1 flex flex-col">
                    <div className="flex items-start justify-between gap-2 mb-2">
                      <h3 className="font-semibold text-gray-900 dark:text-white truncate">{tpl.name}</h3>
                      <Tag size="small" color={tpl.type === 'default' ? 'arcoblue' : 'green'}>
                        {tpl.type === 'default' ? t('jsTemplate.type.default') : t('jsTemplate.type.custom')}
                      </Tag>
                    </div>
                    <p className="text-xs text-gray-500 dark:text-gray-400 line-clamp-2 flex-1">{desc}</p>
                    <div className="mt-3 pt-3 border-t border-gray-100 dark:border-gray-800 flex items-center justify-between text-[11px] text-gray-400">
                      <span>{new Date(tpl.updated_at).toLocaleDateString()}</span>
                      <code className="text-[10px] opacity-60">{tpl.jsSourceId.slice(0, 8)}…</code>
                    </div>
                  </div>

                  <div className="absolute top-2 right-2 flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      type="button"
                      onClick={(e) => void handleExport(tpl, e)}
                      className="p-1.5 rounded-lg bg-white/90 dark:bg-gray-800 shadow hover:text-emerald-600"
                      title={t('petMarket.export')}
                    >
                      <Download className="w-3.5 h-3.5" />
                    </button>
                    <button
                      type="button"
                      onClick={(e) => { e.stopPropagation(); openEmbed(tpl) }}
                      className="p-1.5 rounded-lg bg-white/90 dark:bg-gray-800 shadow hover:text-indigo-600"
                      title={t('petMarket.openEmbed')}
                    >
                      <Code2 className="w-3.5 h-3.5" />
                    </button>
                    {tpl.type === 'custom' && (
                      <button
                        type="button"
                        onClick={(e) => { e.stopPropagation(); handleDelete(tpl) }}
                        className="p-1.5 rounded-lg bg-white/90 dark:bg-gray-800 shadow hover:text-red-500"
                        title={t('jsTemplate.delete')}
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    )}
                  </div>

                  <div className="px-4 pb-3 opacity-0 group-hover:opacity-100 transition-opacity">
                    <span className="text-[11px] text-indigo-600 dark:text-indigo-400 flex items-center gap-1">
                      <ExternalLink className="w-3 h-3" /> {t('petMarket.openEmbed')}
                    </span>
                  </div>
                </article>
              ))}
            </div>
          )}
        </div>
      </div>

      {embedTarget && (
        <EmbedPanel
          jsSourceId={embedTarget.jsSourceId}
          templateName={embedTarget.name}
          onClose={() => setEmbedTarget(null)}
        />
      )}
    </div>
  )
}
