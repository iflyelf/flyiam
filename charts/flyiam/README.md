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
| default | flyiam | 2 | 1 |
| dev | flyiam-dev | 1 | 1 |
| staging | flyiam-staging | 2 | 1 |
| prod | flyiam | 3 | 1 |

## 关键配置

`values/_base.yaml.gotmpl` 集中管理（全部支持环境变量覆盖）：

| 变量 | 说明 | 默认 |
|------|------|------|
| `FLYIAM_NAMESPACE` | 命名空间 | `flyiam` |
| `FLYIAM_REPLICAS` | 应用副本数 | `2` |
| `FLYIAM_IMAGE_TAG` | 应用镜像标签 | `latest` |
| `FLYIAM_DB_HOST` / `FLYIAM_DB_PASSWORD` | 数据库 | - |
| `FLYIAM_REDIS_HOST` / `FLYIAM_REDIS_PASSWORD` | 缓存 | - |
| `FLYIAM_JWT_SECRET` | JWT 密钥 | - |
| `FLYIAM_ADMIN_PASSWORD` | 管理员密码 | - |
| `FLYIAM_CASDOOR_ENDPOINT` | Casdoor 内网地址 | `http://casdoor:8000` |
| `FLYIAM_CASDOOR_PUBLIC_ENDPOINT` | Casdoor 浏览器地址 | - |
| `FLYIAM_CASDOOR_AUTO_SETUP` | 自动初始化 Casdoor | `true` |
| `FLYIAM_CASDOOR_DEFAULT_PASSWORD` | 新用户默认密码 | `ysyh!9Sky` |
| `CASDOOR_ENABLED` | 部署内置 Casdoor | `true` |
| `CASDOOR_IMAGE_TAG` | Casdoor 镜像标签 | `latest` |
| `CASDOOR_TIMEZONE` | Casdoor 时区 | `Asia/Shanghai` |

> Chart 仅暴露 ClusterIP Service，不包含 Ingress；域名/HTTPS 请在集群入口层（Ingress Controller / Gateway）统一配置。

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

## 故障排查

| 现象 | 处理 |
|------|------|
| Pod CrashLoopBackOff | `kubectl logs` 查看；确认数据库可达、`JWT_SECRET` 与 `ADMIN_PASSWORD` 已设置 |
| Casdoor 未就绪 | FlyIAM 启动时会等待 Casdoor（最多 60s），确认 Casdoor Pod 正常 |
| 登录报 Redirect URI 错误 | 开启 `FLYIAM_CASDOOR_AUTO_REDIRECT_URI=true` 或手工加入白名单 |
| 浏览器跳转 localhost | 设置 `FLYIAM_CASDOOR_PUBLIC_ENDPOINT` 为浏览器可达地址 |

## 更多文档

- [Kubernetes 部署](../docs/deployment/kubernetes.md)
- [Casdoor 配置](../docs/deployment/casdoor.md)
- [架构设计](../docs/design/architecture.md)
