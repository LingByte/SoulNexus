import { createContext, useContext, useMemo, useState, ReactNode } from 'react'
import { useLocalStorage } from '@/hooks/useLocalStorage'
import {
  SIDEBAR_COLLAPSED_WIDTH,
  SIDEBAR_DEFAULT_WIDTH,
  SIDEBAR_WIDTH_STORAGE_KEY,
  clampSidebarWidth,
} from '@/constants/sidebar'

interface SidebarContextType {
  isCollapsed: boolean
  setIsCollapsed: (collapsed: boolean) => void
  toggleCollapse: () => void
  sidebarWidth: number
  setSidebarWidth: (width: number) => void
  /** Current layout offset (collapsed icon rail vs expanded width). */
  effectiveSidebarWidth: number
  /** H5 移动端 Drawer 开关 */
  mobileOpen: boolean
  setMobileOpen: (open: boolean) => void
}

const SidebarContext = createContext<SidebarContextType | undefined>(undefined)

export const SidebarProvider = ({ children }: { children: ReactNode }) => {
  const [isCollapsed, setIsCollapsed] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)
  const [sidebarWidth, setSidebarWidthRaw] = useLocalStorage<number>(
    SIDEBAR_WIDTH_STORAGE_KEY,
    SIDEBAR_DEFAULT_WIDTH,
  )

  const setSidebarWidth = (width: number) => {
    setSidebarWidthRaw(clampSidebarWidth(width))
  }

  const toggleCollapse = () => {
    setIsCollapsed((prev) => !prev)
  }

  const effectiveSidebarWidth = isCollapsed ? SIDEBAR_COLLAPSED_WIDTH : clampSidebarWidth(sidebarWidth)

  const value = useMemo(
    () => ({
      isCollapsed,
      setIsCollapsed,
      toggleCollapse,
      sidebarWidth: clampSidebarWidth(sidebarWidth),
      setSidebarWidth,
      effectiveSidebarWidth,
      mobileOpen,
      setMobileOpen,
    }),
    [isCollapsed, sidebarWidth, effectiveSidebarWidth, mobileOpen],
  )

  return <SidebarContext.Provider value={value}>{children}</SidebarContext.Provider>
}

export const useSidebar = () => {
  const context = useContext(SidebarContext)
  if (context === undefined) {
    throw new Error('useSidebar must be used within a SidebarProvider')
  }
  return context
}
