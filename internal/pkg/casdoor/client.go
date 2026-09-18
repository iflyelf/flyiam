package casdoor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	casdoorsdk "github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"github.com/golang-jwt/jwt/v4"
)

// adminRetryInterval 管理凭据补读的最小间隔（失败节流，避免每次请求都查库）
const adminRetryInterval = 30 * time.Second

// Client Casdoor 客户端
type Client struct {
	config       *Config
	authConfig   *casdoorsdk.AuthConfig
	organization string
	application  string
	// bizSDK 使用业务应用凭据的 SDK 实例（用于本组织用户/认证等操作）。
	// 全程使用实例客户端，避免依赖 SDK 包级全局配置被其他凭据覆盖。
	bizSDK *casdoorsdk.Client

	// adminSDK 使用内置应用 app-built-in 凭据的独立 SDK 实例，
	// 专用于「应用 / 组织管理」等需要全局管理员权限的接口。
	// 使用实例客户端而非全局函数，避免与业务凭据的全局配置互相覆盖。
	//
	// 该凭据可能因 Casdoor 晚于本服务启动而暂时读不到，故支持运行时补读：
	//   - adminCredLoader 由调用方注入（从数据库读取 app-built-in 凭据）；
	//   - adminClient() 在缺失时按需补读（带失败节流）；
	//   - WatchAdminCredentials() 后台重试，无需重启服务。
	adminMu          sync.RWMutex
	adminSDK         *casdoorsdk.Client
	adminCredLoader  func() (string, string, error)
	adminLastAttempt time.Time

	protected   map[string]struct{}
	protectedMu sync.RWMutex

	cacheMu      sync.RWMutex
	usersCache   []*casdoorsdk.User
	usersCacheAt time.Time
}

// IsProtected 判断用户是否为受保护用户（不允许删除）
func (c *Client) IsProtected(name string) bool {
	c.protectedMu.RLock()
	defer c.protectedMu.RUnlock()
	_, ok := c.protected[name]
	return ok
}

// ProtectedList 返回受保护用户列表
func (c *Client) ProtectedList() []string {
	c.protectedMu.RLock()
	defer c.protectedMu.RUnlock()
	out := make([]string, 0, len(c.protected))
	for k := range c.protected {
		out = append(out, k)
	}
	return out
}

// SetProtected 动态设置受保护用户列表（admin 始终受保护）
func (c *Client) SetProtected(names []string) {
	m := make(map[string]struct{}, len(names)+1)
	for _, n := range names {
		if n != "" {
			m[n] = struct{}{}
		}
	}
	m["admin"] = struct{}{}
	c.protectedMu.Lock()
	c.protected = m
	c.protectedMu.Unlock()
}

// Config Casdoor 配置
type Config struct {
	Endpoint         string
	PublicEndpoint   string
	ClientId         string
	ClientSecret     string
	Certificate      string
	OrganizationName string
	// AdminClientId/AdminClientSecret 为 Casdoor 内置应用 app-built-in 的凭据。
	// 仅 built-in 身份具备全局管理员权限，应用/组织管理接口需用它，否则报
	// "Unauthorized operation"（应用列表）或 "Please sign in first"（组织列表）。
	AdminClientId     string
	AdminClientSecret string
	// OrganizationDisplayName 组织显示名
	OrganizationDisplayName string
	ApplicationName         string
	// ApplicationDisplayName 应用显示名
	ApplicationDisplayName string
	DefaultPassword        string
	CountryCode            string
	UserCacheTTL           int
	ProtectedUsers         []string
	// AutoRedirectURI 为 true 时，登录回调地址不存在于应用白名单会自动追加
	AutoRedirectURI bool
	// RedirectURIs 应用回调地址白名单（初始化时写入 Casdoor）
	RedirectURIs []string
	// AllowedRedirectHosts 允许自动追加回调地址的主机白名单。
	// 非空时，仅当推导出的回调地址主机在其中才自动追加，
	// 防止 Host 头被伪造把恶意回调写入 Casdoor 白名单（开放重定向）。
	// 为空时保持原有行为（追加）并打印告警，建议生产环境显式配置。
	AllowedRedirectHosts []string
}

