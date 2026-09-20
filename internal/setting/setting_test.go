package setting

import (
	"testing"

	"github.com/iflyelf/flyiam/internal/config"
)

// TestView_Reveal 验证「可查看」的 secret（Reveal=true，如 Casdoor Client Secret）
// 返回明文，其余 secret 仍以占位符脱敏。
func TestView_Reveal(t *testing.T) {
	c := &config.Config{}
	c.Casdoor.ClientSecret = "real-client-secret"
	c.Admin.Password = "real-admin-password"

	svc := NewService(nil, config.NewStore(c))
	view := svc.View()

	byKey := make(map[string]map[string]interface{}, len(view))
	for _, it := range view {
		byKey[it["key"].(string)] = it
	}

	// Client Secret：明文 + secret=true + reveal=true
	cs, ok := byKey["casdoor.client_secret"]
	if !ok {
		t.Fatal("缺少 casdoor.client_secret 项")
	}
	if cs["value"] != "real-client-secret" {
		t.Errorf("client_secret 应返回明文，实际 %v", cs["value"])
	}
	if cs["reveal"] != true || cs["secret"] != true {
		t.Errorf("client_secret 标记异常: secret=%v reveal=%v", cs["secret"], cs["reveal"])
	}

	// 其它 secret：占位符 + reveal=false
	ap, ok := byKey["admin.password"]
	if !ok {
		t.Fatal("缺少 admin.password 项")
	}
	if ap["value"] != MaskedSecret {
		t.Errorf("admin.password 应脱敏为 %q，实际 %v", MaskedSecret, ap["value"])
	}
	if ap["reveal"] != false {
		t.Errorf("admin.password reveal 应为 false，实际 %v", ap["reveal"])
	}
}
