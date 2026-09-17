package datasource

import (
	"context"
	"time"
)

// User 标准化用户结构
type User struct {
	DomainAccount   string // 域账号（唯一标识）
	EmployeeCode    string // 工号
	Name            string // 姓名
	Phone           string // 手机
	WorkEmail       string // 邮箱
	CompileType     string // 编制类型
	SuperiorAccount string // 上级域账号

	// 组织架构
	DeptIDLv0   string // 部门ID - 零级
	DeptIDLv1   string // 部门ID - 一级
	DeptIDLv2   string // 部门ID - 二级
	DeptNameLv0 string // 部门名 - 零级
	DeptNameLv1 string // 部门名 - 一级
	DeptNameLv2 string // 部门名 - 二级

	Source   string // 数据来源标识
	Priority int    // 优先级

	// Extra 自定义字段值（键为 user_field_defs.field_key），
	// 由数据源字段映射（DataSourceConfig.FieldMapping）驱动写入，
	// 最终作为 Casdoor User.Properties 存储。数据源字段变化只需改配置。
	Extra map[string]string
}

// Department 标准化部门结构
type Department struct {
	DeptID   string // 部门ID
	DeptName string // 部门名称
	ParentID string // 父部门ID
	Level    int    // 层级 (0/1/2)
	FullPath string // 完整路径
	Source   string // 数据来源
}

// DataSource 数据源接口
type DataSource interface {
	// Name 返回数据源名称（唯一标识）
	Name() string

	// IsEnabled 返回数据源是否启用
	IsEnabled() bool

	// Priority 返回数据源优先级（数字越大优先级越高）
	Priority() int

	// Sync 执行数据同步
	Sync(ctx context.Context) error

	// GetUsers 获取所有用户
	GetUsers(ctx context.Context) ([]User, error)

	// GetDepartments 获取所有部门
	GetDepartments(ctx context.Context) ([]Department, error)

	// GetSyncInterval 获取同步间隔
	GetSyncInterval() time.Duration

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error
}
