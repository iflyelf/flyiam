package casdoor

import "testing"

// TestHasMoreUsersPage 验证分页遍历终止条件，防止边界处理出错导致死循环或漏读。
func TestHasMoreUsersPage(t *testing.T) {
	const pageSize = 200
	cases := []struct {
		name  string
		page  int
		total int
		got   int
		want  bool
	}{
		{"首次满页且总数更多", 1, 31310, 200, true},
		{"末页不足一页", 157, 31310, 110, false},    // 156*200=31200 < 31310 < 157*200=31400
		{"恰好整除后再取一页", 156, 31200, 200, false}, // 156*200=31200 已达 total
		{"总数远小于一页", 1, 50, 50, false},
		{"total=0 且无数据", 1, 0, 0, false},
		{"total=0 但有数据（防御）", 1, 0, 200, false},
		{"末页返回空", 200, 31310, 0, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasMoreUsersPage(tc.page, pageSize, tc.total, tc.got); got != tc.want {
				t.Fatalf("hasMoreUsersPage(page=%d,total=%d,got=%d) = %v, 期望 %v",
					tc.page, tc.total, tc.got, got, tc.want)
			}
		})
	}
}

// TestHasMoreUsersPage_Terminates 模拟完整翻页，确保能在有限步内读完且不重复。
func TestHasMoreUsersPage_Terminates(t *testing.T) {
	const pageSize = 200
	total := 31310

	pages := 0
	page := 1
	collected := 0
	for {
		got := pageSize
		if remaining := total - collected; remaining < pageSize {
			got = remaining
		}
		if got < 0 {
			got = 0
		}
		collected += got
		pages++
		if !hasMoreUsersPage(page, pageSize, total, got) {
			break
		}
		page++
		if pages > 1000 {
			t.Fatal("翻页未在合理步数内终止（疑似死循环）")
		}
	}
	if collected != total {
		t.Fatalf("共读取 %d，期望 %d", collected, total)
	}
}
