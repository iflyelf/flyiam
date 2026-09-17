package svc

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"github.com/iflyelf/flyiam/internal/config"
	"github.com/iflyelf/flyiam/internal/logic/protected"
	"github.com/iflyelf/flyiam/internal/logic/userfield"
	"github.com/iflyelf/flyiam/internal/pkg/cache"
	"github.com/iflyelf/flyiam/internal/pkg/casdoor"
)

// ServiceContext 服务上下文
type ServiceContext struct {
	Config         config.Config
	DB             sqlx.SqlConn
	CasdoorClient  *casdoor.Client
	Cache          *cache.Cache
	ProtectedLogic *protected.Logic
}

// NewServiceContext 创建服务上下文
func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化数据库连接
	db := initDB(c)

	// 初始化数据库表（自动创建表结构）
	if err := initSchema(db, c); err != nil {
		log.Fatalf("❌ 初始化数据库失败: %v", err)
	}

	// 初始化 Casdoor 客户端（含自动初始化组织/应用/管理员）
	casdoorClient, err := initCasdoorClient(db, c)
	if err != nil {
		log.Fatalf("❌ 初始化 Casdoor 客户端失败: %v", err)
	}

	// 从数据库加载受保护用户（表为空时用配置文件种子初始化）
	protectedLogic := protected.NewLogic(sqlx.NewSqlConnFromDB(db))
	seedCtx, seedCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := protectedLogic.SeedIfEmpty(seedCtx, c.Casdoor.ProtectedUsers); err != nil {
		log.Printf("⚠️ 初始化受保护用户失败: %v", err)
	}
	if accounts, err := protectedLogic.Accounts(seedCtx); err == nil {
		casdoorClient.SetProtected(accounts)
		log.Printf("🛡️  受保护用户已加载: %v", accounts)
	}
	seedCancel()

	// 初始化 Redis 缓存（连接失败自动降级，不影响启动）
	cacheClient := cache.New(cache.Config{
		Enabled:  c.Redis.Enabled,
		Host:     c.Redis.Host,
		Port:     c.Redis.Port,
		Password: c.Redis.Password,
		DB:       c.Redis.DB,
		TTL:      c.Redis.TTL,
	})

	return &ServiceContext{
		Config:         c,
		DB:             sqlx.NewSqlConnFromDB(db),
		CasdoorClient:  casdoorClient,
		Cache:          cacheClient,
		ProtectedLogic: protectedLogic,
	}
}

// initDB 初始化数据库连接
func initDB(c config.Config) *sql.DB {
	dsn := c.GetDSN()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("❌ 连接数据库失败: %v", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(c.Database.MaxOpenConns)
	db.SetMaxIdleConns(c.Database.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(c.Database.ConnMaxLifetime) * time.Second)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("❌ 数据库连接测试失败: %v", err)
	}

	log.Printf("✅ 数据库连接成功: %s:%d/%s", c.Database.Host, c.Database.Port, c.Database.DBName)
	return db
}

// initSchema 初始化数据库表结构
func initSchema(db *sql.DB, c config.Config) error {
	log.Println("🔧 初始化数据库表结构...")

	// 本系统不存储任何用户数据（用户唯一存储于 Casdoor）。
	// 清理历史版本遗留的本地用户相关表，避免数据双写与不一致。
	dropLegacy := `
	DROP TABLE IF EXISTS users;
	DROP TABLE IF EXISTS resigned_users;
	DROP TABLE IF EXISTS departments;
	`
	if _, err := db.Exec(dropLegacy); err != nil {
		return fmt.Errorf("清理本地用户表失败: %w", err)
	}

	// 创建同步日志表
	if err := createSyncLogsTable(db); err != nil {
		return fmt.Errorf("创建同步日志表失败: %w", err)
	}

	// 创建审计日志表
	if err := createAuditLogsTable(db); err != nil {
		return fmt.Errorf("创建审计日志表失败: %w", err)
	}

	// 创建数据源配置表
	if err := createDataSourceConfigsTable(db); err != nil {
		return fmt.Errorf("创建数据源配置表失败: %w", err)
	}

	// 创建定时任务配置表
	if err := createScheduleConfigTable(db); err != nil {
		return fmt.Errorf("创建定时任务配置表失败: %w", err)
	}

	// 创建受保护用户表
	if err := createProtectedUsersTable(db); err != nil {
		return fmt.Errorf("创建受保护用户表失败: %w", err)
	}

	// 创建用户字段定义表，并写入内置字段（幂等）
	if err := createUserFieldDefsTable(db); err != nil {
		return fmt.Errorf("创建用户字段定义表失败: %w", err)
	}
	if err := userfield.NewLogic(sqlx.NewSqlConnFromDB(db)).SeedBuiltin(context.Background()); err != nil {
		log.Printf("⚠️ 初始化内置用户字段失败: %v", err)
	}

	// 创建角色 / 团队 / 成员 / 团队角色表
	if err := createRBACTables(db); err != nil {
		return fmt.Errorf("创建权限相关表失败: %w", err)
	}

	// 迁移历史 timestamp 列为 timestamptz（修复时间显示偏移）
	if err := migrateTimestamps(db); err != nil {
		return fmt.Errorf("迁移时间字段失败: %w", err)
	}

	// 清理因进程重启而残留的“运行中”同步记录
	if _, err := db.Exec(`
		UPDATE sync_logs
		SET status = 'failed',
		    error_message = '服务重启导致任务中断',
		    completed_at = NOW()
		WHERE status = 'running'
	`); err != nil {
		log.Printf("⚠️ 清理残留同步记录失败: %v", err)
	}

	log.Println("✅ 数据库初始化完成")
	return nil
}

