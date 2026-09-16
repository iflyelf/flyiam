package model

import (
	"database/sql"
	"time"
)

// User 用户模型
type User struct {
	ID              int64  `db:"id" json:"id"`
	DomainAccount   string `db:"domain_account" json:"domainAccount"`
	EmployeeCode    string `db:"employee_code" json:"employeeCode"`
	Name            string `db:"name" json:"name"`
	Phone           string `db:"phone" json:"phone"`
	WorkEmail       string `db:"work_email" json:"workEmail"`
	CompileType     string `db:"compile_type" json:"compileType"`
	SuperiorAccount string `db:"superior_account" json:"superiorAccount"`

	DeptIDLv0   string `db:"dept_id_lv0" json:"deptIdLv0"`
	DeptIDLv1   string `db:"dept_id_lv1" json:"deptIdLv1"`
	DeptIDLv2   string `db:"dept_id_lv2" json:"deptIdLv2"`
	DeptNameLv0 string `db:"dept_name_lv0" json:"deptNameLv0"`
	DeptNameLv1 string `db:"dept_name_lv1" json:"deptNameLv1"`
	DeptNameLv2 string `db:"dept_name_lv2" json:"deptNameLv2"`

	CasdoorSynced   bool         `db:"casdoor_synced" json:"casdoorSynced"`
	CasdoorUserID   string       `db:"casdoor_user_id" json:"casdoorUserId"`
	CasdoorSyncTime sql.NullTime `db:"casdoor_sync_time" json:"casdoorSyncTime"`

	Status   string `db:"status" json:"status"`
	Source   string `db:"source" json:"source"`
	Priority int    `db:"priority" json:"priority"`

	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// Department 部门模型
type Department struct {
	ID       int64  `db:"id"`
	DeptID   string `db:"dept_id"`
	DeptName string `db:"dept_name"`
	ParentID string `db:"parent_id"`
	Level    int    `db:"level"`
	FullPath string `db:"full_path"`

	CasdoorSynced   bool       `db:"casdoor_synced"`
	CasdoorOrgID    string     `db:"casdoor_org_id"`
	CasdoorSyncTime *time.Time `db:"casdoor_sync_time"`

	UserCount int       `db:"user_count"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// SyncLog 同步日志模型
type SyncLog struct {
	ID           int64     `db:"id" json:"id"`
	SyncType     string    `db:"sync_type" json:"syncType"`
	DataSource   string    `db:"data_source" json:"dataSource"`
	Status       string    `db:"status" json:"status"`
	TotalCount   int       `db:"total_count" json:"totalCount"`
	SuccessCount int       `db:"success_count" json:"successCount"`
	FailedCount  int       `db:"failed_count" json:"failedCount"`
	NewCount     int       `db:"new_count" json:"newCount"`
	UpdatedCount int       `db:"updated_count" json:"updatedCount"`
	DeletedCount int       `db:"deleted_count" json:"deletedCount"`
	ErrorMessage string    `db:"error_message" json:"errorMessage"`
	Details      string    `db:"details" json:"details"`
	DurationMs   int       `db:"duration_ms" json:"durationMs"`
	TriggeredBy  string    `db:"triggered_by" json:"triggeredBy"`
	StartedAt    time.Time `db:"started_at" json:"startedAt"`
	CompletedAt  NullTime  `db:"completed_at" json:"completedAt"`
}

// AuditLog 审计日志模型
type AuditLog struct {
	ID           int64     `db:"id"`
	UserID       string    `db:"user_id"`
	Username     string    `db:"username"`
	Action       string    `db:"action"`
	ResourceType string    `db:"resource_type"`
	ResourceID   string    `db:"resource_id"`
	ResourceName string    `db:"resource_name"`
	Details      string    `db:"details"`
	IPAddress    string    `db:"ip_address"`
	UserAgent    string    `db:"user_agent"`
	Status       string    `db:"status"`
	ErrorMessage string    `db:"error_message"`
	CreatedAt    time.Time `db:"created_at"`
}

// ResignedUser 离职用户模型
type ResignedUser struct {
	ID                int64      `db:"id"`
	DomainAccount     string     `db:"domain_account"`
	EmployeeCode      string     `db:"employee_code"`
	Name              string     `db:"name"`
	Phone             string     `db:"phone"`
	WorkEmail         string     `db:"work_email"`
	CompileType       string     `db:"compile_type"`
	SuperiorAccount   string     `db:"superior_account"`
	DeptIDLv0         string     `db:"dept_id_lv0"`
	DeptIDLv1         string     `db:"dept_id_lv1"`
	DeptIDLv2         string     `db:"dept_id_lv2"`
	DeptNameLv0       string     `db:"dept_name_lv0"`
	DeptNameLv1       string     `db:"dept_name_lv1"`
	DeptNameLv2       string     `db:"dept_name_lv2"`
	ResignedAt        time.Time  `db:"resigned_at"`
	OriginalCreatedAt *time.Time `db:"original_created_at"`
	Source            string     `db:"source"`
	CreatedAt         time.Time  `db:"created_at"`
}
