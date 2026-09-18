package casdoor

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	casdoorsdk "github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

// BootstrapResult 初始化结果
type BootstrapResult struct {
	ClientId     string
	ClientSecret string
	Organization string
	Application  string
}

// EnsureSetup 确保 Casdoor 中已存在本系统所需的组织、应用、证书与管理员账号。
//
// 设计目标（零手工配置）：
//  1. 复用 Casdoor 自带的内置应用（app-built-in）完成首次鉴权，其凭据从共享数据库读取；
//  2. 幂等地创建/更新业务组织与应用（含回调地址、证书、密码登录等）；
//  3. 创建组织管理员账号，密码取配置的默认密码。
//
// 返回的 ClientId/ClientSecret 为业务应用最终生效的凭据，供运行时客户端使用。
func EnsureSetup(db *sql.DB, cfg *Config) (*BootstrapResult, error) {
	// 0. 端点自检：确认 Endpoint 指向的是 Casdoor API 而非前端页面/其他服务，
	//    否则 SDK 调用会以 JSON 解析错误（invalid character '<'）失败，难以定位。
	if err := ProbeAPI(cfg.Endpoint); err != nil {
		return nil, err
	}

	// 1. 读取内置应用凭据（Casdoor 首次启动会自动创建 app-built-in）
	builtinID, builtinSecret, err := ReadBuiltinApp(db)
	if err != nil {
		return nil, err
	}

	// 使用内置凭据构建**独立的实例客户端**用于管理操作。
	// 不使用 casdoorsdk.InitConfig：它写入 SDK 包级全局变量且无锁，
	// 与业务凭据的全局配置会互相覆盖（顺序敏感），改用实例彻底隔离。
	admin := casdoorsdk.NewClient(cfg.Endpoint, builtinID, builtinSecret, "", "built-in", "app-built-in")

	// 2. 确保组织存在
	if err := ensureOrganization(admin, cfg); err != nil {
		return nil, err
	}

	// 3. 确保应用存在（clientId/secret 以配置为准，未配置则自动生成并持久化）
	clientID, clientSecret, err := ensureApplication(admin, cfg)
	if err != nil {
		return nil, err
	}

	// 4. 确保组织管理员账号存在
	if err := ensureAdminUser(admin, cfg); err != nil {
		return nil, err
	}

	return &BootstrapResult{
		ClientId:     clientID,
		ClientSecret: clientSecret,
		Organization: cfg.OrganizationName,
		Application:  cfg.ApplicationName,
	}, nil
}

// ReadBuiltinApp 从数据库读取内置应用（app-built-in）凭据。
//
// Casdoor 的 API 授权模型中，只有 built-in 组织的身份才是全局管理员
// （authz.IsAllowed 中 appUser.IsGlobalAdmin() 要求 Owner=="built-in"）。
// 因此「应用 / 组织管理」这类全局接口必须用该凭据，业务应用凭据会报
// "Unauthorized operation" / "Please sign in first"。
func ReadBuiltinApp(db *sql.DB) (string, string, error) {
	var clientID, clientSecret string
	query := `SELECT client_id, client_secret FROM casdoor_application WHERE name = 'app-built-in' LIMIT 1`
	if err := db.QueryRow(query).Scan(&clientID, &clientSecret); err != nil {
		return "", "", fmt.Errorf("读取 Casdoor 内置应用失败（请确认 Casdoor 已启动并完成初始化）: %w", err)
	}
	if clientID == "" || clientSecret == "" {
		return "", "", fmt.Errorf("Casdoor 内置应用凭据为空")
	}
	return clientID, clientSecret, nil
}

// ensureOrganization 确保业务组织存在，并配置中文与地区
func ensureOrganization(sdk *casdoorsdk.Client, cfg *Config) error {
	org, err := sdk.GetOrganization(cfg.OrganizationName)
	if err != nil || org == nil || org.Name == "" {
		org = &casdoorsdk.Organization{
			Owner:              "admin",
			Name:               cfg.OrganizationName,
			DisplayName:        cfg.OrganizationDisplayName,
			PasswordType:       "bcrypt",
			CountryCodes:       []string{"CN"},
			Languages:          []string{"zh"},
			DefaultApplication: cfg.ApplicationName,
			DefaultPassword:    cfg.DefaultPassword,
		}
		if _, err := sdk.AddOrganization(org); err != nil {
			return fmt.Errorf("创建组织失败: %w", err)
		}
		log.Printf("✅ Casdoor 组织已创建: %s", cfg.OrganizationName)
		return nil
	}

	// 已存在：补齐关键配置（幂等）
	changed := false
	if org.PasswordType == "" {
		org.PasswordType = "bcrypt"
		changed = true
	}
	if len(org.CountryCodes) == 0 {
		org.CountryCodes = []string{"CN"}
		changed = true
	}
	if len(org.Languages) == 0 {
		org.Languages = []string{"zh"}
		changed = true
	}
	if org.DefaultApplication == "" {
		org.DefaultApplication = cfg.ApplicationName
		changed = true
	}
	if org.DisplayName == "" {
		org.DisplayName = cfg.OrganizationDisplayName
		changed = true
	}
	if changed {
		if _, err := sdk.UpdateOrganization(org); err != nil {
			return fmt.Errorf("更新组织失败: %w", err)
		}
		log.Printf("✅ Casdoor 组织已更新: %s", cfg.OrganizationName)
	}
	return nil
}

