import { 
  FunctionTool, 
  FunctionCall, 
  FunctionResult
} from '../types/functionTools'
import { knowledgeService } from './knowledgeService'

class FunctionToolsService {
  private tools: Map<string, FunctionTool> = new Map()
  private callHistory: FunctionCall[] = []

  constructor() {
    this.initializeSystemTools()
  }

  // 初始化系统工具
  private initializeSystemTools() {
    // 知识库搜索工具
    this.registerTool({
      name: 'search_knowledge',
      description: '在系统知识库中搜索相关信息',
      parameters: [
        {
          name: 'query',
          type: 'string',
          description: '搜索查询',
          required: true
        },
        {
          name: 'category',
          type: 'string',
          description: '知识分类（可选）',
          required: false,
          enum: ['react', 'components', 'ai', 'system']
        },
        {
          name: 'limit',
          type: 'number',
          description: '返回结果数量限制',
          required: false
        }
      ],
      handler: this.handleSearchKnowledge.bind(this),
      category: 'knowledge',
      isEnabled: true
    })

    // 获取知识项详情
    this.registerTool({
      name: 'get_knowledge_item',
      description: '获取特定知识项的详细信息',
      parameters: [
        {
          name: 'id',
          type: 'string',
          description: '知识项ID',
          required: true
        }
      ],
      handler: this.handleGetKnowledgeItem.bind(this),
      category: 'knowledge',
      isEnabled: true
    })

    // 获取知识分类
    this.registerTool({
      name: 'get_knowledge_categories',
      description: '获取所有可用的知识分类',
      parameters: [],
      handler: this.handleGetCategories.bind(this),
      category: 'knowledge',
      isEnabled: true
    })

    // 获取组件信息
    this.registerTool({
      name: 'get_component_info',
      description: '获取特定组件的使用信息',
      parameters: [
        {
          name: 'componentName',
          type: 'string',
          description: '组件名称',
          required: true
        }
      ],
      handler: this.handleGetComponentInfo.bind(this),
      category: 'components',
      isEnabled: true
    })

    // 获取可用组件列表
    this.registerTool({
      name: 'get_available_components',
      description: '获取所有可用的UI组件列表',
      parameters: [],
      handler: this.handleGetAvailableComponents.bind(this),
      category: 'components',
      isEnabled: true
    })

    // 获取系统信息
    this.registerTool({
      name: 'get_system_info',
      description: '获取系统基本信息和状态',
      parameters: [],
      handler: this.handleGetSystemInfo.bind(this),
      category: 'system',
      isEnabled: true
    })

    // 获取当前页面信息
    this.registerTool({
      name: 'get_current_page',
      description: '获取当前页面信息',
      parameters: [],
      handler: this.handleGetCurrentPage.bind(this),
      category: 'system',
      isEnabled: true
    })

    // 页面导航
    this.registerTool({
      name: 'navigate_to_page',
      description: '导航到指定页面',
      parameters: [
        {
          name: 'path',
          type: 'string',
          description: '目标页面路径',
          required: true
        }
      ],
      handler: this.handleNavigateToPage.bind(this),
      category: 'navigation',
      isEnabled: true
    })

    // 显示通知
    this.registerTool({
      name: 'show_notification',
      description: '显示系统通知',
      parameters: [
        {
          name: 'message',
          type: 'string',
          description: '通知消息',
          required: true
        },
        {
          name: 'type',
          type: 'string',
          description: '通知类型',
          required: false,
          enum: ['info', 'success', 'warning', 'error']
        }
      ],
      handler: this.handleShowNotification.bind(this),
      category: 'ui',
      isEnabled: true
    })
  }

  // 注册工具
  registerTool(tool: FunctionTool) {
    this.tools.set(tool.name, tool)
  }

  // 获取所有工具
  getAvailableTools(): FunctionTool[] {
    return Array.from(this.tools.values()).filter(tool => tool.isEnabled)
  }