// migrateTimestamps 将 public schema 下遗留的 `timestamp without time zone`
// 列迁移为 `timestamptz`。
//
// 背景：旧版本使用 timestamp（无时区）存储本地墙钟时间，Go pq 驱动按 UTC
// 读取后会在 JSON 中带 Z 后缀，浏览器再次转换导致时间偏移 8 小时。
// 迁移时按 Asia/Shanghai 解释旧值，得到正确的时间点。该操作幂等。
func migrateTimestamps(db *sql.DB) error {
	// 仅迁移本系统自有表，避免影响 Casdoor / Casbin 的存储语义
	rows, err := db.Query(`
		SELECT table_name, column_name
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND data_type = 'timestamp without time zone'
		  AND table_name IN (
			'sync_logs', 'datasource_configs', 'schedule_config',
			'protected_users', 'roles', 'teams', 'team_members', 'team_roles', 'audit_logs'
		  )
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type col struct{ table, column string }
	var cols []col
	for rows.Next() {
		var c col
		if err := rows.Scan(&c.table, &c.column); err != nil {
			return err
		}
		cols = append(cols, c)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, c := range cols {
		stmt := fmt.Sprintf(
			`ALTER TABLE %s ALTER COLUMN %s TYPE timestamptz USING %s AT TIME ZONE 'Asia/Shanghai'`,
			pq.QuoteIdentifier(c.table), pq.QuoteIdentifier(c.column), pq.QuoteIdentifier(c.column),
		)
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("迁移 %s.%s 失败: %w", c.table, c.column, err)
		}
		log.Printf("🕒 已迁移时间字段: %s.%s -> timestamptz", c.table, c.column)
	}
	return nil
}

// createRBACTables 创建角色与团队相关表
func createRBACTables(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS roles (
		id BIGSERIAL PRIMARY KEY,
		name VARCHAR(100) NOT NULL UNIQUE,
		code VARCHAR(100) NOT NULL DEFAULT '',
		description TEXT,
		permissions TEXT[] DEFAULT '{}',
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS teams (
		id BIGSERIAL PRIMARY KEY,
		name VARCHAR(100) NOT NULL UNIQUE,
		code VARCHAR(100) NOT NULL DEFAULT '',
		description TEXT,
		status SMALLINT DEFAULT 1,
		created_by VARCHAR(100),
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS team_members (
		id BIGSERIAL PRIMARY KEY,
		team_id BIGINT NOT NULL,
		username VARCHAR(100) NOT NULL,
		display_name VARCHAR(200),
		created_at TIMESTAMPTZ DEFAULT NOW(),
		FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
		UNIQUE(team_id, username)
	);

	CREATE TABLE IF NOT EXISTS team_roles (
		id BIGSERIAL PRIMARY KEY,
		team_id BIGINT NOT NULL,
		role_id BIGINT NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
		FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
		UNIQUE(team_id, role_id)
	);

	CREATE INDEX IF NOT EXISTS idx_team_members_username ON team_members(username);
	CREATE INDEX IF NOT EXISTS idx_team_roles_team ON team_roles(team_id);
	`
	_, err := db.Exec(schema)
	return err
}

// createProtectedUsersTable 创建受保护用户表
func createProtectedUsersTable(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS protected_users (
		id BIGSERIAL PRIMARY KEY,
		domain_account VARCHAR(100) NOT NULL UNIQUE,
		remark VARCHAR(255),
		created_at TIMESTAMPTZ DEFAULT NOW()
	);
	`
	_, err := db.Exec(schema)
	return err
}

// createUserFieldDefsTable 创建用户字段定义表
//
// 表为空时由 SeedBuiltin 写入内置人事字段（empCode/deptNameLv0...），
// 之后管理员可在页面增删自定义字段、调整显示名与可见性，无需改代码。
func createUserFieldDefsTable(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS user_field_defs (
		id BIGSERIAL PRIMARY KEY,
		field_key VARCHAR(64) NOT NULL UNIQUE,
		label VARCHAR(128) NOT NULL,
		field_type VARCHAR(32) NOT NULL DEFAULT 'text',
		options TEXT DEFAULT '',
		show_in_list BOOLEAN NOT NULL DEFAULT TRUE,
		show_in_form BOOLEAN NOT NULL DEFAULT TRUE,
		editable BOOLEAN NOT NULL DEFAULT TRUE,
		builtin BOOLEAN NOT NULL DEFAULT FALSE,
		sort_order INT NOT NULL DEFAULT 0,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);
	`
	_, err := db.Exec(schema)
	return err
}

// createScheduleConfigTable 创建定时任务配置表（单例）
func createScheduleConfigTable(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS schedule_config (
		id BIGSERIAL PRIMARY KEY,
		enabled BOOLEAN NOT NULL DEFAULT FALSE,
		run_interval VARCHAR(20) NOT NULL DEFAULT '6h',
		sync_datasource BOOLEAN NOT NULL DEFAULT TRUE,
		delete_missing BOOLEAN NOT NULL DEFAULT TRUE,
		casdoor_batch_size INT NOT NULL DEFAULT 10,
		last_run_at TIMESTAMPTZ,
		last_run_status VARCHAR(20),
		last_run_message TEXT,
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);
	ALTER TABLE schedule_config ADD COLUMN IF NOT EXISTS delete_missing BOOLEAN NOT NULL DEFAULT TRUE;
	ALTER TABLE schedule_config ADD COLUMN IF NOT EXISTS casdoor_batch_size INT NOT NULL DEFAULT 10;
	INSERT INTO schedule_config (id, enabled, run_interval) VALUES (1, FALSE, '6h') ON CONFLICT (id) DO NOTHING;
	`
	_, err := db.Exec(schema)
	return err
}

// createDataSourceConfigsTable 创建数据源配置表
func createDataSourceConfigsTable(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS datasource_configs (
		id BIGSERIAL PRIMARY KEY,
		name VARCHAR(100) NOT NULL UNIQUE,
		type VARCHAR(20) NOT NULL DEFAULT 'httpapi',
		enabled BOOLEAN NOT NULL DEFAULT TRUE,
		url VARCHAR(500) NOT NULL,
		method VARCHAR(10) NOT NULL DEFAULT 'POST',
		auth_type VARCHAR(20) NOT NULL DEFAULT 'none',
		auth_token VARCHAR(500),
		auth_username VARCHAR(100),
		auth_password VARCHAR(200),
		timeout INT NOT NULL DEFAULT 30,
		sync_interval VARCHAR(20) NOT NULL DEFAULT '6h',
		auto_sync BOOLEAN NOT NULL DEFAULT TRUE,
		priority INT NOT NULL DEFAULT 90,
		page_size INT NOT NULL DEFAULT 1000,
		field_mapping TEXT DEFAULT '',
		remark VARCHAR(255),
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_datasource_configs_enabled ON datasource_configs(enabled);
	-- 兼容旧表：补齐字段映射列（数据源字段变化无需改代码）
	ALTER TABLE datasource_configs ADD COLUMN IF NOT EXISTS field_mapping TEXT DEFAULT '';
	`
	_, err := db.Exec(schema)
	return err
}

// createSyncLogsTable 创建同步日志表
func createSyncLogsTable(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS sync_logs (
		id BIGSERIAL PRIMARY KEY,
		sync_type VARCHAR(20) NOT NULL,
		data_source VARCHAR(20),
		status VARCHAR(20) NOT NULL,
		total_count INT DEFAULT 0,
		success_count INT DEFAULT 0,
		failed_count INT DEFAULT 0,
		new_count INT DEFAULT 0,
		updated_count INT DEFAULT 0,
		deleted_count INT DEFAULT 0,
		error_message TEXT,
		details JSONB,
		duration_ms INT,
		triggered_by VARCHAR(50),
		started_at TIMESTAMPTZ DEFAULT NOW(),
		completed_at TIMESTAMPTZ
	);
	CREATE INDEX IF NOT EXISTS idx_sync_logs_type ON sync_logs(sync_type);
	CREATE INDEX IF NOT EXISTS idx_sync_logs_started ON sync_logs(started_at DESC);
	`
	_, err := db.Exec(schema)
	return err
}

// createAuditLogsTable 创建审计日志表
func createAuditLogsTable(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS audit_logs (
		id BIGSERIAL PRIMARY KEY,
		user_id VARCHAR(100),
		username VARCHAR(100),
		action VARCHAR(50),
		resource_type VARCHAR(50),
		resource_id VARCHAR(100),
		resource_name VARCHAR(255),
		details JSONB,
		ip_address VARCHAR(50),
		user_agent TEXT,
		status VARCHAR(20),
		error_message TEXT,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at DESC);
	`
	_, err := db.Exec(schema)
	return err
}

// initCasdoorClient 初始化 Casdoor 客户端
//
// 流程：
//  1. 组装 Casdoor 配置（回调地址缺失时用 PublicEndpoint/Endpoint 兜底生成）
//  2. 若开启 AutoSetup 且服务可用，自动创建/补齐组织、应用、管理员，并回填应用凭据
//  3. 使用最终凭据创建客户端并做健康检查
func initCasdoorClient(db *sql.DB, c config.Config) (*casdoor.Client, error) {
	cfg := &casdoor.Config{
		Endpoint:                c.Casdoor.Endpoint,
		PublicEndpoint:          c.Casdoor.PublicEndpoint,
		ClientId:                c.Casdoor.ClientId,
		ClientSecret:            c.Casdoor.ClientSecret,
		Certificate:             c.Casdoor.Certificate,
		OrganizationName:        c.Casdoor.OrganizationName,
		OrganizationDisplayName: c.Casdoor.OrganizationDisplayName,
		ApplicationName:         c.Casdoor.ApplicationName,
		ApplicationDisplayName:  c.Casdoor.ApplicationDisplayName,
		DefaultPassword:         c.Casdoor.DefaultPassword,
		CountryCode:             c.Casdoor.CountryCode,
		UserCacheTTL:            c.Casdoor.UserCacheTTL,
		ProtectedUsers:          c.Casdoor.ProtectedUsers,
		AutoRedirectURI:         c.Casdoor.AutoRedirectURI,
		RedirectURIs:            c.Casdoor.RedirectURIs,
	}

	// 回调地址白名单：仅使用显式配置。
	// 未配置时不预置任何地址，登录时会根据当前访问域名自动追加（AutoRedirectURI），
	// 避免写入错误的地址（例如误用 Casdoor 自身地址）。
	if len(cfg.RedirectURIs) == 0 && !cfg.AutoRedirectURI {
		log.Printf("⚠️ 未配置 Casdoor 回调地址且未开启自动追加，登录可能失败")
	}

	// 自动初始化 Casdoor（组织/应用/管理员），并回填应用凭据
	if c.Casdoor.AutoSetup {
		if err := waitForCasdoor(cfg.Endpoint, 60*time.Second); err != nil {
			log.Printf("⚠️ 等待 Casdoor 就绪超时（继续尝试初始化）: %v", err)
		}
		if res, err := casdoor.EnsureSetup(db, cfg); err != nil {
			log.Printf("⚠️ Casdoor 自动初始化失败（请检查配置）: %v", err)
		} else {
			cfg.ClientId = res.ClientId
			cfg.ClientSecret = res.ClientSecret
			log.Printf("✅ Casdoor 自动初始化完成（组织=%s 应用=%s clientId=%s）",
				res.Organization, res.Application, res.ClientId)
		}
	}

	if cfg.ClientId == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("Casdoor 应用凭据缺失：请开启 CASDOOR_AUTO_SETUP 或在配置中提供 CASDOOR_CLIENT_ID/SECRET")
	}

	// 读取内置应用（app-built-in）凭据，用于「应用 / 组织管理」等全局接口。
	// Casdoor 权限模型中只有 built-in 身份是全局管理员，业务应用凭据会报
	// "Unauthorized operation" / "Please sign in first"。
	if id, secret, err := casdoor.ReadBuiltinApp(db); err != nil {
		log.Printf("⚠️ 读取 Casdoor 内置应用凭据失败（应用/组织管理功能暂不可用）: %v", err)
	} else {
		cfg.AdminClientId = id
		cfg.AdminClientSecret = secret
	}

	client, err := casdoor.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("创建 Casdoor 客户端失败: %w", err)
	}

	// 注入补读方式 + 后台自动重试：Casdoor 首次建表可能晚于本服务，
	// 这样无需人工重启即可自动恢复（每 10s 重试，最多 1 小时）。
	client.SetAdminCredentialLoader(func() (string, string, error) {
		return casdoor.ReadBuiltinApp(db)
	})
	client.WatchAdminCredentials(10*time.Second, 360)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.HealthCheck(ctx); err != nil {
		log.Printf("⚠️ Casdoor 连接测试失败（将继续运行）: %v", err)
	} else {
		log.Printf("✅ Casdoor 客户端初始化成功: %s", c.Casdoor.Endpoint)
	}

	return client, nil
}

// waitForCasdoor 轮询等待 Casdoor 就绪（容器编排下 Casdoor 可能晚于本服务启动）
func waitForCasdoor(endpoint string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if casdoor.Ping(endpoint) {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("Casdoor 未在 %s 内就绪", timeout)
}
