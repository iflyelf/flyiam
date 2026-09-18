package apitoken

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// TestCreateIntegration 复现并验证 API 令牌创建。
//
// 需要环境变量 TEST_DATABASE_URL；未设置则跳过。
func TestCreateIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL 未设置，跳过")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS api_tokens (
			id BIGSERIAL PRIMARY KEY,
			username VARCHAR(100) NOT NULL,
			name VARCHAR(100) NOT NULL DEFAULT '',
			token_hash VARCHAR(64) NOT NULL UNIQUE,
			token_prefix VARCHAR(64) NOT NULL DEFAULT '',
			expires_at TIMESTAMPTZ,
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			last_used_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`); err != nil {
		t.Fatalf("建表失败: %v", err)
	}

	l := NewLogic(sqlx.NewSqlConnFromDB(db))
	ctx := context.Background()

	t.Run("永久有效", func(t *testing.T) {
		token, info, err := l.Create(ctx, "ut-user", "diag", 0)
		if err != nil {
			t.Fatalf("Create 失败: %v", err)
		}
		if token == "" || info == nil || info.ExpiresAt != nil {
			t.Fatalf("返回异常: token=%q info=%+v", token, info)
		}
		defer db.Exec(`DELETE FROM api_tokens WHERE id=$1`, info.ID)
	})

	t.Run("固定有效期", func(t *testing.T) {
		token, info, err := l.Create(ctx, "ut-user", "diag", 30)
		if err != nil {
			t.Fatalf("Create 失败: %v", err)
		}
		if info.ExpiresAt == nil || !info.ExpiresAt.After(time.Now()) {
			t.Fatalf("有效期异常: %+v", info.ExpiresAt)
		}
		defer db.Exec(`DELETE FROM api_tokens WHERE id=$1`, info.ID)
		_ = token
	})
}
