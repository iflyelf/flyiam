package model

import "time"

// ScheduleConfig 定时任务配置（单例，存储于数据库）
type ScheduleConfig struct {
	ID               int64     `db:"id" json:"id"`
	Enabled          bool      `db:"enabled" json:"enabled"`                     // 定时任务总开关
	Interval         string    `db:"run_interval" json:"interval"`               // 执行间隔，如 6h/30m
	SyncDataSource   bool      `db:"sync_datasource" json:"syncDataSource"`      // 是否从数据源同步
	DeleteMissing    bool      `db:"delete_missing" json:"deleteMissing"`        // 删除数据源中不存在的人员（保护用户除外）
	CasdoorBatchSize int       `db:"casdoor_batch_size" json:"casdoorBatchSize"` // 并发批量大小
	LastRunAt        NullTime  `db:"last_run_at" json:"lastRunAt"`               // 上次执行时间
	LastRunStatus    string    `db:"last_run_status" json:"lastRunStatus"`       // 上次执行结果
	LastRunMessage   string    `db:"last_run_message" json:"lastRunMessage"`     // 上次执行信息
	UpdatedAt        time.Time `db:"updated_at" json:"updatedAt"`
}
