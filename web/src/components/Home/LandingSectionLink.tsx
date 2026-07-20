import type { ReactNode } from 'react'
import { scrollToSection } from '@/utils/scrollToSection'
import { cn } from '@/utils/cn'

type LandingSectionLinkProps = {
  sectionId: string
  children: ReactNode
  className?: string
  onNavigate?: () => void
}

/** In-page anchor that smooth-scrolls (works with React Router; unlike `Link to="/#id"`). */
export default function LandingSectionLink({ sectionId, children, className, onNavigate }: LandingSectionLinkProps) {
  return (
    <a
      href={`#${sectionId}`}
      className={cn(className)}
      onClick={(e) => {
        e.preventDefault()
        scrollToSection(sectionId)
        onNavigate?.()
      }}
    >
      {children}
    </a>
  )
}
