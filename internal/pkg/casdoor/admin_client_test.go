package casdoor

import (
	"strings"
	"testing"
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
