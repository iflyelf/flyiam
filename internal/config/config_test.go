package config

import "testing"

// validBase 构造一份除 Casdoor 凭据外均合法的配置，供凭据校验用例复用。
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

// TestValidate_CasdoorCredentialsWhenAutoSetup 验证：
//   - AutoSetup=true（默认）时，Casdoor 应用凭据可留空（由程序自动创建并回填）；
//   - AutoSetup=false 时，必须显式提供 ClientId / ClientSecret。
func TestValidate_CasdoorCredentialsWhenAutoSetup(t *testing.T) {
	cases := []struct {
		name        string
		autoSetup   bool
		clientID    string
		clientSec   string
		wantErr     bool
		errContains string
	}{
		{"auto=true 凭据留空应通过", true, "", "", false, ""},
		{"auto=true 仅clientId应通过", true, "id", "", false, ""},
		{"auto=true 提供凭据应通过", true, "id", "secret", false, ""},
		{"auto=false 凭据留空应报错", false, "", "", true, "CASDOOR_CLIENT_ID"},
		{"auto=false 仅clientId应报错", false, "id", "", true, "CASDOOR_CLIENT_ID"},
		{"auto=false 提供凭据应通过", false, "id", "secret", false, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := validBase()
			c.Casdoor.AutoSetup = tc.autoSetup
			c.Casdoor.ClientId = tc.clientID
			c.Casdoor.ClientSecret = tc.clientSec

			err := c.Validate()
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

// TestValidate_RequiredFields 验证其它必填项仍被强制校验。
func TestValidate_RequiredFields(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Config)
	}{
		{"数据库缺失", func(c *Config) { c.Database.Host = ""; c.Database.DSN = "" }},
		{"JWT密钥缺失", func(c *Config) { c.JWT.Secret = "" }},
		{"JWT密钥过短", func(c *Config) { c.JWT.Secret = "short" }},
		{"管理员密码缺失", func(c *Config) { c.Admin.Password = "" }},
		{"Casdoor端点缺失", func(c *Config) { c.Casdoor.Endpoint = "" }},
		{"Casdoor默认密码缺失", func(c *Config) { c.Casdoor.DefaultPassword = "" }},
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

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
