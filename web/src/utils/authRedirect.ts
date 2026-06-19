import { useAuthModalStore } from '@/stores/authModalStore'

function sanitizeNext(next?: string | null): string {
  if (!next) {
    return '/assistants'
  }
  if (!next.startsWith('/') || next.startsWith('//')) {
    return '/assistants'
  }
  return next
}

/** Open login modal, preserving return path for post-login navigation. */
export function openLoginModal(next?: string, required = false): void {
  useAuthModalStore.getState().open({
    mode: 'login',
    next: sanitizeNext(next),
    required,
  })
}

export function resolveNextPathFromQuery(search: string): string {
  const params = new URLSearchParams(search)
  return sanitizeNext(params.get('next'))
}
