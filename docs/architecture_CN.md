# 架构文档

## 🏗️ 技术架构

<div align="center">
  <img src="core.png" alt="LingEcho 核心架构" width="800">
</div>

## 核心WebRTC通话流程

<div align="center">
  <img src="core-process.png" alt="WebRTC 核心通话流程" width="800">
</div>

## 服务架构

| 服务 | 端口 | 技术栈 | 说明 |
|------|------|--------|------|
| **主服务** | 7072 | Go + Gin | 核心后端服务 |
| **语音服务** | 8000 | Go | WebSocket语音服务 |
| **VAD服务** | 7073 | Python + FastAPI | 语音活动检测服务 |
| **声纹识别服务** | 7074 | Python + FastAPI | 声纹识别服务 |
| **前端服务** | 3000 | React + Vite | 开发环境前端 |

## 系统架构图

<div align="center">
  <img src="ArchitectureDiagram.png" alt="系统架构" width="800">
</div>

## 语音服务器核心

<div align="center">
  <img src="voice-server-core.png" alt="语音服务器核心" width="800">
</div>

## SSE (服务器推送事件) 流程

<div align="center">
  <img src="sse.png" alt="SSE流程" width="600">
</div>

## 📦 服务组件

### 核心服务

- **主服务** (`cmd/server/`) - 核心后端服务，提供RESTful API和WebSocket支持
- **语音/SIP服务** - 已集成到主服务中，提供SIP软电话和语音处理功能
- **MCP服务** (`cmd/mcp/`) - Model Context Protocol服务

### 独立服务

- **VAD服务** (`services/vad-service/`) - 基于SileroVAD的语音活动检测服务
  - 支持PCM和OPUS格式
  - HTTP RESTful API
  - 实时语音活动检测
  - 静默检测和会话管理

- **声纹识别服务** (`services/voiceprint-api/`) - 基于ModelScope的声纹识别服务
  - 说话人识别
  - 声纹注册和管理
  - MySQL数据库存储
  - RESTful API接口

