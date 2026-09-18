package config

import "testing"

// validBase 构造一份完整合法配置，供各用例按需破坏。
func validBase() *Config {
	c := &Config{}
	c.Database.Host = "localhost"
	c.JWT.Secret = "0123456789abcdef0123456789abcdef" // 32 位
	c.Admin.Password = "admin-password"
	c.Casdoor.Endpoint = "http://casdoor:8000"
	c.Casdoor.DefaultPassword = "user-password"
	c.Casdoor.AutoSetup = true
	return c
}

// TestValidate_PreDatabase 验证 Validate 仅校验与页面设置无关的前置项，
// 且不再因 Casdoor 凭据为空而拦截（Casdoor 校验已下沉到 ValidateCasdoor）。
func TestValidate_PreDatabase(t *testing.T) {
	// 前置项完整即可通过，即使 Casdoor 全部留空
	c := validBase()
	c.Casdoor.Endpoint = ""
	c.Casdoor.DefaultPassword = ""
	c.Casdoor.ClientId = ""
	c.Casdoor.ClientSecret = ""
	c.Casdoor.AutoSetup = false
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate 不应校验 Casdoor，但报错: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*Config)
	}{
		{"数据库缺失", func(c *Config) { c.Database.Host = ""; c.Database.DSN = "" }},
		{"JWT密钥缺失", func(c *Config) { c.JWT.Secret = "" }},
		{"JWT密钥过短", func(c *Config) { c.JWT.Secret = "short" }},
		{"管理员密码缺失", func(c *Config) { c.Admin.Password = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := validBase()
			tc.mutate(c)
			if err := c.Validate(); err == nil {
				t.Fatalf("期望报错，但校验通过")
			}
		})
	}
}

// TestValidateCasdoor 验证 Casdoor 校验：
//   - Endpoint / DefaultPassword 必填；
//   - AutoSetup=true（默认）时应用凭据可留空（程序自动创建并回填）；
//   - AutoSetup=false 时必须显式提供 ClientId / ClientSecret。
func TestValidateCasdoor(t *testing.T) {
	cases := []struct {
		name        string
		autoSetup   bool
		clientID    string
		clientSec   string
		endpoint    string
		defaultPwd  string
		wantErr     bool
		errContains string
	}{
		{"auto=true 凭据留空应通过", true, "", "", "http://casdoor:8000", "pwd", false, ""},
		{"auto=true 仅clientId应通过", true, "id", "", "http://casdoor:8000", "pwd", false, ""},
		{"auto=false 凭据留空应报错", false, "", "", "http://casdoor:8000", "pwd", true, "CASDOOR_CLIENT_ID"},
		{"auto=false 仅clientId应报错", false, "id", "", "http://casdoor:8000", "pwd", true, "CASDOOR_CLIENT_ID"},
		{"auto=false 提供凭据应通过", false, "id", "secret", "http://casdoor:8000", "pwd", false, ""},
		{"端点缺失应报错", true, "", "", "", "pwd", true, "Casdoor 端点"},
		{"默认密码缺失应报错", true, "", "", "http://casdoor:8000", "", true, "默认密码"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := validBase()
			c.Casdoor.AutoSetup = tc.autoSetup
			c.Casdoor.ClientId = tc.clientID
			c.Casdoor.ClientSecret = tc.clientSec
			c.Casdoor.Endpoint = tc.endpoint
			c.Casdoor.DefaultPassword = tc.defaultPwd

			err := c.ValidateCasdoor()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("期望报错，但校验通过")
				}
				if tc.errContains != "" && !contains(err.Error(), tc.errContains) {
					t.Fatalf("错误信息 %q 未包含 %q", err.Error(), tc.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("期望通过，但报错: %v", err)
			}
		})
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