  // 获取工具定义（用于AI调用）
  getToolsForAI(): any[] {
    return this.getAvailableTools().map(tool => ({
      type: 'function',
      function: {
        name: tool.name,
        description: tool.description,
        parameters: {
          type: 'object',
          properties: this.buildParametersSchema(tool.parameters),
          required: tool.parameters.filter(p => p.required).map(p => p.name)
        }
      }
    }))
  }

  // 构建参数模式
  private buildParametersSchema(parameters: any[]) {
    const properties: any = {}
    parameters.forEach(param => {
      properties[param.name] = {
        type: param.type,
        description: param.description
      }
      if (param.enum) {
        properties[param.name].enum = param.enum
      }
    })
    return properties
  }

  // 执行函数调用
  async executeFunctionCall(call: FunctionCall): Promise<FunctionResult> {
    const tool = this.tools.get(call.name)
    if (!tool) {
      return {
        success: false,
        error: `Function ${call.name} not found`
      }
    }

    // 记录调用历史
    this.callHistory.push(call)

    try {
      const result = await tool.handler(call.arguments)
      return result
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Unknown error'
      }
    }
  }

  // 知识库搜索处理
  private async handleSearchKnowledge(args: Record<string, any>): Promise<FunctionResult> {
    try {
      const results = await knowledgeService.searchKnowledge({
        query: args.query,
        category: args.category,
        limit: args.limit || 5
      })

      return {
        success: true,
        data: results,
        message: `找到 ${results.length} 个相关结果`
      }
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : '搜索失败'
      }
    }
  }

  // 获取知识项处理
  private async handleGetKnowledgeItem(args: Record<string, any>): Promise<FunctionResult> {
    try {
      const item = knowledgeService.getKnowledgeItem(args.id)
      if (!item) {
        return {
          success: false,
          error: '知识项不存在'
        }
      }

      return {
        success: true,
        data: item,
        message: `获取知识项: ${item.title}`
      }
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : '获取失败'
      }
    }
  }

  // 获取分类处理
  private async handleGetCategories(_args: Record<string, any>): Promise<FunctionResult> {
    try {
      const categories = knowledgeService.getCategories()
      return {
        success: true,
        data: categories,
        message: `获取到 ${categories.length} 个分类`
      }
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : '获取失败'
      }
    }
  }

  // 获取组件信息处理
  private async handleGetComponentInfo(args: Record<string, any>): Promise<FunctionResult> {
    const componentName = args.componentName
    
    // 完整的组件信息库
    const componentInfo: Record<string, any> = {
      'Button': {
        name: 'Button',
        description: '按钮组件，支持多种样式和状态',
        importPath: '@/components/UI/Button',
        props: {
          variant: { type: 'string', options: ['primary', 'secondary', 'outline', 'ghost'], default: 'primary' },
          size: { type: 'string', options: ['sm', 'md', 'lg'], default: 'md' },
          disabled: { type: 'boolean', default: false },
          onClick: { type: 'function', required: false }
        },
        examples: [
          '<Button variant="primary">主要按钮</Button>',
          '<Button variant="outline" size="sm">小按钮</Button>',
          '<Button disabled>禁用按钮</Button>'
        ],
        usage: 'import Button from "@/components/UI/Button"'
      },
      'Input': {
        name: 'Input',
        description: '输入框组件',
        importPath: '@/components/UI/Input',
        props: {
          type: { type: 'string', options: ['text', 'password', 'email', 'number'], default: 'text' },
          placeholder: { type: 'string', required: false },
          value: { type: 'string', required: false },
          onChange: { type: 'function', required: false },
          disabled: { type: 'boolean', default: false }
        },
        examples: [
          '<Input placeholder="请输入内容" />',
          '<Input type="password" placeholder="密码" />',
          '<Input value={value} onChange={handleChange} />'
        ],
        usage: 'import Input from "@/components/UI/Input"'
      },
      'Card': {
        name: 'Card',
        description: '卡片容器组件',
        importPath: '@/components/UI/Card',
        props: {
          className: { type: 'string', required: false },
          children: { type: 'ReactNode', required: true }
        },
        examples: [
          '<Card><CardContent>卡片内容</CardContent></Card>',
          '<Card className="p-4">自定义卡片</Card>'
        ],
        usage: 'import { Card, CardContent, CardHeader, CardTitle } from "@/components/UI/Card"'
      },
      'Modal': {
        name: 'Modal',
        description: '模态框组件',
        importPath: '@/components/UI/Modal',
        props: {
          isOpen: { type: 'boolean', required: true },
          onClose: { type: 'function', required: true },
          title: { type: 'string', required: false },
          children: { type: 'ReactNode', required: true }
        },
        examples: [
          '<Modal isOpen={isOpen} onClose={() => setIsOpen(false)}>内容</Modal>'
        ],
        usage: 'import Modal from "@/components/UI/Modal"'
      }
    }

    // 支持大小写不敏感的查找
    const normalizedName = Object.keys(componentInfo).find(
      key => key.toLowerCase() === componentName.toLowerCase()
    )

    if (!normalizedName) {
      return {
        success: false,
        error: `组件 "${componentName}" 不存在。可用组件: ${Object.keys(componentInfo).join(', ')}`
      }
    }

    const info = componentInfo[normalizedName]

    return {
      success: true,
      data: info,
      message: `获取组件信息: ${info.name}`
    }
  }

  // 获取可用组件处理
  private async handleGetAvailableComponents(_args: Record<string, any>): Promise<FunctionResult> {
    const components = [
      'Button', 'Input', 'Card', 'Modal', 'Tabs', 'Switch', 'Slider', 'Select',
      'Badge', 'Avatar', 'Tooltip', 'Popover', 'Dialog', 'Alert', 'Progress',
      'TypewriterMarkdown', 'MarkdownRenderer', 'SimpleSelect', 'SimpleTabs'
    ]

    return {
      success: true,
      data: components,
      message: `系统提供 ${components.length} 个UI组件，包括基础组件和自定义组件`
    }
  }

  // 获取系统信息处理
  private async handleGetSystemInfo(_args: Record<string, any>): Promise<FunctionResult> {
    const systemInfo = {
      name: 'SoulNexus 灵枢',
      version: '1.0.0',
      description: '基于React的现代化Web应用框架',
      features: [
        'AI智能助手',
        '组件库',
        '缓存系统',
        'PWA支持',
        '主题切换',
        '响应式设计'
      ],
      technologies: ['React', 'TypeScript', 'Tailwind CSS', 'Framer Motion', 'Zustand']
    }

    return {
      success: true,
      data: systemInfo,
      message: '获取系统信息成功'
    }
  }

  // 获取当前页面处理
  private async handleGetCurrentPage(_args: Record<string, any>): Promise<FunctionResult> {
    const currentPage = {
      path: window.location.pathname,
      title: document.title,
      timestamp: new Date().toISOString()
    }

    return {
      success: true,
      data: currentPage,
      message: '获取当前页面信息成功'
    }
  }

  // 页面导航处理
  private async handleNavigateToPage(args: Record<string, any>): Promise<FunctionResult> {
    try {
      window.location.href = args.path
      return {
        success: true,
        message: `正在导航到: ${args.path}`
      }
    } catch (error) {
      return {
        success: false,
        error: '导航失败'
      }
    }
  }

  // 显示通知处理
  private async handleShowNotification(args: Record<string, any>): Promise<FunctionResult> {
    // 这里可以集成实际的通知系统
    console.log(`通知 [${args.type || 'info'}]: ${args.message}`)
    
    return {
      success: true,
      message: '通知已显示'
    }
  }

  // 获取调用历史
  getCallHistory(): FunctionCall[] {
    return this.callHistory
  }

  // 清除调用历史
  clearCallHistory() {
    this.callHistory = []
  }
}

export const functionToolsService = new FunctionToolsService()
