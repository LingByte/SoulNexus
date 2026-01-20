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
  GitBranch
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

      default:
        return null
    }
  }

  return (
    <div className="space-y-6">
      {content.map((item, index) => renderContent(item, index))}
    </div>
  )
}

export default DocumentRenderer
