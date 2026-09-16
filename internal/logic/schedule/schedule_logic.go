package schedule

import (
	"context"
	"fmt"
	"time"

	"github.com/iflyelf/flyiam/internal/model"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Logic 定时任务配置业务逻辑
type Logic struct {
	db sqlx.SqlConn
}

// NewLogic 创建定时任务配置业务逻辑
func NewLogic(db sqlx.SqlConn) *Logic {
	return &Logic{db: db}
}

// Get 获取定时任务配置
func (l *Logic) Get(ctx context.Context) (*model.ScheduleConfig, error) {
	var cfg model.ScheduleConfig
	query := `SELECT id, enabled, run_interval, sync_datasource, delete_missing,
		casdoor_batch_size, last_run_at,
		COALESCE(last_run_status,'') AS last_run_status,
		COALESCE(last_run_message,'') AS last_run_message, updated_at
		FROM schedule_config WHERE id = 1`
	if err := l.db.QueryRowCtx(ctx, &cfg, query); err != nil {
		return nil, fmt.Errorf("查询定时任务配置失败: %w", err)
	}
	return &cfg, nil
}

// Update 更新定时任务配置
func (l *Logic) Update(ctx context.Context, cfg *model.ScheduleConfig) error {
	query := `UPDATE schedule_config SET
		enabled=$1, run_interval=$2, sync_datasource=$3, delete_missing=$4,
		casdoor_batch_size=$5, updated_at=NOW() WHERE id=1`
	_, err := l.db.ExecCtx(ctx, query,
		cfg.Enabled, cfg.Interval, cfg.SyncDataSource, cfg.DeleteMissing, cfg.CasdoorBatchSize,
	)
	if err != nil {
		return fmt.Errorf("更新定时任务配置失败: %w", err)
	}
	return nil
}

// UpdateLastRun 记录最近一次执行结果
func (l *Logic) UpdateLastRun(ctx context.Context, status, message string) error {
	query := `UPDATE schedule_config SET last_run_at=NOW(), last_run_status=$1, last_run_message=$2, updated_at=NOW() WHERE id=1`
	if _, err := l.db.ExecCtx(ctx, query, status, message); err != nil {
		return fmt.Errorf("更新执行记录失败: %w", err)
	}
	return nil
}

// ParseInterval 解析执行间隔，非法时返回默认 6 小时
func ParseInterval(c *model.ScheduleConfig) time.Duration {
	d, err := time.ParseDuration(c.Interval)
	if err != nil || d <= 0 {
		return 6 * time.Hour
	}
	return d
}
