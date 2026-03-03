import { useState, useEffect } from 'react'
import { Plus, Trash2, Power, PowerOff, Settings, X, TestTube } from 'lucide-react'
import Button from '@/components/UI/Button'
import { listMCPServers, deleteMCPServer, enableMCPServer, disableMCPServer, createMCPServer, updateMCPServer, getUserInstalledMCPs, uninstallMCP, updateInstallationConfig, getMCPTools, callMCPTool } from '@/api/mcp'

interface MCPServer {
  id: number
  name: string
  description: string
  type: string
  status: string
  enabled: boolean
  command: string
  protocol: string
  host?: string
  port?: number
  usage_count: number
  error_count: number
  last_used_at?: string
  created_at: string
  source?: 'manual' | 'marketplace'
  installation_id?: number
}

interface AddServerForm {
  name: string
  description: string
  type: string
  protocol: string
  host: string
  port: number
  command: string
}

interface SettingsForm {
  name: string
  description: string
  enabled: boolean
}

interface MCPTool {
  id: number
  name: string
  description: string
  input_schema?: any
}

const MCPManagement = () => {
  const [servers, setServers] = useState<MCPServer[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAddForm, setShowAddForm] = useState(false)
  const [showSettingsForm, setShowSettingsForm] = useState(false)
  const [showTestForm, setShowTestForm] = useState(false)
  const [selectedServer, setSelectedServer] = useState<MCPServer | null>(null)
  const [tools, setTools] = useState<MCPTool[]>([])
  const [selectedTool, setSelectedTool] = useState<MCPTool | null>(null)
  const [toolArguments, setToolArguments] = useState<string>('{}')
  const [testResult, setTestResult] = useState<any>(null)
  const [testLoading, setTestLoading] = useState(false)
  const [formData, setFormData] = useState<AddServerForm>({
    name: '',
    description: '',
    type: 'remote',
    protocol: 'http',
    host: 'localhost',
    port: 8000,
    command: '',
  })
  const [settingsData, setSettingsData] = useState<SettingsForm>({
    name: '',
    description: '',
    enabled: true,
  })
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    fetchServers()
  }, [])

  const fetchServers = async () => {
    try {
      setLoading(true)
      // 获取手动添加的服务器
      const res = await listMCPServers()
      let allServers: MCPServer[] = []
      
      if (res.code === 200) {
        allServers = (res.data || []).map((server: any) => ({
          ...server,
          source: 'manual'
        }))
      }

      // 获取从广场安装的 MCP
      try {
        const marketplaceRes = await getUserInstalledMCPs()
        if (marketplaceRes.code === 200 && marketplaceRes.data) {
          const marketplaceServers = marketplaceRes.data.map((installation: any) => ({
            // 使用负数ID来避免与MCPServer ID冲突
            id: -(installation.id),
            name: installation.marketplace?.name || 'Unknown',
            description: installation.marketplace?.description || '',
            type: 'marketplace',
            status: installation.enabled ? 'active' : 'inactive',
            enabled: installation.enabled,
            command: '',
            protocol: 'http',
            usage_count: installation.usage_count || 0,
            error_count: 0,
            created_at: installation.created_at,
            source: 'marketplace',
            installation_id: installation.id,
          }))
          allServers = [...allServers, ...marketplaceServers]
        }
      } catch (err) {
        console.error('Failed to fetch marketplace installations', err)
      }

      setServers(allServers)
    } catch (err) {
      setError('Failed to load MCP servers')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (id: number, server: MCPServer) => {
    if (!confirm('Are you sure you want to delete this MCP server?')) return
    try {
      if (server.source === 'marketplace' && server.installation_id) {
        // 从广场卸载
        await uninstallMCP(server.installation_id)
      } else {
        // 删除手动添加的服务器
        await deleteMCPServer(id)
      }
      setServers(servers.filter(s => s.id !== id))
    } catch (err) {
      setError('Failed to delete MCP server')
      console.error(err)
    }
  }

  const handleToggle = async (id: number, enabled: boolean, server: MCPServer) => {
    try {
      if (server.source === 'marketplace' && server.installation_id) {
        // 更新广场安装的 MCP 的启用状态
        await updateInstallationConfig(server.installation_id, { enabled: !enabled })
      } else {
        // 更新手动添加的服务器
        if (enabled) {
          await disableMCPServer(id)
        } else {
          await enableMCPServer(id)
        }
      }
      setServers(servers.map(s => s.id === id ? { ...s, enabled: !enabled } : s))
    } catch (err) {
      setError('Failed to toggle MCP server')
      console.error(err)
    }
  }

  const handleAddServer = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!formData.name.trim()) {
      setError('Server name is required')
      return
    }

    try {
      setSubmitting(true)
      const res = await createMCPServer({
        name: formData.name,
        description: formData.description,
        type: formData.type,
        protocol: formData.protocol,
        host: formData.host,
        port: formData.port,
        command: formData.command,
      })

      if (res.code === 200) {
        setServers([...servers, res.data])
        setShowAddForm(false)
        setFormData({
          name: '',
          description: '',
          type: 'remote',
          protocol: 'http',
          host: 'localhost',
          port: 8000,
          command: '',
        })
        setError(null)
      } else {
        setError(res.msg || 'Failed to add MCP server')
      }
    } catch (err) {
      setError('Failed to add MCP server')
      console.error(err)
    } finally {
      setSubmitting(false)
    }
  }

  const handleFormChange = (field: keyof AddServerForm, value: any) => {
    setFormData(prev => ({
      ...prev,
      [field]: value,
    }))
  }

  const handleOpenSettings = (server: MCPServer) => {
    setSelectedServer(server)
    setSettingsData({
      name: server.name,
      description: server.description,
      enabled: server.enabled,
    })
    setShowSettingsForm(true)
  }

  const handleOpenTest = async (server: MCPServer) => {
    setSelectedServer(server)
    setShowTestForm(true)
    setTools([])
    setSelectedTool(null)
    setTestResult(null)
    setToolArguments('{}')
    
    // 获取工具列表
    try {
      const res = await getMCPTools(server.source === 'marketplace' ? -(server.id) : server.id)
      if (res.code === 200) {
        const toolsData = res.data || []
        // 确保每个工具都有唯一的ID
        const toolsWithIds = toolsData.map((tool: any, index: number) => ({
          ...tool,
          id: tool.id || index,
        }))
        setTools(toolsWithIds)
        console.log('Loaded tools:', toolsWithIds)
      }
    } catch (err) {
      setError('Failed to fetch MCP tools')
      console.error(err)
    }
  }

  const generateExampleFromSchema = (schema: any): any => {
    if (!schema) return {}
    
    const example: any = {}
    
    if (schema.properties) {
      Object.entries(schema.properties).forEach(([key, prop]: [string, any]) => {
        // 优先使用 default 值，其次使用 description，最后使用类型默认值
        if (prop.default !== undefined) {
          example[key] = prop.default
        } else if (prop.type === 'string') {
          example[key] = prop.description || `example_${key}`
        } else if (prop.type === 'number' || prop.type === 'integer') {
          example[key] = 0
        } else if (prop.type === 'boolean') {
          example[key] = false
        } else if (prop.type === 'array') {
          example[key] = []
        } else if (prop.type === 'object') {
          example[key] = {}
        } else {
          example[key] = null
        }
      })
    }
    
    return example
  }

  const handleTestTool = async () => {
    if (!selectedServer || !selectedTool) return
    
    try {
      setTestLoading(true)
      let args = {}
      try {
        args = JSON.parse(toolArguments)
      } catch (e) {
        setError('Invalid JSON arguments')
        return
      }

      const serverId = selectedServer.source === 'marketplace' ? -(selectedServer.id) : selectedServer.id
      const res = await callMCPTool(serverId, {
        tool_name: selectedTool.name,
        arguments: args,
      })

      if (res.code === 200) {
        setTestResult(res.data)
        setError(null)
      } else {
        setError(res.msg || 'Failed to call tool')
      }
    } catch (err) {
      setError('Failed to call MCP tool')
      console.error(err)
    } finally {
      setTestLoading(false)
    }
  }

  const handleSettingsChange = (field: keyof SettingsForm, value: any) => {
    setSettingsData(prev => ({
      ...prev,
      [field]: value,
    }))
  }

  const handleSaveSettings = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedServer) return

    try {
      setSubmitting(true)
      const res = await updateMCPServer(selectedServer.id, {
        name: settingsData.name,
        description: settingsData.description,
        enabled: settingsData.enabled,
      })

      if (res.code === 200) {
        setServers(servers.map(s => 
          s.id === selectedServer.id 
            ? { ...s, ...settingsData }
            : s
        ))
        setShowSettingsForm(false)
        setSelectedServer(null)
        setError(null)
      } else {
        setError(res.msg || 'Failed to update settings')
      }
    } catch (err) {
      setError('Failed to update settings')
      console.error(err)
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto mb-4"></div>
          <p className="text-muted-foreground">Loading MCP servers...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">MCP Management</h1>
          <p className="text-muted-foreground mt-1">Manage your MCP servers and configurations</p>
        </div>
        <Button 
          variant="primary" 
          leftIcon={<Plus className="w-4 h-4" />}
          onClick={() => setShowAddForm(true)}
        >
          Add MCP Server
        </Button>
      </div>

      {/* Error Message */}
      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4 text-red-700 dark:text-red-400">
          {error}
        </div>
      )}

      {/* Add Server Form Modal */}
      {showAddForm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 overflow-y-auto">
          <div className="bg-card border border-border rounded-lg p-6 w-full max-w-md shadow-lg my-8 max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-bold">Add MCP Server</h2>
              <button
                onClick={() => setShowAddForm(false)}
                className="text-muted-foreground hover:text-foreground"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <form onSubmit={handleAddServer} className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1">Server Name *</label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => handleFormChange('name', e.target.value)}
                  placeholder="e.g., SoulMCP"
                  className="w-full px-3 py-2 border border-border rounded-lg bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                />
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">Description</label>
                <input
                  type="text"
                  value={formData.description}
                  onChange={(e) => handleFormChange('description', e.target.value)}
                  placeholder="Server description"
                  className="w-full px-3 py-2 border border-border rounded-lg bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-1">Type</label>
                  <select
                    value={formData.type}
                    onChange={(e) => handleFormChange('type', e.target.value)}
                    className="w-full px-3 py-2 border border-border rounded-lg bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  >
                    <option value="remote">Remote</option>
                    <option value="builtin">Builtin</option>
                    <option value="custom">Custom</option>
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">Protocol</label>
                  <select
                    value={formData.protocol}
                    onChange={(e) => handleFormChange('protocol', e.target.value)}
                    className="w-full px-3 py-2 border border-border rounded-lg bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  >
                    <option value="http">HTTP</option>
                    <option value="stdio">Stdio</option>
                    <option value="sse">SSE</option>
                  </select>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-1">Host</label>
                  <input
                    type="text"
                    value={formData.host}
                    onChange={(e) => handleFormChange('host', e.target.value)}
                    placeholder="localhost"
                    className="w-full px-3 py-2 border border-border rounded-lg bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">Port</label>
                  <input
                    type="number"
                    value={formData.port}
                    onChange={(e) => handleFormChange('port', parseInt(e.target.value))}
                    placeholder="8000"
                    className="w-full px-3 py-2 border border-border rounded-lg bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">Command</label>
                <input
                  type="text"
                  value={formData.command}
                  onChange={(e) => handleFormChange('command', e.target.value)}
                  placeholder="e.g., go run cmd/http-server/main.go"
                  className="w-full px-3 py-2 border border-border rounded-lg bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                />
              </div>

              <div className="flex gap-2 pt-4">
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => setShowAddForm(false)}
                  className="flex-1"
                >
                  Cancel
                </Button>
                <Button
                  type="submit"
                  variant="primary"
                  disabled={submitting}
                  className="flex-1"
                >
                  {submitting ? 'Adding...' : 'Add Server'}
                </Button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Settings Modal */}
      {showSettingsForm && selectedServer && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 overflow-y-auto">
          <div className="bg-card border border-border rounded-lg p-6 w-full max-w-md shadow-lg my-8 max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-bold">Server Settings</h2>
              <button
                onClick={() => {
                  setShowSettingsForm(false)
                  setSelectedServer(null)
                }}
                className="text-muted-foreground hover:text-foreground"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <form onSubmit={handleSaveSettings} className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1">Server Name</label>
                <input
                  type="text"
                  value={settingsData.name}
                  onChange={(e) => handleSettingsChange('name', e.target.value)}
                  placeholder="Server name"
                  className="w-full px-3 py-2 border border-border rounded-lg bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                />
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">Description</label>
                <textarea
                  value={settingsData.description}
                  onChange={(e) => handleSettingsChange('description', e.target.value)}
                  placeholder="Server description"
                  rows={3}
                  className="w-full px-3 py-2 border border-border rounded-lg bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                />
              </div>

              <div className="flex items-center gap-3">
                <input
                  type="checkbox"
                  id="enabled"
                  checked={settingsData.enabled}
                  onChange={(e) => handleSettingsChange('enabled', e.target.checked)}
                  className="w-4 h-4 rounded border-border"
                />
                <label htmlFor="enabled" className="text-sm font-medium">
                  Enable this server
                </label>
              </div>

              <div className="bg-muted/50 rounded-lg p-3 text-sm text-muted-foreground">
                <p><strong>Server ID:</strong> {selectedServer.id}</p>
                <p><strong>Type:</strong> {selectedServer.type}</p>
                <p><strong>Protocol:</strong> {selectedServer.protocol}</p>
                <p><strong>Host:</strong> {selectedServer.host}</p>
                <p><strong>Port:</strong> {selectedServer.port}</p>
              </div>

              <div className="flex gap-2 pt-4">
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => {
                    setShowSettingsForm(false)
                    setSelectedServer(null)
                  }}
                  className="flex-1"
                >
                  Cancel
                </Button>
                <Button
                  type="submit"
                  variant="primary"
                  disabled={submitting}
                  className="flex-1"
                >
                  {submitting ? 'Saving...' : 'Save Settings'}
                </Button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Test Modal */}
      {showTestForm && selectedServer && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 overflow-y-auto">
          <div className="bg-card border border-border rounded-lg p-6 w-full max-w-2xl shadow-lg my-8 max-h-[90vh] overflow-y-auto flex flex-col">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-bold">Test MCP: {selectedServer.name}</h2>
              <button
                onClick={() => {
                  setShowTestForm(false)
                  setSelectedServer(null)
                  setTools([])
                  setSelectedTool(null)
                  setTestResult(null)
                }}
                className="text-muted-foreground hover:text-foreground"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="space-y-4 overflow-y-auto flex-1">
              {/* Tools List */}
              <div>
                <label className="block text-sm font-medium mb-2">Available Tools</label>
                {tools.length === 0 ? (
                  <p className="text-sm text-muted-foreground">No tools available</p>
                ) : (
                  <select
                    value={selectedTool?.id || ''}
                    onChange={(e) => {
                      const toolId = parseInt(e.target.value)
                      const tool = tools.find(t => t.id === toolId)
                      if (tool) {
                        setSelectedTool(tool)
                        setTestResult(null)
                        // 清除之前的参数
                        setToolArguments('{}')
                        console.log('Selected tool:', tool)
                      }
                    }}
                    className="w-full px-3 py-2 border border-border rounded-lg bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  >
                    <option value="">Select a tool...</option>
                    {tools.map(tool => (
                      <option key={tool.id} value={tool.id}>
                        {tool.name}
                      </option>
                    ))}
                  </select>
                )}
              </div>

              {/* Tool Description */}
              {selectedTool && (
                <div className="bg-muted/50 rounded-lg p-3 space-y-2">
                  <p className="text-sm"><strong>Description:</strong> {selectedTool.description}</p>
                  {selectedTool.input_schema && (
                    <div>
                      <p className="text-sm font-medium">Input Schema:</p>
                      <pre className="text-xs overflow-auto max-h-32 bg-background rounded p-2 border border-border">
                        {typeof selectedTool.input_schema === 'string'
                          ? selectedTool.input_schema
                          : JSON.stringify(selectedTool.input_schema, null, 2)}
                      </pre>
                    </div>
                  )}
                </div>
              )}

              {/* Arguments Input */}
              {selectedTool && (
                <div>
                  <div className="flex items-center justify-between mb-2">
                    <label className="block text-sm font-medium">Arguments (JSON)</label>
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        if (selectedTool && selectedTool.input_schema) {
                          try {
                            const schema = typeof selectedTool.input_schema === 'string' 
                              ? JSON.parse(selectedTool.input_schema)
                              : selectedTool.input_schema
                            const example = generateExampleFromSchema(schema)
                            setToolArguments(JSON.stringify(example, null, 2))
                          } catch (e) {
                            console.error('Failed to parse schema:', e)
                            setToolArguments('{}')
                          }
                        } else {
                          setToolArguments('{}')
                        }
                      }}
                    >
                      Generate Example
                    </Button>
                  </div>
                  <textarea
                    value={toolArguments}
                    onChange={(e) => setToolArguments(e.target.value)}
                    placeholder='{}'
                    rows={4}
                    className="w-full px-3 py-2 border border-border rounded-lg bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary font-mono text-xs"
                  />
                </div>
              )}

              {/* Test Result */}
              {testResult && (
                <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-3">
                  <p className="text-sm font-medium text-green-700 dark:text-green-400 mb-2">Result:</p>
                  <pre className="text-xs overflow-auto max-h-48 text-green-700 dark:text-green-400">
                    {JSON.stringify(testResult, null, 2)}
                  </pre>
                </div>
              )}

              {/* Action Buttons */}
              <div className="flex gap-2 pt-4">
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => {
                    setShowTestForm(false)
                    setSelectedServer(null)
                    setTools([])
                    setSelectedTool(null)
                    setTestResult(null)
                  }}
                  className="flex-1"
                >
                  Close
                </Button>
                <Button
                  type="button"
                  variant="primary"
                  disabled={!selectedTool || testLoading}
                  onClick={handleTestTool}
                  className="flex-1"
                >
                  {testLoading ? 'Testing...' : 'Test Tool'}
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}

      {servers.length === 0 ? (
        <div className="text-center py-12 bg-muted/50 rounded-lg">
          <p className="text-muted-foreground mb-4">No MCP servers installed yet</p>
          <Button variant="primary" leftIcon={<Plus className="w-4 h-4" />}>
            Install Your First MCP Server
          </Button>
        </div>
      ) : (
        <div className="grid gap-4">
          {servers.map(server => (
            <div
              key={server.id}
              className="bg-card border border-border rounded-lg p-4 hover:shadow-md transition-shadow"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-3 mb-2">
                    <h3 className="text-lg font-semibold">{server.name}</h3>
                    {server.source === 'marketplace' && (
                      <span className="px-2 py-1 rounded text-xs font-medium bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-400">
                        Marketplace
                      </span>
                    )}
                    <span className={`px-2 py-1 rounded text-xs font-medium ${
                      server.enabled
                        ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400'
                        : 'bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-400'
                    }`}>
                      {server.enabled ? 'Enabled' : 'Disabled'}
                    </span>
                    <span className={`px-2 py-1 rounded text-xs font-medium ${
                      server.status === 'active'
                        ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400'
                        : 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-400'
                    }`}>
                      {server.status}
                    </span>
                  </div>
                  <p className="text-sm text-muted-foreground mb-3">{server.description}</p>
                  
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                    <div>
                      <span className="text-muted-foreground">Type:</span>
                      <p className="font-medium">{server.type}</p>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Protocol:</span>
                      <p className="font-medium">{server.protocol}</p>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Usage Count:</span>
                      <p className="font-medium">{server.usage_count}</p>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Errors:</span>
                      <p className="font-medium text-red-600">{server.error_count}</p>
                    </div>
                  </div>
                </div>

                {/* Actions */}
                <div className="flex items-center gap-2 ml-4">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleToggle(server.id, server.enabled, server)}
                    title={server.enabled ? 'Disable' : 'Enable'}
                  >
                    {server.enabled ? (
                      <Power className="w-4 h-4 text-green-600" />
                    ) : (
                      <PowerOff className="w-4 h-4 text-gray-400" />
                    )}
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleOpenSettings(server)}
                    title="Settings"
                  >
                    <Settings className="w-4 h-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleOpenTest(server)}
                    title="Test"
                  >
                    <TestTube className="w-4 h-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleDelete(server.id, server)}
                    title="Delete"
                  >
                    <Trash2 className="w-4 h-4 text-red-600" />
                  </Button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export default MCPManagement
