import type { ReactNode } from 'react'
import { Tooltip } from '@/components/ui/tooltip'
import { IconQuestionCircle } from '@arco-design/web-react/icon'

export function FieldHint({ content }: { content: ReactNode }) {
  if (content == null || content === '') return null
  return (
    <span className="inline-flex align-middle" style={{ marginLeft: 4 }}>
      <Tooltip
        variant="hint"
        hintLabel={<IconQuestionCircle style={{ fontSize: 10 }} />}
        position="top"
        trigger="hover"
        content={
          <div
            style={{
              maxWidth: 360,
              fontSize: 12,
              lineHeight: 1.55,
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-word',
            }}
          >
            {content}
          </div>
        }
      />
    </span>
  )
}
