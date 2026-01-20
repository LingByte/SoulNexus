// 布局默认数据配置

import { 
  GridItem, 
  BlogPost, 
  GalleryItem, 
  PortfolioItem, 
  Product, 
  KanbanColumn, 
  TimelineEvent, 
  MasonryItem, 
  DashboardStat, 
  DashboardSidebarItem,
  AdminSidebarItem,
  MagazineArticle,
  MagazineSidebarItem,
  SplitPanel
} from '@/types/layout'

// 网格布局默认数据
export const defaultGridItems: GridItem[] = Array.from({ length: 24 }, (_, index) => ({
  id: index + 1,
  title: `项目 ${index + 1}`,
  description: `这是第 ${index + 1} 个项目的详细描述，展示了网格布局的强大功能。`,
  category: ['design', 'development', 'marketing', 'business'][index % 4],
  image: `https://picsum.photos/400/300?random=${index + 1}`,
  likes: Math.floor(Math.random() * 1000),
  views: Math.floor(Math.random() * 5000),
  featured: index % 7 === 0,
  tags: ['React', 'TypeScript', 'UI/UX', '设计', '开发', '创新'].slice(0, Math.floor(Math.random() * 3) + 1)
}))

export const defaultGridCategories = [
  { id: 'all', name: '全部', count: 24 },
  { id: 'design', name: '设计', count: 8 },
  { id: 'development', name: '开发', count: 6 },
  { id: 'marketing', name: '营销', count: 5 },
  { id: 'business', name: '商业', count: 5 }
]

// 博客布局默认数据
export const defaultBlogPosts: BlogPost[] = [
  {
    id: 1,
    title: 'React 18 新特性深度解析',
    excerpt: '深入了解 React 18 带来的并发特性、自动批处理、Suspense 改进等新功能，以及如何在实际项目中应用这些特性。',
    author: '张三',
    date: '2024-01-15',
    readTime: '8 分钟',
    tags: ['React', 'JavaScript', '前端'],
    likes: 128,
    image: 'https://picsum.photos/600/300?random=1',
    featured: true
  },
  {
    id: 2,
    title: 'TypeScript 高级类型技巧',
    excerpt: '掌握 TypeScript 的高级类型系统，包括条件类型、映射类型、模板字面量类型等，提升代码的类型安全性。',
    author: '李四',
    date: '2024-01-12',
    readTime: '12 分钟',
    tags: ['TypeScript', '类型系统', '开发'],
    likes: 95,
    image: 'https://picsum.photos/600/300?random=2'
  },
  {
    id: 3,
    title: '现代 CSS 布局技术指南',
    excerpt: '从 Flexbox 到 Grid，从容器查询到子网格，全面了解现代 CSS 布局技术的最佳实践。',
    author: '王五',
    date: '2024-01-10',
    readTime: '15 分钟',
    tags: ['CSS', '布局', '前端'],
    likes: 156,
    image: 'https://picsum.photos/600/300?random=3'
  },
  {
    id: 4,
    title: 'Vue 3 Composition API 实践',
    excerpt: '探索 Vue 3 Composition API 的强大功能，学习如何构建可复用的组合式函数。',
    author: '赵六',
    date: '2024-01-08',
    readTime: '10 分钟',
    tags: ['Vue', 'Composition API', '前端'],
    likes: 89,
    image: 'https://picsum.photos/600/300?random=4'
  },
  {
    id: 5,
    title: 'Node.js 性能优化技巧',
    excerpt: '提升 Node.js 应用性能的实用技巧和最佳实践，包括内存管理、异步处理等。',
    author: '钱七',
    date: '2024-01-05',
    readTime: '14 分钟',
    tags: ['Node.js', '性能优化', '后端'],
    likes: 203,
    image: 'https://picsum.photos/600/300?random=5'
  }
]

