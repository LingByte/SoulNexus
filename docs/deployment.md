# SoulNexus 一键部署指南

本文说明如何使用 Docker 与 Makefile 在单机环境快速部署 SoulNexus（前端 + HTTP API）。

## 架构说明

容器内运行两个进程：

| 组件 | 端口 | 说明 |
|------|------|------|
| **Nginx** | `80`（映射到宿主机 `HTTP_PORT`，默认 `8080`） | 托管 React 静态资源，并将 `/api`、`/uploads` 等反代到后端 |
| **Go Server** | `7072`（仅容器内） | HTTP API、WebSocket、业务逻辑 |

数据持久化目录：`/data`（SQLite 数据库 `ling.db`、上传文件 `uploads/`）。

```
浏览器 ──► :8080 (Nginx)
              ├── /          → web/dist (SPA)
              ├── /api/*     → 127.0.0.1:7072
              └── /uploads/* → 127.0.0.1:7072

```

## 前置要求

- Docker 20.10+
- Docker Compose v2（`docker compose`）
- Make（macOS / Linux 自带；Windows 可使用 WSL 或 Git Bash）

### 原生编译依赖（CGO）

后端链接 C / Rust 音频库（降噪、编解码）。**默认 Docker 镜像**使用 `-tags lingcodec`，编解码走 vendored Rust（G.711 / G.722 / G.729 / Opus），构建阶段会安装 Rust 并编译 `lingllm/media/encoder/lecodec`。

本地不带 `lingcodec` tag 时仍可用纯 Go G.711/G.722 + `libopus`（hraban）：

| 平台 | 命令 |
|------|------|
| **Debian / Ubuntu** | `sudo apt-get install -y pkg-config gcc libc6-dev libopus-dev libopusfile-dev` |
| **macOS (Homebrew)** | `brew install pkg-config opus opusfile`；Rust 路径另需 `rustup` + `./lingllm/media/encoder/lecodec/build.sh` |

Rust 编解码：

```bash
./lingllm/media/encoder/lecodec/build.sh
CGO_ENABLED=1 go build -tags lingcodec ./cmd/server
```

### TLS / ACME（可选）

| 能力 | 环境变量 / 入口 |
|------|----------------|
| Let's Encrypt | `SSL_ACME_DOMAINS=a.example.com` + `SSL_ACME_CACHE_DIR` + 可选 `SSL_ACME_HTTP_ADDR=:80` |


## 快速开始（一键部署）

```bash
# 1. 克隆项目
git clone https://github.com/LingByte/SoulNexus.git
cd SoulNexus

# 2. 准备环境变量（首次会自动从 env.example 复制）
make env

# 3. 编辑 .env —— 生产环境必须设置 SESSION_SECRET（32+ 字节随机字符串）
#    以及 PLATFORM_ADMIN_PASSWORD（≥10，禁止 admin123）
#    示例：openssl rand -hex 32
#    单机交付约束见 docs/ops-single-node.md；分布式扩展清单见 docs/distributed.md
nano .env

# 4. 一键构建并启动
make deploy
```

启动成功后访问：

- **控制台**：http://localhost:8080（若未改 `HTTP_PORT`）
- **默认管理员**：通过 `PLATFORM_ADMIN_EMAIL` / `PLATFORM_ADMIN_PASSWORD` 注入（生产禁止 `admin123`，长度 ≥10）；首次登录后请立即修改

## Makefile 命令

| 命令 | 说明 |
|------|------|
| `make help` | 查看所有命令 |
| `make env` | 从 `env.example` 生成 `.env` |
| `make build` | 仅构建镜像 |
| `make up` | 启动容器 |
| `make down` | 停止容器 |
| `make deploy` | **构建 + 启动**（推荐） |
| `make deploy-seed` | 部署并写入演示种子数据（仅开发/演示） |
| `make logs` | 查看日志 |
| `make restart` | 重启 |
| `make ps` | 容器状态 |
| `make shell` | 进入容器 bash |
| `make clean` | 停止并**删除数据卷**（清空数据库） |

## 环境变量（Docker 常用）

在 `.env` 中配置，由 `docker-compose.yml` 注入容器：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `HTTP_PORT` | `8080` | 宿主机 Web 访问端口 |
| `SESSION_SECRET` | （必填） | 会话加密密钥，生产必须设置 |
| `LINGECHO_SEED` | `auto` | `auto`：SQLite 首次建库时自动 `-seed`；`true` 强制 seed；`false` 禁用 |
| `LINGECHO_STARTUP_TIMEOUT` | `600` | 等待后端就绪的最长秒数（远程 MySQL 首次 migration 可能较久） |
| `GOPROXY` | `https://goproxy.cn,direct` | Docker **构建时** Go 模块与 toolchain 下载代理 |
| `GOSUMDB` | `sum.golang.google.cn` | Docker 构建时 Go checksum 数据库 |
| `NPM_REGISTRY` | `https://registry.npmmirror.com` | Docker 构建时 npm 源 |
| `APT_MIRROR` | `https://mirrors.tuna.tsinghua.edu.cn/debian/` | Docker 构建时 Debian apt 源（可改为阿里等镜像） |

