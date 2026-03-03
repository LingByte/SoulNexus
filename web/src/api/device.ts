import { get, post, put, ApiResponse } from '@/utils/request'

// 设备信息
export interface Device {
    id: string
    userId: number
    macAddress: string
    deviceName?: string
    board?: string
    appVersion?: string
    autoUpdate: number
    assistantId?: number
    alias?: string

    // 运行状态
    isOnline: boolean
    lastSeen?: string
    startTime?: string
    uptime: number
    errorCount: number
    lastError?: string
    lastErrorAt?: string

    // 性能状态
    cpuUsage: number
    memoryUsage: number
    temperature: number

    // 系统信息
    systemInfo?: string
    hardwareInfo?: string
    networkInfo?: string
    audioStatus?: string
    serviceStatus?: string

    // 时间戳
    lastConnected?: string
    createdAt: string
    updatedAt: string
}

// 绑定设备请求
export interface BindDeviceRequest {
    agentId: string
    deviceCode: string
}

// 解绑设备请求
export interface UnbindDeviceRequest {
    deviceId: string
}

// 更新设备信息请求
export interface UpdateDeviceRequest {
    alias?: string
    autoUpdate?: number
}

// 手动添加设备请求
export interface ManualAddDeviceRequest {
    agentId: string
    board: string
    appVersion?: string
    macAddress: string
}

// 绑定设备（激活设备）
export const bindDevice = async (agentId: string, deviceCode: string): Promise<ApiResponse<null>> => {
    return post(`/device/bind/${agentId}/${deviceCode}`, {})
}

// 获取已绑定设备列表
export const getUserDevices = async (agentId: string): Promise<ApiResponse<Device[]>> => {
    return get(`/device/bind/${agentId}`)
}

// 解绑设备
export const unbindDevice = async (data: UnbindDeviceRequest): Promise<ApiResponse<null>> => {
    return post('/device/unbind', data)
}

// 更新设备信息
export const updateDevice = async (deviceId: string, data: UpdateDeviceRequest): Promise<ApiResponse<Device>> => {
    return put(`/device/update/${deviceId}`, data)
}

// 手动添加设备
export const manualAddDevice = async (data: ManualAddDeviceRequest): Promise<ApiResponse<Device>> => {
    return post('/device/manual-add', data)
}

// 获取设备详情
export const getDeviceDetail = async (deviceId: string): Promise<ApiResponse<Device>> => {
    return get(`/device/${deviceId}`)
}

// 获取设备错误日志
export const getDeviceErrorLogs = async (deviceId: string, page = 1, pageSize = 20, errorType?: string, errorLevel?: string): Promise<ApiResponse<{
    logs: Array<{
        id: number
        deviceId: number
        macAddress: string
        errorType: string
        errorLevel: string
        errorCode: string
        errorMsg: string
        stackTrace: string
        context: string
        resolved: boolean
        resolvedAt?: string
        resolvedBy?: string
        createdAt: string
    }>
    total: number
    page: number
    pageSize: number
}>> => {
    let url = `/device/${deviceId}/error-logs?page=${page}&page_size=${pageSize}`
    if (errorType) url += `&error_type=${errorType}`
    if (errorLevel) url += `&error_level=${errorLevel}`
    return get(url)
}

// 标记设备错误为已解决
export const resolveDeviceError = async (errorId: number): Promise<ApiResponse<null>> => {
    return post(`/device/error-logs/${errorId}/resolve`, {})
}

// 获取通话录音列表
export const getCallRecordings = async (params: {
    page?: number
    pageSize?: number
    assistantId?: string
    macAddress?: string
}): Promise<ApiResponse<{
    recordings: Array<{
        id: number
        userId: number
        assistantId: number
        deviceId?: number
        macAddress: string
        sessionId: string
        audioPath: string
        storageUrl?: string
        audioFormat: string
        audioSize: number
        duration: number
        sampleRate: number
        channels: number
        callType: string
        callStatus: string
        startTime: string
        endTime: string
        userInput: string
        aiResponse: string
        summary: string
        keywords: string
        audioQuality: number
        noiseLevel: number
        tags: string
        category: string
        isImportant: boolean
        isArchived: boolean
        // AI分析相关字段
        aiAnalysis?: string
        analysisStatus?: 'pending' | 'analyzing' | 'completed' | 'failed'
        analysisError?: string
        analyzedAt?: string
        autoAnalyzed?: boolean
        analysisVersion?: number
        createdAt: string
    }>
    total: number
    page: number
    pageSize: number
}>> => {
    const queryParams = new URLSearchParams()
    if (params.page) queryParams.append('page', params.page.toString())
    if (params.pageSize) queryParams.append('page_size', params.pageSize.toString())
    if (params.assistantId) queryParams.append('assistant_id', params.assistantId)
    if (params.macAddress) queryParams.append('mac_address', params.macAddress)

    return get(`/device/call-recordings?${queryParams.toString()}`)
}