// NewClient 创建 Casdoor 客户端
func NewClient(cfg *Config) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("Casdoor endpoint is required")
	}
	if cfg.ClientId == "" {
		return nil, fmt.Errorf("Casdoor client ID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, fmt.Errorf("Casdoor client secret is required")
	}

	// 创建 Casdoor SDK 配置
	authConfig := &casdoorsdk.AuthConfig{
		Endpoint:         cfg.Endpoint,
		ClientId:         cfg.ClientId,
		ClientSecret:     cfg.ClientSecret,
		Certificate:      cfg.Certificate,
		OrganizationName: cfg.OrganizationName,
		ApplicationName:  cfg.ApplicationName,
	}

	// 为 SDK 注入带超时的 HTTP 客户端（SDK 默认 &http.Client{} 无超时）
	casdoorsdk.SetHttpClient(&http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     90 * time.Second,
		},
	})

	// 业务应用凭据的实例客户端。
	// 不再调用 casdoorsdk.InitConfig：它写入包级全局变量且无锁，
	// 与管理凭据会互相覆盖（顺序敏感），全程使用实例客户端以彻底隔离。
	bizSDK := casdoorsdk.NewClient(
		cfg.Endpoint,
		cfg.ClientId,
		cfg.ClientSecret,
		cfg.Certificate,
		cfg.OrganizationName,
		cfg.ApplicationName,
	)

	// 内置应用（app-built-in）凭据：用于应用 / 组织管理等全局管理员接口。
	// 用独立的实例客户端，不触碰全局配置，避免与业务凭据互相覆盖。
	protected := make(map[string]struct{}, len(cfg.ProtectedUsers)+1)
	for _, name := range cfg.ProtectedUsers {
		if name != "" {
			protected[name] = struct{}{}
		}
	}
	protected["admin"] = struct{}{}

	c := &Client{
		config:       cfg,
		authConfig:   authConfig,
		organization: cfg.OrganizationName,
		application:  cfg.ApplicationName,
		bizSDK:       bizSDK,
		protected:    protected,
	}
	if cfg.AdminClientId != "" && cfg.AdminClientSecret != "" {
		c.adminSDK = newAdminSDK(cfg.Endpoint, cfg.AdminClientId, cfg.AdminClientSecret)
	}
	return c, nil
}

// newAdminSDK 构建使用内置应用凭据的 SDK 实例（全局管理员身份）
func newAdminSDK(endpoint, clientId, clientSecret string) *casdoorsdk.Client {
	return casdoorsdk.NewClient(endpoint, clientId, clientSecret, "", "built-in", "app-built-in")
}

// SetAdminCredentialLoader 注入内置应用凭据的读取方式（通常从数据库读取）。
// 注入后，若启动时未取到凭据，会在首次使用管理接口时按需补读，并可配合
// WatchAdminCredentials 在后台自动重试，无需重启服务。
func (c *Client) SetAdminCredentialLoader(fn func() (string, string, error)) {
	c.adminMu.Lock()
	c.adminCredLoader = fn
	c.adminMu.Unlock()
}

// AdminReady 管理凭据（内置应用）是否已就绪
func (c *Client) AdminReady() bool {
	c.adminMu.RLock()
	defer c.adminMu.RUnlock()
	return c.adminSDK != nil
}

// tryLoadAdmin 尝试补读内置应用凭据（线程安全，失败后按 adminRetryInterval 节流）。
//
// force=true 时忽略节流（供后台定时任务使用）。
func (c *Client) tryLoadAdmin(force bool) error {
	c.adminMu.Lock()
	defer c.adminMu.Unlock()

	if c.adminSDK != nil {
		return nil
	}
	if c.adminCredLoader == nil {
		return fmt.Errorf("未注入内置应用凭据读取方式")
	}
	if !force && !c.adminLastAttempt.IsZero() && time.Since(c.adminLastAttempt) < adminRetryInterval {
		return fmt.Errorf("Casdoor 内置应用凭据尚未就绪（读取失败，%s 内不重试）", adminRetryInterval)
	}
	c.adminLastAttempt = time.Now()

	id, secret, err := c.adminCredLoader()
	if err != nil {
		return err
	}
	if id == "" || secret == "" {
		return fmt.Errorf("Casdoor 内置应用凭据为空")
	}

	c.adminSDK = newAdminSDK(c.config.Endpoint, id, secret)
	c.config.AdminClientId = id
	c.config.AdminClientSecret = secret
	return nil
}

