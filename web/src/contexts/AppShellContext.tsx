import { createContext, useContext } from 'react'

const AppShellContext = createContext(false)

export function AppShellProvider({ children }: { children: React.ReactNode }) {
  return <AppShellContext.Provider value={true}>{children}</AppShellContext.Provider>
}

export function useInAppShell(): boolean {
  return useContext(AppShellContext)
}
