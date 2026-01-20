// 布局相关的类型定义

export interface BaseLayoutProps {
  children: React.ReactNode
  title?: string
  subtitle?: string
  className?: string
}

// 网格布局相关类型
export interface GridItem {
  id: string | number
  title: string
  description: string
  image?: string
  category?: string
  tags?: string[]
  featured?: boolean
  likes?: number
  views?: number
  [key: string]: any
}

export interface GridLayoutProps extends BaseLayoutProps {
  items?: GridItem[]
  categories?: Array<{ id: string; name: string; count: number }>
  showSearch?: boolean
  showFilters?: boolean
  showViewToggle?: boolean
  columns?: {
    mobile: number
    tablet: number
    desktop: number
    wide: number
  }
}

// 博客布局相关类型
export interface BlogPost {
  id: string | number
  title: string
  excerpt: string
  content?: string
  author: string
  date: string
  readTime: string
  tags: string[]
  likes: number
  image: string
  featured?: boolean
  category?: string
}

export interface BlogLayoutProps extends BaseLayoutProps {
  posts?: BlogPost[]
  featuredPost?: BlogPost
  showAuthor?: boolean
  showDate?: boolean
  showReadTime?: boolean
  showLikes?: boolean
  showTags?: boolean
}

// 画廊布局相关类型
export interface GalleryItem {
  id: string | number
  title: string
  description: string
  image: string
  category?: string
  tags?: string[]
  likes?: number
  downloads?: number
  featured?: boolean
}

export interface GalleryLayoutProps extends BaseLayoutProps {
  items?: GalleryItem[]
  categories?: Array<{ id: string; name: string; count: number }>
  showSearch?: boolean
  showFilters?: boolean
  showViewToggle?: boolean
  enablePreview?: boolean
}

// 作品集布局相关类型
export interface PortfolioItem {
  id: string | number
  title: string
  description: string
  image: string
  category: string
  technologies: string[]
  liveUrl?: string
  githubUrl?: string
  featured?: boolean
}

export interface PortfolioLayoutProps extends BaseLayoutProps {
  items?: PortfolioItem[]
  categories?: Array<{ id: string; name: string; count: number }>
  showFilters?: boolean
  showViewToggle?: boolean
}

// 电商布局相关类型
export interface Product {
  id: string | number
  name: string
  description: string
  price: number
  originalPrice?: number
  image: string
  category: string
  rating: number
  reviewCount: number
  inStock: boolean
  isNew?: boolean
  isSale?: boolean
  discount?: number
  tags?: string[]
}

export interface EcommerceLayoutProps extends BaseLayoutProps {
  products?: Product[]
  categories?: Array<{ id: string; name: string; count: number }>
  showSearch?: boolean
  showFilters?: boolean
  showViewToggle?: boolean
  cartCount?: number
}

// 看板布局相关类型
export interface KanbanColumn {
  id: string
  title: string
  color: string
  items: KanbanItem[]
}

export interface KanbanItem {
  id: string | number
  title: string
  description: string
  priority: 'high' | 'medium' | 'low'
  assignee?: string
  dueDate?: string
  tags?: string[]
}

export interface KanbanLayoutProps extends BaseLayoutProps {
  columns?: KanbanColumn[]
  enableDragDrop?: boolean
  showAddButton?: boolean
}

// 时间线布局相关类型
export interface TimelineEvent {
  id: string | number
  title: string
  description: string
  date: string
  time: string
  author: string
  tags: string[]
  type: 'mileline' | 'task' | 'design' | 'development' | 'meeting'
}

export interface TimelineLayoutProps extends BaseLayoutProps {
  events?: TimelineEvent[]
  showAuthor?: boolean
  showTime?: boolean
  showTags?: boolean
}

// 瀑布流布局相关类型
export interface MasonryItem {
  id: string | number
  title: string
  description: string
  image?: string
  height?: number
  category?: string
  tags?: string[]
}

export interface MasonryLayoutProps extends BaseLayoutProps {
  items?: MasonryItem[]
  columns?: {
    mobile: number
    tablet: number
    desktop: number
    wide: number
  }
}

// 仪表板布局相关类型
export interface DashboardStat {
  title: string
  value: string | number
  change: string
  trend: 'up' | 'down'
  icon?: React.ComponentType<any>
}

export interface DashboardSidebarItem {
  id: string
  name: string
  icon: React.ComponentType<any>
  count?: number
  children?: DashboardSidebarItem[]
}

export interface DashboardLayoutProps extends BaseLayoutProps {
  stats?: DashboardStat[]
  sidebarItems?: DashboardSidebarItem[]
  showStats?: boolean
  showSidebar?: boolean
}

// 管理后台布局相关类型
export interface AdminSidebarItem {
  id: string
  name: string
  icon: React.ComponentType<any>
  count?: number
  children?: AdminSidebarItem[]
}

export interface AdminLayoutProps extends BaseLayoutProps {
  sidebarItems?: AdminSidebarItem[]
  stats?: DashboardStat[]
  showSidebar?: boolean
  showStats?: boolean
  showSearch?: boolean
  notificationCount?: number
}

// 杂志布局相关类型
export interface MagazineArticle {
  id: string | number
  title: string
  excerpt: string
  image: string
  category: string
  author: string
  date: string
  readTime: string
  featured?: boolean
}

export interface MagazineSidebarItem {
  title: string
  items: Array<{
    title: string
    description: string
    color: string
  }>
}

export interface MagazineLayoutProps extends BaseLayoutProps {
  articles?: MagazineArticle[]
  sidebarItems?: MagazineSidebarItem[]
  tags?: string[]
  showSidebar?: boolean
}

// 分割布局相关类型
export interface SplitPanel {
  id: string
  title: string
  content: React.ReactNode
  width?: number
  minWidth?: number
  maxWidth?: number
}

export interface SplitLayoutProps extends BaseLayoutProps {
  panels?: SplitPanel[]
  defaultLeftWidth?: number
  enableResize?: boolean
  showGrip?: boolean
}

// 卡片布局相关类型
export interface CardLayoutProps extends BaseLayoutProps {
  showHeader?: boolean
  showFooter?: boolean
  padding?: 'sm' | 'md' | 'lg' | 'xl'
  shadow?: 'sm' | 'md' | 'lg' | 'xl'
  rounded?: 'sm' | 'md' | 'lg' | 'xl'
}

// 极简布局相关类型
export interface MinimalLayoutProps extends BaseLayoutProps {
  maxWidth?: 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '4xl' | '6xl' | '7xl'
  padding?: 'sm' | 'md' | 'lg' | 'xl'
  center?: boolean
}

// 顶部导航布局相关类型
export interface TopNavLayoutProps extends BaseLayoutProps {
  showBreadcrumb?: boolean
  showSearch?: boolean
  showNotifications?: boolean
  notificationCount?: number
}
