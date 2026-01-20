// Function Tools类型定义
export interface FunctionTool {
  name: string
  description: string
  parameters: FunctionParameter[]
  handler: FunctionToolHandler
  category: string
  isEnabled: boolean
}

export interface FunctionParameter {
  name: string
  type: 'string' | 'number' | 'boolean' | 'object' | 'array'
  description: string
  required: boolean
  enum?: string[]
}

export interface FunctionCall {
  name: string
  arguments: Record<string, any>
  id: string
  timestamp: string
}

export interface FunctionResult {
  success: boolean
  data?: any
  error?: string
  message?: string
}

export type FunctionToolHandler = (args: Record<string, any>) => Promise<FunctionResult>

// 系统Function Tools
export interface SystemFunctionTools {
  // 知识库相关
  searchKnowledge: FunctionTool
  getKnowledgeItem: FunctionTool
  getCategories: FunctionTool
  
  // 组件相关
  getComponentInfo: FunctionTool
  getAvailableComponents: FunctionTool
  
  // 系统信息
  getSystemInfo: FunctionTool
  getCurrentPage: FunctionTool
  
  // 用户操作
  navigateToPage: FunctionTool
  showNotification: FunctionTool
}
