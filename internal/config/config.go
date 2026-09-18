package config

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
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
		Secret string `json:",env=JWT_SECRET"`
		// 注意：go-zero 会把带 env 标签的 int64 字段当作 time.Duration 解析，
		// 导致 JWT_ACCESS_EXPIRE=7200 报 "missing unit in duration"。
		// 此处使用 int（单位：秒），env 传纯数字即可。
		AccessExpire  int    `json:",default=7200,env=JWT_ACCESS_EXPIRE"`    // 2 hours
		RefreshExpire int    `json:",default=604800,env=JWT_REFRESH_EXPIRE"` // 7 days
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
		// DefaultPassword 新增用户/重置密码的默认密码。
		// 不设代码默认值，必须显式注入（环境变量 CASDOOR_DEFAULT_PASSWORD / Secret），
		// 避免弱口令被静默沿用。
		DefaultPassword string `json:",optional,env=CASDOOR_DEFAULT_PASSWORD"`
		// CountryCode 用户手机号所属国家/地区代码（ISO 3166-1 alpha-2，如 CN/US），
		// 用于 Casdoor 按区域正确解析并校验手机号，避免使用组织默认区域导致误判。
		CountryCode string `json:",default=CN,env=CASDOOR_COUNTRY_CODE"`
		// ProtectedUsers 受保护用户（域账号），仅在数据库表为空时作为种子写入
		ProtectedUsers []string `json:",optional"`
		// AutoRedirectURI 登录回调地址不存在于应用白名单时自动追加
		AutoRedirectURI bool `json:",default=true,env=CASDOOR_AUTO_REDIRECT_URI"`
		// RedirectURIs 应用回调地址白名单（初始化时写入 Casdoor）
		RedirectURIs []string `json:",optional"`
		// AllowedRedirectHosts 允许自动追加回调地址的主机白名单
		// （环境变量 CASDOOR_ALLOWED_REDIRECT_HOSTS，逗号分隔）
		AllowedRedirectHosts []string `json:",optional"`
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

	// Service 服务间调用配置（供其它系统如 Consul Manager 拉取配置）
	Service struct {
		// Token 服务间调用凭证（环境变量 SERVICE_TOKEN）。
		// 留空则不开放服务间接口。
		Token string `json:",optional,env=SERVICE_TOKEN"`
	}

	// Security 登录凭证与跨域相关配置
	Security struct {
		// CookieSameSite 登录 Cookie 的 SameSite 策略：lax / strict / none
		//   同源部署用 lax（默认，更安全）；
		//   前后端跨域部署必须 none（且需 HTTPS + Secure）。
		CookieSameSite string `json:",default=lax,env=AUTH_COOKIE_SAMESITE"`
		// CookieSecure 是否仅通过 HTTPS 发送：auto / true / false
		//   auto（默认）：按请求是否 TLS / X-Forwarded-Proto 自动判断；
		//   SameSite=none 时浏览器强制要求 Secure，此时自动视为 true。
		CookieSecure string `json:",default=auto,env=AUTH_COOKIE_SECURE"`
		// CookieDomain Cookie 作用域（跨子域共享时设为 .example.com），默认当前域
		CookieDomain string `json:",optional,env=AUTH_COOKIE_DOMAIN"`
		// CORSAllowedOrigins 允许的跨域来源（逗号分隔，精确匹配）。
		//   为空则不启用跨域；跨域时必须显式列出来源（不能用 *）。
		CORSAllowedOrigins []string `json:",optional"`
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

	CasdoorSync struct {
		Enabled                 bool `json:",default=true,env=CASDOOR_SYNC_ENABLED"`
		AutoSyncAfterDataSource bool `json:",default=true,env=CASDOOR_SYNC_AUTO"`
		BatchSize               int  `json:",default=50,env=CASDOOR_SYNC_BATCH_SIZE"`
		RetryTimes              int  `json:",default=3,env=CASDOOR_SYNC_RETRY_TIMES"`
		RetryInterval           int  `json:",default=5,env=CASDOOR_SYNC_RETRY_INTERVAL"` // seconds
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
	//
	// 无代码内默认值：不硬编码任何用户名。超级管理员可由两种途径产生：
	//  1. Casdoor 中该用户被标记为管理员（isAdmin），登录时由 Casdoor 权威返回；
	//  2. 通过 PERMISSION_ADMIN_USERS 显式配置的超级管理员名单。
	if v := os.Getenv("PERMISSION_ADMIN_USERS"); v != "" {
		c.Permission.AdminUsers = splitAndTrim(v)
	}

	// 受保护用户支持环境变量（逗号分隔）
	//
	// 同样无代码内默认值，避免硬编码具体账号。系统内置管理员账号
	// （bootstrap 创建的 <组织>/admin）在客户端层始终受保护，不依赖此列表。
	if v := os.Getenv("CASDOOR_PROTECTED_USERS"); v != "" {
		c.Casdoor.ProtectedUsers = splitAndTrim(v)
	}

	// 回调地址主机白名单支持环境变量（逗号分隔）
	if v := os.Getenv("CASDOOR_ALLOWED_REDIRECT_HOSTS"); v != "" {
		c.Casdoor.AllowedRedirectHosts = splitAndTrim(v)
	}

	// 跨域来源白名单支持环境变量（逗号分隔）
	if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		c.Security.CORSAllowedOrigins = splitAndTrim(v)
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

// Validate 验证启动前置配置（不依赖数据库中的页面设置）。
//
// 说明：Casdoor 连接类配置可由「系统设置」页面（存 app_settings 表）提供，
// 需在 settings.Load 之后才能读到，故由 ValidateCasdoor 单独校验。
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

	return nil
}

// ValidateCasdoor 校验 Casdoor 连接配置。
//
// 必须在 settings.Load 之后调用：这些字段可由「系统设置」页面（数据库）配置，
// 此时内存配置已合并 DB 值（DB 优先 / env 兜底）。
func (c *Config) ValidateCasdoor() error {
	if c.Casdoor.Endpoint == "" {
		return fmt.Errorf("Casdoor 端点未设置: 请设置 CASDOOR_ENDPOINT 环境变量或在「系统设置」页面配置")
	}
	if c.Casdoor.DefaultPassword == "" {
		return fmt.Errorf("Casdoor 默认密码未设置: 请设置 CASDOOR_DEFAULT_PASSWORD 环境变量/Secret，或在「系统设置」页面配置")
	}
	// 应用凭据：
	//   - 开启 AutoSetup（默认）时可留空，由程序在启动时从内置应用读取/创建业务应用
	//     并回填凭据（见 svc.buildCasdoorClient → casdoor.EnsureSetup）；
	//   - 关闭 AutoSetup 时必须显式提供，否则无法建立客户端。
	if !c.Casdoor.AutoSetup && (c.Casdoor.ClientId == "" || c.Casdoor.ClientSecret == "") {
		return fmt.Errorf("Casdoor 应用凭据未设置: 请设置 CASDOOR_CLIENT_ID / CASDOOR_CLIENT_SECRET，开启 CASDOOR_AUTO_SETUP 自动创建，或在「系统设置」页面配置")
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

// MaintenanceDSN 返回连接 postgres 维护库的连接串（用于首次自动建库）。
//
// 目标库尚不存在时，普通连接无法建立，需先连到 postgres 库执行 CREATE DATABASE。
func (c *Config) MaintenanceDSN() string {
	if c.Database.DSN != "" {
		return replaceDBNameInDSN(c.Database.DSN, "postgres")
	}
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
		c.Database.Host, c.Database.Port, c.Database.User, c.Database.Password, c.Database.SSLMode)
}

// replaceDBNameInDSN 将 DSN 中的库名替换为指定值，兼容 URL 与 key=value 两种格式。
func replaceDBNameInDSN(dsn, dbName string) string {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		if u, err := url.Parse(dsn); err == nil {
			u.Path = "/" + dbName
			return u.String()
		}
		return dsn
	}
	// key=value 形式：替换 dbname=xxx
	re := regexp.MustCompile(`(^|\s)dbname=[^\s]+`)
	if re.MatchString(dsn) {
		return re.ReplaceAllString(dsn, "${1}dbname="+dbName)
	}
	return dsn + " dbname=" + dbName
}
