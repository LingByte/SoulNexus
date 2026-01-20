/**
 * LingEcho SDK 完整简单示例
 *
 * 包含：
 * - SDK检测
 * - 语音通话
 * - 获取助手信息
 * - 简单的UI界面
 */

(function() {
    'use strict';

    // ========== 配置 ==========
    const CONFIG = {
        assistantId: /*{{.AssistantID}}*/ 1,  // 模板变量，后端会自动替换为实际ID
        apiKey: 'your-api-key',        // 请替换为你的API密钥
        apiSecret: 'your-api-secret'   // 请替换为你的API密钥
    };

    // ========== 初始化 ==========
    async function init() {
        // 先检查SDK是否已经就绪
        if (window.__LINGECHO_SDK_READY__ && window.lingEcho) {
            createUI(window.lingEcho);
            return;
        }

        // 监听SDK就绪事件
        if (typeof window.addEventListener !== 'undefined') {
            window.addEventListener('lingecho-sdk-ready', () => {
                if (window.lingEcho) {
                    createUI(window.lingEcho);
                }
            }, { once: true });
        }

        // 检测并等待SDK
        const sdk = await getSDK();
        if (!sdk) {
            // 提供更友好的错误提示和重试机制
            showErrorWithRetry();
            return;
        }

        // 初始化UI
        createUI(sdk);
    }

    // 显示错误并提供重试
    function showErrorWithRetry() {
        const errorDiv = document.createElement('div');
        errorDiv.id = 'lingecho-error';
        errorDiv.innerHTML = `
            <div style="position: fixed; top: 50%; left: 50%; transform: translate(-50%, -50%); 
                        background: white; padding: 30px; border-radius: 12px; 
                        box-shadow: 0 4px 20px rgba(0,0,0,0.15); z-index: 10000; max-width: 400px;">
                <h3 style="margin: 0 0 12px 0; color: #1f2937;">SDK加载中...</h3>
                <p style="margin: 0 0 20px 0; color: #6b7280; font-size: 14px;">
                    正在加载LingEcho SDK，请稍候...
                </p>
                <div style="display: flex; gap: 12px;">
                    <button id="retry-btn" style="flex: 1; padding: 10px; background: #3b82f6; 
                            color: white; border: none; border-radius: 6px; cursor: pointer; font-size: 14px;">
                        重试
                    </button>
                    <button id="close-btn" style="flex: 1; padding: 10px; background: #e5e7eb; 
                            color: #6b7280; border: none; border-radius: 6px; cursor: pointer; font-size: 14px;">
                        关闭
                    </button>
                </div>
            </div>
        `;
        document.body.appendChild(errorDiv);

        // 自动重试
        let retryCount = 0;
        const maxRetries = 10;
        const autoRetry = setInterval(async () => {
            retryCount++;
            const sdk = await getSDK();
            if (sdk) {
                clearInterval(autoRetry);
                errorDiv.remove();
                createUI(sdk);
            } else if (retryCount >= maxRetries) {
                clearInterval(autoRetry);
                errorDiv.querySelector('h3').textContent = 'SDK加载失败';
                errorDiv.querySelector('p').textContent = '无法加载SDK，请检查网络连接或刷新页面';
            }
        }, 1000);

        // 手动重试按钮
        document.getElementById('retry-btn').onclick = async () => {
            const sdk = await getSDK();
            if (sdk) {
                clearInterval(autoRetry);
                errorDiv.remove();
                createUI(sdk);
            } else {
                alert('SDK仍未加载，请刷新页面');
            }
        };

        // 关闭按钮
        document.getElementById('close-btn').onclick = () => {
            clearInterval(autoRetry);
            errorDiv.remove();
        };
    }

    // ========== 获取SDK实例 ==========
    async function getSDK() {
        // 如果已加载，直接返回
        if (window.lingEcho && window.lingEcho instanceof LingEchoSDK) {
            return window.lingEcho;
        }

        // 检查SDK是否已就绪（通过全局标记）
        if (window.__LINGECHO_SDK_READY__ && window.lingEcho) {
            return window.lingEcho;
        }

        // 等待加载（增加等待时间和重试机制）
        if (typeof LingEchoSDK !== 'undefined' && LingEchoSDK.waitFor) {
            try {
                return await LingEchoSDK.waitFor(10000); // 增加到10秒
            } catch (error) {
                console.error('SDK加载失败:', error);
                // 尝试手动创建实例
                return tryCreateSDKInstance();
            }
        }

        // 如果SDK类未定义，等待一段时间后重试
        return await waitAndRetry();
    }

    // 尝试手动创建SDK实例
    function tryCreateSDKInstance() {
        if (typeof LingEchoSDK !== 'undefined') {
            const serverBase = typeof SERVER_BASE !== 'undefined' ? SERVER_BASE :
                (window.SERVER_BASE || '{{.BaseURL}}');
            const assistantName = typeof ASSISTANT_NAME !== 'undefined' ? ASSISTANT_NAME :
                (window.ASSISTANT_NAME || '{{.Name}}');

            try {
                const sdk = new LingEchoSDK({
                    baseURL: serverBase,
                    assistantName: assistantName
                });
                window.lingEcho = sdk;
                return sdk;
            } catch (error) {
                console.error('创建SDK实例失败:', error);
            }
        }
        return null;
    }

    // 等待并重试
    async function waitAndRetry(maxRetries = 20, delay = 500) {
        for (let i = 0; i < maxRetries; i++) {
            await new Promise(resolve => setTimeout(resolve, delay));

            // 检查SDK是否已加载
            if (window.lingEcho && window.lingEcho instanceof LingEchoSDK) {
                return window.lingEcho;
            }

            // 检查SDK类是否已定义
            if (typeof LingEchoSDK !== 'undefined') {
                const sdk = tryCreateSDKInstance();
                if (sdk) return sdk;
            }
        }
        return null;
    }

    // ========== 创建UI ==========
    function createUI(sdk) {
        const container = document.createElement('div');
        container.id = 'lingecho-widget';
        container.innerHTML = `
            <div style="position: fixed; bottom: 20px; right: 20px; z-index: 1000;">
                <div style="background: white; border-radius: 16px; padding: 16px; box-shadow: 0 4px 20px rgba(0,0,0,0.15); min-width: 200px;">
                    <div style="text-align: center; margin-bottom: 12px;">
                        <h3 style="margin: 0; font-size: 16px; color: #1f2937;">语音助手</h3>
                        <p id="status" style="margin: 4px 0 0 0; font-size: 12px; color: #6b7280;">就绪</p>
                    </div>
                    <button id="voice-btn" style="width: 100%; padding: 12px; border-radius: 8px; border: none; background: #3b82f6; color: white; font-size: 14px; cursor: pointer; font-weight: 500;">
                        🎤 开始语音
                    </button>
                    <button id="info-btn" style="width: 100%; padding: 8px; margin-top: 8px; border-radius: 8px; border: 1px solid #e5e7eb; background: white; color: #6b7280; font-size: 12px; cursor: pointer;">
                        查看助手信息
                    </button>
                </div>
            </div>
        `;
        document.body.appendChild(container);

        // 绑定事件
        setupEvents(sdk);
    }

    // ========== 设置事件 ==========
    function setupEvents(sdk) {
        const voiceBtn = document.getElementById('voice-btn');
        const infoBtn = document.getElementById('info-btn');
        const status = document.getElementById('status');
        let isConnected = false;

        // 语音按钮
        voiceBtn.onclick = async () => {
            if (!isConnected) {
                try {
                    voiceBtn.disabled = true;
                    voiceBtn.textContent = '⏳ 连接中...';
                    status.textContent = '正在连接...';

                    await sdk.connectVoice({
                        assistantId: CONFIG.assistantId,
                        apiKey: CONFIG.apiKey,
                        apiSecret: CONFIG.apiSecret,
                        onOpen: () => {
                            isConnected = true;
                            voiceBtn.textContent = '📞 结束通话';
                            voiceBtn.style.background = '#ef4444';
                            status.textContent = '通话中';
                            voiceBtn.disabled = false;
                            showToast('语音通话已连接', 'success');
                        },
                        onClose: () => {
                            isConnected = false;
                            voiceBtn.textContent = '🎤 开始语音';
                            voiceBtn.style.background = '#3b82f6';
                            status.textContent = '已断开';
                            showToast('语音通话已断开', 'info');
                        },
                        onError: (error) => {
                            voiceBtn.textContent = '🎤 开始语音';
                            voiceBtn.style.background = '#3b82f6';
                            status.textContent = '连接失败';
                            voiceBtn.disabled = false;
                            showToast('连接失败: ' + error.message, 'error');
                        }
                    });
                } catch (error) {
                    voiceBtn.textContent = '🎤 开始语音';
                    voiceBtn.style.background = '#3b82f6';
                    status.textContent = '连接失败';
                    voiceBtn.disabled = false;
                    showToast('连接失败: ' + error.message, 'error');
                }
            } else {
                sdk.disconnectVoice();
            }
        };

        // 信息按钮
        infoBtn.onclick = async () => {
            try {
                infoBtn.disabled = true;
                infoBtn.textContent = '加载中...';

                const response = await sdk.getAssistant(CONFIG.assistantId);
                const assistant = response.data;

                alert(`助手信息:\n\n名称: ${assistant.name}\n描述: ${assistant.description || '无'}\n语言: ${assistant.language}\n温度: ${assistant.temperature}`);
            } catch (error) {
                showToast('获取信息失败: ' + error.message, 'error');
            } finally {
                infoBtn.disabled = false;
                infoBtn.textContent = '查看助手信息';
            }
        };
    }

    // ========== 工具函数 ==========
    function showToast(message, type = 'info') {
        const colors = {
            success: '#10b981',
            error: '#ef4444',
            info: '#3b82f6'
        };

        const toast = document.createElement('div');
        toast.textContent = message;
        toast.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            padding: 12px 24px;
            background: ${colors[type] || colors.info};
            color: white;
            border-radius: 8px;
            font-size: 14px;
            z-index: 10001;
            animation: slideIn 0.3s ease;
        `;

        // 添加动画样式（如果还没有）
        if (!document.querySelector('#toast-anim')) {
            const style = document.createElement('style');
            style.id = 'toast-anim';
            style.textContent = `
                @keyframes slideIn {
                    from { transform: translateX(100%); opacity: 0; }
                    to { transform: translateX(0); opacity: 1; }
                }
            `;
            document.head.appendChild(style);
        }

        document.body.appendChild(toast);
        setTimeout(() => toast.remove(), 3000);
    }

    function showError(message) {
        const error = document.createElement('div');
        error.textContent = message;
        error.style.cssText = `
            position: fixed;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            padding: 20px 40px;
            background: #fee2e2;
            border: 2px solid #ef4444;
            border-radius: 8px;
            color: #991b1b;
            z-index: 10000;
        `;
        document.body.appendChild(error);
    }

    // ========== 启动 ==========
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

})();

