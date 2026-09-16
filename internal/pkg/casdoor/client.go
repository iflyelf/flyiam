package casdoor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	casdoorsdk "github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"github.com/golang-jwt/jwt/v4"
)

// Client Casdoor 客户端
type Client struct {
	config       *Config
	authConfig   *casdoorsdk.AuthConfig
	organization string
	application  string

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

	// 初始化 Casdoor SDK
	casdoorsdk.InitConfig(
		cfg.Endpoint,
		cfg.ClientId,
		cfg.ClientSecret,
		cfg.Certificate,
		cfg.OrganizationName,
		cfg.ApplicationName,
	)

	protected := make(map[string]struct{}, len(cfg.ProtectedUsers)+1)
	for _, name := range cfg.ProtectedUsers {
		if name != "" {
			protected[name] = struct{}{}
		}
	}
	protected["admin"] = struct{}{}

	return &Client{
		config:       cfg,
		authConfig:   authConfig,
		organization: cfg.OrganizationName,
		application:  cfg.ApplicationName,
		protected:    protected,
	}, nil
}

// Organization 返回当前组织名
func (c *Client) Organization() string { return c.organization }

// Application 返回当前应用名
func (c *Client) Application() string { return c.application }

// ListApplications 获取全部应用
func (c *Client) ListApplications() ([]*casdoorsdk.Application, error) {
	apps, err := casdoorsdk.GetApplications()
	if err != nil {
		return nil, fmt.Errorf("获取应用列表失败: %w", err)
	}
	return apps, nil
}

// GetApplication 获取单个应用
func (c *Client) GetApplication(name string) (*casdoorsdk.Application, error) {
	app, err := casdoorsdk.GetApplication(name)
	if err != nil {
		return nil, fmt.Errorf("获取应用失败: %w", err)
	}
	return app, nil
}

// AddApplication 新增应用
func (c *Client) AddApplication(app *casdoorsdk.Application) (bool, error) {
	ok, err := casdoorsdk.AddApplication(app)
	if err != nil {
		return false, fmt.Errorf("新增应用失败: %w", err)
	}
	return ok, nil
}

// UpdateApplication 更新应用
func (c *Client) UpdateApplication(app *casdoorsdk.Application) (bool, error) {
	ok, err := casdoorsdk.UpdateApplication(app)
	if err != nil {
		return false, fmt.Errorf("更新应用失败: %w", err)
	}
	return ok, nil
}

// DeleteApplication 删除应用
func (c *Client) DeleteApplication(name string) (bool, error) {
	app := &casdoorsdk.Application{Owner: "admin", Name: name}
	ok, err := casdoorsdk.DeleteApplication(app)
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
	app, err := casdoorsdk.GetApplication(c.application)
	if err != nil || app == nil {
		return false, fmt.Errorf("获取应用失败: %w", err)
	}
	for _, u := range app.RedirectUris {
		if u == redirectURI {
			return false, nil
		}
	}
	app.RedirectUris = append(app.RedirectUris, redirectURI)
	if _, err := casdoorsdk.UpdateApplication(app); err != nil {
		return false, fmt.Errorf("追加回调地址失败: %w", err)
	}
	return true, nil
}

// ListOrganizations 获取全部组织
//
// 说明：SDK 的 GetOrganizations 会按当前组织名作为 owner 过滤，
// 而 Casdoor 中组织的 owner 为 admin（非当前组织名），会导致返回空列表。
// 因此这里以应用凭据直接调用接口且不传 owner，返回全部组织。
func (c *Client) ListOrganizations() ([]*casdoorsdk.Organization, error) {
	endpoint := strings.TrimRight(c.config.Endpoint, "/") + "/api/get-organizations"
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}
	req.SetBasicAuth(c.config.ClientId, c.config.ClientSecret)

	resp, err := http.DefaultClient.Do(req)
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
func (c *Client) GetSigninUrl(redirectUri string) string {
	base := strings.TrimRight(c.PublicEndpoint(), "/")
	return fmt.Sprintf(
		"%s/login/oauth/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=openid profile email&state=%s",
		base,
		url.QueryEscape(c.config.ClientId),
		url.QueryEscape(redirectUri),
		url.QueryEscape(redirectUri),
	)
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
	return casdoorsdk.GetPaginationUsers(page, pageSize, queryMap)
}

