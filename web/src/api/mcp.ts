import { get, post, patch, del } from '@/utils/request';

// 列出 MCP 服务器
export const listMCPServers = async () => {
  return get<any>('/mcp/servers');
};

// 获取 MCP 服务器详情
export const getMCPServer = async (id: number) => {
  return get<any>(`/mcp/servers/${id}`);
};

// 创建 MCP 服务器
export const createMCPServer = async (data: any) => {
  return post<any>('/mcp/servers', data);
};

// 更新 MCP 服务器配置
export const updateMCPServer = async (id: number, data: any) => {
  return patch<any>(`/mcp/servers/${id}`, data);
};

// 删除 MCP 服务器
export const deleteMCPServer = async (id: number) => {
  return del<any>(`/mcp/servers/${id}`);
};

// 启用 MCP 服务器
export const enableMCPServer = async (id: number) => {
  return post<any>(`/mcp/servers/${id}/enable`);
};

// 禁用 MCP 服务器
export const disableMCPServer = async (id: number) => {
  return post<any>(`/mcp/servers/${id}/disable`);
};

// 获取 MCP 工具列表
export const getMCPTools = async (id: number) => {
  return get<any>(`/mcp/servers/${id}/tools`);
};

// 调用 MCP 工具
export const callMCPTool = async (id: number, data: any) => {
  return post<any>(
    `/mcp/servers/${id}/call-tool`,
    data
  );
};

// 获取 MCP 调用日志
export const getMCPLogs = async (id: number, limit: number = 100) => {
  return get<any>(
    `/mcp/servers/${id}/logs?limit=${limit}`
  );
};

// ============ MCP 广场 API ============

// 列出广场项目
export const listMarketplace = async (params: {
  query?: string;
  category?: string;
  sort_by?: string;
  page?: number;
  page_size?: number;
}) => {
  return get<any>('/mcp/marketplace', { params });
};

// 获取广场项目详情
export const getMarketplaceItem = async (id: number) => {
  return get<any>(`/mcp/marketplace/${id}`);
};

// 安装 MCP
export const installMCP = async (id: number, config: any = {}) => {
  return post<any>(`/mcp/marketplace/${id}/install`, { config });
};

// 卸载 MCP
export const uninstallMCP = async (installationId: number) => {
  return del<any>(`/mcp/marketplace/installations/${installationId}`);
};

// 获取用户已安装的 MCP
export const getUserInstalledMCPs = async () => {
  return get<any>('/mcp/marketplace/my-installations');
};

// 更新安装配置
export const updateInstallationConfig = async (installationId: number, config: any) => {
  return patch<any>(`/mcp/marketplace/installations/${installationId}/config`, config);
};

// 评论 MCP
export const reviewMCP = async (id: number, data: { rating: number; title: string; content: string }) => {
  return post<any>(`/mcp/marketplace/${id}/reviews`, data);
};

// 获取 MCP 评论
export const getMCPReviews = async (id: number, page: number = 1, pageSize: number = 20) => {
  return get<any>(`/mcp/marketplace/${id}/reviews`, {
    params: { page, page_size: pageSize },
  });
};

// 获取分类列表
export const getCategories = async () => {
  return get<any>('/mcp/marketplace/categories');
};

// 获取推荐的 MCP
export const getFeaturedMCPs = async (limit: number = 10) => {
  return get<any>('/mcp/marketplace/featured', { params: { limit } });
};

// 获取热门 MCP
export const getTrendingMCPs = async (limit: number = 10) => {
  return get<any>('/mcp/marketplace/trending', { params: { limit } });
};

// 按标签搜索
export const searchByTag = async (tag: string, limit: number = 20) => {
  return get<any>(`/mcp/marketplace/search/tag/${tag}`, { params: { limit } });
};

