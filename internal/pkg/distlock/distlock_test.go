package distlock

import (
	"context"
	"testing"
)

// TestTryAcquire_NilDB 验证无原生连接时直接视为获取成功（退化为进程内互斥），
// 且 Release 可安全调用。
func TestTryAcquire_NilDB(t *testing.T) {
	lock, ok, err := TryAcquire(context.Background(), nil, 123)
	if err != nil {
		t.Fatalf("nil db 不应报错: %v", err)
	}
	if !ok {
		t.Fatal("nil db 应视为获取成功")
	}
	// 不应 panic
	lock.Release()
}

// TestLockRelease_Nil 验证 nil 锁的 Release 安全。
func TestLockRelease_Nil(t *testing.T) {
	var l *Lock
	l.Release() // 不应 panic
}
