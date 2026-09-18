package model

import "time"

// ApiToken 用户 API 令牌（用于服务间调用，如 Consul Manager 同步字段定义）。
//
// 安全设计：
//   - 仅存储令牌的 SHA256 哈希（明文只在创建时返回一次，不落库）；
//   - ExpiresAt 为 nil 表示永久有效；
//   - 可随时在页面吊销。
type ApiToken struct {
	ID        int64  `db:"id" json:"id"`
	Username  string `db:"username" json:"username"`
	Name      string `db:"name" json:"name"`
	TokenHash string `db:"token_hash" json:"-"`
	// TokenPrefix 令牌前缀（便于辨认，非敏感）
	TokenPrefix string     `db:"token_prefix" json:"tokenPrefix"`
	ExpiresAt   *time.Time `db:"expires_at" json:"expiresAt,omitempty"`
	Enabled     bool       `db:"enabled" json:"enabled"`
	LastUsedAt  *time.Time `db:"last_used_at" json:"lastUsedAt,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"createdAt"`
}

// Expired 判断令牌是否已过期
func (t *ApiToken) Expired() bool {
	if t.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*t.ExpiresAt)
}
