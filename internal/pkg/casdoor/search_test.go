package casdoor

import (
	"testing"

	casdoorsdk "github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

// TestDedupeUsers 验证合并多字段搜索结果时的去重、顺序与截断。
func TestDedupeUsers(t *testing.T) {
	u := func(owner, name string) *casdoorsdk.User {
		return &casdoorsdk.User{Owner: owner, Name: name}
	}

	cases := []struct {
		name  string
		in    []*casdoorsdk.User
		limit int
		want  []string // owner/name
	}{
		{
			name:  "按 owner/name 去重",
			in:    []*casdoorsdk.User{u("flyiam", "zhangsan"), u("flyiam", "zhangsan"), u("flyiam", "lisi")},
			limit: 10,
			want:  []string{"flyiam/zhangsan", "flyiam/lisi"},
		},
		{
			name:  "同名不同 owner 不去重",
			in:    []*casdoorsdk.User{u("flyiam", "admin"), u("built-in", "admin")},
			limit: 10,
			want:  []string{"flyiam/admin", "built-in/admin"},
		},
		{
			name:  "保持首次出现顺序",
			in:    []*casdoorsdk.User{u("flyiam", "c"), u("flyiam", "a"), u("flyiam", "b")},
			limit: 10,
			want:  []string{"flyiam/c", "flyiam/a", "flyiam/b"},
		},
		{
			name:  "按 limit 截断",
			in:    []*casdoorsdk.User{u("flyiam", "a"), u("flyiam", "b"), u("flyiam", "c")},
			limit: 2,
			want:  []string{"flyiam/a", "flyiam/b"},
		},
		{
			name:  "跳过 nil",
			in:    []*casdoorsdk.User{nil, u("flyiam", "a"), nil},
			limit: 10,
			want:  []string{"flyiam/a"},
		},
		{
			name:  "空输入",
			in:    nil,
			limit: 10,
			want:  []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := dedupeUsers(tc.in, tc.limit)
			if len(got) != len(tc.want) {
				t.Fatalf("数量 = %d，期望 %d（got=%v）", len(got), len(tc.want), got)
			}
			for i, w := range tc.want {
				if got[i].Owner+"/"+got[i].Name != w {
					t.Fatalf("第 %d 项 = %s/%s，期望 %s", i, got[i].Owner, got[i].Name, w)
				}
			}
		})
	}
}
