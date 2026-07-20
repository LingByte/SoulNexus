const DEFAULT_OFFSET = 88

/** Smooth-scroll to a page section id (accounts for sticky header). */
export function scrollToSection(sectionId: string, offset = DEFAULT_OFFSET) {
  const id = sectionId.replace(/^#/, '')
  const el = document.getElementById(id)
  if (!el) return false
  const top = el.getBoundingClientRect().top + window.scrollY - offset
  window.scrollTo({ top: Math.max(0, top), behavior: 'smooth' })
  window.history.replaceState(null, '', `#${id}`)
  return true
}

export function landingSectionIdFromHash(hash: string): string | undefined {
  const id = hash.replace(/^#/, '').trim()
  return id || undefined
}
