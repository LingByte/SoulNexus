# 服务文档

## 核心服务

### 主服务

**位置**: `cmd/server/`

**说明**: 核心后端服务，提供RESTful API和WebSocket支持

**端口**: 7072

**功能**:
- RESTful API端点
- WebSocket连接
- 数据库管理
- 身份验证和授权
- 文件上传和存储

### 语音/SIP服务

**说明**: 已集成到主服务中，提供SIP软电话和语音处理功能

**功能**:
- SIP协议实现
- 实时音频处理
- 通话管理
- 通话录音
- ACD（自动呼叫分配）

### MCP服务

**位置**: `cmd/mcp/`

**说明**: Model Context Protocol服务

## 独立服务

### VAD服务

**位置**: `services/vad-service/`

**端口**: 7073

**技术**: Python + FastAPI + SileroVAD

**功能**:
- 语音活动检测
- 支持PCM和OPUS格式
- HTTP RESTful API
- 实时语音活动检测
- 静默检测和会话管理
- 双阈值机制，滑动窗口平滑处理

**快速启动**：
```bash
cd services/vad-service
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
python vad_service.py
```

**API端点**:
- `POST /vad/detect` - 检测语音活动
- `POST /vad/session/create` - 创建新会话
- `POST /vad/session/update` - 更新会话状态
- `DELETE /vad/session/{session_id}` - 删除会话

### 声纹识别服务

**位置**: `services/voiceprint-api/`

**端口**: 7074

**技术**: Python + FastAPI + ModelScope

**功能**:
- 说话人识别
- 声纹注册和管理
- 多说话人识别
- 相似度计算
- MySQL数据库存储
- RESTful API接口

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

**API端点**:
- `POST /voiceprint/register` - 注册声纹
- `POST /voiceprint/identify` - 识别说话人
- `GET /voiceprint/list` - 列出所有声纹
- `DELETE /voiceprint/{voiceprint_id}` - 删除声纹

**API文档**: http://localhost:7074/voiceprint/docs

