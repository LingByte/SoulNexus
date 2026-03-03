import React, { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar, PieChart, Pie, Cell, AreaChart, Area } from 'recharts';
import { getCallRecordings } from '@/api/device';
import Card from '@/components/UI/Card';
import Button from '@/components/UI/Button';
import { showAlert } from '@/utils/notification';

const CallRecordingAnalytics: React.FC = () => {
  const { deviceId } = useParams<{ deviceId: string }>();
  const [recordings, setRecordings] = useState<any[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [timeRange, setTimeRange] = useState<'7d' | '30d' | '90d'>('30d');

  useEffect(() => {
    if (deviceId) {
      fetchRecordings();
    }
  }, [deviceId, timeRange]);

  const fetchRecordings = async () => {
    try {
      setIsLoading(true);
      const response = await getCallRecordings({
        macAddress: deviceId,
        page: 1,
        pageSize: 100, // 获取更多数据用于分析
      });

      if (response.code === 200) {
        setRecordings(response.data.recordings);
      }
    } catch (error) {
      console.error('获取通话记录失败:', error);
      showAlert('获取通话记录失败', 'error');
    } finally {
      setIsLoading(false);
    }
  };

  // 准备时间趋势数据
  const prepareTimeTrendData = () => {
    const groupedByDate = recordings.reduce((acc, recording) => {
      const date = new Date(recording.startTime).toLocaleDateString();
      if (!acc[date]) {
        acc[date] = {
          date,
          count: 0,
          totalDuration: 0,
          avgQuality: 0,
          qualitySum: 0,
        };
      }
      acc[date].count++;
      acc[date].totalDuration += recording.duration;
      acc[date].qualitySum += recording.audioQuality;
      acc[date].avgQuality = acc[date].qualitySum / acc[date].count;
      return acc;
    }, {} as any);

    return Object.values(groupedByDate).sort((a: any, b: any) => 
      new Date(a.date).getTime() - new Date(b.date).getTime()
    );
  };

  // 准备质量分布数据
  const prepareQualityDistribution = () => {
    const qualityRanges = {
      '优秀 (90-100%)': 0,
      '良好 (80-90%)': 0,
      '一般 (70-80%)': 0,
      '较差 (60-70%)': 0,
      '很差 (<60%)': 0,
    };

    recordings.forEach(recording => {
      const quality = recording.audioQuality * 100;
      if (quality >= 90) qualityRanges['优秀 (90-100%)']++;
      else if (quality >= 80) qualityRanges['良好 (80-90%)']++;
      else if (quality >= 70) qualityRanges['一般 (70-80%)']++;
      else if (quality >= 60) qualityRanges['较差 (60-70%)']++;
      else qualityRanges['很差 (<60%)']++;
    });

    return Object.entries(qualityRanges).map(([name, value]) => ({ name, value }));
  };

  // 准备通话时长分布数据
  const prepareDurationDistribution = () => {
    const durationRanges = {
      '< 30秒': 0,
      '30秒-1分钟': 0,
      '1-3分钟': 0,
      '3-5分钟': 0,
      '> 5分钟': 0,
    };

    recordings.forEach(recording => {
      const duration = recording.duration;
      if (duration < 30) durationRanges['< 30秒']++;
      else if (duration < 60) durationRanges['30秒-1分钟']++;
      else if (duration < 180) durationRanges['1-3分钟']++;
      else if (duration < 300) durationRanges['3-5分钟']++;
      else durationRanges['> 5分钟']++;
    });

    return Object.entries(durationRanges).map(([name, value]) => ({ name, value }));
  };

  // 准备通话状态统计
  const prepareStatusStats = () => {
    const statusCount = recordings.reduce((acc, recording) => {
      const status = recording.callStatus;
      acc[status] = (acc[status] || 0) + 1;
      return acc;
    }, {} as any);

    return Object.entries(statusCount).map(([name, value]) => ({ 
      name: name === 'completed' ? '完成' : name === 'interrupted' ? '中断' : '错误', 
      value 
    }));
  };

  const timeTrendData = prepareTimeTrendData();
  const qualityDistribution = prepareQualityDistribution();
  const durationDistribution = prepareDurationDistribution();
  const statusStats = prepareStatusStats();

  const COLORS = ['#10B981', '#3B82F6', '#F59E0B', '#EF4444', '#8B5CF6'];

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-500">加载中...</div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* 页面标题和控制 */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
            通话记录分析
          </h1>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            设备 {deviceId} 的通话数据分析和趋势
          </p>
        </div>
        <div className="flex gap-2">
          {['7d', '30d', '90d'].map((range) => (
            <Button
              key={range}
              variant={timeRange === range ? 'primary' : 'outline'}
              size="sm"
              onClick={() => setTimeRange(range as any)}
            >
              {range === '7d' ? '7天' : range === '30d' ? '30天' : '90天'}
            </Button>
          ))}
        </div>
      </div>

      {/* 统计概览 */}
      <div className="grid grid-cols-4 gap-4">
        <Card variant="outlined" padding="md">
          <div className="text-center">
            <div className="text-2xl font-bold text-blue-600">{recordings.length}</div>
            <div className="text-sm text-gray-600 dark:text-gray-400">总通话数</div>
          </div>
        </Card>
        <Card variant="outlined" padding="md">
          <div className="text-center">
            <div className="text-2xl font-bold text-green-600">
              {Math.round(recordings.reduce((sum, r) => sum + r.duration, 0) / 60)}
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400">总时长(分钟)</div>
          </div>
        </Card>
        <Card variant="outlined" padding="md">
          <div className="text-center">
            <div className="text-2xl font-bold text-purple-600">
              {recordings.length > 0 ? Math.round(recordings.reduce((sum, r) => sum + r.audioQuality, 0) / recordings.length * 100) : 0}%
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400">平均质量</div>
          </div>
        </Card>
        <Card variant="outlined" padding="md">
          <div className="text-center">
            <div className="text-2xl font-bold text-orange-600">
              {recordings.length > 0 ? Math.round(recordings.reduce((sum, r) => sum + r.duration, 0) / recordings.length) : 0}s
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400">平均时长</div>
          </div>
        </Card>
      </div>

      {/* 图表区域 */}
      <div className="grid grid-cols-2 gap-6">
        {/* 时间趋势图 */}
        <Card variant="outlined" padding="md">
          <h3 className="text-lg font-semibold mb-4">通话趋势</h3>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={timeTrendData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="date" />
                <YAxis />
                <Tooltip />
                <Area type="monotone" dataKey="count" stroke="#3B82F6" fill="#3B82F6" fillOpacity={0.3} name="通话次数" />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </Card>

        {/* 音频质量分布 */}
        <Card variant="outlined" padding="md">
          <h3 className="text-lg font-semibold mb-4">音频质量分布</h3>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie
                  data={qualityDistribution}
                  cx="50%"
                  cy="50%"
                  outerRadius={80}
                  fill="#8884d8"
                  dataKey="value"
                  label={({ name, value }) => `${name}: ${value}`}
                >
                  {qualityDistribution.map((_, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </div>
        </Card>

        {/* 通话时长分布 */}
        <Card variant="outlined" padding="md">
          <h3 className="text-lg font-semibold mb-4">通话时长分布</h3>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={durationDistribution}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="name" />
                <YAxis />
                <Tooltip />
                <Bar dataKey="value" fill="#10B981" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </Card>

        {/* 通话状态统计 */}
        <Card variant="outlined" padding="md">
          <h3 className="text-lg font-semibold mb-4">通话状态统计</h3>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie
                  data={statusStats}
                  cx="50%"
                  cy="50%"
                  outerRadius={80}
                  fill="#8884d8"
                  dataKey="value"
                  label={({ name, value }) => `${name}: ${value}`}
                >
                  {statusStats.map((_, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </div>
        </Card>
      </div>

      {/* 质量趋势图 */}
      <Card variant="outlined" padding="md">
        <h3 className="text-lg font-semibold mb-4">音频质量趋势</h3>
        <div className="h-64">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={timeTrendData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="date" />
              <YAxis domain={[0, 1]} tickFormatter={(value) => `${(value * 100).toFixed(0)}%`} />
              <Tooltip formatter={(value: any) => [`${(value * 100).toFixed(1)}%`, '平均质量']} />
              <Line type="monotone" dataKey="avgQuality" stroke="#8B5CF6" strokeWidth={2} name="平均质量" />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </Card>
    </div>
  );
};

export default CallRecordingAnalytics;