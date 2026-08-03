<p align="center">
  <a href="README.md">English</a> · <a href="README_zh.md">简体中文</a> · 繁體中文 · <a href="README_ja.md">日本語</a>
</p>

<p align="center">
  <img src="web/public/icon-lingyu.png" alt="SoulNexus" width="140" style="border-radius:24px;box-shadow:0 4px 24px rgba(0,0,0,0.12);">
</p>

<h1 align="center">SoulNexus</h1>

<p align="center">
  <strong>AI 語音對話平台</strong>
</p>

<p align="center">
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go"></a>
  <a href="https://react.dev"><img src="https://img.shields.io/badge/React-18-61DAFB?style=for-the-badge&logo=react&logoColor=black" alt="React"></a>
  <a href="https://www.typescriptlang.org"><img src="https://img.shields.io/badge/TypeScript-5.3-3178C6?style=for-the-badge&logo=typescript&logoColor=white" alt="TypeScript"></a>
  <a href="https://vitejs.dev"><img src="https://img.shields.io/badge/Vite-5-646CFF?style=for-the-badge&logo=vite&logoColor=white" alt="Vite"></a>
  <a href="https://www.mysql.com"><img src="https://img.shields.io/badge/PostgreSQL-16-4169E1?style=for-the-badge&logo=postgresql&logoColor=white" alt="PostgreSQL"></a>
  <a href="https://github.com/LingByte/SoulNexus/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-AGPL%203.0-red?style=for-the-badge" alt="License"></a>
</p>

<p align="center">
  <a href="#-功能特性">功能特性</a> ·
  <a href="#-快速開始">快速開始</a> ·
  <a href="#-系統架構">系統架構</a> ·
  <a href="#-技術棧">技術棧</a> ·
  <a href="#-開發指南">開發指南</a> ·
  <a href="#-部署方案">部署方案</a> ·
  <a href="#-文件">文件</a>
</p>

---

## ✨ 功能特性

<table>
  <tr>
    <td width="50%" valign="top">

#### 即時語音

- 瀏覽器 WebRTC / WebSocket 語音工作階段
- Embed 嵌入元件 + 桌寵用戶端
- 級聯對話引擎：`ASR → LLM → TTS`
- 可選即時多模態對話 Agent

    </td>
    <td width="50%" valign="top">

#### AI 語音助手

- 多供應商 ASR / TTS / LLM 整合
- 熱詞偵測與打斷支援
- 知識庫增強對話
- 聲音克隆與聲紋

    </td>
  </tr>
  <tr>
    <td width="50%" valign="top">

#### 平台能力

- 助手版本發布 / 回滾
- 視覺化工作流與外掛市場
- MCP 工具市場
- JS 模板（H5 / 小程序嵌入）

    </td>
    <td width="50%" valign="top">

#### 安全與多租戶

- JWT + AK/SK 認證
- 基於角色的存取控制 (RBAC)
- 租戶隔離與資料隔離
- GitHub OAuth 整合

    </td>
  </tr>
  <tr>
    <td width="50%" valign="top">

#### 知識庫

- 多向量庫：Qdrant、Milvus、PGVector、Elasticsearch、Weaviate
- 文件匯入：PDF、DOCX、TXT、Markdown、HTML
- 混合檢索 (關鍵字 + 向量 + RRF 融合)
- Rerank 支援 (Jina 等)

    </td>
    <td width="50%" valign="top">

#### 雲端儲存與資料庫

- 多儲存：本地、S3、OSS、COS、MinIO、TOS、OBS、KS3
- SQLite (開發) / PostgreSQL / MySQL (正式環境)
- Redis 快取 (可選)
- 郵件：SMTP 與 SendCloud

    </td>
  </tr>
</table>

---

## 🚀 快速開始

### 推薦：Docker 一鍵啟動

```bash
git clone https://github.com/LingByte/SoulNexus.git
cd SoulNexus
make deploy
```

瀏覽器開啟 **http://localhost:8080**

