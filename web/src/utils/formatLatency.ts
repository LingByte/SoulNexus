/** Coerce turn latency for display (handles legacy ns-as-ms rows). */
export function coerceLatencyMs(ms: number | undefined | null): number {
  const v = Math.round(Number(ms) || 0)
  if (v <= 0) return 0
  const max = 300_000
  if (v <= max) return v
  let n = v
  if (n >= 1_000_000_000) n = Math.round(n / 1_000_000)
  else if (n >= 1_000_000) n = Math.round(n / 1_000)
  if (n <= 0 || n > max) return 0
  return n
}

/** KPI-friendly latency label: sub-second in ms, otherwise seconds. */
export function formatLatencyKpi(ms: number | undefined | null): {
  value: number
  suffix: string
  precision: number
} {
  const value = coerceLatencyMs(ms)
  if (value >= 1000) {
    return { value: value / 1000, suffix: 's', precision: 1 }
  }
  return { value, suffix: 'ms', precision: 0 }
}
