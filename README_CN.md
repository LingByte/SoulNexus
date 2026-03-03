# SoulNexus 灵枢

<div align="center">
<div align="center">
  <img src="docs/logo.png" alt="LingEcho Logo" width="100" height="110">
</div>

**智能语音交互平台 - 让AI拥有真实的声音**

[![Go Version](https://img.shields.io/badge/Go-1.25.1-blue.svg)](https://golang.org/)
[![React](https://img.shields.io/badge/React-18.2.0-61dafb.svg)](https://reactjs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.2.2-3178c6.svg)](https://www.typescriptlang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)]()
[![在线演示](https://img.shields.io/badge/在线演示-lingecho.com-brightgreen.svg)](https://lingecho.com)

[English](README.md) | [中文](README_CN.md)

### 🌐 在线演示

**在线体验 LingEcho**: [https://lingecho.com](https://lingecho.com)

</div>

---

## 📖 项目简介

LingEcho 灵语回响是一个基于 Go + React 的企业级智能语音交互平台，为用户提供完整的AI语音交互解决方案。集成了先进的语音识别（ASR）、语音合成（TTS）、大语言模型（LLM）和实时通信技术，支持实时通话、声音克隆、知识库管理、工作流自动化、设备管理、告警、账单、组织管理等企业级功能。

### ✨ 核心特性

- **AI人物实时通话** - 基于WebRTC技术实现与AI人物的实时语音通话，支持高质量音频传输和低延迟交互
- **声音克隆与训练** - 支持自定义声音训练和克隆，让AI助手拥有专属音色，打造个性化语音体验
- **工作流自动化** - 可视化工作流设计器，支持多种触发方式（API、事件、定时、Webhook、智能体），实现复杂业务流程自动化
- **知识库管理** - 强大的知识库管理系统，支持文档存储、检索和AI分析，为企业提供智能知识服务
- **应用接入功能** - 通过JS注入方式快速接入新应用，API网关和密钥管理，实现无痛集成
- **设备管理** - 完整的设备管理系统，支持OTA固件升级、设备监控和远程控制
- **告警系统** - 完善的告警系统，支持基于规则的监控、多渠道通知和告警管理
- **账单系统** - 灵活的计费和用量追踪系统，支持详细的使用记录、账单生成和配额管理
- **组织管理** - 多租户组织管理，支持团队协作、成员管理和资源共享
- **密钥管理与API平台** - 企业级密钥管理系统和API开发平台
- **VAD语音活动检测** - 独立的SileroVAD服务，支持PCM和OPUS格式
- **声纹识别服务** - 基于ModelScope的声纹识别服务，支持说话人识别
- **ASR-TTS服务** - 独立的ASR（Whisper）和TTS（edge-tts）服务，支持语音识别和文本转语音合成
- **MCP服务** - Model Context Protocol服务，支持SSE和stdio传输方式
- **硬件设备支持** - 支持xiaozhi协议的硬件设备接入，提供完整的WebSocket通信

---

## 🖼️ 平台截图

### 工作流自动化
<div align="center">
  <img src="docs/page-workflow.png" alt="工作流设计器" width="800">
  <p><em>可视化工作流设计器，支持拖拽式操作</em></p>
</div>

### 声音克隆
<div align="center">
  <img src="docs/page-voice-clone.png" alt="声音克隆" width="800">
  <p><em>声音克隆和训练界面</em></p>
</div>

### 助手调试
<div align="center">
  <img src="docs/page-debug-assistant.png" alt="助手调试" width="800">
  <p><em>AI助手调试和测试界面</em></p>
</div>

### JS模板集成
<div align="center">
  <img src="docs/page-js-template.png" alt="JS模板" width="800">
  <p><em>用于应用集成的JavaScript模板</em></p>
</div>

---

## 🏗️ 技术架构

<div align="center">
  <img src="docs/core.png" alt="LingEcho 核心架构" width="800">
</div>

### 服务架构

| 服务 | 端口 | 技术栈 | 说明 |
|------|------|--------|------|
| **主服务** | 7072 | Go + Gin | 核心后端服务，提供RESTful API和WebSocket支持 |
| **VAD服务** | 7073 | Python + FastAPI | 语音活动检测服务（SileroVAD） |
| **声纹识别服务** | 7074 | Python + FastAPI | 声纹识别服务（ModelScope） |
| **ASR-TTS服务** | 7075 | Python + FastAPI | ASR（Whisper）和TTS（edge-tts）服务 |
| **MCP服务** | 3001 | Go | Model Context Protocol服务（SSE传输，可选） |
| **前端服务** | 5173 | React + Vite | 开发环境前端（Vite开发服务器） |

详细的架构文档请查看 [架构文档](docs/architecture_CN.md)。

---

## 🚀 快速开始

### 环境要求

- **Go** >= 1.24.0
- **Node.js** >= 18.0.0
- **npm** >= 8.0.0 或 **pnpm** >= 8.0.0
- **Git**
- **Python** >= 3.10 (可选服务需要：VAD、声纹识别、ASR-TTS)
- **Docker** & **Docker Compose** (容器化部署，推荐)

### 安装方法

#### 方法一：Docker Compose（推荐）

使用 Docker Compose 是最简单的启动方式：

```bash
# 启动 Neo4j（如果需要）
docker run -d --name neo4j \
  -p 7474:7474 -p 7687:7687 \
  -e NEO4J_AUTH=neo4j/admin123 \
  neo4j:latest

# 克隆项目
git clone https://github.com/code-100-precent/LingEcho-App.git
cd LingEcho-App

# 复制环境配置
cp server/env.example .env

# 编辑 .env 文件并配置你的设置
# 至少需要设置：SESSION_SECRET, LLM_API_KEY

# 使用 Docker Compose 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f lingecho
```

**访问应用：**
- **前端界面**: http://localhost:7072
- **后端API**: http://localhost:7072/api
- **API文档**: http://localhost:7072/api/docs

**可选服务：**
```bash
# 启动 PostgreSQL 数据库
docker-compose --profile postgres up -d

# 启动 Redis 缓存
docker-compose --profile redis up -d

# 启动 Nginx 反向代理
docker-compose --profile nginx up -d

# 启动前端开发服务器
docker-compose --profile dev up -d
```

#### 方法二：手动安装

```bash
# 克隆项目
git clone https://github.com/code-100-precent/LingEcho-App.git
cd LingEcho-App

# 后端设置
cd server
go mod tidy
cp env.example .env
# 编辑 .env 文件配置你的设置

# 前端设置
cd ../web
npm install  # 或 pnpm install
npm run build  # 生产环境
# 或
npm run dev    # 开发环境（运行在端口 5173）

# 启动后端（在 server 目录）
cd ../server
go run ./cmd/server/main.go -mode=dev
```

**访问应用：**
- **前端界面**: http://localhost:5173 (开发) 或 http://localhost:7072 (生产)
- **后端API**: http://localhost:7072/api
- **API文档**: http://localhost:7072/api/docs

**可选服务（如需要）：**
```bash
# 启动 VAD 服务
cd services/vad-service
docker-compose up -d
# 或手动启动: python vad_service.py

# 启动声纹识别服务
cd services/voiceprint-api
docker-compose up -d
# 或手动启动: python -m app.main

# 启动 ASR-TTS 服务
cd services/asr-tts-service
docker-compose up -d
# 或手动启动: python -m app.main

# 启动 MCP 服务（可选）
cd server
go run ./cmd/mcp/main.go --transport sse --port 3001
```

详细的安装说明请查看 [安装指南](docs/installation_CN.md)。

---

## 📚 文档

- **[安装指南](docs/installation_CN.md)** - 详细的安装和配置说明
- **[功能文档](docs/features_CN.md)** - 完整的功能列表，包含截图和示例
- **[架构文档](docs/architecture_CN.md)** - 系统架构和设计说明
- **[开发指南](docs/development_CN.md)** - 开发环境设置和贡献指南
- **[服务文档](docs/services_CN.md)** - 详细的服务组件说明

---

## 🤝 贡献指南

我们欢迎所有形式的贡献！请查看我们的 [开发指南](docs/development_CN.md) 了解详情。

### 快速贡献步骤

1. **Fork项目** - 点击右上角的Fork按钮
2. **创建分支** - `git checkout -b feature/your-feature`
3. **提交更改** - `git commit -m 'Add some feature'`
4. **推送分支** - `git push origin feature/your-feature`
5. **创建PR** - 在GitHub上创建Pull Request

---

## 👥 我们的团队

由两位全栈工程师组成的核心团队，专注于AI语音技术的创新与应用。

| 成员 | 角色 | 职责 |
|------|------|------|
| **chenting** | 全栈工程师 + 项目经理 | 负责项目整体架构设计和全栈开发，主导产品方向和技术选型 |
| **wangyueran** | 全栈工程师 | 负责前端界面开发和用户体验优化，确保产品易用性 |

## 📧 联系我们

- **邮箱**: 19511899044@163.com

---

## ⭐ Star 历史

[![Star History Chart](https://api.star-history.com/svg?repos=LingByte/SoulNexus&type=Date)](https://star-history.com/#your-username/LingEcho&Date)

---