| 角色 | 電子郵件 | 密碼 |
|------|------|------|
| 平台管理員 | `admin@lingecho.com` | `admin123` |

可透過環境變數覆寫種子帳號（僅 `platform_admins` 為空時生效）：`PLATFORM_ADMIN_EMAIL` / `PLATFORM_ADMIN_PASSWORD` / `PLATFORM_ADMIN_DISPLAY_NAME`。

### 本地開發

| 依賴 | 版本 |
|------|------|
| Go | 1.26+ |
| Node.js | 18+ |

```bash
cp env.example .env
go run ./cmd/server -init -seed   # http://localhost:7072
cd web && npm ci && npm run dev   # http://localhost:3000
```

---

## 🏗️ 系統架構

### 整體架構

```mermaid
graph TB
    subgraph Client["用戶端層"]
        Web["Web 管控台<br/>React + TypeScript"]
        DesktopPet["桌寵 / Embed<br/>WebSocket · WebRTC"]
    end

    subgraph Gateway["閘道層"]
        HTTP["HTTP API<br/>:7072<br/>Gin 框架"]
        VoiceSession["Voice Session<br/>WS / WebRTC"]
        WS["WebSocket Hub<br/>即時推送"]
    end

    subgraph Core["核心引擎"]
        Dialog["對話引擎<br/>ASR → LLM → TTS"]
        Assistants["助手 / 知識庫 / 工作流"]
        MCP["MCP 工具"]
    end

    subgraph AI["AI / ML 層"]
        ASR["ASR 供應商<br/>Deepgram · Google · 騰訊雲<br/>百度 · 訊飛 · 本地"]
        TTS["TTS 供應商<br/>OpenAI · Azure · Google<br/>訊飛 · 火山引擎 · 本地"]
        LLM["LLM 供應商<br/>OpenAI · Deepseek<br/>Coze · 自訂"]
    end

    subgraph Data["資料層"]
        DB[("資料庫<br/>SQLite / PostgreSQL / MySQL")]
        Vector[("向量庫<br/>Qdrant · Milvus · PGVector")]
        Cache[("快取<br/>Local / Redis")]
        Store[("物件儲存<br/>S3 · OSS · MinIO")]
    end

    subgraph Monitor["可觀測性"]
        Prometheus["Prometheus<br/>指標監控"]
        Logging["結構化日誌<br/>Zap + Lumberjack"]
        Profiling["效能分析<br/>pprof · Pyroscope"]
    end

    Web --> HTTP
    DesktopPet --> VoiceSession
    VoiceSession --> Dialog
    HTTP --> Assistants
    HTTP --> Dialog
    HTTP --> MCP
    WS --> Web

    Dialog --> ASR
    Dialog --> LLM
    Dialog --> TTS

    Assistants --> DB
    Dialog --> DB
    Dialog --> Vector
    Dialog --> Store

    Dialog --> Prometheus
    Dialog --> Logging
    Dialog --> Profiling
```

### 語音對話流程

```mermaid
sequenceDiagram
    participant Client as Browser / Desktop Pet
    participant VS as Voice Session
    participant Dialog as 對話引擎
    participant ASR as ASR 供應商
    participant LLM as LLM 供應商
    participant TTS as TTS 供應商

    Client->>VS: WebSocket / WebRTC Offer
    VS->>Dialog: 綁定助手工作階段

    loop 語音對話迴圈
        Client->>VS: 上行 PCM / Opus
        VS->>Dialog: 音訊幀
        Dialog->>ASR: 語音轉文字
        ASR->>Dialog: 文字結果
        Dialog->>LLM: 產生回覆
        LLM->>Dialog: 回覆文字
        Dialog->>TTS: 合成語音
        TTS->>Dialog: 音訊資料
        Dialog->>VS: 下行 PCM
        VS->>Client: 播放音訊
    end

    Client->>VS: 結束工作階段
    VS->>Dialog: Detach
```

### 知識庫資料流

