# 开发文档

## 1. 环境要求

| 工具 | 版本 |
|------|------|
| Go | 1.26+ |
| Node.js | 22+ |
| PostgreSQL | 14+ |
| Casdoor | 任意版本（可 Docker 启动） |
| Redis | 6+（可选） |

## 2. 本地开发

```bash
# 1. 准备数据库（空库即可，程序自动建表）
createdb -h <host> -p <port> -U <user> flyiam

# 2. 启动 Casdoor（可选：使用 docker）
docker run -d --name casdoor -p 8000:8000 \
  -e driverName=postgres \
  -e dataSourceName="host=<host> port=<port> user=<user> password=<pwd> dbname=flyiam sslmode=disable" \
  casbin/casdoor:latest

# 3. 本地配置（含真实凭据，已被 .gitignore 忽略，不会提交）
cp etc/config.yaml etc/config-local.yaml
# 按需修改数据库 / Casdoor / 管理员密码

# 4. 后端
go run ./cmd/api -c etc/config-local.yaml

# 5. 前端（开发模式，热更新；生产构建会嵌入二进制）
cd web && npm install && npm run dev
```

> 首次启动会自动创建业务表、Casdoor 组织/应用/管理员，无需手工 SQL。

## 3. 构建

```bash
# 完整构建（前端 + 后端 → 单二进制）
make build

# 或用脚本逐步构建
cd web && npm run build          # 输出到 internal/pkg/web/dist
cd .. && CGO_ENABLED=0 go build -o build/flyiam ./cmd/api
```

## 4. 常用命令

```bash
make help            # 查看全部命令
make build           # 完整构建
make run             # 构建并运行
make dev             # 前端开发服务器
make test            # 运行测试
make clean           # 清理产物
make docker-build    # 构建镜像
```

## 5. 代码结构约定

| 分层 | 目录 | 职责 |
|------|------|------|
| 配置 | `internal/config` | 配置结构、环境变量覆盖、校验 |
| 上下文 | `internal/svc` | 依赖注入、自动建表、自动初始化 Casdoor |
| 处理器 | `internal/handler` | HTTP 路由与参数解析 |
| 逻辑 | `internal/logic` | 业务逻辑 |
| 中间件 | `internal/middleware` | JWT 鉴权、权限校验 |
| 模型 | `internal/model` | 数据结构 |
| 公共包 | `internal/pkg` | Casdoor / 数据源 / 缓存 / 前端嵌入 |

## 6. 扩展数据源

1. 在 `internal/pkg/datasource/` 下新建包，实现 `DataSource` 接口：

```go
type DataSource interface {
    Name() string
    IsEnabled() bool
    Priority() int
    Sync(ctx context.Context) error
    GetUsers(ctx context.Context) ([]User, error)
    GetDepartments(ctx context.Context) ([]Department, error)
    GetSyncInterval() time.Duration
    HealthCheck(ctx context.Context) error
}
```

2. 在 `internal/logic/datasource` 的 `buildSource` 中按类型注册；
3. 页面「数据源」中配置即可生效。

## 7. 配置约定

- 所有配置项均支持环境变量覆盖（结构体 tag `env=XXX`）；
- 配置默认值集中在 `etc/config.yaml`（示例）与 `internal/config/config.go`；
- 敏感信息（密码、密钥）通过环境变量或 Secret 注入，不写入代码。

## 8. 相关文档

- [架构设计](../design/architecture.md)
- [数据库设计](../design/database.md)
- [API 设计](../design/api.md)
- [测试文档](../testing/testing.md)
- [Kubernetes 部署](../deployment/kubernetes.md)