// 分析通话录音
export const analyzeCallRecording = async (recordingId: number, force = false): Promise<ApiResponse<null>> => {
    return post(`/device/call-recordings/${recordingId}/analyze${force ? '?force=true' : ''}`, {})
}

// 批量分析通话录音
export const batchAnalyzeCallRecordings = async (data: { assistantId?: number; limit?: number }): Promise<ApiResponse<null>> => {
    return post('/device/call-recordings/batch-analyze', data)
}

// 获取通话录音分析结果
export const getCallRecordingAnalysis = async (recordingId: number): Promise<ApiResponse<{
    recordingId: number
    analysisStatus: 'pending' | 'analyzing' | 'completed' | 'failed'
    analysisError?: string
    analyzedAt?: string
    autoAnalyzed: boolean
    analysisVersion: number
    analysis?: {
        summary: string
        keywords: string[]
        tags: string[]
        category: string
        isImportant: boolean
        sentimentScore: number
        satisfactionScore: number
        actionItems: string[]
        issues: string[]
        insights: string
    }
}>> => {
    return get(`/device/call-recordings/${recordingId}/analysis`)
}

// 获取通话录音详情（包含详细指标）
export const getCallRecordingDetail = async (recordingId: number): Promise<ApiResponse<{
    id: number
    userId: number
    assistantId: number
    deviceId?: number
    macAddress: string
    sessionId: string
    audioPath: string
    storageUrl?: string
    audioFormat: string
    audioSize: number
    duration: number
    sampleRate: number
    channels: number
    callType: string
    callStatus: string
    startTime: string
    endTime: string
    userInput: string
    aiResponse: string
    summary: string
    keywords: string
    audioQuality: number
    noiseLevel: number
    tags: string
    category: string
    isImportant: boolean
    isArchived: boolean
    // 详细对话指标
    conversationDetailsData?: {
        sessionId: string
        startTime: string
        endTime: string
        totalTurns: number
        userTurns: number
        aiTurns: number
        interruptions: number
        turns: Array<{
            turnId: number
            timestamp: string
            type: 'user' | 'ai'
            content: string
            startTime: string
            endTime: string
            duration: number
            // 用户输入特有字段
            asrStartTime?: string
            asrEndTime?: string
            asrDuration?: number
            // AI回复特有字段
            llmStartTime?: string
            llmEndTime?: string
            llmDuration?: number
            ttsStartTime?: string
            ttsEndTime?: string
            ttsDuration?: number
            // 延迟指标
            responseDelay?: number
            totalDelay?: number
        }>
    }
    // 时间指标
    timingMetricsData?: {
        sessionDuration: number
        // ASR指标
        asrCalls: number
        asrTotalTime: number
        asrAverageTime: number
        asrMinTime: number
        asrMaxTime: number
        // LLM指标
        llmCalls: number
        llmTotalTime: number
        llmAverageTime: number
        llmMinTime: number
        llmMaxTime: number
        // TTS指标
        ttsCalls: number
        ttsTotalTime: number
        ttsAverageTime: number
        ttsMinTime: number
        ttsMaxTime: number
        // 响应延迟指标
        responseDelays: number[]
        averageResponseDelay: number
        minResponseDelay: number
        maxResponseDelay: number
        // 总延迟指标
        totalDelays: number[]
        averageTotalDelay: number
        minTotalDelay: number
        maxTotalDelay: number
    }
    // AI分析相关字段
    aiAnalysis?: string
    analysisStatus?: 'pending' | 'analyzing' | 'completed' | 'failed'
    analysisError?: string
    analyzedAt?: string
    autoAnalyzed?: boolean
    analysisVersion?: number
    createdAt: string
}>> => {
    return get(`/device/call-recordings/${recordingId}`)
}