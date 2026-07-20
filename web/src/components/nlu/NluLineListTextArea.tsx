import { useEffect, useState, type KeyboardEvent } from 'react'
import { Input } from '@/components/ui'

const TextArea = Input.TextArea

/** Multiline editor for NLU keyword/sample lists (one entry per line). */
export function NluLineListTextArea({
  value,
  onChange,
  placeholder,
  className,
  minRows = 2,
  maxRows = 6,
}: {
  value: string[]
  onChange: (lines: string[]) => void
  placeholder?: string
  className?: string
  minRows?: number
  maxRows?: number
}) {
  const joined = (value ?? []).join('\n')
  const [draft, setDraft] = useState(joined)

  useEffect(() => {
    setDraft(joined)
  }, [joined])

  const commit = (text: string) => {
    onChange(
      text
        .split('\n')
        .map((s) => s.trim())
        .filter(Boolean),
    )
  }

  return (
    <TextArea
      className={className}
      placeholder={placeholder}
      value={draft}
      autoSize={{ minRows, maxRows }}
      onKeyDown={(e: KeyboardEvent) => {
        if (e.key === 'Enter') e.stopPropagation()
      }}
      onChange={setDraft}
      onBlur={() => commit(draft)}
    />
  )
}