// 画廊布局默认数据
export const defaultGalleryItems: GalleryItem[] = Array.from({ length: 20 }, (_, index) => ({
  id: index + 1,
  title: `图片 ${index + 1}`,
  description: `这是第 ${index + 1} 张图片的描述`,
  category: ['风景', '人物', '建筑', '抽象'][index % 4],
  image: `https://picsum.photos/600/400?random=${index + 1}`,
  likes: Math.floor(Math.random() * 1000),
  downloads: Math.floor(Math.random() * 500),
  featured: index % 8 === 0,
  tags: ['摄影', '设计', '艺术', '创意'].slice(0, Math.floor(Math.random() * 2) + 1)
}))

export const defaultGalleryCategories = [
  { id: 'all', name: '全部', count: 20 },
  { id: '风景', name: '风景', count: 5 },
  { id: '人物', name: '人物', count: 5 },
  { id: '建筑', name: '建筑', count: 5 },
  { id: '抽象', name: '抽象', count: 5 }
]

// 作品集布局默认数据
export const defaultPortfolioItems: PortfolioItem[] = [
  {
    id: 1,
    title: '电商平台设计',
    description: '现代化的电商平台界面设计，注重用户体验和视觉美感',
    category: 'web',
    image: 'https://picsum.photos/600/400?random=1',
    technologies: ['React', 'TypeScript', 'Tailwind CSS'],
    liveUrl: '#',
    githubUrl: '#',
    featured: true
  },
  {
    id: 2,
    title: '移动银行应用',
    description: '安全、直观的移动银行应用界面设计',
    category: 'mobile',
    image: 'https://picsum.photos/600/400?random=2',
    technologies: ['React Native', 'TypeScript', 'Redux'],
    liveUrl: '#',
    githubUrl: '#'
  },
  {
    id: 3,
    title: '品牌视觉识别',
    description: '为初创公司设计的完整品牌视觉识别系统',
    category: 'branding',
    image: 'https://picsum.photos/600/400?random=3',
    technologies: ['Figma', 'Illustrator', 'Photoshop'],
    liveUrl: '#',
    githubUrl: '#',
    featured: true
  },
  {
    id: 4,
    title: '数据可视化仪表板',
    description: '企业级数据可视化仪表板设计',
    category: 'web',
    image: 'https://picsum.photos/600/400?random=4',
    technologies: ['D3.js', 'React', 'Chart.js'],
    liveUrl: '#',
    githubUrl: '#'
  },
  {
    id: 5,
    title: '健身追踪应用',
    description: '个人健身追踪和社交分享应用',
    category: 'mobile',
    image: 'https://picsum.photos/600/400?random=5',
    technologies: ['Flutter', 'Firebase', 'Dart'],
    liveUrl: '#',
    githubUrl: '#'
  },
  {
    id: 6,
    title: '咖啡品牌设计',
    description: '精品咖啡品牌的完整视觉设计',
    category: 'branding',
    image: 'https://picsum.photos/600/400?random=6',
    technologies: ['Illustrator', 'Photoshop', 'InDesign'],
    liveUrl: '#',
    githubUrl: '#',
    featured: true
  }
]

export const defaultPortfolioCategories = [
  { id: 'all', name: '全部项目', count: 6 },
  { id: 'web', name: '网站设计', count: 2 },
  { id: 'mobile', name: '移动应用', count: 2 },
  { id: 'branding', name: '品牌设计', count: 2 }
]

// 电商布局默认数据
export const defaultProducts: Product[] = Array.from({ length: 24 }, (_, index) => ({
  id: index + 1,
  name: `商品 ${index + 1}`,
  description: `这是第 ${index + 1} 个商品的详细描述，具有优秀的品质和合理的价格。`,
  category: ['electronics', 'clothing', 'home', 'books'][index % 4],
  price: Math.floor(Math.random() * 1000 + 10),
  originalPrice: Math.floor(Math.random() * 1200 + 15),
  image: `https://picsum.photos/300/300?random=${index + 1}`,
  rating: Math.round((Math.random() * 2 + 3) * 10) / 10,
  reviewCount: Math.floor(Math.random() * 500),
  inStock: Math.random() > 0.1,
  isNew: index % 8 === 0,
  isSale: index % 6 === 0,
  discount: Math.floor(Math.random() * 30 + 10),
  tags: ['热销', '新品', '推荐', '特价'].slice(0, Math.floor(Math.random() * 2) + 1)
}))