其余 AI、存储、知识库等配置与 `env.example` 一致，按需追加到 `.env` 即可。

### `.env` 如何进入容器

- **不会**打进镜像；通过 `env_file: .env` 注入，`ADDR`、`DB_DRIVER`、`DSN`、`MODE` 等均以你的 `.env` 为准。
- `docker-compose.yml` 的 `environment` 只补充容器专用项（`METRICS_ALLOWED_IPS`、启动超时等），**不再覆盖**数据库与 HTTP 端口。
- Nginx 反代端口随 `.env` 里的 `ADDR` 自动匹配（如 `ADDR=:9003` → `127.0.0.1:9003`）。
- 首次远程 MySQL migration 可能较慢，默认最多等待 600 秒；日志会周期性输出 `still waiting...`。
- 首次部署请先 `make env` 或 `cp env.example .env`。

首次启动 entrypoint 固定带 `-init`；SQLite 且库文件不存在、`LINGECHO_SEED=auto` 时会自动 `-seed`。


## 手动 Docker 命令

不使用 Makefile 时：

```bash
cp env.example .env
# 编辑 SESSION_SECRET 等

docker compose build
docker compose up -d
docker compose logs -f
```

## 数据备份

数据卷 `lingecho-data` 挂载在容器 `/data`：

```bash
# 备份
docker compose exec lingecho tar -czf - -C /data . > lingecho-backup-$(date +%F).tar.gz

# 恢复（需先 stop）
docker compose down
docker run --rm -v soulnexus_lingecho-data:/data -v "$PWD":/backup alpine \
  sh -c 'cd /data && tar -xzf /backup/lingecho-backup-YYYY-MM-DD.tar.gz'
docker compose up -d
```

> 卷名前缀可能随项目目录名变化，可用 `docker volume ls` 确认。

## 使用 PostgreSQL（可选）

默认使用 SQLite。生产大规模部署建议改用 PostgreSQL：

1. 在 `.env` 中设置：
   ```env
   DB_DRIVER=postgres
   DSN=postgres://user:pass@postgres:5432/lingecho?sslmode=disable
   ```
2. 在 `docker-compose.yml` 中增加 `postgres` 服务并加入同一 network（需自行扩展 compose）。

## 故障排查

| 现象 | 处理 |
|------|------|
| 页面 502 / 空白 | `make logs` 查看 API 是否启动失败；常见原因：`SESSION_SECRET` 未设或配置校验失败 |
| 登录后频繁掉线 | 确认 `SESSION_SECRET` 固定且未在重启间变化 |
| 构建 Go 失败 | 确保网络可拉取 module；Go 版本由 `GOTOOLCHAIN=auto` 自动匹配 `go.mod` |
| 拉取 `node:*-bookworm` / `golang:*-bookworm` 失败 | 已改用 `node:20-alpine`、`golang:1.22-alpine` 等常见标签；若镜像站仍失败，请更换 Docker 镜像加速或直连 Docker Hub |
| `go mod download` 超时 / `proxy.golang.org` 不可达 | 默认已通过 `GOPROXY=https://goproxy.cn,direct` 构建；海外环境可在 `.env` 设 `GOPROXY=https://proxy.golang.org,direct` |
| `build constraints exclude all Go files` / `denoise` | 后端依赖 CGO，镜像已设 `CGO_ENABLED=1` 并安装 `gcc` |
| `Package 'opus' / 'opusfile' not found` | 构建镜像需 `pkg-config libopus-dev libopusfile-dev`（见 `deploy/apt-packages` # BUILD 段） |
| `npm ci` 很慢或失败 | 默认使用 `registry.npmmirror.com`；海外可设 `NPM_REGISTRY=https://registry.npmjs.org` |

## 开发 vs 生产

| 模式 | 前端 | 后端 |
|------|------|------|
| **开发** | `cd web && npm run dev`（:3000） | `go run ./cmd/server`（:7072） |
| **Docker 生产** | 构建进镜像，Nginx :80 | 同容器 :7072，反代 `/api` |

开发时前端通过 `VITE_API_BASE_URL` 指向后端；Docker 构建使用 `web/.env.production` 中的 `/api` 相对路径，与 Nginx 反代一致。

---

更多配置项见 [env.example](../env.example)，分布式扩展见 [distributed.md](distributed.md)。
