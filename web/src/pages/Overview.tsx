import React, { useEffect, useState, useMemo, useCallback, useRef } from 'react'
import { useAuthStore } from '@/stores/authStore'
import { useI18nStore } from '@/stores/i18nStore'
import { getGroup } from '@/api/group'
import { getOverviewConfig, getOrganizationStats } from '@/api/overview'
import { showAlert } from '@/utils/notification'
import { Building2 } from 'lucide-react'
import Card, { CardContent } from '@/components/UI/Card'
import DynamicOverview from '@/components/Overview/DynamicOverview'
import OverviewSkeleton from '@/components/Overview/OverviewSkeleton'
import { overviewCache } from '@/utils/overviewCache'
import { requestDeduplication } from '@/utils/requestDeduplication'
import { shallowEqual } from '@/utils/deepEqual'
import { OverviewConfig, defaultOverviewConfig, WidgetConfig, widgetSizeMap, widgetHeightMap } from '@/types/overview'

const Overview: React.FC = () => {
  const { t } = useI18nStore()
  const { currentOrganizationId, isAuthenticated } = useAuthStore()
  const [loading, setLoading] = useState(true)
  const [config, setConfig] = useState<OverviewConfig | null>(null)
  const [widgetData, setWidgetData] = useState<Record<string, any>>({})
  const previousWidgetDataRef = useRef<Record<string, any>>({})
  const loadDataAbortControllerRef = useRef<AbortController | null>(null)

  useEffect(() => {
    if (!isAuthenticated) {
      return
    }

    if (!currentOrganizationId) {
      setLoading(false)
      return
    }

    loadOrganizationData()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentOrganizationId, isAuthenticated])

  const loadOrganizationData = useCallback(async () => {
    if (!currentOrganizationId) return

    // 取消之前的请求
    if (loadDataAbortControllerRef.current) {
      loadDataAbortControllerRef.current.abort()
    }
    loadDataAbortControllerRef.current = new AbortController()

    try {
      setLoading(true)
      
      // 检查缓存
      const cachedConfig = overviewCache.getConfig(currentOrganizationId)
      const cachedStats = overviewCache.getStats(currentOrganizationId)
      
      // 使用请求去重，并行加载数据（如果缓存不存在）
      const [groupRes, configRes, statsRes] = await Promise.allSettled([
        requestDeduplication.dedupe(
          `group-${currentOrganizationId}`,
          () => getGroup(currentOrganizationId)
        ),
        cachedConfig 
          ? Promise.resolve({ code: 200, data: cachedConfig })
          : requestDeduplication.dedupe(
              `config-${currentOrganizationId}`,
              () => getOverviewConfig(currentOrganizationId)
            ),
        cachedStats 
          ? Promise.resolve({ code: 200, data: cachedStats })
          : requestDeduplication.dedupe(
              `stats-${currentOrganizationId}`,
              () => getOrganizationStats(currentOrganizationId)
            )
      ])
      
      // 处理组织信息
      if (groupRes.status === 'rejected' || (groupRes.status === 'fulfilled' && groupRes.value.code !== 200)) {
        showAlert('获取组织信息失败', 'error')
        setLoading(false)
        return
      }
      
      const group = groupRes.value.data
      
      // 处理配置
      let loadedConfig: OverviewConfig | null = null
      
      if (configRes.status === 'fulfilled' && configRes.value.code === 200 && configRes.value.data) {
        const backendConfig = configRes.value.data as any
        loadedConfig = {
          id: backendConfig.id || `config-${Date.now()}`,
          organizationId: backendConfig.organizationId || currentOrganizationId,
          name: backendConfig.name || `${group.name} - 概览`,
          description: backendConfig.description,
          layout: {
            ...defaultOverviewConfig.layout,
            ...(backendConfig.layout || {})
          },
          widgets: backendConfig.widgets || [],
          theme: {
            ...defaultOverviewConfig.theme,
            ...(backendConfig.theme || {})
          },
          header: backendConfig.header || defaultOverviewConfig.header,
          footer: backendConfig.footer || defaultOverviewConfig.footer,
          createdAt: backendConfig.createdAt,
          updatedAt: backendConfig.updatedAt
        }
        
        // 缓存配置
        if (!cachedConfig) {
          overviewCache.setConfig(currentOrganizationId, backendConfig)
        }
      } else {
        // 如果没有配置，创建默认配置
        loadedConfig = {
          ...defaultOverviewConfig,
          id: `config-${Date.now()}`,
          organizationId: currentOrganizationId,
          name: `${group.name} - 概览`,
          widgets: createDefaultWidgets(currentOrganizationId, group.memberCount || 0)
        }
      }
      
      setConfig(loadedConfig)

      // 处理统计数据
      if (statsRes.status === 'fulfilled' && statsRes.value.code === 200 && statsRes.value.data && loadedConfig) {
        const stats = statsRes.value.data as any
        
        // 缓存统计数据
        if (!cachedStats) {
          overviewCache.setStats(currentOrganizationId, stats)
        }
        
        // 将统计数据映射到widget ID（使用 Map 提高性能）
        const dataMap: Record<string, any> = {}
        
        // 预计算常用数据键映射（避免重复计算）
        const keyMap: Record<string, string> = {
          '成员': 'totalMembers',
          '助手': 'totalAssistants',
          '知识库': 'totalKnowledgeBases',
          '通话': 'totalCalls',
          '工作流': 'totalWorkflows',
          '脚本': 'totalScripts',
          '设备': 'totalDevices',
          '音色': 'totalVoices',
        }
        
        // 使用 for 循环替代 forEach（性能更好）
        const widgets = loadedConfig.widgets
        for (let i = 0; i < widgets.length; i++) {
          const widget = widgets[i]
              if (widget.type === 'stat-card') {
                // 优先使用 dataKey，如果没有则尝试从 title 推断
                const dataKey = widget.props?.dataKey
                if (dataKey && stats[dataKey] !== undefined) {
                  // 直接使用 dataKey 匹配
                  dataMap[widget.id] = { value: stats[dataKey] }
                } else if (dataKey) {
                  // dataKey 存在但统计数据中没有，使用默认值
                  dataMap[widget.id] = { value: widget.props?.defaultValue ?? 0 }
                } else {
                  // 没有 dataKey，尝试从 title 推断
                  const titleLower = widget.title.toLowerCase()
                  let matched = false
                  
                  // 尝试匹配常见的数据键（使用预计算的 keyMap）
                  for (const [key, statKey] of Object.entries(keyMap)) {
                    if (titleLower.includes(key.toLowerCase())) {
                      if (stats[statKey] !== undefined) {
                        dataMap[widget.id] = { value: stats[statKey] }
                        matched = true
                        break
                      }
                    }
                  }
                  
                  if (!matched) {
                    // 如果都不匹配，使用默认值
                    dataMap[widget.id] = { value: widget.props?.defaultValue ?? 0 }
                  }
                }
              } else if (widget.type === 'activity-feed') {
                dataMap[widget.id] = Array.isArray(stats.recentActivity) ? stats.recentActivity : []
              } else if (widget.type.startsWith('chart-')) {
                // 图表数据映射
                const chartDataKey = widget.props?.dataKey
                if (chartDataKey && stats[chartDataKey] && Array.isArray(stats[chartDataKey])) {
                  dataMap[widget.id] = stats[chartDataKey]
                } else if (stats.chartData && Array.isArray(stats.chartData)) {
                  dataMap[widget.id] = stats.chartData
                } else if (stats.usageTrend && Array.isArray(stats.usageTrend)) {
                  dataMap[widget.id] = stats.usageTrend
                } else if (stats.activityData && Array.isArray(stats.activityData)) {
                  dataMap[widget.id] = stats.activityData
                } else {
                  // 使用真实的图表数据（如果有）
                  dataMap[widget.id] = Array.isArray(stats.chartData) && stats.chartData.length > 0 
                    ? stats.chartData 
                    : []
                }
              } else if (widget.type === 'progress-ring') {
                const valueKey = widget.props?.dataKey || 'value'
                const maxKey = widget.props?.maxKey || 'max'
                // 支持从账单统计中获取数据
                let value = widget.props?.value ?? 0
                let max = widget.props?.max ?? 100
                
                if (stats.billStatistics && typeof stats.billStatistics === 'object') {
                  const billStats = stats.billStatistics as any
                  if (valueKey === 'totalLLMCalls' && billStats.totalLLMCalls !== undefined) {
                    value = billStats.totalLLMCalls
                  } else if (valueKey === 'totalCallCount' && billStats.totalCallCount !== undefined) {
                    value = billStats.totalCallCount
                  } else if (stats[valueKey] !== undefined) {
                    value = stats[valueKey]
                  }
                  
                  if (maxKey === 'totalLLMCalls' && billStats.totalLLMCalls !== undefined) {
                    max = billStats.totalLLMCalls
                  } else if (maxKey === 'totalCallCount' && billStats.totalCallCount !== undefined) {
                    max = billStats.totalCallCount
                  } else if (stats[maxKey] !== undefined) {
                    max = stats[maxKey]
                  }
                } else if (stats[valueKey] !== undefined) {
                  value = stats[valueKey]
                }
                if (stats[maxKey] !== undefined) {
                  max = stats[maxKey]
                }
                
                dataMap[widget.id] = {
                  value,
                  max,
                  color: widget.props?.color
                }
              } else if (widget.type === 'metric-comparison') {
                const metricsKey = widget.props?.dataKey || 'metrics'
                // 如果指定了 billStatistics，从账单统计中构建指标数据
                if (metricsKey === 'billStatistics' && stats.billStatistics && typeof stats.billStatistics === 'object') {
                  const billStats = stats.billStatistics as any
                  dataMap[widget.id] = [
                    { label: 'LLM调用', value: billStats.totalLLMCalls || 0 },
                    { label: 'Token总数', value: billStats.totalLLMTokens || 0 },
                    { label: '通话次数', value: billStats.totalCallCount || 0 },
                    { label: '通话时长', value: billStats.totalCallDuration || 0 },
                    { label: 'ASR次数', value: billStats.totalASRCount || 0 },
                    { label: 'TTS次数', value: billStats.totalTTSCount || 0 },
                    { label: 'API调用', value: billStats.totalAPICalls || 0 },
                  ]
                } else {
                  dataMap[widget.id] = Array.isArray(stats[metricsKey]) ? stats[metricsKey] : []
                }
              } else if (widget.type === 'testimonial') {
                const testimonialsKey = widget.props?.dataKey || 'testimonials'
                dataMap[widget.id] = Array.isArray(stats[testimonialsKey]) ? stats[testimonialsKey] : []
              } else if (widget.type === 'timeline') {
                const timelineKey = widget.props?.dataKey || 'timeline'
                dataMap[widget.id] = Array.isArray(stats[timelineKey]) ? stats[timelineKey] : []
              } else if (widget.type === 'table') {
                // 表格数据映射
                const tableKey = widget.props?.dataKey
                if (tableKey && stats[tableKey] && typeof stats[tableKey] === 'object') {
                  dataMap[widget.id] = stats[tableKey]
                } else if (stats.table && typeof stats.table === 'object' && Array.isArray(stats.table.columns) && Array.isArray(stats.table.rows)) {
                  dataMap[widget.id] = stats.table
                } else {
                  // 如果没有数据，使用widget配置的列和行
                  const columns = widget.props?.columns || ['名称', '类型', '状态', '数量', '日期']
                  const rows = widget.props?.rows || []
                  
                  dataMap[widget.id] = {
                    columns,
                    rows
                  }
                }
              } else if (widget.type === 'image') {
                const imageKey = widget.props?.dataKey || 'image'
                dataMap[widget.id] = {
                  url: stats[imageKey]?.url || stats[imageKey] || widget.props?.imageUrl,
                  alt: stats[imageKey]?.alt || widget.props?.alt
                }
              } else if (widget.type === 'video') {
                const videoKey = widget.props?.dataKey || 'video'
                dataMap[widget.id] = {
                  url: stats[videoKey]?.url || stats[videoKey] || widget.props?.videoUrl,
                  type: stats[videoKey]?.type || widget.props?.type || 'direct'
                }
              } else if (widget.type === 'iframe') {
                const iframeKey = widget.props?.dataKey || 'iframe'
                dataMap[widget.id] = {
                  url: stats[iframeKey]?.url || stats[iframeKey] || widget.props?.iframeUrl
                }
              } else if (widget.type === 'markdown') {
                const markdownKey = widget.props?.dataKey || 'markdown'
                dataMap[widget.id] = {
                  content: stats[markdownKey]?.content || stats[markdownKey] || widget.props?.content || widget.props?.markdown
                }
              }
        }
        
        setWidgetData(dataMap)
      } else {
        console.warn('获取统计数据失败')
        // 即使统计数据获取失败，也继续显示页面
      }
    } catch (error: any) {
      console.error('加载组织数据失败:', error)
      showAlert(error?.msg || t('overview.messages.loadFailed'), 'error')
    } finally {
      setLoading(false)
    }
  }, [currentOrganizationId, t])

  // 使用 useMemo 优化数据映射，避免不必要的重新计算
  // 使用浅比较替代 JSON.stringify（性能提升 10-100 倍）
  // 必须在所有条件渲染之前调用，遵守 Hooks 规则
  const memoizedWidgetData = useMemo(() => {
    // 如果数据没有变化，返回之前的引用（避免子组件重渲染）
    if (shallowEqual(widgetData, previousWidgetDataRef.current)) {
      return previousWidgetDataRef.current
    }
    previousWidgetDataRef.current = widgetData
    return widgetData
  }, [widgetData])

  // 创建默认Widget配置
  const createDefaultWidgets = (_orgId: number, memberCount: number): WidgetConfig[] => {

    const defaultWidgets: WidgetConfig[] = [
      {
        id: 'widget-members',
        type: 'stat-card',
        title: t('overview.stats.members'),
        size: 'medium',
        position: { x: 0, y: 0, w: widgetSizeMap.medium, h: widgetHeightMap['stat-card'] },
        props: { defaultValue: memberCount, icon: 'members' },
        visible: true
      },
      {
        id: 'widget-assistants',
        type: 'stat-card',
        title: t('overview.stats.assistants'),
        size: 'medium',
        position: { x: 6, y: 0, w: widgetSizeMap.medium, h: widgetHeightMap['stat-card'] },
        props: { defaultValue: 0, icon: 'assistants' },
        visible: true
      },
      {
        id: 'widget-knowledge',
        type: 'stat-card',
        title: t('overview.stats.knowledgeBases'),
        size: 'medium',
        position: { x: 0, y: 2, w: widgetSizeMap.medium, h: widgetHeightMap['stat-card'] },
        props: { defaultValue: 0, icon: 'knowledgeBases' },
        visible: true
      },
      {
        id: 'widget-calls',
        type: 'stat-card',
        title: t('overview.stats.calls'),
        size: 'medium',
        position: { x: 6, y: 2, w: widgetSizeMap.medium, h: widgetHeightMap['stat-card'] },
        props: { defaultValue: 0, icon: 'calls' },
        visible: true
      },
      {
        id: 'widget-workflows',
        type: 'stat-card',
        title: '工作流',
        size: 'medium',
        position: { x: 0, y: 4, w: widgetSizeMap.medium, h: widgetHeightMap['stat-card'] },
        props: { dataKey: 'totalWorkflows', defaultValue: 0, icon: 'GitBranch' },
        visible: true
      },
      {
        id: 'widget-devices',
        type: 'stat-card',
        title: '设备',
        size: 'medium',
        position: { x: 6, y: 4, w: widgetSizeMap.medium, h: widgetHeightMap['stat-card'] },
        props: { dataKey: 'totalDevices', defaultValue: 0, icon: 'Smartphone' },
        visible: true
      },
      {
        id: 'widget-activity',
        type: 'activity-feed',
        title: t('overview.recentActivity.title'),
        size: 'full',
        position: { x: 0, y: 4, w: widgetSizeMap.full, h: widgetHeightMap['activity-feed'] },
        props: {},
        visible: true
      }
    ]

    return defaultWidgets
  }


  // 如果没有选择组织，显示提示
  if (!currentOrganizationId) {
    return (
      <div className="container mx-auto px-4 py-8">
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-16">
            <Building2 className="w-16 h-16 text-muted-foreground mb-4" />
            <h2 className="text-xl font-semibold mb-2">{t('overview.noOrganization.title')}</h2>
            <p className="text-muted-foreground text-center mb-6">
              {t('overview.noOrganization.description')}
            </p>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (loading) {
    return <OverviewSkeleton />
  }

  if (!config) {
    return (
      <div className="container mx-auto px-4 py-8">
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-16">
            <p className="text-muted-foreground">加载配置失败</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div>
      <DynamicOverview config={config} data={memoizedWidgetData} />
    </div>
  )
}

export default Overview

