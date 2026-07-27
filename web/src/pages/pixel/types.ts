import type { ComponentType } from 'react'

export type PixelTab = 'sheet' | 'asset'

export type TabMeta = {
  label: string
  title: string
  description: string
  icon: ComponentType<{ size?: number | string }>
  shortHint: string
}
