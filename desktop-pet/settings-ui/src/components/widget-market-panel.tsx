import { useEffect, useState } from 'react'
import { Download, Loader2, Package, Search, Store, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  fetchPublicWidgetMarket,
  trackPublicWidgetDownload,
  type PublicWidgetMarketItem,
} from '@/lib/widget-market-public'
import { createPetEntry } from '@/lib/pet-config'
import type { PetEntry } from '@/vite-env'
import { cn } from '@/lib/utils'

const CATEGORY_LABEL: Record<string, string> = {
  desktop_pet: '桌面桌宠',
  chat_widget: '聊天浮窗',
  live2d: 'Live2D',
  utility: '工具',
  custom: '自定义',
}

const DRAWER_MS = 320

type Props = {
  open: boolean
  onClose: () => void
  serverBase: string
  onAdd: (pet: PetEntry) => void
}

export function WidgetMarketPanel({ open, onClose, serverBase, onAdd }: Props) {
  const [mounted, setMounted] = useState(false)
  const [visible, setVisible] = useState(false)
  const [loading, setLoading] = useState(false)
  const [items, setItems] = useState<PublicWidgetMarketItem[]>([])
  const [keyword, setKeyword] = useState('')
  const [searchDebounce, setSearchDebounce] = useState('')
  const [err, setErr] = useState('')

  useEffect(() => {
    if (open) {
      setMounted(true)
      const raf = requestAnimationFrame(() => {
        requestAnimationFrame(() => setVisible(true))
      })
      return () => cancelAnimationFrame(raf)
    }
    setVisible(false)
    const t = window.setTimeout(() => setMounted(false), DRAWER_MS)
    return () => window.clearTimeout(t)
  }, [open])

  useEffect(() => {
    const t = setTimeout(() => setSearchDebounce(keyword), 300)
    return () => clearTimeout(t)
  }, [keyword])

  useEffect(() => {
    if (!open) return
    let cancelled = false
    ;(async () => {
      setLoading(true)
      setErr('')
      try {
        const data = await fetchPublicWidgetMarket(serverBase, { keyword: searchDebounce, size: 48 })
        if (!cancelled) setItems(data.list)
      } catch (e) {
        if (!cancelled) setErr((e as Error).message || String(e))
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [open, serverBase, searchDebounce])

  useEffect(() => {
    if (!open) {
      setKeyword('')
      setSearchDebounce('')
      setErr('')
    }
  }, [open])

  if (!mounted) return null

  const addItem = (item: PublicWidgetMarketItem) => {
    void trackPublicWidgetDownload(serverBase, item.id)
    onAdd(
      createPetEntry({
        name: item.displayName,
        title: item.displayName,
        jsSourceId: item.jsSourceId,
        enabled: true,
      }),
    )
    onClose()
  }

  return (
    <div className="fixed inset-0 z-40 flex" role="presentation">
      <button
        type="button"
        className={cn(
          'flex-1 bg-black/50 backdrop-blur-[3px] transition-opacity duration-300 ease-out motion-reduce:transition-none',
          visible ? 'opacity-100' : 'opacity-0',
        )}
        aria-label="关闭市场"
        onClick={onClose}
      />
      <div
        className={cn(
          'widget-market-drawer-panel flex h-full w-full max-w-[420px] flex-col border-l border-border bg-card shadow-2xl',
          'transition-transform duration-300 ease-out motion-reduce:transition-none',
          visible ? 'translate-x-0' : 'translate-x-full',
        )}
        role="dialog"
        aria-modal="true"
        aria-label="挂件市场"
      >
        <div className="flex items-start justify-between gap-3 border-b border-border px-5 py-4">
          <div className="min-w-0">
            <div className="flex items-center gap-2 text-foreground">
              <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <Store className="h-4 w-4" />
              </div>
              <div>
                <h2 className="text-sm font-semibold">挂件市场</h2>
                <p className="text-xs text-muted-foreground mt-0.5">公开列表，无需登录</p>
              </div>
            </div>
          </div>
          <Button type="button" variant="ghost" size="icon" className="h-8 w-8 shrink-0" onClick={onClose}>
            <X className="h-4 w-4" />
          </Button>
        </div>

        <div className="border-b border-border px-4 py-3 space-y-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              placeholder="搜索名称或 jsSourceId…"
              className="h-9 pl-8 text-xs bg-background"
            />
          </div>
          <p className="text-[11px] text-muted-foreground truncate" title={serverBase || undefined}>
            API：{serverBase.trim() || '请先在「连接」中填写 API 地址'}
          </p>
        </div>

        <div className="flex-1 overflow-y-auto p-4 space-y-3">
          {loading ? (
            <div className="flex flex-col items-center justify-center gap-2 py-16 text-sm text-muted-foreground">
              <Loader2 className="h-5 w-5 animate-spin text-primary" />
              加载挂件…
            </div>
          ) : err ? (
            <Card className="border-destructive/30 bg-destructive/5">
              <CardContent className="pt-4 text-xs text-destructive">{err}</CardContent>
            </Card>
          ) : items.length === 0 ? (
            <div className="flex flex-col items-center gap-2 py-16 text-center text-muted-foreground">
              <Package className="h-10 w-10 opacity-40" />
              <p className="text-sm">暂无上架挂件</p>
              <p className="text-xs max-w-[240px]">在 SoulNexus 网页端将挂件发布到市场后，这里会自动出现</p>
            </div>
          ) : (
            items.map((item) => (
              <Card key={item.id} className="overflow-hidden hover:border-primary/25 transition-colors">
                <CardHeader className="pb-2">
                  <div className="flex gap-3">
                    {item.avatarUrl ? (
                      <img
                        src={item.avatarUrl}
                        alt=""
                        className="h-11 w-11 rounded-lg border border-border object-cover shrink-0"
                      />
                    ) : (
                      <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg bg-muted">
                        <Package className="h-5 w-5 text-muted-foreground" />
                      </div>
                    )}
                    <div className="min-w-0 flex-1">
                      <CardTitle className="truncate">{item.displayName}</CardTitle>
                      <CardDescription className="font-mono text-[10px] truncate mt-0.5">
                        {item.jsSourceId}
                      </CardDescription>
                      <span
                        className={cn(
                          'inline-block mt-1.5 rounded-md px-1.5 py-0.5 text-[10px] font-medium',
                          'bg-primary/10 text-primary',
                        )}
                      >
                        {CATEGORY_LABEL[item.category] || item.category}
                      </span>
                    </div>
                  </div>
                </CardHeader>
                {item.description ? (
                  <CardContent className="pt-0 pb-2">
                    <p className="text-xs text-muted-foreground line-clamp-2">{item.description}</p>
                  </CardContent>
                ) : null}
                <CardContent className="pt-0">
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-[10px] text-muted-foreground">{item.installCount ?? 0} 次选用</span>
                    <Button type="button" size="sm" className="h-8 text-xs gap-1.5" onClick={() => addItem(item)}>
                      <Download className="h-3.5 w-3.5" />
                      添加到仓库
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))
          )}
        </div>
      </div>
    </div>
  )
}
