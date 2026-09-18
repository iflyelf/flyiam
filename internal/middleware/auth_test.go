package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestExtractToken 验证登录凭证提取顺序：
//  1. 优先 HttpOnly Cookie（浏览器登录态依赖此项）
//  2. 其次 Authorization: Bearer（API 调用/旧会话）
//  3. 最后查询参数 token
//
// 回归背景：handler 层曾使用另一份「只读 Bearer/query」的实现，
// 导致 Cookie 登录态在 /api/auth/userinfo 等接口失效（401 未登录）。
func TestExtractToken(t *testing.T) {
	cases := []struct {
		name   string
		cookie string
		bearer string
		query  string
		want   string
	}{
		{"仅 Cookie", "cookie-token", "", "", "cookie-token"},
		{"仅 Bearer", "", "bearer-token", "", "bearer-token"},
		{"仅 query", "", "", "query-token", "query-token"},
		{"Cookie 优先于 Bearer", "cookie-token", "bearer-token", "", "cookie-token"},
		{"Cookie 优先于 query", "cookie-token", "", "query-token", "cookie-token"},
		{"Bearer 优先于 query", "", "bearer-token", "query-token", "bearer-token"},
		{"空 Cookie 回退 Bearer", "", "bearer-token", "", "bearer-token"},
		{"全空", "", "", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			url := "/api/auth/userinfo"
			if tc.query != "" {
				url += "?token=" + tc.query
			}
			r := httptest.NewRequest("GET", url, nil)
			if tc.cookie != "" {
				r.AddCookie(&http.Cookie{Name: AuthCookieName, Value: tc.cookie})
			}
			if tc.bearer != "" {
				r.Header.Set("Authorization", "Bearer "+tc.bearer)
			}
			if got := ExtractToken(r); got != tc.want {
				t.Fatalf("ExtractToken() = %q, 期望 %q", got, tc.want)
			}
		})
	}
}