// WatchAdminCredentials 后台重试补读管理凭据，直到成功或达到最大尝试次数。
//
// 场景：Casdoor 首次启动建表（app-built-in）可能晚于 FlyIAM，启动时读不到凭据。
// 该方法让服务自动恢复，无需人工重启。
func (c *Client) WatchAdminCredentials(interval time.Duration, maxAttempts int) {
	if c.AdminReady() {
		return
	}
	if interval <= 0 {
		interval = 10 * time.Second
	}
	go func() {
		for i := 0; i < maxAttempts; i++ {
			time.Sleep(interval)
			if c.AdminReady() {
				return
			}
			if err := c.tryLoadAdmin(true); err == nil {
				log.Printf("✅ Casdoor 管理凭据已就绪（内置应用 app-built-in），应用/组织管理功能可用")
				return
			}
		}
		if !c.AdminReady() {
			log.Printf("⚠️ 已重试 %d 次仍未读到 Casdoor 内置应用凭据，应用/组织管理功能暂不可用"+
				"（请确认 casdoor_application 表中存在 app-built-in）", maxAttempts)
		}
	}()
}

// adminClient 返回具备全局管理员权限的 SDK 实例（内置应用 app-built-in）。
// 应用 / 组织管理等全局接口必须用它，业务应用凭据会因权限不足被 Casdoor 拒绝。
//
// 凭据缺失时会按需补读一次（节流），因此 Casdoor 晚于本服务就绪后可自动恢复。
func (c *Client) adminClient() (*casdoorsdk.Client, error) {
	c.adminMu.RLock()
	sdk := c.adminSDK
	c.adminMu.RUnlock()
	if sdk != nil {
		return sdk, nil
	}

	if err := c.tryLoadAdmin(false); err != nil {
		return nil, fmt.Errorf("Casdoor 管理凭据未就绪（未能读取内置应用 app-built-in）: %w", err)
	}

	c.adminMu.RLock()
	defer c.adminMu.RUnlock()
	return c.adminSDK, nil
}

// adminCreds 返回内置应用凭据（供需要直连 HTTP 的接口使用）
func (c *Client) adminCreds() (string, string, error) {
	if _, err := c.adminClient(); err != nil {
		return "", "", err
	}
	c.adminMu.RLock()
	defer c.adminMu.RUnlock()
	return c.config.AdminClientId, c.config.AdminClientSecret, nil
}

// Organization 返回当前组织名
func (c *Client) Organization() string { return c.organization }

// Application 返回当前应用名
func (c *Client) Application() string { return c.application }

// ListApplications 获取全部应用
//
// 注意：应用（owner=admin）属于全局对象，仅内置应用凭据有权限，
// 业务应用凭据调用会报 "Unauthorized operation"。
func (c *Client) ListApplications() ([]*casdoorsdk.Application, error) {
	sdk, err := c.adminClient()
	if err != nil {
		return nil, err
	}
	apps, err := sdk.GetApplications()
	if err != nil {
		return nil, fmt.Errorf("获取应用列表失败: %w", err)
	}
	return apps, nil
}

// GetApplication 获取单个应用
func (c *Client) GetApplication(name string) (*casdoorsdk.Application, error) {
	sdk, err := c.adminClient()
	if err != nil {
		return nil, err
	}
	app, err := sdk.GetApplication(name)
	if err != nil {
		return nil, fmt.Errorf("获取应用失败: %w", err)
	}
	return app, nil
}

