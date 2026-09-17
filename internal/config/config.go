package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/rest"
)

// Config 应用配置
type Config struct {
	rest.RestConf

	Database struct {
		DSN             string `json:",optional,env=DATABASE_URL"`
		Host            string `json:",optional,default=localhost,env=DB_HOST"`
		Port            int    `json:",optional,default=5432,env=DB_PORT"`
		User            string `json:",optional,default=postgres,env=DB_USER"`
		Password        string `json:",optional,env=DB_PASSWORD"`
		DBName          string `json:",optional,default=flyiam,env=DB_NAME"`
		SSLMode         string `json:",optional,default=disable,env=DB_SSLMODE"`
		MaxOpenConns    int    `json:",optional,default=100,env=DB_MAX_OPEN_CONNS"`
		MaxIdleConns    int    `json:",optional,default=10,env=DB_MAX_IDLE_CONNS"`
		ConnMaxLifetime int    `json:",optional,default=3600,env=DB_CONN_MAX_LIFETIME"` // seconds
	}

	JWT struct {
		Secret        string `json:",env=JWT_SECRET"`
		AccessExpire  int64  `json:",default=7200,env=JWT_ACCESS_EXPIRE"`    // 2 hours
		RefreshExpire int64  `json:",default=604800,env=JWT_REFRESH_EXPIRE"` // 7 days
		Issuer        string `json:",default=flyiam,env=JWT_ISSUER"`
	}

	Admin struct {
		Username string `json:",default=admin,env=ADMIN_USERNAME"`
		Password string `json:",env=ADMIN_PASSWORD"`
		Email    string `json:",optional,env=ADMIN_EMAIL"`
	}

	Casdoor struct {
		Endpoint       string `json:",env=CASDOOR_ENDPOINT"`
		PublicEndpoint string `json:",optional,env=CASDOOR_PUBLIC_ENDPOINT"`
		// ClientId/ClientSecret 业务应用凭据；留空时由程序在 Casdoor 中自动创建并持久化
		ClientId     string `json:",optional,env=CASDOOR_CLIENT_ID"`
		ClientSecret string `json:",optional,env=CASDOOR_CLIENT_SECRET"`
		// Certificate 应用使用的证书名（Casdoor 内置证书为 cert-built-in）
		Certificate      string `json:",default=cert-built-in,env=CASDOOR_CERTIFICATE"`
		OrganizationName string `json:",default=flyiam,env=CASDOOR_ORGANIZATION"`
		// OrganizationDisplayName 组织显示名
		OrganizationDisplayName string `json:",default=FlyIAM,env=CASDOOR_ORGANIZATION_DISPLAY_NAME"`
		ApplicationName         string `json:",default=flyiam,env=CASDOOR_APPLICATION"`
		// ApplicationDisplayName 应用显示名
		ApplicationDisplayName string `json:",default=FlyIAM,env=CASDOOR_APPLICATION_DISPLAY_NAME"`
		DefaultPassword        string `json:",default=ysyh!9Sky,env=CASDOOR_DEFAULT_PASSWORD"`
		// CountryCode 用户手机号所属国家/地区代码（ISO 3166-1 alpha-2，如 CN/US），
		// 用于 Casdoor 按区域正确解析并校验手机号，避免使用组织默认区域导致误判。
		CountryCode string `json:",default=CN,env=CASDOOR_COUNTRY_CODE"`
		// UserCacheTTL 用户列表内存缓存时长（秒），降低 Casdoor 全量查询压力
		UserCacheTTL int `json:",default=30,env=CASDOOR_USER_CACHE_TTL"`
		// ProtectedUsers 受保护用户（域账号），仅在数据库表为空时作为种子写入
		ProtectedUsers []string `json:",optional"`
		// AutoRedirectURI 登录回调地址不存在于应用白名单时自动追加
		AutoRedirectURI bool `json:",default=true,env=CASDOOR_AUTO_REDIRECT_URI"`
		// RedirectURIs 应用回调地址白名单（初始化时写入 Casdoor）
		RedirectURIs []string `json:",optional"`
		// AutoSetup 是否在启动时自动初始化 Casdoor（组织/应用/管理员）
		AutoSetup bool `json:",default=true,env=CASDOOR_AUTO_SETUP"`
	}

	Redis struct {
		Enabled  bool   `json:",default=true,env=REDIS_ENABLED"`
		Host     string `json:",default=localhost,env=REDIS_HOST"`
		Port     int    `json:",default=6379,env=REDIS_PORT"`
		Password string `json:",optional,env=REDIS_PASSWORD"`
		DB       int    `json:",default=0,env=REDIS_DB"`
		TTL      int    `json:",default=300,env=REDIS_CACHE_TTL"` // seconds
	}

	Web struct {
		Embedded  bool   `json:",default=true,env=WEB_EMBEDDED"`
		StaticDir string `json:",default=./web/dist,env=WEB_STATIC_DIR"`
	}

	LogConfig struct {
		Level  string `json:",default=info,env=LOG_LEVEL"`
		Format string `json:",default=json,env=LOG_FORMAT"`
	}

	Audit struct {
		Enabled       bool `json:",default=true,env=AUDIT_ENABLED"`
		RetentionDays int  `json:",default=90,env=AUDIT_RETENTION_DAYS"`
	}

	Permission struct {
		EnableRBAC  bool     `json:",default=true,env=PERMISSION_ENABLE_RBAC"`
		DefaultRole string   `json:",default=user,env=PERMISSION_DEFAULT_ROLE"`
		AdminUsers  []string `json:",optional"`
	}

	DataSources struct {
		HttpApi HttpApiDataSourceConfig `json:",optional"`
	}

	CasdoorSync struct {
		Enabled                 bool `json:",default=true,env=CASDOOR_SYNC_ENABLED"`
		AutoSyncAfterDataSource bool `json:",default=true,env=CASDOOR_SYNC_AUTO"`
		BatchSize               int  `json:",default=50,env=CASDOOR_SYNC_BATCH_SIZE"`
		RetryTimes              int  `json:",default=3,env=CASDOOR_SYNC_RETRY_TIMES"`
		RetryInterval           int  `json:",default=5,env=CASDOOR_SYNC_RETRY_INTERVAL"` // seconds
	}
}

