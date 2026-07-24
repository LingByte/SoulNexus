import { useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Brush, Film, Package, Palette, Sparkles, Grid2x2 } from 'lucide-react'
import { Button, Card } from '@/components/UI'
import { cn } from '@/utils/cn'
import BrushWorkspace from '@/pages/PixelCraftForge/brushWorkspace'
import AssetWorkspace from '@/pages/PixelCraftForge/assetWorkspace'
import GifWorkspace from '@/pages/PixelCraftForge/gifWorkspace'
import SheetWorkspace from '@/pages/PixelCraftForge/sheetWorkspace'
import GenericWorkspace from '@/pages/PixelCraftForge/genericWorkspace'
import type { PixelTab, TabMeta } from '@/pages/PixelCraftForge/types'

const tabMeta: Record<PixelTab, TabMeta> = {
  brush: { label: '像素画笔', title: '像素画笔编辑器', description: '逐像素绘制、填充、吸色与网格辅助。', icon: Brush, shortHint: '画笔 / 橡皮 / 填充 / 吸管 / 抓手' },
  gif: { label: 'GIF 拆帧', title: 'GIF 拆帧 / 序列帧转 GIF', description: 'GIF 与 PNG 序列互转。', icon: Film, shortHint: '拆帧 / 合成 / 帧间隔 / 范围裁剪' },
  sheet: { label: '精灵图工具', title: '精灵图拆分 / 合并 / 裁切排版', description: '行列拆分、序列帧合成、动画裁切。', icon: Grid2x2, shortHint: '拆分 / 合并 / 排版 / 间隙调整' },
  asset: { label: '素材库', title: '本地素材库', description: '浏览、筛选、编辑与管理本地素材资源。', icon: Package, shortHint: '导入 / 搜索 / 分类 / 导出' },
  pixelate: { label: '像素化', title: '图片像素化处理', description: '把普通图片快速处理成像素风格。', icon: Sparkles, shortHint: '块大小 / 像素风 / 预览' },
  matte: { label: '抠图', title: '色度键抠图', description: '去除纯色背景并导出透明 PNG。', icon: Palette, shortHint: '绿幕 / 蓝幕 / 透明导出' },
}

const routes = [
  { key: 'brush', href: '/pixel/tools', ...tabMeta.brush },
  { key: 'gif', href: '/pixel/tools?tab=gif', ...tabMeta.gif },
  { key: 'sheet', href: '/pixel/tools?tab=sheet', ...tabMeta.sheet },
  { key: 'asset', href: '/pixel/tools?tab=asset', ...tabMeta.asset },
  { key: 'pixelate', href: '/pixel/tools?tab=pixelate', ...tabMeta.pixelate },
  { key: 'matte', href: '/pixel/tools?tab=matte', ...tabMeta.matte },
] as const

export default function PixelCraftForgePage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = (searchParams.get('tab') as PixelTab) || 'brush'
  const activeTab = tabMeta[tab] ? tab : 'brush'
  const activeMeta = tabMeta[activeTab]
  const activeRoute = useMemo(() => routes.find((item) => item.key === activeTab) ?? routes[0], [activeTab])

  const onChange = (next: PixelTab) => {
    if (next === 'brush') {
      searchParams.delete('tab')
      setSearchParams(searchParams, { replace: true })
      return
    }
    setSearchParams({ tab: next }, { replace: true })
  }

  const ActiveIcon = activeMeta.icon

  return (
    <div className="space-y-6 p-6 lg:p-8">
      <header className="space-y-4">
        <div className="flex items-center gap-3">
          <div className="flex size-12 items-center justify-center rounded-2xl bg-primary/10 text-primary">
            <Brush size={24} />
          </div>
          <div>
            <h1 className="text-2xl font-semibold tracking-tight text-foreground">PixelCraftForge</h1>
            <p className="text-sm text-muted-foreground">像素工具箱工作台，保留原项目 UI 风格并接入 SoulNexus。</p>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {routes.map((item) => {
            const Icon = item.icon
            const selected = item.key === activeTab
            return (
              <Button
                key={item.key}
                variant={selected ? 'primary' : 'outline'}
                size="sm"
                className={cn('gap-2 rounded-full px-4', selected && 'shadow-sm')}
                onClick={() => onChange(item.key)}
              >
                <Icon size={16} />
                {item.label}
              </Button>
            )
          })}
        </div>
      </header>

      <div className="space-y-6">
        <Card className="border border-border/60 bg-card/80 p-4 shadow-sm">
          <div className="flex items-center gap-3">
            <div className="flex size-11 items-center justify-center rounded-xl bg-primary/10 text-primary">
              <ActiveIcon size={22} />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-foreground">{activeMeta.title}</h2>
              <p className="text-sm text-muted-foreground">{activeMeta.description}</p>
            </div>
          </div>
        </Card>

        {activeTab === 'brush' && <BrushWorkspace onOpenLibrary={() => onChange('asset')} />}
        {activeTab === 'gif' && <GifWorkspace />}
        {activeTab === 'sheet' && <SheetWorkspace />}
        {activeTab === 'asset' && <AssetWorkspace />}
        {activeTab !== 'brush' && activeTab !== 'gif' && activeTab !== 'sheet' && activeTab !== 'asset' && (
          <GenericWorkspace meta={activeMeta} />
        )}
      </div>
    </div>
  )
}
