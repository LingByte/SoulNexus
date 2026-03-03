// MCP 类型定义

export type MCPType = 'builtin' | 'external';
export type MCPStatus = 'running' | 'stopped' | 'error';
export type CallStatus = 'success' | 'error';

export interface MCPServer {
  id: number;
  name: string;
  description: string;
  version: string;
  type: MCPType;
  source: string;
  enabled: boolean;
  status: MCPStatus;
  lastError: string;
  toolCount: number;
  createdAt: string;
  updatedAt: string;
}

export interface MCPServerDetail extends MCPServer {
  tools?: MCPTool[];
}

export interface MCPTool {
  id: number;
  name: string;
  description: string;
  inputSchema: Record<string, any>;
  category: string;
  enabled: boolean;
}

export interface MCPCallLog {
  id: number;
  toolName: string;
  arguments: Record<string, any>;
  result?: Record<string, any>;
  error?: string;
  duration: number;
  status: CallStatus;
  createdAt: string;
}

export interface CreateMCPServerRequest {
  name: string;
  description?: string;
  version?: string;
  type: MCPType;
  source?: string;
  command?: string;
  args?: string[];
  env?: Record<string, string>;
  autoApprove?: string[];
  config?: Record<string, any>;
}

export interface UpdateMCPServerRequest {
  name?: string;
  description?: string;
  command?: string;
  args?: string[];
  env?: Record<string, string>;
  autoApprove?: string[];
  config?: Record<string, any>;
}

export interface CallMCPToolRequest {
  toolName: string;
  arguments?: Record<string, any>;
}

export interface CallMCPToolResponse {
  success: boolean;
  result?: Record<string, any>;
  error?: string;
}

// MCP 市场相关类型

export interface MCPMarketplaceItem {
  id: string;
  name: string;
  description: string;
  version: string;
  author: string;
  repository: string;
  downloads: number;
  rating: number;
  tags: string[];
  icon?: string;
  featured?: boolean;
}

export interface MCPMarketplaceResponse {
  code: number;
  data: MCPMarketplaceItem[];
  total: number;
  page: number;
  pageSize: number;
}

// 状态管理相关

export interface MCPState {
  servers: MCPServer[];
  selectedServer: MCPServerDetail | null;
  loading: boolean;
  error: string | null;
  logs: MCPCallLog[];
}