// GetUserCount 获取组织用户总数
func (c *Client) GetUserCount() (int, error) {
	_, total, err := c.GetUsersPage(1, 1, "", "")
	return total, err
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
	return casdoorsdk.GetSignupUrl(enablePassword, redirectUri)
}

// GetUserProfileUrl 获取用户信息 URL
func (c *Client) GetUserProfileUrl(userName string, redirectUri string) string {
	return casdoorsdk.GetUserProfileUrl(userName, redirectUri)
}

// ParseJwtToken 解析 JWT Token 并获取用户信息
func (c *Client) ParseJwtToken(token string) (*casdoorsdk.Claims, error) {
	claims, err := casdoorsdk.ParseJwtToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT token: %w", err)
	}
	return claims, nil
}

// GetOAuthToken 通过 code 获取 OAuth Token
func (c *Client) GetOAuthToken(code string, state string) (string, error) {
	token, err := casdoorsdk.GetOAuthToken(code, state)
	if err != nil {
		return "", fmt.Errorf("failed to get OAuth token: %w", err)
	}
	return token.AccessToken, nil
}

// GetUser 获取用户信息
func (c *Client) GetUser(name string) (*casdoorsdk.User, error) {
	user, err := casdoorsdk.GetUser(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// GetUsers 获取用户列表
func (c *Client) GetUsers() ([]*casdoorsdk.User, error) {
	users, err := casdoorsdk.GetUsers()
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

	affected, err := casdoorsdk.AddUser(user)
	if err != nil {
		return false, fmt.Errorf("failed to add user: %w", err)
	}
	return affected, nil
}

// UpdateUser 更新用户
func (c *Client) UpdateUser(user *casdoorsdk.User) (bool, error) {
	affected, err := casdoorsdk.UpdateUser(user)
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
	affected, err := casdoorsdk.DeleteUser(user)
	if err != nil {
		return false, fmt.Errorf("failed to delete user: %w", err)
	}
	return affected, nil
}

// GetOrganizations 获取组织列表
func (c *Client) GetOrganizations() ([]*casdoorsdk.Organization, error) {
	orgs, err := casdoorsdk.GetOrganizations()
	if err != nil {
		return nil, fmt.Errorf("failed to get organizations: %w", err)
	}
	return orgs, nil
}

// GetOrganization 获取组织信息
func (c *Client) GetOrganization(name string) (*casdoorsdk.Organization, error) {
	org, err := casdoorsdk.GetOrganization(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	return org, nil
}

// AddOrganization 添加组织
func (c *Client) AddOrganization(org *casdoorsdk.Organization) (bool, error) {
	affected, err := casdoorsdk.AddOrganization(org)
	if err != nil {
		return false, fmt.Errorf("failed to add organization: %w", err)
	}
	return affected, nil
}

// UpdateOrganization 更新组织
func (c *Client) UpdateOrganization(org *casdoorsdk.Organization) (bool, error) {
	affected, err := casdoorsdk.UpdateOrganization(org)
	if err != nil {
		return false, fmt.Errorf("failed to update organization: %w", err)
	}
	return affected, nil
}

// DeleteOrganization 删除组织
func (c *Client) DeleteOrganization(name string) (bool, error) {
	org := &casdoorsdk.Organization{
		Name: name,
	}
	affected, err := casdoorsdk.DeleteOrganization(org)
	if err != nil {
		return false, fmt.Errorf("failed to delete organization: %w", err)
	}
	return affected, nil
}

// SetUserPassword 设置用户密码（oldPassword 为空表示管理员重置）
func (c *Client) SetUserPassword(name, oldPassword, newPassword string) (bool, error) {
	ok, err := casdoorsdk.SetPassword(c.organization, name, oldPassword, newPassword)
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
	if _, err := casdoorsdk.UpdateUser(user); err != nil {
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

// Ping 探测 Casdoor 服务是否可用（用于启动等待）
func Ping(endpoint string) bool {
	if endpoint == "" {
		return false
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(strings.TrimRight(endpoint, "/") + "/api/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
