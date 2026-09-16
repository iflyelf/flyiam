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
或外置域名 `https://casdoor.example.com`，默认 `admin/ysyh!9Sky`，**首次登录请立即修改**）：

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
| 默认密码 | `FLYIAM_CASDOOR_DEFAULT_PASSWORD` | `ysyh!9Sky` |
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
| 手机号校验失败 | 确认 `FLYIAM_CASDOOR_COUNTRY_CODE=CN`（组织默认区域影响手机号解析） |
| 自动初始化失败 | 确认 Casdoor 已就绪且 `casdoor_application` 中存在 `app-built-in` |
