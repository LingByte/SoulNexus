// stores/uiStore.ts
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// 布局类型定义
export type LayoutType = 
    | 'sidebar'           // 侧边栏布局（当前默认）
    | 'top-nav'          // 顶部导航布局
    | 'grid'             // 网格布局
    | 'card'             // 卡片布局
    | 'minimal'          // 极简布局
    | 'dashboard'        // 仪表板布局
    | 'magazine'         // 杂志布局
    | 'split'            // 分割布局
    | 'masonry'          // 瀑布流布局
    | 'kanban'           // 看板布局
    | 'timeline'         // 时间线布局
    | 'gallery'          // 画廊布局
    | 'blog'             // 博客布局
    | 'portfolio'        // 作品集布局
    | 'ecommerce'        // 电商布局
    | 'admin'            // 管理后台布局
    | 'mobile-first'     // 移动优先布局
    | 'fullscreen'       // 全屏布局
    | 'floating'         // 浮动布局

// 统一声明并导出通知位置和通知项类型
export type ToastPosition =
    | 'top-left'
    | 'top-right'
    | 'bottom-left'
    | 'bottom-right'
    | 'top-center'
    | 'bottom-center'

export interface UINotification {
    id: string
    type: 'success' | 'error' | 'warning' | 'info'
    title: string
    message?: string  // Make message optional
    duration?: number
    scrollToPosition?: {
        x: number
        y: number
        behavior?: 'smooth' | 'instant' | 'auto'
    }
    position?: ToastPosition   // 新增：位置
}

interface UIState {
    // 布局
    layoutType: LayoutType
    setLayoutType: (layout: LayoutType) => void
    
    // 侧边栏
    sidebarOpen: boolean
    setSidebarOpen: (open: boolean) => void
    toggleSidebar: () => void

    // 模态框
    modals: Record<string, boolean>
    openModal: (modalId: string) => void
    closeModal: (modalId: string) => void
    closeAllModals: () => void

    // 通知
    notifications: UINotification[]
    addNotification: (notification: Omit<UINotification, 'id'>) => void
    removeNotification: (id: string) => void
    clearNotifications: () => void

    // 加载
    loading: Record<string, boolean>
    setLoading: (key: string, loading: boolean) => void

    // 页面
    pageTitle: string
    setPageTitle: (title: string) => void

    // 搜索
    searchQuery: string
    setSearchQuery: (query: string) => void

    // 筛选
    filters: Record<string, any>
    setFilter: (key: string, value: any) => void
    clearFilters: () => void
}

export const useUIStore = create<UIState>()(
    persist(
        (set, get) => ({
            layoutType: 'sidebar',
            setLayoutType: (layout) => set({ layoutType: layout }),
            
            sidebarOpen: true,
            setSidebarOpen: (open) => set({ sidebarOpen: open }),
            toggleSidebar: () => set((s) => ({ sidebarOpen: !s.sidebarOpen })),

            modals: {},
            openModal: (modalId) => set((s) => ({ modals: { ...s.modals, [modalId]: true } })),
            closeModal: (modalId) => set((s) => ({ modals: { ...s.modals, [modalId]: false } })),
            closeAllModals: () => set({ modals: {} }),

            notifications: [],
            addNotification: (notification) => {
                const id = Math.random().toString(36).slice(2, 11)
                const newNotification: UINotification = { ...notification, id }
                set((s) => ({ notifications: [...s.notifications, newNotification] }))

                if (notification.duration !== 0) {
                    setTimeout(() => get().removeNotification(id), notification.duration || 5000)
                }
            },
            removeNotification: (id) =>
                set((s) => ({ notifications: s.notifications.filter((n) => n.id !== id) })),
            clearNotifications: () => set({ notifications: [] }),

            loading: {},
            setLoading: (key, loading) =>
                set((s) => ({ loading: { ...s.loading, [key]: loading } })),

            pageTitle: '',
            setPageTitle: (title) => set({ pageTitle: title }),

            searchQuery: '',
            setSearchQuery: (query) => set({ searchQuery: query }),

            filters: {},
            setFilter: (key, value) => set((s) => ({ filters: { ...s.filters, [key]: value } })),
            clearFilters: () => set({ filters: {} }),
        }),
        {
            name: 'ui-storage',
            partialize: (state) => ({
                layoutType: state.layoutType,
                sidebarOpen: state.sidebarOpen,
                pageTitle: state.pageTitle,
            }),
        }
    )
)
