import { useMemo } from 'react'
import {
  LineChart,
  Line,
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer
} from 'recharts'
import { useI18nStore } from '@/stores/i18nStore'
import Card, { CardContent, CardHeader, CardTitle } from '@/components/UI/Card'
import type { DailyUsageData, UsageStatistics } from '@/api/billing'

interface UsageChartsProps {
  dailyData: DailyUsageData[]
  statistics: UsageStatistics | null
}

const COLORS = ['#3b82f6', '#ef4444', '#10b981', '#f59e0b', '#8b5cf6', '#06b6d4']

// 格式化数字
const formatNumber = (num: number): string => {
  if (num >= 1000000) {
    return (num / 1000000).toFixed(1) + 'M'
  }
  if (num >= 1000) {
    return (num / 1000).toFixed(1) + 'K'
  }
  return num.toString()
}

// 格式化时长（秒）
const formatDuration = (seconds: number): string => {
  if (seconds < 60) {
    return `${seconds}秒`
  }
  if (seconds < 3600) {
    return `${Math.floor(seconds / 60)}分钟`
  }
  return `${Math.floor(seconds / 3600)}小时`
}

export default function UsageCharts({ dailyData, statistics }: UsageChartsProps) {
  const { t } = useI18nStore()

  // 准备时间趋势图数据
  const trendData = useMemo(() => {
    return dailyData.map(item => ({
      date: new Date(item.date).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' }),
      llmCalls: item.llmCalls,
      llmTokens: item.llmTokens,
      asrCount: item.asrCount,
      asrDuration: item.asrDuration,
      ttsCount: item.ttsCount,
      ttsDuration: item.ttsDuration
    }))
  }, [dailyData])

  // 准备使用类型分布数据（饼图）
  const usageTypeDistribution = useMemo(() => {
    if (!statistics) return []
    
    return [
      {
        name: t('billing.usageType.llm'),
        value: statistics.llmCalls,
        color: COLORS[0]
      },
      {
        name: t('billing.usageType.asr'),
        value: statistics.asrCount,
        color: COLORS[1]
      },
      {
        name: t('billing.usageType.tts'),
        value: statistics.ttsCount,
        color: COLORS[2]
      },
      {
        name: t('billing.usageType.api'),
        value: statistics.apiCalls,
        color: COLORS[3]
      }
    ].filter(item => item.value > 0)
  }, [statistics, t])

  // 准备Token使用趋势数据
  const tokenTrendData = useMemo(() => {
    return dailyData.map(item => ({
      date: new Date(item.date).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' }),
      tokens: item.llmTokens
    }))
  }, [dailyData])

  // 准备时长趋势数据
  const durationTrendData = useMemo(() => {
    return dailyData.map(item => ({
      date: new Date(item.date).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' }),
      asrDuration: item.asrDuration,
      ttsDuration: item.ttsDuration
    }))
  }, [dailyData])

  return (
    <div className="space-y-6">
      {/* 使用量趋势图 */}
      <Card>
        <CardHeader>
          <CardTitle>{t('billing.charts.usageTrend')}</CardTitle>
        </CardHeader>
        <CardContent>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={trendData}>
              <CartesianGrid strokeDasharray="3 3" className="stroke-gray-200 dark:stroke-gray-700" />
              <XAxis 
                dataKey="date" 
                className="text-xs"
                tick={{ fill: 'currentColor' }}
              />
              <YAxis 
                className="text-xs"
                tick={{ fill: 'currentColor' }}
              />
              <Tooltip 
                contentStyle={{
                  backgroundColor: 'var(--color-bg-secondary)',
                  border: '1px solid var(--color-border)',
                  borderRadius: '8px'
                }}
              />
              <Legend />
              <Line 
                type="monotone" 
                dataKey="llmCalls" 
                stroke={COLORS[0]} 
                name={t('billing.charts.llmCalls')}
                strokeWidth={2}
              />
              <Line 
                type="monotone" 
                dataKey="asrCount" 
                stroke={COLORS[1]} 
                name={t('billing.charts.asrCount')}
                strokeWidth={2}
              />
              <Line 
                type="monotone" 
                dataKey="ttsCount" 
                stroke={COLORS[2]} 
                name={t('billing.charts.ttsCount')}
                strokeWidth={2}
              />
            </LineChart>
          </ResponsiveContainer>
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Token使用趋势 */}
        <Card>
          <CardHeader>
            <CardTitle>{t('billing.charts.tokenTrend')}</CardTitle>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={tokenTrendData}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-gray-200 dark:stroke-gray-700" />
                <XAxis 
                  dataKey="date" 
                  className="text-xs"
                  tick={{ fill: 'currentColor' }}
                />
                <YAxis 
                  className="text-xs"
                  tick={{ fill: 'currentColor' }}
                  tickFormatter={formatNumber}
                />
                <Tooltip 
                  contentStyle={{
                    backgroundColor: 'var(--color-bg-secondary)',
                    border: '1px solid var(--color-border)',
                    borderRadius: '8px'
                  }}
                  formatter={(value: number) => formatNumber(value)}
                />
                <Bar dataKey="tokens" fill={COLORS[0]} name={t('billing.charts.tokens')} />
              </BarChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        {/* 使用类型分布 */}
        <Card>
          <CardHeader>
            <CardTitle>{t('billing.charts.usageDistribution')}</CardTitle>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={300}>
              <PieChart>
                <Pie
                  data={usageTypeDistribution}
                  cx="50%"
                  cy="50%"
                  labelLine={false}
                  label={({ name, percent }) => `${name}: ${(percent as number * 100).toFixed(0)}%`}
                  outerRadius={80}
                  fill="#8884d8"
                  dataKey="value"
                >
                  {usageTypeDistribution.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.color} />
                  ))}
                </Pie>
                <Tooltip 
                  contentStyle={{
                    backgroundColor: 'var(--color-bg-secondary)',
                    border: '1px solid var(--color-border)',
                    borderRadius: '8px'
                  }}
                />
              </PieChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>
      </div>

      {/* 时长趋势图 */}
      <Card>
        <CardHeader>
          <CardTitle>{t('billing.charts.durationTrend')}</CardTitle>
        </CardHeader>
        <CardContent>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={durationTrendData}>
              <CartesianGrid strokeDasharray="3 3" className="stroke-gray-200 dark:stroke-gray-700" />
              <XAxis 
                dataKey="date" 
                className="text-xs"
                tick={{ fill: 'currentColor' }}
              />
              <YAxis 
                className="text-xs"
                tick={{ fill: 'currentColor' }}
                tickFormatter={(value: number) => formatDuration(value)}
              />
              <Tooltip 
                contentStyle={{
                  backgroundColor: 'var(--color-bg-secondary)',
                  border: '1px solid var(--color-border)',
                  borderRadius: '8px'
                }}
                formatter={(value: number) => formatDuration(value)}
              />
              <Legend />
              <Line 
                type="monotone" 
                dataKey="asrDuration" 
                stroke={COLORS[1]} 
                name={t('billing.charts.asrDuration')}
                strokeWidth={2}
              />
              <Line 
                type="monotone" 
                dataKey="ttsDuration" 
                stroke={COLORS[2]} 
                name={t('billing.charts.ttsDuration')}
                strokeWidth={2}
              />
            </LineChart>
          </ResponsiveContainer>
        </CardContent>
      </Card>
    </div>
  )
}

