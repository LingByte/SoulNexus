import React from 'react'

/** SVG defs for workflow canvas edges — smaller arrowheads */
export const WorkflowConnectionDefs: React.FC = () => (
  <defs>
    <linearGradient id="wf-edge-default" x1="0%" y1="0%" x2="100%" y2="0%">
      <stop offset="0%" stopColor="#93c5fd" />
      <stop offset="100%" stopColor="#3b82f6" />
    </linearGradient>
    <linearGradient id="wf-edge-true" x1="0%" y1="0%" x2="100%" y2="0%">
      <stop offset="0%" stopColor="#6ee7b7" />
      <stop offset="100%" stopColor="#10b981" />
    </linearGradient>
    <linearGradient id="wf-edge-false" x1="0%" y1="0%" x2="100%" y2="0%">
      <stop offset="0%" stopColor="#fca5a5" />
      <stop offset="100%" stopColor="#ef4444" />
    </linearGradient>
    <linearGradient id="wf-edge-error" x1="0%" y1="0%" x2="100%" y2="0%">
      <stop offset="0%" stopColor="#fcd34d" />
      <stop offset="100%" stopColor="#f59e0b" />
    </linearGradient>
    <linearGradient id="wf-edge-branch" x1="0%" y1="0%" x2="100%" y2="0%">
      <stop offset="0%" stopColor="#c4b5fd" />
      <stop offset="100%" stopColor="#8b5cf6" />
    </linearGradient>
    {(
      [
        ['default', 'url(#wf-edge-default)'],
        ['true', 'url(#wf-edge-true)'],
        ['false', 'url(#wf-edge-false)'],
        ['error', 'url(#wf-edge-error)'],
        ['branch', 'url(#wf-edge-branch)'],
      ] as const
    ).map(([id, fill]) => (
      <marker
        key={id}
        id={`arrowhead-${id}`}
        markerWidth="5"
        markerHeight="5"
        refX="4.5"
        refY="2.5"
        orient="auto"
        markerUnits="userSpaceOnUse"
      >
        <path d="M0,0 L5,2.5 L0,5 Z" fill={fill} />
      </marker>
    ))}
  </defs>
)
