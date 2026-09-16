# 数据库设计

FlyIAM 使用 **PostgreSQL**，所有表由程序自动创建（无需手工执行 SQL）。
数据库采用「一库两域」设计：同库内包含 **Casdoor 表**（前缀 `casdoor_`）与 **FlyIAM 业务表**。

## 1. 数据库准备

程序**不自动创建库**，需管理员预先创建空库：

```sql
CREATE DATABASE flyiam;
-- 如需独立账号
CREATE USER flyiam WITH PASSWORD 'your-password';
GRANT ALL PRIVILEGES ON DATABASE flyiam TO flyiam;
```

> 建库完成后，启动 FlyIAM 与 Casdoor 即会自动建表。

## 2. 表清单

### 2.1 Casdoor 表（前缀 `casdoor_`，由 Casdoor 自动创建）

共 44 张表，核心包括：`casdoor_user`（用户）、`casdoor_organization`（组织）、
`casdoor_application`（应用）、`casdoor_role`（角色）、`casdoor_permission`（权限）、
`casdoor_cert`（证书）等。**用户数据全部存储于此**。

### 2.2 FlyIAM 业务表

| 表 | 说明 |
|----|------|
| `datasource_configs` | 数据源配置（页面可维护） |
| `schedule_config` | 定时同步配置（单例，id=1） |
| `sync_logs` | 同步日志（含进度、结果、详情） |
| `protected_users` | 受保护用户（同步/删除时跳过） |
| `roles` | 角色（权限集合） |
| `teams` | 团队（权限分配主体） |
| `team_members` | 团队成员 |
| `team_roles` | 团队-角色授权 |
| `audit_logs` | 审计日志 |

## 3. 核心表结构

### 3.1 数据源配置 datasource_configs

```sql
CREATE TABLE datasource_configs (
    id             BIGSERIAL PRIMARY KEY,
    name           VARCHAR(100) NOT NULL UNIQUE,   -- 数据源名称
    type           VARCHAR(20)  NOT NULL DEFAULT 'httpapi',
    enabled        BOOLEAN      NOT NULL DEFAULT TRUE,
    url            VARCHAR(500) NOT NULL,          -- 接口地址
    method         VARCHAR(10)  NOT NULL DEFAULT 'POST',
    auth_type      VARCHAR(20)  NOT NULL DEFAULT 'none',
    auth_token     VARCHAR(500),
    auth_username  VARCHAR(100),
    auth_password  VARCHAR(200),
    timeout        INT          NOT NULL DEFAULT 30,
    sync_interval  VARCHAR(20)  NOT NULL DEFAULT '6h',
    auto_sync      BOOLEAN      NOT NULL DEFAULT TRUE,
    priority       INT          NOT NULL DEFAULT 90,
    page_size      INT          NOT NULL DEFAULT 1000,
    remark         VARCHAR(255),
    created_at     TIMESTAMPTZ DEFAULT NOW(),
    updated_at     TIMESTAMPTZ DEFAULT NOW()
);
```

### 3.2 定时任务配置 schedule_config（单例）

```sql
CREATE TABLE schedule_config (
    id               BIGSERIAL PRIMARY KEY,
    enabled          BOOLEAN NOT NULL DEFAULT FALSE,  -- 总开关
    run_interval     VARCHAR(20) NOT NULL DEFAULT '6h',
    sync_datasource  BOOLEAN NOT NULL DEFAULT TRUE,
    delete_missing   BOOLEAN NOT NULL DEFAULT TRUE,   -- 删除数据源中不存在的人员
    casdoor_batch_size INT   NOT NULL DEFAULT 10,     -- 并发数
    last_run_at      TIMESTAMPTZ,
    last_run_status  VARCHAR(20),
    last_run_message TEXT,
    updated_at       TIMESTAMPTZ DEFAULT NOW()
);
```

### 3.3 同步日志 sync_logs

```sql
CREATE TABLE sync_logs (
    id           BIGSERIAL PRIMARY KEY,
    sync_type    VARCHAR(20) NOT NULL,   -- datasource / full
    data_source  VARCHAR(20),
    status       VARCHAR(20) NOT NULL,   -- running / success / failed
    total_count  INT DEFAULT 0,
    success_count INT DEFAULT 0,
    failed_count INT DEFAULT 0,
    new_count    INT DEFAULT 0,
    updated_count INT DEFAULT 0,
    deleted_count INT DEFAULT 0,
    error_message TEXT,
    details      JSONB,
    duration_ms  INT,
    triggered_by VARCHAR(50),           -- manual / auto / startup
    started_at   TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);
```

### 3.4 受保护用户 protected_users

```sql
CREATE TABLE protected_users (
    id             BIGSERIAL PRIMARY KEY,
    domain_account VARCHAR(100) NOT NULL UNIQUE,
    remark         VARCHAR(255),
    created_at     TIMESTAMPTZ DEFAULT NOW()
);
```

### 3.5 角色与团队

```sql
CREATE TABLE roles (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL UNIQUE,
    code        VARCHAR(100) NOT NULL DEFAULT '',
    description TEXT,
    permissions TEXT[] DEFAULT '{}',     -- 如 {user:read,user:write}
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE teams (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL UNIQUE,
    code        VARCHAR(100) NOT NULL DEFAULT '',
    description TEXT,
    status      SMALLINT DEFAULT 1,
    created_by  VARCHAR(100),
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE team_members (
    id           BIGSERIAL PRIMARY KEY,
    team_id      BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    username     VARCHAR(100) NOT NULL,   -- Casdoor 域账号
    display_name VARCHAR(200),
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(team_id, username)
);

CREATE TABLE team_roles (
    id         BIGSERIAL PRIMARY KEY,
    team_id    BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    role_id    BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(team_id, role_id)
);
```

### 3.6 审计日志 audit_logs

```sql
CREATE TABLE audit_logs (
    id            BIGSERIAL PRIMARY KEY,
    user_id       VARCHAR(100),
    username      VARCHAR(100),
    action        VARCHAR(50),
    resource_type VARCHAR(50),
    resource_id   VARCHAR(100),
    resource_name VARCHAR(255),
    details       JSONB,
    ip_address    VARCHAR(50),
    user_agent    TEXT,
    status        VARCHAR(20),
    error_message TEXT,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);
```

## 4. 设计约定

- **时间字段**统一使用 `TIMESTAMPTZ`（带时区），避免时区偏移；
- **数组字段**使用 PostgreSQL `TEXT[]`；
- 所有表定义均为 `CREATE TABLE IF NOT EXISTS`，可重复执行（幂等）；
- 启动时自动清理历史遗留的本地用户表（`users` / `departments` / `resigned_users`）；
- 启动时将残留 `running` 的同步日志标记为 `failed`（服务重启导致中断）。

## 5. 相关文件

- 建表逻辑：`internal/svc/service_context.go`
- 完整 DDL 参考：`deploy/sql/schema.sql`
