package scheduler

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	dslogic "github.com/iflyelf/flyiam/internal/logic/datasource"
	"github.com/iflyelf/flyiam/internal/logic/schedule"
	synclogic "github.com/iflyelf/flyiam/internal/logic/sync"
	"github.com/iflyelf/flyiam/internal/model"
	"github.com/iflyelf/flyiam/internal/pkg/casdoor"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Scheduler 定时任务调度器
type Scheduler struct {
	db sqlx.SqlConn
	// casdoorFn 运行时获取当前 Casdoor 客户端（支持热重载后自动用新客户端）
	casdoorFn func() *casdoor.Client
	baseTick  time.Duration
}

// New 创建调度器。casdoorFn 在每次执行同步时调用，返回当前生效的客户端。
func New(db sqlx.SqlConn, casdoorFn func() *casdoor.Client) *Scheduler {
	return &Scheduler{
		db:        db,
		casdoorFn: casdoorFn,
		baseTick:  time.Minute,
	}
}

// Start 启动后台调度循环
func (s *Scheduler) Start(ctx context.Context) {
	log.Println("⏰ 定时任务调度器已启动（每分钟检查一次配置）")
	go s.loop(ctx)
}

// loop 调度主循环
func (s *Scheduler) loop(ctx context.Context) {
	ticker := time.NewTicker(s.baseTick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("⏰ 定时任务调度器已停止")
			return
		case <-ticker.C:
			// panic 兜底：单次调度异常不应终止调度循环/整个进程。
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("💥 定时任务调度 panic（已恢复）: %v\n%s", r, debug.Stack())
					}
				}()
				s.tick(ctx)
			}()
		}
	}
}

// tick 单次检查与执行
func (s *Scheduler) tick(ctx context.Context) {
	scheduleLogic := schedule.NewLogic(s.db)
	cfg, err := scheduleLogic.Get(ctx)
	if err != nil {
		log.Printf("⚠️ 读取定时任务配置失败: %v", err)
		return
	}
	if !cfg.Enabled {
		return
	}

	interval := schedule.ParseInterval(cfg)
	if cfg.LastRunAt.Valid && time.Since(cfg.LastRunAt.Time) < interval {
		return
	}

	log.Printf("⏰ 定时任务触发（间隔 %s）", interval)
	s.run(ctx, scheduleLogic, cfg)
}

// run 执行定时同步
func (s *Scheduler) run(ctx context.Context, scheduleLogic *schedule.Logic, cfg *model.ScheduleConfig) {
	opts := synclogic.SyncOptions{
		DeleteMissing: cfg.DeleteMissing,
		Concurrency:   cfg.CasdoorBatchSize,
	}

	// 运行时获取当前 Casdoor 客户端（热重载后自动使用新客户端）
	var cd *casdoor.Client
	if s.casdoorFn != nil {
		cd = s.casdoorFn()
	}
	syncer := synclogic.NewSyncLogic(s.db, cd, dslogic.NewLogic(s.db))

	var runErr error
	if cfg.SyncDataSource {
		runErr = syncer.SyncFromDataSources(ctx, opts, "auto")
	}

	if runErr != nil {
		log.Printf("❌ 定时任务执行失败: %v", runErr)
		_ = scheduleLogic.UpdateLastRun(ctx, "failed", runErr.Error())
		return
	}
	log.Println("✅ 定时任务执行成功")
	_ = scheduleLogic.UpdateLastRun(ctx, "success", fmt.Sprintf("执行完成于 %s", time.Now().Format("2006-01-02 15:04:05")))
}
