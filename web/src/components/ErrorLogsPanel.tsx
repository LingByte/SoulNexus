import React, { useState, useEffect } from 'react';
import { AlertTriangle, CheckCircle2, AlertCircle, Info, X, Filter, RefreshCw } from 'lucide-react';
import Badge from '@/components/UI/Badge';
import { resolveDeviceError } from '@/api/device';
import { showAlert } from '@/utils/notification';

interface ErrorLog {
    id: number;
    deviceId: string;
    macAddress: string;
    errorType: string;
    errorLevel: string;
    errorCode: string;
    errorMsg: string;
    stackTrace: string;
    context: string;
    resolved: boolean;
    resolvedAt?: string;
    resolvedBy?: string;
    createdAt: string;
}

interface ErrorLogsPanelProps {
    errorLogs: ErrorLog[];
    onRefresh: () => void;
    isLoading?: boolean;
}

const ErrorLogsPanel: React.FC<ErrorLogsPanelProps> = ({ errorLogs, onRefresh, isLoading = false }) => {
    const [filteredLogs, setFilteredLogs] = useState<ErrorLog[]>(errorLogs);
    const [selectedErrorType, setSelectedErrorType] = useState<string>('');
    const [selectedErrorLevel, setSelectedErrorLevel] = useState<string>('');
    const [showResolved, setShowResolved] = useState(false);
    const [resolvingId, setResolvingId] = useState<number | null>(null);
    const [expandedId, setExpandedId] = useState<number | null>(null);

    useEffect(() => {
        let filtered = errorLogs;

        if (!showResolved) {
            filtered = filtered.filter(log => !log.resolved);
        }

        if (selectedErrorType) {
            filtered = filtered.filter(log => log.errorType === selectedErrorType);
        }

        if (selectedErrorLevel) {
            filtered = filtered.filter(log => log.errorLevel === selectedErrorLevel);
        }

        setFilteredLogs(filtered);
    }, [errorLogs, selectedErrorType, selectedErrorLevel, showResolved]);

    const getErrorLevelIcon = (level: string) => {
        switch (level.toUpperCase()) {
            case 'FATAL':
                return <AlertTriangle className="w-5 h-5 text-red-600" />;
            case 'ERROR':
                return <AlertCircle className="w-5 h-5 text-red-500" />;
            case 'WARN':
                return <AlertTriangle className="w-5 h-5 text-yellow-500" />;
            case 'INFO':
                return <Info className="w-5 h-5 text-blue-500" />;
            default:
                return <Info className="w-5 h-5 text-gray-500" />;
        }
    };

    const getErrorLevelColor = (level: string) => {
        switch (level.toUpperCase()) {
            case 'FATAL':
                return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300';
            case 'ERROR':
                return 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-400';
            case 'WARN':
                return 'bg-yellow-50 text-yellow-700 dark:bg-yellow-900/20 dark:text-yellow-400';
            case 'INFO':
                return 'bg-blue-50 text-blue-700 dark:bg-blue-900/20 dark:text-blue-400';
            default:
                return 'bg-gray-50 text-gray-700 dark:bg-gray-900/20 dark:text-gray-400';
        }
    };

    const handleResolveError = async (errorId: number) => {
        try {
            setResolvingId(errorId);
            const response = await resolveDeviceError(errorId);
            if (response.code === 200) {
                showAlert('错误已标记为解决', 'success');
                onRefresh();
            } else {
                showAlert(response.msg || '标记失败', 'error');
            }
        } catch (error: any) {
            showAlert(error?.msg || error?.message || '标记失败', 'error');
        } finally {
            setResolvingId(null);
        }
    };

    const errorTypes = Array.from(new Set(errorLogs.map(log => log.errorType)));
    const errorLevels = Array.from(new Set(errorLogs.map(log => log.errorLevel)));

    const unresolvedCount = errorLogs.filter(log => !log.resolved).length;
    const resolvedCount = errorLogs.filter(log => log.resolved).length;

    return (
        <div className="space-y-4">
            {/* 统计信息 */}
            <div className="grid grid-cols-3 gap-4">
                <div className="p-3 bg-red-50 dark:bg-red-900/20 rounded-lg">
                    <div className="text-sm text-gray-600 dark:text-gray-400">未解决</div>
                    <div className="text-2xl font-semibold text-red-600">{unresolvedCount}</div>
                </div>
                <div className="p-3 bg-green-50 dark:bg-green-900/20 rounded-lg">
                    <div className="text-sm text-gray-600 dark:text-gray-400">已解决</div>
                    <div className="text-2xl font-semibold text-green-600">{resolvedCount}</div>
                </div>
                <div className="p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                    <div className="text-sm text-gray-600 dark:text-gray-400">总计</div>
                    <div className="text-2xl font-semibold text-blue-600">{errorLogs.length}</div>
                </div>
            </div>

            {/* 过滤器 */}
            <div className="flex flex-wrap gap-2 items-center p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
                <Filter className="w-4 h-4 text-gray-600 dark:text-gray-400" />
                
                <select
                    value={selectedErrorType}
                    onChange={(e) => setSelectedErrorType(e.target.value)}
                    className="px-3 py-1 text-sm border rounded-md dark:bg-gray-700 dark:border-gray-600 dark:text-gray-100"
                >
                    <option value="">所有类型</option>
                    {errorTypes.map(type => (
                        <option key={type} value={type}>{type}</option>
                    ))}
                </select>

                <select
                    value={selectedErrorLevel}
                    onChange={(e) => setSelectedErrorLevel(e.target.value)}
                    className="px-3 py-1 text-sm border rounded-md dark:bg-gray-700 dark:border-gray-600 dark:text-gray-100"
                >
                    <option value="">所有级别</option>
                    {errorLevels.map(level => (
                        <option key={level} value={level}>{level}</option>
                    ))}
                </select>

                <label className="flex items-center gap-2 text-sm cursor-pointer">
                    <input
                        type="checkbox"
                        checked={showResolved}
                        onChange={(e) => setShowResolved(e.target.checked)}
                        className="rounded"
                    />
                    <span className="text-gray-700 dark:text-gray-300">显示已解决</span>
                </label>

                <button
                    onClick={onRefresh}
                    disabled={isLoading}
                    className="ml-auto p-1 hover:bg-gray-200 dark:hover:bg-gray-700 rounded-md transition-colors disabled:opacity-50"
                >
                    <RefreshCw className={`w-4 h-4 ${isLoading ? 'animate-spin' : ''}`} />
                </button>
            </div>

            {/* 错误日志列表 */}
            <div className="space-y-2 max-h-[600px] overflow-y-auto">
                {filteredLogs.length === 0 ? (
                    <div className="text-center py-8 text-gray-500 dark:text-gray-400">
                        {errorLogs.length === 0 ? '暂无错误日志' : '没有匹配的错误日志'}
                    </div>
                ) : (
                    filteredLogs.map((log) => (
                        <div
                            key={log.id}
                            className={`p-3 rounded-lg border transition-colors ${
                                log.resolved
                                    ? 'bg-gray-50 dark:bg-gray-800 border-gray-200 dark:border-gray-700'
                                    : getErrorLevelColor(log.errorLevel)
                            }`}
                        >
                            <div className="flex items-start justify-between gap-3">
                                <div className="flex items-start gap-3 flex-1 min-w-0">
                                    {getErrorLevelIcon(log.errorLevel)}
                                    <div className="flex-1 min-w-0">
                                        <div className="flex items-center gap-2 mb-1 flex-wrap">
                                            <span className="font-medium text-sm">{log.errorCode}</span>
                                            <Badge variant="muted" size="sm">{log.errorType}</Badge>
                                            <Badge variant="muted" size="sm">{log.errorLevel}</Badge>
                                            {log.resolved && (
                                                <Badge variant="success" size="sm" className="flex items-center gap-1">
                                                    <CheckCircle2 className="w-3 h-3" />
                                                    已解决
                                                </Badge>
                                            )}
                                        </div>
                                        <p className="text-sm break-words">{log.errorMsg}</p>
                                        <div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400 mt-2 flex-wrap">
                                            <span>{new Date(log.createdAt).toLocaleString()}</span>
                                            {log.resolved && log.resolvedAt && (
                                                <span>解决于: {new Date(log.resolvedAt).toLocaleString()}</span>
                                            )}
                                            {log.resolved && log.resolvedBy && (
                                                <span>解决人: {log.resolvedBy}</span>
                                            )}
                                        </div>
                                    </div>
                                </div>

                                <div className="flex items-center gap-2">
                                    {!log.resolved && (
                                        <button
                                            onClick={() => handleResolveError(log.id)}
                                            disabled={resolvingId === log.id}
                                            className="p-1 hover:bg-green-100 dark:hover:bg-green-900/30 rounded-md transition-colors disabled:opacity-50"
                                            title="标记为已解决"
                                        >
                                            {resolvingId === log.id ? (
                                                <RefreshCw className="w-4 h-4 animate-spin" />
                                            ) : (
                                                <CheckCircle2 className="w-4 h-4 text-green-600" />
                                            )}
                                        </button>
                                    )}
                                    <button
                                        onClick={() => setExpandedId(expandedId === log.id ? null : log.id)}
                                        className="p-1 hover:bg-gray-200 dark:hover:bg-gray-700 rounded-md transition-colors"
                                        title="查看详情"
                                    >
                                        {expandedId === log.id ? (
                                            <X className="w-4 h-4" />
                                        ) : (
                                            <AlertTriangle className="w-4 h-4" />
                                        )}
                                    </button>
                                </div>
                            </div>

                            {/* 展开详情 */}
                            {expandedId === log.id && (
                                <div className="mt-3 pt-3 border-t border-gray-200 dark:border-gray-700 space-y-2">
                                    {log.stackTrace && (
                                        <div>
                                            <div className="text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">堆栈跟踪:</div>
                                            <pre className="text-xs bg-gray-100 dark:bg-gray-900 p-2 rounded overflow-x-auto max-h-32 overflow-y-auto">
                                                {log.stackTrace}
                                            </pre>
                                        </div>
                                    )}
                                    {log.context && (
                                        <div>
                                            <div className="text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">上下文:</div>
                                            <pre className="text-xs bg-gray-100 dark:bg-gray-900 p-2 rounded overflow-x-auto max-h-32 overflow-y-auto">
                                                {log.context}
                                            </pre>
                                        </div>
                                    )}
                                </div>
                            )}
                        </div>
                    ))
                )}
            </div>
        </div>
    );
};

export default ErrorLogsPanel;
