package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/iflyelf/flyiam/internal/logic/rbac"
	"github.com/iflyelf/flyiam/internal/middleware"
	"github.com/iflyelf/flyiam/internal/model"
	"github.com/iflyelf/flyiam/internal/pkg/casdoor"
	"github.com/iflyelf/flyiam/internal/svc"
)

// oauthStateCookie OAuth state 校验用 Cookie 名
const oauthStateCookie = "flyiam_oauth_state"

// randomState 生成随机 OAuth state
func randomState() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// requestBaseURL 根据请求推导外部可访问的基础地址
//
// 优先使用反向代理透传的 X-Forwarded-* 头，否则回退 Host 头，
// 保证不同域名/IP 下都能得到正确的回调地址（零硬编码）。
func requestBaseURL(r *http.Request) string {
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}

// LoginHandler 跳转 Casdoor 登录（OAuth2 授权码模式）
func LoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svcCtx.CasdoorClient == nil {
			fail(w, http.StatusInternalServerError, "Casdoor 未初始化")
			return
		}
		redirectUri := requestBaseURL(r) + "/api/auth/callback"

		// 回调地址不存在于应用白名单时自动追加，避免登录报
		// "Redirect URI doesn't exist in the allowed Redirect URI list"
		if changed, err := svcCtx.CasdoorClient.EnsureRedirectURI(redirectUri); err != nil {
			log.Printf("⚠️ 自动追加回调地址失败: %v", err)
		} else if changed {
			log.Printf("✅ 已自动追加回调地址到应用白名单: %s", redirectUri)
		}

		// 生成随机 state 并写入 Cookie，回调时校验，防止登录 CSRF
		state := randomState()
		http.SetCookie(w, &http.Cookie{
			Name:     oauthStateCookie,
			Value:    state,
			Path:     "/",
			MaxAge:   600,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
		})

		loginURL := svcCtx.CasdoorClient.GetSigninUrl(redirectUri, state)

		if r.URL.Query().Get("format") == "json" {
			ok(w, map[string]string{"loginUrl": loginURL, "redirectUri": redirectUri})
			return
		}
		http.Redirect(w, r, loginURL, http.StatusFound)
	}
}

// CallbackHandler Casdoor OAuth 回调：换取 Token、签发本地 JWT 并跳转前端
func CallbackHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			fail(w, http.StatusBadRequest, "缺少授权码 code")
			return
		}
		if svcCtx.CasdoorClient == nil {
			fail(w, http.StatusInternalServerError, "Casdoor 未初始化")
			return
		}

		// 校验 OAuth state，防止登录 CSRF（Cookie 与回调参数必须一致）
		stateCookie, err := r.Cookie(oauthStateCookie)
		if err != nil || stateCookie.Value == "" || r.URL.Query().Get("state") != stateCookie.Value {
			log.Printf("🚫 OAuth state 校验失败（可能存在 CSRF）")
			fail(w, http.StatusBadRequest, "登录校验失败（state 不匹配），请重新登录")
			return
		}
		// 一次性使用：校验后立即失效
		http.SetCookie(w, &http.Cookie{Name: oauthStateCookie, Value: "", Path: "/", MaxAge: -1})

		accessToken, err := svcCtx.CasdoorClient.GetOAuthToken(code, "")
		if err != nil {
			fail(w, http.StatusBadGateway, "获取 Casdoor Token 失败: "+err.Error())
			return
		}
		claims, err := svcCtx.CasdoorClient.ParseToken(accessToken)
		if err != nil {
			fail(w, http.StatusBadGateway, "解析 Casdoor Token 失败: "+err.Error())
			return
		}

		// 仅管理员可登录本系统：普通用户即使通过 Casdoor 认证也拒绝进入。
		// 管理员判定：Casdoor 管理员标记，或配置的超级管理员名单（ADMIN_USERS）。
		if !isAdminUser(svcCtx, claims) {
			log.Printf("🚫 拒绝非管理员登录: %s（仅管理员可访问本系统）", claims.Name)
			denyLogin(w, r, "无权访问",
				fmt.Sprintf("账号「%s」不是本系统管理员，无法登录。请联系管理员开通权限。", claims.DisplayName))
			return
		}

		// 签发本地 JWT
		localToken, err := signLocalToken(svcCtx, claims)
		if err != nil {
			fail(w, http.StatusInternalServerError, "签发本地 Token 失败: "+err.Error())
			return
		}

		// 跳转前端回调页，携带本地 token
		target := fmt.Sprintf("/callback?token=%s&name=%s&displayName=%s&email=%s",
			localToken, claims.Name, claims.DisplayName, claims.Email)
		http.Redirect(w, r, target, http.StatusFound)
	}
}

// isAdminUser 判断 Casdoor 用户是否为本系统管理员。
//
// 规则（满足其一即可）：
//  1. Casdoor 中该用户被标记为管理员（claims.IsAdmin）；
//  2. 用户名在配置的超级管理员名单（Permission.AdminUsers，通过
//     PERMISSION_ADMIN_USERS 环境变量配置，无代码内默认值）。
func isAdminUser(svcCtx *svc.ServiceContext, claims *casdoor.Claims) bool {
	if claims == nil {
		return false
	}
	if claims.IsAdmin {
		return true
	}
	logic := rbac.NewLogic(svcCtx.DB, svcCtx.Config.Permission.AdminUsers)
	return logic.IsSuperAdmin(claims.Name)
}

