# 部署 - Casdoor 认证配置

FlyIAM 的认证（OAuth2 授权码模式）依赖 **Casdoor**。
本项目的 Helm Chart 与 docker-compose **内置 Casdoor**，默认自动完成初始化，通常无需手工配置。

## 1. 自动初始化（推荐）

开启 `CASDOOR_AUTO_SETUP=true`（默认）后，FlyIAM 首次启动会自动：

1. 读取 Casdoor 内置应用 `app-built-in` 凭据（从共享数据库读取）；
2. 创建业务组织（默认 `flyiam`，语言 `zh`，地区 `CN`）；
3. 创建业务应用（默认 `flyiam`，含回调白名单、密码登录、证书）；
4. 创建组织管理员账号（`flyiam/admin`，密码取 `CASDOOR_DEFAULT_PASSWORD`）；
5. 回填应用 `clientId/clientSecret` 供运行时使用。

启动日志示例：

```
✅ Casdoor 组织已创建: flyiam
✅ Casdoor 应用已创建: flyiam (clientId=93ccb59a50e29541b6ba)
✅ Casdoor 管理员账号已创建: flyiam/admin
✅ Casdoor 自动初始化完成（组织=flyiam 应用=flyiam clientId=...）
```

## 2. 回调地址（Redirect URI）

FlyIAM 登录回调：`http(s)://<FlyIAM 域名>:<端口>/api/auth/callback`

- 开启 `CASDOOR_AUTO_REDIRECT_URI=true`（默认）后，
  登录时会**自动把当前访问域名的回调地址加入应用白名单**，无需手工维护；
- 也可通过配置 `CASDOOR_REDIRECT_URIS`（列表）预置固定地址；
- 若报错 `Redirect URI ... doesn't exist in the allowed Redirect URI list`，
  说明自动追加被关闭且白名单缺失，请开启自动追加或到 Casdoor 应用页手工添加。

### 外置域名访问（标准端口）

Casdoor 通过集群入口（Ingress Controller / Gateway）以域名对外暴露时，
`FLYIAM_CASDOOR_PUBLIC_ENDPOINT` 直接填写域名即可，**默认走标准端口**：

```
https://casdoor.example.com   → 443
http://casdoor.example.com    → 80
```

标准端口无需显式书写端口号；仅当使用非标准端口时才需写成 `https://casdoor.example.com:8443`。
该地址同时写入内置 Casdoor 的 `origin`（app.conf），用于生成正确的登录跳转地址。

### Casdoor 自带 Swagger（/swagger）

Casdoor 仅在 **`runmode = dev`** 时注册 `/swagger` 静态路由（见上游 `main.go`），
因此本 Chart 默认的 **`CASDOOR_RUN_MODE=prod` 下访问 `/swagger` 会返回 404**，
这是 Casdoor 的设计，不是网络或入口配置问题。

如需临时查看 Swagger UI（**不建议长期用于生产**，dev 模式会同时打开调试错误页与详细日志）：

```bash
export CASDOOR_RUN_MODE=dev
helmfile -f helmfile.yaml.gotmpl sync
# 访问 http://<casdoor>:8000/swagger/
```

> 生产环境请保持 `prod`。API 文档可参考 Casdoor 官方：
> https://casdoor.org/docs/basic/server-installation 或上游仓库 `swagger/` 目录。

## 3. 手动配置（可选）

如需手工维护，登录 Casdoor 管理后台（集群内 `http://<casdoor>:8000`，
或外置域名 `https://casdoor.example.com`）：

> **登录账号说明**：
> - **Casdoor 内置管理员**：`built-in/admin`，密码固定为 **`123`**（Casdoor 上游源码硬编码创建，
>   与 `CASDOOR_DEFAULT_PASSWORD` 无关，**首次登录请立即修改**）；
> - **业务组织管理员**：`flyiam/admin`，密码为 `CASDOOR_DEFAULT_PASSWORD`（默认 `ysyh!9Sky`）。

1. **组织**：创建组织 `flyiam`（语言选中文，地区选中国）；
2. **应用**：创建应用 `flyiam`
   - 组织：`flyiam`
   - 客户端 ID / 密钥：填入 `CASDOOR_CLIENT_ID` / `CASDOOR_CLIENT_SECRET`
   - 回调 URL：
     ```
     http://<FlyIAM 域名>:8081/api/auth/callback
     ```
   - 证书：`cert-built-in`
