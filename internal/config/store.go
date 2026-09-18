package config

import "sync/atomic"

// Store 持有当前生效的配置快照，支持并发读与安全的整体替换。
//
// 设计（写时复制，copy-on-write）：
//   - 读者调用 Get() 获取一个不可变快照指针，可安全并发读取；
//   - 写者（页面保存设置）先 Clone() 出副本、在副本上修改，再 Store() 原子替换；
//   - 已发布的快照永不被修改，因此不存在读写同一字段的数据竞争。
//
// 这与直接共享 *Config 并就地修改不同：后者在并发读写下属于数据竞争
// （slice 头等可能被撕裂，Go 内存模型下为未定义行为）。
type Store struct {
	v atomic.Pointer[Config]
}

// NewStore 以初始配置创建 Store。
func NewStore(c *Config) *Store {
	s := &Store{}
	s.v.Store(c)
	return s
}

// Get 返回当前配置快照。调用方仅可读取，不得修改。
func (s *Store) Get() *Config {
	return s.v.Load()
}

// Store 原子替换为新的配置快照（旧快照保持不变，供在途读者继续使用）。
func (s *Store) Store(c *Config) {
	s.v.Store(c)
}

// Clone 深拷贝配置（浅拷贝整体 + 逐个复制可变切片），供写时复制使用。
//
// 说明：结构体内所有字段均为值类型或已知切片；只需深拷贝设置项会修改的
// 切片，避免副本与快照共享底层数组。
func (c *Config) Clone() *Config {
	cp := *c
	cp.Casdoor.ProtectedUsers = cloneStrings(c.Casdoor.ProtectedUsers)
	cp.Casdoor.RedirectURIs = cloneStrings(c.Casdoor.RedirectURIs)
	cp.Casdoor.AllowedRedirectHosts = cloneStrings(c.Casdoor.AllowedRedirectHosts)
	cp.Security.CORSAllowedOrigins = cloneStrings(c.Security.CORSAllowedOrigins)
	cp.Permission.AdminUsers = cloneStrings(c.Permission.AdminUsers)
	return &cp
}

func cloneStrings(in []string) []string {
	if in == nil {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}
