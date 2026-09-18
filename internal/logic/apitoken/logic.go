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

	var expiresAt *time.Time
	if expiresInDays > 0 {
		t := time.Now().AddDate(0, 0, expiresInDays)
		expiresAt = &t
	}

	prefix := token
	if len(prefix) > 16 {
		prefix = prefix[:16] + "..."
	}

	var id int64
	query := `INSERT INTO api_tokens (username, name, token_hash, token_prefix, expires_at, enabled, created_at)
		VALUES ($1,$2,$3,$4,$5,TRUE,NOW()) RETURNING id`
	if err := l.db.QueryRowCtx(ctx, &id, query, username, name, hashToken(token), prefix, expiresAt); err != nil {
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
	// 异步更新最后使用时间（失败不影响校验）
	go func() {
		_, _ = l.db.ExecCtx(context.Background(),
			`UPDATE api_tokens SET last_used_at = NOW() WHERE id = $1`, t.ID)
	}()
	return t.Username, nil
}
