package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/iflyelf/flyiam/internal/logic/apitoken"
	"github.com/iflyelf/flyiam/internal/logic/rbac"
	"github.com/iflyelf/flyiam/internal/svc"
)

// ServiceTokenHeader 服务间调用凭证请求头
const ServiceTokenHeader = "X-Service-Token"

// ServiceAuth 服务间调用认证中间件（供其它系统如 Consul Manager 拉取配置）。
//
// 支持两种凭证（任一生效即可）：
//  1. 用户 API 令牌（推荐）：页面生成，可设有效期/永久，归属用户须为管理员；
//     通过 `Authorization: Bearer <token>` 或 `X-Service-Token: <token>` 传递。
//  2. 全局 SERVICE_TOKEN（可选，环境变量；用于无页面的自动化场景）。
//
// 两者都未配置时一律拒绝（不开放）。
func ServiceAuth(svcCtx *svc.ServiceContext) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			got := extractServiceToken(r)

			// 1. 用户 API 令牌
			if got != "" && strings.HasPrefix(got, apitoken.TokenPrefix) {
				username, err := apitoken.NewLogic(svcCtx.DB).Validate(r.Context(), got)
				if err != nil {
					writeUnauthorized(w, "API 令牌无效或已过期: "+err.Error())
					return
				}
				if !isAdminUserByUsername(svcCtx, username) {
					writeForbidden(w, "令牌归属用户不是管理员，无权访问")
					return
				}
				next(w, r)
				return
			}

			// 2. 全局 SERVICE_TOKEN（常量时间比较）
			expected := svcCtx.Config().Service.Token
			if expected != "" && got != "" &&
				subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1 {
				next(w, r)
				return
			}

			if expected == "" && got == "" {
				writeForbidden(w, "服务间接口未启用（请生成 API 令牌，或配置 SERVICE_TOKEN）")
				return
			}
			writeUnauthorized(w, "服务间凭证无效")
		}
	}
}

// extractServiceToken 提取服务间凭证（X-Service-Token 优先，其次 Bearer）
func extractServiceToken(r *http.Request) string {
	if v := r.Header.Get(ServiceTokenHeader); v != "" {
		return v
	}
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}

// isAdminUserByUsername 判断用户是否为管理员（Casdoor isAdmin 或超级管理员名单）。
// Casdoor 不可用时仅信任超级管理员名单（fail-closed 倾向）。
func isAdminUserByUsername(svcCtx *svc.ServiceContext, username string) bool {
	logic := rbac.NewLogic(svcCtx.DB, svcCtx.Config().Permission.AdminUsers)
	if logic.IsSuperAdmin(username) {
		return true
	}
	if svcCtx.Casdoor() != nil {
		if u, err := svcCtx.Casdoor().GetUser(username); err == nil && u != nil {
			return u.IsAdmin
		}
	}
	return false
}