export const defaultEcommerceCategories = [
  { id: 'all', name: '全部商品', count: 24 },
  { id: 'electronics', name: '电子产品', count: 6 },
  { id: 'clothing', name: '服装', count: 6 },
  { id: 'home', name: '家居', count: 6 },
  { id: 'books', name: '图书', count: 6 }
]

// 看板布局默认数据
export const defaultKanbanColumns: KanbanColumn[] = [
  {
    id: 'todo',
    title: '待办',
    color: 'bg-gray-100 dark:bg-gray-700',
    items: [
      { id: 1, title: '设计新界面', description: '完成主要页面的UI设计', priority: 'high', assignee: '张三', tags: ['设计', 'UI'] },
      { id: 2, title: '编写文档', description: '更新项目技术文档', priority: 'medium', assignee: '李四', tags: ['文档'] },
      { id: 3, title: '测试功能', description: '对新增功能进行测试', priority: 'high', assignee: '王五', tags: ['测试'] }
    ]
  },
  {
    id: 'in-progress',
    title: '进行中',
    color: 'bg-blue-100 dark:bg-blue-900',
    items: [
      { id: 4, title: '实现用户认证', description: '开发用户登录注册功能', priority: 'medium', assignee: '赵六', tags: ['开发', '认证'] },
      { id: 5, title: '部署应用', description: '将应用部署到生产环境', priority: 'high', assignee: '钱七', tags: ['部署'] }
    ]
  },
  {
    id: 'review',
    title: '审核',
    color: 'bg-yellow-100 dark:bg-yellow-900',
    items: [
      { id: 6, title: '优化性能', description: '提升应用运行性能', priority: 'low', assignee: '孙八', tags: ['优化', '性能'] }
    ]
  },
  {
    id: 'done',
    title: '完成',
    color: 'bg-green-100 dark:bg-green-900',
    items: [
      { id: 7, title: '项目初始化', description: '完成项目基础架构搭建', priority: 'medium', assignee: '周九', tags: ['初始化'] }
    ]
  }
]

// 时间线布局默认数据
export const defaultTimelineEvents: TimelineEvent[] = [
  {
    id: 1,
    title: '项目启动',
    description: '开始新的项目开发',
    date: '2024-01-15',
    time: '09:00',
    author: '张三',
    tags: ['项目', '启动'],
    type: 'mileline'
  },
  {
    id: 2,
    title: '需求分析完成',
    description: '完成了详细的需求分析文档',
    date: '2024-01-20',
    time: '14:30',
    author: '李四',
    tags: ['分析', '文档'],
    type: 'task'
  },
  {
    id: 3,
    title: 'UI设计稿',
    description: '完成了主要页面的UI设计',
    date: '2024-01-25',
    time: '16:45',
    author: '王五',
    tags: ['设计', 'UI'],
    type: 'design'
  },
  {
    id: 4,
    title: '开发环境搭建',
    description: '配置了开发环境和CI/CD流程',
    date: '2024-01-28',
    time: '11:20',
    author: '赵六',
    tags: ['开发', '环境'],
    type: 'development'
  },
  {
    id: 5,
    title: '核心功能开发',
    description: '实现了用户认证和主要业务逻辑',
    date: '2024-02-05',
    time: '15:10',
    author: '钱七',
    tags: ['开发', '功能'],
    type: 'development'
  }
]

