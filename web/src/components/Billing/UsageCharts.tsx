import { useMemo } from 'react'
import {
  LineChart, Line, BarChart, Bar, PieChart, Pie, Cell,
  XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer
} from 'recharts'
import { useI18nStore } from '@/stores/i18nStore'
import type { DailyUsageData, UsageStatistics } from '@/api/billing'

interface UsageChartsProps {
  dailyData: DailyUsageData[]
  statistics: UsageStatistics | null
}

const COLORS = ['#3b82f6', '#ef4444', '#10b981', '#f59e0b']

const formatNumber = (num: number) => {
  if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
  return num.toString()
}

const formatDuration = (seconds: number) => {
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`
  return `${Math.floor(seconds / 3600)}h`
}

const tooltipStyle = {
  backgroundColor: 'rgba(255,255,255,0.95)',
  border: '1px solid #e5e7eb',
  borderRadius: '8px',
  boxShadow: '0 4px 12px rgba(0,0,0,0.08)',
  fontSize: 12,
}

export default function UsageCharts({ dailyData, statistics }: UsageChartsProps) {
  const { t } = useI18nStore()

  const trendData = useMemo(() => dailyData.map(d => ({
    date: new Date(d.date).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' }),
    llm: d.llmCalls, asr: d.asrCount, tts: d.ttsCount,
  })), [dailyData])

  const tokenData = useMemo(() => dailyData.map(d => ({
    date: new Date(d.date).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' }),
    tokens: d.llmTokens,
  })), [dailyData])

  const durationData = useMemo(() => dailyData.map(d => ({
    date: new Date(d.date).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' }),
    asr: d.asrDuration, tts: d.ttsDuration,
  })), [dailyData])

  const pieData = useMemo(() => {
    if (!statistics) return []
    return [
      { name: t('billing.usageType.llm'), value: statistics.llmCalls, color: COLORS[0] },
      { name: t('billing.usageType.asr'), value: statistics.asrCount, color: COLORS[1] },
      { name: t('billing.usageType.tts'), value: statistics.ttsCount, color: COLORS[2] },
      { name: t('billing.usageType.api'), value: statistics.apiCalls, color: COLORS[3] },
    ].filter(i => i.value > 0)
  }, [statistics, t])

  if (!dailyData.length) return null

  return (
    <div className="space-y-4">
      {/* Usage Trend */}
      <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl p-4">
        <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">{t('billing.charts.usageTrend')}</h4>
        <ResponsiveContainer width="100%" height={220}>
          <LineChart data={trendData}>
            <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
            <XAxis dataKey="date" tick={{ fontSize: 11 }} />
            <YAxis tick={{ fontSize: 11 }} />
            <Tooltip contentStyle={tooltipStyle} />
            <Legend iconSize={10} wrapperStyle={{ fontSize: 11 }} />
            <Line type="monotone" dataKey="llm" stroke={COLORS[0]} name={t('billing.charts.llmCalls')} strokeWidth={2} dot={false} />
            <Line type="monotone" dataKey="asr" stroke={COLORS[1]} name={t('billing.charts.asrCount')} strokeWidth={2} dot={false} />
            <Line type="monotone" dataKey="tts" stroke={COLORS[2]} name={t('billing.charts.ttsCount')} strokeWidth={2} dot={false} />
          </LineChart>
        </ResponsiveContainer>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Token Trend */}
        <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl p-4">
          <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">{t('billing.charts.tokenTrend')}</h4>
          <ResponsiveContainer width="100%" height={200}>
            <BarChart data={tokenData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
              <XAxis dataKey="date" tick={{ fontSize: 11 }} />
              <YAxis tick={{ fontSize: 11 }} tickFormatter={formatNumber} />
              <Tooltip contentStyle={tooltipStyle} formatter={(v: number) => formatNumber(v)} />
              <Bar dataKey="tokens" fill={COLORS[0]} radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>

        {/* Usage Distribution */}
        <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl p-4">
          <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">{t('billing.charts.usageDistribution')}</h4>
          <ResponsiveContainer width="100%" height={200}>
            <PieChart>
              <Pie data={pieData} cx="50%" cy="50%" innerRadius={50} outerRadius={80} paddingAngle={2} dataKey="value">
                {pieData.map((e, i) => <Cell key={i} fill={e.color} />)}
              </Pie>
              <Tooltip contentStyle={tooltipStyle} />
              <Legend iconSize={10} wrapperStyle={{ fontSize: 11 }} />
            </PieChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Duration Trend */}
      <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl p-4">
        <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">{t('billing.charts.durationTrend')}</h4>
        <ResponsiveContainer width="100%" height={220}>
          <LineChart data={durationData}>
            <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
            <XAxis dataKey="date" tick={{ fontSize: 11 }} />
            <YAxis tick={{ fontSize: 11 }} tickFormatter={formatDuration} />
            <Tooltip contentStyle={tooltipStyle} formatter={(v: number) => formatDuration(v)} />
            <Legend iconSize={10} wrapperStyle={{ fontSize: 11 }} />
            <Line type="monotone" dataKey="asr" stroke={COLORS[1]} name={t('billing.charts.asrDuration')} strokeWidth={2} dot={false} />
            <Line type="monotone" dataKey="tts" stroke={COLORS[2]} name={t('billing.charts.ttsDuration')} strokeWidth={2} dot={false} />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
}
