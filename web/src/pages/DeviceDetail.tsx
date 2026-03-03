import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
    ArrowLeft,
    Activity,
    Clock,
    AlertTriangle,
    Thermometer,
    Cpu,
    MemoryStick,
    Wifi,
    Headphones,
    Mic,
    PlayCircle,
    Calendar,
    Volume2,
    FileAudio,
    Eye,
    Brain,
    Loader2,
    CheckCircle,
    XCircle,
    RefreshCw,
    BarChart3
} from 'lucide-react';
import {
    getDeviceDetail,
    getDeviceErrorLogs,
    getCallRecordings,
    getCallRecordingDetail,
    analyzeCallRecording,
    batchAnalyzeCallRecordings,
    getCallRecordingAnalysis,
    type Device
} from '@/api/device';
import { showAlert } from '@/utils/notification';
import { useI18nStore } from '@/stores/i18nStore';
import Button from '@/components/UI/Button';
import Card from '@/components/UI/Card';
import Badge from '@/components/UI/Badge';
import Modal from '@/components/UI/Modal';
import CallRecordingDetail from '@/components/CallRecordingDetail';
import ErrorLogsPanel from '@/components/ErrorLogsPanel';

interface CallRecording {
    id: number;
    sessionId: string;
    storageUrl?: string; // 云存储URL
    audioFormat: string;
    audioSize: number;
    duration: number;
    callType: string;
    callStatus: string;
    startTime: string;
    endTime: string;
    summary: string;
    audioQuality: number;
    noiseLevel: number;
    createdAt: string;
    // 对话相关字段
    totalTurns?: number;
    userTurns?: number;
    aiTurns?: number;
    // 提供商信息
    llmModel?: string;
    ttsProvider?: string;
    asrProvider?: string;
    // AI分析相关字段
    aiAnalysis?: string;
    analysisStatus: 'pending' | 'analyzing' | 'completed' | 'failed';
    analysisError?: string;
    analyzedAt?: string;
    autoAnalyzed: boolean;
    analysisVersion: number;
}

interface ErrorLog {
    id: number;
    errorType: string;
    errorLevel: string;
    errorCode: string;
    errorMsg: string;
    resolved: boolean;
    createdAt: string;
}

// interface PerformanceData {
//   id: number;
//   cpuUsage: number;
//   memoryUsage: number;
//   temperature: number;
//   networkLatency: number;
//   recordedAt: string;
// }

