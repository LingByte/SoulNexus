import { useCallback, useEffect } from 'react'
import { useBlocker } from 'react-router-dom'
import { confirmAsync } from '@/utils/notification'

type Options = {
  /** When true, leaving / switching routes prompts the user. */
  dirty: boolean
  message: string
  title?: string
}

/**
 * Blocks in-app navigations via useBlocker + Arco Modal.confirm.
 * Browser refresh/close still uses beforeunload (browser limitation).
 */
export function useUnsavedLeaveGuard({ dirty, message, title = '确认离开' }: Options) {
  const blocker = useBlocker(
    ({ currentLocation, nextLocation }) =>
      dirty && currentLocation.pathname !== nextLocation.pathname,
  )

  useEffect(() => {
    if (blocker.state !== 'blocked') return
    let cancelled = false
    void (async () => {
      const ok = await confirmAsync(message, title)
      if (cancelled) return
      if (ok) blocker.proceed?.()
      else blocker.reset?.()
    })()
    return () => {
      cancelled = true
    }
  }, [blocker, message, title])

  useEffect(() => {
    if (!dirty) return
    const onBeforeUnload = (e: BeforeUnloadEvent) => {
      e.preventDefault()
      e.returnValue = message
      return message
    }
    window.addEventListener('beforeunload', onBeforeUnload)
    return () => window.removeEventListener('beforeunload', onBeforeUnload)
  }, [dirty, message])

  const confirmLeave = useCallback(async (): Promise<boolean> => {
    if (!dirty) return true
    return confirmAsync(message, title)
  }, [dirty, message, title])

  return { confirmLeave }
}
