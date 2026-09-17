# FlyIAM Helm Chart

[![Chart Version](https://img.shields.io/badge/Chart%20Version-1.0.0-blue)](https://github.com/iflyelf/flyiam)
[![App Version](https://img.shields.io/badge/App%20Version-1.0.0-green)](https://github.com/iflyelf/flyiam)

FlyIAM 统一用户管理系统的 Helm Chart，**内置 Casdoor 认证中心**，采用 Helmfile 结构，支持全量环境变量覆盖。

## 特性

- ✅ **内置 Casdoor**（`casbin/casdoor:latest`，中文 / 上海时区）
- ✅ **零手工初始化**：自动建表、自动创建组织 / 应用 / 管理员
- ✅ **Helmfile 结构** + 多环境（default / dev / staging / prod）
- ✅ **全量环境变量覆盖**（`{{ env "VAR" | default "值" }}`）
- ✅ **部署前置条件检查**（hooks）
- ✅ 高可用（多副本 + Pod 反亲和性）、HPA
- ✅ 外置 PostgreSQL / Redis（密码集中在 `values/_base.yaml.gotmpl`）

## 前置要求

- Kubernetes 1.24+、Helm 3.x、Helmfile 0.150+
- 外置 PostgreSQL（已创建空库 `flyiam`）
- 外置 Redis（可选）
- **目标节点已打上 `flyiam=true` 标签**（硬性节点亲和性要求，见[节点亲和性配置](#节点亲和性配置)）

## 快速开始

```bash
cd charts/flyiam

export FLYIAM_DB_HOST="postgres.default.svc.cluster.local"
export FLYIAM_DB_PORT="5432"
export FLYIAM_DB_NAME="flyiam"
export FLYIAM_DB_USER="flyiam"
export FLYIAM_DB_PASSWORD="your-db-password"
export FLYIAM_JWT_SECRET="your-jwt-secret-at-least-32-chars"
export FLYIAM_ADMIN_PASSWORD="your-admin-password"

# 为目标节点打标签（硬性节点亲和性要求）
kubectl get nodes
kubectl label nodes <node-1> flyiam=true
kubectl label nodes <node-2> flyiam=true

helmfile sync
```

首次启动会自动：建表 → 创建 Casdoor 组织/应用/管理员 → 回填应用凭据。

## 项目结构

```
charts/flyiam/
├── Chart.yaml
├── helmfile.yaml.gotmpl          # Helmfile 主配置（多环境）
├── values/
│   ├── _base.yaml.gotmpl        # 公共默认值（唯一需要编辑的配置文件）
│   └── flyiam.yaml.gotmpl       # Chart 值模板
└── templates/
    ├── deployment.yaml           # FlyIAM 应用
    ├── service.yaml
    ├── secret.yaml
    ├── serviceaccount.yaml
    ├── hpa.yaml
    ├── casdoor-configmap.yaml    # Casdoor app.conf 模板
    ├── casdoor-deployment.yaml   # 内置 Casdoor
    ├── casdoor-service.yaml
    └── NOTES.txt
```

## 多环境

| 环境 | 命名空间 | 副本数 | Casdoor |
|------|---------|--------|---------|
| default | flyiam | 2 | 2 |
| dev | flyiam-dev | 1 | 1 |
| staging | flyiam-staging | 2 | 2 |
| prod | flyiam | 3 | 2 |

> ⚠️ Pod 反亲和为**硬性打散**（`required`），带 `flyiam=true` 标签的节点数需
> **≥ 应用副本数 + Casdoor 副本数**（prod 为 `3 + 2 = 5`）。
> 启用 HPA 时上限同样受节点数限制（`maxReplicas` 过大将出现 Pending）。

> ⚠️ Casdoor 多副本**必须共享会话**（默认已开启 Redis 会话，会话 DB 为 `1`），
> 否则登录会出现 `Unauthorized operation`。关闭 `CASDOOR_REDIS_SESSION_ENABLED`
> 时 `CASDOOR_REPLICAS` 必须为 `1`（前置检查会强制校验）。

## 关键配置

`values/_base.yaml.gotmpl` 集中管理（全部支持环境变量覆盖）：

| 变量 | 说明 | 默认 |
|------|------|------|
| `FLYIAM_NAMESPACE` | 命名空间 | `flyiam` |
| `FLYIAM_REPLICAS` | 应用副本数 | `2` |
| `FLYIAM_IMAGE_REGISTRY` | 应用镜像仓库（华为云，国内可访问） | `swr.cn-east-3.myhuaweicloud.com` |
| `FLYIAM_IMAGE_REPOSITORY` | 应用镜像路径 | `danxiaonuo/flyiam` |
| `FLYIAM_IMAGE_TAG` | 应用镜像标签 | `latest` |
| `FLYIAM_IMAGE_PULL_POLICY` | 应用镜像拉取策略 | `Always` |
| `FLYIAM_NODE_LABEL` / `FLYIAM_NODE_LABEL_VALUE` | 硬性节点亲和性标签 | `flyiam` / `true` |
| `FLYIAM_DB_HOST` / `FLYIAM_DB_PASSWORD` | 数据库 | - |
| `FLYIAM_REDIS_HOST` / `FLYIAM_REDIS_PASSWORD` | 缓存 | - |
| `FLYIAM_JWT_SECRET` | JWT 密钥 | - |
| `FLYIAM_ADMIN_PASSWORD` | 管理员密码 | - |
| `FLYIAM_CASDOOR_ENDPOINT` | Casdoor 后端地址（集群内） | `http://casdoor:8000` |
| `FLYIAM_CASDOOR_PUBLIC_ENDPOINT` | Casdoor 浏览器地址（外置域名，标准端口 80/443） | - |
| `FLYIAM_CASDOOR_AUTO_SETUP` | 自动初始化 Casdoor | `true` |
| `FLYIAM_CASDOOR_DEFAULT_PASSWORD` | 新用户默认密码 | `ysyh!9Sky` |
| `CASDOOR_ENABLED` | 部署内置 Casdoor | `true` |
| `CASDOOR_REPLICAS` | Casdoor 副本数 | `2` |
| `CASDOOR_REDIS_SESSION_ENABLED` | Casdoor 会话是否用 Redis 共享（多副本必须开启） | 跟随 `FLYIAM_REDIS_ENABLED`（默认 `true`） |
| `CASDOOR_REDIS_DB` | Casdoor 会话 Redis 数据库编号（地址/端口/密码复用 `FLYIAM_REDIS_*`） | `1` |
| `CASDOOR_IMAGE_TAG` | Casdoor 镜像标签 | `latest` |
| `CASDOOR_IMAGE_PULL_POLICY` | Casdoor 镜像拉取策略 | `Always` |
| `CASDOOR_INIT_IMAGE_TAG` | Casdoor initContainer 镜像标签（生成 app.conf） | `1.36` |
| `CASDOOR_TIMEZONE` | Casdoor 时区 | `Asia/Shanghai` |

> Chart 仅暴露 ClusterIP Service，不包含 Ingress；域名/HTTPS 请在集群入口层（Ingress Controller / Gateway）统一配置。

### 外置域名访问 Casdoor

集群入口以域名暴露 Casdoor 时，`FLYIAM_CASDOOR_PUBLIC_ENDPOINT` 填写域名即可，
**默认走标准端口，无需显式书写端口号**：

```
https://casdoor.example.com   → 443
http://casdoor.example.com    → 80
```

非标准端口才需写成 `https://casdoor.example.com:8443`。该地址同时写入内置 Casdoor 的
`origin`（app.conf），用于生成正确的登录跳转地址。

## 节点亲和性配置

FlyIAM 应用与内置 Casdoor 均配置了**硬性节点亲和性**，必须调度到带
`flyiam=true` 标签的 Linux 节点。部署前需为目标节点打标签：

```bash
# 查看节点
kubectl get nodes

# 为节点打标签
kubectl label nodes <node-1> flyiam=true
kubectl label nodes <node-2> flyiam=true

# 确认标签
kubectl get nodes -l flyiam=true
```

渲染后的亲和性规则：

```yaml
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
        - matchExpressions:
            - key: flyiam        # nodeLabel
              operator: In
              values:
                - "true"          # nodeLabelValue
            - key: kubernetes.io/os
              operator: In
              values:
                - linux
  podAntiAffinity:            # 硬性打散：同一 release 的 Pod 不共节点
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
            - key: app.kubernetes.io/name
              operator: In
              values:
                - flyiam
        topologyKey: kubernetes.io/hostname
```

标签由 `FLYIAM_NODE_LABEL` / `FLYIAM_NODE_LABEL_VALUE` 控制（默认 `flyiam` / `true`）：

```bash
export FLYIAM_NODE_LABEL="flyiam"
export FLYIAM_NODE_LABEL_VALUE="true"
```

> ⚠️ **硬性调度**：节点数不满足时 Pod 会一直 `Pending`，可用
> `kubectl describe pod -n flyiam <pod>` 查看调度事件。
>
> 打散为硬性（`required`），**带 `flyiam=true` 标签的节点数需 ≥ 4**
> （应用默认 2 副本 + 内置 Casdoor 默认 2 副本，两者共用
> `app.kubernetes.io/name=flyiam` 标签、互相排斥）。
> 节点不足时可下调副本数：`FLYIAM_REPLICAS` / `CASDOOR_REPLICAS`。

## 安装后验证

```bash
kubectl get pods -n flyiam
kubectl logs -n flyiam deploy/flyiam | grep -E "初始化|Casdoor"
```

预期：

```
✅ 数据库初始化完成
✅ Casdoor 组织已创建: flyiam
✅ Casdoor 应用已创建: flyiam (clientId=...)
✅ Casdoor 管理员账号已创建: flyiam/admin
✅ Casdoor 自动初始化完成
```

## 使用 existingSecret（推荐生产环境）

```bash
kubectl create secret generic flyiam-secret \
  --from-literal=DB_PASSWORD='...' \
  --from-literal=REDIS_PASSWORD='...' \
  --from-literal=JWT_SECRET='...' \
  --from-literal=ADMIN_PASSWORD='...' \
  -n flyiam

export FLYIAM_EXISTING_SECRET="flyiam-secret"
helmfile sync
```

> Secret 需包含 key：`DB_PASSWORD`、`REDIS_PASSWORD`、`JWT_SECRET`、`ADMIN_PASSWORD`。
> 若应用凭据由自动初始化生成，则无需在 Secret 中提供 `CASDOOR_CLIENT_ID/SECRET`。

## 运维操作

统一在 `charts/flyiam` 目录执行：

```bash
# 安装部署
helmfile -f helmfile.yaml.gotmpl sync

# 更新（修改配置/镜像后重新同步）
helmfile -f helmfile.yaml.gotmpl diff     # 查看变更
helmfile -f helmfile.yaml.gotmpl sync     # 应用变更

# 指定环境
helmfile -f helmfile.yaml.gotmpl -e prod sync

# 卸载
helmfile -f helmfile.yaml.gotmpl destroy
```

## 故障排查

| 现象 | 处理 |
|------|------|
| Pod Pending（节点亲和性不满足） | `kubectl get nodes -l flyiam=true` 确认节点已打标签，或调整 `FLYIAM_NODE_LABEL` |
| Pod CrashLoopBackOff | `kubectl logs` 查看；确认数据库可达、`JWT_SECRET` 与 `ADMIN_PASSWORD` 已设置 |
| Casdoor 未就绪 | FlyIAM 启动时会等待 Casdoor（最多 60s），确认 Casdoor Pod 正常 |
| 登录报 Redirect URI 错误 | 开启 `FLYIAM_CASDOOR_AUTO_REDIRECT_URI=true` 或手工加入白名单 |
| 浏览器跳转 localhost | 设置 `FLYIAM_CASDOOR_PUBLIC_ENDPOINT` 为浏览器可达的外置域名（标准端口 80/443） |

## 更多文档

- [Kubernetes 部署](../docs/deployment/kubernetes.md)
- [Casdoor 配置](../docs/deployment/casdoor.md)
- [架构设计](../docs/design/architecture.md)