// AddApplication 新增应用
func (c *Client) AddApplication(app *casdoorsdk.Application) (bool, error) {
	sdk, err := c.adminClient()
	if err != nil {
		return false, err
	}
	ok, err := sdk.AddApplication(app)
	if err != nil {
		return false, fmt.Errorf("新增应用失败: %w", err)
	}
	return ok, nil
}

// UpdateApplication 更新应用
func (c *Client) UpdateApplication(app *casdoorsdk.Application) (bool, error) {
	sdk, err := c.adminClient()
	if err != nil {
		return false, err
	}
	ok, err := sdk.UpdateApplication(app)
	if err != nil {
		return false, fmt.Errorf("更新应用失败: %w", err)
	}
	return ok, nil
}

// DeleteApplication 删除应用
func (c *Client) DeleteApplication(name string) (bool, error) {
	sdk, err := c.adminClient()
	if err != nil {
		return false, err
	}
	app := &casdoorsdk.Application{Owner: "admin", Name: name}
	ok, err := sdk.DeleteApplication(app)
	if err != nil {
		return false, fmt.Errorf("删除应用失败: %w", err)
	}
	return ok, nil
}

// EnsureRedirectURI 确保回调地址在应用白名单中（解决 Redirect URI 未配置导致的登录失败）
//
// 返回是否发生了变更。
func (c *Client) EnsureRedirectURI(redirectURI string) (bool, error) {
	if !c.config.AutoRedirectURI || redirectURI == "" {
		return false, nil
	}

	// 回调地址主机白名单校验：防止 Host 头被伪造导致把恶意回调写入 Casdoor
	if len(c.config.AllowedRedirectHosts) > 0 {
		host := hostOf(redirectURI)
		if host == "" || !containsStr(c.config.AllowedRedirectHosts, host) {
			return false, fmt.Errorf("回调地址主机 %q 不在允许列表内，拒绝自动追加（如确需请加入 CASDOOR_ALLOWED_REDIRECT_HOSTS）", host)
		}
	} else {
		log.Printf("⚠️ 未配置 CASDOOR_ALLOWED_REDIRECT_HOSTS，回调地址将依据请求 Host 自动追加；" +
			"生产环境建议配置主机白名单以避免 Host 头伪造")
	}

	sdk, err := c.adminClient()
	if err != nil {
		return false, err
	}
	app, err := sdk.GetApplication(c.application)
	if err != nil || app == nil {
		return false, fmt.Errorf("获取应用失败: %w", err)
	}
	for _, u := range app.RedirectUris {
		if u == redirectURI {
			return false, nil
		}
	}
	app.RedirectUris = append(app.RedirectUris, redirectURI)
	if _, err := sdk.UpdateApplication(app); err != nil {
		return false, fmt.Errorf("追加回调地址失败: %w", err)
	}
	return true, nil
}

