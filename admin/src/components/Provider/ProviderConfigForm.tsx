// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 根据 ProviderSchema 渲染厂商专属配置表单。
//   - 受控：value 是 plain object，onChange 透出整体 object
//   - 序列化为 channel.config_json 由调用方完成（JSON.stringify）
//   - 未在 schema 注册的 provider，调用方应回退到「原始 JSON 编辑器」

import { useMemo } from 'react'
import { Input, Select, InputNumber, Switch, Tooltip } from '@arco-design/web-react'
import { ExternalLink, HelpCircle } from 'lucide-react'
import type { ProviderSchema, ProviderField } from '@/data/providerSchemas'

const Option = Select.Option
const TextArea = Input.TextArea

interface Props {
  schema: ProviderSchema
  value: Record<string, any>
  onChange: (next: Record<string, any>) => void
  /** 仅 TTS 模式下传 'tts'，用于过滤 kindOnly 字段 */
  kind?: 'asr' | 'tts'
  /** 编辑模式下，敏感字段允许「保留原值」时显示占位 */
  editing?: boolean
}

export const ProviderConfigForm = ({ schema, value, onChange, kind, editing }: Props) => {
  const fields = useMemo(
    () => schema.fields.filter((f) => !f.kindOnly || (kind && f.kindOnly.includes(kind))),
    [schema, kind],
  )

  const setField = (name: string, v: any) => onChange({ ...value, [name]: v })

  return (
    <div className="space-y-3 rounded-lg border border-[var(--color-border-2)] bg-[var(--color-fill-1)] p-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">{schema.label}</span>
          {schema.docs ? (
            <a
              className="text-xs text-[var(--color-link)] inline-flex items-center gap-0.5"
              href={schema.docs}
              target="_blank"
              rel="noreferrer"
            >
              <ExternalLink size={12} /> 文档
            </a>
          ) : null}
        </div>
        <span className="text-xs text-[var(--color-text-3)]">provider: {schema.provider}</span>
      </div>

      {schema.notes ? (
        <div className="rounded bg-[var(--color-fill-2)] px-2 py-1 text-xs text-[var(--color-text-2)]">
          {schema.notes}
        </div>
      ) : null}

      <div className="grid grid-cols-2 gap-3">
        {fields.map((f) => (
          <FieldInput
            key={f.name}
            field={f}
            value={value?.[f.name]}
            onChange={(v) => setField(f.name, v)}
            editing={editing}
          />
        ))}
      </div>
    </div>
  )
}

interface FieldInputProps {
  field: ProviderField
  value: any
  onChange: (v: any) => void
  editing?: boolean
}

const FieldInput = ({ field, value, onChange, editing }: FieldInputProps) => {
  const placeholder = field.placeholder ?? (editing && field.secret ? '已保存（留空表示不修改）' : undefined)

  let control: React.ReactNode
  switch (field.type) {
    case 'password':
      control = (
        <Input.Password
          value={value ?? ''}
          onChange={(v) => onChange(v)}
          placeholder={placeholder}
          allowClear
        />
      )
      break
    case 'number':
      control = (
        <InputNumber
          value={value}
          onChange={(v) => onChange(v)}
          placeholder={placeholder}
          style={{ width: '100%' }}
        />
      )
      break
    case 'boolean':
      control = <Switch checked={!!value} onChange={(v) => onChange(v)} />
      break
    case 'select':
      control = (
        <Select value={value ?? field.default ?? ''} onChange={(v) => onChange(v)} placeholder={placeholder}>
          {(field.options || []).map((o) => (
            <Option key={o.value} value={o.value}>
              {o.label}
            </Option>
          ))}
        </Select>
      )
      break
    case 'textarea':
      control = (
        <TextArea
          value={value ?? ''}
          onChange={(v) => onChange(v)}
          placeholder={placeholder}
          autoSize={{ minRows: 2, maxRows: 6 }}
        />
      )
      break
    default:
      control = (
        <Input value={value ?? ''} onChange={(v) => onChange(v)} placeholder={placeholder} allowClear />
      )
  }

  return (
    <div className={`flex flex-col gap-1 ${field.type === 'textarea' ? 'col-span-2' : ''}`}>
      <div className="flex items-center gap-1 text-xs text-[var(--color-text-2)]">
        <span>
          {field.label}
          {field.required ? <span className="ml-0.5 text-red-500">*</span> : null}
        </span>
        {field.help ? (
          <Tooltip content={field.help}>
            <HelpCircle size={12} className="text-[var(--color-text-3)]" />
          </Tooltip>
        ) : null}
      </div>
      {control}
    </div>
  )
}

/**
 * 把 schema + 原始 config_json 字符串解析为受控对象；解析失败返回空对象。
 */
export function parseConfigJSON(raw?: string | null): Record<string, any> {
  if (!raw) return {}
  try {
    const v = JSON.parse(raw)
    return v && typeof v === 'object' && !Array.isArray(v) ? v : {}
  } catch {
    return {}
  }
}

/**
 * 校验必填字段是否齐全；返回缺失字段名数组（编辑模式下，密码类必填若已存在视为已填）。
 */
export function validateProviderConfig(
  schema: ProviderSchema,
  value: Record<string, any>,
  opts?: { editing?: boolean; alreadyFilledSecrets?: Set<string> },
): string[] {
  const missing: string[] = []
  for (const f of schema.fields) {
    if (!f.required) continue
    const v = value?.[f.name]
    const hasValue = v !== undefined && v !== null && String(v).trim() !== ''
    if (hasValue) continue
    if (opts?.editing && f.secret && opts.alreadyFilledSecrets?.has(f.name)) continue
    missing.push(f.label)
  }
  return missing
}
