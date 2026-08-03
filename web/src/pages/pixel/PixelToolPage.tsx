import { Navigate, useParams } from 'react-router-dom'
import BaseLayout from '@/components/Layout/BaseLayout'
import AssetWorkspace from '@/pages/pixel/assetWorkspace'
import SheetWorkspace from '@/pages/pixel/sheetWorkspace'
import GifWorkspace from '@/pages/pixel/gifWorkspace'
import AdjustWorkspace from '@/pages/pixel/adjustWorkspace'
import ProcessWorkspace from '@/pages/pixel/processWorkspace'
import FineWorkspace from '@/pages/pixel/fineWorkspace'
import PixelateWorkspace from '@/pages/pixel/pixelateWorkspace'
import ExpandShrinkWorkspace from '@/pages/pixel/expandShrinkWorkspace'
import GeminiWatermarkWorkspace from '@/pages/pixel/geminiWatermarkWorkspace'
import RpgmakerWorkspace from '@/pages/pixel/rpgmakerWorkspace'
import ProWorkspace from '@/pages/pixel/proWorkspace'
import ControlTestWorkspace from '@/pages/pixel/controlTestWorkspace'
import MapStitchWorkspace from '@/pages/pixel/mapStitchWorkspace'
import InfiniteMapWorkspace from '@/pages/pixel/infiniteMapWorkspace'
import type { PixelTab, TabMeta } from '@/pages/pixel/types'
import {
  Package,
  Grid2x2,
  Film,
  LayoutGrid,
  SlidersHorizontal,
  Paintbrush,
  Aperture,
  Maximize2,
  Eraser,
  Sparkles,
  Wrench,
  Gamepad2,
  Map,
  Globe2,
} from 'lucide-react'

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
  gif: {
    label: 'GIF ↔ 序列帧',
    title: 'GIF ↔ 序列帧',
    description: 'GIF 拆帧、序列帧合成 GIF、多图拼成 Sprite Sheet',
    icon: Film,
    shortHint: '拆帧 / 合 GIF / 多图合成',
  },
  adjust: {
    label: 'Sheet 调整',
    title: 'Sprite Sheet 调整',
    description: '勾选帧、动帧预览与重组合导出',
    icon: LayoutGrid,
    shortHint: '勾选 / 预览 / 重组合',
  },
  process: {
    label: '像素处理',
    title: '像素图片处理',
    description: '裁切、绿蓝幕抠图、缩放与内描边',
    icon: SlidersHorizontal,
    shortHint: '裁切 / 抠图 / 缩放 / 描边',
  },
  fine: {
    label: '精细编辑',
    title: '精细画笔编辑',
    description: '画笔、橡皮与连通域超级橡皮',
    icon: Paintbrush,
    shortHint: '画笔 / 橡皮 / 超级橡皮',
  },
  pixelate: {
    label: '图片像素化',
    title: '图片像素化',
    description: '块像素化、相近色合并与 16 色量化',
    icon: Aperture,
    shortHint: '像素块 / 合并 / 16 色',
  },
  'expand-shrink': {
    label: '扩图与缩图',
    title: '扩图与缩图（N×M）',
    description: '按网格中心裁切并统一单元格尺寸',
    icon: Maximize2,
    shortHint: 'N×M / 格宽高',
  },
  'gemini-watermark': {
    label: 'Gemini 去水印',
    title: 'Gemini 水印去除',
    description: '本地 Reverse Alpha 去除右下角 Gemini 水印',
    icon: Eraser,
    shortHint: '48/96 logo',
  },
  rpgmaker: {
    label: 'RPGMAKER 管线',
    title: 'RPGMAKER 一键管线',
    description: '打开 Gemini Gem 生成入口，本地再精修',
    icon: Sparkles,
    shortHint: 'Gem 链接枢纽',
  },
  pro: {
    label: 'RoninPro',
    title: 'RoninPro 工具箱',
    description: '自定义缩放 / 切片 / 统一尺寸 / 单图调整',
    icon: Wrench,
    shortHint: '缩放 / 切片 / 统一 / Pro',
  },
  'control-test': {
    label: '控制测试',
    title: '控制测试场景',
    description: 'Top-down / 街机手感测试',
    icon: Gamepad2,
    shortHint: 'WASD 试玩',
  },
  'map-stitch': {
    label: '地图拼接',
    title: '地图拼接',
    description: '扩图生成与边缘一致拼接',
    icon: Map,
    shortHint: 'API 扩图 / 拼接',
  },
  'infinite-map': {
    label: 'Infinite Map',
    title: 'Infinite Map',
    description: '过程化无限地图漫游（非 AI 生成）',
    icon: Globe2,
    shortHint: '过程化地形',
  },
}

function isPixelTab(v: string | undefined): v is PixelTab {
  return Boolean(v && v in TOOL_META)
}

export default function PixelToolPage() {
  const { tool } = useParams<{ tool: string }>()
  if (tool === 'matte') {
    return <Navigate to="/pixel/process" replace />
  }
  if (!isPixelTab(tool)) {
    return <Navigate to="/pixel/sheet" replace />
  }

  const meta = TOOL_META[tool]
  const dense = tool === 'map-stitch' || tool === 'infinite-map' || tool === 'control-test'

  return (
    <BaseLayout
      title={meta.title}
      description={meta.description}
      contentPadding={dense ? '12px 16px' : undefined}
      contentClassName={dense ? 'min-h-0' : undefined}
    >
      {tool === 'sheet' ? <SheetWorkspace /> : null}
      {tool === 'asset' ? <AssetWorkspace /> : null}
      {tool === 'gif' ? <GifWorkspace /> : null}
      {tool === 'adjust' ? <AdjustWorkspace /> : null}
      {tool === 'process' ? <ProcessWorkspace /> : null}
      {tool === 'fine' ? <FineWorkspace /> : null}
      {tool === 'pixelate' ? <PixelateWorkspace /> : null}
      {tool === 'expand-shrink' ? <ExpandShrinkWorkspace /> : null}
      {tool === 'gemini-watermark' ? <GeminiWatermarkWorkspace /> : null}
      {tool === 'rpgmaker' ? <RpgmakerWorkspace /> : null}
      {tool === 'pro' ? <ProWorkspace /> : null}
      {tool === 'control-test' ? <ControlTestWorkspace /> : null}
      {tool === 'map-stitch' ? <MapStitchWorkspace /> : null}
      {tool === 'infinite-map' ? <InfiniteMapWorkspace /> : null}
    </BaseLayout>
  )
}

export { TOOL_META, isPixelTab }
