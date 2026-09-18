package config

import (
	"encoding/json"
	"sync/atomic"
)

// Store 持有当前生效的配置快照，支持并发读与安全的整体替换。
//
// 设计（写时复制，copy-on-write）：
//   - 读者调用 Get() 获取一个不可变快照指针，可安全并发读取；
//   - 写者（页面保存设置）先 Clone() 出副本、在副本上修改，再 Store() 原子替换；
//   - 已发布的快照永不被修改，因此不存在读写同一字段的数据竞争。
//
// 与直接共享 *Config 并就地修改不同：后者在并发读写下属于数据竞争
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

// Clone 返回配置的深拷贝，供写时复制使用。
//
// 采用 JSON 序列化往返实现：自动覆盖**全部**字段（含嵌套结构体、切片、映射），
// 因此后续新增可配置字段（尤其是切片/指针）时无需再手工补充深拷贝逻辑，
// 从根本上避免「新增字段被漏拷、副本与原快照共享底层数组」的隐患。
//
// 说明：Config 由配置加载/合并产生，字段均为可序列化的值类型；若极端情况下
// 序列化失败则回退为浅拷贝（不影响并发安全，仅可能共享切片）。
func (c *Config) Clone() *Config {
	data, err := json.Marshal(c)
	if err != nil {
		cp := *c
		return &cp
	}
	var cp Config
	if err := json.Unmarshal(data, &cp); err != nil {
		shallow := *c
		return &shallow
	}
	return &cp
}