```mermaid
flowchart LR
    subgraph Ingest["知識庫匯入"]
        Doc["文件<br/>PDF·DOCX·TXT·MD"]
        Parse["解析器<br/>lingllm/parser"]
        Chunk["分塊器<br/>規則 / LLM"]
        Embed["向量化<br/>OpenAI · Gitee AI"]
    end

    subgraph Retrieval["知識庫檢索"]
        Query["使用者查詢"]
        Hybrid["混合檢索<br/>關鍵字 + 向量"]
        RRF["RRF 融合"]
        Rerank["重排序<br/>Jina"]
        Result["Top-K 結果"]
    end

    Doc --> Parse --> Chunk --> Embed --> VectorDB[("向量庫")]
    Query --> Hybrid
    VectorDB --> Hybrid
    Hybrid --> RRF --> Rerank --> Result
    Result --> LLM["LLM 產生回覆"]
```

### 多租戶資料隔離

```mermaid
graph TB
    subgraph Platform["平台層"]
        PA["平台管理員"]
        Router["路由閘道"]
    end

    subgraph Tenant_A["租戶 A"]
        UA["成員"]
        DA["資料隔離<br/>DB Schema / Row"]
        CA["設定隔離"]
    end

    subgraph Tenant_B["租戶 B"]
        UB["成員"]
        DB["資料隔離<br/>DB Schema / Row"]
        CB["設定隔離"]
    end

    PA --> Router
    Router --> Tenant_A
    Router --> Tenant_B
    UA --> DA
    UA --> CA
    UB --> DB
    UB --> CB
```

---

## 🛠️ 技術棧

