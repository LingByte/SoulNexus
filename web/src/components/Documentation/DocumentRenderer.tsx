import { useState } from 'react'
import { motion } from 'framer-motion'
import { 
  CheckCircle, 
  Copy, 
  Check,
  ChevronDown,
  ChevronRight,
  Globe,
  Lock,
  Database,
  Zap,
  Shield,
  Server,
  Activity,
  Bug,
  Lightbulb,
  BookOpen,
  Code,
  GitBranch,
  Play,
  Key
} from 'lucide-react'
import Card, { CardContent } from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Badge from '@/components/UI/Badge'

interface DocumentRendererProps {
  content: any[]
}

const DocumentRenderer = ({ content }: DocumentRendererProps) => {
  const [copiedCode, setCopiedCode] = useState<string | null>(null)
  const [expandedEndpoints, setExpandedEndpoints] = useState<Set<string>>(new Set())
  const [apiKey, setApiKey] = useState('')
  const [apiSecret, setApiSecret] = useState('')
  const [testResults, setTestResults] = useState<Record<string, { loading: boolean; result: string }>>({})

  const getIcon = (iconName: string) => {
    const icons: { [key: string]: any } = {
      CheckCircle,
      Copy,
      Check,
      ChevronDown,
      ChevronRight,
      Globe,
      Lock,
      Database,
      Zap,
      Shield,
      Server,
      Activity,
      Bug,
      Lightbulb,
      BookOpen,
      Code,
      GitBranch
    }
    return icons[iconName] || Code
  }

  const getMethodColor = (method: string) => {
    switch (method) {
      case 'GET': return 'bg-blue-100 text-blue-800'
      case 'POST': return 'bg-green-100 text-green-800'
      case 'PUT': return 'bg-yellow-100 text-yellow-800'
      case 'DELETE': return 'bg-red-100 text-red-800'
      case 'PATCH': return 'bg-purple-100 text-purple-800'
      default: return 'bg-gray-100 text-gray-800'
    }
  }

  const getMethodIcon = (method: string) => {
    switch (method) {
      case 'GET': return <Globe className="w-3 h-3" />
      case 'POST': return <Database className="w-3 h-3" />
      case 'PUT': return <Code className="w-3 h-3" />
      case 'DELETE': return <Zap className="w-3 h-3" />
      case 'PATCH': return <Code className="w-3 h-3" />
      default: return <Code className="w-3 h-3" />
    }
  }

  const copyToClipboard = async (text: string, id: string) => {
    try {
      await navigator.clipboard.writeText(text)
      setCopiedCode(id)
      setTimeout(() => setCopiedCode(null), 2000)
    } catch (err) {
      console.error('Failed to copy text: ', err)
    }
  }

  const toggleEndpoint = (path: string) => {
    const newExpanded = new Set(expandedEndpoints)
    if (newExpanded.has(path)) {
      newExpanded.delete(path)
    } else {
      newExpanded.add(path)
    }
    setExpandedEndpoints(newExpanded)
  }

  const formatJSON = (obj: any) => {
    return JSON.stringify(obj, null, 2)
  }

  const renderContent = (item: any, index: number) => {
    switch (item.type) {
      case 'heading':
        const HeadingTag = `h${item.level}` as keyof JSX.IntrinsicElements
        return (
          <HeadingTag key={index} className={`font-semibold text-foreground mb-4 ${
            item.level === 2 ? 'text-2xl' : 
            item.level === 3 ? 'text-xl' : 
            'text-lg'
          }`}>
            {item.text}
          </HeadingTag>
        )

      case 'paragraph':
        return (
          <p key={index} className="text-muted-foreground mb-4 leading-relaxed">
            {item.text}
          </p>
        )

      case 'code':
        const codeId = `code-${index}`
        return (
          <div key={index} className="mb-6">
            <div className="relative">
              <pre className="bg-muted p-4 rounded-lg text-sm overflow-x-auto">
                <code>{item.content}</code>
              </pre>
              <Button
                size="sm"
                variant="outline"
                className="absolute top-2 right-2"
                onClick={() => copyToClipboard(item.content, codeId)}
              >
                {copiedCode === codeId ? (
                  <Check className="w-3 h-3" />
                ) : (
                  <Copy className="w-3 h-3" />
                )}
              </Button>
            </div>
          </div>
        )

      case 'list':
        return (
          <ul key={index} className="space-y-2 mb-6">
            {item.items.map((listItem: any, itemIndex: number) => {
              const Icon = listItem.icon ? getIcon(listItem.icon) : null
              return (
                <li key={itemIndex} className="flex items-start gap-3">
                  {Icon && (
                    <Icon className={`w-4 h-4 mt-0.5 ${
                      listItem.status === 'success' ? 'text-green-500' :
                      listItem.status === 'info' ? 'text-blue-500' :
                      'text-muted-foreground'
                    }`} />
                  )}
                  <span className="text-muted-foreground">{listItem.text}</span>
                </li>
              )
            })}
          </ul>
        )

      case 'api_endpoint':
        const isExpanded = expandedEndpoints.has(item.path)
        const endpointId = `endpoint-${item.path.replace(/[^a-zA-Z0-9]/g, '-')}`
        
        return (
          <Card key={index} className="mb-4">
            <CardContent className="p-0">
              <div 
                className="p-4 cursor-pointer hover:bg-muted/50 transition-colors"
                onClick={() => toggleEndpoint(item.path)}
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <Badge className={getMethodColor(item.method)}>
                      <div className="flex items-center gap-1">
                        {getMethodIcon(item.method)}
                        {item.method}
                      </div>
                    </Badge>
                    <code className="text-sm font-mono bg-muted px-2 py-1 rounded">
                      {item.path}
                    </code>
                    {item.auth && (
                      <Badge variant="outline" className="text-xs">
                        <Lock className="w-3 h-3 mr-1" />
                        需要认证
                      </Badge>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    <h4 className="font-medium">{item.title}</h4>
                    {isExpanded ? (
                      <ChevronDown className="w-4 h-4" />
                    ) : (
                      <ChevronRight className="w-4 h-4" />
                    )}
                  </div>
                </div>
                <p className="text-sm text-muted-foreground mt-2">
                  {item.description}
                </p>
              </div>

              {isExpanded && (
                <motion.div
                  initial={{ opacity: 0, height: 0 }}
                  animate={{ opacity: 1, height: 'auto' }}
                  exit={{ opacity: 0, height: 0 }}
                  transition={{ duration: 0.2 }}
                  className="border-t"
                >
                  <div className="p-4 space-y-4">
                    {/* 参数 */}
                    {item.parameters && item.parameters.length > 0 && (
                      <div>
                        <h5 className="font-medium mb-2">参数</h5>
                        <div className="overflow-x-auto">
                          <table className="w-full text-sm">
                            <thead>
                              <tr className="border-b">
                                <th className="text-left py-2">名称</th>
                                <th className="text-left py-2">类型</th>
                                <th className="text-left py-2">必需</th>
                                <th className="text-left py-2">描述</th>
                              </tr>
                            </thead>
                            <tbody>
                              {item.parameters.map((param: any, paramIndex: number) => (
                                <tr key={paramIndex} className="border-b">
                                  <td className="py-2 font-mono">{param.name}</td>
                                  <td className="py-2">{param.type}</td>
                                  <td className="py-2">
                                    {param.required ? (
                                      <Badge className="bg-red-100 text-red-800 text-xs">必需</Badge>
                                    ) : (
                                      <Badge variant="outline" className="text-xs">可选</Badge>
                                    )}
                                  </td>
                                  <td className="py-2">{param.description}</td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      </div>
                    )}

                    {/* 请求体 */}
                    {item.requestBody && (
                      <div>
                        <h5 className="font-medium mb-2">请求体</h5>
                        <div className="relative">
                          <pre className="bg-muted p-3 rounded text-xs overflow-x-auto">
                            <code>
                              {typeof item.requestBody === 'string' 
                                ? item.requestBody 
                                : formatJSON(item.requestBody)
                              }
                            </code>
                          </pre>
                          <Button
                            size="sm"
                            variant="outline"
                            className="absolute top-2 right-2"
                            onClick={() => copyToClipboard(
                              typeof item.requestBody === 'string' 
                                ? item.requestBody 
                                : formatJSON(item.requestBody),
                              `${endpointId}-request`
                            )}
                          >
                            {copiedCode === `${endpointId}-request` ? (
                              <Check className="w-3 h-3" />
                            ) : (
                              <Copy className="w-3 h-3" />
                            )}
                          </Button>
                        </div>
                      </div>
                    )}

                    {/* 响应 */}
                    {item.response && (
                      <div>
                        <h5 className="font-medium mb-2">响应</h5>
                        <div className="relative">
                          <pre className="bg-muted p-3 rounded text-xs overflow-x-auto">
                            <code>{formatJSON(item.response)}</code>
                          </pre>
                          <Button
                            size="sm"
                            variant="outline"
                            className="absolute top-2 right-2"
                            onClick={() => copyToClipboard(
                              formatJSON(item.response),
                              `${endpointId}-response`
                            )}
                          >
                            {copiedCode === `${endpointId}-response` ? (
                              <Check className="w-3 h-3" />
                            ) : (
                              <Copy className="w-3 h-3" />
                            )}
                          </Button>
                        </div>
                      </div>
                    )}

                    {/* 在线测试 */}
                    <div className="border-t pt-4">
                      <h5 className="font-medium mb-3 flex items-center gap-2">
                        <Play className="w-4 h-4 text-primary" />
                        在线测试
                      </h5>
                      {!apiKey || !apiSecret ? (
                        <p className="text-xs text-muted-foreground">请先在上方填写 API Key 和 API Secret</p>
                      ) : (
                        <div className="space-y-3">
                          {item.path === '/api/open/tts' && (
                            <div>
                              <label className="text-xs font-medium text-muted-foreground mb-1 block">合成文本</label>
                              <input
                                id={`${endpointId}-tts-text`}
                                type="text"
                                defaultValue="你好，这是一段测试语音。"
                                className="w-full px-3 py-2 text-sm border rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
                              />
                            </div>
                          )}
                          {item.path === '/api/open/asr' && (
                            <div>
                              <label className="text-xs font-medium text-muted-foreground mb-1 block">上传 WAV 文件</label>
                              <input
                                id={`${endpointId}-asr-file`}
                                type="file"
                                accept=".wav,audio/wav"
                                className="text-sm text-muted-foreground"
                              />
                            </div>
                          )}
                          <Button
                            size="sm"
                            onClick={() => {
                              let extra: any = {}
                              if (item.path === '/api/open/tts') {
                                const el = document.getElementById(`${endpointId}-tts-text`) as HTMLInputElement
                                extra.text = el?.value || '你好，这是一段测试语音。'
                              }
                              if (item.path === '/api/open/asr') {
                                const el = document.getElementById(`${endpointId}-asr-file`) as HTMLInputElement
                                extra.file = el?.files?.[0]
                              }
                              runTest(item, extra)
                            }}
                            disabled={testResults[item.path]?.loading}
                          >
                            <Play className="w-3 h-3 mr-1" />
                            {testResults[item.path]?.loading ? '请求中...' : '发送请求'}
                          </Button>
                          {testResults[item.path]?.result && (
                            <pre className="bg-muted p-3 rounded text-xs overflow-x-auto whitespace-pre-wrap">
                              {testResults[item.path].result}
                            </pre>
                          )}
                        </div>
                      )}
                    </div>
                  </div>
                </motion.div>
              )}
            </CardContent>
          </Card>
        )

      case 'feature_grid':
        return (
          <div key={index} className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-6">
            {item.features.map((feature: any, featureIndex: number) => {
              const Icon = getIcon(feature.icon)
              const colorClasses = {
                blue: 'bg-blue-100 text-blue-600',
                green: 'bg-green-100 text-green-600',
                purple: 'bg-purple-100 text-purple-600',
                orange: 'bg-orange-100 text-orange-600',
                cyan: 'bg-cyan-100 text-cyan-600',
                red: 'bg-red-100 text-red-600'
              }
              
              return (
                <Card key={featureIndex}>
                  <CardContent className="p-6">
                    <div className="flex items-center gap-3 mb-4">
                      <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${colorClasses[feature.color as keyof typeof colorClasses]}`}>
                        <Icon className="w-5 h-5" />
                      </div>
                      <h3 className="text-lg font-semibold">{feature.title}</h3>
                    </div>
                    <p className="text-muted-foreground mb-4">
                      {feature.description}
                    </p>
                    <ul className="space-y-2 text-sm">
                      {feature.features.map((feat: string, featIndex: number) => (
                        <li key={featIndex} className="flex items-center gap-2">
                          <CheckCircle className="w-4 h-4 text-green-500" />
                          {feat}
                        </li>
                      ))}
                    </ul>
                  </CardContent>
                </Card>
              )
            })}
          </div>
        )

      case 'tech_stack':
        return (
          <div key={index} className="space-y-6 mb-6">
            {item.categories.map((category: any, catIndex: number) => (
              <div key={catIndex}>
                <h4 className="text-lg font-semibold mb-3">{category.name}</h4>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {category.technologies.map((tech: any, techIndex: number) => (
                    <Card key={techIndex}>
                      <CardContent className="p-4">
                        <div className="flex items-center justify-between mb-2">
                          <h5 className="font-medium">{tech.name}</h5>
                          <Badge variant="outline">{tech.version}</Badge>
                        </div>
                        <p className="text-sm text-muted-foreground">{tech.description}</p>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )

      case 'env_vars':
        return (
          <div key={index} className="space-y-4 mb-6">
            {item.variables.map((variable: any, varIndex: number) => (
              <Card key={varIndex}>
                <CardContent className="p-4">
                  <div className="flex items-center justify-between mb-2">
                    <code className="text-sm font-mono bg-muted px-2 py-1 rounded">
                      {variable.name}
                    </code>
                    <div className="flex items-center gap-2">
                      <Badge variant="outline">{variable.type}</Badge>
                      {variable.required && (
                        <Badge className="bg-red-100 text-red-800 text-xs">必需</Badge>
                      )}
                    </div>
                  </div>
                  <p className="text-sm text-muted-foreground mb-2">{variable.description}</p>
                  <div className="text-xs text-muted-foreground">
                    <strong>示例:</strong> <code className="bg-muted px-1 rounded">{variable.example}</code>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )

      case 'optimization_tips':
        return (
          <div key={index} className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
            {item.tips.map((tip: any, tipIndex: number) => {
              const Icon = getIcon(tip.icon)
              return (
                <Card key={tipIndex}>
                  <CardContent className="p-4">
                    <div className="flex items-center gap-3 mb-2">
                      <Icon className="w-5 h-5 text-primary" />
                      <h5 className="font-medium">{tip.title}</h5>
                    </div>
                    <p className="text-sm text-muted-foreground">{tip.description}</p>
                  </CardContent>
                </Card>
              )
            })}
          </div>
        )

      case 'step_list':
        return (
          <div key={index} className="space-y-4 mb-6">
            {item.steps.map((step: any, stepIndex: number) => (
              <div key={stepIndex} className="flex gap-4">
                <div className="flex-shrink-0 w-8 h-8 bg-primary text-primary-foreground rounded-full flex items-center justify-center font-semibold text-sm">
                  {step.step}
                </div>
                <div className="flex-1">
                  <h5 className="font-medium mb-1">{step.title}</h5>
                  <p className="text-sm text-muted-foreground">{step.description}</p>
                </div>
              </div>
            ))}
          </div>
        )

      case 'code_standards':
        return (
          <div key={index} className="space-y-4 mb-6">
            {item.languages.map((lang: any, langIndex: number) => (
              <div key={langIndex}>
                <h4 className="text-lg font-semibold mb-3">{lang.name}</h4>
                <ul className="space-y-2">
                  {lang.standards.map((standard: string, stdIndex: number) => (
                    <li key={stdIndex} className="flex items-start gap-2">
                      <CheckCircle className="w-4 h-4 text-green-500 mt-0.5" />
                      <span className="text-sm text-muted-foreground">{standard}</span>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        )

      case 'api_tester':
        return (
          <Card key={index} className="mb-6 border-primary/30">
            <CardContent className="p-5">
              <div className="flex items-center gap-2 mb-4">
                <Key className="w-5 h-5 text-primary" />
                <h4 className="font-semibold text-base">API 鉴权配置</h4>
              </div>
              <p className="text-sm text-muted-foreground mb-4">
                填入凭证后，可在下方各接口卡片中点击「测试」直接发起请求。
              </p>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                <div>
                  <label className="text-xs font-medium text-muted-foreground mb-1 block">X-API-KEY</label>
                  <input
                    type="text"
                    placeholder="your-api-key"
                    value={apiKey}
                    onChange={e => setApiKey(e.target.value)}
                    className="w-full px-3 py-2 text-sm border rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-primary/50 font-mono"
                  />
                </div>
                <div>
                  <label className="text-xs font-medium text-muted-foreground mb-1 block">X-API-SECRET</label>
                  <input
                    type="password"
                    placeholder="your-api-secret"
                    value={apiSecret}
                    onChange={e => setApiSecret(e.target.value)}
                    className="w-full px-3 py-2 text-sm border rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-primary/50 font-mono"
                  />
                </div>
              </div>
            </CardContent>
          </Card>
        )

      default:
        return null
    }
  }

  const runTest = async (endpoint: any, extraData?: { text?: string; file?: File }) => {
    const key = endpoint.path
    setTestResults(prev => ({ ...prev, [key]: { loading: true, result: '' } }))

    const baseURL = (import.meta.env.VITE_API_BASE_URL as string || 'http://localhost:7072/api').replace(/\/$/, '')
    // endpoint.path is like /api/open/me — strip the leading /api since baseURL already ends with /api
    const relativePath = endpoint.path.replace(/^\/api/, '')
    const url = `${baseURL}${relativePath}`

    try {
      const headers: Record<string, string> = {
        'X-API-KEY': apiKey,
        'X-API-SECRET': apiSecret,
      }

      let response: Response

      if (endpoint.path === '/api/open/asr') {
        const form = new FormData()
        if (extraData?.file) form.append('file', extraData.file)
        response = await fetch(url, { method: 'POST', headers, body: form })
      } else if (endpoint.method === 'POST') {
        headers['Content-Type'] = 'application/json'
        const body = endpoint.path === '/api/open/tts'
          ? JSON.stringify({ text: extraData?.text || '你好，这是一段测试语音。' })
          : '{}'
        response = await fetch(url, { method: 'POST', headers, body })
      } else {
        response = await fetch(url, { method: 'GET', headers })
      }

      const json = await response.json()
      setTestResults(prev => ({
        ...prev,
        [key]: { loading: false, result: JSON.stringify(json, null, 2) }
      }))
    } catch (err: any) {
      setTestResults(prev => ({
        ...prev,
        [key]: { loading: false, result: `Error: ${err.message}` }
      }))
    }
  }

  return (
    <div className="space-y-6">
      {content.map((item, index) => renderContent(item, index))}
    </div>
  )
}

export default DocumentRenderer
