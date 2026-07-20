export const CONFIG_NODE_TYPES = new Set([
  'knowledge_base',
  'ai_chat',
  'script',
  'gateway',
  'task',
  'workflow_plugin',
])

export function getNodeCanvasWidth(type: string): number {
  if (CONFIG_NODE_TYPES.has(type)) return 380
  if (type === 'start' || type === 'end') return 280
  return 300
}

export const CONNECTION_STYLES = {
  default: { color: '#60a5fa', gradient: 'url(#wf-edge-default)' },
  true: { color: '#34d399', gradient: 'url(#wf-edge-true)' },
  false: { color: '#f87171', gradient: 'url(#wf-edge-false)' },
  error: { color: '#fbbf24', gradient: 'url(#wf-edge-error)', dashArray: '6,4' },
  branch: { color: '#a78bfa', gradient: 'url(#wf-edge-branch)' },
} as const

export const ARROW_MARKERS = `
  <linearGradient id="wf-edge-default" x1="0%" y1="0%" x2="100%" y2="0%">
    <stop offset="0%" stopColor="#93c5fd" /><stop offset="100%" stopColor="#3b82f6" />
  </linearGradient>
  <linearGradient id="wf-edge-true" x1="0%" y1="0%" x2="100%" y2="0%">
    <stop offset="0%" stopColor="#6ee7b7" /><stop offset="100%" stopColor="#10b981" />
  </linearGradient>
  <linearGradient id="wf-edge-false" x1="0%" y1="0%" x2="100%" y2="0%">
    <stop offset="0%" stopColor="#fca5a5" /><stop offset="100%" stopColor="#ef4444" />
  </linearGradient>
  <linearGradient id="wf-edge-error" x1="0%" y1="0%" x2="100%" y2="0%">
    <stop offset="0%" stopColor="#fcd34d" /><stop offset="100%" stopColor="#f59e0b" />
  </linearGradient>
  <linearGradient id="wf-edge-branch" x1="0%" y1="0%" x2="100%" y2="0%">
    <stop offset="0%" stopColor="#c4b5fd" /><stop offset="100%" stopColor="#8b5cf6" />
  </linearGradient>
  ${(['default', 'true', 'false', 'error', 'branch'] as const).map((t) => {
    const fill = t === 'default' ? 'url(#wf-edge-default)'
      : t === 'true' ? 'url(#wf-edge-true)'
      : t === 'false' ? 'url(#wf-edge-false)'
      : t === 'error' ? 'url(#wf-edge-error)'
      : 'url(#wf-edge-branch)'
    return `<marker id="arrowhead-${t}" markerWidth="5" markerHeight="5" refX="4.5" refY="2.5" orient="auto" markerUnits="userSpaceOnUse">
      <path d="M0,0 L5,2.5 L0,5 Z" fill="${fill}" />
    </marker>`
  }).join('')}
`
