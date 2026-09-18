package config

import (
	"sync"
	"testing"
)

// TestStoreConcurrentReadWrite 验证快照存储的并发读写下无数据竞争（配合 -race）。
//
// 场景模拟：一个 goroutine 反复「克隆-修改-替换」（页面保存设置），
// 多个 goroutine 并发读取快照字段（请求读取配置）。
func TestStoreConcurrentReadWrite(t *testing.T) {
	c := &Config{}
	c.Admin.Username = "admin"
	c.Permission.AdminUsers = []string{"admin"}
	c.Security.CORSAllowedOrigins = []string{"https://a.example.com"}

	store := NewStore(c)

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// 读者：并发读取快照
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				snap := store.Get()
				_ = snap.Admin.Username
				_ = snap.Permission.AdminUsers
				_ = snap.Security.CORSAllowedOrigins
				_ = len(snap.Permission.AdminUsers)
			}
		}()
	}

	// 写者：写时复制替换
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			snap := store.Get().Clone()
			snap.Admin.Username = "admin"
			snap.Permission.AdminUsers = []string{"admin", "ops"}
			snap.Security.CORSAllowedOrigins = []string{"https://a.example.com", "https://b.example.com"}
			store.Store(snap)
		}
		close(stop)
	}()

	wg.Wait()

	// 已发布的旧快照不应被后续修改影响（写时复制语义）
	got := store.Get()
	if len(got.Permission.AdminUsers) != 2 {
		t.Fatalf("最终快照 AdminUsers = %v, 期望 2 项", got.Permission.AdminUsers)
	}
}

// TestCloneDeepCopiesSlices 验证 Clone 深拷贝切片，修改副本不影响原快照。
func TestCloneDeepCopiesSlices(t *testing.T) {
	c := &Config{}
	c.Permission.AdminUsers = []string{"admin"}
	c.Security.CORSAllowedOrigins = []string{"https://a"}
	c.Casdoor.ProtectedUsers = []string{"admin"}
	c.Casdoor.RedirectURIs = []string{"https://a/cb"}
	c.Casdoor.AllowedRedirectHosts = []string{"a"}

	cp := c.Clone()
	cp.Permission.AdminUsers[0] = "changed"
	cp.Security.CORSAllowedOrigins[0] = "changed"
	cp.Casdoor.ProtectedUsers[0] = "changed"
	cp.Casdoor.RedirectURIs[0] = "changed"
	cp.Casdoor.AllowedRedirectHosts[0] = "changed"

	if c.Permission.AdminUsers[0] != "admin" {
		t.Error("Clone 未深拷贝 Permission.AdminUsers")
	}
	if c.Security.CORSAllowedOrigins[0] != "https://a" {
		t.Error("Clone 未深拷贝 Security.CORSAllowedOrigins")
	}
	if c.Casdoor.ProtectedUsers[0] != "admin" {
		t.Error("Clone 未深拷贝 Casdoor.ProtectedUsers")
	}
	if c.Casdoor.RedirectURIs[0] != "https://a/cb" {
		t.Error("Clone 未深拷贝 Casdoor.RedirectURIs")
	}
	if c.Casdoor.AllowedRedirectHosts[0] != "a" {
		t.Error("Clone 未深拷贝 Casdoor.AllowedRedirectHosts")
	}
}
