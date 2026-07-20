# 3D-Speaker 声纹识别API

基于3D-Speaker模型的声纹识别服务，提供声纹注册、识别、删除等功能。

与 **SoulNexus 主库共用同一张表 `voiceprints`**。数据库驱动与主项目对齐，支持：

| 驱动 | `DB_DRIVER` | DSN 示例 |
|------|-------------|---------|
| SQLite | `sqlite` | `./lingecho.db` |
| PostgreSQL | `postgres` | `host=127.0.0.1 user=postgres password=xxx dbname=soulnexus sslmode=disable` |
| MySQL | `mysql` | `user:pass@127.0.0.1:3306/soulnexus?charset=utf8mb4` |

## 🛠️ 安装和配置

### 1. 安装依赖
```bash
conda remove -n voiceprint-api --all -y
conda create -n voiceprint-api python=3.10 -y
conda activate voiceprint-api
pip config set global.index-url https://mirrors.aliyun.com/pypi/simple/

pip install -r requirements.txt
```

### 2. 数据库配置（与 SoulNexus 共用）

优先顺序：

1. **环境变量**（推荐，与主项目一致）
   ```bash
   export DB_DRIVER=sqlite
   export DSN=./lingecho.db
   ```
2. `data/.voiceprint.yaml` 中的 `database:` 段
3. 旧版 `mysql:` 段（仅 MySQL）

复制示例配置：
```bash
mkdir -p data
cp voiceprint.yaml data/.voiceprint.yaml
```

SQLite 示例（`data/.voiceprint.yaml`）：
```yaml
database:
  driver: sqlite
  dsn: ./lingecho.db   # 请指向与 SoulNexus 相同的库文件
```

PostgreSQL 示例：
```yaml
database:
  driver: postgres
  dsn: host=127.0.0.1 user=postgres password=secret dbname=soulnexus sslmode=disable
```

表结构由 SoulNexus GORM AutoMigrate 维护即可；也可对照 `scripts/schema.sql`（MySQL）/ `scripts/schema_sqlite.sql` / `scripts/schema_postgres.sql`。

账号声纹归属在 `tenant_users.voiceprint_id`，**不在** `voiceprints` 业务外键上。

## 🚀 启动服务

### 开发环境
```bash
# 与主项目共用 sqlite
export DB_DRIVER=sqlite
export DSN=/absolute/path/to/lingecho.db
python -m app.main
```

### 生产环境
```bash
python start_server.py
```

## 📚 API文档

启动服务后：
- Swagger UI: http://localhost:8005/voiceprint/docs
