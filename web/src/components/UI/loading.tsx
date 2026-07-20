import {type CSSProperties } from 'react'
import { cn } from '@/utils/utils.ts'
import {createPortal} from "react-dom";

export type LoadingSize = 'sm' | 'md' | 'lg'

const sizeScale: Record<LoadingSize, number> = {
  sm: 0.55,
  md: 1,
  lg: 1.35,
}

export interface LoadingProps {
  /** Spinner scale */
  size?: LoadingSize
  /** Text below spinner */
  tip?: string
  /** Center in a block with vertical padding */
  block?: boolean
  className?: string
  style?: CSSProperties
}

export function LoadingBoxes() {
  return (
    <div className="boxes-container">
      <div className="box-group">
        <div className="box"><div /><div /><div /><div /></div>
        <div className="box"><div /><div /><div /><div /></div>
        <div className="box"><div /><div /><div /><div /></div>
        <div className="box"><div /><div /><div /><div /></div>
      </div>
    </div>
  )
}

/** Inline loading indicator — replaces Arco Spin across the app. */
export function Loading({ size = 'md', tip, block = false, className, style }: LoadingProps) {
  const scale = sizeScale[size]

  return (
    <div
      className={cn(
        'inline-flex flex-col items-center justify-center',
        block && 'w-full py-10',
        className,
      )}
      style={style}
      role="status"
      aria-live="polite"
      aria-busy="true"
    >
      <div style={{ transform: `scale(${scale})`, transformOrigin: 'center center' }}>
        <LoadingBoxes />
      </div>
      {tip ? (
        <p className="mt-4 text-sm text-[hsl(var(--muted-foreground))]">{tip}</p>
      ) : null}
    </div>
  )
}

interface LoadingOverlayProps {
  visible: boolean
  text?: string
  className?: string
}

export function LoadingOverlay({ visible, text, className }: LoadingOverlayProps) {
  if (!visible) return null

  return createPortal(
    <div
      className={cn(
        'fixed inset-0 z-[9998] flex flex-col items-center justify-center',
        'bg-white/60 dark:bg-neutral-950/60 backdrop-blur-sm',
        'transition-opacity duration-300',
        className,
      )}
    >
      <LoadingBoxes />
      {text && <p className="mt-6 text-sm text-neutral-500 dark:text-neutral-400">{text}</p>}
    </div>,
    document.body,
  )
}

export default Loading
