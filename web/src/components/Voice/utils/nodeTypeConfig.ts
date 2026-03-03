import type { WorkflowNode } from '../types/workflow'

export interface NodeTypeConfig {
  label: string
  iconName: string // 改为存储图标名称而不是 React 组件
  color: string
  gradient: string
  shadowColor: string
  inputs: number
  outputs: number
}

/**
 * 获取节点类型配置（支持国际化）
 * @param t 国际化翻译函数
 * @returns 节点类型配置对象
 */
export const getNodeTypes = (t: (key: string) => string): Record<WorkflowNode['type'], NodeTypeConfig> => ({
  start: {
    label: t('workflow.nodes.start'),
    iconName: 'Play',
    color: '#059669',
    gradient: 'from-emerald-400 to-emerald-600',
    shadowColor: 'shadow-emerald-200',
    inputs: 0,
    outputs: 1
  },
  end: {
    label: t('workflow.nodes.end'),
    iconName: 'Square',
    color: '#dc2626',
    gradient: 'from-red-400 to-red-600',
    shadowColor: 'shadow-red-200',
    inputs: 1,
    outputs: 0
  },
  task: {
    label: t('workflow.nodes.task'),
    iconName: 'FileText',
    color: '#2563eb',
    gradient: 'from-blue-400 to-blue-600',
    shadowColor: 'shadow-blue-200',
    inputs: 1,
    outputs: 1
  },
  gateway: {
    label: t('workflow.nodes.gateway'),
    iconName: 'GitBranch',
    color: '#7c3aed',
    gradient: 'from-violet-400 to-violet-600',
    shadowColor: 'shadow-violet-200',
    inputs: 1,
    outputs: 2
  },
  event: {
    label: t('workflow.nodes.event'),
    iconName: 'Zap',
    color: '#d97706',
    gradient: 'from-amber-400 to-orange-500',
    shadowColor: 'shadow-amber-200',
    inputs: 0,
    outputs: 1
  },
  subflow: {
    label: t('workflow.nodes.subflow'),
    iconName: 'Settings',
    color: '#4f46e5',
    gradient: 'from-indigo-400 to-indigo-600',
    shadowColor: 'shadow-indigo-200',
    inputs: 1,
    outputs: 1
  },
  parallel: {
    label: t('workflow.nodes.parallel'),
    iconName: 'GitBranch',
    color: '#0891b2',
    gradient: 'from-cyan-400 to-cyan-600',
    shadowColor: 'shadow-cyan-200',
    inputs: 1,
    outputs: 2
  },
  wait: {
    label: t('workflow.nodes.wait'),
    iconName: 'Clock',
    color: '#be185d',
    gradient: 'from-pink-400 to-pink-600',
    shadowColor: 'shadow-pink-200',
    inputs: 1,
    outputs: 1
  },
  timer: {
    label: t('workflow.nodes.timer'),
    iconName: 'Timer',
    color: '#0d9488',
    gradient: 'from-teal-400 to-teal-600',
    shadowColor: 'shadow-teal-200',
    inputs: 1,
    outputs: 1
  },
  script: {
    label: t('workflow.nodes.script'),
    iconName: 'Code',
    color: '#475569',
    gradient: 'from-slate-400 to-slate-600',
    shadowColor: 'shadow-slate-200',
    inputs: 1,
    outputs: 1
  },
  workflow_plugin: {
    label: t('workflow.nodes.workflowPlugin'),
    iconName: 'Package',
    color: '#7c3aed',
    gradient: 'from-purple-400 to-purple-600',
    shadowColor: 'shadow-purple-200',
    inputs: 1,
    outputs: 1
  },
  ai_chat: {
    label: 'AI对话',
    iconName: 'Bot',
    color: '#8b5cf6',
    gradient: 'from-purple-500 to-purple-700',
    shadowColor: 'shadow-purple-300',
    inputs: 1,
    outputs: 1
  }
})
