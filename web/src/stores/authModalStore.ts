import { create } from 'zustand'

interface AuthModalState {
  isOpen: boolean
  mode: 'login' | 'register'
  nextPath: string | null
  required: boolean
  open: (options?: { mode?: 'login' | 'register'; next?: string; required?: boolean }) => void
  close: () => void
}

export const useAuthModalStore = create<AuthModalState>((set) => ({
  isOpen: false,
  mode: 'login',
  nextPath: null,
  required: false,
  open: (options) =>
    set({
      isOpen: true,
      mode: options?.mode ?? 'login',
      nextPath: options?.next ?? null,
      required: options?.required ?? false,
    }),
  close: () => set({ isOpen: false, nextPath: null, required: false }),
}))
