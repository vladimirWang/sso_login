# SSO 登录中心 - 其他系统接入指南

> go-zero + Go + PostgreSQL 实现的统一登录（SSO）服务，支持 **OAuth2 授权码模式**（类似淘宝/阿里云跳转支付宝登录的体验）与 **gRPC 会话校验** 两种接入方式。仅涉及普通用户（role=merchant）登录。

## 一、系统概览

| 服务 | 端口 | 说明 |
|---|---|---|
| auth-api（REST 网关 + SSO 登录页 + OAuth2 端点） | 8888 | 业务系统与浏览器的统一入口 |
| auth-rpc（gRPC 核心服务） | 8081 | 登录/注册/会话/授权码逻辑，etcd 注册（key: `auth.rpc`） |
| audit（事件消费者） | - | 订阅 RabbitMQ `auth.events`（user.login / user.logout） |

依赖基础设施：etcd:2379、PostgreSQL:15432（库 sso_login）、Redis:16379、RabbitMQ:5672。

```
浏览器                业务系统                  SSO(auth-api:8888)            auth-rpc(gRPC)
  │ 跳"使用XX登录"        │                             │                            │
  │ ─────────────────────────────────▶ /oauth/authorize ──┐                         │
  │                                     （无 cookie，302）│                         │
  │ ◀── 302 /oauth/login?client_id&redirect_uri&state ────┘                         │
  │ 输入邮箱/密码/验证码 → POST /api/v1/auth/login（成功种 sso_token cookie）         │
  │ ── GET /oauth/authorize（带 cookie）────────────────▶ OAuthAuthorize(gRPC) ────▶ │
  │ ◀── 302 redirect_uri?code=xxx&state=yyy ──────────────────────────────────────  │
  │ 回跳业务系统           │                            │                            │
  │                       │ POST /oauth/token          │                            │
  │                       │ (code+client_secret) ─────▶│ OAuthExchangeToken ──────▶ │
  │                       │ ◀── access_token(JWT) ─────│◀──── (会话写 Redis 1 天) ── │
  │                       │ GET /oauth/userinfo(Bearer)│── ValidateSession ───────▶ │
```

关键设计：
- **授权码**存 Redis `oauth:code:{code}`，5 分钟有效、一次性（用后即删），并校验 client_id / redirect_uri 与授权时一致。
- **access_token** 是 HS256 JWT（1 天），会话写 Redis `token:{token}`，与密码登录会话同构——登出、gRPC 校验、事件审计全部统一。
- **redirect_uri 必须与注册的完全一致**，未注册时返回 400 错误页，防开放重定向。
- SSO 域通过 httpOnly cookie `sso_token` 识别登录态（登录接口成功时由服务端写入，登出时清除）。

## 二、方式一：OAuth2 授权码模式接入（推荐）

### 1. 注册你的应用（client）

在 SSO 库插入一条 `oauth_clients`（`client_secret` 存 `sha256(明文)` 的十六进制）：

```sql
INSERT INTO oauth_clients (client_id, client_secret, name, redirect_uris)
VALUES (
  'your-app',
  '<sha256(your-secret) 的 hex>',
  '你的系统名称',
  'https://your-app.example.com/auth/sso/callback'  -- 逗号分隔可多个，精确匹配
);
```

生成 secret 哈希：`printf '%s' 'your-secret' | shasum -a 256 | awk '{print $1}'`

### 2. 引导用户跳转 SSO

业务系统点击「使用 SSO 登录」时（浏览器 302 或 location.href）：

```
GET http://<sso-host>:8888/oauth/authorize
    ?client_id=your-app
    &redirect_uri=https://your-app.example.com/auth/sso/callback   # 必须已注册
    &response_type=code
    &state=随机串                                                  # 防 CSRF，回调时回传
```

- 用户未登录：SSO 自动 302 到独立登录页 `/oauth/login?...`（页面读取参数，登录成功后自动回到 authorize 并跳回你系统）。
- 用户已登录：直接 302 回 `redirect_uri?code=xxx&state=yyy`，**全程免密**（单点登录效果）。

### 3. 回调处理：用 code 换 access_token（服务端完成）

```
POST http://<sso-host>:8888/oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code
&client_id=your-app
&client_secret=your-secret
&code=<回调得到的 code>
&redirect_uri=https://your-app.example.com/auth/sso/callback   # 与授权时完全一致
```

