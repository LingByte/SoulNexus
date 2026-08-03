<p align="center">
  <a href="README.md">English</a> · <a href="README_zh.md">简体中文</a> · <a href="README_zh-TW.md">繁體中文</a> · 日本語
</p>

<p align="center">
  <img src="web/public/icon-lingyu.png" alt="SoulNexus" width="140" style="border-radius:24px;box-shadow:0 4px 24px rgba(0,0,0,0.12);">
</p>

<h1 align="center">SoulNexus</h1>

<p align="center">
  <strong>AI 音声対話プラットフォーム</strong>
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
  <a href="#-機能">機能</a> ·
  <a href="#-クイックスタート">クイックスタート</a> ·
  <a href="#-システムアーキテクチャ">システムアーキテクチャ</a> ·
  <a href="#-技術スタック">技術スタック</a> ·
  <a href="#-開発ガイド">開発ガイド</a> ·
  <a href="#-デプロイ">デプロイ</a> ·
  <a href="#-ドキュメント">ドキュメント</a>
</p>

---

## ✨ 機能

<table>
  <tr>
    <td width="50%" valign="top">

#### リアルタイム音声

- ブラウザ WebRTC / WebSocket 音声セッション
- Embed 埋め込みコンポーネント + デスクトップペットクライアント
- カスケード対話エンジン：`ASR → LLM → TTS`
- オプションのリアルタイムマルチモーダル対話 Agent

    </td>
    <td width="50%" valign="top">

#### AI 音声アシスタント

- 複数ベンダー ASR / TTS / LLM 統合
- ホットワード検出と割り込み（バージイン）対応
- ナレッジベース強化対話
- ボイスクローンと声紋

    </td>
  </tr>
  <tr>
    <td width="50%" valign="top">

#### プラットフォーム機能

- アシスタントバージョンの公開 / ロールバック
- ビジュアルワークフローとプラグインマーケット
- MCP ツールマーケット
- JS テンプレート（H5 / ミニプログラム埋め込み）

    </td>
    <td width="50%" valign="top">

#### セキュリティとマルチテナント

- JWT + AK/SK 認証
- ロールベースアクセス制御 (RBAC)
- テナント分離とデータ分離
- GitHub OAuth 統合

    </td>
  </tr>
  <tr>
    <td width="50%" valign="top">

#### ナレッジベース

- 複数ベクトル DB：Qdrant、Milvus、PGVector、Elasticsearch、Weaviate
- ドキュメント取り込み：PDF、DOCX、TXT、Markdown、HTML
- ハイブリッド検索（キーワード + ベクトル + RRF 融合）
- Rerank 対応（Jina など）

    </td>
    <td width="50%" valign="top">

#### クラウドストレージとデータベース

- 複数ストレージ：ローカル、S3、OSS、COS、MinIO、TOS、OBS、KS3
- SQLite（開発）/ PostgreSQL / MySQL（本番）
- Redis キャッシュ（オプション）
- メール：SMTP と SendCloud

    </td>
  </tr>
</table>

---

## 🚀 クイックスタート

### 推奨：Docker ワンクリック起動

```bash
git clone https://github.com/LingByte/SoulNexus.git
cd SoulNexus
make deploy
```

ブラウザで **http://localhost:8080** を開く

| ロール | メール | パスワード |
|------|------|------|
| プラットフォーム管理者 | `admin@lingecho.com` | `admin123` |

環境変数でシードアカウントを上書き可能（`platform_admins` が空の場合のみ）：`PLATFORM_ADMIN_EMAIL` / `PLATFORM_ADMIN_PASSWORD` / `PLATFORM_ADMIN_DISPLAY_NAME`。

### ローカル開発

| 依存関係 | バージョン |
|------|------|
| Go | 1.26+ |
| Node.js | 18+ |

```bash
cp env.example .env
go run ./cmd/server -init -seed   # http://localhost:7072
cd web && npm ci && npm run dev   # http://localhost:3000
```

---

## 🏗️ システムアーキテクチャ

### 全体アーキテクチャ

