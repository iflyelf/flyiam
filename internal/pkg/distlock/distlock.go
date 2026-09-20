// Package distlock 提供基于 PostgreSQL 会话级 advisory lock 的跨副本互斥。
//
// 适用场景：定时任务在多副本部署下，每个副本都有独立的进程内调度器与
// 互斥标记；进程内的 atomic/mutex 无法跨 Pod 生效，会导致同一时刻多个
// 副本重复执行同一任务。本包用数据库全局锁保证「同一任务同一时刻只有
// 一个副本在执行」。
//
// 实现要点：advisory lock 是「会话级」的，必须绑定同一条连接，故这里
// 取一条专用连接持有锁，释放后再归还连接池；副本异常退出时连接断开，
// 锁由数据库自动释放，不会死锁。
package distlock

import (
	"context"
	"database/sql"
	"time"
)

// Lock 已持有的分布式锁
type Lock struct {
	conn *sql.Conn
	key  int64
}

// TryAcquire 尝试获取 advisory lock。
//
// 返回：
//   - (*Lock, true, nil)  获取成功，调用方须在结束时 Release
//   - (nil, false, nil)   锁已被其它副本持有（正常跳过，非错误）
//   - (nil, false, err)   发生错误（连接失败等）
//
// db 为 nil 时不启用跨副本互斥（直接视为获取成功），便于单测与无原生
// 连接的场景。
func TryAcquire(ctx context.Context, db *sql.DB, key int64) (*Lock, bool, error) {
	if db == nil {
		return nil, true, nil
	}

	// 获取连接加短超时：连接池耗尽时快速失败，避免长时间阻塞调用方
	acquireCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, err := db.Conn(acquireCtx)
	if err != nil {
		return nil, false, err
	}

	var ok bool
	if err := conn.QueryRowContext(acquireCtx, "SELECT pg_try_advisory_lock($1)", key).Scan(&ok); err != nil {
		_ = conn.Close()
		return nil, false, err
	}
	if !ok {
		// 未获取到锁：立即归还连接，不占用
		_ = conn.Close()
		return nil, false, nil
	}
	return &Lock{conn: conn, key: key}, true, nil
}

// Release 释放锁并归还连接。可安全地对 nil 调用。
func (l *Lock) Release() {
	if l == nil || l.conn == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = l.conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", l.key)
	_ = l.conn.Close()
	l.conn = nil
}
