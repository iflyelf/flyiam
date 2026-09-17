package casdoor

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 模拟 SPA/前端：所有路径都返回 200 + HTML
func TestProbeAPI_HTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<!DOCTYPE html><html><body>FlyIAM</body></html>"))
	}))
	defer srv.Close()

	err := ProbeAPI(srv.URL)
	if err == nil {
		t.Fatal("期望报错，实际通过（HTML 端点被误判为 Casdoor API）")
	}
	if !strings.Contains(err.Error(), "不是 JSON") {
		t.Fatalf("错误信息不符: %v", err)
	}
	t.Logf("HTML 端点错误信息: %v", err)
	if Ping(srv.URL) {
		t.Fatal("Ping 不应把 HTML 端点判定为就绪")
	}
}

// 正常 Casdoor API
func TestProbeAPI_JSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	if err := ProbeAPI(srv.URL); err != nil {
		t.Fatalf("JSON 端点应通过，实际: %v", err)
	}
	if !Ping(srv.URL) {
		t.Fatal("Ping 应判定 JSON 端点为就绪")
	}
}
