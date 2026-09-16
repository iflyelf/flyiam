# 部署 - Kubernetes

本项目提供 Helm Chart（`charts/flyiam`），**内置 Casdoor**，采用 Helmfile 多环境管理。

## 1. 架构

```
Helmfile
  └── flyiam Chart
       ├── Deployment  flyiam         # 应用（含嵌入前端）
       ├── Service     flyiam
       ├── Deployment  flyiam-casdoor # 内置认证中心
       ├── Service     casdoor
       └── ConfigMap   flyiam-casdoor-config  # Casdoor app.conf 模板
```

数据库（PostgreSQL）与缓存（Redis）为**外置依赖**，Chart 不部署。
Chart 仅暴露 ClusterIP Service，不包含 Ingress；域名访问请在集群入口层（Ingress Controller / Gateway）统一配置。

## 2. 前置条件

- Kubernetes 1.24+
- Helm 3.x、Helmfile 0.150+
- 外置 PostgreSQL（已创建空库 `flyiam`）
- 外置 Redis（可选）

## 3. 快速部署

```bash
cd charts/flyiam

# 必填环境变量
export FLYIAM_DB_HOST="postgres.default.svc.cluster.local"
export FLYIAM_DB_PORT="5432"
export FLYIAM_DB_NAME="flyiam"
export FLYIAM_DB_USER="flyiam"
export FLYIAM_DB_PASSWORD="your-db-password"
export FLYIAM_JWT_SECRET="your-jwt-secret-at-least-32-chars"
export FLYIAM_ADMIN_PASSWORD="your-admin-password"

# 可选：Redis
export FLYIAM_REDIS_HOST="redis.default.svc.cluster.local"
export FLYIAM_REDIS_PASSWORD="your-redis-password"

# 可选：对外访问地址（跨域名部署必填）
export FLYIAM_CASDOOR_PUBLIC_ENDPOINT="http://flyiam.example.com:8000"

# 部署
helmfile sync

# 指定环境
helmfile -e prod sync
```

## 4. 内置 Casdoor 说明

| 项 | 默认 | 环境变量 |
|----|------|---------|
| 镜像 | `casbin/casdoor:latest` | `CASDOOR_IMAGE_TAG` |
| 副本数 | `1` | `CASDOOR_REPLICAS` |
| 端口 | `8000` | `CASDOOR_SERVICE_PORT` |
| 时区 | `Asia/Shanghai` | `CASDOOR_TIMEZONE` |
| 运行模式 | `prod` | `CASDOOR_RUN_MODE` |
| 是否部署 | `true` | `CASDOOR_ENABLED` |

Casdoor 与 FlyIAM **共用同一数据库**（表前缀 `casdoor_`），
其 `app.conf` 由 ConfigMap 提供，数据库密码通过 initContainer 注入（不在 ConfigMap 明文保存）。

**首次启动自动完成**：

1. Casdoor 自动创建 `casdoor_*` 表、内置组织与应用；
2. FlyIAM 自动创建业务组织 `flyiam`、应用 `flyiam`、管理员账号，并回填应用凭据；
3. 业务表由 FlyIAM 自动创建。

## 5. 配置项（节选）

`charts/flyiam/values/_base.yaml.gotmpl` 集中管理，全部支持环境变量覆盖：

| 变量 | 说明 | 默认 |
|------|------|------|
| `FLYIAM_NAMESPACE` | 命名空间 | `flyiam` |
| `FLYIAM_REPLICAS` | 应用副本数 | `2` |
| `FLYIAM_IMAGE_TAG` | 应用镜像标签 | `latest` |
| `FLYIAM_DB_HOST` | 数据库地址 | `postgres.default.svc.cluster.local` |
| `FLYIAM_DB_PASSWORD` | 数据库密码 | - |
| `FLYIAM_REDIS_HOST` | Redis 地址 | `redis.default.svc.cluster.local` |
| `FLYIAM_JWT_SECRET` | JWT 密钥（≥32 位） | - |
| `FLYIAM_ADMIN_PASSWORD` | 管理员密码 | - |
| `FLYIAM_CASDOOR_ENDPOINT` | Casdoor 内网地址 | `http://casdoor:8000` |
| `FLYIAM_CASDOOR_PUBLIC_ENDPOINT` | Casdoor 浏览器地址 | - |
| `FLYIAM_CASDOOR_AUTO_SETUP` | 自动初始化 Casdoor | `true` |
| `FLYIAM_CASDOOR_DEFAULT_PASSWORD` | 新用户默认密码 | `ysyh!9Sky` |
| `FLYIAM_CASDOOR_COUNTRY_CODE` | 手机号区域 | `CN` |

## 6. 访问

```bash
# 应用
kubectl port-forward -n flyiam svc/flyiam 8081:8081
# 浏览器访问 http://localhost:8081

# Casdoor
kubectl port-forward -n flyiam svc/casdoor 8000:8000
# 浏览器访问 http://localhost:8000
```

> 通过集群入口（Ingress Controller / Gateway）暴露时，需将 Casdoor 对外地址填入
> `FLYIAM_CASDOOR_PUBLIC_ENDPOINT`；FlyIAM 登录时会自动把回调地址加入 Casdoor 应用白名单
> （`FLYIAM_CASDOOR_AUTO_REDIRECT_URI`）。

## 7. 验证

```bash
helmfile -e default lint      # 语法检查
helmfile -e default template  # 渲染清单
helmfile -e default diff      # 查看变更
helmfile -e default sync      # 部署

kubectl get pods -n flyiam
kubectl logs -n flyiam deploy/flyiam | head -30
```

预期日志：

```
✅ 数据库连接成功
✅ 数据库初始化完成
✅ Casdoor 组织已创建: flyiam
✅ Casdoor 应用已创建: flyiam (clientId=...)
✅ Casdoor 管理员账号已创建: flyiam/admin
✅ Casdoor 自动初始化完成
```

## 8. 卸载

```bash
helmfile -e default destroy
```
