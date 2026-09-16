# 架构设计

FlyIAM 是**统一用户管理系统**：本系统不落地存储用户数据，用户唯一存储于 Casdoor；
其他系统（如 Consul Manager）通过本系统的对外 API 获取用户与组织信息。

## 1. 总体架构

```
                         ┌──────────────────────────────────────┐
                         │            FlyIAM（本系统）           │
   人员数据源(HTTP API)   │  ┌────────────────────────────────┐  │
        ────────────────►│  │ 数据源插件（HTTP API / LDAP…）  │  │
                         │  └───────────────┬────────────────┘  │
                         │                  │ 定时/手动同步     │
                         │                  ▼                   │
                         │  ┌────────────────────────────────┐  │
                         │  │ 同步引擎（并发 Upsert/删除）     │  │
                         │  └───────────────┬────────────────┘  │
                         └──────────────────┼───────────────────┘
                                            ▼
                         ┌──────────────────────────────────────┐
                         │             Casdoor（认证中心）        │
                         │  组织 / 应用 / 用户 / 角色 / 权限      │
                         └───────────────┬──────────────────────┘
                                         │ 共享 PostgreSQL（表前缀 casdoor_）
                                         ▼
                         ┌──────────────────────────────────────┐
                         │        PostgreSQL（外置数据库）        │
                         │  flyiam 库：Casdoor 表 + 本系统业务表  │
                         └──────────────────────────────────────┘
                                         ▲
        Consul Manager 等外部系统 ────────┘
        （调用 FlyIAM 对外 API 获取用户）
```

## 2. 组件说明

| 组件 | 说明 |
|------|------|
| **FlyIAM API** | go-zero 单体服务，内置 Vue 3 前端（Go embed），单二进制交付 |
| **Casdoor** | 认证中心（OAuth2 / RBAC），可由本 Chart 一并部署 |
| **PostgreSQL** | 外置数据库；Casdoor 表（前缀 `casdoor_`）与业务表共用同一库 |
| **Redis** | 外置缓存（可选，不可用时自动降级） |
| **人员数据源** | 外部 HTTP API，通过插件方式接入，可扩展多数据源 |

## 3. 目录结构

```
flyiam/
├── cmd/api/                程序入口
├── internal/
│   ├── config/             配置结构（全量环境变量支持）
│   ├── svc/                服务上下文（DB / Casdoor / 缓存 / 数据源）
│   ├── handler/            HTTP 处理器与路由
│   ├── logic/              业务逻辑
│   │   ├── datasource/     数据源配置管理
│   │   ├── sync/           同步引擎（数据源 → Casdoor）
│   │   ├── scheduler/      定时任务调度
│   │   ├── rbac/           角色 / 团队 / 权限
│   │   ├── protected/      受保护用户
│   │   └── schedule/       定时配置
│   ├── middleware/         JWT 鉴权 + 权限校验
│   ├── model/              数据模型
│   └── pkg/
│       ├── casdoor/        Casdoor 客户端（含自动初始化）
│       ├── datasource/     数据源插件接口与实现
│       ├── cache/          Redis 封装
│       └── web/            前端资源嵌入（//go:embed）
├── web/                    Vue 3 前端源码
├── charts/flyiam/          Helm Chart（含内置 Casdoor）
├── deploy/                 部署物料（sql / docker / nginx）
├── docs/                   文档
└── etc/                    示例配置
```

## 4. 关键设计

### 4.1 零手工初始化

- **数据库**：程序启动时自动建表（`CREATE TABLE IF NOT EXISTS`），无需手工执行 SQL；
- **Casdoor**：首次启动自动创建组织、应用、管理员账号，并回填应用凭据（`CASDOOR_AUTO_SETUP=true`）；
- **建库**：需管理员预先创建空库 `flyiam`（参照 `deploy/sql/schema.sql` 顶部说明）。

### 4.2 数据源插件化

所有数据源实现统一接口（`internal/pkg/datasource/interface.go`），
通过数据库表 `datasource_configs` 在页面配置后动态装载，新增数据源只需实现接口。

### 4.3 用户唯一存储在 Casdoor

- 本系统**不建用户表**，所有用户数据存于 Casdoor；
- 业务表仅保存：数据源配置、定时任务、同步日志、角色、团队、受保护用户、审计日志。

### 4.4 权限模型

- **超级管理员**：配置项 `Permission.AdminUsers` 或 Casdoor `isAdmin`；
- **普通用户**：权限来自「团队 → 角色 → 权限」的聚合（`<资源>:<动作>`）；
- 中间件 `middleware.Auth` + `middleware.RequirePermission` 统一校验。

### 4.5 前端嵌入

前端构建产物输出到 `internal/pkg/web/dist`，通过 `//go:embed` 打包进二进制，
部署仅需一个进程、一个端口。

## 5. 请求链路

```
浏览器 ──► FlyIAM API ──► Casdoor（登录/OAuth2）
              │
              ├──► PostgreSQL（业务数据）
              └──► Redis（缓存）
外部系统 ──► FlyIAM 对外 API ──► Casdoor（用户/组织只读）
```
