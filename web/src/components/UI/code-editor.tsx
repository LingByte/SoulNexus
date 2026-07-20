import CodeMirror from '@uiw/react-codemirror'
import { html } from '@codemirror/lang-html'
import { oneDark } from '@codemirror/theme-one-dark'
import { cn } from '@/utils/cn'

export interface CodeEditorProps {
  value: string
  onChange: (value: string) => void
  language?: 'html' | 'text'
  height?: string
  readOnly?: boolean
  className?: string
  placeholder?: string
}

export function CodeEditor({
  value,
  onChange,
  language = 'html',
  height = '420px',
  readOnly = false,
  className,
  placeholder,
}: CodeEditorProps) {
  const extensions = language === 'html' ? [html()] : []

  return (
    <div
      className={cn(
        'overflow-hidden rounded-md border border-[var(--color-border-2)]',
        '[&_.cm-editor]:outline-none [&_.cm-scroller]:font-mono [&_.cm-scroller]:text-xs',
        className,
      )}
    >
      <CodeMirror
        value={value}
        height={height}
        extensions={extensions}
        theme={oneDark}
        readOnly={readOnly}
        placeholder={placeholder}
        onChange={onChange}
        basicSetup={{
          lineNumbers: true,
          foldGutter: true,
          highlightActiveLine: true,
          autocompletion: true,
          bracketMatching: true,
          indentOnInput: true,
        }}
      />
    </div>
  )
}