```mermaid
graph TB
    subgraph Client["クライアント層"]
        Web["Web 管理コンソール<br/>React + TypeScript"]
        DesktopPet["デスクトップペット / Embed<br/>WebSocket · WebRTC"]
    end

    subgraph Gateway["ゲートウェイ層"]
        HTTP["HTTP API<br/>:7072<br/>Gin フレームワーク"]
        VoiceSession["Voice Session<br/>WS / WebRTC"]
        WS["WebSocket Hub<br/>リアルタイムプッシュ"]
    end

    subgraph Core["コアエンジン"]
        Dialog["対話エンジン<br/>ASR → LLM → TTS"]
        Assistants["アシスタント / ナレッジベース / ワークフロー"]
        MCP["MCP ツール"]
    end

    subgraph AI["AI / ML 層"]
        ASR["ASR ベンダー<br/>Deepgram · Google · テンセントクラウド<br/>Baidu · iFlytek · ローカル"]
        TTS["TTS ベンダー<br/>OpenAI · Azure · Google<br/>iFlytek · Volcano Engine · ローカル"]
        LLM["LLM ベンダー<br/>OpenAI · Deepseek<br/>Coze · カスタム"]
    end

    subgraph Data["データ層"]
        DB[("データベース<br/>SQLite / PostgreSQL / MySQL")]
        Vector[("ベクトル DB<br/>Qdrant · Milvus · PGVector")]
        Cache[("キャッシュ<br/>Local / Redis")]
        Store[("オブジェクトストレージ<br/>S3 · OSS · MinIO")]
    end

    subgraph Monitor["可観測性"]
        Prometheus["Prometheus<br/>メトリクス監視"]
        Logging["構造化ログ<br/>Zap + Lumberjack"]
        Profiling["パフォーマンス分析<br/>pprof · Pyroscope"]
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

### 音声対話フロー

```mermaid
sequenceDiagram
    participant Client as Browser / Desktop Pet
    participant VS as Voice Session
    participant Dialog as 対話エンジン
    participant ASR as ASR ベンダー
    participant LLM as LLM ベンダー
    participant TTS as TTS ベンダー

    Client->>VS: WebSocket / WebRTC Offer
    VS->>Dialog: アシスタントセッションをバインド

    loop 音声対話ループ
        Client->>VS: 上行 PCM / Opus
        VS->>Dialog: 音声フレーム
        Dialog->>ASR: 音声をテキストに変換
        ASR->>Dialog: テキスト結果
        Dialog->>LLM: 応答を生成
        LLM->>Dialog: 応答テキスト
        Dialog->>TTS: 音声を合成
        TTS->>Dialog: 音声データ
        Dialog->>VS: 下行 PCM
        VS->>Client: 音声を再生
    end

    Client->>VS: セッション終了
    VS->>Dialog: Detach
```

### ナレッジベースデータフロー

```mermaid
flowchart LR
    subgraph Ingest["ナレッジベース取り込み"]
        Doc["ドキュメント<br/>PDF·DOCX·TXT·MD"]
        Parse["パーサー<br/>lingllm/parser"]
        Chunk["チャンク分割<br/>ルール / LLM"]
        Embed["ベクトル化<br/>OpenAI · Gitee AI"]
    end

    subgraph Retrieval["ナレッジベース検索"]
        Query["ユーザークエリ"]
        Hybrid["ハイブリッド検索<br/>キーワード + ベクトル"]
        RRF["RRF 融合"]
        Rerank["再ランキング<br/>Jina"]
        Result["Top-K 結果"]
    end

    Doc --> Parse --> Chunk --> Embed --> VectorDB[("ベクトル DB")]
    Query --> Hybrid
    VectorDB --> Hybrid
    Hybrid --> RRF --> Rerank --> Result
    Result --> LLM["LLM 応答生成"]
```

### マルチテナントデータ分離

```mermaid
graph TB
    subgraph Platform["プラットフォーム層"]
        PA["プラットフォーム管理者"]
        Router["ルーティングゲートウェイ"]
    end

    subgraph Tenant_A["テナント A"]
        UA["メンバー"]
        DA["データ分離<br/>DB Schema / Row"]
        CA["設定分離"]
    end

    subgraph Tenant_B["テナント B"]
        UB["メンバー"]
        DB["データ分離<br/>DB Schema / Row"]
        CB["設定分離"]
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

## 🛠️ 技術スタック

