# sso-service

`sso-service` 是 SoulNexus 的独立认证服务骨架，采用 OAuth2/OIDC（Authorization Code + PKCE）协议形态。SSO 侧通过 `cookie/session` 维持全局登录态，子系统通过 OIDC token 完成鉴权。

核心端点：

- `/auth/login`
- `/auth/logout`
- `/.well-known/openid-configuration`
- `/oauth/authorize`
- `/oauth/token`
- `/oauth/jwks`
- `/oauth/userinfo`
- `/oauth/revoke`
- `/oauth/introspect`

## Quick Start

1. 进入目录：`cd sso-service`
2. 拉取依赖：`go mod tidy`
3. 启动服务：`go run ./cmd/server`

默认端口 `8090`，数据库为本地 `sqlite` 文件 `sso.db`。

## Seed Data

首次启动会自动创建：

- OAuth Client
  - `client_id`: `portal-web`
  - `client_secret`: `portal-web-secret`
  - `redirect_uri`: `http://localhost:5173/callback`
- Demo User
  - `user_id`: `user-demo-1`

## Demo Flow

1. 先登录（建立 SSO 会话）：
   - `POST /auth/login`
   - json body:
     - `email=demo@soulnexus.local`
     - `password=demo123456`
2. 获取授权码（登录后携带 `sso_session` Cookie）：
   - `GET /oauth/authorize?response_type=code&client_id=portal-web&redirect_uri=http://localhost:5173/callback&scope=openid%20profile%20email&state=abc&code_challenge=xyz&code_challenge_method=plain`
3. 用授权码换 token：
   - `POST /oauth/token`
   - form fields:
     - `grant_type=authorization_code`
     - `code=<from_step_1>`
     - `client_id=portal-web`
     - `client_secret=portal-web-secret`
     - `redirect_uri=http://localhost:5173/callback`
     - `code_verifier=xyz`

## Notes

- 当前版本用于项目内拆分和联调，已包含用户密码校验、会话持久化、令牌签发、审计日志基础模块。
- 业务服务应切换到“本地 JWT 验签 + claims 授权”，不再自行签发访问令牌。

## 认证行为

- `/auth/login`：校验账号密码并设置 `sso_session` Cookie。
- `/oauth/authorize`：要求已存在有效 SSO Session，成功后签发授权码。
- `/oauth/token`：通过授权码（或 refresh token）签发 token。
- `/oauth/userinfo`、`/oauth/introspect`：通过 Bearer access token 访问。
