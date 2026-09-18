// Package setting 应用设置：页面可配置，DB 优先 / env 兜底。
//
// 设计：
//   - 设置项以 key/value 存于 app_settings 表；
//   - 启动时把 DB 值应用到共享的 *config.Config（指针），使全量既有读取点自动生效；
//   - 页面修改后同样写回该 Config，无需重启（连接类配置见各字段说明）。
package setting

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"github.com/iflyelf/flyiam/internal/config"
)

// Item 可配置项定义
type Item struct {
	Key    string
	Group  string
	Label  string
	Type   string // string / int / bool / list / secret
	Secret bool
	Get    func(*config.Config) string
	Set    func(*config.Config, string)
}

// Registry 全部可页面配置项（按分组展示）
var Registry = []Item{
	// 安全 / 跨域
	{Key: "security.cookie_samesite", Group: "安全与跨域", Label: "Cookie SameSite", Type: "string",
		Get: func(c *config.Config) string { return c.Security.CookieSameSite },
		Set: func(c *config.Config, v string) { c.Security.CookieSameSite = v }},
	{Key: "security.cookie_secure", Group: "安全与跨域", Label: "Cookie Secure", Type: "string",
		Get: func(c *config.Config) string { return c.Security.CookieSecure },
		Set: func(c *config.Config, v string) { c.Security.CookieSecure = v }},
	{Key: "security.cookie_domain", Group: "安全与跨域", Label: "Cookie Domain", Type: "string",
		Get: func(c *config.Config) string { return c.Security.CookieDomain },
		Set: func(c *config.Config, v string) { c.Security.CookieDomain = v }},
	{Key: "security.cors_origins", Group: "安全与跨域", Label: "跨域来源（逗号分隔）", Type: "list",
		Get: func(c *config.Config) string { return strings.Join(c.Security.CORSAllowedOrigins, ",") },
		Set: func(c *config.Config, v string) { c.Security.CORSAllowedOrigins = splitTrim(v) }},

	// 审计
	{Key: "audit.enabled", Group: "审计日志", Label: "启用审计日志", Type: "bool",
		Get: func(c *config.Config) string { return strconv.FormatBool(c.Audit.Enabled) },
		Set: func(c *config.Config, v string) { c.Audit.Enabled = v == "true" }},
	{Key: "audit.retention_days", Group: "审计日志", Label: "保留天数", Type: "int",
		Get: func(c *config.Config) string { return strconv.Itoa(c.Audit.RetentionDays) },
		Set: func(c *config.Config, v string) {
			if n, err := strconv.Atoi(v); err == nil {
				c.Audit.RetentionDays = n
			}
		}},

	// 权限
	{Key: "permission.enable_rbac", Group: "权限", Label: "启用 RBAC", Type: "bool",
		Get: func(c *config.Config) string { return strconv.FormatBool(c.Permission.EnableRBAC) },
		Set: func(c *config.Config, v string) { c.Permission.EnableRBAC = v == "true" }},
	{Key: "permission.default_role", Group: "权限", Label: "默认角色", Type: "string",
		Get: func(c *config.Config) string { return c.Permission.DefaultRole },
		Set: func(c *config.Config, v string) { c.Permission.DefaultRole = v }},
	{Key: "permission.admin_users", Group: "权限", Label: "超级管理员名单（逗号分隔）", Type: "list",
		Get: func(c *config.Config) string { return strings.Join(c.Permission.AdminUsers, ",") },
		Set: func(c *config.Config, v string) { c.Permission.AdminUsers = splitTrim(v) }},

	// 日志
	{Key: "log.level", Group: "日志", Label: "日志级别", Type: "string",
		Get: func(c *config.Config) string { return c.LogConfig.Level },
		Set: func(c *config.Config, v string) { c.LogConfig.Level = v }},
	{Key: "log.format", Group: "日志", Label: "日志格式", Type: "string",
		Get: func(c *config.Config) string { return c.LogConfig.Format },
		Set: func(c *config.Config, v string) { c.LogConfig.Format = v }},

	// 管理员
	{Key: "admin.username", Group: "管理员", Label: "管理员用户名", Type: "string",
		Get: func(c *config.Config) string { return c.Admin.Username },
		Set: func(c *config.Config, v string) { c.Admin.Username = v }},
	{Key: "admin.password", Group: "管理员", Label: "管理员密码", Type: "secret", Secret: true,
		Get: func(c *config.Config) string { return c.Admin.Password },
		Set: func(c *config.Config, v string) { c.Admin.Password = v }},
	{Key: "admin.email", Group: "管理员", Label: "管理员邮箱", Type: "string",
		Get: func(c *config.Config) string { return c.Admin.Email },
		Set: func(c *config.Config, v string) { c.Admin.Email = v }},

	// JWT
	{Key: "jwt.access_expire", Group: "JWT", Label: "Access 过期(秒)", Type: "int",
		Get: func(c *config.Config) string { return strconv.Itoa(c.JWT.AccessExpire) },
		Set: func(c *config.Config, v string) {
			if n, err := strconv.Atoi(v); err == nil {
				c.JWT.AccessExpire = n
			}
		}},
	{Key: "jwt.refresh_expire", Group: "JWT", Label: "Refresh 过期(秒)", Type: "int",
		Get: func(c *config.Config) string { return strconv.Itoa(c.JWT.RefreshExpire) },
		Set: func(c *config.Config, v string) {
			if n, err := strconv.Atoi(v); err == nil {
				c.JWT.RefreshExpire = n
			}
		}},
	{Key: "jwt.issuer", Group: "JWT", Label: "签发者", Type: "string",
		Get: func(c *config.Config) string { return c.JWT.Issuer },
		Set: func(c *config.Config, v string) { c.JWT.Issuer = v }},

	// Casdoor 连接（保存后自动热重载客户端，无需重启）
	{Key: "casdoor.endpoint", Group: "Casdoor 连接", Label: "后端地址", Type: "string",
		Get: func(c *config.Config) string { return c.Casdoor.Endpoint },
		Set: func(c *config.Config, v string) { c.Casdoor.Endpoint = v }},
	{Key: "casdoor.public_endpoint", Group: "Casdoor 连接", Label: "浏览器地址", Type: "string",
		Get: func(c *config.Config) string { return c.Casdoor.PublicEndpoint },
		Set: func(c *config.Config, v string) { c.Casdoor.PublicEndpoint = v }},
	{Key: "casdoor.organization", Group: "Casdoor 连接", Label: "组织名", Type: "string",
		Get: func(c *config.Config) string { return c.Casdoor.OrganizationName },
		Set: func(c *config.Config, v string) { c.Casdoor.OrganizationName = v }},
	{Key: "casdoor.application", Group: "Casdoor 连接", Label: "应用名", Type: "string",
		Get: func(c *config.Config) string { return c.Casdoor.ApplicationName },
		Set: func(c *config.Config, v string) { c.Casdoor.ApplicationName = v }},
	{Key: "casdoor.certificate", Group: "Casdoor 连接", Label: "证书名", Type: "string",
		Get: func(c *config.Config) string { return c.Casdoor.Certificate },
		Set: func(c *config.Config, v string) { c.Casdoor.Certificate = v }},
	{Key: "casdoor.client_id", Group: "Casdoor 连接", Label: "Client ID", Type: "string",
		Get: func(c *config.Config) string { return c.Casdoor.ClientId },
		Set: func(c *config.Config, v string) { c.Casdoor.ClientId = v }},
	{Key: "casdoor.client_secret", Group: "Casdoor 连接", Label: "Client Secret", Type: "secret", Secret: true,
		Get: func(c *config.Config) string { return c.Casdoor.ClientSecret },
		Set: func(c *config.Config, v string) { c.Casdoor.ClientSecret = v }},
	{Key: "casdoor.user_cache_ttl", Group: "Casdoor 连接", Label: "用户缓存(秒)", Type: "int",
		Get: func(c *config.Config) string { return strconv.Itoa(c.Casdoor.UserCacheTTL) },
		Set: func(c *config.Config, v string) {
			if n, err := strconv.Atoi(v); err == nil {
				c.Casdoor.UserCacheTTL = n
			}
		}},
	{Key: "casdoor.default_password", Group: "Casdoor 连接", Label: "默认密码", Type: "secret", Secret: true,
		Get: func(c *config.Config) string { return c.Casdoor.DefaultPassword },
		Set: func(c *config.Config, v string) { c.Casdoor.DefaultPassword = v }},
	{Key: "casdoor.country_code", Group: "Casdoor 连接", Label: "手机号区域", Type: "string",
		Get: func(c *config.Config) string { return c.Casdoor.CountryCode },
		Set: func(c *config.Config, v string) { c.Casdoor.CountryCode = v }},
	{Key: "casdoor.auto_redirect_uri", Group: "Casdoor 连接", Label: "自动追加回调白名单", Type: "bool",
		Get: func(c *config.Config) string { return strconv.FormatBool(c.Casdoor.AutoRedirectURI) },
		Set: func(c *config.Config, v string) { c.Casdoor.AutoRedirectURI = v == "true" }},

	// Casdoor 同步
	{Key: "casdoor_sync.enabled", Group: "数据同步", Label: "启用同步", Type: "bool",
		Get: func(c *config.Config) string { return strconv.FormatBool(c.CasdoorSync.Enabled) },
		Set: func(c *config.Config, v string) { c.CasdoorSync.Enabled = v == "true" }},
	{Key: "casdoor_sync.auto", Group: "数据同步", Label: "数据源变更后自动同步", Type: "bool",
		Get: func(c *config.Config) string { return strconv.FormatBool(c.CasdoorSync.AutoSyncAfterDataSource) },
		Set: func(c *config.Config, v string) { c.CasdoorSync.AutoSyncAfterDataSource = v == "true" }},
	{Key: "casdoor_sync.batch_size", Group: "数据同步", Label: "批量大小", Type: "int",
		Get: func(c *config.Config) string { return strconv.Itoa(c.CasdoorSync.BatchSize) },
		Set: func(c *config.Config, v string) {
			if n, err := strconv.Atoi(v); err == nil {
				c.CasdoorSync.BatchSize = n
			}
		}},
	{Key: "casdoor_sync.retry_times", Group: "数据同步", Label: "失败重试次数", Type: "int",
		Get: func(c *config.Config) string { return strconv.Itoa(c.CasdoorSync.RetryTimes) },
		Set: func(c *config.Config, v string) {
			if n, err := strconv.Atoi(v); err == nil {
				c.CasdoorSync.RetryTimes = n
			}
		}},
	{Key: "casdoor_sync.retry_interval", Group: "数据同步", Label: "重试间隔(秒)", Type: "int",
		Get: func(c *config.Config) string { return strconv.Itoa(c.CasdoorSync.RetryInterval) },
		Set: func(c *config.Config, v string) {
			if n, err := strconv.Atoi(v); err == nil {
				c.CasdoorSync.RetryInterval = n
			}
		}},

	// 服务间调用
	{Key: "service.token", Group: "服务集成", Label: "全局服务令牌", Type: "secret", Secret: true,
		Get: func(c *config.Config) string { return c.Service.Token },
		Set: func(c *config.Config, v string) { c.Service.Token = v }},
}

