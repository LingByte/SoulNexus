# 安装指南

## 环境要求

- **Go** >= 1.25.1
- **Node.js** >= 18.0.0
- **npm** >= 8.0.0
- **Git**
- **Python** >= 3.10 (可选服务需要)

## 安装步骤

### 1. 克隆项目

```bash
git clone https://github.com/your-username/LingEcho.git
cd LingEcho
```

### 2. 后端配置

```bash
# 进入项目目录
cd LingEcho

# 安装Go依赖
go mod tidy

# 配置环境变量
cp env.example .env.dev
# 编辑 .env 文件，配置数据库和API密钥
```

### 3. 前端配置

```bash
# 进入前端目录
cd ui

# 安装依赖
npm install

# 启动开发服务器
npm run dev
```

### 4. 启动后端服务

```bash
# 返回项目根目录
cd ..

# 启动后端服务
go run ./cmd/server/. -mode=dev
```

### 5. 启动可选服务（VAD和声纹识别）

**VAD服务**（可选）：
```bash
cd services/vad-service
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
python vad_service.py
# 服务将在 http://localhost:7073 启动
```

**声纹识别服务**（可选）：
```bash
cd services/voiceprint-api
python3.10 -m venv venv  # 推荐使用Python 3.10
source venv/bin/activate
pip install --upgrade pip setuptools wheel
pip install torch torchaudio --index-url https://download.pytorch.org/whl/cpu
pip install -r requirements.txt
python -m app.main
# 服务将在 http://localhost:7074 启动
```

### 6. 访问应用

- **前端界面**: http://localhost:3000
- **后端API**: http://localhost:7072
- **API文档**: http://localhost:7072/api/docs
- **VAD服务**: http://localhost:7073 (如果启动)
- **声纹识别服务**: http://localhost:7074/voiceprint/docs (如果启动)

## 生产环境配置

```bash
# 使用生产模式启动
go run ./cmd/server/. -mode=production

# 或使用systemd服务
sudo systemctl start lingecho
sudo systemctl enable lingecho
```

## 配置说明

详细的环境变量配置请参考 [`env.example`](../server/env.example) 文件。

配置文件包含以下设置：
- 基础配置（端口、环境）
- 数据库配置（SQLite、PostgreSQL、MySQL）
- API配置
- LLM配置
- ASR/TTS提供商配置（七牛、腾讯云等）
- 日志、邮件、搜索、备份、监控配置
- SIP配置
- 缓存配置（本地、Redis）
- 阿里云百炼（知识库服务）配置
- 工作流触发器配置
- 设备管理配置
- 告警系统配置
- 账单系统配置

