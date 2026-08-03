import type { ComponentType } from 'react'

export type PixelTab =
  | 'sheet'
  | 'asset'
  | 'gif'
  | 'adjust'
  | 'process'
  | 'fine'
  | 'pixelate'
  | 'expand-shrink'
  | 'gemini-watermark'
  | 'rpgmaker'
  | 'pro'
  | 'control-test'
  | 'map-stitch'
  | 'infinite-map'

export type TabMeta = {
  label: string
  title: string
  description: string
  icon: ComponentType<{ size?: number | string }>
  shortHint: string
}
