import React, { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { getAssistantGraphData, getAssistant, type AssistantGraphData as GraphData } from '@/api/assistant'
import GraphVisualization from '@/components/Graph/GraphVisualization'
import { showAlert } from '@/utils/notification'
import { useI18nStore } from '@/stores/i18nStore'
import { ArrowLeft, Loader2 } from 'lucide-react'
import Button from '@/components/UI/Button'
import Card, { CardContent, CardHeader, CardTitle } from '@/components/UI/Card'

const AssistantGraph: React.FC = () => {
  const { t } = useI18nStore()
  const { id } = useParams<{ id: string }>()
  const assistantId = id ? parseInt(id) : 0
  const navigate = useNavigate()

  const [graphData, setGraphData] = useState<GraphData | null>(null)
  const [assistantName, setAssistantName] = useState<string>('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchData = async () => {
      if (!assistantId) {
        setError('Invalid assistant ID')
        setLoading(false)
        return
      }

      try {
        setLoading(true)
        setError(null)

        // 获取助手信息
        const assistantRes = await getAssistant(assistantId)
        if (assistantRes.code === 200 && assistantRes.data) {
          setAssistantName(assistantRes.data.name)
          
          // 检查是否启用了图记忆
          if (!assistantRes.data.enableGraphMemory) {
            setError('该助手未启用图记忆功能')
            setLoading(false)
            return
          }
        }

        // 获取图数据
        const graphRes = await getAssistantGraphData(assistantId)
        if (graphRes.code === 200) {
          setGraphData(graphRes.data)
        } else {
          setError(graphRes.msg || '获取图数据失败')
        }
      } catch (err: any) {
        const errorMsg = err?.msg || err?.message || '获取图数据失败'
        setError(errorMsg)
        showAlert(errorMsg, 'error')
      } finally {
        setLoading(false)
      }
    }

    fetchData()
  }, [assistantId])

  if (loading) {
    return (
      <div className="min-h-screen dark:bg-neutral-900 flex items-center justify-center">
        <div className="flex flex-col items-center gap-4">
          <Loader2 className="w-8 h-8 animate-spin text-primary" />
          <p className="text-gray-600 dark:text-gray-400">加载中...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="min-h-screen dark:bg-neutral-900 p-6">
        <div className="max-w-4xl mx-auto">
          <Card>
            <CardHeader>
              <div className="flex items-center gap-4">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => navigate('/assistants')}
                  leftIcon={<ArrowLeft className="w-4 h-4" />}
                >
                  返回
                </Button>
                <CardTitle>助手图数据</CardTitle>
              </div>
            </CardHeader>
            <CardContent>
              <div className="text-center py-12">
                <p className="text-red-600 dark:text-red-400 mb-4">{error}</p>
                <Button
                  variant="primary"
                  onClick={() => navigate('/assistants')}
                >
                  返回助手列表
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen dark:bg-neutral-900 p-6">
      <div className="max-w-7xl mx-auto">
        <div className="mb-6">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => navigate('/assistants')}
            leftIcon={<ArrowLeft className="w-4 h-4" />}
            className="mb-4"
          >
            返回助手列表
          </Button>
          <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100">
            {assistantName} - 知识图谱
          </h1>
          <p className="text-gray-600 dark:text-gray-400 mt-2">
            展示该助手在 Neo4j 图数据库中的知识图谱关系
          </p>
        </div>

        {graphData && (
          <GraphVisualization
            nodes={graphData.nodes}
            edges={graphData.edges}
            stats={graphData.stats}
            loading={false}
          />
        )}
      </div>
    </div>
  )
}

export default AssistantGraph

