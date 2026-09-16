package datasource

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// Manager 数据源管理器
type Manager struct {
	sources map[string]DataSource
	mu      sync.RWMutex
}

// NewManager 创建数据源管理器
func NewManager() *Manager {
	return &Manager{
		sources: make(map[string]DataSource),
	}
}

// Register 注册数据源
func (m *Manager) Register(source DataSource) {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := source.Name()
	if _, exists := m.sources[name]; exists {
		log.Printf("⚠️  数据源 %s 已存在，将被覆盖", name)
	}

	m.sources[name] = source
	log.Printf("✅ 数据源 %s 注册成功 (启用: %v, 优先级: %d)",
		name, source.IsEnabled(), source.Priority())
}

// Unregister 取消注册数据源
func (m *Manager) Unregister(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sources, name)
	log.Printf("ℹ️  数据源 %s 已取消注册", name)
}

// GetSource 获取指定数据源
func (m *Manager) GetSource(name string) (DataSource, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	source, ok := m.sources[name]
	return source, ok
}

// GetEnabledSources 获取所有启用的数据源
func (m *Manager) GetEnabledSources() []DataSource {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var enabled []DataSource
	for _, source := range m.sources {
		if source.IsEnabled() {
			enabled = append(enabled, source)
		}
	}
	return enabled
}

// ListAll 列出所有数据源
func (m *Manager) ListAll() []DataSource {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var all []DataSource
	for _, source := range m.sources {
		all = append(all, source)
	}
	return all
}

// SyncAll 同步所有启用的数据源
func (m *Manager) SyncAll(ctx context.Context) error {
	sources := m.GetEnabledSources()
	if len(sources) == 0 {
		log.Println("⚠️  没有启用的数据源")
		return nil
	}

	log.Printf("🔄 开始同步 %d 个数据源...", len(sources))

	var lastErr error
	for _, source := range sources {
		log.Printf("📥 同步数据源: %s", source.Name())
		if err := source.Sync(ctx); err != nil {
			log.Printf("❌ 数据源 %s 同步失败: %v", source.Name(), err)
			lastErr = err
			// 继续同步其他数据源
			continue
		}
		log.Printf("✅ 数据源 %s 同步完成", source.Name())
	}

	if lastErr != nil {
		return fmt.Errorf("部分数据源同步失败: %w", lastErr)
	}

	log.Println("✅ 所有数据源同步完成")
	return nil
}

// HealthCheckAll 检查所有数据源健康状态
func (m *Manager) HealthCheckAll(ctx context.Context) map[string]error {
	sources := m.ListAll()
	results := make(map[string]error)

	for _, source := range sources {
		err := source.HealthCheck(ctx)
		results[source.Name()] = err
	}

	return results
}