3. **用户**：在 `flyiam` 组织下创建用户（或由数据源同步自动创建）。

## 3.0 登录限制：仅管理员可登录

FlyIAM 是管理后台，**只有管理员可以使用**。OAuth 回调阶段即校验，
非管理员即使通过 Casdoor 认证也会被拒绝，并返回「无权访问」提示页。

管理员判定（满足其一）：

1. Casdoor 中该用户被标记为管理员（`isAdmin`）；
2. 用户名在配置的超级管理员名单（`PERMISSION_ADMIN_USERS`，无代码内默认值）。

Helm 部署时可通过以下变量配置（逗号分隔）：

| 变量 | 说明 | 默认 |
|------|------|------|
| `FLYIAM_PERMISSION_ADMIN_USERS` | 超级管理员名单 | 空（仅依赖 Casdoor `isAdmin`） |
| `FLYIAM_CASDOOR_PROTECTED_USERS` | 受保护用户（同步/删除时跳过） | 空（`<组织>/admin` 程序内始终受保护） |

```bash
export FLYIAM_PERMISSION_ADMIN_USERS="junwang66,admin"
export FLYIAM_CASDOOR_PROTECTED_USERS="junwang66"
helmfile sync
```

被拒绝时会记录日志：

```
🚫 拒绝非管理员登录: zhangsan（仅管理员可访问本系统）
```

> 注意：这与 **Consul Manager 不同**——Consul Manager 允许普通用户登录
> （仅「用户/团队/角色」等管理页面需要管理员），FlyIAM 则是整体仅管理员可登录。

## 3.0.1 登录安全（state 校验与回调白名单）

- **OAuth state 校验**：登录时生成随机 state 并写入 HttpOnly Cookie，回调时比对，
  防止登录 CSRF。无需配置。
- **回调地址主机白名单**：`AutoRedirectURI` 会依据请求 Host 自动把回调地址写入
  Casdoor 应用白名单。为防止 Host 头被伪造导致开放重定向，生产环境应显式配置
  允许的主机：

  ```bash
  export FLYIAM_CASDOOR_ALLOWED_REDIRECT_HOSTS="flyiam.example.com"
  ```

  非空时，仅白名单内主机的回调会被自动追加；为空时保持原行为并打印告警。

## 3.0.2 默认密码

`CASDOOR_DEFAULT_PASSWORD` 用于新增用户与重置密码，**生产环境请务必覆盖**
（Chart 默认值为 `ysyh!9Sky`，仅便于快速开始）。

## 3.0.3 跨域部署（前后端不同源）

默认前后端**同源**（前端构建产物嵌入后端二进制，同一端口），此时
`SameSite=Lax` 即可，无需任何额外配置。

仅当把前端与后端部署到**不同域名**时，才需要配置以下三项：

**1. 前端构建**（指定后端地址）：
```bash
export VITE_API_BASE_URL="https://flyiam-api.example.com"
cd web && npm run build
```

**2. 后端 Cookie**（必须 None + HTTPS）：
```bash
export FLYIAM_AUTH_COOKIE_SAMESITE="none"
# Secure：SameSite=none 时自动置 true（也可显式 AUTH_COOKIE_SECURE=true）
# 跨子域共享时：export FLYIAM_AUTH_COOKIE_DOMAIN=".example.com"
```

**3. 后端 CORS**（显式列出来源，不能用 `*`）：
```bash
export FLYIAM_CORS_ALLOWED_ORIGINS="https://flyiam.example.com"
```
配置后 go-zero 会自动下发 `Access-Control-Allow-Origin` 与
`Access-Control-Allow-Credentials: true`，允许跨域携带登录 Cookie。

> ⚠️ `SameSite=None` 会削弱 CSRF 防护，非跨域部署请保持默认 `lax`。

## 3.1 权限模型：应用 / 组织管理需内置应用凭据

Casdoor 的 API 授权中，**只有 `built-in` 组织的身份是全局管理员**
（`authz.IsAllowed` 要求 `appUser.IsGlobalAdmin()`，即 `Owner == "built-in"`）。
FlyIAM 的常规业务请求使用业务应用凭据（`flyiam` 应用），因此：

