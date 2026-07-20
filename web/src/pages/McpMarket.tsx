import { useCallback, useEffect, useState } from 'react'
import { motion } from 'framer-motion'
import { Link as RouterLink } from 'react-router-dom'
import { Package, Search, ShoppingBag, Zap, Code, Wrench, RefreshCw } from 'lucide-react'
import { Tag } from '@arco-design/web-react'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Button, DataList, Input, Select } from '@/components/ui'
import { activateMcpMarketItem, listMcpMarket, type McpMarketItem } from '@/api/mcpMarket'
import { getUploadsBaseURL } from '@/config/apiConfig'
import { useTranslation } from '@/i18n'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'

const CATEGORIES = ['all', 'order', 'crm', 'utility', 'custom'] as const
const categoryIcons: Record<string, typeof Package> = { order: ShoppingBag, crm: Code, utility: Wrench, custom: Zap }
const categoryColors: Record<string, string> = { order: '#10b981', crm: '#3b82f6', utility: '#64748b', custom: '#ec4899' }

export default function McpMarketPage() {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(true)
  const [rows, setRows] = useState<McpMarketItem[]>([])
  const [category, setCategory] = useState('all')
  const [keyword, setKeyword] = useState('')
  const [activating, setActivating] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listMcpMarket({ category: category === 'all' ? undefined : category, keyword: keyword.trim() || undefined })
      if (res.code !== 200) { showAlert(res.msg || t('mcpMarket.loadFailed'), 'error'); return }
      setRows(Array.isArray(res.data) ? res.data : [])
    } catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('mcpMarket.loadFailed')), 'error') }
    finally { setLoading(false) }
  }, [category, keyword, t])

  useEffect(() => { void load() }, [load])

  const handleActivate = async (item: McpMarketItem) => {
    setActivating(item.id)
    try {
      const res = await activateMcpMarketItem(item.id)
      if (res.code !== 200) { showAlert(res.msg || t('mcpMarket.activateFailed'), 'error'); return }
      showAlert(t('mcpMarket.activateOk'), 'success'); void load()
    } catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('mcpMarket.activateFailed')), 'error') }
    finally { setActivating(null) }
  }

  const resolveLogoUrl = (url?: string) => {
    if (!url) return ''
    const u = url.trim()
    if (/^https?:\/\//i.test(u)) return u
    const base = getUploadsBaseURL().replace(/\/$/, '')
    return u.startsWith('/') ? `${base}${u}` : `${base}/${u}`
  }

  const parseTags = (tags?: string) => (tags || '').split(/[,，]/).map((s) => s.trim()).filter(Boolean)

  return (
    <BaseLayout title={t('pages.mcpMarket.title')} description={t('pages.mcpMarket.description')}>
      <p className="mb-4 text-xs text-muted-foreground">
        {t('mcpMarket.mineHint')}{' '}
        <RouterLink className="text-primary underline" to="/mcp">{t('nav.myMcp')}</RouterLink>
      </p>

      <DataList
        data={rows as unknown as (McpMarketItem & Record<string, unknown>)[]}
        columns={[]}
        loading={loading}
        rowKey="id"
        emptyText={t('mcpMarket.empty')}
        renderRow={(item) => {
          const r = item as unknown as McpMarketItem
          const cat = r.category || 'custom'
          const CategoryIcon = categoryIcons[cat] || Package
          const iconColor = categoryColors[cat] || '#6366f1'
          return (
            <motion.div initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }}>
              <div className="rounded-xl border border-neutral-100 bg-white p-4 transition hover:border-neutral-200 hover:shadow-sm">
                <div className="mb-3 flex items-start justify-between gap-2">
                  <div className="flex min-w-0 flex-1 items-center gap-3">
                    {r.logoUrl ? (
                      <img src={resolveLogoUrl(r.logoUrl)} alt="" className="h-12 w-12 shrink-0 rounded-xl border border-neutral-100 object-cover" />
                    ) : (
                      <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl" style={{ backgroundColor: iconColor, color: '#fff' }}>
                        <CategoryIcon size={24} />
                      </div>
                    )}
                    <div className="min-w-0">
                      <div className="truncate text-sm font-medium text-neutral-900">{r.displayName || r.name}</div>
                      <div className="font-mono text-xs text-neutral-400">{r.slug} · v{r.version || '1.0.0'}</div>
                    </div>
                  </div>
                  {r.activated ? <Tag color="green" size="small" className="!rounded-full">{t('mcpMarket.activated')}</Tag> : null}
                </div>
                <p className="mb-3 line-clamp-3 text-sm text-neutral-500">{r.description || '—'}</p>
                <div className="mb-3 flex flex-wrap gap-1">
                  <Tag size="small" className="!rounded-full">{t(`mcpMarket.category.${cat}`) || cat}</Tag>
                  <Tag size="small" color="arcoblue" className="!rounded-full">MCP</Tag>
                  {parseTags(r.tags).map((tag) => <Tag key={tag} size="small" color="purple" className="!rounded-full">{tag}</Tag>)}
                  {(r.installCount ?? 0) > 0 ? <Tag size="small" className="!rounded-full">{t('mcpMarket.installs', { count: r.installCount ?? 0 })}</Tag> : null}
                </div>
                <div className="mb-3 truncate font-mono text-xs text-neutral-400" title={r.mcpSseUrl}>{r.mcpSseUrl}</div>
                <div className="mt-auto flex gap-2">
                  {r.activated ? (
                    <Button long type="outline" onClick={() => { window.location.href = '/mcp?tab=activated' }}>{t('mcpMarket.gotoMine')}</Button>
                  ) : (
                    <Button long type="primary" loading={activating === r.id} onClick={() => void handleActivate(r)}>{t('mcpMarket.activate')}</Button>
                  )}
                </div>
              </div>
            </motion.div>
          )
        }}
        header={
          <div className="flex flex-wrap items-center gap-2">
            <Input allowClear prefix={<Search size={14} className="text-neutral-400" />} placeholder={t('mcpMarket.searchPlaceholder')} value={keyword} onChange={setKeyword} onPressEnter={() => void load()} style={{ width: 240 }} />
            <Select value={category} onChange={setCategory} style={{ width: 160 }} options={CATEGORIES.map((c) => ({ value: c, label: t(`mcpMarket.category.${c}`) }))} />
            <div className="flex-1" />
            <Button type="outline" icon={<RefreshCw size={14} />} loading={loading} onClick={() => void load()}>{t('common.refresh')}</Button>
          </div>
        }
      />
    </BaseLayout>
  )
}
