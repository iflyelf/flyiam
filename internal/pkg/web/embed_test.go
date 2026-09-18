package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 缺失的静态资源必须返回 404，不能回退 index.html。
// 否则浏览器把 HTML 当 JS 执行，动态 import 报
// "Failed to fetch dynamically imported module"（发版后旧 chunk 被删除的场景）。
func TestSPAHandler_MissingAssetReturns404(t *testing.T) {
	h := SPAHandler()
	for _, p := range []string{
		"/assets/Dashboard-CSe9RM1t.js",
		"/assets/old-chunk.css",
		"/favicon-removed.svg",
	} {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s 期望 404，实际 %d（Content-Type=%s）", p, rec.Code, rec.Header().Get("Content-Type"))
		}
	}
}

// 前端路由（无扩展名）应回退 index.html，保证 SPA 刷新可用。
func TestSPAHandler_RouteFallsBackToIndex(t *testing.T) {
	h := SPAHandler()
	for _, p := range []string{"/", "/users", "/user-fields"} {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("%s 期望 200，实际 %d", p, rec.Code)
			continue
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
			t.Errorf("%s 期望 text/html，实际 %s", p, ct)
		}
	}
}

// index.html 不可缓存，避免发版后仍使用旧 HTML。
func TestSPAHandler_IndexNoStore(t *testing.T) {
	h := SPAHandler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "no-store") {
		t.Fatalf("index.html Cache-Control 应包含 no-store，实际 %q", cc)
	}
}

func TestIsStaticFile(t *testing.T) {
	cases := map[string]bool{
		"assets/index-abc.js": true,
		"favicon.svg":         true,
		"logo.png":            true,
		"users":               false,
		"user-fields":         false,
		"":                    false,
	}
	for in, want := range cases {
		if got := isStaticFile(in); got != want {
			t.Errorf("isStaticFile(%q) = %v，期望 %v", in, got, want)
		}
	}
}
