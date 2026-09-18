package model

import "time"

// ApiToken 用户 API 令牌（用于服务间调用，如 Consul Manager 同步字段定义）。
//
// 安全设计：
//   - 仅存储令牌的 SHA256 哈希（明文只在创建时返回一次，不落库）；
//   - ExpiresAt 无效（Valid=false）表示永久有效；
//   - 可随时在页面吊销。
//
// 说明：可空时间列用 NullTime 而非 *time.Time——go-zero sqlx 对指针字段会
// 强制分配后再 Scan，遇到 NULL 会报 "unsupported Scan ... into type *time.Time"。
type ApiToken struct {
	ID        int64  `db:"id" json:"id"`
	Username  string `db:"username" json:"username"`
	Name      string `db:"name" json:"name"`
	TokenHash string `db:"token_hash" json:"-"`
	// TokenPrefix 令牌前缀（便于辨认，非敏感）
	TokenPrefix string    `db:"token_prefix" json:"tokenPrefix"`
	ExpiresAt   NullTime  `db:"expires_at" json:"expiresAt"`
	Enabled     bool      `db:"enabled" json:"enabled"`
	LastUsedAt  NullTime  `db:"last_used_at" json:"lastUsedAt"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
}

// Expired 判断令牌是否已过期（永久有效返回 false）
func (t *ApiToken) Expired() bool {
	if !t.ExpiresAt.Valid {
		return false
	}
	return time.Now().After(t.ExpiresAt.Time)
}