// registryMap key -> Item
var registryMap = func() map[string]Item {
	m := make(map[string]Item, len(Registry))
	for _, it := range Registry {
		m[it.Key] = it
	}
	return m
}()

// Service 设置服务
type Service struct {
	mu    sync.RWMutex
	db    sqlx.SqlConn
	cfg   *config.Config
	items map[string]Item
}

// NewService 创建设置服务（cfg 为共享指针，应用值会直接写入）
func NewService(db sqlx.SqlConn, cfg *config.Config) *Service {
	return &Service{db: db, cfg: cfg, items: registryMap}
}

// Load 从数据库加载全部设置并应用到 Config（DB 覆盖 env）
func (s *Service) Load(ctx context.Context) error {
	var rows []struct {
		Key   string `db:"key"`
		Value string `db:"value"`
	}
	if err := s.db.QueryRowsCtx(ctx, &rows, `SELECT key, value FROM app_settings`); err != nil {
		return fmt.Errorf("读取应用设置失败: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range rows {
		if it, ok := s.items[r.Key]; ok {
			it.Set(s.cfg, r.Value)
		}
	}
	return nil
}

// Apply 将若干设置写入 DB 并立即应用到内存 Config（无需重启）。
//
// 返回实际生效的 key 列表（供调用方判断是否需要热重载相关组件，如 Casdoor 客户端）。
func (s *Service) Apply(ctx context.Context, kv map[string]string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	applied := make([]string, 0, len(kv))
	for k, v := range kv {
		it, ok := s.items[k]
		if !ok {
			continue
		}
		if _, err := s.db.ExecCtx(ctx,
			`INSERT INTO app_settings (key, value, updated_at) VALUES ($1,$2,NOW())
			 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
			k, v); err != nil {
			return applied, fmt.Errorf("保存设置 %s 失败: %w", k, err)
		}
		it.Set(s.cfg, v)
		applied = append(applied, k)
	}
	return applied, nil
}

// NeedsCasdoorReload 判断本次变更是否涉及 Casdoor 连接配置（需重建客户端）
func NeedsCasdoorReload(applied []string) bool {
	for _, k := range applied {
		if strings.HasPrefix(k, "casdoor.") {
			return true
		}
	}
	return false
}

// View 返回全部可配置项（含当前值；secret 以占位符返回）
func (s *Service) View() []map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]map[string]interface{}, 0, len(Registry))
	for _, it := range Registry {
		val := it.Get(s.cfg)
		if it.Secret && val != "" {
			val = "******"
		}
		out = append(out, map[string]interface{}{
			"key": it.Key, "group": it.Group, "label": it.Label,
			"type": it.Type, "secret": it.Secret, "value": val,
		})
	}
	return out
}

// splitTrim 逗号切分去空白
func splitTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// MaskedSecret 占位符（提交该值表示保持原值）
const MaskedSecret = "******"

// IsMasked 判断值是否为占位符
func IsMasked(v string) bool { return v == MaskedSecret }
