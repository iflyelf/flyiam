-- =============================================================================
-- FlyIAM 数据库表结构（快照，请勿手工编辑）
--
-- ⚠️ 权威来源：程序启动时按代码内嵌 DDL 自动建表（CREATE TABLE IF NOT EXISTS）
--    并自动维护（加列/改类型/清理废弃表列），无需手工执行任何 SQL。
--
-- 重新生成本快照：
--     make schema            # 用 pg_dump 从运行中的数据库导出
--   或
--     pg_dump --schema-only "$DATABASE_URL" > deploy/sql/schema.sql
--
-- 说明：
--   1. Casdoor 表（前缀 casdoor_）由 Casdoor 首次启动自动创建，不在此列。
--   2. 数据库采用「一库两域」：同一库内包含 Casdoor 表与本系统业务表。
--   3. 数据库本身也会在首次启动时自动创建（程序连 postgres 库执行 CREATE DATABASE）。
-- =============================================================================

-- -----------------------------------------------------------------------------
-- 数据源配置（页面可维护，支持 HTTP API，可扩展 LDAP / MySQL）
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS datasource_configs (
    id             BIGSERIAL PRIMARY KEY,
    name           VARCHAR(100) NOT NULL UNIQUE,             -- 数据源名称
    type           VARCHAR(20)  NOT NULL DEFAULT 'httpapi',  -- 类型
    enabled        BOOLEAN      NOT NULL DEFAULT TRUE,       -- 是否启用
    url            VARCHAR(500) NOT NULL,                    -- 接口地址
    method         VARCHAR(10)  NOT NULL DEFAULT 'POST',     -- 请求方法
    auth_type      VARCHAR(20)  NOT NULL DEFAULT 'none',     -- 认证类型
    auth_token     VARCHAR(500),                             -- Token / API Key
    auth_username  VARCHAR(100),                             -- Basic 用户名
    auth_password  VARCHAR(200),                             -- Basic 密码
    timeout        INT          NOT NULL DEFAULT 30,         -- 超时（秒）
    sync_interval  VARCHAR(20)  NOT NULL DEFAULT '6h',       -- 同步间隔
    auto_sync      BOOLEAN      NOT NULL DEFAULT TRUE,       -- 启动/定时自动同步
    priority       INT          NOT NULL DEFAULT 90,         -- 数据优先级
    page_size      INT          NOT NULL DEFAULT 1000,       -- 分页大小
    remark         VARCHAR(255),                             -- 备注
    created_at     TIMESTAMPTZ  DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_datasource_configs_enabled ON datasource_configs(enabled);

-- -----------------------------------------------------------------------------
-- 定时任务配置（单例，id = 1）
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS schedule_config (
    id                 BIGSERIAL PRIMARY KEY,
    enabled            BOOLEAN     NOT NULL DEFAULT FALSE,  -- 定时任务总开关
    run_interval       VARCHAR(20) NOT NULL DEFAULT '6h',   -- 执行间隔
    sync_datasource    BOOLEAN     NOT NULL DEFAULT TRUE,   -- 从数据源同步
    delete_missing     BOOLEAN     NOT NULL DEFAULT TRUE,   -- 删除数据源中不存在的人员
    casdoor_batch_size INT         NOT NULL DEFAULT 10,     -- 并发批量大小
    last_run_at        TIMESTAMPTZ,                         -- 上次执行时间
    last_run_status    VARCHAR(20),                         -- 上次执行结果
    last_run_message   TEXT,                                -- 上次执行信息
    updated_at         TIMESTAMPTZ DEFAULT NOW()
);
INSERT INTO schedule_config (id, enabled, run_interval) VALUES (1, FALSE, '6h')
    ON CONFLICT (id) DO NOTHING;

-- -----------------------------------------------------------------------------
-- 同步日志
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS sync_logs (
    id            BIGSERIAL PRIMARY KEY,
    sync_type     VARCHAR(20) NOT NULL,                     -- datasource / full
    data_source   VARCHAR(20),                              -- 数据源名称
    status        VARCHAR(20) NOT NULL,                     -- running / success / failed
    total_count   INT DEFAULT 0,
    success_count INT DEFAULT 0,
    failed_count  INT DEFAULT 0,
    new_count     INT DEFAULT 0,
    updated_count INT DEFAULT 0,
    deleted_count INT DEFAULT 0,
    error_message TEXT,
    details       JSONB,                                    -- 详细信息
    duration_ms   INT,
    triggered_by  VARCHAR(50),                              -- manual / auto / startup
    started_at    TIMESTAMPTZ DEFAULT NOW(),
    completed_at  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_sync_logs_type ON sync_logs(sync_type);
CREATE INDEX IF NOT EXISTS idx_sync_logs_started ON sync_logs(started_at DESC);

-- -----------------------------------------------------------------------------
-- 受保护用户（同步与删除时跳过）
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS protected_users (
    id             BIGSERIAL PRIMARY KEY,
    domain_account VARCHAR(100) NOT NULL UNIQUE,            -- 域账号
    remark         VARCHAR(255),
    created_at     TIMESTAMPTZ DEFAULT NOW()
);

-- -----------------------------------------------------------------------------
-- 角色（权限集合）
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS roles (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL UNIQUE,               -- 角色名称
    code        VARCHAR(100) NOT NULL DEFAULT '',           -- 角色代码
    description TEXT,
    permissions TEXT[] DEFAULT '{}',                        -- 权限列表
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

-- -----------------------------------------------------------------------------
-- 团队（权限分配主体）
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS teams (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL UNIQUE,               -- 团队名称
    code        VARCHAR(100) NOT NULL DEFAULT '',           -- 团队代码
    description TEXT,
    status      SMALLINT DEFAULT 1,                         -- 1 启用 / 0 停用
    created_by  VARCHAR(100),
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

-- -----------------------------------------------------------------------------
-- 团队成员（username 为 Casdoor 域账号）
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS team_members (
    id           BIGSERIAL PRIMARY KEY,
    team_id      BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    username     VARCHAR(100) NOT NULL,
    display_name VARCHAR(200),
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(team_id, username)
);
CREATE INDEX IF NOT EXISTS idx_team_members_username ON team_members(username);

-- -----------------------------------------------------------------------------
-- 团队-角色授权
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS team_roles (
    id         BIGSERIAL PRIMARY KEY,
    team_id    BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    role_id    BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(team_id, role_id)
);
CREATE INDEX IF NOT EXISTS idx_team_roles_team ON team_roles(team_id);

-- -----------------------------------------------------------------------------
-- 审计日志
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS audit_logs (
    id            BIGSERIAL PRIMARY KEY,
    user_id       VARCHAR(100),
    username      VARCHAR(100),
    action        VARCHAR(50),                              -- create/update/delete/sync/login
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
CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at DESC);

-- =============================================================================
-- 说明：用户数据不存储于本库业务表，统一存储于 Casdoor（casdoor_user 表）。
-- =============================================================================