| 操作 | 业务应用凭据 | 说明 |
|------|-------------|------|
| 登录 / OAuth、本组织用户增删改查 | ✅ 可用 | 对象 owner 与自身组织一致 |
| 应用列表、应用创建/修改/删除 | ❌ 报 `Unauthorized operation` | 应用 owner 为 `admin`（全局对象） |
| 组织列表、组织创建/修改/删除 | ❌ 报 `Please sign in first` | 控制器要求全局管理员 |

FlyIAM 启动时会读取内置应用 `app-built-in` 的凭据（`CASDOOR_AUTO_SETUP=true`
创建后即为共享数据库中的 `casdoor_application` 记录），并创建一个**独立的管理客户端**，
仅「应用 / 组织管理」接口使用它。业务接口仍用业务凭据，两者互不干扰。

**自动恢复（无需人工重启）**：Casdoor 首次建表可能晚于 FlyIAM 启动，
因此 FlyIAM 做了两层兜底：

1. **按需补读**：首次访问应用/组织管理接口时，若凭据仍缺失会自动补读一次（失败有 30s 节流）；
2. **后台重试**：启动后每 10s 重试一次，最长 1 小时，成功即打印
   `✅ Casdoor 管理凭据已就绪（内置应用 app-built-in）`。

只有在 Casdoor 长时间未完成初始化（`casdoor_application` 表始终没有 `app-built-in`）时，
管理页面才会持续返回「管理凭据未就绪」。此时请排查 Casdoor 自身状态：

```bash
# Casdoor 是否在用同一个库、表前缀是否正确
kubectl exec -n flyiam deploy/flyiam-casdoor -- grep -E "dbName|tableNamePrefix" /conf/app.conf

# Casdoor 启动日志
kubectl logs -n flyiam deploy/flyiam-casdoor | tail -50

# app-built-in 是否存在
psql "$DATABASE_URL" -c "SELECT name FROM casdoor_application WHERE name='app-built-in';"
```

> 该警告**只影响应用/组织管理页面**，登录与用户管理功能不受影响。

## 3.1.1 配置热重载（免重启）

Casdoor 连接配置（`casdoor.endpoint` / `public_endpoint` / `organization` /
`application` / `certificate` / `client_id` / `client_secret` 等）已支持
**页面修改后即时生效**，无需重启 Pod：

- 在「系统设置 → Casdoor 连接」修改并保存；
- 服务端检测到 `casdoor.*` 变更后，会用最新配置**原子重建** Casdoor 客户端；
- 定时同步、登录、用户管理等后续请求自动使用新客户端。

> 内部实现：`ServiceContext` 持有 `atomic.Pointer[casdoor.Client]`，
> 通过 `Casdoor()` 访问器读取、`ReloadCasdoor()` 重建替换，切换过程对并发请求安全。
> 若重建失败（如地址不可达），配置已保存但旧客户端仍继续服务，页面会提示错误。

## 3.2 多副本与会话共享（重要）

Casdoor 默认把登录会话保存在**本 Pod 的文件**中（`./tmp`）。内置 Casdoor 多副本
（`CASDOOR_REPLICAS > 1`）且 Service 无会话亲和时，同一浏览器的请求可能落到另一个
Pod，读不到会话即被当作匿名，表现为：

```
Unauthorized operation
```

因此 Chart 默认开启 **Redis 共享会话**：把 `redisEndpoint` 写入 `app.conf`，所有副本
共享同一份会话（复用外置 Redis，会话 DB 默认独立为 `1`，避免与 FlyIAM 缓存键冲突）。

| 变量 | 说明 | 默认 |
|------|------|------|
| `CASDOOR_REDIS_SESSION_ENABLED` | 是否用 Redis 共享会话 | 跟随 `FLYIAM_REDIS_ENABLED`（默认 `true`） |
| `CASDOOR_REDIS_DB` | 会话 Redis 数据库编号 | `1` |

> **地址、端口、密码直接复用 FlyIAM 的 Redis 配置**（`FLYIAM_REDIS_HOST` /
> `FLYIAM_REDIS_PORT` / `FLYIAM_REDIS_PASSWORD`），无需重复配置；
> 密码由 initContainer 从 Secret 的 `REDIS_PASSWORD` 注入，不在 ConfigMap 中明文保存。
> 使用 `existingSecret` 时需确保其包含 `REDIS_PASSWORD`。
> 仅数据库编号 `CASDOOR_REDIS_DB` 需要单独设置（与缓存 DB 分开，避免键冲突）。

