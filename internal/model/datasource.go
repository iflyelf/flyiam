package model

import "time"

// DataSourceConfig 数据源配置（页面可管理，存储于数据库，零硬编码）
type DataSourceConfig struct {
	ID           int64  `db:"id" json:"id"`
	Name         string `db:"name" json:"name"`                  // 数据源名称（唯一）
	Type         string `db:"type" json:"type"`                  // 类型: httpapi
	Enabled      bool   `db:"enabled" json:"enabled"`            // 是否启用
	URL          string `db:"url" json:"url"`                    // 接口地址
	Method       string `db:"method" json:"method"`              // 请求方法: GET/POST
	AuthType     string `db:"auth_type" json:"authType"`         // 认证: none/bearer/basic/apikey
	AuthToken    string `db:"auth_token" json:"authToken"`       // Token/APIKey
	AuthUsername string `db:"auth_username" json:"authUsername"` // Basic 用户名
	AuthPassword string `db:"auth_password" json:"authPassword"` // Basic 密码
	Timeout      int    `db:"timeout" json:"timeout"`            // 超时(秒)
	SyncInterval string `db:"sync_interval" json:"syncInterval"` // 同步间隔
	AutoSync     bool   `db:"auto_sync" json:"autoSync"`         // 启动/定时自动同步
	Priority     int    `db:"priority" json:"priority"`          // 优先级
	PageSize     int    `db:"page_size" json:"pageSize"`         // 分页大小
	// FieldMapping 外部字段名 → 内部字段键 的映射（JSON 对象字符串）。
	// 例：{"DOMACT":"domainAccount","NAME":"name","DEPT_NAME_LV0":"deptNameLv0"}
	// 目标键可为内置字段（domainAccount/name/...）或 user_field_defs 中的自定义字段键。
	// 为空时回退到内置的默认字段名（向后兼容）。
	FieldMapping string    `db:"field_mapping" json:"fieldMapping"`
	Remark       string    `db:"remark" json:"remark"` // 备注
	CreatedAt    time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt    time.Time `db:"updated_at" json:"updatedAt"`
}

// UserView 统一的用户视图（本地库 + Casdoor 合并展示）
type UserView struct {
	Source          string `json:"source"`          // local / casdoor
	DomainAccount   string `json:"domainAccount"`   // 域账号
	EmployeeCode    string `json:"employeeCode"`    // 工号
	Name            string `json:"name"`            // 姓名
	Avatar          string `json:"avatar"`          // 头像 URL
	Phone           string `json:"phone"`           // 手机号
	WorkEmail       string `json:"workEmail"`       // 邮箱
	CompileType     string `json:"compileType"`     // 编制类型
	SuperiorAccount string `json:"superiorAccount"` // 直属上级
	DeptNameLv0     string `json:"deptNameLv0"`
	DeptNameLv1     string `json:"deptNameLv1"`
	DeptNameLv2     string `json:"deptNameLv2"`
	Status          string `json:"status"` // 在职状态
	CasdoorSynced   bool   `json:"casdoorSynced"`
	IsProtected     bool   `json:"isProtected"` // 受保护用户（不允许删除）
	IsAdmin         bool   `json:"isAdmin"`     // 是否管理员
	// Extra 为用户全部扩展属性（Casdoor User.Properties），
	// 键与用户字段定义 user_field_defs.field_key 对应，
	// 前端据此动态渲染自定义字段的列表列与表单项。
	Extra map[string]string `json:"extra"`
}