成功响应（失败返回 400 + `{"error":"invalid_grant","error_description":"..."}`）：

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 86400,
  "userinfo": { "userId": 3, "email": "a@b.com", "username": "张三",
                "tenantId": 2, "role": "merchant", "isSuperUser": true, "exp": 1792000000 }
}
```

`curl` 示例：

```bash
curl -s -XPOST http://127.0.0.1:8888/oauth/token \
  --data-urlencode grant_type=authorization_code \
  --data-urlencode client_id=demo-app \
  --data-urlencode client_secret=demo-secret \
  --data-urlencode code=$CODE \
  --data-urlencode redirect_uri=http://127.0.0.1:8888/demo-app/callback
```

### 4. 拉取用户信息（可选）

```bash
curl -s http://127.0.0.1:8888/oauth/userinfo -H "Authorization: Bearer <access_token>"
# → {"code":200,"message":"ok","data":{ userId, email, username, tenantId, role, isSuperUser, exp }}
```

### 5. Demo

本项目自带演示业务系统：浏览器打开 `http://127.0.0.1:8888/demo-app`，点击「使用 SSO 登录」即可体验完整流程。
自动化测试：`make test-oauth`（18 项断言）。

## 三、方式二：gRPC ValidateSession 接入（网关 / 内部服务）

已有 token（密码登录返回的 token 或 OAuth access_token 均可）的校验方式：

- etcd 地址 `127.0.0.1:2379`，服务 key `auth.rpc`；proto 见 [proto/auth/auth.proto](proto/auth/auth.proto)。

```go
// go-zero 客户端（或任意 gRPC 客户端）
authClient.ValidateSession(ctx, &auth.ValidateSessionReq{Token: token})
// code==200 → resp.User 为 UserInfo；401 → 会话无效/过期
```

内部 RPC 一览：`Login / Register / Logout / CurrentUser / UpdatePassword / GetSalt / CheckEmailExisted / GetNonce / GetCaptcha / ValidateSession / OAuthAuthorize / OAuthExchangeToken / OAuthClientInfo`。

## 四、SSO 自身 REST 接口（auth-api :8888，前缀 /api/v1）

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | /auth/register | 注册（自动建租户，注册人即超管） |
| POST | /auth/login | 登录（需 salt/nonce/captcha 三件套） |
| GET | /auth/getSalt/:email | 取用户 salt（32 位 hex） |
| GET | /auth/checkEmailExisted/:email | 邮箱是否已注册 |
| GET | /util/nonce | 取一次性 nonce（5 分钟） |
| GET | /util/captcha | 取图形验证码（base64 图 + captchaId，5 分钟一次性） |
| POST | /auth/logout | 登出（需 authorization 头） |
| GET | /auth/current | 当前用户信息（需 authorization 头） |
| POST | /auth/updatePassword | 修改密码（需 authorization 头） |

> 鉴权方式：请求头 `authorization: <token 原文>`（也接受 `Bearer <token>`）。若要自建登录前端，必须遵守密码协议：库中 `password = sha256(明文 + "_" + salt)`，登录提交 `sha256(库中password + "_" + nonce)`。直接用 SSO 登录页则无需关心。

## 五、错误码（与 repo_backend 对齐）

| code | 含义 | | code | 含义 |
|---|---|---|---|---|
| 200 | 成功 | | 10008 | 密码不正确（累计 6 次锁定 1 小时） |
| 10001 | 邮箱已存在 | | 10010 | 账号已锁定 |
| 10002 | 校验失败 | | 10011 | nonce 无效/已使用（防重放） |
| 10003 | 用户不存在 | | 10015 | 系统错误 |
| 10006 | 验证码已过期 | | 401 | 会话无效/已过期 |
| 10007 | 验证码不正确 | | | |

## 六、本地运维

```bash
make build        # 编译三个服务到 bin/
make sql          # 建表（001_init.sql）
psql "host=127.0.0.1 port=15432 user=postgres password=postgres dbname=sso_login sslmode=disable" \
  -f deploy/sql/002_oauth.sql   # oauth_clients 表 + demo-app 种子
psql "host=127.0.0.1 port=15432 user=postgres password=postgres dbname=sso_login sslmode=disable" \
  -f deploy/sql/003_repo_cms_client.sql   # repo-cms 客户端注册（仓库管理系统）
make run-rpc && make run-api && make run-audit
make test-api     # 登录主链路 e2e（15 项）
make test-oauth   # OAuth2 授权码 e2e（18 项）
```

