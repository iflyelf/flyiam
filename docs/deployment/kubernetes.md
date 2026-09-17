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
- **目标节点已打上 `flyiam=true` 标签**（硬性节点亲和性要求）

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

# 可选：Casdoor 浏览器访问地址（外置域名 / 跨域名部署时填写）
#   外置域名默认使用标准端口，无需显式书写端口号：
#     https://casdoor.example.com  → 443
#     http://casdoor.example.com   → 80
export FLYIAM_CASDOOR_PUBLIC_ENDPOINT="https://casdoor.example.com"

# 为目标节点打标签（硬性节点亲和性要求）
kubectl get nodes
kubectl label nodes <node-1> flyiam=true
kubectl label nodes <node-2> flyiam=true

# 部署
helmfile sync

# 指定环境
helmfile -e prod sync
```

### 3.1 节点亲和性

FlyIAM 应用与内置 Casdoor 均配置了**硬性节点亲和性**，必须调度到带
`flyiam=true` 标签的 Linux 节点：

```bash
kubectl get nodes                 # 查看节点
kubectl label nodes <node-1> flyiam=true
kubectl get nodes -l flyiam=true  # 确认标签
```

```yaml
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
        - matchExpressions:
            - key: flyiam          # nodeLabel
              operator: In
              values:
                - "true"            # nodeLabelValue
            - key: kubernetes.io/os
              operator: In
              values:
                - linux
  podAntiAffinity:                 # 硬性打散：同一 release 的 Pod 不共节点
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
            - key: app.kubernetes.io/name
              operator: In
              values:
                - flyiam
        topologyKey: kubernetes.io/hostname
```

标签可通过环境变量覆盖：

```bash
export FLYIAM_NODE_LABEL="flyiam"
export FLYIAM_NODE_LABEL_VALUE="true"
```

> **硬性调度**：不满足时 Pod 会一直 `Pending`，用 `kubectl describe pod` 查看调度事件。
> 打散为硬性（`required`），**带 `flyiam=true` 标签的节点数需 ≥ 4**
> （应用 2 副本 + 内置 Casdoor 2 副本，共用 `app.kubernetes.io/name=flyiam` 标签互相排斥）。
> 节点不足时下调 `FLYIAM_REPLICAS` / `CASDOOR_REPLICAS`。

## 4. 内置 Casdoor 说明

| 项 | 默认 | 环境变量 |
|----|------|---------|
| 镜像 | `casbin/casdoor:latest` | `CASDOOR_IMAGE_TAG` |
| initContainer 镜像 | `busybox:1.36` | `CASDOOR_INIT_IMAGE_TAG` |
| 副本数 | `2` | `CASDOOR_REPLICAS` |
| 端口 | `8000` | `CASDOOR_SERVICE_PORT` |
| 时区 | `Asia/Shanghai` | `CASDOOR_TIMEZONE` |
| 运行模式 | `prod` | `CASDOOR_RUN_MODE` |
| 是否部署 | `true` | `CASDOOR_ENABLED` |

> **关于 `/swagger` 404**：Casdoor 仅在 `runmode = dev` 时注册 `/swagger` 静态路由，
> 因此默认 `CASDOOR_RUN_MODE=prod` 下访问 `/swagger` 返回 404 属正常现象。
> 如需临时查看可 `export CASDOOR_RUN_MODE=dev && helmfile sync`（dev 还会开启调试错误页，生产勿用）。
> 详见 [Casdoor 配置](casdoor.md)。

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
| `FLYIAM_IMAGE_PULL_POLICY` | 应用镜像拉取策略 | `Always` |
| `CASDOOR_IMAGE_PULL_POLICY` | Casdoor 镜像拉取策略 | `Always` |
| `CASDOOR_INIT_IMAGE_TAG` | Casdoor initContainer 镜像标签 | `1.36` |
| `FLYIAM_NODE_LABEL` / `FLYIAM_NODE_LABEL_VALUE` | 硬性节点亲和性标签 | `flyiam` / `true` |
| `FLYIAM_DB_HOST` | 数据库地址 | `postgres.default.svc.cluster.local` |
| `FLYIAM_DB_PASSWORD` | 数据库密码 | - |
| `FLYIAM_REDIS_HOST` | Redis 地址 | `redis.default.svc.cluster.local` |
| `FLYIAM_JWT_SECRET` | JWT 密钥（≥32 位） | - |
| `FLYIAM_ADMIN_PASSWORD` | 管理员密码 | - |
| `FLYIAM_CASDOOR_ENDPOINT` | Casdoor 后端地址（集群内） | `http://casdoor:8000` |
| `FLYIAM_CASDOOR_PUBLIC_ENDPOINT` | Casdoor 浏览器地址（外置域名，标准端口 80/443） | - |
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

> **外置域名访问**：通过集群入口（Ingress Controller / Gateway）以域名暴露 Casdoor 时，
> 将 `FLYIAM_CASDOOR_PUBLIC_ENDPOINT` 设为该域名（默认走标准端口 **443(https) / 80(http)**，
> 无需显式书写端口号，例如 `https://casdoor.example.com`）。
> 该地址同时作为内置 Casdoor 的 `origin`，用于生成正确的登录跳转地址。
> FlyIAM 登录时会自动把回调地址加入 Casdoor 应用白名单（`FLYIAM_CASDOOR_AUTO_REDIRECT_URI`）。

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

## 8. 更新与卸载

```bash
# 更新（修改配置/镜像后）
helmfile -f helmfile.yaml.gotmpl diff    # 查看变更
helmfile -f helmfile.yaml.gotmpl sync    # 应用变更

# 卸载
helmfile -e default destroy
```