// 获取 MCP 市场列表（模拟数据）
export const getMCPMarketplace = async (
  _type: 'official' | 'community' = 'official',
  _search: string = '',
  page: number = 1,
  pageSize: number = 20
) => {
  // 这里可以连接到真实的 MCP 市场 API
  // 暂时返回模拟数据
  const mockData: any = {
    code: 200,
    data: [
      {
        id: '1',
        name: 'Database Query MCP',
        description: '提供数据库查询工具，支持 SQL 查询、数据导出等功能',
        version: '1.0.0',
        author: 'LingEcho Team',
        repository: 'https://github.com/lingecho/mcp-database',
        downloads: 1250,
        rating: 4.8,
        tags: ['database', 'query', 'sql'],
        featured: true,
      },
      {
        id: '2',
        name: 'File Operations MCP',
        description: '提供文件操作工具，支持读写、列表、删除等操作',
        version: '1.0.0',
        author: 'LingEcho Team',
        repository: 'https://github.com/lingecho/mcp-file',
        downloads: 980,
        rating: 4.6,
        tags: ['file', 'filesystem', 'io'],
        featured: true,
      },
      {
        id: '3',
        name: 'HTTP API MCP',
        description: '提供 HTTP 请求工具，支持 GET、POST、PUT、DELETE 等方法',
        version: '1.0.0',
        author: 'LingEcho Team',
        repository: 'https://github.com/lingecho/mcp-http',
        downloads: 750,
        rating: 4.5,
        tags: ['http', 'api', 'rest'],
        featured: true,
      },
      {
        id: '4',
        name: 'Weather API MCP',
        description: '提供天气查询工具，支持实时天气、预报等功能',
        version: '1.0.0',
        author: 'Community',
        repository: 'https://github.com/community/mcp-weather',
        downloads: 520,
        rating: 4.3,
        tags: ['weather', 'api', 'forecast'],
      },
      {
        id: '5',
        name: 'Email Sender MCP',
        description: '提供邮件发送工具，支持 SMTP、模板等功能',
        version: '1.0.0',
        author: 'Community',
        repository: 'https://github.com/community/mcp-email',
        downloads: 380,
        rating: 4.2,
        tags: ['email', 'smtp', 'notification'],
      },
      {
        id: '6',
        name: 'Redis Cache MCP',
        description: '提供 Redis 缓存工具，支持 GET、SET、DEL 等操作',
        version: '1.0.0',
        author: 'Community',
        repository: 'https://github.com/community/mcp-redis',
        downloads: 290,
        rating: 4.4,
        tags: ['redis', 'cache', 'database'],
      },
    ],
    total: 6,
    page,
    pageSize,
  };

  return Promise.resolve(mockData);
};

// 获取内置 MCP 列表
export const getBuiltinMCPs = async () => {
  const builtins = [
    {
      id: 'builtin-database',
      name: 'Database Query MCP',
      description: '提供数据库查询工具',
      version: '1.0.0',
      type: 'builtin' as const,
      enabled: true,
      status: 'running' as const,
      toolCount: 4,
    },
    {
      id: 'builtin-file',
      name: 'File Operations MCP',
      description: '提供文件操作工具',
      version: '1.0.0',
      type: 'builtin' as const,
      enabled: true,
      status: 'running' as const,
      toolCount: 4,
    },
    {
      id: 'builtin-http',
      name: 'HTTP API MCP',
      description: '提供 HTTP 请求工具',
      version: '1.0.0',
      type: 'builtin' as const,
      enabled: true,
      status: 'running' as const,
      toolCount: 4,
    },
  ];

  return Promise.resolve({ code: 200, data: builtins });
};

// 导出所有 API 方法
export const mcpApi = {
  listMCPServers,
  getMCPServer,
  createMCPServer,
  updateMCPServer,
  deleteMCPServer,
  enableMCPServer,
  disableMCPServer,
  getMCPTools,
  callMCPTool,
  getMCPLogs,
  listMarketplace,
  getMarketplaceItem,
  installMCP,
  uninstallMCP,
  getUserInstalledMCPs,
  updateInstallationConfig,
  reviewMCP,
  getMCPReviews,
  getCategories,
  getFeaturedMCPs,
  getTrendingMCPs,
  searchByTag,
  getMCPMarketplace,
  getBuiltinMCPs,
};
