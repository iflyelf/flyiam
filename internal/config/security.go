package config

import (
	"net/http"
	"strings"
)

// CookieConfig 登录 Cookie 配置（支持同源与跨域两种部署形态）
type CookieConfig struct {
	// SameSite: lax / strict / none（前后端跨域部署必须 none）
	SameSite string
	// Secure: auto / true / false
	Secure string
	// Domain: Cookie 作用域（跨子域共享时设为 .example.com），留空为当前域
	Domain string
}

// SameSiteMode 解析 SameSite 配置为 http.SameSite
func (c CookieConfig) SameSiteMode() http.SameSite {
	switch strings.ToLower(strings.TrimSpace(c.SameSite)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

// ResolveSecure 判断是否设置 Cookie 的 Secure 属性。
//
//   - 显式 true/false：直接采用；
//   - auto（默认）：SameSite=None 时浏览器强制要求 Secure，故置 true；
//     否则按请求是否 HTTPS（TLS 或 X-Forwarded-Proto）自动判断。
func (c CookieConfig) ResolveSecure(r *http.Request) bool {
	switch strings.ToLower(strings.TrimSpace(c.Secure)) {
	case "true":
		return true
	case "false":
		return false
	default:
		if c.SameSiteMode() == http.SameSiteNoneMode {
			return true
		}
		if r != nil && r.TLS != nil {
			return true
		}
		if r != nil && strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			return true
		}
		return false
	}
}

// NewAuthCookie 按配置构造登录 Cookie
func (c CookieConfig) NewAuthCookie(name, value string, maxAge int, r *http.Request) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		Domain:   c.Domain,
		HttpOnly: true,
		SameSite: c.SameSiteMode(),
		Secure:   c.ResolveSecure(r),
	}
}

// ClearAuthCookie 按配置构造用于清除登录 Cookie 的 Cookie
func (c CookieConfig) ClearAuthCookie(name string) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Domain:   c.Domain,
		HttpOnly: true,
		SameSite: c.SameSiteMode(),
		Secure:   c.ResolveSecure(nil),
	}
}
