import { useEffect } from 'react'

const SCRIPT_URL = '//at.alicdn.com/t/c/font_5142790_3fc9joii8pv.js'

function ensureScript() {
  if (typeof document === 'undefined') return
  if (document.querySelector(`script[data-alicdn-iconfont="${SCRIPT_URL}"]`)) return
  const script = document.createElement('script')
  script.src = SCRIPT_URL
  script.dataset.alicdnIconfont = SCRIPT_URL
  script.async = true
  document.head.appendChild(script)
}

export function useIconFontScript() {
  useEffect(() => {
    ensureScript()
  }, [])
}

export function IconFont({ type, className = '', size = 18 }: { type: string; className?: string; size?: number }) {
  useIconFontScript()
  return <svg aria-hidden="true" className={className} style={{ width: size, height: size, fill: 'currentColor' }}>
    <use href={`#icon-${type}`} />
  </svg>
}