SSO 登录页：`http://127.0.0.1:8888/oauth/login`（静态资源内嵌于二进制，源码 [services/auth/api/web/login.html](services/auth/api/web/login.html)）。

## 七、已接入系统：repo_cms（仓库管理系统）

> 接入策略：**SSO 仅作身份源（IdP），repo_cms 按 email 匹配本地已有 User，不自动开通**。
> 注册/激活仍走 repo_cms 自身流程（申请人 → 审核 → 邮件激活）；SSO 只提供"用统一账号登录"的入口。
> SSO userinfo 中的 tenantId/role/isSuperUser 是 SSO 自身账户模型字段，repo_cms 不作为授权依据
>（本地租户/超管由 repo_cms 自己的 User/Tenant 表决定）。

### 1. 客户端注册

[deploy/sql/003_repo_cms_client.sql](deploy/sql/003_repo_cms_client.sql)：client_id=`repo-cms`，secret 明文由 repo_cms 侧保管（repo_backend `.env` 的 `SSO_CLIENT_SECRET`），库中只存 sha256。redirect_uri 采用 **hash 路由形态**（见下）。

### 2. redirect_uri 的 hash 路由形态（SPA 接入要点）

repo_cms 前端是 hash 路由（`#/...`），注册的 redirect_uri 为：

```
http://localhost:5174/#/landing/sso/callback
```

Go `url.Parse` 把 `#` 后内容视为 Fragment，追加 code 时落在 Fragment 之前，最终落地 URL：

```
http://localhost:5174/?code=xxx&state=yyy#/landing/sso/callback
```

即 **code/state 在真实 query string 上（hash 之前）**。前端回调页因此不能用 react-router 的 `useSearchParams`（它读 hash 内参数），必须用 `window.location.search` 取参。

### 3. repo_cms 侧实现

- **前端入口**：[repo_frontend/src/pages/Login.tsx](../repo_cms/repo_frontend/src/pages/Login.tsx) 登录表单下方「使用 SSO 统一登录」按钮；`buildSsoAuthorizeUrl()`（api/user.ts）生成 authorize URL，随机 state 存 sessionStorage 兼做 CSRF 校验，回跳地址另存 `sso_redirect`。
- **前端回调页**：[repo_frontend/src/pages/SSOOAuthCallback.tsx](../repo_cms/repo_frontend/src/pages/SSOOAuthCallback.tsx)（路由 `/#/landing/sso/callback`）：校验 state → 取 `window.location.search` 的 code → POST `/user/sso/exchange` → 存 access_token → 进应用。
- **后端换 token**：[repo_backend/src/modules/user/ssoService.ts](../repo_cms/repo_backend/src/modules/user/ssoService.ts)（`POST /nodejs_api/user/sso/exchange`，公共路由）：服务端调 `POST /oauth/token` 换 userinfo → 按 email 查本地 User（查不到返回 10003 拒绝）→ 查 `Tenant.superUserId` 算 isSuperUser → 签发**本地 JWT** 写本地 Redis（payload 与密码登录完全同构）。SSO 颁发的 access_token 仅用于取 userinfo，用后即弃。

### 4. repo_backend 环境变量

| 变量 | 示例值 | 说明 |
|---|---|---|
| SSO_BASE_URL | http://127.0.0.1:8888 | SSO auth-api 地址 |
| SSO_CLIENT_ID | repo-cms | 客户端 ID |
| SSO_CLIENT_SECRET | repo-cms-xxx | 客户端明文 secret（与 003 SQL 中的 sha256 对应） |
| SSO_REDIRECT_URI | http://localhost:5174/#/landing/sso/callback | 必须与注册的完全一致 |

### 5. 验证

- 正向：浏览器打开 `http://localhost:5174` → 登录页点「使用 SSO 统一登录」→ SSO 页输入账号密码验证码 → 自动回跳进 dashboard（已验证）。
- 反向：SSO 有号但 repo_cms 本地无同邮箱 User → 回调换 token 返回 `{"code":10003,"message":"该邮箱未在本系统开通，请先完成注册激活"}`（已验证）。