// denyLogin 拒绝登录：浏览器跳转场景返回可读的提示页（非 JSON）
func denyLogin(w http.ResponseWriter, r *http.Request, title, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="zh">
<head><meta charset="utf-8"><title>%s</title>
<style>body{font-family:system-ui,-apple-system,sans-serif;background:#f5f5f5;display:flex;align-items:center;justify-content:center;height:100vh;margin:0}
.box{background:#fff;padding:40px 48px;border-radius:12px;box-shadow:0 6px 24px rgba(0,0,0,.08);text-align:center;max-width:420px}
h1{font-size:20px;margin:0 0 12px;color:#c0392b}p{color:#555;line-height:1.7;margin:0 0 20px}
a{display:inline-block;padding:8px 20px;background:#2f6fed;color:#fff;text-decoration:none;border-radius:8px}</style>
</head>
<body><div class="box"><h1>%s</h1><p>%s</p><a href="/login">返回登录</a></div></body>
</html>`, title, title, message)
}

// UserInfoHandler 返回当前登录用户信息（含有效权限列表）
func UserInfoHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			fail(w, http.StatusUnauthorized, "未登录")
			return
		}
		claims, err := parseLocalToken(svcCtx, token)
		if err != nil {
			fail(w, http.StatusUnauthorized, "Token 无效: "+err.Error())
			return
		}

		username, _ := claims["username"].(string)
		isAdmin, _ := claims["isAdmin"].(bool)
		logic := rbac.NewLogic(svcCtx.DB, svcCtx.Config.Permission.AdminUsers)
		isSuperAdmin := isAdmin || logic.IsSuperAdmin(username)

		// 防御：历史签发的非管理员 Token 在过期前仍可能被使用，此处再次拦截
		if !isSuperAdmin {
			fail(w, http.StatusForbidden, "无权访问：仅管理员可使用本系统")
			return
		}

		permissions := []string{}
		if isSuperAdmin {
			permissions = append(permissions, model.AllPermissions...)
		} else if p, err := logic.UserPermissions(r.Context(), username); err == nil {
			permissions = p
		}

		ok(w, map[string]interface{}{
			"username":     username,
			"name":         claims["name"],
			"email":        claims["email"],
			"phone":        claims["phone"],
			"avatar":       claims["avatar"],
			"isAdmin":      isAdmin,
			"isSuperAdmin": isSuperAdmin,
			"permissions":  permissions,
		})
	}
}

// ChangePasswordHandler 当前用户修改自己的密码（需校验原密码）
func ChangePasswordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svcCtx.CasdoorClient == nil {
			fail(w, http.StatusServiceUnavailable, "Casdoor 未初始化")
			return
		}
		username := middleware.Username(r.Context())
		if username == "" {
			fail(w, http.StatusUnauthorized, "未登录")
			return
		}
		var req struct {
			OldPassword string `json:"oldPassword"`
			NewPassword string `json:"newPassword"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败")
			return
		}
		if req.NewPassword == "" {
			fail(w, http.StatusBadRequest, "新密码不能为空")
			return
		}
		if okPwd, err := svcCtx.CasdoorClient.SetUserPassword(username, req.OldPassword, req.NewPassword); err != nil || !okPwd {
			msg := "修改密码失败，请检查原密码是否正确"
			if err != nil {
				msg += ": " + friendlyCasdoorErr(err.Error())
			}
			fail(w, http.StatusBadGateway, msg)
			return
		}
		ok(w, nil)
	}
}

// AuthConfigHandler 返回前端登录所需的公开配置
func AuthConfigHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok(w, map[string]string{
			"appName":   svcCtx.Config.Name,
			"loginPath": "/api/auth/login",
		})
	}
}

// signLocalToken 基于 Casdoor 用户信息签发本地 JWT
func signLocalToken(svcCtx *svc.ServiceContext, claims *casdoor.Claims) (string, error) {
	expire := svcCtx.Config.JWT.AccessExpire
	if expire <= 0 {
		expire = 7200
	}
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      claims.Name,
		"username": claims.Name,
		"name":     claims.DisplayName,
		"email":    claims.Email,
		"phone":    claims.Phone,
		"avatar":   claims.Avatar,
		"isAdmin":  claims.IsAdmin,
		"iat":      now.Unix(),
		"exp":      now.Add(time.Duration(expire) * time.Second).Unix(),
	})
	return token.SignedString([]byte(svcCtx.Config.JWT.Secret))
}

// parseLocalToken 解析本地 JWT
func parseLocalToken(svcCtx *svc.ServiceContext, tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(svcCtx.Config.JWT.Secret), nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("无效 Token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("Token 声明错误")
	}
	return claims, nil
}

// extractToken 从请求头或查询参数提取 Token
func extractToken(r *http.Request) string {
	if h := r.Header.Get("Authorization"); len(h) > 7 && h[:7] == "Bearer " {
		return h[7:]
	}
	return r.URL.Query().Get("token")
}