// HttpApiDataSourceConfig HTTP API 数据源配置
type HttpApiDataSourceConfig struct {
	Enabled           bool   `json:",default=false,env=DATASOURCE_HTTPAPI_ENABLED"`
	URL               string `json:",optional,env=DATASOURCE_HTTPAPI_URL"`
	Timeout           int    `json:",default=30,env=DATASOURCE_HTTPAPI_TIMEOUT"`
	SyncInterval      string `json:",default=6h,env=DATASOURCE_HTTPAPI_SYNC_INTERVAL"`
	AutoSyncOnStartup bool   `json:",default=true,env=DATASOURCE_HTTPAPI_AUTO_SYNC"`
	Priority          int    `json:",default=90,env=DATASOURCE_HTTPAPI_PRIORITY"`
	Auth              struct {
		Type     string `json:",default=bearer,env=DATASOURCE_HTTPAPI_AUTH_TYPE"`
		Token    string `json:",optional,env=DATASOURCE_HTTPAPI_AUTH_TOKEN"`
		Username string `json:",optional,env=DATASOURCE_HTTPAPI_AUTH_USERNAME"`
		Password string `json:",optional,env=DATASOURCE_HTTPAPI_AUTH_PASSWORD"`
	}
}

// ApplyEnvOverrides 应用环境变量覆盖
func (c *Config) ApplyEnvOverrides() {
	// 服务配置
	if v := os.Getenv("SERVER_HOST"); v != "" {
		c.Host = v
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.Port = p
		}
	}
	if v := os.Getenv("SERVER_MODE"); v != "" {
		c.Mode = v
	}
	if v := os.Getenv("SERVER_TIMEOUT"); v != "" {
		if t, err := strconv.ParseInt(v, 10, 64); err == nil {
			c.Timeout = t
		}
	}
	// 兜底默认值：纯环境变量部署（无配置文件）且未设置 SERVER_PORT 时避免监听 0 端口
	if c.Host == "" {
		c.Host = "0.0.0.0"
	}
	if c.Port == 0 {
		c.Port = 8081
	}

	// Permission AdminUsers 支持环境变量（逗号分隔）
	if v := os.Getenv("PERMISSION_ADMIN_USERS"); v != "" {
		c.Permission.AdminUsers = splitAndTrim(v)
	}
	// 默认至少包含 admin，避免无人可管理
	if len(c.Permission.AdminUsers) == 0 {
		c.Permission.AdminUsers = []string{"admin"}
	}

	// 受保护用户支持环境变量（逗号分隔）
	if v := os.Getenv("CASDOOR_PROTECTED_USERS"); v != "" {
		c.Casdoor.ProtectedUsers = splitAndTrim(v)
	}

	// 受保护用户默认值（至少包含 admin，避免误删管理员）
	if len(c.Casdoor.ProtectedUsers) == 0 {
		c.Casdoor.ProtectedUsers = []string{"admin"}
	}
}

// splitAndTrim 按逗号切分并去除空白
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证数据库配置
	if c.Database.DSN == "" && c.Database.Host == "" {
		return fmt.Errorf("数据库配置缺失: 请设置 DATABASE_URL 或 DB_HOST")
	}

	// 验证 JWT 密钥
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT 密钥未设置: 请设置 JWT_SECRET 环境变量")
	}
	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT 密钥长度不足: 至少需要 32 个字符，当前 %d 个", len(c.JWT.Secret))
	}

	// 验证管理员配置
	if c.Admin.Password == "" {
		return fmt.Errorf("管理员密码未设置: 请设置 ADMIN_PASSWORD 环境变量")
	}

	// 验证 Casdoor 配置
	if c.Casdoor.Endpoint == "" {
		return fmt.Errorf("Casdoor 端点未设置: 请设置 CASDOOR_ENDPOINT 环境变量")
	}
	if c.Casdoor.ClientId == "" {
		return fmt.Errorf("Casdoor Client ID 未设置: 请设置 CASDOOR_CLIENT_ID 环境变量")
	}
	if c.Casdoor.ClientSecret == "" {
		return fmt.Errorf("Casdoor Client Secret 未设置: 请设置 CASDOOR_CLIENT_SECRET 环境变量")
	}

	return nil
}

// GetDSN 获取数据库连接字符串
func (c *Config) GetDSN() string {
	if c.Database.DSN != "" {
		return c.Database.DSN
	}
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}
