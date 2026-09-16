package model

import "time"

// ProtectedUser 受保护用户（不允许被同步或接口删除），页面可维护
type ProtectedUser struct {
	ID            int64     `db:"id" json:"id"`
	DomainAccount string    `db:"domain_account" json:"domainAccount"`
	Remark        string    `db:"remark" json:"remark"`
	CreatedAt     time.Time `db:"created_at" json:"createdAt"`
}
