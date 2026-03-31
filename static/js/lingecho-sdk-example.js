// /**
//  * LingEcho SDK 使用示例
//  *
//  * 这个文件展示了如何在JS模板中使用LingEcho SDK
//  */
//
// (function() {
//     'use strict';
//
//     // ========== 示例1: 基本初始化 ==========
//     // SDK会自动加载，你也可以手动初始化
//     const sdk = new LingEchoSDK({
//         baseURL: '{{.BaseURL}}',  // 或使用 SERVER_BASE 变量
//         apiKey: 'your-api-key',
//         apiSecret: 'your-api-secret',
//         assistantId: {{.AssistantID}},
//         assistantName: '{{.Name}}'
//     });
//
//     // 或者使用全局实例（如果SDK已自动加载）
//     // const sdk = window.lingEcho;
//
//     // ========== 示例2: 连接语音通话 ==========
//     async function connectVoiceCall() {
//         try {
//             const connection = await sdk.connectVoice({
//                 assistantId: {{.AssistantID}},
//                 apiKey: 'your-api-key',
//                 apiSecret: 'your-api-secret',
//                 onMessage: (message) => {
//                     console.log('收到消息:', message);
//                 },
//                 onError: (error) => {
//                     console.error('连接错误:', error);
//                 },
//                 onOpen: () => {
//                     console.log('语音通话已连接');
//                 },
//                 onClose: () => {
//                     console.log('语音通话已断开');
//                 },
//                 onTrack: (stream) => {
//                     // 自定义处理远端音频流
//                     const audio = new Audio();
//                     audio.srcObject = stream;
//                     audio.play();
//                 }
//             });
//
//             console.log('连接成功:', connection);
//         } catch (error) {
//             console.error('连接失败:', error);
//         }
//     }
//
//     // ========== 示例3: 使用事件系统 ==========
//     sdk.on('open', () => {
//         console.log('WebSocket已打开');
//     });
//
//     sdk.on('message', (data) => {
//         console.log('收到消息:', data);
//     });
//
//     sdk.on('error', (error) => {
//         console.error('发生错误:', error);
//     });
//
//     // ========== 示例4: 获取助手信息 ==========
//     async function getAssistantInfo() {
//         try {
//             const response = await sdk.getAssistant({{.AssistantID}});
//             console.log('助手信息:', response.data);
//         } catch (error) {
//             console.error('获取助手信息失败:', error);
//         }
//     }
//
//     // ========== 示例5: 获取助手列表 ==========
//     async function listAssistants() {
//         try {
//             const response = await sdk.getAssistants();
//             console.log('助手列表:', response.data);
//         } catch (error) {
//             console.error('获取助手列表失败:', error);
//         }
//     }
//
//     // ========== 示例6: 创建助手 ==========
//     async function createAssistant() {
//         try {
//             const response = await sdk.createAssistant({
//                 name: '我的新助手',
//                 description: '这是一个测试助手',
//                 icon: '🤖'
//             });
//             console.log('创建成功:', response.data);
//         } catch (error) {
//             console.error('创建失败:', error);
//         }
//     }
//
//     // ========== 示例7: 更新助手 ==========
//     async function updateAssistant() {
//         try {
//             const response = await sdk.updateAssistant({{.AssistantID}}, {
//                 name: '更新后的名称',
//                 temperature: 0.8,
//                 maxTokens: 200
//             });
//             console.log('更新成功:', response.data);
//         } catch (error) {
//             console.error('更新失败:', error);
//         }
//     }
//
//     // ========== 示例8: 获取聊天日志 ==========
//     async function getChatLogs() {
//         try {
//             // 获取指定助手的聊天日志
//             const response = await sdk.getChatLogsByAssistant({{.AssistantID}});
//             console.log('聊天日志:', response.data);
//         } catch (error) {
//             console.error('获取聊天日志失败:', error);
//         }
//     }
//
//     // ========== 示例10: 完整的语音助手UI示例 ==========
//     function createVoiceAssistantUI() {
//         const container = document.createElement('div');
//         container.innerHTML = `
//             <div style="position: fixed; bottom: 20px; right: 20px; z-index: 1000;">
//                 <button id="voice-btn" style="width: 60px; height: 60px; border-radius: 50%; background: #3b82f6; color: white; border: none; cursor: pointer; font-size: 24px;">
//                     🎤
//                 </button>
//             </div>
//         `;
//         document.body.appendChild(container);
//
//         const btn = document.getElementById('voice-btn');
//         let isConnected = false;
//
//         btn.addEventListener('click', async () => {
//             if (!isConnected) {
//                 // 连接语音通话
//                 try {
//                     await sdk.connectVoice({
//                         assistantId: {{.AssistantID}},
//                         onOpen: () => {
//                             isConnected = true;
//                             btn.style.background = '#ef4444';
//                             btn.textContent = '📞';
//                             sdk.showNotification('语音通话已连接', 'success');
//                         },
//                         onClose: () => {
//                             isConnected = false;
//                             btn.style.background = '#3b82f6';
//                             btn.textContent = '🎤';
//                             sdk.showNotification('语音通话已断开', 'info');
//                         },
//                         onError: (error) => {
//                             sdk.showNotification('连接失败: ' + error.message, 'error');
//                         }
//                     });
//                 } catch (error) {
//                     sdk.showNotification('连接失败: ' + error.message, 'error');
//                 }
//             } else {
//                 // 断开连接
//                 sdk.disconnectVoice();
//             }
//         });
//     }
//
//     // ========== 示例11: 使用模板变量 ==========
//     // 在模板中，你可以使用以下变量：
//     // - {{.BaseURL}} - 服务器基础URL
//     // - {{.Name}} - 助手名称
//     // - {{.AssistantID}} - 助手ID
//     // - {{.JsSourceID}} - JS源ID
//     // - {{.Description}} - 助手描述
//     // - {{.Language}} - 语言设置
//     // - {{.Speaker}} - 发音人
//     // - {{.TtsProvider}} - TTS提供商
//     // - {{.LLMModel}} - LLM模型
//     // - {{.Temperature}} - 温度参数
//     // - {{.MaxTokens}} - 最大token数
//     // - {{.Speed}} - 语速
//     // - {{.Volume}} - 音量
//     // - {{.SERVER_BASE}} - 服务器基础URL（别名）
//     // - {{.ASSISTANT_NAME}} - 助手名称（别名）
//
//     console.log('当前助手:', {
//         name: '{{.Name}}',
//         id: {{.AssistantID}},
//         baseURL: '{{.BaseURL}}'
//     });
//
//     // ========== 示例12: 简单的语音助手按钮 ==========
//     // 在页面加载时创建语音助手按钮
//     if (document.readyState === 'loading') {
//         document.addEventListener('DOMContentLoaded', createVoiceAssistantUI);
//     } else {
//         createVoiceAssistantUI();
//     }
//
// })();
//