<table>
  <tr>
    <th>レイヤー</th>
    <th>技術</th>
  </tr>
  <tr>
    <td><strong>バックエンド</strong></td>
    <td>
      <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go">
      <img src="https://img.shields.io/badge/Gin-1.10-00ADD8?style=flat-square" alt="Gin">
      <img src="https://img.shields.io/badge/GORM-2.0-00ADD8?style=flat-square" alt="GORM">
      <img src="https://img.shields.io/badge/WebSocket-gorilla-00ADD8?style=flat-square" alt="WebSocket">
    </td>
  </tr>
  <tr>
    <td><strong>フロントエンド</strong></td>
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
    <td><strong>リアルタイム音声</strong></td>
    <td>
      <img src="https://img.shields.io/badge/WebRTC-ブラウザ-FF6633?style=flat-square" alt="WebRTC">
      <img src="https://img.shields.io/badge/WebSocket-PCM-FF6633?style=flat-square" alt="WebSocket">
      <img src="https://img.shields.io/badge/Desktop_Pet-Embed-FF6633?style=flat-square" alt="Desktop Pet">
    </td>
  </tr>
  <tr>
    <td><strong>AI / ML</strong></td>
    <td>
      <img src="https://img.shields.io/badge/ASR-複数ベンダー-8B5CF6?style=flat-square" alt="ASR">
      <img src="https://img.shields.io/badge/TTS-複数ベンダー-8B5CF6?style=flat-square" alt="TTS">
      <img src="https://img.shields.io/badge/LLM-OpenAI_/_Deepseek-8B5CF6?style=flat-square" alt="LLM">
      <img src="https://img.shields.io/badge/Embedding-Qwen3--8B-8B5CF6?style=flat-square" alt="Embedding">
    </td>
  </tr>
  <tr>
    <td><strong>データベース</strong></td>
    <td>
      <img src="https://img.shields.io/badge/SQLite-3-003B57?style=flat-square&logo=sqlite&logoColor=white" alt="SQLite">
      <img src="https://img.shields.io/badge/PostgreSQL-16-4169E1?style=flat-square&logo=postgresql&logoColor=white" alt="PostgreSQL">
      <img src="https://img.shields.io/badge/MySQL-8-4479A1?style=flat-square&logo=mysql&logoColor=white" alt="MySQL">
      <img src="https://img.shields.io/badge/Neo4j-オプション-018BFF?style=flat-square&logo=neo4j&logoColor=white" alt="Neo4j">
    </td>
  </tr>
  <tr>
    <td><strong>ベクトル DB</strong></td>
    <td>
      <img src="https://img.shields.io/badge/Qdrant-デフォルト-DC382C?style=flat-square" alt="Qdrant">
      <img src="https://img.shields.io/badge/Milvus-オプション-00A1E0?style=flat-square" alt="Milvus">
      <img src="https://img.shields.io/badge/PGVector-オプション-4169E1?style=flat-square" alt="PGVector">
      <img src="https://img.shields.io/badge/Elasticsearch-オプション-005571?style=flat-square" alt="Elasticsearch">
      <img src="https://img.shields.io/badge/Weaviate-オプション-BD00FF?style=flat-square" alt="Weaviate">
    </td>
  </tr>
  <tr>
    <td><strong>ストレージ</strong></td>
    <td>
      <img src="https://img.shields.io/badge/AWS_S3-オプション-FF9900?style=flat-square&logo=amazonwebservices&logoColor=white" alt="S3">
      <img src="https://img.shields.io/badge/Alibaba_Cloud_OSS-オプション-FF6A00?style=flat-square" alt="OSS">
      <img src="https://img.shields.io/badge/MinIO-オプション-FF6A00?style=flat-square" alt="MinIO">
      <img src="https://img.shields.io/badge/Tencent_Cloud_COS-オプション-006EFF?style=flat-square" alt="COS">
    </td>
  </tr>
  <tr>
    <td><strong>監視</strong></td>
    <td>
      <img src="https://img.shields.io/badge/Prometheus-E6522C?style=flat-square&logo=prometheus&logoColor=white" alt="Prometheus">
      <img src="https://img.shields.io/badge/WebSocket-リアルタイム-00ADD8?style=flat-square" alt="WebSocket">
      <img src="https://img.shields.io/badge/pprof-パフォーマンス分析-00ADD8?style=flat-square" alt="pprof">
      <img src="https://img.shields.io/badge/Pyroscope-オプション-7B3BF5?style=flat-square" alt="Pyroscope">
    </td>
  </tr>
  <tr>
    <td><strong>インフラ</strong></td>
    <td>
      <img src="https://img.shields.io/badge/Docker-24.0-2496ED?style=flat-square&logo=docker&logoColor=white" alt="Docker">
      <img src="https://img.shields.io/badge/Nginx-リバースプロキシ-009639?style=flat-square&logo=nginx&logoColor=white" alt="Nginx">
      <img src="https://img.shields.io/badge/Coturn-TURN/STUN-EA4335?style=flat-square" alt="Coturn">
    </td>
  </tr>
</table>

---

## 📁 プロジェクト構成