const DeviceDetail: React.FC = () => {
    const { deviceId } = useParams<{ deviceId: string }>();
    const navigate = useNavigate();
    const { t } = useI18nStore();

    const [device, setDevice] = useState<Device | null>(null);
    const [callRecordings, setCallRecordings] = useState<CallRecording[]>([]);
    const [errorLogs, setErrorLogs] = useState<ErrorLog[]>([]);
    // const [performanceData, setPerformanceData] = useState<PerformanceData[]>([]);
    const [isLoading, setIsLoading] = useState(true);

    // 模态框状态
    const [showRecordingModal, setShowRecordingModal] = useState(false);
    const [selectedRecording, setSelectedRecording] = useState<CallRecording | null>(null);
    const [selectedRecordingDetail, setSelectedRecordingDetail] = useState<any>(null);

    // AI分析相关状态
    const [analyzingRecordings, setAnalyzingRecordings] = useState<Set<number>>(new Set());
    const [analysisResults, setAnalysisResults] = useState<Map<number, any>>(new Map());
    const [showAnalysisModal, setShowAnalysisModal] = useState(false);
    const [selectedAnalysis, setSelectedAnalysis] = useState<any>(null);
    const [isBatchAnalyzing, setIsBatchAnalyzing] = useState(false);

    useEffect(() => {
        if (deviceId) {
            fetchDeviceData();
        }
    }, [deviceId]);

    const fetchDeviceData = async () => {
        if (!deviceId) return;

        try {
            setIsLoading(true);

            // 并行获取所有数据
            const [deviceRes, recordingsRes, errorsRes] = await Promise.all([
                getDeviceDetail(deviceId),
                getCallRecordings({ macAddress: deviceId, page: 1, pageSize: 20 }),
                getDeviceErrorLogs(deviceId, 1, 20)
            ]);

            if (deviceRes.code === 200) {
                setDevice(deviceRes.data);
            }

            if (recordingsRes.code === 200) {
                // 为录音数据添加默认的AI分析字段
                const recordingsWithAnalysis = recordingsRes.data.recordings.map(recording => ({
                    ...recording,
                    analysisStatus: recording.analysisStatus || 'pending' as const,
                    autoAnalyzed: recording.autoAnalyzed || false,
                    analysisVersion: recording.analysisVersion || 1,
                }));
                setCallRecordings(recordingsWithAnalysis);
            }

            if (errorsRes.code === 200) {
                setErrorLogs(errorsRes.data.logs);
            }
        } catch (error: any) {
            showAlert(error?.msg || error?.message || t('device.messages.fetchDevicesFailed'), 'error');
        } finally {
            setIsLoading(false);
        }
    };

    const formatUptime = (seconds: number) => {
        if (!seconds) return 'N/A';
        const hours = Math.floor(seconds / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        if (hours > 0) {
            return `${hours}h ${minutes}m`;
        }
        return `${minutes}m`;
    };

    const formatFileSize = (bytes: number) => {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    };

    const formatDuration = (seconds: number) => {
        const mins = Math.floor(seconds / 60);
        const secs = seconds % 60;
        return `${mins}:${secs.toString().padStart(2, '0')}`;
    };

    const getStatusColor = (isOnline: boolean) => {
        return isOnline ? 'text-green-500' : 'text-red-500';
    };

    const getPerformanceColor = (value: number, type: 'cpu' | 'memory' | 'temperature') => {
        if (type === 'temperature') {
            if (value > 80) return 'text-red-500';
            if (value > 60) return 'text-yellow-500';
            return 'text-green-500';
        }
        if (value > 80) return 'text-red-500';
        if (value > 60) return 'text-yellow-500';
        return 'text-green-500';
    };

    const handlePlayRecording = async (recording: CallRecording) => {
        setSelectedRecording(recording);
        setShowRecordingModal(true);

        // 获取详细的通话记录数据
        try {
            const response = await getCallRecordingDetail(recording.id);
            if (response.code === 200) {
                setSelectedRecordingDetail(response.data);
            }
        } catch (error) {
            console.error('获取通话记录详情失败:', error);
        }
    };

    // AI分析相关函数
    const handleAnalyzeRecording = async (recording: CallRecording, force = false) => {
        try {
            setAnalyzingRecordings(prev => new Set(prev).add(recording.id));

            const response = await analyzeCallRecording(recording.id, force);
            if (response.code === 200) {
                showAlert('分析已启动，请稍后查看结果', 'success');

                // 轮询检查分析结果
                pollAnalysisResult(recording.id);
            } else {
                showAlert(response.msg || '启动分析失败', 'error');
            }
        } catch (error: any) {
            showAlert(error?.msg || error?.message || '启动分析失败', 'error');
        } finally {
            setAnalyzingRecordings(prev => {
                const newSet = new Set(prev);
                newSet.delete(recording.id);
                return newSet;
            });
        }
    };

    const pollAnalysisResult = async (recordingId: number) => {
        const maxAttempts = 30; // 最多轮询30次（约5分钟）
        let attempts = 0;

        const poll = async () => {
            try {
                const response = await getCallRecordingAnalysis(recordingId);
                if (response.code === 200) {
                    const { analysisStatus, analysis } = response.data;

                    if (analysisStatus === 'completed' && analysis) {
                        setAnalysisResults(prev => new Map(prev).set(recordingId, analysis));
                        showAlert('AI分析完成', 'success');
                        return;
                    } else if (analysisStatus === 'failed') {
                        showAlert('AI分析失败', 'error');
                        return;
                    }
                }

                attempts++;
                if (attempts < maxAttempts) {
                    setTimeout(poll, 10000); // 10秒后重试
                }
            } catch (error) {
                console.error('轮询分析结果失败:', error);
            }
        };

        poll();
    };

    const handleViewAnalysis = async (recording: CallRecording) => {
        try {
            // 先检查本地缓存
            const cachedResult = analysisResults.get(recording.id);
            if (cachedResult) {
                setSelectedAnalysis({ recording, analysis: cachedResult });
                setShowAnalysisModal(true);
                return;
            }

            // 从服务器获取分析结果
            const response = await getCallRecordingAnalysis(recording.id);
            if (response.code === 200 && response.data.analysis) {
                setSelectedAnalysis({ recording, analysis: response.data.analysis });
                setShowAnalysisModal(true);
                setAnalysisResults(prev => new Map(prev).set(recording.id, response.data.analysis));
            } else {
                showAlert('暂无分析结果', 'info');
            }
        } catch (error: any) {
            showAlert(error?.msg || error?.message || '获取分析结果失败', 'error');
        }
    };

    const handleBatchAnalyze = async () => {
        if (isBatchAnalyzing) {
            return; // 防止重复点击
        }

        try {
            setIsBatchAnalyzing(true);
            const response = await batchAnalyzeCallRecordings({
                assistantId: device?.assistantId,
                limit: 10
            });
            if (response.code === 200) {
                showAlert('批量分析已启动', 'success');
                // 刷新录音列表
                fetchDeviceData();
            } else {
                showAlert(response.msg || '批量分析启动失败', 'error');
            }
        } catch (error: any) {
            showAlert(error?.msg || error?.message || '批量分析启动失败', 'error');
        } finally {
            // 延迟重置状态，防止用户立即重复点击
            setTimeout(() => {
                setIsBatchAnalyzing(false);
            }, 3000);
        }
    };

    const getAnalysisStatusIcon = (recording: CallRecording) => {
        const isAnalyzing = analyzingRecordings.has(recording.id);

        if (isAnalyzing) {
            return <Loader2 className="w-4 h-4 animate-spin text-blue-500" />;
        }

        switch (recording.analysisStatus) {
            case 'completed':
                return <CheckCircle className="w-4 h-4 text-green-500" />;
            case 'failed':
                return <XCircle className="w-4 h-4 text-red-500" />;
            case 'analyzing':
                return <Loader2 className="w-4 h-4 animate-spin text-blue-500" />;
            default:
                return <Brain className="w-4 h-4 text-gray-400" />;
        }
    };

    const getAnalysisStatusText = (recording: CallRecording) => {
        const isAnalyzing = analyzingRecordings.has(recording.id);

        if (isAnalyzing) return '分析中...';

        switch (recording.analysisStatus) {
            case 'completed':
                return '已分析';
            case 'failed':
                return '分析失败';
            case 'analyzing':
                return '分析中...';
            default:
                return '待分析';
        }
    };

    if (isLoading) {
        return (
            <div className="min-h-screen dark:bg-neutral-900 flex items-center justify-center">
                <div className="text-center">
                    <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto mb-4"></div>
                    <p className="text-gray-600 dark:text-gray-400">{t('device.loading')}</p>
                </div>
            </div>
        );
    }

    if (!device) {
        return (
            <div className="min-h-screen dark:bg-neutral-900 flex items-center justify-center">
                <div className="text-center">
                    <p className="text-gray-600 dark:text-gray-400">设备不存在</p>
                    <Button onClick={() => navigate('/devices')} className="mt-4">
                        返回设备管理
                    </Button>
                </div>
            </div>
        );
    }

    return (
        <div className="min-h-screen dark:bg-neutral-900">
            <div className="max-w-7xl mx-auto pt-6 pb-4 px-4">
                {/* 头部导航 */}
                <div className="flex items-center gap-4 mb-6">
                    <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => navigate('/devices')}
                    >
                        <ArrowLeft className="w-5 h-5" />
                    </Button>
                    <div>
                        <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100">
                            {device.deviceName || device.macAddress}
                        </h1>
                        <p className="text-sm text-gray-500 dark:text-gray-400">
                            设备详情和通话记录
                        </p>
                    </div>
                </div>

                <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                    {/* 左侧：设备信息 */}
                    <div className="lg:col-span-1 space-y-6">
                        {/* 基本信息 */}
                        <Card variant="outlined" padding="md">
                            <h3 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">
                                基本信息
                            </h3>
                            <div className="space-y-3">
                                <div className="flex items-center gap-3">
                                    <Activity className={`w-5 h-5 ${getStatusColor(device.isOnline)}`} />
                                    <span className="text-sm text-gray-600 dark:text-gray-400">状态:</span>
                                    <Badge variant={device.isOnline ? 'success' : 'muted'}>
                                        {device.isOnline ? t('device.online') : t('device.offline')}
                                    </Badge>
                                </div>

                                <div className="flex items-center gap-3">
                                    <Wifi className="w-5 h-5 text-gray-500" />
                                    <span className="text-sm text-gray-600 dark:text-gray-400">MAC:</span>
                                    <span className="text-sm font-mono text-gray-900 dark:text-gray-100">
                    {device.macAddress}
                  </span>
                                </div>

                                {device.board && (
                                    <div className="flex items-center gap-3">
                                        <Cpu className="w-5 h-5 text-gray-500" />
                                        <span className="text-sm text-gray-600 dark:text-gray-400">型号:</span>
                                        <span className="text-sm text-gray-900 dark:text-gray-100">
                      {device.board}
                    </span>
                                    </div>
                                )}

                                {device.appVersion && (
                                    <div className="flex items-center gap-3">
                                        <span className="text-sm text-gray-600 dark:text-gray-400">版本:</span>
                                        <span className="text-sm text-gray-900 dark:text-gray-100">
                      {device.appVersion}
                    </span>
                                    </div>
                                )}

                                {device.uptime > 0 && (
                                    <div className="flex items-center gap-3">
                                        <Clock className="w-5 h-5 text-gray-500" />
                                        <span className="text-sm text-gray-600 dark:text-gray-400">运行时长:</span>
                                        <span className="text-sm text-gray-900 dark:text-gray-100">
                      {formatUptime(device.uptime)}
                    </span>
                                    </div>
                                )}

                                {device.errorCount > 0 && (
                                    <div className="flex items-center gap-3">
                                        <AlertTriangle className="w-5 h-5 text-red-500" />
                                        <span className="text-sm text-gray-600 dark:text-gray-400">错误次数:</span>
                                        <span className="text-sm text-red-500">
                      {device.errorCount}
                    </span>
                                    </div>
                                )}
                            </div>
                        </Card>

                        {/* 性能指标 */}
                        {(device.cpuUsage > 0 || device.memoryUsage > 0 || device.temperature > 0) && (
                            <Card variant="outlined" padding="md">
                                <h3 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">
                                    性能指标
                                </h3>
                                <div className="space-y-3">
                                    {device.cpuUsage > 0 && (
                                        <div className="flex items-center justify-between">
                                            <div className="flex items-center gap-2">
                                                <Cpu className={`w-4 h-4 ${getPerformanceColor(device.cpuUsage, 'cpu')}`} />
                                                <span className="text-sm text-gray-600 dark:text-gray-400">CPU</span>
                                            </div>
                                            <span className={`text-sm font-medium ${getPerformanceColor(device.cpuUsage, 'cpu')}`}>
                        {device.cpuUsage.toFixed(1)}%
                      </span>
                                        </div>
                                    )}

                                    {device.memoryUsage > 0 && (
                                        <div className="flex items-center justify-between">
                                            <div className="flex items-center gap-2">
                                                <MemoryStick className={`w-4 h-4 ${getPerformanceColor(device.memoryUsage, 'memory')}`} />
                                                <span className="text-sm text-gray-600 dark:text-gray-400">内存</span>
                                            </div>
                                            <span className={`text-sm font-medium ${getPerformanceColor(device.memoryUsage, 'memory')}`}>
                        {device.memoryUsage.toFixed(1)}%
                      </span>
                                        </div>
                                    )}

                                    {device.temperature > 0 && (
                                        <div className="flex items-center justify-between">
                                            <div className="flex items-center gap-2">
                                                <Thermometer className={`w-4 h-4 ${getPerformanceColor(device.temperature, 'temperature')}`} />
                                                <span className="text-sm text-gray-600 dark:text-gray-400">温度</span>
                                            </div>
                                            <span className={`text-sm font-medium ${getPerformanceColor(device.temperature, 'temperature')}`}>
                        {device.temperature.toFixed(1)}°C
                      </span>
                                        </div>
                                    )}
                                </div>
                            </Card>
                        )}

                        {/* 音频设备状态 */}
                        {device.audioStatus && (
                            <Card variant="outlined" padding="md">
                                <h3 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">
                                    音频设备
                                </h3>
                                <div className="space-y-3">
                                    <div className="flex items-center gap-3">
                                        <Mic className="w-5 h-5 text-green-500" />
                                        <span className="text-sm text-gray-600 dark:text-gray-400">麦克风</span>
                                        <Badge variant="success" size="sm">正常</Badge>
                                    </div>
                                    <div className="flex items-center gap-3">
                                        <Headphones className="w-5 h-5 text-blue-500" />
                                        <span className="text-sm text-gray-600 dark:text-gray-400">扬声器</span>
                                        <Badge variant="success" size="sm">正常</Badge>
                                    </div>
                                </div>
                            </Card>
                        )}
                    </div>

                    {/* 右侧：通话记录和错误日志 */}
                    <div className="lg:col-span-2 space-y-6">
                        {/* 通话记录 */}
                        <Card variant="outlined" padding="md">
                            <div className="flex items-center justify-between mb-4">
                                <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                                    通话记录
                                </h3>
                                <div className="flex items-center gap-2">
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        onClick={() => navigate(`/call-recording-analytics/${deviceId}`)}
                                        className="inline-flex items-center gap-1 whitespace-nowrap"
                                    >
                                        <BarChart3 className="w-4 h-4" />
                                    </Button>
                                    <button
                                        onClick={handleBatchAnalyze}
                                        disabled={isBatchAnalyzing}
                                        className={`inline-flex items-center gap-1 px-3 py-1.5 text-sm rounded-md transition-colors whitespace-nowrap ${
                                            isBatchAnalyzing
                                                ? 'text-gray-400 bg-gray-100 cursor-not-allowed'
                                                : 'text-gray-700 hover:text-gray-900 hover:bg-gray-100'
                                        }`}
                                    >
                                        {isBatchAnalyzing ? (
                                            <Loader2 className="w-4 h-4 animate-spin" />
                                        ) : (
                                            <Brain className="w-4 h-4" />
                                        )}
                                        {isBatchAnalyzing ? '分析中...' : '批量分析'}
                                    </button>
                                    <Badge variant="muted" size="sm">
                                        {callRecordings.length} 条记录
                                    </Badge>
                                </div>
                            </div>

                            {callRecordings.length === 0 ? (
                                <div className="text-center py-8 text-gray-500 dark:text-gray-400">
                                    暂无通话记录
                                </div>
                            ) : (
                                <div className="space-y-3 max-h-96 overflow-y-auto">
                                    {callRecordings.map((recording) => (
                                        <div
                                            key={recording.id}
                                            className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-800 rounded-lg"
                                        >
                                            <div className="flex-1 min-w-0">
                                                <div className="flex items-center gap-2 mb-1">
                                                    <FileAudio className="w-4 h-4 text-blue-500" />
                                                    <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                            {recording.summary || '语音通话'}
                          </span>
                                                    <Badge
                                                        variant={recording.callStatus === 'completed' ? 'success' : 'muted'}
                                                        size="sm"
                                                    >
                                                        {recording.callStatus}
                                                    </Badge>
                                                </div>
                                                <div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400 flex-wrap">
                          <span className="flex items-center gap-1">
                            <Clock className="w-3 h-3" />
                              {formatDuration(recording.duration)}
                          </span>
                                                    <span className="flex items-center gap-1">
                            <Volume2 className="w-3 h-3" />
                                                        {formatFileSize(recording.audioSize)}
                          </span>
                                                    <span className="flex items-center gap-1">
                            <Calendar className="w-3 h-3" />
                                                        {new Date(recording.createdAt).toLocaleString()}
                          </span>
                                                    {recording.totalTurns && (
                                                      <span className="flex items-center gap-1">
                                                        轮次: {recording.totalTurns}
                                                      </span>
                                                    )}
                                                    {recording.llmModel && (
                                                      <span className="flex items-center gap-1">
                                                        LLM: {recording.llmModel}
                                                      </span>
                                                    )}
                                                </div>
                                            </div>
                                            <div className="flex items-center gap-2">
                                                <Button
                                                    variant="ghost"
                                                    size="sm"
                                                    onClick={() => handlePlayRecording(recording)}
                                                >
                                                    <PlayCircle className="w-4 h-4" />
                                                </Button>
                                                <Button
                                                    variant="ghost"
                                                    size="sm"
                                                    onClick={() => handleViewError(recording as any)}
                                                >
                                                    <Eye className="w-4 h-4" />
                                                </Button>
                                                {/* AI分析按钮 */}
                                                <div className="flex items-center gap-1">
                                                    {getAnalysisStatusIcon(recording)}
                                                    {recording.analysisStatus === 'completed' ? (
                                                        <Button
                                                            variant="ghost"
                                                            size="sm"
                                                            onClick={() => handleViewAnalysis(recording)}
                                                            title="查看AI分析结果"
                                                        >
                                                            <Brain className="w-4 h-4" />
                                                        </Button>
                                                    ) : (
                                                        <Button
                                                            variant="ghost"
                                                            size="sm"
                                                            onClick={() => handleAnalyzeRecording(recording)}
                                                            disabled={analyzingRecordings.has(recording.id) || recording.analysisStatus === 'analyzing'}
                                                            title={getAnalysisStatusText(recording)}
                                                        >
                                                            <RefreshCw className="w-4 h-4" />
                                                        </Button>
                                                    )}
                                                </div>
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            )}
                        </Card>

                        {/* 错误日志 */}
                        <Card variant="outlined" padding="md">
                            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
                                设备错误日志
                            </h3>
                            <ErrorLogsPanel
                                errorLogs={errorLogs}
                                onRefresh={fetchDeviceData}
                                isLoading={isLoading}
                            />
                        </Card>
                    </div>
                </div>

                {/* 通话记录详情模态框 */}
                <Modal
                    isOpen={showRecordingModal}
                    onClose={() => {
                        setShowRecordingModal(false);
                        setSelectedRecordingDetail(null);
                    }}
                    title="通话记录详情"
                    size="xl"
                >
                    {selectedRecording && (
                        <CallRecordingDetail
                            recording={selectedRecording}
                            recordingDetail={selectedRecordingDetail}
                            onAnalyze={async (recordingId) => {
                                await handleAnalyzeRecording(selectedRecording);
                            }}
                            onGetAnalysis={async (recordingId) => {
                                try {
                                    const response = await getCallRecordingAnalysis(recordingId);
                                    if (response.code === 200) {
                                        return response.data;
                                    }
                                    throw new Error('获取分析结果失败');
                                } catch (error: any) {
                                    throw error;
                                }
                            }}
                        />
                    )}
                </Modal>

                {/* AI分析结果模态框 */}
                <Modal
                    isOpen={showAnalysisModal}
                    onClose={() => setShowAnalysisModal(false)}
                    title="AI分析结果"
                    size="lg"
                >
                    {selectedAnalysis && (
                        <div className="space-y-6">
                            {/* 基本信息 */}
                            <div className="grid grid-cols-2 gap-4 text-sm">
                                <div>
                                    <span className="text-gray-600 dark:text-gray-400">会话ID:</span>
                                    <p className="font-mono text-xs">{selectedAnalysis.recording.sessionId}</p>
                                </div>
                                <div>
                                    <span className="text-gray-600 dark:text-gray-400">分析时间:</span>
                                    <p>{selectedAnalysis.recording.analyzedAt ? new Date(selectedAnalysis.recording.analyzedAt).toLocaleString() : '未知'}</p>
                                </div>
                            </div>

                            {/* 对话摘要 */}
                            <div>
                                <h4 className="font-medium text-gray-900 dark:text-gray-100 mb-2 flex items-center gap-2">
                                    <FileAudio className="w-4 h-4" />
                                    对话摘要
                                </h4>
                                <div className="p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg text-sm">
                                    {selectedAnalysis.analysis.summary}
                                </div>
                            </div>

                            {/* 分类和重要性 */}
                            <div className="grid grid-cols-2 gap-4">
                                <div>
                                    <h4 className="font-medium text-gray-900 dark:text-gray-100 mb-2">分类</h4>
                                    <Badge variant="muted">{selectedAnalysis.analysis.category}</Badge>
                                </div>
                                <div>
                                    <h4 className="font-medium text-gray-900 dark:text-gray-100 mb-2">重要性</h4>
                                    <Badge variant={selectedAnalysis.analysis.isImportant ? 'error' : 'muted'}>
                                        {selectedAnalysis.analysis.isImportant ? '重要' : '普通'}
                                    </Badge>
                                </div>
                            </div>

                            {/* 关键词和标签 */}
                            <div className="grid grid-cols-2 gap-4">
                                <div>
                                    <h4 className="font-medium text-gray-900 dark:text-gray-100 mb-2">关键词</h4>
                                    <div className="flex flex-wrap gap-1">
                                        {selectedAnalysis.analysis.keywords?.map((keyword: string, index: number) => (
                                            <Badge key={index} variant="outline" size="sm">
                                                {keyword}
                                            </Badge>
                                        ))}
                                    </div>
                                </div>
                                <div>
                                    <h4 className="font-medium text-gray-900 dark:text-gray-100 mb-2">标签</h4>
                                    <div className="flex flex-wrap gap-1">
                                        {selectedAnalysis.analysis.tags?.map((tag: string, index: number) => (
                                            <Badge key={index} variant="secondary" size="sm">
                                                {tag}
                                            </Badge>
                                        ))}
                                    </div>
                                </div>
                            </div>

                            {/* 情感和满意度分数 */}
                            <div className="grid grid-cols-2 gap-4">
                                <div>
                                    <h4 className="font-medium text-gray-900 dark:text-gray-100 mb-2">情感分数</h4>
                                    <div className="flex items-center gap-2">
                                        <div className="flex-1 bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                                            <div
                                                className={`h-2 rounded-full ${
                                                    selectedAnalysis.analysis.sentimentScore > 0 ? 'bg-green-500' :
                                                        selectedAnalysis.analysis.sentimentScore < 0 ? 'bg-red-500' : 'bg-gray-400'
                                                }`}
                                                style={{
                                                    width: `${Math.abs(selectedAnalysis.analysis.sentimentScore) * 50 + 50}%`,
                                                    marginLeft: selectedAnalysis.analysis.sentimentScore < 0 ?
                                                        `${(1 + selectedAnalysis.analysis.sentimentScore) * 50}%` : '0'
                                                }}
                                            />
                                        </div>
                                        <span className="text-sm font-mono">
                      {selectedAnalysis.analysis.sentimentScore?.toFixed(2)}
                    </span>
                                    </div>
                                </div>
                                <div>
                                    <h4 className="font-medium text-gray-900 dark:text-gray-100 mb-2">满意度</h4>
                                    <div className="flex items-center gap-2">
                                        <div className="flex-1 bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                                            <div
                                                className="h-2 bg-blue-500 rounded-full"
                                                style={{ width: `${selectedAnalysis.analysis.satisfactionScore * 100}%` }}
                                            />
                                        </div>
                                        <span className="text-sm font-mono">
                      {(selectedAnalysis.analysis.satisfactionScore * 100).toFixed(0)}%
                    </span>
                                    </div>
                                </div>
                            </div>

                            {/* 行动项 */}
                            {selectedAnalysis.analysis.actionItems?.length > 0 && (
                                <div>
                                    <h4 className="font-medium text-gray-900 dark:text-gray-100 mb-2">行动项</h4>
                                    <ul className="space-y-1">
                                        {selectedAnalysis.analysis.actionItems.map((item: string, index: number) => (
                                            <li key={index} className="text-sm text-gray-700 dark:text-gray-300 flex items-start gap-2">
                                                <span className="text-blue-500 mt-1">•</span>
                                                {item}
                                            </li>
                                        ))}
                                    </ul>
                                </div>
                            )}

                            {/* 问题点 */}
                            {selectedAnalysis.analysis.issues?.length > 0 && (
                                <div>
                                    <h4 className="font-medium text-gray-900 dark:text-gray-100 mb-2">问题点</h4>
                                    <ul className="space-y-1">
                                        {selectedAnalysis.analysis.issues.map((issue: string, index: number) => (
                                            <li key={index} className="text-sm text-gray-700 dark:text-gray-300 flex items-start gap-2">
                                                <span className="text-red-500 mt-1">•</span>
                                                {issue}
                                            </li>
                                        ))}
                                    </ul>
                                </div>
                            )}

                            {/* 深度洞察 */}
                            {selectedAnalysis.analysis.insights && (
                                <div>
                                    <h4 className="font-medium text-gray-900 dark:text-gray-100 mb-2">深度洞察</h4>
                                    <div className="p-3 bg-green-50 dark:bg-green-900/20 rounded-lg text-sm">
                                        {selectedAnalysis.analysis.insights}
                                    </div>
                                </div>
                            )}
                        </div>
                    )}
                </Modal>


            </div>
        </div>
    );
};

export default DeviceDetail;