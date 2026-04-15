import { useState, useEffect } from 'react'
import { 
  Lightbulb, 
  Wand2, 
  BookOpen, 
  Target, 
  TrendingUp,
  CheckCircle,
  Clock,
  Brain,
  Sparkles
} from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'
import { cn } from '@/utils/cn'
import Card from '../UI/Card'
import Button from '../UI/Button'
import Badge from '../UI/Badge'
import ProgressBar from '../Data/ProgressBar'

interface WritingTip {
  id: string
  type: 'grammar' | 'style' | 'plot' | 'character' | 'dialogue' | 'description'
  title: string
  description: string
  suggestion: string
  priority: 'low' | 'medium' | 'high'
  category: string
}

interface WritingStats {
  wordCount: number
  characterCount: number
  paragraphCount: number
  sentenceCount: number
  averageSentenceLength: number
  readingTime: number
  complexityScore: number
  readabilityScore: number
}

interface WritingAssistantProps {
  content: string
  onSuggestionApply: (suggestion: string) => void
  className?: string
}

const WritingAssistant = ({
  content,
  onSuggestionApply,
  className
}: WritingAssistantProps) => {
  const [activeTab, setActiveTab] = useState<'tips' | 'stats' | 'suggestions'>('tips')
  const [tips, setTips] = useState<WritingTip[]>([])
  const [stats, setStats] = useState<WritingStats | null>(null)
  const [isAnalyzing, setIsAnalyzing] = useState(false)

  // 分析文本内容
  useEffect(() => {
    if (content.length > 0) {
      analyzeContent()
    }
  }, [content])

  const analyzeContent = async () => {
    setIsAnalyzing(true)
    
    // 模拟分析过程
    setTimeout(() => {
      const newStats = calculateStats(content)
      setStats(newStats)
      
      const newTips = generateTips(content, newStats)
      setTips(newTips)
      
      setIsAnalyzing(false)
    }, 1000)
  }

  const calculateStats = (text: string): WritingStats => {
    const words = text.trim().split(/\s+/).filter(word => word.length > 0)
    const sentences = text.split(/[.!?]+/).filter(s => s.trim().length > 0)
    const paragraphs = text.split(/\n\s*\n/).filter(p => p.trim().length > 0)
    
    const averageSentenceLength = sentences.length > 0 ? words.length / sentences.length : 0
    const readingTime = Math.ceil(words.length / 200) // 假设每分钟阅读200字
    
    // 简单的复杂度计算
    const complexWords = words.filter(word => word.length > 6).length
    const complexityScore = Math.min((complexWords / words.length) * 100, 100)
    
    // 简单的可读性计算 (基于句子长度和词汇复杂度)
    const readabilityScore = Math.max(0, 100 - (averageSentenceLength * 2) - (complexityScore * 0.5))
    
    return {
      wordCount: words.length,
      characterCount: text.length,
      paragraphCount: paragraphs.length,
      sentenceCount: sentences.length,
      averageSentenceLength: Math.round(averageSentenceLength * 10) / 10,
      readingTime,
      complexityScore: Math.round(complexityScore),
      readabilityScore: Math.round(readabilityScore)
    }
  }

  const generateTips = (text: string, stats: WritingStats): WritingTip[] => {
    const tips: WritingTip[] = []
    
    // 基于统计数据的建议
    if (stats.averageSentenceLength > 20) {
      tips.push({
        id: 'sentence-length',
        type: 'style',
        title: '句子过长',
        description: '平均句子长度较长，可能影响阅读体验',
        suggestion: '尝试将长句子拆分成多个短句，提高可读性',
        priority: 'medium',
        category: '可读性'
      })
    }
    
    if (stats.complexityScore > 70) {
      tips.push({
        id: 'complexity',
        type: 'style',
        title: '词汇复杂度过高',
        description: '使用了较多复杂词汇，可能影响理解',
        suggestion: '考虑使用更简单的词汇，或添加解释',
        priority: 'low',
        category: '可读性'
      })
    }
    
    if (stats.readabilityScore < 60) {
      tips.push({
        id: 'readability',
        type: 'style',
        title: '可读性较低',
        description: '文本可读性有待提高',
        suggestion: '简化句子结构，使用更常见的词汇',
        priority: 'high',
        category: '可读性'
      })
    }
    
    // 基于内容的建议
    if (text.includes('。') && !text.includes('！') && !text.includes('？')) {
      tips.push({
        id: 'punctuation',
        type: 'style',
        title: '标点符号单一',
        description: '主要使用句号，缺乏其他标点符号',
        suggestion: '适当使用问号、感叹号等标点符号，增加文本表现力',
        priority: 'low',
        category: '标点符号'
      })
    }
    
    if (text.split('。').length > 5 && text.split('，').length < text.split('。').length) {
      tips.push({
        id: 'comma-usage',
        type: 'grammar',
        title: '逗号使用不足',
        description: '句子中逗号使用较少，可能影响节奏',
        suggestion: '在适当位置添加逗号，改善句子节奏',
        priority: 'medium',
        category: '语法'
      })
    }
    
    return tips
  }

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'high': return 'error'
      case 'medium': return 'warning'
      case 'low': return 'success'
      default: return 'default'
    }
  }

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'grammar': return '📝'
      case 'style': return '🎨'
      case 'plot': return '📖'
      case 'character': return '👤'
      case 'dialogue': return '💬'
      case 'description': return '🖼️'
      default: return '💡'
    }
  }

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'grammar': return 'blue'
      case 'style': return 'purple'
      case 'plot': return 'green'
      case 'character': return 'orange'
      case 'dialogue': return 'pink'
      case 'description': return 'cyan'
      default: return 'gray'
    }
  }

  return (
    <div className={cn('space-y-6', className)}>
      {/* 标签页导航 */}
      <div className="flex space-x-1 bg-neutral-100 dark:bg-neutral-800 p-1 rounded-lg">
        <button
          onClick={() => setActiveTab('tips')}
          className={cn(
            'flex-1 px-3 py-2 text-sm font-medium rounded-md transition-colors',
            activeTab === 'tips'
              ? 'bg-white dark:bg-neutral-700 text-neutral-900 dark:text-white shadow-sm'
              : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'
          )}
        >
          <Lightbulb className="w-4 h-4 inline mr-2" />
          写作建议
        </button>
        <button
          onClick={() => setActiveTab('stats')}
          className={cn(
            'flex-1 px-3 py-2 text-sm font-medium rounded-md transition-colors',
            activeTab === 'stats'
              ? 'bg-white dark:bg-neutral-700 text-neutral-900 dark:text-white shadow-sm'
              : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'
          )}
        >
          <TrendingUp className="w-4 h-4 inline mr-2" />
          文本统计
        </button>
        <button
          onClick={() => setActiveTab('suggestions')}
          className={cn(
            'flex-1 px-3 py-2 text-sm font-medium rounded-md transition-colors',
            activeTab === 'suggestions'
              ? 'bg-white dark:bg-neutral-700 text-neutral-900 dark:text-white shadow-sm'
              : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'
          )}
        >
          <Wand2 className="w-4 h-4 inline mr-2" />
          智能建议
        </button>
      </div>

      {/* 内容区域 */}
      <AnimatePresence mode="wait">
        {activeTab === 'tips' && (
          <motion.div
            key="tips"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.2 }}
            className="space-y-4"
          >
            {isAnalyzing ? (
              <Card className="p-8 text-center">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
                <p className="text-neutral-600 dark:text-neutral-400">正在分析文本...</p>
              </Card>
            ) : tips.length > 0 ? (
              tips.map((tip) => (
                <Card key={tip.id} className="p-4 hover:shadow-lg transition-all">
                  <div className="flex items-start gap-3">
                    <div className="text-2xl">{getTypeIcon(tip.type)}</div>
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <h3 className="font-semibold text-neutral-900 dark:text-white">
                          {tip.title}
                        </h3>
                        <Badge variant={getPriorityColor(tip.priority) as any} size="sm">
                          {tip.priority === 'high' ? '高' : tip.priority === 'medium' ? '中' : '低'}
                        </Badge>
                        <Badge variant={getTypeColor(tip.type) as any} size="sm">
                          {tip.category}
                        </Badge>
                      </div>
                      <p className="text-sm text-neutral-600 dark:text-neutral-400 mb-3">
                        {tip.description}
                      </p>
                      <div className="bg-neutral-50 dark:bg-neutral-800 p-3 rounded-lg">
                        <p className="text-sm font-medium text-neutral-900 dark:text-white mb-1">
                          建议：
                        </p>
                        <p className="text-sm text-neutral-700 dark:text-neutral-300">
                          {tip.suggestion}
                        </p>
                      </div>
                    </div>
                    <Button
                      size="sm"
                      onClick={() => onSuggestionApply(tip.suggestion)}
                    >
                      应用
                    </Button>
                  </div>
                </Card>
              ))
            ) : (
              <Card className="p-8 text-center">
                <CheckCircle className="w-12 h-12 text-green-500 mx-auto mb-4" />
                <h3 className="text-lg font-semibold text-neutral-900 dark:text-white mb-2">
                  文本质量良好
                </h3>
                <p className="text-neutral-600 dark:text-neutral-400">
                  没有发现需要改进的地方，继续保持！
                </p>
              </Card>
            )}
          </motion.div>
        )}

        {activeTab === 'stats' && (
          <motion.div
            key="stats"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.2 }}
            className="space-y-6"
          >
            {stats ? (
              <>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  <Card className="p-4 text-center">
                    <BookOpen className="w-8 h-8 text-primary mx-auto mb-2" />
                    <div className="text-2xl font-bold text-neutral-900 dark:text-white">
                      {stats.wordCount.toLocaleString()}
                    </div>
                    <div className="text-sm text-neutral-600 dark:text-neutral-400">字数</div>
                  </Card>
                  
                  <Card className="p-4 text-center">
                    <Target className="w-8 h-8 text-secondary mx-auto mb-2" />
                    <div className="text-2xl font-bold text-neutral-900 dark:text-white">
                      {stats.sentenceCount}
                    </div>
                    <div className="text-sm text-neutral-600 dark:text-neutral-400">句子</div>
                  </Card>
                  
                  <Card className="p-4 text-center">
                    <Clock className="w-8 h-8 text-accent mx-auto mb-2" />
                    <div className="text-2xl font-bold text-neutral-900 dark:text-white">
                      {stats.readingTime}
                    </div>
                    <div className="text-sm text-neutral-600 dark:text-neutral-400">分钟阅读</div>
                  </Card>
                  
                  <Card className="p-4 text-center">
                    <Brain className="w-8 h-8 text-green-500 mx-auto mb-2" />
                    <div className="text-2xl font-bold text-neutral-900 dark:text-white">
                      {stats.averageSentenceLength}
                    </div>
                    <div className="text-sm text-neutral-600 dark:text-neutral-400">平均句长</div>
                  </Card>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <Card className="p-4">
                    <h3 className="text-lg font-semibold text-neutral-900 dark:text-white mb-4">
                      可读性评分
                    </h3>
                    <ProgressBar
                      value={stats.readabilityScore}
                      variant={stats.readabilityScore > 70 ? 'success' : stats.readabilityScore > 50 ? 'warning' : 'error'}
                      showValue
                    />
                    <p className="text-sm text-neutral-600 dark:text-neutral-400 mt-2">
                      {stats.readabilityScore > 70 ? '可读性良好' : 
                       stats.readabilityScore > 50 ? '可读性一般' : '可读性较差'}
                    </p>
                  </Card>

                  <Card className="p-4">
                    <h3 className="text-lg font-semibold text-neutral-900 dark:text-white mb-4">
                      复杂度评分
                    </h3>
                    <ProgressBar
                      value={stats.complexityScore}
                      variant={stats.complexityScore > 70 ? 'warning' : 'success'}
                      showValue
                    />
                    <p className="text-sm text-neutral-600 dark:text-neutral-400 mt-2">
                      {stats.complexityScore > 70 ? '词汇较复杂' : '词汇适中'}
                    </p>
                  </Card>
                </div>
              </>
            ) : (
              <Card className="p-8 text-center">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
                <p className="text-neutral-600 dark:text-neutral-400">正在分析文本...</p>
              </Card>
            )}
          </motion.div>
        )}

        {activeTab === 'suggestions' && (
          <motion.div
            key="suggestions"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.2 }}
            className="space-y-4"
          >
            <Card className="p-6">
              <div className="flex items-center gap-3 mb-4">
                <Sparkles className="w-6 h-6 text-primary" />
                <h3 className="text-lg font-semibold text-neutral-900 dark:text-white">
                  智能写作建议
                </h3>
              </div>
              
              <div className="space-y-4">
                <div className="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                  <h4 className="font-medium text-blue-900 dark:text-blue-200 mb-2">
                    💡 情节发展建议
                  </h4>
                  <p className="text-sm text-blue-800 dark:text-blue-300">
                    考虑在当前位置添加一个转折点，增加故事的紧张感。可以引入一个意外事件或角色冲突。
                  </p>
                </div>
                
                <div className="p-4 bg-green-50 dark:bg-green-900/20 rounded-lg">
                  <h4 className="font-medium text-green-900 dark:text-green-200 mb-2">
                    🎭 角色塑造建议
                  </h4>
                  <p className="text-sm text-green-800 dark:text-green-300">
                    通过角色的内心独白或行为细节来展现其性格特点，让读者更好地理解角色。
                  </p>
                </div>
                
                <div className="p-4 bg-purple-50 dark:bg-purple-900/20 rounded-lg">
                  <h4 className="font-medium text-purple-900 dark:text-purple-200 mb-2">
                    🖼️ 环境描写建议
                  </h4>
                  <p className="text-sm text-purple-800 dark:text-purple-300">
                    增加对环境的感官描写，如声音、气味、触觉等，让场景更加生动。
                  </p>
                </div>
              </div>
            </Card>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

export default WritingAssistant
