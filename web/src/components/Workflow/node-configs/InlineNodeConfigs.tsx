import React, { useEffect, useState } from 'react'
import { Input, Select } from '@/components/ui'
import { listKnowledgeNamespaces, type KnowledgeNamespace } from '@/api/knowledgeNamespaces'
import { NodeField, NodeSection, NodeTag } from '../node-ui/NodeSection'
import type { WorkflowNode } from '../types/workflow'

export const KnowledgeBaseNodeConfigBody: React.FC<{
  node: WorkflowNode
  onConfigChange: (config: Record<string, unknown>) => void
  compact?: boolean
}> = ({ node, onConfigChange, compact }) => {
  const [namespaces, setNamespaces] = useState<KnowledgeNamespace[]>([])
  const [loadingNs, setLoadingNs] = useState(false)
  const cfg = node.data.config || {}

  useEffect(() => {
    setLoadingNs(true)
    void listKnowledgeNamespaces()
      .then((res) => {
        if (res.code === 200 && Array.isArray(res.data)) setNamespaces(res.data)
      })
      .finally(() => setLoadingNs(false))
  }, [])

  const patch = (partial: Record<string, unknown>) => {
    onConfigChange({ ...cfg, ...partial })
  }

  return (
    <>
      <NodeSection title="输入">
        <NodeField label="选择知识库" required>
          <Select
            size="sm"
            loading={loadingNs}
            placeholder="选择知识库"
            value={cfg.namespaceId || undefined}
            onChange={(val) => {
              const selected = namespaces.find((n) => n.id === val)
              patch({ namespaceId: val, namespaceName: selected?.name || '' })
            }}
            options={namespaces.map((ns) => ({ label: ns.name, value: ns.id }))}
          />
        </NodeField>

        <NodeField label="用户问题" required hint={<NodeTag tone="blue">String</NodeTag>}>
          <Input
            size="sm"
            value={cfg.inputVariable || 'query'}
            onChange={(val) => patch({ inputVariable: val })}
            placeholder="query"
          />
        </NodeField>

        {!compact ? (
          <>
            <div className="grid grid-cols-2 gap-2">
              <NodeField label="Top K">
                <Input
                  size="sm"
                  type="number"
                  min={1}
                  max={20}
                  value={String(cfg.topK ?? 5)}
                  onChange={(val) => patch({ topK: Math.min(20, Math.max(1, parseInt(val, 10) || 5)) })}
                />
              </NodeField>
              <NodeField label="最低相关度">
                <Input
                  size="sm"
                  type="number"
                  step="0.01"
                  value={String(cfg.minScore ?? 0)}
                  onChange={(val) => patch({ minScore: parseFloat(val) || 0 })}
                />
              </NodeField>
            </div>
            <NodeField label="输出格式">
              <Select
                size="sm"
                value={cfg.outputFormat || 'text_block'}
                onChange={(val) => patch({ outputFormat: val })}
                options={[
                  { label: '文本块', value: 'text_block' },
                  { label: '结构化 hits', value: 'hits' },
                ]}
              />
            </NodeField>
          </>
        ) : null}
      </NodeSection>

      <NodeSection title="输出" className="!bg-[var(--color-fill-1)]">
        <NodeField label="知识库引用">
          <Input
            size="sm"
            value={cfg.outputVariable || 'kb_result'}
            onChange={(val) => patch({ outputVariable: val })}
            placeholder="kb_result"
          />
        </NodeField>
      </NodeSection>
    </>
  )
}

export const AIChatNodeConfigBody: React.FC<{
  node: WorkflowNode
  onConfigChange: (config: Record<string, unknown>) => void
  compact?: boolean
}> = ({ node, onConfigChange, compact }) => {
  const cfg = node.data.config || {}
  const patch = (partial: Record<string, unknown>) => onConfigChange({ ...cfg, ...partial })

  return (
    <>
      <NodeSection title="输入">
        <NodeField label="AI 模型" required>
          <Select
            size="sm"
            value={cfg.provider || 'openai'}
            onChange={(val) => patch({ provider: val })}
            options={[
              { label: 'OpenAI', value: 'openai' },
              { label: 'Anthropic', value: 'anthropic' },
              { label: '本地模型', value: 'local' },
            ]}
          />
        </NodeField>
        <NodeField label="模型名称">
          <Input
            size="sm"
            value={cfg.model || 'gpt-4'}
            onChange={(val) => patch({ model: val })}
            placeholder="gpt-4"
          />
        </NodeField>
        {!compact ? (
          <NodeField label="系统提示词">
            <Input.TextArea
              size="sm"
              rows={3}
              value={cfg.systemPrompt || ''}
              onChange={(val) => patch({ systemPrompt: val })}
              placeholder="可选系统提示词"
            />
          </NodeField>
        ) : null}
        <NodeField label="用户问题" required hint={<NodeTag tone="blue">String</NodeTag>}>
          <Input
            size="sm"
            value={cfg.inputVariable || 'user_input'}
            onChange={(val) => patch({ inputVariable: val })}
          />
        </NodeField>
      </NodeSection>
      <NodeSection title="输出" className="!bg-[var(--color-fill-1)]">
        <NodeField label="AI 回复内容">
          <Input
            size="sm"
            value={cfg.outputVariable || 'ai_response'}
            onChange={(val) => patch({ outputVariable: val })}
          />
        </NodeField>
      </NodeSection>
    </>
  )
}
