/**
 * LingEcho JavaScript SDK
 * 为JS模板开发者提供封装的API接口
 * 
 * 版本: 1.0.0
 * 
 * 使用示例:
 * const sdk = new LingEchoSDK({
 *   baseURL: '{{.BaseURL}}',
 *   apiKey: 'your-api-key',
 *   apiSecret: 'your-api-secret'
 * });
 * 
 * // 检测SDK是否已加载
 * if (LingEchoSDK.isReady()) {
 *   const sdk = window.lingEcho;
 * }
 * 
 * // 连接语音通话
 * await sdk.connectVoice({
 *   assistantId: 1,
 *   onMessage: (message) => console.log(message),
 *   onError: (error) => console.error(error)
 * });
 */

(function(global) {
    'use strict';

    /**
     * LingEcho SDK 主类
     */
    class LingEchoSDK {
        constructor(config = {}) {
            // 从模板变量或配置中获取baseURL（支持多种格式）
            let serverBase = '';
            if (config.baseURL) {
                serverBase = config.baseURL;
            } else if (typeof SERVER_BASE !== 'undefined') {
                serverBase = SERVER_BASE;
            } else if (typeof window !== 'undefined' && window.SERVER_BASE) {
                serverBase = window.SERVER_BASE;
            }
            
            let assistantName = '';
            if (config.assistantName) {
                assistantName = config.assistantName;
            } else if (typeof ASSISTANT_NAME !== 'undefined') {
                assistantName = ASSISTANT_NAME;
            } else if (typeof window !== 'undefined' && window.ASSISTANT_NAME) {
                assistantName = window.ASSISTANT_NAME;
            }
            
            this.baseURL = serverBase;
            this.apiKey = config.apiKey || '';
            this.apiSecret = config.apiSecret || '';
            this.assistantId = config.assistantId || null;
            this.assistantName = assistantName;
            
            // WebSocket连接状态
            this.wsConnection = null;
            this.peerConnection = null;
            this.localStream = null;
            this.isCalling = false;
            
            // 事件监听器
            this.eventListeners = {};
        }

        /**
         * 工具函数：构建完整的API URL
         */
        buildURL(path) {
            if (path.startsWith('http://') || path.startsWith('https://')) {
                return path;
            }
            const base = this.baseURL.replace(/\/$/, '');
            const cleanPath = path.startsWith('/') ? path : '/' + path;
            return base + cleanPath;
        }

        /**
         * 工具函数：构建WebSocket URL
         */
        buildWebSocketURL(path) {
            let wsUrl;
            if (window.location.protocol === 'file:') {
                // 如果是从本地文件运行，使用默认的WebSocket服务器地址
                wsUrl = 'ws://localhost:7072' + path;
            } else {
                // 正常情况下的WebSocket URL构建
                wsUrl = this.buildURL(path).replace(/^http/, 'ws');
            }
            return wsUrl;
        }

        /**
         * HTTP请求封装
         */
        async request(method, path, data = null, options = {}) {
            const url = this.buildURL(path);
            const headers = {
                'Content-Type': 'application/json',
                ...options.headers
            };

            // 添加认证信息
            if (this.apiKey && this.apiSecret) {
                headers['X-API-Key'] = this.apiKey;
                headers['X-API-Secret'] = this.apiSecret;
            }

            const config = {
                method: method.toUpperCase(),
                headers: headers,
                ...options
            };

            if (data && (method.toUpperCase() === 'POST' || method.toUpperCase() === 'PUT' || method.toUpperCase() === 'PATCH')) {
                config.body = JSON.stringify(data);
            }

            try {
                const response = await fetch(url, config);
                const result = await response.json();
                
                if (!response.ok) {
                    throw new Error(result.msg || result.message || `HTTP ${response.status}`);
                }
                
                return result;
            } catch (error) {
                console.error(`[LingEchoSDK] Request failed: ${method} ${path}`, error);
                throw error;
            }
        }

        /**
         * GET请求
         */
        async get(path, options = {}) {
            return this.request('GET', path, null, options);
        }

        /**
         * POST请求
         */
        async post(path, data, options = {}) {
            return this.request('POST', path, data, options);
        }

        /**
         * PUT请求
         */
        async put(path, data, options = {}) {
            return this.request('PUT', path, data, options);
        }

        /**
         * DELETE请求
         */
        async delete(path, options = {}) {
            return this.request('DELETE', path, null, options);
        }

        /**
         * 事件系统
         */
        on(event, callback) {
            if (!this.eventListeners[event]) {
                this.eventListeners[event] = [];
            }
            this.eventListeners[event].push(callback);
        }

        off(event, callback) {
            if (!this.eventListeners[event]) return;
            this.eventListeners[event] = this.eventListeners[event].filter(cb => cb !== callback);
        }

        emit(event, data) {
            if (!this.eventListeners[event]) return;
            this.eventListeners[event].forEach(callback => {
                try {
                    callback(data);
                } catch (error) {
                    console.error(`[LingEchoSDK] Error in event listener for ${event}:`, error);
                }
            });
        }

        /**
         * ========== 助手管理 API ==========
         */

        /**
         * 获取助手列表
         */
        async getAssistants() {
            return this.get('/api/assistant');
        }

        /**
         * 获取单个助手信息
         */
        async getAssistant(id) {
            return this.get(`/api/assistant/${id}`);
        }

        /**
         * 创建助手
         */
        async createAssistant(data) {
            return this.post('/api/assistant/add', data);
        }

        /**
         * 更新助手
         */
        async updateAssistant(id, data) {
            return this.put(`/api/assistant/${id}`, data);
        }

        /**
         * 删除助手
         */
        async deleteAssistant(id) {
            return this.delete(`/api/assistant/${id}`);
        }

        /**
         * 获取助手工具列表
         */
        async getAssistantTools(assistantId) {
            return this.get(`/api/assistant/${assistantId}/tools`);
        }

        /**
         * 创建助手工具
         */
        async createAssistantTool(assistantId, toolData) {
            return this.post(`/api/assistant/${assistantId}/tools`, toolData);
        }

        /**
         * ========== 语音通话功能 ==========
         */

        /**
         * 连接语音通话（WebRTC + WebSocket）
         */
        async connectVoice(options = {}) {
            const {
                assistantId = this.assistantId,
                apiKey = this.apiKey,
                apiSecret = this.apiSecret,
                onMessage = null,
                onError = null,
                onOpen = null,
                onClose = null,
                onTrack = null
            } = options;

            if (!assistantId) {
                throw new Error('assistantId is required');
            }

            try {
                // 1. 构建WebSocket URL
                const wsUrl = `${this.buildWebSocketURL('/api/chat/call')}?apiKey=${encodeURIComponent(apiKey)}&apiSecret=${encodeURIComponent(apiSecret)}`;
                
                // 2. 创建WebSocket连接
                const ws = new WebSocket(wsUrl);
                this.wsConnection = ws;

                // 3. 创建RTCPeerConnection
                const pc = new RTCPeerConnection({
                    iceServers: [
                        { urls: 'stun:stun.l.google.com:19302' }
                    ]
                });
                this.peerConnection = pc;

                // 4. 获取麦克风音频流
                const stream = await navigator.mediaDevices.getUserMedia({
                    audio: {
                        echoCancellation: true,
                        noiseSuppression: true,
                        autoGainControl: true
                    }
                });
                this.localStream = stream;

                // 添加音频轨道到PeerConnection
                stream.getTracks().forEach(track => {
                    pc.addTrack(track, stream);
                });

                // 5. 处理ICE候选
                pc.onicecandidate = (event) => {
                    if (event.candidate && ws.readyState === WebSocket.OPEN) {
                        ws.send(JSON.stringify({
                            type: 'ice-candidate',
                            candidate: event.candidate
                        }));
                    }
                };

                // 6. 处理远端音频流
                pc.ontrack = (event) => {
                    if (onTrack) {
                        onTrack(event.streams[0]);
                    } else {
                        // 默认播放远端音频
                        const audio = new Audio();
                        audio.srcObject = event.streams[0];
                        audio.play().catch(err => {
                            console.error('[LingEchoSDK] Failed to play remote audio:', err);
                        });
                    }
                    this.emit('track', event.streams[0]);
                };

                // 7. WebSocket事件处理
                ws.onopen = () => {
                    console.log('[LingEchoSDK] WebSocket connected');
                    this.isCalling = true;
                    if (onOpen) onOpen();
                    this.emit('open');
                };

                ws.onmessage = async (event) => {
                    try {
                        const message = JSON.parse(event.data);
                        
                        if (message.type === 'offer') {
                            // 收到offer，创建answer
                            await pc.setRemoteDescription(new RTCSessionDescription(message));
                            const answer = await pc.createAnswer();
                            await pc.setLocalDescription(answer);
                            ws.send(JSON.stringify({
                                type: 'answer',
                                sdp: answer.sdp
                            }));
                        } else if (message.type === 'ice-candidate') {
                            // 添加ICE候选
                            await pc.addIceCandidate(new RTCIceCandidate(message.candidate));
                        } else if (message.type === 'error') {
                            const error = new Error(message.message || 'Unknown error');
                            if (onError) onError(error);
                            this.emit('error', error);
                        } else {
                            // 其他消息
                            if (onMessage) onMessage(message);
                            this.emit('message', message);
                        }
                    } catch (error) {
                        console.error('[LingEchoSDK] Error processing WebSocket message:', error);
                        if (onError) onError(error);
                        this.emit('error', error);
                    }
                };

                ws.onerror = (error) => {
                    console.error('[LingEchoSDK] WebSocket error:', error);
                    if (onError) onError(error);
                    this.emit('error', error);
                };

                ws.onclose = () => {
                    console.log('[LingEchoSDK] WebSocket closed');
                    this.isCalling = false;
                    this.cleanup();
                    if (onClose) onClose();
                    this.emit('close');
                };

                // 8. 连接状态监控
                pc.onconnectionstatechange = () => {
                    console.log('[LingEchoSDK] Connection state:', pc.connectionState);
                    this.emit('connectionstatechange', pc.connectionState);
                    
                    if (pc.connectionState === 'failed' || pc.connectionState === 'disconnected') {
                        this.cleanup();
                    }
                };

                return { ws, pc, stream };

            } catch (error) {
                console.error('[LingEchoSDK] Failed to connect voice:', error);
                this.cleanup();
                throw error;
            }
        }

        /**
         * 断开语音通话
         */
        disconnectVoice() {
            this.cleanup();
        }

        /**
         * 清理资源
         */
        cleanup() {
            this.isCalling = false;

            // 关闭WebSocket
            if (this.wsConnection) {
                this.wsConnection.close();
                this.wsConnection = null;
            }

            // 关闭PeerConnection
            if (this.peerConnection) {
                this.peerConnection.close();
                this.peerConnection = null;
            }

            // 停止本地音频流
            if (this.localStream) {
                this.localStream.getTracks().forEach(track => track.stop());
                this.localStream = null;
            }
        }

        /**
         * ========== 聊天会话 API ==========
         */

        /**
         * 获取聊天会话日志
         */
        async getChatSessionLogs(params = {}) {
            const queryString = new URLSearchParams(params).toString();
            const path = `/api/chat/chat-session-log${queryString ? '?' + queryString : ''}`;
            return this.get(path);
        }

        /**
         * 获取指定会话的日志
         */
        async getChatLogsBySession(sessionId) {
            return this.get(`/api/chat/chat-session-log/by-session/${sessionId}`);
        }

        /**
         * 获取指定助手的日志
         */
        async getChatLogsByAssistant(assistantId) {
            return this.get(`/api/chat/chat-session-log/by-assistant/${assistantId}`);
        }

        /**
         * ========== JS模板 API ==========
         */

        /**
         * 获取JS模板列表
         */
        async getJSTemplates(params = {}) {
            const queryString = new URLSearchParams(params).toString();
            const path = `/api/js-templates${queryString ? '?' + queryString : ''}`;
            return this.get(path);
        }

        /**
         * 获取单个JS模板
         */
        async getJSTemplate(id) {
            return this.get(`/api/js-templates/${id}`);
        }

        /**
         * 创建JS模板
         */
        async createJSTemplate(data) {
            return this.post('/api/js-templates', data);
        }

        /**
         * 更新JS模板
         */
        async updateJSTemplate(id, data) {
            return this.put(`/api/js-templates/${id}`, data);
        }

        /**
         * 删除JS模板
         */
        async deleteJSTemplate(id) {
            return this.delete(`/api/js-templates/${id}`);
        }

        /**
         * ========== 工具函数 ==========
         */

        /**
         * 格式化时间
         */
        formatTime(date = new Date()) {
            return date.toLocaleTimeString('zh-CN', {
                hour: '2-digit',
                minute: '2-digit',
                second: '2-digit'
            });
        }

        /**
         * 转义HTML
         */
        escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        /**
         * 显示通知（需要页面有通知系统）
         */
        showNotification(message, type = 'info') {
            // 尝试使用常见的通知库
            if (typeof showAlert === 'function') {
                showAlert(message, type);
            } else if (typeof toast === 'function') {
                toast(message, type);
            } else {
                console.log(`[${type.toUpperCase()}] ${message}`);
            }
        }
    }

    // 导出到全局
    if (typeof module !== 'undefined' && module.exports) {
        module.exports = LingEchoSDK;
    } else {
        global.LingEchoSDK = LingEchoSDK;
    }

    /**
     * ========== SDK 检测和工具函数 ==========
     */

    /**
     * 检测LingEcho SDK是否已加载
     * @returns {boolean} 是否已加载
     */
    function isSDKLoaded() {
        return typeof LingEchoSDK !== 'undefined';
    }

    /**
     * 检测全局实例是否已创建
     * @returns {boolean} 是否已创建
     */
    function isSDKInstanceReady() {
        return typeof window !== 'undefined' && 
               typeof window.lingEcho !== 'undefined' && 
               window.lingEcho instanceof LingEchoSDK;
    }

    /**
     * 等待SDK加载完成
     * @param {number} timeout - 超时时间（毫秒），默认10秒
     * @returns {Promise<LingEchoSDK>} SDK实例
     */
    function waitForSDK(timeout = 10000) {
        return new Promise((resolve, reject) => {
            // 如果已经加载，直接返回
            if (isSDKInstanceReady()) {
                resolve(window.lingEcho);
                return;
            }

            // 设置超时
            const timeoutId = setTimeout(() => {
                reject(new Error('LingEcho SDK加载超时'));
            }, timeout);

            // 轮询检查
            const checkInterval = setInterval(() => {
                if (isSDKInstanceReady()) {
                    clearTimeout(timeoutId);
                    clearInterval(checkInterval);
                    resolve(window.lingEcho);
                }
            }, 100);
        });
    }

    /**
     * 获取SDK版本信息
     * @returns {string} 版本号
     */
    LingEchoSDK.version = '1.0.0';

    /**
     * 静态方法：检测SDK是否已加载
     */
    LingEchoSDK.isLoaded = isSDKLoaded;

    /**
     * 静态方法：检测SDK实例是否就绪
     */
    LingEchoSDK.isReady = isSDKInstanceReady;

    /**
     * 静态方法：等待SDK加载
     */
    LingEchoSDK.waitFor = waitForSDK;

    // 导出到全局
    if (typeof module !== 'undefined' && module.exports) {
        module.exports = LingEchoSDK;
    } else {
        global.LingEchoSDK = LingEchoSDK;
        
        // 导出检测函数到全局
        global.isLingEchoSDKLoaded = isSDKLoaded;
        global.isLingEchoSDKReady = isSDKInstanceReady;
        global.waitForLingEchoSDK = waitForSDK;
    }

    // 如果是在模板环境中，自动创建默认实例
    // 支持多种模板变量格式
    let serverBase = null;
    let assistantName = null;
    
    if (typeof SERVER_BASE !== 'undefined') {
        serverBase = SERVER_BASE;
    } else if (typeof window !== 'undefined' && window.SERVER_BASE) {
        serverBase = window.SERVER_BASE;
    }
    
    if (typeof ASSISTANT_NAME !== 'undefined') {
        assistantName = ASSISTANT_NAME;
    } else if (typeof window !== 'undefined' && window.ASSISTANT_NAME) {
        assistantName = window.ASSISTANT_NAME;
    }
    
    if (serverBase) {
        global.lingEcho = new LingEchoSDK({
            baseURL: serverBase,
            assistantName: assistantName || ''
        });
        
        // 标记SDK已加载
        if (typeof window !== 'undefined') {
            window.__LINGECHO_SDK_LOADED__ = true;
            window.__LINGECHO_SDK_VERSION__ = LingEchoSDK.version;
        }
    }

})(typeof window !== 'undefined' ? window : this);

