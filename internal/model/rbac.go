package model

import "time"

// Role 角色：可复用的权限集合
type Role struct {
	ID          int64       `db:"id" json:"id"`
	Name        string      `db:"name" json:"name"`
	Code        string      `db:"code" json:"code"`
	Description string      `db:"description" json:"description"`
	Permissions StringArray `db:"permissions" json:"permissions"`
	CreatedAt   time.Time   `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time   `db:"updated_at" json:"updatedAt"`
}

// Team 团队：权限分配的主体
type Team struct {
	ID          int64     `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Code        string    `db:"code" json:"code"`
	Description string    `db:"description" json:"description"`
	Status      int       `db:"status" json:"status"`
	CreatedBy   string    `db:"created_by" json:"createdBy"`
	MemberCount int       `db:"member_count" json:"memberCount"`
	RoleCount   int       `db:"role_count" json:"roleCount"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
}

// TeamMember 团队成员
type TeamMember struct {
	ID          int64     `db:"id" json:"id"`
	TeamID      int64     `db:"team_id" json:"teamId"`
	Username    string    `db:"username" json:"username"`
	DisplayName string    `db:"display_name" json:"displayName"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
}

// 系统权限清单（前端展示与后端校验共用）
var AllPermissions = []string{
	"user:read", "user:write", "user:delete",
	"userfield:read", "userfield:write",
	"team:read", "team:write", "team:delete",
	"role:read", "role:write", "role:delete",
	"sync:read", "sync:write",
	"datasource:read", "datasource:write", "datasource:delete",
	"schedule:read", "schedule:write",
	"casdoor:read", "casdoor:write",
}
