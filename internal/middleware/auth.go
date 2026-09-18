package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/iflyelf/flyiam/internal/logic/rbac"
	"github.com/iflyelf/flyiam/internal/svc"
)

// ctxKey 上下文键类型
type ctxKey string

const (
	// CtxUsername 上下文中保存用户名
	CtxUsername ctxKey = "username"
	// CtxCasdoorAdmin 上下文中保存 Casdoor isAdmin 标记
	CtxCasdoorAdmin ctxKey = "casdoorAdmin"
	// AuthCookieName 登录凭证 Cookie 名（HttpOnly，避免 JS 读取，降低 XSS 窃取风险）
	AuthCookieName = "flyiam_token"
)

// Auth 鉴权中间件：校验本地 JWT 并注入用户名
func Auth(svcCtx *svc.ServiceContext) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				writeUnauthorized(w, "未登录")
				return
			}
			claims, err := parseToken(svcCtx.Config.JWT.Secret, token)
			if err != nil {
				writeUnauthorized(w, "登录已过期，请重新登录")
				return
			}
			username, _ := claims["username"].(string)
			if username == "" {
				writeUnauthorized(w, "无效的登录凭证")
				return
			}
			isAdmin, _ := claims["isAdmin"].(bool)
			ctx := context.WithValue(r.Context(), CtxUsername, username)
			ctx = context.WithValue(ctx, CtxCasdoorAdmin, isAdmin)
			next(w, r.WithContext(ctx))
		}
	}
}

// RequirePermission 权限中间件：在 Auth 之后使用
func RequirePermission(svcCtx *svc.ServiceContext, permission string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			username := Username(r.Context())
			if username == "" {
				writeUnauthorized(w, "未登录")
				return
			}
			// Casdoor 管理员标记直接放行
			if IsCasdoorAdmin(r.Context()) {
				next(w, r)
				return
			}
			logic := rbac.NewLogic(svcCtx.DB, svcCtx.Config.Permission.AdminUsers)
			allowed, err := logic.HasPermission(r.Context(), username, permission)
			if err != nil {
				writeForbidden(w, "权限校验失败: "+err.Error())
				return
			}
			if !allowed {
				writeForbidden(w, "无权限执行该操作")
				return
			}
			next(w, r)
		}
	}
}

// Username 从上下文获取用户名
func Username(ctx context.Context) string {
	if v, ok := ctx.Value(CtxUsername).(string); ok {
		return v
	}
	return ""
}

// IsCasdoorAdmin 从上下文获取 Casdoor 管理员标记
func IsCasdoorAdmin(ctx context.Context) bool {
	if v, ok := ctx.Value(CtxCasdoorAdmin).(bool); ok {
		return v
	}
	return false
}

// extractToken 提取登录凭证。
//
// 优先 HttpOnly Cookie（推荐，JS 无法读取，降低 XSS 窃取风险），
// 兼容 Authorization: Bearer（API 调用/旧会话）。
func extractToken(r *http.Request) string {
	if c, err := r.Cookie(AuthCookieName); err == nil && c.Value != "" {
		return c.Value
	}
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return r.URL.Query().Get("token")
}

// parseToken 解析本地 JWT
//
// 使用 WithValidMethods 限定仅接受 HS256，避免算法混淆
// （如 alg=none 或 RS256 公钥被当作 HMAC 密钥）。
func parseToken(secret, tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, err
	}
	return claims, nil
}

// writeUnauthorized 返回 401 JSON
func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"code":401,"message":"` + msg + `"}`))
}

// writeForbidden 返回 403 JSON
func writeForbidden(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(`{"code":403,"message":"` + msg + `"}`))
}
