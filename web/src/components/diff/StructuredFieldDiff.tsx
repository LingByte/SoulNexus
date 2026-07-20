import { useMemo } from 'react'
import { cn } from '@/utils/cn'

export type FieldDiffRow = {
  key: string
  label: string
  left: string
  right: string
  kind: 'same' | 'change' | 'add' | 'del'
}

const ASSISTANT_FIELD_LABELS: Record<string, string> = {
  version: '版本号',
  versionNo: '版本序号',
  scene: '场景',
  description: '描述',
  welcome: '开场白',
  prompt: '提示词',
  knowledgeNamespace: '知识库',
  enabled: '启用',
  name: '名称',
  ttsVoice: 'TTS 音色',
  realtimeVoice: 'Realtime 音色',
  voiceDialogWsUrl: '对话 WebSocket',
  agentConfig: 'Agent 配置（含工具绑定）',
  vadConfig: 'VAD',
  hotWords: '热词',
  interruptionConfig: '打断',
  audioTrackConfig: '音轨',
  audioProcessConfig: '上行音频',
  queryRewriter: '查询改写',
  mcpServers: '进程型 MCP（高级）',
  asrConfig: 'ASR',
  ttsConfig: 'TTS',
  llmConfig: 'LLM',
  realtimeConfig: 'Realtime',
  collect: '采集',
  boundJsTemplateSourceId: 'JS 模板',
}

function asObject(raw: unknown): Record<string, unknown> {
  if (raw == null) return {}
  if (typeof raw === 'string') {
    try {
      const parsed = JSON.parse(raw) as unknown
      if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
        return parsed as Record<string, unknown>
      }
      return { value: raw }
    } catch {
      return { value: raw }
    }
  }
  if (typeof raw === 'object' && !Array.isArray(raw)) {
    return raw as Record<string, unknown>
  }
  return { value: raw }
}

function formatValue(v: unknown): string {
  if (v == null || v === '') return '—'
  if (typeof v === 'string') {
    const t = v.trim()
    if (!t) return '—'
    try {
      const parsed = JSON.parse(t) as unknown
      if (typeof parsed === 'object') return JSON.stringify(parsed, null, 2)
    } catch {
      /* plain text */
    }
    return t
  }
  if (typeof v === 'boolean' || typeof v === 'number') return String(v)
  try {
    return JSON.stringify(v, null, 2)
  } catch {
    return String(v)
  }
}

function normalizeComparable(v: unknown): string {
  if (v == null) return ''
  if (typeof v === 'string') {
    const t = v.trim()
    try {
      return JSON.stringify(JSON.parse(t))
    } catch {
      return t
    }
  }
  try {
    return JSON.stringify(v)
  } catch {
    return String(v)
  }
}

/** Build per-field rows for assistant / script snapshot diffs. */
export function buildStructuredFieldDiff(
  left: unknown,
  right: unknown,
  labelMap: Record<string, string> = ASSISTANT_FIELD_LABELS,
): FieldDiffRow[] {
  const a = asObject(left)
  const b = asObject(right)
  const keys = Array.from(new Set([...Object.keys(a), ...Object.keys(b)])).sort((x, y) => {
    const order = Object.keys(labelMap)
    const ix = order.indexOf(x)
    const iy = order.indexOf(y)
    if (ix >= 0 && iy >= 0) return ix - iy
    if (ix >= 0) return -1
    if (iy >= 0) return 1
    return x.localeCompare(y)
  })

  return keys.map((key) => {
    const hasL = Object.prototype.hasOwnProperty.call(a, key)
    const hasR = Object.prototype.hasOwnProperty.call(b, key)
    const lv = hasL ? a[key] : undefined
    const rv = hasR ? b[key] : undefined
    const ln = normalizeComparable(lv)
    const rn = normalizeComparable(rv)
    let kind: FieldDiffRow['kind'] = 'same'
    if (!hasL && hasR) kind = 'add'
    else if (hasL && !hasR) kind = 'del'
    else if (ln !== rn) kind = 'change'
    return {
      key,
      label: labelMap[key] || key,
      left: formatValue(lv),
      right: formatValue(rv),
      kind,
    }
  })
}

type Props = {
  left: unknown
  right: unknown
  leftTitle?: string
  rightTitle?: string
  changedKeys?: string[]
  className?: string
  onlyChanged?: boolean
  labelMap?: Record<string, string>
}

/** Structured field-level side-by-side comparison (not raw JSON dump). */
export default function StructuredFieldDiff({
  left,
  right,
  leftTitle = 'A',
  rightTitle = 'B',
  changedKeys,
  className,
  onlyChanged = true,
  labelMap,
}: Props) {
  const rows = useMemo(() => {
    const all = buildStructuredFieldDiff(left, right, labelMap)
    if (!onlyChanged) return all
    const focus = new Set(changedKeys || [])
    return all.filter((r) => r.kind !== 'same' || focus.has(r.key))
  }, [left, right, onlyChanged, changedKeys, labelMap])

  const shown = rows.length ? rows : buildStructuredFieldDiff(left, right, labelMap).slice(0, 12)

  return (
    <div className={cn('overflow-hidden rounded-md border border-border', className)}>
      <div className="grid grid-cols-[140px_1fr_1fr] border-b border-border bg-muted/40 text-xs font-medium">
        <div className="border-r border-border px-3 py-2">字段</div>
        <div className="border-r border-border px-3 py-2">{leftTitle}</div>
        <div className="px-3 py-2">{rightTitle}</div>
      </div>
      <div className="max-h-[min(70vh,560px)] overflow-auto text-xs">
        {shown.map((r) => (
          <div
            key={r.key}
            className={cn(
              'grid grid-cols-[140px_1fr_1fr] border-b border-border/50 last:border-b-0',
              r.kind === 'change' && 'bg-amber-500/5',
              r.kind === 'add' && 'bg-emerald-500/5',
              r.kind === 'del' && 'bg-red-500/5',
            )}
          >
            <div className="border-r border-border/50 px-3 py-2">
              <div className="font-medium text-foreground">{r.label}</div>
              <div className="font-mono text-[10px] text-muted-foreground">{r.key}</div>
              {r.kind !== 'same' ? (
                <span
                  className={cn(
                    'mt-1 inline-block rounded px-1 py-0.5 text-[10px] font-semibold',
                    r.kind === 'change' && 'bg-amber-500/15 text-amber-700 dark:text-amber-300',
                    r.kind === 'add' && 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-300',
                    r.kind === 'del' && 'bg-red-500/15 text-red-700 dark:text-red-300',
                  )}
                >
                  {r.kind === 'change' ? '变更' : r.kind === 'add' ? '新增' : '删除'}
                </span>
              ) : null}
            </div>
            <pre
              className={cn(
                'whitespace-pre-wrap break-words border-r border-border/50 px-3 py-2 font-mono text-[11px] leading-5',
                r.kind === 'del' && 'bg-red-500/10',
                r.kind === 'change' && 'text-muted-foreground',
              )}
            >
              {r.left}
            </pre>
            <pre
              className={cn(
                'whitespace-pre-wrap break-words px-3 py-2 font-mono text-[11px] leading-5',
                r.kind === 'add' && 'bg-emerald-500/10',
                r.kind === 'change' && 'text-foreground',
              )}
            >
              {r.right}
            </pre>
          </div>
        ))}
      </div>
    </div>
  )
}
