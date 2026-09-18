// Package apitoken 用户 API 令牌业务逻辑（页面可生成/吊销，支持永久有效）
package apitoken

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"github.com/iflyelf/flyiam/internal/model"
)

// TokenPrefix 令牌前缀（便于识别来源）
const TokenPrefix = "flyiam_"

// Logic API 令牌业务逻辑
type Logic struct {
	db sqlx.SqlConn
}

// NewLogic 创建 API 令牌业务逻辑
func NewLogic(db sqlx.SqlConn) *Logic {
	return &Logic{db: db}
}

// hashToken 计算令牌哈希（仅存哈希，不存明文）
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// genToken 生成随机令牌
func genToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return TokenPrefix + hex.EncodeToString(b), nil
}

const tokenColumns = `id, username, name, token_hash, token_prefix,
	expires_at, enabled, last_used_at, created_at`

// List 列出指定用户的全部令牌（不含哈希）
func (l *Logic) List(ctx context.Context, username string) ([]*model.ApiToken, error) {
	var list []*model.ApiToken
	query := `SELECT ` + tokenColumns + ` FROM api_tokens
		WHERE username = $1 ORDER BY id DESC`
	if err := l.db.QueryRowsCtx(ctx, &list, query, username); err != nil {
		return nil, fmt.Errorf("查询令牌失败: %w", err)
	}
	return list, nil
}

// Create 生成新令牌，返回明文（仅此一次）
func (l *Logic) Create(ctx context.Context, username, name string, expiresInDays int) (string, *model.ApiToken, error) {
	token, err := genToken()
	if err != nil {
		return "", nil, fmt.Errorf("生成令牌失败: %w", err)
	}
	if name == "" {
		name = "API 令牌"
	}

	// 可空时间用 model.NullTime：既避免 (*time.Time)(nil) 触发 go-zero
	// sqlx.writeValue 的 String() panic，也避免读取时无法扫描 NULL。
	expiresAt := model.NullTime{}
	if expiresInDays > 0 {
		expiresAt = model.NullTime{Time: time.Now().AddDate(0, 0, expiresInDays), Valid: true}
	}

	prefix := token
	if len(prefix) > 16 {
		prefix = prefix[:16] + "..."
	}

	var id int64
	query := `INSERT INTO api_tokens (username, name, token_hash, token_prefix, expires_at, enabled, created_at)
		VALUES ($1,$2,$3,$4,$5,TRUE,NOW()) RETURNING id`
	// 传 driver.Value（nil 或 time.Time），使 SQL 日志可读且兼容 go-zero 格式化。
	expiresVal, _ := expiresAt.Value()
	if err := l.db.QueryRowCtx(ctx, &id, query, username, name, hashToken(token), prefix, expiresVal); err != nil {
		return "", nil, fmt.Errorf("保存令牌失败: %w", err)
	}

	return token, &model.ApiToken{
		ID: id, Username: username, Name: name, TokenPrefix: prefix,
		ExpiresAt: expiresAt, Enabled: true, CreatedAt: time.Now(),
	}, nil
}

// Revoke 吊销令牌（仅限本人）
func (l *Logic) Revoke(ctx context.Context, id int64, username string) error {
	if _, err := l.db.ExecCtx(ctx, `DELETE FROM api_tokens WHERE id = $1 AND username = $2`, id, username); err != nil {
		return fmt.Errorf("吊销令牌失败: %w", err)
	}
	return nil
}

// lastUsedSem 限制「异步更新最后使用时间」的并发 goroutine 数。
//
// 背景：Validate 每次调用都会更新 last_used_at，若直接 `go func()` 则高并发下
// goroutine 数量无界增长。这里用带缓冲信号量兜底：并发已满时直接跳过更新
// （last_used_at 非关键数据，允许丢失），保证 goroutine 数恒定有界。
var lastUsedSem = make(chan struct{}, 64)

// touchLastUsed 异步更新令牌最后使用时间（有界并发，失败不影响校验）。
func (l *Logic) touchLastUsed(id int64) {
	select {
	case lastUsedSem <- struct{}{}:
		go func() {
			defer func() { <-lastUsedSem }()
			_, _ = l.db.ExecCtx(context.Background(),
				`UPDATE api_tokens SET last_used_at = NOW() WHERE id = $1`, id)
		}()
	default:
		// 并发已满，跳过本次更新（非关键路径）
	}
}

// Validate 校验令牌，返回归属用户名（有效则同时更新最后使用时间）
func (l *Logic) Validate(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("令牌为空")
	}
	var t model.ApiToken
	query := `SELECT ` + tokenColumns + ` FROM api_tokens WHERE token_hash = $1 LIMIT 1`
	if err := l.db.QueryRowCtx(ctx, &t, query, hashToken(token)); err != nil {
		return "", fmt.Errorf("令牌无效")
	}
	if !t.Enabled {
		return "", fmt.Errorf("令牌已禁用")
	}
	if t.Expired() {
		return "", fmt.Errorf("令牌已过期")
	}
	l.touchLastUsed(t.ID)
	return t.Username, nil
}
