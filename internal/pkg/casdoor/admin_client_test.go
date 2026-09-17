package casdoor

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// 有内置应用凭据时，应创建独立的 adminSDK 实例
func TestNewClient_WithAdminCredentials(t *testing.T) {
	c, err := NewClient(&Config{
		Endpoint:          "http://casdoor:8000",
		ClientId:          "biz-id",
		ClientSecret:      "biz-secret",
		OrganizationName:  "flyiam",
		ApplicationName:   "flyiam",
		AdminClientId:     "builtin-id",
		AdminClientSecret: "builtin-secret",
	})
	if err != nil {
		t.Fatalf("NewClient 失败: %v", err)
	}
	if c.adminSDK == nil {
		t.Fatal("提供了内置应用凭据，adminSDK 不应为 nil")
	}
	if _, err := c.adminClient(); err != nil {
		t.Fatalf("adminClient() 不应报错: %v", err)
	}
	// adminSDK 必须是独立实例，其 ClientId 为内置应用凭据
	if c.adminSDK.ClientId != "builtin-id" {
		t.Fatalf("adminSDK.ClientId = %q, 期望 builtin-id", c.adminSDK.ClientId)
	}
	if c.adminSDK.OrganizationName != "built-in" {
		t.Fatalf("adminSDK.OrganizationName = %q, 期望 built-in", c.adminSDK.OrganizationName)
	}
}

// 未提供内置应用凭据时，全局管理接口应给出明确错误而非静默失败
func TestNewClient_WithoutAdminCredentials(t *testing.T) {
	c, err := NewClient(&Config{
		Endpoint:         "http://casdoor:8000",
		ClientId:         "biz-id",
		ClientSecret:     "biz-secret",
		OrganizationName: "flyiam",
		ApplicationName:  "flyiam",
	})
	if err != nil {
		t.Fatalf("NewClient 失败: %v", err)
	}
	if c.adminSDK != nil {
		t.Fatal("未提供内置应用凭据时 adminSDK 应为 nil")
	}

	wantMsg := "管理凭据未就绪"
	if _, err := c.ListApplications(); err == nil || !strings.Contains(err.Error(), wantMsg) {
		t.Fatalf("ListApplications 应返回含 %q 的错误，实际: %v", wantMsg, err)
	}
	if _, err := c.GetApplication("flyiam"); err == nil || !strings.Contains(err.Error(), wantMsg) {
		t.Fatalf("GetApplication 应返回含 %q 的错误，实际: %v", wantMsg, err)
	}
	if _, err := c.AddOrganization(nil); err == nil || !strings.Contains(err.Error(), wantMsg) {
		t.Fatalf("AddOrganization 应返回含 %q 的错误，实际: %v", wantMsg, err)
	}
	if _, err := c.DeleteOrganization("x"); err == nil || !strings.Contains(err.Error(), wantMsg) {
		t.Fatalf("DeleteOrganization 应返回含 %q 的错误，实际: %v", wantMsg, err)
	}
	if _, err := c.ListOrganizations(); err == nil || !strings.Contains(err.Error(), wantMsg) {
		t.Fatalf("ListOrganizations 应返回含 %q 的错误，实际: %v", wantMsg, err)
	}
}

// 注入 loader 后，管理接口应在首次使用时按需补读凭据（模拟 Casdoor 晚就绪）
func TestAdminClient_LazyLoad(t *testing.T) {
	c := newTestClient(t)

	var calls int32
	c.SetAdminCredentialLoader(func() (string, string, error) {
		atomic.AddInt32(&calls, 1)
		return "builtin-id", "builtin-secret", nil
	})

	if c.AdminReady() {
		t.Fatal("初始不应就绪")
	}
	sdk, err := c.adminClient()
	if err != nil {
		t.Fatalf("adminClient() 应自动补读成功，实际: %v", err)
	}
	if sdk.ClientId != "builtin-id" {
		t.Fatalf("ClientId = %q，期望 builtin-id", sdk.ClientId)
	}
	if !c.AdminReady() {
		t.Fatal("补读后应就绪")
	}
	if n := atomic.LoadInt32(&calls); n != 1 {
		t.Fatalf("loader 调用 %d 次，期望 1 次", n)
	}
}

// 后台 Watch 应在 loader 恢复后自动补齐凭据，无需重启
func TestWatchAdminCredentials_EventuallyReady(t *testing.T) {
	c := newTestClient(t)

	var calls int32
	c.SetAdminCredentialLoader(func() (string, string, error) {
		// 前两次模拟 Casdoor 尚未建表，第三次成功
		if atomic.AddInt32(&calls, 1) < 3 {
			return "", "", fmt.Errorf("relation \"casdoor_application\" does not exist")
		}
		return "builtin-id", "builtin-secret", nil
	})

	c.WatchAdminCredentials(5*time.Millisecond, 100)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if c.AdminReady() {
			return // 自动恢复成功
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("后台重试后仍未就绪（loader 调用 %d 次）", atomic.LoadInt32(&calls))
}

// loader 持续失败时，管理接口应返回明确错误且不 panic
func TestAdminClient_LoaderAlwaysFails(t *testing.T) {
	c := newTestClient(t)
	c.SetAdminCredentialLoader(func() (string, string, error) {
		return "", "", fmt.Errorf("relation \"casdoor_application\" does not exist")
	})

	if _, err := c.adminClient(); err == nil {
		t.Fatal("凭据不可用时应返回错误")
	} else if !strings.Contains(err.Error(), "管理凭据未就绪") {
		t.Fatalf("错误信息不符: %v", err)
	}
}

// newTestClient 构造一个不含内置凭据的测试客户端
func newTestClient(t *testing.T) *Client {
	t.Helper()
	c, err := NewClient(&Config{
		Endpoint:         "http://casdoor:8000",
		ClientId:         "biz-id",
		ClientSecret:     "biz-secret",
		OrganizationName: "flyiam",
		ApplicationName:  "flyiam",
	})
	if err != nil {
		t.Fatalf("NewClient 失败: %v", err)
	}
	return c
}