// ensureApplication 确保业务应用存在，返回最终生效的 clientId/clientSecret
func ensureApplication(sdk *casdoorsdk.Client, cfg *Config) (string, string, error) {
	app, err := sdk.GetApplication(cfg.ApplicationName)
	exists := err == nil && app != nil && app.Name != ""
	if !exists {
		clientID := cfg.ClientId
		clientSecret := cfg.ClientSecret
		if clientID == "" {
			clientID = genHex(10) // 20 位
		}
		if clientSecret == "" {
			clientSecret = genHex(20) // 40 位
		}
		app = &casdoorsdk.Application{
			Owner:               "admin",
			Name:                cfg.ApplicationName,
			DisplayName:         cfg.ApplicationDisplayName,
			Organization:        cfg.OrganizationName,
			ClientId:            clientID,
			ClientSecret:        clientSecret,
			Cert:                cfg.Certificate,
			RedirectUris:        cfg.RedirectURIs,
			EnablePassword:      true,
			EnableSigninSession: true,
			EnableAutoSignin:    true,
			EnableCodeSignin:    true,
			GrantTypes:          []string{"authorization_code", "password", "client_credentials", "refresh_token"},
			SigninMethods: []*casdoorsdk.SigninMethod{
				{Name: "Password", DisplayName: "密码", Rule: "All"},
				{Name: "Code", DisplayName: "验证码", Rule: "All"},
			},
			ExpireInHours:        2,
			RefreshExpireInHours: 168,
			CookieExpireInHours:  720,
		}
		if _, err := sdk.AddApplication(app); err != nil {
			return "", "", fmt.Errorf("创建应用失败: %w", err)
		}
		log.Printf("✅ Casdoor 应用已创建: %s (clientId=%s)", cfg.ApplicationName, clientID)
		return clientID, clientSecret, nil
	}

	// 已存在：补齐回调地址与证书，确保登录可用
	changed := false
	if app.Cert == "" {
		app.Cert = cfg.Certificate
		changed = true
	}
	for _, uri := range cfg.RedirectURIs {
		if uri == "" {
			continue
		}
		if !containsStr(app.RedirectUris, uri) {
			app.RedirectUris = append(app.RedirectUris, uri)
			changed = true
		}
	}
	if app.Organization == "" {
		app.Organization = cfg.OrganizationName
		changed = true
	}
	if !app.EnablePassword {
		app.EnablePassword = true
		changed = true
	}
	if changed {
		if _, err := sdk.UpdateApplication(app); err != nil {
			return "", "", fmt.Errorf("更新应用失败: %w", err)
		}
		log.Printf("✅ Casdoor 应用已更新: %s", cfg.ApplicationName)
	}
	return app.ClientId, app.ClientSecret, nil
}

// ensureAdminUser 确保组织管理员账号存在（幂等）
func ensureAdminUser(sdk *casdoorsdk.Client, cfg *Config) error {
	admin := &casdoorsdk.User{
		Owner:             cfg.OrganizationName,
		Name:              "admin",
		DisplayName:       "管理员",
		Password:          cfg.DefaultPassword,
		Email:             "admin@flyiam.local",
		Phone:             "",
		CountryCode:       cfg.CountryCode,
		Type:              "normal-user",
		IsAdmin:           true,
		SignupApplication: cfg.ApplicationName,
		Properties:        map[string]string{"empCode": "admin"},
	}
	if _, err := sdk.AddUser(admin); err != nil {
		// 已存在视为成功（幂等）
		if strings.Contains(err.Error(), "already exists") {
			return nil
		}
		return fmt.Errorf("创建管理员账号失败: %w", err)
	}
	log.Printf("✅ Casdoor 管理员账号已创建: %s/admin", cfg.OrganizationName)
	return nil
}

// genHex 生成指定字节长度的随机十六进制字符串
func genHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// 兜底：基于时间生成，保证不中断启动
		return fmt.Sprintf("%0*x", n*2, time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// containsStr 判断字符串切片是否包含目标值
func containsStr(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}
