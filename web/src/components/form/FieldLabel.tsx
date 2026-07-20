import type { ReactNode } from 'react'
import { Typography } from '@arco-design/web-react'
import { FieldHint } from '@/components/Form/FieldHint'

type FieldLabelProps = {
  label: ReactNode
  required?: boolean
  hint?: ReactNode
  style?: React.CSSProperties
}

/** 表单项标题 + 可选必填星号 + 问号悬停说明 */
export function FieldLabel({ label, required, hint, style }: FieldLabelProps) {
  return (
    <div className="flex items-center flex-wrap gap-0" style={{ marginBottom: 6, ...style }}>
      <Typography.Text style={{ fontSize: 12, lineHeight: '20px' }}>
        {label}
        {required ? <span style={{ color: 'rgb(var(--danger-6))' }}> *</span> : null}
      </Typography.Text>
      {hint != null && hint !== '' ? <FieldHint content={hint} /> : null}
    </div>
  )
}
