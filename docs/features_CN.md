# 功能文档

## 🎯 核心功能

### 1. 🎤 实时语音交互

<div align="center">
  <img src="webrtc.png" alt="WebRTC 通信流程" width="600">
</div>

- **WebRTC技术** - 实现低延迟实时语音通话
- **多路音频处理** - 支持多用户同时通话
- **音频质量优化** - 自适应码率和降噪处理
- **跨平台支持** - 支持Web、移动端和桌面端

### 2. 🔄 可视化工作流自动化

<div align="center">
  <img src="page-workflow.png" alt="工作流设计器" width="800">
</div>

- **拖拽式设计器** - 直观的节点连接界面
- **多种节点类型** - 开始、结束、脚本、任务、条件等多种节点类型
- **实时执行监控** - 可视化执行进度和状态，支持WebSocket实时日志流
- **节点自测功能** - 支持单个节点独立测试，自定义输入参数
- **多种触发方式**：
  - **API触发** - 公开或需要认证的API端点，供外部系统调用
  - **事件触发** - 监听系统事件，自动触发工作流执行
  - **定时触发** - 基于Cron表达式的定时执行
  - **Webhook触发** - 接收外部服务的Webhook请求
  - **智能体触发** - 允许AI智能体将工作流作为工具调用
- **错误处理机制** - 自动重试和异常恢复
- **参数管理** - 为每个节点定义输入/输出参数

**工作流API调用示例**：
```bash
# 通过公开API触发工作流
curl -X POST http://localhost:7072/api/public/workflows/my-workflow/execute \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "parameters": {
      "userName": "张三",
      "orderId": "12345",
      "amount": 99.99
    }
  }'
```

### 3. 🎨 声音克隆技术

<div align="center">
  <img src="page-voice-clone.png" alt="声音克隆" width="800">
</div>

- **音色训练** - 支持自定义声音模型训练
- **多音色支持** - 男声、女声、儿童声等多种音色
- **音质优化** - 高保真音频合成
- **个性化定制** - 专属AI助手音色
- **训练任务管理** - 跟踪训练进度，管理音色模型

### 4. 🤖 智能对话引擎

<div align="center">
  <img src="page-debug-assistant.png" alt="助手调试" width="800">
</div>

- **多模型支持** - GPT、Claude、DeepSeek等主流模型
- **上下文理解** - 长对话记忆和上下文关联
- **知识库集成** - 知识库智能问答
- **工作流集成** - AI智能体可以将工作流作为工具调用

### 5. 🔧 应用接入功能

<div align="center">
  <img src="applicationclient.png" alt="应用接入流程" width="800">
</div>

<div align="center">
  <img src="page-js-template.png" alt="JS模板" width="800">
</div>

- **JS注入方式** - 快速接入新应用，实现无痛集成
- **API网关** - 统一的API管理和访问控制
- **密钥管理** - 企业级密钥管理系统，支持自定义密钥配置
- **计费系统** - 灵活的计费策略和监控
- **凭证管理** - 安全的API凭证存储和管理

### 6. 📞 SIP软电话系统

<div align="center">
  <img src="page-sip-call-record.png" alt="SIP通话记录" width="800">
</div>

<div align="center">
  <img src="page-sip-call-details.png" alt="SIP通话详情" width="800">
</div>

- **SIP协议支持** - 标准SIP协议实现
- **软电话功能** - 企业级软电话解决方案
- **通话记录** - 完整的通话历史管理，支持详细查看
- **高性能语音处理** - 实时音频编解码和处理
- **ACD自动呼叫分配** - 智能呼叫路由和坐席管理
- **音频播放** - 支持通话录音播放，带波形可视化
- **追踪文件下载** - 支持下载对话追踪文件进行分析

### 7. 🔌 设备管理

- **设备注册** - 注册和管理IoT设备
- **OTA固件升级** - 支持设备固件远程升级
- **设备监控** - 实时设备状态和健康监控
- **远程控制** - 通过平台远程控制设备
- **设备分组** - 将设备分组管理，支持批量操作

### 8. 🚨 告警系统

- **基于规则的监控** - 根据指标和条件定义自定义告警规则
- **多渠道通知** - 支持邮件、站内通知、Webhook和短信（预留）
- **告警严重程度** - 严重、高、中、低四个级别分类
- **告警管理** - 跟踪、解决和静音告警
- **告警历史** - 完整的告警审计记录

### 9. 💰 账单系统

- **用量追踪** - 详细记录所有服务的使用情况
- **账单生成** - 自动和手动账单生成
- **配额管理** - 用户和组织配额管理
- **用量分析** - 全面的用量统计和报告
- **成本分配** - 按用户、组织或服务跟踪成本

### 10. 👥 组织管理

- **多租户支持** - 支持多个组织
- **团队管理** - 在组织内创建和管理团队
- **成员管理** - 邀请、管理和移除成员
- **资源共享** - 在团队内共享知识库、工作流等资源
- **权限控制** - 细粒度的权限管理

### 11. 📚 知识库管理

- **文档存储** - 在知识库中存储和管理文档
- **智能检索** - AI驱动的文档搜索和检索
- **多提供商支持** - 支持多个知识库提供商（如阿里云百炼）
- **版本控制** - 跟踪文档版本和变更
- **组织共享** - 在组织内共享知识库

### 12. 🔊 VAD语音活动检测服务

- **独立服务** - 基于SileroVAD的独立HTTP服务（端口7073）
- **多格式支持** - 支持PCM和OPUS音频格式输入
- **实时检测** - 提供`have_voice`和`voice_stop`状态检测
- **会话管理** - 支持多会话状态管理
- **双阈值机制** - 高/低阈值检测，滑动窗口平滑处理

**快速启动**：
```bash
cd services/vad-service
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
python vad_service.py
```

### 13. 🎙️ 声纹识别服务

- **ModelScope集成** - 基于ModelScope的声纹识别模型
- **说话人识别** - 支持多说话人识别和相似度计算
- **声纹注册** - 支持声纹特征提取和数据库存储
- **RESTful API** - 提供完整的HTTP API接口（端口7074）

**快速启动**：
```bash
cd services/voiceprint-api
python3.10 -m venv venv  # 推荐使用Python 3.10
source venv/bin/activate
pip install --upgrade pip setuptools wheel
pip install torch torchaudio --index-url https://download.pytorch.org/whl/cpu
pip install -r requirements.txt
python -m app.main
```

### 14. 🔌 硬件设备支持

- **xiaozhi协议** - 完整的xiaozhi WebSocket协议支持
- **实时音频处理** - 支持OPUS和PCM音频格式
- **多ASR/TTS提供商** - 支持腾讯、七牛、FunASR、Google、Volcengine、Gladia等
- **LLM集成** - 支持多种大语言模型接入
- **状态管理** - 完整的客户端状态和会话管理

