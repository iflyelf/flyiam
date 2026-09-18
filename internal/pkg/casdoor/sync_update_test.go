package casdoor

import (
	"testing"

	casdoorsdk "github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

// updateExistingUser 不应原地修改传入的 existing 对象（避免污染调用方共享数据）
func TestUpdateExistingUser_DoesNotMutateInput(t *testing.T) {
	c := &Client{
		config:       &Config{Endpoint: "http://127.0.0.1:1", CountryCode: "CN"},
		organization: "flyiam",
		application:  "flyiam",
		// 提供一个指向不可达地址的实例，使 UpdateUser 返回错误而非 nil 解引用
		bizSDK: casdoorsdk.NewClient("http://127.0.0.1:1", "id", "secret", "", "flyiam", "flyiam"),
	}

	existing := &casdoorsdk.User{
		Owner:       "flyiam",
		Name:        "zhangsan",
		DisplayName: "旧姓名",
		Email:       "old@x.com",
		Phone:       "13800000000",
		Tag:         "manual",
		Properties:  map[string]string{"empCode": "E1", "keep": "v"},
	}
	// 记录原始快照
	origName := existing.DisplayName
	origEmail := existing.Email
	origPhone := existing.Phone
	origTag := existing.Tag
	origProps := map[string]string{}
	for k, v := range existing.Properties {
		origProps[k] = v
	}

	// 调用更新（会因网络失败，但对象的修改应发生在副本上）
	_ = c.updateExistingUser(existing, SyncUser{
		DomainAccount: "zhangsan",
		Name:          "新姓名",
		Email:         "new@x.com",
		Phone:         "13900000000",
		Affiliation:   "研发部",
		Properties:    map[string]string{"empCode": "E2", "deptNameLv1": "研发部"},
	})

	if existing.DisplayName != origName {
		t.Errorf("DisplayName 被原地修改: %q -> %q", origName, existing.DisplayName)
	}
	if existing.Email != origEmail || existing.Phone != origPhone {
		t.Errorf("Email/Phone 被原地修改: %q/%q -> %q/%q", origEmail, origPhone, existing.Email, existing.Phone)
	}
	if existing.Tag != origTag {
		t.Errorf("Tag 被原地修改: %q -> %q", origTag, existing.Tag)
	}
	for k, v := range origProps {
		if existing.Properties[k] != v {
			t.Errorf("Properties[%s] 被原地修改: %q -> %q", k, v, existing.Properties[k])
		}
	}
	if _, ok := existing.Properties["deptNameLv1"]; ok {
		t.Error("新字段 deptNameLv1 被写入了原对象")
	}
}
