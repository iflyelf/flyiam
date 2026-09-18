package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/iflyelf/flyiam/internal/svc"
)

// ServiceTokenHeader 服务间调用凭证请求头
const ServiceTokenHeader = "X-Service-Token"

// ServiceAuth 服务间调用认证中间件（用于其它系统如 Consul Manager 拉取配置）。
//
// 安全约束：
//   - 需在配置中显式设置 SERVICE_TOKEN（环境变量），未设置则一律拒绝（不开放）；
//   - 使用常量时间比较，避免时序攻击；
//   - 仅用于受控的内部接口（如导出用户字段定义）。
func ServiceAuth(svcCtx *svc.ServiceContext) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			expected := svcCtx.Config.Service.Token
			if expected == "" {
				writeForbidden(w, "服务间接口未启用（未配置 SERVICE_TOKEN）")
				return
			}
			got := r.Header.Get(ServiceTokenHeader)
			if got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1 {
				writeUnauthorized(w, "服务间凭证无效")
				return
			}
			next(w, r)
		}
	}
}