// ListOrganizations 获取全部组织
//
// 说明：SDK 的 GetOrganizations 会按当前组织名作为 owner 过滤，
// 而 Casdoor 中组织的 owner 为 admin（非当前组织名），会导致返回空列表。
// 因此这里直接调用接口且不传 owner，返回全部组织。
//
// 权限：组织属于全局对象，且控制器要求当前身份为全局管理员，
// 故必须使用内置应用（app-built-in）凭据，业务凭据会报 "Please sign in first"。
func (c *Client) ListOrganizations() ([]*casdoorsdk.Organization, error) {
	adminID, adminSecret, err := c.adminCreds()
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimRight(c.config.Endpoint, "/") + "/api/get-organizations"
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}
	req.SetBasicAuth(adminID, adminSecret)

	// 使用带超时的客户端，避免 Casdoor/反向代理 hang 住导致连接堆积
	httpClient := &http.Client{Timeout: 15 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 Casdoor 失败: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取组织响应失败: %w", err)
	}

	var envelope struct {
		Status string                     `json:"status"`
		Msg    string                     `json:"msg"`
		Data   []*casdoorsdk.Organization `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("解析组织响应失败: %w", err)
	}
	if envelope.Status != "ok" {
		return nil, fmt.Errorf("获取组织失败: %s", envelope.Msg)
	}
	return envelope.Data, nil
}

// GetAuthConfig 获取认证配置（供前端使用）
func (c *Client) GetAuthConfig() map[string]interface{} {
	endpoint := c.config.Endpoint
	if c.config.PublicEndpoint != "" {
		endpoint = c.config.PublicEndpoint
	}

	return map[string]interface{}{
		"serverUrl":        endpoint,
		"clientId":         c.config.ClientId,
		"appName":          c.config.ApplicationName,
		"organizationName": c.config.OrganizationName,
	}
}

// PublicEndpoint 浏览器可达的 Casdoor 地址
func (c *Client) PublicEndpoint() string {
	if c.config.PublicEndpoint != "" {
		return c.config.PublicEndpoint
	}
	return c.config.Endpoint
}

// GetSigninUrl 获取登录 URL（基于浏览器可达地址，支持跨域名部署）
func (c *Client) GetSigninUrl(redirectUri, state string) string {
	base := strings.TrimRight(c.PublicEndpoint(), "/")
	return fmt.Sprintf(
		"%s/login/oauth/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=openid profile email&state=%s",
		base,
		url.QueryEscape(c.config.ClientId),
		url.QueryEscape(redirectUri),
		url.QueryEscape(state),
	)
}

// hostOf 从 URL 中提取主机名（含端口前的主机部分）
func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// Claims 精简的 Token 声明
type Claims struct {
	Owner       string
	Name        string
	Id          string
	DisplayName string
	Email       string
	Phone       string
	Avatar      string
	IsAdmin     bool
	Exp         int64
	Iat         int64
}

// ParseToken 解析 Casdoor Access Token（不校验签名，仅取载荷）
func (c *Client) ParseToken(token string) (*Claims, error) {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return nil, nil
	})
	if err != nil && parsed == nil {
		return nil, fmt.Errorf("Token 解析失败: %w", err)
	}
	raw, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("Token 声明格式错误")
	}
	result := &Claims{}
	if v, ok := raw["owner"].(string); ok {
		result.Owner = v
	}
	if v, ok := raw["name"].(string); ok {
		result.Name = v
	}
	if v, ok := raw["id"].(string); ok {
		result.Id = v
	}
	if v, ok := raw["displayName"].(string); ok {
		result.DisplayName = v
	}
	if v, ok := raw["email"].(string); ok {
		result.Email = v
	}
	if v, ok := raw["phone"].(string); ok {
		result.Phone = v
	}
	if v, ok := raw["avatar"].(string); ok {
		result.Avatar = v
	}
	if v, ok := raw["picture"].(string); ok && result.Avatar == "" {
		result.Avatar = v
	}
	if v, ok := raw["isAdmin"].(bool); ok {
		result.IsAdmin = v
	}
	if v, ok := raw["exp"].(float64); ok {
		result.Exp = int64(v)
	}
	if v, ok := raw["iat"].(float64); ok {
		result.Iat = int64(v)
	}
	if result.Name == "" {
		return nil, fmt.Errorf("Token 中无用户信息")
	}
	return result, nil
}

// GetUsersByOrg 按组织过滤用户（实时）
func (c *Client) GetUsersByOrg() ([]*casdoorsdk.User, error) {
	all, err := c.GetUsers()
	if err != nil {
		return nil, err
	}
	out := make([]*casdoorsdk.User, 0, len(all))
	for _, u := range all {
		if u.Owner == c.organization {
			out = append(out, u)
		}
	}
	return out, nil
}

// GetUsersByOrgCached 按组织获取用户（带内存缓存，降低全量查询压力）
func (c *Client) GetUsersByOrgCached() ([]*casdoorsdk.User, error) {
	ttl := time.Duration(c.config.UserCacheTTL) * time.Second
	if ttl <= 0 {
		return c.GetUsersByOrg()
	}

	c.cacheMu.RLock()
	if c.usersCache != nil && time.Since(c.usersCacheAt) < ttl {
		data := c.usersCache
		c.cacheMu.RUnlock()
		return data, nil
	}
	c.cacheMu.RUnlock()

	users, err := c.GetUsersByOrg()
	if err != nil {
		return nil, err
	}
	c.cacheMu.Lock()
	c.usersCache = users
	c.usersCacheAt = time.Now()
	c.cacheMu.Unlock()
	return users, nil
}

// InvalidateUsersCache 使缓存失效（同步后调用）
func (c *Client) InvalidateUsersCache() {
	c.cacheMu.Lock()
	c.usersCache = nil
	c.usersCacheAt = time.Time{}
	c.cacheMu.Unlock()
}

// GetUsersPage 服务端分页查询用户（支持按字段模糊搜索）
//
// 说明：直接使用 Casdoor 服务端分页，避免每次拉取全量用户导致的慢与卡顿。
func (c *Client) GetUsersPage(page, pageSize int, field, value string) ([]*casdoorsdk.User, int, error) {
	queryMap := map[string]string{}
	if field != "" && value != "" {
		queryMap["field"] = field
		queryMap["value"] = value
	}
	return c.bizSDK.GetPaginationUsers(page, pageSize, queryMap)
}

// GetUserCount 获取组织用户总数
func (c *Client) GetUserCount() (int, error) {
	_, total, err := c.GetUsersPage(1, 1, "", "")
	return total, err
}

// SearchUsers 按关键字跨字段搜索用户（域账号 name + 姓名 displayName）。
//
// 背景：Casdoor 的 get-users 仅支持单字段 LIKE 查询，无法一次匹配多个字段；
// 故分别以 name / displayName 查询后合并去重，供用户选择器（下拉搜索）使用。
//
// 参数：
//   - keyword 为关键字；为空时返回前 limit 个用户。
//   - limit   返回上限（<=0 或 >200 时取 50）。
func (c *Client) SearchUsers(keyword string, limit int) ([]*casdoorsdk.User, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	keyword = strings.TrimSpace(keyword)

	// 无条件拉取（无关键字）：直接返回前 limit 条
	if keyword == "" {
		users, _, err := c.GetUsersPage(1, limit, "", "")
		return users, err
	}

	// 域账号（name）模糊匹配
	nameUsers, _, err := c.GetUsersPage(1, limit, "name", keyword)
	if err != nil {
		return nil, err
	}
	if merged := dedupeUsers(nameUsers, limit); len(merged) >= limit {
		return merged, nil
	}

	// 姓名（displayName）模糊匹配（补充）
	displayUsers, _, err := c.GetUsersPage(1, limit, "displayName", keyword)
	if err != nil {
		return nil, err
	}
	return dedupeUsers(append(nameUsers, displayUsers...), limit), nil
}

// dedupeUsers 按 owner/name 去重并截断到 limit。
//
// 抽出为纯函数以便单测（合并多字段搜索结果、保持首次出现顺序）。
func dedupeUsers(users []*casdoorsdk.User, limit int) []*casdoorsdk.User {
	seen := make(map[string]struct{}, len(users))
	out := make([]*casdoorsdk.User, 0, len(users))
	for _, u := range users {
		if u == nil {
			continue
		}
		key := u.Owner + "/" + u.Name
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, u)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

// BatchDeleteUsers 批量删除用户（跳过受保护用户）
//
// 返回：删除成功数、被跳过（受保护）用户、失败明细
func (c *Client) BatchDeleteUsers(ctx context.Context, names []string) (int, []string, []string) {
	deleted := 0
	skipped := make([]string, 0)
	failed := make([]string, 0)
	for _, name := range names {
		if c.IsProtected(name) {
			skipped = append(skipped, name)
			continue
		}
		select {
		case <-ctx.Done():
			failed = append(failed, name)
			return deleted, skipped, failed
		default:
		}
		if _, err := c.DeleteUser(name); err != nil {
			failed = append(failed, name)
			continue
		}
		deleted++
	}
	c.InvalidateUsersCache()
	return deleted, skipped, failed
}

// ProtectedUsers 返回受保护用户列表
func (c *Client) ProtectedUsers() []string {
	return c.ProtectedList()
}

// GetSignupUrl 获取注册 URL
func (c *Client) GetSignupUrl(enablePassword bool, redirectUri string) string {
	return c.bizSDK.GetSignupUrl(enablePassword, redirectUri)
}

// GetUserProfileUrl 获取用户信息 URL
func (c *Client) GetUserProfileUrl(userName string, redirectUri string) string {
	return c.bizSDK.GetUserProfileUrl(userName, redirectUri)
}

// ParseJwtToken 解析 JWT Token 并获取用户信息
func (c *Client) ParseJwtToken(token string) (*casdoorsdk.Claims, error) {
	claims, err := c.bizSDK.ParseJwtToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT token: %w", err)
	}
	return claims, nil
}

// GetOAuthToken 通过 code 获取 OAuth Token
func (c *Client) GetOAuthToken(code string, state string) (string, error) {
	token, err := c.bizSDK.GetOAuthToken(code, state)
	if err != nil {
		return "", fmt.Errorf("failed to get OAuth token: %w", err)
	}
	return token.AccessToken, nil
}

// GetUser 获取用户信息
func (c *Client) GetUser(name string) (*casdoorsdk.User, error) {
	user, err := c.bizSDK.GetUser(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// GetUsers 获取用户列表
func (c *Client) GetUsers() ([]*casdoorsdk.User, error) {
	users, err := c.bizSDK.GetUsers()
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}
	return users, nil
}

// AddUser 添加用户
func (c *Client) AddUser(user *casdoorsdk.User) (bool, error) {
	// 设置默认组织
	if user.Owner == "" {
		user.Owner = c.organization
	}

	// 设置默认密码（如果未设置）
	if user.Password == "" {
		user.Password = c.config.DefaultPassword
	}

	affected, err := c.bizSDK.AddUser(user)
	if err != nil {
		return false, fmt.Errorf("failed to add user: %w", err)
	}
	return affected, nil
}

// UpdateUser 更新用户
func (c *Client) UpdateUser(user *casdoorsdk.User) (bool, error) {
	affected, err := c.bizSDK.UpdateUser(user)
	if err != nil {
		return false, fmt.Errorf("failed to update user: %w", err)
	}
	return affected, nil
}

// DeleteUser 删除用户
func (c *Client) DeleteUser(name string) (bool, error) {
	user := &casdoorsdk.User{
		Owner: c.organization,
		Name:  name,
	}
	affected, err := c.bizSDK.DeleteUser(user)
	if err != nil {
		return false, fmt.Errorf("failed to delete user: %w", err)
	}
	return affected, nil
}

// GetOrganizations 获取组织列表
//
// 注意：组织属于全局对象（owner=admin），需内置应用凭据。
func (c *Client) GetOrganizations() ([]*casdoorsdk.Organization, error) {
	sdk, err := c.adminClient()
	if err != nil {
		return nil, err
	}
	orgs, err := sdk.GetOrganizations()
	if err != nil {
		return nil, fmt.Errorf("failed to get organizations: %w", err)
	}
	return orgs, nil
}

// GetOrganization 获取组织信息
//
// 优先用内置应用凭据（可读任意组织）；未就绪时回退业务凭据
// （业务凭据可读自身组织，供启动健康检查与用户同步使用）。
func (c *Client) GetOrganization(name string) (*casdoorsdk.Organization, error) {
	if sdk, err := c.adminClient(); err == nil {
		if org, err := sdk.GetOrganization(name); err == nil && org != nil {
			return org, nil
		}
	}
	org, err := c.bizSDK.GetOrganization(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	return org, nil
}

// AddOrganization 添加组织
//
// 注意：组织属于全局对象，仅内置应用凭据有权限。
func (c *Client) AddOrganization(org *casdoorsdk.Organization) (bool, error) {
	sdk, err := c.adminClient()
	if err != nil {
		return false, err
	}
	affected, err := sdk.AddOrganization(org)
	if err != nil {
		return false, fmt.Errorf("failed to add organization: %w", err)
	}
	return affected, nil
}

// UpdateOrganization 更新组织
//
// 注意：组织属于全局对象，仅内置应用凭据有权限。
func (c *Client) UpdateOrganization(org *casdoorsdk.Organization) (bool, error) {
	sdk, err := c.adminClient()
	if err != nil {
		return false, err
	}
	affected, err := sdk.UpdateOrganization(org)
	if err != nil {
		return false, fmt.Errorf("failed to update organization: %w", err)
	}
	return affected, nil
}

// DeleteOrganization 删除组织
//
// 注意：组织属于全局对象，仅内置应用凭据有权限。
func (c *Client) DeleteOrganization(name string) (bool, error) {
	sdk, err := c.adminClient()
	if err != nil {
		return false, err
	}
	org := &casdoorsdk.Organization{
		Name: name,
	}
	affected, err := sdk.DeleteOrganization(org)
	if err != nil {
		return false, fmt.Errorf("failed to delete organization: %w", err)
	}
	return affected, nil
}

// SetUserPassword 设置用户密码（oldPassword 为空表示管理员重置）
func (c *Client) SetUserPassword(name, oldPassword, newPassword string) (bool, error) {
	ok, err := c.bizSDK.SetPassword(c.organization, name, oldPassword, newPassword)
	if err != nil {
		return false, fmt.Errorf("设置密码失败: %w", err)
	}
	return ok, nil
}

// DefaultPassword 返回配置的默认密码
func (c *Client) DefaultPassword() string { return c.config.DefaultPassword }

// SetUserAdmin 设置用户是否为管理员
func (c *Client) SetUserAdmin(name string, isAdmin bool) error {
	user, err := c.GetUser(name)
	if err != nil || user == nil {
		return fmt.Errorf("用户不存在")
	}
	user.IsAdmin = isAdmin
	if _, err := c.bizSDK.UpdateUser(user); err != nil {
		return fmt.Errorf("更新用户管理员标记失败: %w", err)
	}
	return nil
}

// HealthCheck 健康检查
func (c *Client) HealthCheck(ctx context.Context) error {
	// 尝试获取组织信息来验证连接
	_, err := c.GetOrganization(c.organization)
	if err != nil {
		return fmt.Errorf("Casdoor health check failed: %w", err)
	}
	return nil
}

// ProbeAPI 校验 endpoint 指向的是否为可用的 Casdoor API。
//
// /api/health 应返回 JSON（{"status":"ok"}）。若返回 HTML（例如端点误指向
// 前端页面、Ingress 首页、或是其他 Web 服务），后续 SDK 调用会报
// "invalid character '<' looking for beginning of value"，此处提前给出明确错误。
func ProbeAPI(endpoint string) error {
	if endpoint == "" {
		return fmt.Errorf("Casdoor 端点未配置（请设置 CASDOOR_ENDPOINT）")
	}
	url := strings.TrimRight(endpoint, "/") + "/api/health"
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("无法访问 Casdoor 端点 %s: %w（请确认地址可达、Service/DNS 正确）", url, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Casdoor 端点 %s 返回 HTTP %d，响应片段: %s", url, resp.StatusCode, bodySnippet(body))
	}

	var probe map[string]any
	if err := json.Unmarshal(body, &probe); err != nil {
		return fmt.Errorf("Casdoor 端点 %s 返回的不是 JSON（疑似误指向前端页面/其他服务，而非 Casdoor API），响应片段: %s",
			url, bodySnippet(body))
	}
	return nil
}

// bodySnippet 截断响应体用于错误提示
func bodySnippet(b []byte) string {
	s := strings.Join(strings.Fields(string(b)), " ")
	if s == "" {
		return "(空)"
	}
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}

// Ping 探测 Casdoor 服务是否可用（用于启动等待）。
// 除 HTTP 200 外还要求响应为 JSON，避免把「返回 HTML 的其他服务」误判为就绪。
func Ping(endpoint string) bool {
	return ProbeAPI(endpoint) == nil
}