// 瀑布流布局默认数据
export const defaultMasonryItems: MasonryItem[] = Array.from({ length: 12 }, (_, index) => ({
  id: index + 1,
  title: `卡片 ${index + 1}`,
  description: `这是瀑布流布局的卡片内容。每个卡片都有不同的高度，创造出自然的瀑布流效果。`,
  image: `https://picsum.photos/400/300?random=${index + 1}`,
  height: Math.floor(Math.random() * 200 + 200),
  category: ['设计', '开发', '创意', '技术'][index % 4],
  tags: ['React', 'TypeScript', 'UI/UX', '设计', '开发', '创新'].slice(0, Math.floor(Math.random() * 3) + 1)
}))

// 仪表板布局默认数据
export const defaultDashboardStats: DashboardStat[] = [
  { title: '总用户数', value: '12,847', change: '+12.5%', trend: 'up' },
  { title: '活跃用户', value: '8,234', change: '+8.2%', trend: 'up' },
  { title: '总收入', value: '¥245,678', change: '+15.3%', trend: 'up' },
  { title: '转化率', value: '3.24%', change: '-2.1%', trend: 'down' }
]

export const defaultDashboardSidebarItems: DashboardSidebarItem[] = [
  { id: 'overview', name: '概览', icon: null as any },
  { id: 'analytics', name: '分析', icon: null as any },
  { id: 'reports', name: '报告', icon: null as any }
]

// 管理后台布局默认数据
export const defaultAdminSidebarItems: AdminSidebarItem[] = [
  { id: 'dashboard', name: '仪表板', icon: null as any, count: undefined },
  { id: 'users', name: '用户管理', icon: null as any, count: 1247 },
  { id: 'analytics', name: '数据分析', icon: null as any, count: undefined },
  { id: 'content', name: '内容管理', icon: null as any, count: 89 },
  { id: 'database', name: '数据库', icon: null as any, count: undefined },
  { id: 'security', name: '安全设置', icon: null as any, count: undefined },
  { id: 'reports', name: '报告', icon: null as any, count: 12 },
  { id: 'monitoring', name: '监控', icon: null as any, count: undefined },
  { id: 'settings', name: '系统设置', icon: null as any, count: undefined }
]

// 杂志布局默认数据
export const defaultMagazineArticles: MagazineArticle[] = [
  {
    id: 1,
    title: '2024年设计趋势预测',
    excerpt: '探索即将到来的设计趋势，从色彩到布局，从交互到体验。',
    image: 'https://picsum.photos/600/400?random=1',
    category: '设计',
    author: '设计团队',
    date: '2024-01-15',
    readTime: '5 分钟',
    featured: true
  },
  {
    id: 2,
    title: '前端开发最佳实践',
    excerpt: '分享前端开发中的最佳实践和常见问题的解决方案。',
    image: 'https://picsum.photos/600/400?random=2',
    category: '技术',
    author: '技术团队',
    date: '2024-01-12',
    readTime: '8 分钟'
  },
  {
    id: 3,
    title: '用户体验设计原则',
    excerpt: '深入探讨用户体验设计的核心原则和实现方法。',
    image: 'https://picsum.photos/600/400?random=3',
    category: 'UX',
    author: 'UX团队',
    date: '2024-01-10',
    readTime: '6 分钟'
  }
]

export const defaultMagazineSidebarItems: MagazineSidebarItem[] = [
  {
    title: '相关文章',
    items: [
      { title: '热门话题', description: '探索最新趋势', color: 'border-blue-500' },
      { title: '技术前沿', description: '了解最新技术', color: 'border-green-500' },
      { title: '设计灵感', description: '获取创意灵感', color: 'border-purple-500' }
    ]
  }
]

export const defaultMagazineTags = ['React', 'TypeScript', 'UI/UX', '设计', '开发', '创新']

// 分割布局默认数据
export const defaultSplitPanels: SplitPanel[] = [
  {
    id: 'left',
    title: '左侧面板',
    content: null,
    width: 50,
    minWidth: 20,
    maxWidth: 80
  },
  {
    id: 'right',
    title: '主内容区域',
    content: null,
    width: 50,
    minWidth: 20,
    maxWidth: 80
  }
]