<table>
  <tr>
    <th>層級</th>
    <th>技術</th>
  </tr>
  <tr>
    <td><strong>後端</strong></td>
    <td>
      <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go">
      <img src="https://img.shields.io/badge/Gin-1.10-00ADD8?style=flat-square" alt="Gin">
      <img src="https://img.shields.io/badge/GORM-2.0-00ADD8?style=flat-square" alt="GORM">
      <img src="https://img.shields.io/badge/WebSocket-gorilla-00ADD8?style=flat-square" alt="WebSocket">
    </td>
  </tr>
  <tr>
    <td><strong>前端</strong></td>
    <td>
      <img src="https://img.shields.io/badge/React-18-61DAFB?style=flat-square&logo=react&logoColor=black" alt="React">
      <img src="https://img.shields.io/badge/TypeScript-5.3-3178C6?style=flat-square&logo=typescript&logoColor=white" alt="TypeScript">
      <img src="https://img.shields.io/badge/Vite-5-646CFF?style=flat-square&logo=vite&logoColor=white" alt="Vite">
      <img src="https://img.shields.io/badge/Arco_Design-2.x-165DFF?style=flat-square" alt="Arco Design">
      <img src="https://img.shields.io/badge/Tailwind_CSS-3-06B6D4?style=flat-square&logo=tailwindcss&logoColor=white" alt="Tailwind CSS">
      <img src="https://img.shields.io/badge/Zustand-4-FF6B35?style=flat-square" alt="Zustand">
    </td>
  </tr>
  <tr>
    <td><strong>即時語音</strong></td>
    <td>
      <img src="https://img.shields.io/badge/WebRTC-瀏覽器-FF6633?style=flat-square" alt="WebRTC">
      <img src="https://img.shields.io/badge/WebSocket-PCM-FF6633?style=flat-square" alt="WebSocket">
      <img src="https://img.shields.io/badge/Desktop_Pet-Embed-FF6633?style=flat-square" alt="Desktop Pet">
    </td>
  </tr>
  <tr>
    <td><strong>AI / ML</strong></td>
    <td>
      <img src="https://img.shields.io/badge/ASR-多供應商-8B5CF6?style=flat-square" alt="ASR">
      <img src="https://img.shields.io/badge/TTS-多供應商-8B5CF6?style=flat-square" alt="TTS">
      <img src="https://img.shields.io/badge/LLM-OpenAI_/_Deepseek-8B5CF6?style=flat-square" alt="LLM">
      <img src="https://img.shields.io/badge/Embedding-Qwen3--8B-8B5CF6?style=flat-square" alt="Embedding">
    </td>
  </tr>
  <tr>
    <td><strong>資料庫</strong></td>
    <td>
      <img src="https://img.shields.io/badge/SQLite-3-003B57?style=flat-square&logo=sqlite&logoColor=white" alt="SQLite">
      <img src="https://img.shields.io/badge/PostgreSQL-16-4169E1?style=flat-square&logo=postgresql&logoColor=white" alt="PostgreSQL">
      <img src="https://img.shields.io/badge/MySQL-8-4479A1?style=flat-square&logo=mysql&logoColor=white" alt="MySQL">
      <img src="https://img.shields.io/badge/Neo4j-可選-018BFF?style=flat-square&logo=neo4j&logoColor=white" alt="Neo4j">
    </td>
  </tr>
  <tr>
    <td><strong>向量庫</strong></td>
    <td>
      <img src="https://img.shields.io/badge/Qdrant-預設-DC382C?style=flat-square" alt="Qdrant">
      <img src="https://img.shields.io/badge/Milvus-可選-00A1E0?style=flat-square" alt="Milvus">
      <img src="https://img.shields.io/badge/PGVector-可選-4169E1?style=flat-square" alt="PGVector">
      <img src="https://img.shields.io/badge/Elasticsearch-可選-005571?style=flat-square" alt="Elasticsearch">
      <img src="https://img.shields.io/badge/Weaviate-可選-BD00FF?style=flat-square" alt="Weaviate">
    </td>
  </tr>
  <tr>
    <td><strong>儲存</strong></td>
    <td>
      <img src="https://img.shields.io/badge/AWS_S3-可選-FF9900?style=flat-square&logo=amazonwebservices&logoColor=white" alt="S3">
      <img src="https://img.shields.io/badge/阿里雲_OSS-可選-FF6A00?style=flat-square" alt="OSS">
      <img src="https://img.shields.io/badge/MinIO-可選-FF6A00?style=flat-square" alt="MinIO">
      <img src="https://img.shields.io/badge/騰訊雲_COS-可選-006EFF?style=flat-square" alt="COS">
    </td>
  </tr>
  <tr>
    <td><strong>監控</strong></td>
    <td>
      <img src="https://img.shields.io/badge/Prometheus-E6522C?style=flat-square&logo=prometheus&logoColor=white" alt="Prometheus">
      <img src="https://img.shields.io/badge/WebSocket-即時-00ADD8?style=flat-square" alt="WebSocket">
      <img src="https://img.shields.io/badge/pprof-效能分析-00ADD8?style=flat-square" alt="pprof">
      <img src="https://img.shields.io/badge/Pyroscope-可選-7B3BF5?style=flat-square" alt="Pyroscope">
    </td>
  </tr>
  <tr>
    <td><strong>基礎設施</strong></td>
    <td>
      <img src="https://img.shields.io/badge/Docker-24.0-2496ED?style=flat-square&logo=docker&logoColor=white" alt="Docker">
      <img src="https://img.shields.io/badge/Nginx-反向代理-009639?style=flat-square&logo=nginx&logoColor=white" alt="Nginx">
      <img src="https://img.shields.io/badge/Coturn-TURN/STUN-EA4335?style=flat-square" alt="Coturn">
    </td>
  </tr>
</table>

---

## 📁 專案結構

