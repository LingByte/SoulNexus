(function() {
    const SERVER_BASE = "{{.BaseURL}}";
    let isDragging = false;
    let dragOffsetX = 0;
    let dragOffsetY = 0;
    let messageHistory = [];
    let isTyping = false;
    let currentTheme = 'light';
    
    // 语音助手相关状态
    let isCalling = false;
    let socket = null;
    let peerConnection = null;
    let localStream = null;
    let callDuration = 0;
    let callTimer = null;
    let pendingCandidates = [];
    let selectedAssistant = 1;
    let chatMessages = [];
    
    // 音频分析相关
    let audioContext = null;
    let analyser = null;
    let microphone = null;
    let dataArray = null;
    let animationId = null;
    let volumeLevel = 0;
    
    // 控制面板状态
    let apiKey = '';
    let apiSecret = '';
    let language = 'zh-cn';
    let selectedSpeaker = '101016';
    let systemPrompt = '';
    let instruction = '你是一个专业的语音助手，请用简洁的语言回答问题';
    let temperature = 0.6;
    let maxTokens = 150;
    let speed = 1.0;
    let volume = 5;
    
    // UI状态
    let isControlPanelCollapsed = true;
    let showConfirmModal = false;
    let pendingAgent = null;

    // 工具函数
    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    function formatTime() {
        return new Date().toLocaleTimeString('zh-CN', { 
            hour: '2-digit', 
            minute: '2-digit' 
        });
    }

    function addMessage(content, type = 'assistant', timestamp = null) {
        const message = {
            content: escapeHtml(content),
            type,
            timestamp: timestamp || formatTime(),
            id: Date.now() + Math.random(),
            actions: type === 'user' ? ['copy', 'resend'] : ['copy', 'like', 'dislike']
        };
        messageHistory.push(message);
        return message;
    }

    function showTypingIndicator(container) {
        if (isTyping) return;
        isTyping = true;
        
        const typingEl = document.createElement("div");
        typingEl.id = "typingIndicator";
        typingEl.className = "message-enter flex justify-start";
        
        const typingContent = document.createElement("div");
        typingContent.className = "max-w-[80%] px-4 py-2 rounded-2xl bg-gray-100 text-gray-800";
        typingContent.innerHTML = `
            <div class="flex items-center space-x-2">
                <div class="typing-indicator">
                    <div class="typing-dot"></div>
                    <div class="typing-dot"></div>
                    <div class="typing-dot"></div>
                </div>
                <span class="text-sm text-gray-500">AI正在思考...</span>
            </div>
        `;
        
        typingEl.appendChild(typingContent);
        container.appendChild(typingEl);
        container.scrollTop = container.scrollHeight;
    }

    function hideTypingIndicator(container) {
        const typingEl = document.getElementById('typingIndicator');
        if (typingEl) {
            typingEl.remove();
        }
        isTyping = false;
    }

    function copyToClipboard(text) {
        if (navigator.clipboard) {
            navigator.clipboard.writeText(text).then(() => {
                showNotification('已复制到剪贴板', 'success');
            });
        } else {
            // 降级处理
            const textArea = document.createElement('textarea');
            textArea.value = text;
            document.body.appendChild(textArea);
            textArea.select();
            document.execCommand('copy');
            document.body.removeChild(textArea);
            showNotification('已复制到剪贴板', 'success');
        }
    }

    function showNotification(message, type = 'info') {
        const notification = document.createElement('div');
        notification.className = `fixed top-4 right-4 z-[10000] px-4 py-3 rounded-2xl text-white text-sm font-medium notification-enter backdrop-blur-sm border ${
            type === 'success' ? 'bg-emerald-500/90 border-emerald-400/50' : 
            type === 'error' ? 'bg-red-500/90 border-red-400/50' : 
            'bg-blue-500/90 border-blue-400/50'
        }`;
        notification.textContent = message;
        
        document.body.appendChild(notification);
        
        setTimeout(() => {
            notification.style.transform = 'translateX(100%)';
            setTimeout(() => {
                if (notification.parentNode) {
                    notification.parentNode.removeChild(notification);
                }
            }, 300);
        }, 3000);
    }

    function toggleTheme() {
        currentTheme = currentTheme === 'light' ? 'dark' : 'light';
        localStorage.setItem('aiChatTheme', currentTheme);
        applyTheme();
    }

    function applyTheme() {
        const panel = document.querySelector('.ai-chat-container');
        if (!panel) return;
        
        if (currentTheme === 'dark') {
            panel.classList.add('dark-theme');
        } else {
            panel.classList.remove('dark-theme');
        }
    }

    function loadAxios(callback) {
        if (window.axios) {
            callback();
            return;
        }
        const script = document.createElement("script");
        script.src = "https://cdn.jsdelivr.net/npm/axios@1.6.0/dist/axios.min.js";
        script.onload = callback;
        script.onerror = () => {
            console.warn('Axios CDN加载失败，使用备用CDN');
            const fallbackScript = document.createElement("script");
            fallbackScript.src = "https://cdnjs.cloudflare.com/ajax/libs/axios/1.6.0/axios.min.js";
            fallbackScript.onload = callback;
            fallbackScript.onerror = () => {
                console.error('所有Axios CDN都加载失败，请检查网络连接');
                // 即使Axios加载失败，也继续执行，只是某些功能可能不可用
                callback();
            };
            document.head.appendChild(fallbackScript);
        };
        document.head.appendChild(script);
    }

    // 语音助手相关函数
    function addAIMessage(text) {
        const messageId = `ai-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
        const message = {
            type: 'agent',
            content: text,
            timestamp: new Date().toLocaleTimeString(),
            id: messageId
        };
        chatMessages.push(message);
        return message;
    }

    // 音频分析函数
    function initAudioAnalysis() {
        if (!audioContext) {
            audioContext = new (window.AudioContext || window.webkitAudioContext)();
            analyser = audioContext.createAnalyser();
            analyser.fftSize = 256;
            dataArray = new Uint8Array(analyser.frequencyBinCount);
        }
    }

    function startAudioAnalysis(stream) {
        initAudioAnalysis();
        
        if (microphone) {
            microphone.disconnect();
        }
        
        microphone = audioContext.createMediaStreamSource(stream);
        microphone.connect(analyser);
        
        updateVolumeLevel();
    }

    function stopAudioAnalysis() {
        if (animationId) {
            cancelAnimationFrame(animationId);
            animationId = null;
        }
        
        if (microphone) {
            microphone.disconnect();
            microphone = null;
        }
        
        volumeLevel = 0;
        updateVoiceBallVisual();
    }

    function updateVolumeLevel() {
        if (!analyser || !isCalling) {
            volumeLevel = 0;
            updateVoiceBallVisual();
            return;
        }
        
        analyser.getByteFrequencyData(dataArray);
        
        // 计算平均音量
        let sum = 0;
        for (let i = 0; i < dataArray.length; i++) {
            sum += dataArray[i];
        }
        volumeLevel = sum / dataArray.length;
        
        updateVoiceBallVisual();
        
        if (isCalling) {
            animationId = requestAnimationFrame(updateVolumeLevel);
        }
    }

    function updateVoiceBallVisual() {
        const voiceBall = document.querySelector('.voice-ball');
        if (!voiceBall) return;
        
        if (isCalling && volumeLevel > 0) {
            // 根据音量调整语音球的大小和颜色
            const scale = 1 + (volumeLevel / 255) * 0.3; // 最大放大30%
            const intensity = volumeLevel / 255;
            
            voiceBall.style.transform = `scale(${scale})`;
            voiceBall.style.boxShadow = `0 8px 32px rgba(102, 126, 234, ${0.3 + intensity * 0.4})`;
            
            // 添加脉冲效果
            if (volumeLevel > 50) {
                voiceBall.classList.add('voice-pulse');
            } else {
                voiceBall.classList.remove('voice-pulse');
            }
        } else {
            voiceBall.style.transform = 'scale(1)';
            voiceBall.style.boxShadow = '0 8px 32px rgba(102, 126, 234, 0.3)';
            voiceBall.classList.remove('voice-pulse');
        }
    }

    function connectWebSocket() {
        // 先关闭现有连接
        if (socket && socket.readyState !== WebSocket.CLOSED) {
            socket.close();
        }

        // 检查当前协议并构建WebSocket URL
        let wsUrl;
        if (window.location.protocol === 'file:') {
            // 如果是从本地文件运行，使用默认的WebSocket服务器地址
            wsUrl = 'ws://localhost:7072/api/chat/call';
        } else {
            // 正常情况下的WebSocket URL构建
            wsUrl = `${SERVER_BASE.replace('http', 'ws')}/api/chat/call`;
        }
        
        // 添加认证参数
        wsUrl += `?apiKey=${encodeURIComponent(apiKey)}&apiSecret=${encodeURIComponent(apiSecret)}`;
        
        console.log('[WebSocket] 连接URL:', wsUrl);
        const newSocket = new WebSocket(wsUrl);

        newSocket.onopen = async () => {
            console.log('[WebSocket] 已连接');

            try {
                // 1. 创建 RTCPeerConnection
                const newPeerConnection = new RTCPeerConnection({
                    iceServers: [
                        { urls: 'stun:stun.l.google.com:19302' } // 公共 STUN 服务器
                    ]
                });

                // 2. 获取麦克风音频
                const stream = await navigator.mediaDevices.getUserMedia({
                    audio: {
                        echoCancellation: true,
                    }
                });

                stream.getTracks().forEach(track => {
                    newPeerConnection.addTrack(track, stream);
                });

                // 启动音频分析
                startAudioAnalysis(stream);

                // 3. 收集 ICE 候选，并发送给后端
                newPeerConnection.onicecandidate = (event) => {
                    if (event.candidate && newSocket.readyState === WebSocket.OPEN) {
                        newSocket.send(JSON.stringify({
                            type: 'ice-candidate',
                            candidate: event.candidate
                        }));
                    } else if (event.candidate) {
                        console.warn('[WebRTC] WebSocket连接已关闭，无法发送ICE候选');
                    }
                };

                newPeerConnection.ontrack = (event) => {
                    const remoteAudio = new Audio();
                    remoteAudio.srcObject = event.streams[0];
                    remoteAudio.play().catch(err => {
                        console.error('[WebRTC] 播放远端音频失败:', err);
                    });
                };

                newPeerConnection.onconnectionstatechange = () => {
                    switch (newPeerConnection.connectionState) {
                        case 'connected':
                            console.log('[WebRTC] 已连接');
                            break;
                        case 'disconnected':
                        case 'failed':
                        case 'closed':
                            console.log('[WebRTC] 连接关闭/失败');
                            break;
                    }
                };

                // 4. 创建 offer
                const offer = await newPeerConnection.createOffer();
                await newPeerConnection.setLocalDescription(offer);

                // 5. 发送 offer to websocket
                if (newSocket.readyState === WebSocket.OPEN) {
                    newSocket.send(JSON.stringify({
                        type: 'offer',
                        sdp: offer.sdp,
                        assistantId: selectedAssistant || 1,
                        instruction: instruction,
                        language: language,
                        maxTokens: maxTokens,
                        personaTag: "语音助手",
                        speaker: selectedSpeaker,
                        speed: speed,
                        systemPrompt: systemPrompt,
                        temperature: temperature,
                        volume: volume,
                    }));
                } else {
                    console.error('[WebSocket] 连接未就绪，无法发送offer');
                }

                peerConnection = newPeerConnection;
                localStream = stream;

                // 设置WebSocket消息处理，确保peerConnection已经创建
                newSocket.onmessage = async (event) => {
                    console.log('[WebSocket] 收到消息:', event.data);
                    const data = JSON.parse(event.data);
                    console.log('[WebSocket] 解析后的数据:', data);

                    switch (data.type) {
                        case 'answer':
                            console.log('[WebRTC] 收到answer消息，检查条件:');
                            console.log('- peerConnection存在:', !!newPeerConnection);
                            console.log('- data.sdp存在:', !!data.sdp);
                            console.log('- data.sdp内容:', data.sdp);

                            if (newPeerConnection && data.sdp) {
                                const remoteDesc = new RTCSessionDescription({
                                    type: 'answer',
                                    sdp: data.sdp,
                                });
                                console.log('[WebRTC] 设置远端 SDP answer', remoteDesc);
                                await newPeerConnection.setRemoteDescription(remoteDesc);
                                console.log('[WebRTC] 已设置远端 SDP answer');

                                // 设置完 remoteDescription 后再处理缓存的 ICE 候选
                                for (const candidate of pendingCandidates) {
                                    try {
                                        await newPeerConnection.addIceCandidate(new RTCIceCandidate(candidate));
                                        console.log('[WebRTC] 添加缓存 ICE 候选成功');
                                    } catch (err) {
                                        console.error('[WebRTC] 添加缓存 ICE 候选失败:', err);
                                    }
                                }
                                pendingCandidates = [];
                            } else {
                                console.error('[WebRTC] 条件不满足，无法设置远端SDP');
                                console.error('- peerConnection:', newPeerConnection);
                                console.error('- data.sdp:', data.sdp);
                            }
                            break;
                        case 'asrFinal':
                            console.log('[WebSocket] 收到ASR结果:', data.text);
                            addAIMessage(data.text);
                            updateChatDisplay();
                            break;
                        case 'ice-candidate':
                            if (newPeerConnection) {
                                const candidate = new RTCIceCandidate(data.candidate);
                                if (newPeerConnection.remoteDescription && newPeerConnection.remoteDescription.type) {
                                    try {
                                        await newPeerConnection.addIceCandidate(candidate);
                                        console.log('[WebRTC] 添加 ICE 候选成功');
                                    } catch (err) {
                                        console.error('[WebRTC] 添加 ICE 候选失败:', err);
                                    }
                                } else {
                                    pendingCandidates.push(data.candidate);
                                    console.log('[WebRTC] 缓存 ICE 候选，等待 remoteDescription 设置');
                                }
                            }
                            break;
                    }
                };

            } catch (error) {
                console.error('[WebRTC] 初始化失败:', error);
                showNotification('音频设备初始化失败', 'error');
            }
        };

        newSocket.onerror = (error) => {
            console.error('[WebSocket] 连接出错:', error);
            showNotification('WebSocket连接出错', 'error');
        };

        newSocket.onclose = () => {
            console.log('[WebSocket] 连接关闭');
        };

        socket = newSocket;
    }

    function startCall() {
        if (selectedAssistant === 0) {
            showNotification('请先选择一个AI助手', 'warning');
            return;
        }

        try {
            isCalling = true;
            callDuration = 0;
            chatMessages = []; // 清空当前聊天记录
            
            // 连接WebSocket
            connectWebSocket();
            
            // 开始通话计时器
            const timer = setInterval(() => {
                callDuration++;
                updateCallDuration();
            }, 1000);
            callTimer = timer;
            
            showNotification('通话已开始', 'success');
            updateCallStatus();
        } catch (err) {
            console.error('通话启动失败:', err);
            isCalling = false;
            showNotification('通话启动失败', 'error');
        }
    }

    function stopCall() {
        try {
            console.log('[StopCall] 开始停止通话');
            
            // 停止通话计时器
            if (callTimer) {
                clearInterval(callTimer);
                callTimer = null;
            }
            
            // 关闭WebSocket连接
            if (socket && socket.readyState !== WebSocket.CLOSED) {
                socket.close();
                socket = null;
            }
            
            // 关闭WebRTC连接
            if (peerConnection && peerConnection.connectionState !== 'closed') {
                peerConnection.close();
                peerConnection = null;
            }
            
            // 停止本地音频流
            if (localStream) {
                localStream.getTracks().forEach(track => track.stop());
                localStream = null;
            }
            
            // 停止音频分析
            stopAudioAnalysis();
            
            isCalling = false;
            callDuration = 0;
            pendingCandidates = [];
            
            console.log('[StopCall] 通话已结束');
            showNotification('通话已结束', 'success');
            updateCallStatus();
            return true;
        } catch (err) {
            console.error('终止通话失败:', err);
            showNotification('终止通话失败', 'error');
            return false;
        }
    }

    function updateCallStatus() {
        const statusElement = document.getElementById('callStatus');
        if (statusElement) {
            statusElement.textContent = isCalling ? '通话中...' : '待机中';
            statusElement.className = isCalling ? 'text-purple-600' : 'text-blue-500';
        }
        
        // 更新语音球状态
        const voiceBall = document.querySelector('.voice-ball');
        const statusIndicator = document.querySelector('.status-indicator');
        if (voiceBall) {
            if (isCalling) {
                voiceBall.classList.add('calling');
            } else {
                voiceBall.classList.remove('calling');
            }
        }
        if (statusIndicator) {
            if (isCalling) {
                statusIndicator.classList.add('active');
            } else {
                statusIndicator.classList.remove('active');
            }
        }
    }

    function updateCallDuration() {
        const durationElement = document.getElementById('callDuration');
        if (durationElement) {
            if (isCalling) {
                const minutes = Math.floor(callDuration / 60).toString().padStart(2, '0');
                const seconds = (callDuration % 60).toString().padStart(2, '0');
                durationElement.textContent = `${minutes}:${seconds}`;
                durationElement.style.display = 'block';
            } else {
                durationElement.style.display = 'none';
            }
        }
    }

    function addMessageToUI(container, message) {
        const messageEl = document.createElement("div");
        messageEl.className = `message-enter flex ${message.type === 'user' ? 'justify-end' : 'justify-start'} message-container`;
        
        const messageContent = document.createElement("div");
        messageContent.className = `max-w-[85%] px-4 py-3 rounded-2xl ${
            message.type === 'user' 
                ? 'user-message' 
                : 'ai-message'
        }`;
        
        messageContent.innerHTML = `
            <div class="text-sm leading-relaxed">${message.content}</div>
            <div class="text-xs mt-2 opacity-60">${message.timestamp}</div>
        `;
        
        // 添加消息操作按钮
        if (message.type !== 'system') {
            const actionsEl = document.createElement("div");
            actionsEl.className = "message-actions absolute -top-2 right-2 flex space-x-1 bg-white/90 backdrop-blur-sm rounded-xl shadow-lg p-1 border border-gray-200/50";
            
            if (message.actions.includes('copy')) {
                const copyBtn = document.createElement("button");
                copyBtn.innerHTML = `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" class="text-gray-500 hover:text-gray-700"><path d="M8 5H6C4.9 5 4 5.9 4 7V19C4 20.1 4.9 21 6 21H16C17.1 21 18 20.1 18 19V7C18 5.9 17.1 5 16 5H14M8 5C8 6.1 8.9 7 10 7H14C15.1 7 16 6.1 16 5M8 5C8 3.9 8.9 3 10 3H14C15.1 3 16 3.9 16 5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>`;
                copyBtn.title = "复制";
                copyBtn.className = "w-6 h-6 rounded-lg hover:bg-gray-100 flex items-center justify-center transition-colors";
                copyBtn.onclick = () => copyToClipboard(message.content);
                actionsEl.appendChild(copyBtn);
            }
            
            if (message.actions.includes('like')) {
                const likeBtn = document.createElement("button");
                likeBtn.innerHTML = `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" class="text-gray-500 hover:text-emerald-500"><path d="M7 13C7 12.4 7.4 12 8 12H16C16.6 12 17 12.4 17 13V19C17 19.6 16.6 20 16 20H8C7.4 20 7 19.6 7 19V13Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/><path d="M7 13V7C7 6.4 7.4 6 8 6H10C10.6 6 11 6.4 11 7V9C11 9.6 10.6 10 10 10H8C7.4 10 7 9.6 7 9V13Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>`;
                likeBtn.title = "点赞";
                likeBtn.className = "w-6 h-6 rounded-lg hover:bg-gray-100 flex items-center justify-center transition-colors";
                likeBtn.onclick = () => showNotification('已点赞', 'success');
                actionsEl.appendChild(likeBtn);
            }
            
            if (message.actions.includes('dislike')) {
                const dislikeBtn = document.createElement("button");
                dislikeBtn.innerHTML = `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" class="text-gray-500 hover:text-red-500"><path d="M17 11C17 10.4 16.6 10 16 10H8C7.4 10 7 10.4 7 11V17C7 17.6 7.4 18 8 18H16C16.6 18 17 17.6 17 17V11Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/><path d="M17 11V5C17 4.4 16.6 4 16 4H14C13.4 4 13 4.4 13 5V7C13 7.6 13.4 8 14 8H16C16.6 8 17 7.6 17 7V11Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>`;
                dislikeBtn.title = "点踩";
                dislikeBtn.className = "w-6 h-6 rounded-lg hover:bg-gray-100 flex items-center justify-center transition-colors";
                dislikeBtn.onclick = () => showNotification('已点踩', 'info');
                actionsEl.appendChild(dislikeBtn);
            }
            
            if (message.actions.includes('resend')) {
                const resendBtn = document.createElement("button");
                resendBtn.innerHTML = `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" class="text-gray-500 hover:text-blue-500"><path d="M1 4V10H7" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/><path d="M23 20V14H17" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/><path d="M20.49 9A9 9 0 0 0 5.64 5.64L1 10M23 14L18.36 18.36A9 9 0 0 1 3.51 15" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>`;
                resendBtn.title = "重新发送";
                resendBtn.className = "w-6 h-6 rounded-lg hover:bg-gray-100 flex items-center justify-center transition-colors";
                resendBtn.onclick = () => {
                    const input = document.getElementById('messageInput');
                    if (input) {
                        input.value = message.content;
                        sendMessage(container);
                    }
                };
                actionsEl.appendChild(resendBtn);
            }
            
            messageEl.appendChild(actionsEl);
        }
        
        messageEl.appendChild(messageContent);
        container.appendChild(messageEl);
        container.scrollTop = container.scrollHeight;
    }

    function updateChatDisplay() {
        const messagesContainer = document.getElementById('voiceMessages');
        if (messagesContainer) {
            messagesContainer.innerHTML = '';
            chatMessages.forEach(message => {
                addMessageToUI(messagesContainer, message);
            });
        }
    }
    function getAssistantList() {
        return axios.get(`${SERVER_BASE}/api/assistants`, {
            headers: {
                "X-API-KEY": apiKey,
                "X-API-SECRET": apiSecret,
            }
        });
    }

    function getAssistant(assistantId) {
        return axios.get(`${SERVER_BASE}/api/assistants/${assistantId}`, {
            headers: {
                "X-API-KEY": apiKey,
                "X-API-SECRET": apiSecret,
            }
        });
    }

    function createAssistant(assistant) {
        return axios.post(`${SERVER_BASE}/api/assistants`, assistant, {
            headers: {
                "X-API-KEY": apiKey,
                "X-API-SECRET": apiSecret,
            }
        });
    }

    function updateAssistant(assistantId, data) {
        return axios.put(`${SERVER_BASE}/api/assistants/${assistantId}`, data, {
            headers: {
                "X-API-KEY": apiKey,
                "X-API-SECRET": apiSecret,
            }
        });
    }

    function deleteAssistant(assistantId) {
        return axios.delete(`${SERVER_BASE}/api/assistants/${assistantId}`, {
            headers: {
                "X-API-KEY": apiKey,
                "X-API-SECRET": apiSecret,
            }
        });
    }

    function handleSelectAgent(agentId) {
        if (isCalling) {
            pendingAgent = agentId;
            showConfirmModal = true;
            showConfirmDialog();
            return;
        }

        try {
            selectedAssistant = agentId;
            updateAssistantDisplay();
            showNotification('助手切换成功', 'success');
        } catch (err) {
            console.error('获取助手详情失败:', err);
            showNotification('获取助手详情失败', 'error');
        }
    }

    function confirmSwitch() {
        showConfirmModal = false;
        hideConfirmDialog();
        const success = stopCall();
        if (success && pendingAgent) {
            selectedAssistant = pendingAgent;
            pendingAgent = null;
            updateAssistantDisplay();
        }
    }

    function updateAssistantDisplay() {
        const assistantNameElement = document.getElementById('assistantName');
        if (assistantNameElement) {
            assistantNameElement.textContent = '语音助手';
        }
    }

    function updateAssistantList() {
        const assistantListElement = document.getElementById('assistantList');
        if (assistantListElement) {
            assistantListElement.innerHTML = `
                <div class="text-center text-xs text-gray-500 py-2">
                    语音助手已就绪
                </div>
            `;
        }
    }

    function toggleControlPanel() {
        const content = document.getElementById('controlPanelContent');
        const toggleBtn = document.getElementById('toggleControlPanel');
        if (content && toggleBtn) {
            if (content.style.display === 'none') {
                content.style.display = 'block';
                toggleBtn.innerHTML = `
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" class="text-gray-600">
                        <path d="M9 5L15 12L9 19" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                    </svg>
                `;
            } else {
                content.style.display = 'none';
                toggleBtn.innerHTML = `
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" class="text-gray-600">
                        <path d="M15 19L9 12L15 5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                    </svg>
                `;
            }
        }
    }

    function saveSettings() {
        try {
            // 这里应该调用API保存设置
            showNotification('设置保存成功', 'success');
        } catch (err) {
            console.error('保存设置失败:', err);
            showNotification('保存设置失败', 'error');
        }
    }

    function resetSettings() {
        speed = 1.0;
        volume = 5;
        language = 'zh-cn';
        
        document.getElementById('speedSlider').value = speed;
        document.getElementById('volumeSlider').value = volume;
        document.getElementById('languageSelect').value = language;
        
        showNotification('设置已重置', 'info');
    }

    function showSettingsModal() {
        // 显示当前配置信息
        showNotification(`当前配置: API Key已设置, 助手ID: ${selectedAssistant}`, 'info');
    }

    function showAddAssistantModal() {
        // 助手功能已移除
        showNotification('助手功能已简化', 'info');
    }

    function showConfirmDialog() {
        const modal = document.createElement('div');
        modal.className = 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-[10000]';
        modal.innerHTML = `
            <div class="bg-white dark:bg-neutral-800 p-6 rounded-xl max-w-md w-full mx-4">
                <h3 class="text-lg font-semibold mb-4">切换助手确认</h3>
                <p class="text-gray-600 dark:text-gray-300 mb-6">
                    当前正在通话中，切换助手将结束当前通话。确定要切换吗？
                </p>
                <div class="flex justify-end space-x-4">
                    <button id="cancelSwitch" class="px-4 py-2 text-gray-600 hover:bg-gray-100 rounded-lg">取消</button>
                    <button id="confirmSwitch" class="px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700">确定切换</button>
                </div>
            </div>
        `;
        
        document.body.appendChild(modal);
        
        document.getElementById('cancelSwitch').onclick = () => {
            showConfirmModal = false;
            pendingAgent = null;
            document.body.removeChild(modal);
        };
        
        document.getElementById('confirmSwitch').onclick = () => {
            confirmSwitch();
            document.body.removeChild(modal);
        };
    }

    function hideConfirmDialog() {
        const modal = document.querySelector('.fixed.inset-0.bg-black.bg-opacity-50');
        if (modal) {
            document.body.removeChild(modal);
        }
    }

    function loadTailwind(callback) {
        if (document.getElementById("__tailwindcss")) {
            callback();
            return;
        }
        const script = document.createElement("script");
        script.id = "__tailwindcss";
        script.src = "https://cdn.tailwindcss.com";
        script.onload = callback;
        script.onerror = () => {
            console.warn('Tailwind官方CDN加载失败，使用备用CDN');
            // 如果官方CDN失败，使用备用CDN
            const fallbackLink = document.createElement("link");
            fallbackLink.id = "__tailwindcss";
            fallbackLink.rel = "stylesheet";
            fallbackLink.href = "https://cdnjs.cloudflare.com/ajax/libs/tailwindcss/3.3.0/tailwind.min.css";
            fallbackLink.onload = callback;
            fallbackLink.onerror = () => {
                console.error('所有Tailwind CDN都加载失败，使用内联样式');
                // 如果所有CDN都失败，至少确保基本样式可用
                addFallbackStyles();
                callback();
            };
            document.head.appendChild(fallbackLink);
        };
        document.head.appendChild(script);
    }

    function addFallbackStyles() {
        const fallbackStyle = document.createElement('style');
        fallbackStyle.textContent = `
            /* 基本样式回退 */
            .fixed { position: fixed !important; }
            .relative { position: relative !important; }
            .absolute { position: absolute !important; }
            .flex { display: flex !important; }
            .hidden { display: none !important; }
            .block { display: block !important; }
            .inline-block { display: inline-block !important; }
            .w-full { width: 100% !important; }
            .h-full { height: 100% !important; }
            .rounded { border-radius: 0.25rem !important; }
            .rounded-lg { border-radius: 0.5rem !important; }
            .rounded-xl { border-radius: 0.75rem !important; }
            .rounded-2xl { border-radius: 1rem !important; }
            .rounded-full { border-radius: 9999px !important; }
            .p-2 { padding: 0.5rem !important; }
            .p-4 { padding: 1rem !important; }
            .px-4 { padding-left: 1rem !important; padding-right: 1rem !important; }
            .py-2 { padding-top: 0.5rem !important; padding-bottom: 0.5rem !important; }
            .text-sm { font-size: 0.875rem !important; }
            .text-lg { font-size: 1.125rem !important; }
            .font-medium { font-weight: 500 !important; }
            .font-semibold { font-weight: 600 !important; }
            .text-white { color: white !important; }
            .text-gray-500 { color: #6b7280 !important; }
            .text-gray-800 { color: #1f2937 !important; }
            .bg-white { background-color: white !important; }
            .bg-gray-100 { background-color: #f3f4f6 !important; }
            .bg-gray-500 { background-color: #6b7280 !important; }
            .bg-blue-500 { background-color: #3b82f6 !important; }
            .bg-green-500 { background-color: #10b981 !important; }
            .shadow-lg { box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05) !important; }
            .shadow-2xl { box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25) !important; }
            .transition-all { transition: all 0.3s ease !important; }
            .cursor-pointer { cursor: pointer !important; }
            .cursor-move { cursor: move !important; }
            .opacity-50 { opacity: 0.5 !important; }
            .opacity-70 { opacity: 0.7 !important; }
            .hover\\:bg-gray-100:hover { background-color: #f3f4f6 !important; }
            .hover\\:bg-gray-600:hover { background-color: #4b5563 !important; }
            .focus\\:outline-none:focus { outline: none !important; }
            .focus\\:ring-2:focus { box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.5) !important; }
            .justify-center { justify-content: center !important; }
            .justify-between { justify-content: space-between !important; }
            .items-center { align-items: center !important; }
            .space-x-2 > * + * { margin-left: 0.5rem !important; }
            .space-x-3 > * + * { margin-left: 0.75rem !important; }
            .space-y-4 > * + * { margin-top: 1rem !important; }
            .max-w-\\[80\\%\\] { max-width: 80% !important; }
            .max-w-\\[calc\\(100vw-3rem\\)\\] { max-width: calc(100vw - 3rem) !important; }
            .max-h-\\[70vh\\] { max-height: 70vh !important; }
            .overflow-y-auto { overflow-y: auto !important; }
            .border { border: 1px solid #e5e7eb !important; }
            .border-t { border-top: 1px solid #e5e7eb !important; }
            .border-b { border-bottom: 1px solid #e5e7eb !important; }
            .border-gray-300 { border-color: #d1d5db !important; }
            .border-gray-200\\/50 { border-color: rgba(229, 231, 235, 0.5) !important; }
            .z-\\[9999\\] { z-index: 9999 !important; }
            .z-\\[10000\\] { z-index: 10000 !important; }
            .bottom-6 { bottom: 1.5rem !important; }
            .right-6 { right: 1.5rem !important; }
            .bottom-24 { bottom: 6rem !important; }
            .w-16 { width: 4rem !important; }
            .h-16 { height: 4rem !important; }
            .w-12 { width: 3rem !important; }
            .h-12 { height: 3rem !important; }
            .w-8 { width: 2rem !important; }
            .h-8 { height: 2rem !important; }
            .w-4 { width: 1rem !important; }
            .h-4 { height: 1rem !important; }
            .w-2 { width: 0.5rem !important; }
            .h-2 { height: 0.5rem !important; }
            .mt-1 { margin-top: 0.25rem !important; }
            .flex-1 { flex: 1 1 0% !important; }
            .text-center { text-align: center !important; }
            .text-left { text-align: left !important; }
            .text-right { text-align: right !important; }
            .justify-start { justify-content: flex-start !important; }
            .justify-end { justify-content: flex-end !important; }
            .flex-col { flex-direction: column !important; }
            .flex-row { flex-direction: row !important; }
            .gap-2 { gap: 0.5rem !important; }
            .gap-3 { gap: 0.75rem !important; }
            .gap-4 { gap: 1rem !important; }
            .animate-spin { animation: spin 1s linear infinite !important; }
            .animate-pulse { animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite !important; }
            @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
            @keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.5; } }
        `;
        document.head.appendChild(fallbackStyle);
    }

    function main() {
        const config = window.__AIPetConfig || {};
        
        // 从配置中获取API信息
        if (config.apiKey) {
            apiKey = config.apiKey;
        }
        if (config.apiSecret) {
            apiSecret = config.apiSecret;
        }
        if (config.systemPrompt) {
            systemPrompt = config.systemPrompt;
        }
        if (config.temperature !== undefined) {
            temperature = config.temperature;
        }
        if (config.volume !== undefined) {
            volume = config.volume;
        }
        if (config.assistantId) {
            selectedAssistant = config.assistantId;
        }
        
        console.log('[AIPet] 配置已应用:', {
            apiKey: apiKey ? '已设置' : '未设置',
            apiSecret: apiSecret ? '已设置' : '未设置',
            systemPrompt: systemPrompt,
            temperature: temperature,
            volume: volume,
            selectedAssistant: selectedAssistant
        });
        
        // 添加全局样式
        const globalStyles = document.createElement('style');
        globalStyles.textContent = `
            @import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap');
            
            .ai-chat-container {
                font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            }
            
            .glass-effect {
                background: rgba(255, 255, 255, 0.95);
                backdrop-filter: blur(24px);
                border: 1px solid rgba(255, 255, 255, 0.3);
                box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
            }
            
            .floating-animation {
                animation: float 8s ease-in-out infinite;
            }
            
            @keyframes float {
                0%, 100% { transform: translateY(0px) rotate(0deg); }
                50% { transform: translateY(-8px) rotate(1deg); }
            }
            
            .pulse-glow {
                animation: pulseGlow 3s ease-in-out infinite alternate;
            }
            
            @keyframes pulseGlow {
                from { box-shadow: 0 0 0 0 rgba(59, 130, 246, 0.4); }
                to { box-shadow: 0 0 0 8px rgba(59, 130, 246, 0.1); }
            }
            
            .message-enter {
                animation: messageSlideIn 0.4s cubic-bezier(0.16, 1, 0.3, 1);
            }
            
            @keyframes messageSlideIn {
                from { 
                    opacity: 0; 
                    transform: translateY(16px) scale(0.96); 
                }
                to { 
                    opacity: 1; 
                    transform: translateY(0) scale(1); 
                }
            }
            
            .scrollbar-custom::-webkit-scrollbar {
                width: 4px;
            }
            
            .scrollbar-custom::-webkit-scrollbar-track {
                background: transparent;
            }
            
            .scrollbar-custom::-webkit-scrollbar-thumb {
                background: rgba(156, 163, 175, 0.3);
                border-radius: 2px;
            }
            
            .scrollbar-custom::-webkit-scrollbar-thumb:hover {
                background: rgba(156, 163, 175, 0.5);
            }
            
            .dark-theme {
                background: rgba(17, 24, 39, 0.98) !important;
                border-color: rgba(75, 85, 99, 0.2) !important;
            }
            
            .dark-theme .text-gray-800 {
                color: #f9fafb !important;
            }
            
            .dark-theme .text-gray-500 {
                color: #9ca3af !important;
            }
            
            .dark-theme .bg-gray-100 {
                background-color: #374151 !important;
                color: #f9fafb !important;
            }
            
            .dark-theme .border-gray-200\/50 {
                border-color: rgba(75, 85, 99, 0.2) !important;
            }
            
            .dark-theme .bg-gray-50\/50 {
                background-color: rgba(31, 41, 55, 0.3) !important;
            }
            
            .message-actions {
                opacity: 0;
                transition: all 0.2s ease-in-out;
                transform: translateY(-4px);
            }
            
            .message-container:hover .message-actions {
                opacity: 1;
                transform: translateY(0);
            }
            
            .message-container {
                position: relative;
            }
            
            .voice-recording {
                animation: pulse 1.5s infinite;
            }
            
            @keyframes pulse {
                0%, 100% { opacity: 1; transform: scale(1); }
                50% { opacity: 0.7; transform: scale(1.05); }
            }
            
            .search-highlight {
                background: linear-gradient(120deg, #fef3c7 0%, #fde68a 100%);
                padding: 2px 4px;
                border-radius: 4px;
                font-weight: 500;
            }
            
            .dark-theme .search-highlight {
                background: linear-gradient(120deg, #451a03 0%, #78350f 100%);
            }
            
            .gradient-text {
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                -webkit-background-clip: text;
                -webkit-text-fill-color: transparent;
                background-clip: text;
            }
            
            .minimal-button {
                background: rgba(255, 255, 255, 0.1);
                border: 1px solid rgba(255, 255, 255, 0.2);
                backdrop-filter: blur(10px);
                transition: all 0.2s ease;
            }
            
            .minimal-button:hover {
                background: rgba(255, 255, 255, 0.2);
                border-color: rgba(255, 255, 255, 0.3);
                transform: translateY(-1px);
            }
            
            .ai-avatar {
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                box-shadow: 0 4px 20px rgba(102, 126, 234, 0.3);
            }
            
            .user-message {
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                color: white;
                box-shadow: 0 2px 12px rgba(102, 126, 234, 0.2);
            }
            
            .ai-message {
                background: rgba(248, 250, 252, 0.8);
                border: 1px solid rgba(226, 232, 240, 0.8);
                color: #1e293b;
            }
            
            .dark-theme .ai-message {
                background: rgba(30, 41, 59, 0.8);
                border-color: rgba(51, 65, 85, 0.8);
                color: #f1f5f9;
            }
            
            .typing-dot {
                width: 4px;
                height: 4px;
                border-radius: 50%;
                background-color: #667eea;
                margin: 0 1px;
                animation: typing 1.4s infinite ease-in-out;
            }
            
            .typing-dot:nth-child(1) { animation-delay: -0.32s; }
            .typing-dot:nth-child(2) { animation-delay: -0.16s; }
            
            @keyframes typing {
                0%, 80%, 100% { transform: scale(0.8); opacity: 0.5; }
                40% { transform: scale(1); opacity: 1; }
            }
            
            .panel-enter {
                animation: panelSlideIn 0.4s cubic-bezier(0.16, 1, 0.3, 1);
            }
            
            @keyframes panelSlideIn {
                from { 
                    opacity: 0; 
                    transform: translateY(20px) scale(0.95); 
                }
                to { 
                    opacity: 1; 
                    transform: translateY(0) scale(1); 
                }
            }
            
            .notification-enter {
                animation: notificationSlideIn 0.3s cubic-bezier(0.16, 1, 0.3, 1);
            }
            
            @keyframes notificationSlideIn {
                from { 
                    opacity: 0; 
                    transform: translateX(100%) scale(0.9); 
                }
                to { 
                    opacity: 1; 
                    transform: translateX(0) scale(1); 
                }
            }
            
            /* 语音球样式 */
            .voice-ball-container {
                position: relative;
                width: 64px;
                height: 64px;
            }
            
            .voice-ball {
                width: 64px;
                height: 64px;
                border-radius: 50%;
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                display: flex;
                align-items: center;
                justify-content: center;
                box-shadow: 0 8px 32px rgba(102, 126, 234, 0.3);
                transition: all 0.3s ease;
                position: relative;
                cursor: pointer;
                border: 3px solid rgba(255, 255, 255, 0.2);
            }
            
            .voice-ball:hover {
                transform: scale(1.1);
                box-shadow: 0 12px 40px rgba(102, 126, 234, 0.4);
            }
            
            .voice-ball.calling {
                animation: pulse-call 2s infinite;
                background: linear-gradient(135deg, #ff6b6b 0%, #ee5a24 100%);
            }
            
            .voice-ball.voice-pulse {
                animation: voice-pulse 0.5s ease-in-out infinite alternate;
            }
            
            .voice-icon {
                color: white;
                transition: all 0.3s ease;
            }
            
            .voice-ball:hover .voice-icon {
                transform: scale(1.1);
            }
            
            .status-indicator {
                position: absolute;
                top: -2px;
                right: -2px;
                width: 16px;
                height: 16px;
                border-radius: 50%;
                background: #10b981;
                border: 3px solid white;
                opacity: 0;
                transition: all 0.3s ease;
            }
            
            .status-indicator.active {
                opacity: 1;
                animation: pulse-status 1.5s infinite;
            }
            
            @keyframes pulse-call {
                0%, 100% { 
                    transform: scale(1);
                    box-shadow: 0 8px 32px rgba(255, 107, 107, 0.3);
                }
                50% { 
                    transform: scale(1.05);
                    box-shadow: 0 12px 40px rgba(255, 107, 107, 0.5);
                }
            }
            
            @keyframes pulse-status {
                0%, 100% { 
                    transform: scale(1);
                    opacity: 1;
                }
                50% { 
                    transform: scale(1.2);
                    opacity: 0.8;
                }
            }
            
            @keyframes voice-pulse {
                0% { 
                    transform: scale(1);
                    box-shadow: 0 8px 32px rgba(102, 126, 234, 0.3);
                }
                100% { 
                    transform: scale(1.1);
                    box-shadow: 0 12px 40px rgba(102, 126, 234, 0.6);
                }
            }
            
            /* 浮动动画 */
            .ai-voice-assistant-btn {
                animation: float-gentle 6s ease-in-out infinite;
            }
            
            @keyframes float-gentle {
                0%, 100% { 
                    transform: translateY(0px) rotate(0deg); 
                }
                50% { 
                    transform: translateY(-8px) rotate(2deg); 
                }
            }
        `;
        document.head.appendChild(globalStyles);

        function createUI() {
            // 加载保存的主题
            const savedTheme = localStorage.getItem('aiChatTheme');
            if (savedTheme) {
                currentTheme = savedTheme;
            }
            
            // 助手数据从配置中获取
            
            // 创建浮动按钮
            const petBtn = document.createElement("button");
            petBtn.innerHTML = `
                <div class="voice-ball-container">
                    <div class="voice-ball ${isCalling ? 'calling' : ''}">
                        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" class="voice-icon">
                            <path d="M12 1C13.1 1 14 1.9 14 3V11C14 12.1 13.1 13 12 13C10.9 13 10 12.1 10 11V3C10 1.9 10.9 1 12 1ZM19 11C19 15.4 15.4 19 11 19H10V21H14V23H10V21H6V19H5C0.6 19 -3 15.4 -3 11H1C1 14.3 3.7 17 7 17H17C20.3 17 23 14.3 23 11H19Z" fill="currentColor"/>
                        </svg>
                        <div class="status-indicator ${isCalling ? 'active' : ''}"></div>
                    </div>
                </div>
            `;
            
            petBtn.className = "ai-voice-assistant-btn";
            
            // 添加内联样式
            petBtn.style.cssText = `
                position: fixed !important;
                bottom: 24px !important;
                right: 24px !important;
                z-index: 9999 !important;
                border: none !important;
                background: none !important;
                padding: 0 !important;
                cursor: pointer !important;
                transition: all 0.3s ease !important;
                outline: none !important;
            `;
            
            // 添加调试信息
            console.log('[AIPet] 创建语音球按钮:', petBtn);
            console.log('[AIPet] 按钮样式:', petBtn.className);
            console.log('[AIPet] 按钮内联样式:', petBtn.style.cssText);
            
            // 恢复位置
            const savedPosition = JSON.parse(localStorage.getItem("petBtnPosition") || 'null');
            if (savedPosition) {
                petBtn.style.left = savedPosition.left + 'px';
                petBtn.style.top = savedPosition.top + 'px';
                petBtn.style.right = 'auto';
                petBtn.style.bottom = 'auto';
            }

            // 拖拽功能（支持鼠标和触摸）
            function startDrag(e) {
                isDragging = true;
                const clientX = e.touches ? e.touches[0].clientX : e.clientX;
                const clientY = e.touches ? e.touches[0].clientY : e.clientY;
                dragOffsetX = clientX - petBtn.getBoundingClientRect().left;
                dragOffsetY = clientY - petBtn.getBoundingClientRect().top;
                document.body.style.userSelect = 'none';
                petBtn.style.animation = 'none';
                e.preventDefault();
            }

            function drag(e) {
                if (!isDragging) return;
                petBtn.style.transition = 'none';
                const clientX = e.touches ? e.touches[0].clientX : e.clientX;
                const clientY = e.touches ? e.touches[0].clientY : e.clientY;
                let x = clientX - dragOffsetX;
                let y = clientY - dragOffsetY;
                x = Math.max(0, Math.min(window.innerWidth - petBtn.offsetWidth, x));
                y = Math.max(0, Math.min(window.innerHeight - petBtn.offsetHeight, y));
                petBtn.style.left = x + 'px';
                petBtn.style.top = y + 'px';
                petBtn.style.right = 'auto';
                petBtn.style.bottom = 'auto';
                petBtn.style.position = 'fixed';
                localStorage.setItem("petBtnPosition", JSON.stringify({left: x, top: y}));
                e.preventDefault();
            }

            function endDrag() {
                isDragging = false;
                petBtn.style.transition = '';
                petBtn.style.animation = 'float 6s ease-in-out infinite';
                document.body.style.userSelect = '';
            }

            // 鼠标事件
            petBtn.addEventListener('mousedown', startDrag);
            document.addEventListener('mousemove', drag);
            document.addEventListener('mouseup', endDrag);

            // 触摸事件
            petBtn.addEventListener('touchstart', startDrag, {passive: false});
            document.addEventListener('touchmove', drag, {passive: false});
            document.addEventListener('touchend', endDrag);

            // 创建语音助手面板
            const panel = document.createElement("div");
            panel.className = "ai-chat-container fixed bottom-20 right-6 w-80 max-w-[calc(100vw-2rem)] max-h-[80vh] glass-effect rounded-2xl z-[9999] flex flex-col";
            panel.style.display = "none";
            panel.style.transform = "translateY(20px) scale(0.95)";
            panel.style.opacity = "0";
            panel.style.transition = "all 0.4s cubic-bezier(0.16, 1, 0.3, 1)";

            // 面板头部
            const panelHeader = document.createElement("div");
            panelHeader.className = "flex items-center justify-between p-3 border-b border-gray-200/30";
            panelHeader.innerHTML = `
                <div class="flex items-center space-x-2">
                    <div class="w-8 h-8 rounded-xl ai-avatar flex items-center justify-center">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" class="text-white">
                            <path d="M12 1C13.1 1 14 1.9 14 3V11C14 12.1 13.1 13 12 13C10.9 13 10 12.1 10 11V3C10 1.9 10.9 1 12 1ZM19 11C19 15.4 15.4 19 11 19H10V21H14V23H10V21H6V19H5C0.6 19 -3 15.4 -3 11H1C1 14.3 3.7 17 7 17H17C20.3 17 23 14.3 23 11H19Z" fill="currentColor"/>
                        </svg>
                    </div>
                    <div>
                        <h3 class="text-sm font-semibold gradient-text" id="assistantName">智能助手</h3>
                        <p class="text-xs text-gray-500" id="callStatus">待机中</p>
                    </div>
                </div>
                <div class="flex items-center space-x-1">
                    <button id="themeBtn" class="w-7 h-7 rounded-lg minimal-button flex items-center justify-center transition-all duration-200 hover:scale-105" title="主题">
                        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" class="text-gray-600">
                            <path d="M12 3V4M12 20V21M4 12H3M6.31412 6.31412L5.5 5.5M17.6859 6.31412L18.5 5.5M6.31412 17.69L5.5 18.5M17.6859 17.69L18.5 18.5M21 12H20M16 12C16 14.2091 14.2091 16 12 16C9.79086 16 8 14.2091 8 12C8 9.79086 9.79086 8 12 8C14.2091 8 16 9.79086 16 12Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                        </svg>
                    </button>
                    <button id="closePanel" class="w-7 h-7 rounded-lg minimal-button flex items-center justify-center transition-all duration-200 hover:scale-105" title="关闭">
                        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" class="text-gray-600">
                            <path d="M18 6L6 18M6 6L18 18" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
                        </svg>
                    </button>
                </div>
            `;

            // 语音球区域
            const voiceBallContainer = document.createElement("div");
            voiceBallContainer.className = "p-3 border-b border-gray-200/30 text-center";
            voiceBallContainer.innerHTML = `
                <div class="flex justify-center mb-3">
                    <button 
                        id="voiceBall" 
                        class="w-16 h-16 rounded-full ai-avatar flex items-center justify-center transition-all duration-300 hover:scale-105 ${isCalling ? 'animate-pulse' : ''}"
                        title="${isCalling ? '停止通话' : '开始通话'}"
                    >
                        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" class="text-white">
                            <path d="M12 1C13.1 1 14 1.9 14 3V11C14 12.1 13.1 13 12 13C10.9 13 10 12.1 10 11V3C10 1.9 10.9 1 12 1ZM19 11C19 15.4 15.4 19 11 19H10V21H14V23H10V21H6V19H5C0.6 19 -3 15.4 -3 11H1C1 14.3 3.7 17 7 17H17C20.3 17 23 14.3 23 11H19Z" fill="currentColor"/>
                        </svg>
                    </button>
                    </div>
                <div class="text-center">
                    <p class="text-xs text-purple-600 font-mono" id="callDuration" style="display: none;">00:00</p>
                </div>
            `;

            // 助手列表区域已移除

            // 聊天消息区域
            const messagesContainer = document.createElement("div");
            messagesContainer.id = "voiceMessages";
            messagesContainer.className = "flex-1 overflow-y-auto scrollbar-custom p-3 space-y-2";
            messagesContainer.style.maxHeight = "200px";

            // 控制面板区域
            const controlContainer = document.createElement("div");
            controlContainer.className = "p-3 border-t border-gray-200/30 bg-gray-50/30";
            controlContainer.innerHTML = `
                <div class="space-y-2">
                <div class="flex items-center justify-between">
                        <span class="text-xs font-medium text-gray-700">语音设置</span>
                        <button id="toggleControlPanel" class="w-5 h-5 rounded minimal-button flex items-center justify-center" title="展开/折叠">
                            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" class="text-gray-600">
                                <path d="M9 5L15 12L9 19" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                            </svg>
                    </button>
                    </div>
                    <div id="controlPanelContent" class="space-y-2" style="display: none;">
                        <div class="grid grid-cols-2 gap-2">
                            <div>
                                <label class="text-xs text-gray-500">语速</label>
                                <input type="range" id="speedSlider" min="0.5" max="2" step="0.1" value="${speed}" class="w-full h-1">
                            </div>
                            <div>
                                <label class="text-xs text-gray-500">音量</label>
                                <input type="range" id="volumeSlider" min="1" max="10" value="${volume}" class="w-full h-1">
                            </div>
                        </div>
                        <div>
                            <label class="text-xs text-gray-500">语言</label>
                            <select id="languageSelect" class="w-full px-2 py-1 text-xs border border-gray-200 rounded">
                                <option value="zh-cn" ${language === 'zh-cn' ? 'selected' : ''}>中文</option>
                                <option value="en-us" ${language === 'en-us' ? 'selected' : ''}>English</option>
                            </select>
                        </div>
                        <div class="flex space-x-1">
                            <button id="saveSettingsBtn" class="flex-1 px-2 py-1 bg-blue-500 text-white text-xs rounded hover:bg-blue-600 transition-colors">
                                保存
                        </button>
                            <button id="resetSettingsBtn" class="px-2 py-1 bg-gray-500 text-white text-xs rounded hover:bg-gray-600 transition-colors">
                                重置
                        </button>
                        </div>
                    </div>
                </div>
            `;

            // 组装面板
            panel.appendChild(panelHeader);
            panel.appendChild(voiceBallContainer);
            panel.appendChild(messagesContainer);
            panel.appendChild(controlContainer);
            document.body.appendChild(panel);

            // 助手列表已移除

            // 事件监听
            petBtn.onclick = (e) => {
                if (isDragging) return;
                togglePanel();
            };

            document.getElementById('closePanel').onclick = () => {
                hidePanel();
            };

            // 语音球点击事件
            document.getElementById('voiceBall').onclick = () => {
                if (isCalling) {
                    stopCall();
                } else {
                    startCall();
                }
            };

            // 设置按钮已移除，配置在代码中

            // 主题切换
            document.getElementById('themeBtn').onclick = () => {
                toggleTheme();
            };

            // 添加助手按钮已移除

            // 控制面板折叠
            document.getElementById('toggleControlPanel').onclick = () => {
                toggleControlPanel();
            };

            // 设置滑块事件
            document.getElementById('speedSlider').addEventListener('input', (e) => {
                speed = parseFloat(e.target.value);
            });

            document.getElementById('volumeSlider').addEventListener('input', (e) => {
                volume = parseInt(e.target.value);
            });

            document.getElementById('languageSelect').addEventListener('change', (e) => {
                language = e.target.value;
            });

            // 保存设置
            document.getElementById('saveSettingsBtn').onclick = () => {
                saveSettings();
            };

            // 重置设置
            document.getElementById('resetSettingsBtn').onclick = () => {
                resetSettings();
            };

            document.body.appendChild(petBtn);
            
            // 添加调试信息
            console.log('[AIPet] 语音球已添加到页面');
            console.log('[AIPet] 页面中的 .ai-chat-container 元素数量:', document.querySelectorAll('.ai-chat-container').length);
            
            // 应用主题
            applyTheme();

            function togglePanel() {
                const isVisible = panel.style.display !== "none";
                if (isVisible) {
                    hidePanel();
                } else {
                    showPanel();
                }
            }

            function showPanel() {
                panel.style.display = "flex";
                panel.style.flexDirection = "column";
                panel.classList.add('panel-enter');
                setTimeout(() => {
                    panel.style.transform = "translateY(0) scale(1)";
                    panel.style.opacity = "1";
                }, 10);
            }

            function hidePanel() {
                panel.style.transform = "translateY(20px) scale(0.95)";
                panel.style.opacity = "0";
                panel.classList.remove('panel-enter');
                setTimeout(() => {
                    panel.style.display = "none";
                }, 400);
            }

            // addMessageToUI函数已移到全局作用域

            function clearMessages(container) {
                container.innerHTML = '';
                messageHistory = [];
            }

            function sendMessage(container) {
                const input = document.getElementById('messageInput');
                const message = input.value.trim();
                if (!message) return;

                const userMessage = addMessage(message, 'user');
                addMessageToUI(container, userMessage);
                input.value = '';

                // 显示打字指示器
                showTypingIndicator(container);
                
                // 模拟AI回复（实际应用中这里应该调用后端API）
                setTimeout(() => {
                    hideTypingIndicator(container);
                    const aiResponse = addMessage("这是一个模拟的AI回复。在实际应用中，这里会调用后端API获取真实的AI响应。", 'assistant');
                    addMessageToUI(container, aiResponse);
                }, 2000);
            }

            // 移除未使用的函数

            // 清理资源函数
            function cleanup() {
                console.log('[Cleanup] 清理资源');
                if (callTimer) {
                    clearInterval(callTimer);
                }
                if (socket && socket.readyState !== WebSocket.CLOSED) {
                    socket.close();
                }
                if (peerConnection && peerConnection.connectionState !== 'closed') {
                    peerConnection.close();
                }
                if (localStream) {
                    localStream.getTracks().forEach(track => track.stop());
                }
                stopAudioAnalysis();
                if (audioContext) {
                    audioContext.close();
                }
            }

            // 页面卸载时清理资源
            window.addEventListener('beforeunload', cleanup);
        }

        createUI();
        
        // 标记已加载
        window.__AIPetLoaded = true;
        console.log('[AIPet] 语音助手界面创建完成');
    }

    window.addEventListener("DOMContentLoaded", () => {
        loadTailwind(() => loadAxios(main));
    });
})();