```
SoulNexus/
├── cmd/
│   ├── server/                 # アプリケーションエントリ
│   ├── bootstrap/              # DB 初期化、マイグレーション、シードデータ
│   └── backfill/               # データバックフィルツール
├── internal/
│   ├── config/                 # 環境設定
│   ├── handlers/               # HTTP API ハンドラー
│   ├── models/                 # GORM データベースモデル
│   ├── listeners/              # イベントリスナー
│   ├── tasks/                  # バックグラウンドタスク
│   └── workflow/               # ワークフロー定義
├── pkg/
│   ├── dialog/                 # 音声対話エンジン（voice-session 含む）
│   ├── voice/                  # 音声処理 ASR/TTS
│   ├── vad/                    # 音声区間検出 (VAD)
│   ├── knowledge/              # ナレッジベースサービス
│   ├── billing/                # 課金
│   ├── notification/           # 通知システム
│   ├── middleware/             # HTTP ミドルウェア
│   ├── i18n/                   # 国際化
│   └── stores/                 # オブジェクトストレージアダプター
├── lingllm/                    # LLM / RAG / リアルタイム音声基盤
├── lingmcp/                    # MCP 関連モジュール
├── voiceprint/                 # 声紋サービス
├── desktop-pet/                # デスクトップペットクライアント
├── web/
│   └── src/
│       ├── pages/              # コンソールページ
│       ├── api/                # API クライアントモジュール
│       ├── stores/             # Zustand 状態管理
│       ├── components/         # 共有 React コンポーネント
│       ├── i18n/               # 国際化翻訳
│       └── utils/              # ユーティリティ関数
├── deploy/                     # Docker / Helm / Nginx
├── docs/                       # ドキュメント
├── scripts/                    # ビルドとデプロイスクリプト
├── nginx/                      # Nginx 設定
├── Dockerfile                  # Docker イメージ
├── docker-compose.yml          # Docker Compose
├── Makefile                    # ビルドコマンド
└── env.example                 # 環境変数テンプレート
```

---

## 💻 開発ガイド

### バックエンドコマンド

```bash
# 開発モードで起動
go run ./cmd/server

# データベースマイグレーション + デモデータ取り込み
go run ./cmd/server -init -seed

# すべてのテストを実行
go test ./... -cover

# 指定パッケージのテストを実行
go test ./pkg/dialog/... -v
```

### フロントエンドコマンド

```bash
cd web

# 依存関係をインストール
npm install

# 開発サーバー
npm run dev

# 本番ビルド
npm run build

# リントと型チェック
npm run lint
npm run type-check
```

### Docker

```bash
# 初回：.env を生成
make env

# すべてのサービスをビルドして起動
make deploy

# ログを表示
make logs

# サービスを停止
make down
```

---

## 🐳 デプロイ

### Docker Compose ワンクリックデプロイ

```bash
make deploy
# コンソール http://localhost:8080
# make logs / make clean / make deploy-seed
```

### 本番環境チェックリスト

- [ ] `GIN_MODE=release` を設定
- [ ] `SESSION_SECRET` を設定（32 バイト以上のランダム文字列）
- [ ] `CORS_ALLOWED_ORIGINS` を自分のドメインに設定
- [ ] SSL/TLS 証明書を設定
- [ ] SQLite の代わりに PostgreSQL を使用
- [ ] マルチインスタンスキャッシュ用に Redis を設定
- [ ] `UPLOADS_RECORDINGS_PUBLIC` を無効化
- [ ] ベクトル DB を設定（Qdrant 推奨）

---

## 📚 ドキュメント

| ドキュメント | 説明 |
|------|------|
| [機能概要](docs/features-overview.md) | 現在の機能一覧と成熟度 |
| [デプロイガイド](docs/deployment.md) | Docker ワンクリックデプロイ |
| [ナレッジベース運用](docs/knowledge-ops-closed-loop-zh.md) | ナレッジベースワークフロー |
| [MCP マーケット](docs/mcp-market.md) | テナント MCP 有効化とバインド |
| [環境変数設定](env.example) | 設定項目の説明 |

---

## 🤝 コントリビューション

```bash
# 1. リポジトリを Fork
# 2. 機能ブランチを作成
git checkout -b feature/amazing-feature

# 3. 変更をコミット
git commit -m 'feat: add amazing feature'

# 4. ブランチにプッシュ
git push origin feature/amazing-feature

# 5. Pull Request を作成
```

---

## 📄 ライセンス

本プロジェクトは **GNU Affero General Public License v3.0** に基づいてライセンスされています — 詳細は [LICENSE](LICENSE) ファイルを参照してください。

---

<p align="center">
  <a href="https://github.com/LingByte">LingByte</a> が心を込めて制作 ❤️
</p>
