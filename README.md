# FlyIAM

> 统一用户管理系统 — 用户数据统一存储于 Casdoor，支持多数据源自动同步，Go (go-zero) + Vue 3 + PostgreSQL，单二进制交付。

[![Release](https://img.shields.io/github/v/release/iflyelf/flyiam)](https://github.com/iflyelf/flyiam/releases/latest)
[![License](https://img.shields.io/github/license/iflyelf/flyiam)](LICENSE)

## 简介

FlyIAM 面向企业统一身份治理，提供「用户 + 组织 + 权限」的集中管理能力：

- **不落地存储用户**：用户唯一存储于 Casdoor，FlyIAM 通过 Casdoor 完成认证与用户管理；
- **多数据源同步**：可从人员 API（插件式扩展）定期同步用户，自动更新、离职删除；
- **统一对外服务**：其他系统（如 Consul Manager）通过 FlyIAM 的 API 获取用户与组织信息；
- **零手工初始化**：自动建表、自动配置 Casdoor，部署只需一个进程、一个端口。

## 功能特性

**用户管理**
- 列表 / 搜索（服务端分页）、新增 / 编辑 / 删除、批量删除
- 密码重置（默认密码）与自定义密码
- 管理员标记、受保护用户（禁止误删）

**组织与权限**
- 角色（权限集合）、团队（权限分配主体）、成员与角色授权
- 权限格式 `<资源>:<动作>`，超级管理员或团队角色授权
- 认证复用 Casdoor，自动初始化组织 / 应用 / 管理员

**数据同步**
- 数据源页面化配置（HTTP API，可扩展 LDAP / MySQL）
- 定时自动同步（可配置间隔）与手动触发
- 增量更新（含手机号等字段）、离职自动删除（跳过受保护用户）
- 同步进度实时展示、同步日志与可中断恢复

**认证与安全**
- Casdoor OAuth2 登录，支持跨域名 / IP 部署
- JWT 鉴权 + 权限中间件、操作审计日志
- 回调地址自动追加白名单

**界面**
- 三套主题：🌞 暖沙米 / 🌊 冷蓝 / 🌙 暗黑
- Mac 圆角风格，H5 自适应

## 快速开始

### Docker（推荐）

```bash
git clone https://github.com/iflyelf/flyiam.git
cd flyiam
docker compose up -d

# 应用：http://localhost:8081
# Casdoor：http://localhost:8000
# Casdoor 内置管理员：admin / 123（上游硬编码，与 CASDOOR_DEFAULT_PASSWORD 无关）
# FlyIAM 业务管理员：flyiam/admin / ysyh!9Sky（密码取 CASDOOR_DEFAULT_PASSWORD）
```

### Kubernetes

```bash
cd charts/flyiam

export FLYIAM_DB_HOST="postgres.default.svc.cluster.local"
export FLYIAM_DB_PASSWORD="your-db-password"
export FLYIAM_JWT_SECRET="your-jwt-secret-at-least-32-chars"
export FLYIAM_ADMIN_PASSWORD="your-admin-password"

helmfile sync
```

详见 [Kubernetes 部署](docs/deployment/kubernetes.md)。

### 本地开发

```bash
# 本地配置（含真实凭据，已被 .gitignore 忽略）
cp etc/config.yaml etc/config-local.yaml

# 后端
go run ./cmd/api -c etc/config-local.yaml

# 前端（开发模式）
cd web && npm install && npm run dev
```

## 配置

**零硬编码**：所有配置通过配置文件 + 环境变量注入，端口、地址、凭据均可自定义。

| 变量 | 说明 | 默认 |
|------|------|------|
| `SERVER_PORT` | 监听端口 | `8081` |
| `DATABASE_URL` | PostgreSQL 连接串（或分项 `DB_*`） | — |
| `JWT_SECRET` | JWT 密钥（必填，≥32 位） | — |
| `ADMIN_PASSWORD` | 管理员密码（必填） | — |
| `REDIS_ENABLED` / `REDIS_HOST` / `REDIS_PORT` / `REDIS_PASSWORD` | 缓存 | `true` / `localhost` / `6379` / 空 |
| `CASDOOR_ENDPOINT` | Casdoor 后端地址（集群内） | `http://casdoor:8000` |
| `CASDOOR_PUBLIC_ENDPOINT` | Casdoor 浏览器地址（外置域名，标准端口 80/443） | 空 |
| `CASDOOR_AUTO_SETUP` | 自动初始化 Casdoor | `true` |
| `CASDOOR_DEFAULT_PASSWORD` | 新用户默认密码 | `ysyh!9Sky` |
| `CASDOOR_COUNTRY_CODE` | 手机号区域 | `CN` |

完整清单见 [配置说明](etc/config.yaml) 与 [Kubernetes 部署](docs/deployment/kubernetes.md)。

## 文档

| 分类 | 文档 |
|------|------|
| **设计** | [架构设计](docs/design/architecture.md) · [数据库设计](docs/design/database.md) · [API 设计](docs/design/api.md) |
| **开发** | [开发文档](docs/development/development.md) |
| **测试** | [测试文档](docs/testing/testing.md) |
| **部署** | [Kubernetes](docs/deployment/kubernetes.md) · [Docker](docs/deployment/docker.md) · [Casdoor](docs/deployment/casdoor.md) · [GitHub Actions](docs/deployment/github-secrets.md) |

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go 1.26 · go-zero |
| 前端 | Vue 3 · Element Plus · Pinia · Vite |
| 存储 | PostgreSQL（业务 + Casdoor） · Redis（缓存） |
| 认证 | Casdoor（OAuth2 / RBAC） |

## 目录结构

```
cmd/api/              程序入口
internal/             后端源码（config/svc/handler/logic/middleware/model/pkg）
web/                  Vue 3 前端（构建产物嵌入二进制）
charts/flyiam/        Helm Chart（含内置 Casdoor）
deploy/               部署物料（sql / docker / nginx）
docs/                 文档（design / development / testing / deployment）
etc/                  示例配置
```

## 许可证

[MIT](LICENSE) © iflyelf