> ⚠️ 关闭 `CASDOOR_REDIS_SESSION_ENABLED` 时，必须将 `CASDOOR_REPLICAS` 设为 `1`，
> 否则 `helmfile sync` 会在前置检查阶段直接报错并终止（避免部署后登录异常）。

## 4. 关键配置项

| 配置 | 环境变量 | 默认 |
|------|---------|------|
| Casdoor 后端地址（集群内） | `CASDOOR_ENDPOINT` / `FLYIAM_CASDOOR_ENDPOINT` | `http://casdoor:8000` |
| Casdoor 浏览器地址（外置域名） | `FLYIAM_CASDOOR_PUBLIC_ENDPOINT` | 空（用访问域名；外置域名标准端口 80/443） |
| 应用 Client ID | `FLYIAM_CASDOOR_CLIENT_ID` | 空（自动创建） |
| 应用 Client Secret | `FLYIAM_CASDOOR_CLIENT_SECRET` | 空（自动创建） |
| 组织名 | `FLYIAM_CASDOOR_ORGANIZATION` | `flyiam` |
| 应用名 | `FLYIAM_CASDOOR_APPLICATION` | `flyiam` |
| 证书名 | `FLYIAM_CASDOOR_CERTIFICATE` | `cert-built-in` |
| 默认密码（业务组织管理员/新用户） | `FLYIAM_CASDOOR_DEFAULT_PASSWORD` | `ysyh!9Sky` |
| 内置管理员密码（不可配，上游硬编码） | - | `123` |
| 手机号区域 | `FLYIAM_CASDOOR_COUNTRY_CODE` | `CN` |
| 自动初始化 | `FLYIAM_CASDOOR_AUTO_SETUP` | `true` |
| 自动追加回调 | `FLYIAM_CASDOOR_AUTO_REDIRECT_URI` | `true` |

## 5. 数据库约定

Casdoor 与 FlyIAM **共用同一 PostgreSQL 数据库**，Casdoor 表统一前缀 `casdoor_`。
Casdoor 首次启动会自动创建其全部表，无需手工执行 SQL。

## 6. 常见问题

| 现象 | 原因与处理 |
|------|-----------|
| `Redirect URI ... doesn't exist` | 开启自动追加回调（`CASDOOR_AUTO_REDIRECT_URI=true`）或手工加入白名单 |
| 浏览器跳转 `localhost:8000` 打不开 | 设置 `FLYIAM_CASDOOR_PUBLIC_ENDPOINT` 为浏览器可达的外置域名（标准端口 80/443） |
| 登录后回到登录页 | 查看 FlyIAM 日志中 `/api/auth/callback` 报错 |
| 登录时报 `Unauthorized operation` | Casdoor 多副本会话不共享：确认 `CASDOOR_REDIS_SESSION_ENABLED=true` 且各副本可访问同一 Redis；或将 `CASDOOR_REPLICAS` 设为 `1` |
| 手机号校验失败 | 确认 `FLYIAM_CASDOOR_COUNTRY_CODE=CN`（组织默认区域影响手机号解析） |
| 自动初始化失败 | 确认 Casdoor 已就绪且 `casdoor_application` 中存在 `app-built-in` |
| 报 `invalid character '<' looking for beginning of value` | `CASDOOR_ENDPOINT` 指向的不是 Casdoor API（返回了 HTML，如前端页面/Ingress 首页）。用 `kubectl exec -n flyiam deploy/flyiam -- curl -sS -i "$CASDOOR_ENDPOINT/api/health" \| head` 确认应返回 JSON `{"status":"ok"}`；集群内正确值通常为 `http://casdoor:8000` |
| 应用/组织管理页报 `Unauthorized operation` 或 `Please sign in first` | 全局对象需内置应用权限。正常由启动时自动读取 `app-built-in` 解决；若日志有「读取 Casdoor 内置应用凭据失败」，确认 `casdoor_application` 表中存在 `app-built-in` |
| 修改了 `CASDOOR_DEFAULT_PASSWORD` 但 Casdoor 登录仍为 `123` | `123` 是 Casdoor 内置管理员 `built-in/admin` 的硬编码密码，不受配置影响；业务管理员请用 `flyiam/admin` + 配置的默认密码登录 |
| 不知道 Casdoor 后台登录密码 | 内置管理员 `admin` / `123`（首次登录请立即修改） |