```
SoulNexus/
├── cmd/
│   ├── server/                 # 應用程式入口
│   ├── bootstrap/              # 資料庫初始化、遷移、種子資料
│   └── backfill/               # 資料回填工具
├── internal/
│   ├── config/                 # 環境設定
│   ├── handlers/               # HTTP API 處理器
│   ├── models/                 # GORM 資料庫模型
│   ├── listeners/              # 事件監聽器
│   ├── tasks/                  # 背景任務
│   └── workflow/               # 工作流定義
├── pkg/
│   ├── dialog/                 # 語音對話引擎（含 voice-session）
│   ├── voice/                  # 語音處理 ASR/TTS
│   ├── vad/                    # 語音活動偵測
│   ├── knowledge/              # 知識庫服務
│   ├── billing/                # 計費
│   ├── notification/           # 通知系統
│   ├── middleware/             # HTTP 中介軟體
│   ├── i18n/                   # 國際化
│   └── stores/                 # 物件儲存適配器
├── lingllm/                    # LLM / RAG / 即時語音底座
├── lingmcp/                    # MCP 相關模組
├── voiceprint/                 # 聲紋服務
├── desktop-pet/                # 桌寵用戶端
├── web/
│   └── src/
│       ├── pages/              # 控制台頁面
│       ├── api/                # API 用戶端模組
│       ├── stores/             # Zustand 狀態管理
│       ├── components/         # 共用 React 元件
│       ├── i18n/               # 國際化翻譯
│       └── utils/              # 工具函式
├── deploy/                     # Docker / Helm / Nginx
├── docs/                       # 文件
├── scripts/                    # 建置與部署腳本
├── nginx/                      # Nginx 設定
├── Dockerfile                  # Docker 映像
├── docker-compose.yml          # Docker Compose
├── Makefile                    # 建置命令
└── env.example                 # 環境變數範本
```

---

## 💻 開發指南

### 後端命令

```bash
# 開發模式啟動
go run ./cmd/server

# 資料庫遷移 + 匯入演示資料
go run ./cmd/server -init -seed

# 執行所有測試
go test ./... -cover

# 執行指定套件的測試
go test ./pkg/dialog/... -v
```

### 前端命令

```bash
cd web

# 安裝依賴
npm install

# 開發伺服器
npm run dev

# 正式環境建置
npm run build

# 程式碼檢查與型別檢查
npm run lint
npm run type-check
```

### Docker

```bash
# 首次：產生 .env
make env

# 建置並啟動所有服務
make deploy

# 查看日誌
make logs

# 停止服務
make down
```

---

## 🐳 部署方案

### Docker Compose 一鍵部署

```bash
make deploy
# 控制台 http://localhost:8080
# make logs / make clean / make deploy-seed
```

### 正式環境檢查清單

- [ ] 設定 `GIN_MODE=release`
- [ ] 設定 `SESSION_SECRET` (32+ 位元組隨機字串)
- [ ] 設定 `CORS_ALLOWED_ORIGINS` 為您的網域
- [ ] 設定 SSL/TLS 憑證
- [ ] 使用 PostgreSQL 替代 SQLite
- [ ] 設定 Redis 用於多實例快取
- [ ] 關閉 `UPLOADS_RECORDINGS_PUBLIC`
- [ ] 設定向量庫 (推薦 Qdrant)

---

## 📚 文件

| 文件 | 說明 |
|------|------|
| [功能匯總](docs/features-overview.md) | 目前功能清單與成熟度 |
| [部署指南](docs/deployment.md) | Docker 一鍵部署 |
| [知識庫營運](docs/knowledge-ops-closed-loop-zh.md) | 知識庫工作流 |
| [MCP 市場](docs/mcp-market.md) | 租戶 MCP 開通與綁定 |
| [環境變數設定](env.example) | 設定項說明 |

---

## 🤝 貢獻指南

```bash
# 1. Fork 儲存庫
# 2. 建立功能分支
git checkout -b feature/amazing-feature

# 3. 提交變更
git commit -m 'feat: add amazing feature'

# 4. 推送到分支
git push origin feature/amazing-feature

# 5. 建立 Pull Request
```

---

## 📄 授權條款

本專案基於 **GNU Affero 通用公共授權條款 v3.0** — 詳見 [LICENSE](LICENSE) 文件。

---

<p align="center">
  由 <a href="https://github.com/LingByte">LingByte</a> 傾心打造 ❤️
</p>
