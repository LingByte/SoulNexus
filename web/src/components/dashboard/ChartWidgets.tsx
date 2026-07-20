import { useEffect, useRef, useState } from 'react'
import { VChart } from '@visactor/react-vchart'
import { initVChartArcoTheme } from '@visactor/vchart-arco-theme'

let arcoVChartThemeRegistered = false
function ensureArcoVChartTheme() {
  if (!arcoVChartThemeRegistered) {
    initVChartArcoTheme()
    arcoVChartThemeRegistered = true
  }
}

export function useVChartTheme() {
  useEffect(() => { ensureArcoVChartTheme() }, [])
}

interface DashboardChartProps {
  spec: Record<string, unknown>
  height?: number
}

export function DashboardChart({ spec, height = 240 }: DashboardChartProps) {
  const ref = useRef<HTMLDivElement>(null)
  const [size, setSize] = useState({ width: 0, height })

  useEffect(() => { setSize((s) => ({ ...s, height })) }, [height])

  useEffect(() => {
    const el = ref.current
    if (!el) return
    const ro = new ResizeObserver((entries) => {
      const w = Math.floor(entries[0]?.contentRect.width ?? 0)
      if (w > 0) setSize((s) => ({ ...s, width: w }))
    })
    ro.observe(el)
    return () => ro.disconnect()
  }, [])

  return (
    <div ref={ref} style={{ width: '100%', height }}>
      {size.width > 0 ? (
        <VChart
          // @ts-expect-error VChart ISpec is a discriminated union requiring 'type'; our data-driven spec merges won't match a single branch
          spec={{
            theme: 'arcoDesign' as const,
            ...spec,
            width: size.width,
            height: size.height,
          } as Record<string, unknown>}
        />
      ) : null}
    </div>
  )
}

export function EmptyChart({ tall }: { tall?: boolean }) {
  return (
    <div
      style={{
        height: tall ? 200 : 160,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        color: 'var(--color-text-3)',
        fontSize: 13,
      }}
    >
      —
    </div>
  )
}
