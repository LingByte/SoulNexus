import { Navigate, useParams } from 'react-router-dom'
import BaseLayout from '@/components/Layout/BaseLayout'
import AssetWorkspace from '@/pages/pixel/assetWorkspace'
import SheetWorkspace from '@/pages/pixel/sheetWorkspace'
import type { PixelTab, TabMeta } from '@/pages/pixel/types'
import { Package, Grid2x2 } from 'lucide-react'

const TOOL_META: Record<PixelTab, TabMeta> = {
  sheet: {
    label: '精灵图拆分',
    title: '精灵图拆分',
    description: '行列拆分、序列帧合成与动画裁切排版',
    icon: Grid2x2,
    shortHint: '拆分 / 合并 / 排版 / 间隙调整',
  },
  asset: {
    label: '素材库',
    title: '素材库',
    description: '浏览、筛选、编辑与管理本地素材资源',
    icon: Package,
    shortHint: '导入 / 搜索 / 分类 / 导出',
  },
}

function isPixelTab(v: string | undefined): v is PixelTab {
  return Boolean(v && v in TOOL_META)
}

export default function PixelToolPage() {
  const { tool } = useParams<{ tool: string }>()
  if (tool === 'gif' || tool === 'pixelate' || tool === 'matte') {
    return <Navigate to="/pixel/sheet" replace />
  }
  if (!isPixelTab(tool)) {
    return <Navigate to="/pixel/sheet" replace />
  }

  const meta = TOOL_META[tool]

  return (
    <BaseLayout title={meta.title} description={meta.description}>
      {tool === 'sheet' ? <SheetWorkspace /> : null}
      {tool === 'asset' ? <AssetWorkspace /> : null}
    </BaseLayout>
  )
}

export { TOOL_META, isPixelTab }
