package sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	casdoorsdk "github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	dslogic "github.com/iflyelf/flyiam/internal/logic/datasource"
	"github.com/iflyelf/flyiam/internal/model"
	"github.com/iflyelf/flyiam/internal/pkg/casdoor"
	"github.com/iflyelf/flyiam/internal/pkg/datasource"
	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// SyncOptions 同步选项
type SyncOptions struct {
	// DeleteMissing 删除数据源中不存在的人员（受保护用户除外）
	DeleteMissing bool
	// Concurrency 并发批量大小
	Concurrency int
}

// runningGuard 全局同步互斥：同一时刻只允许一个同步任务执行，避免并发冲突
var runningGuard int32

// ErrSyncRunning 已有同步任务在执行
var ErrSyncRunning = fmt.Errorf("已有同步任务正在执行，请稍后再试")

// tryAcquire 尝试获取同步锁
func tryAcquire() bool {
	return atomic.CompareAndSwapInt32(&runningGuard, 0, 1)
}

// release 释放同步锁
func release() {
	atomic.StoreInt32(&runningGuard, 0)
}

// SyncLogic 同步业务逻辑
//
// 本系统不存储用户数据，用户唯一存储于 Casdoor：
//   - 数据源（人事 API）为权威来源
//   - 同步即直接对 Casdoor 做新增/更新/删除
type SyncLogic struct {
	db              sqlx.SqlConn
	casdoorClient   *casdoor.Client
	datasourceLogic *dslogic.Logic
}

// NewSyncLogic 创建同步业务逻辑
func NewSyncLogic(db sqlx.SqlConn, casdoorClient *casdoor.Client, datasourceLogic *dslogic.Logic) *SyncLogic {
	return &SyncLogic{
		db:              db,
		casdoorClient:   casdoorClient,
		datasourceLogic: datasourceLogic,
	}
}

// SyncFromDataSources 从数据源同步用户到 Casdoor
func (l *SyncLogic) SyncFromDataSources(ctx context.Context, opts SyncOptions, triggeredBy string) (err error) {
	// 并发保护：同一时刻仅允许一个同步任务
	if !tryAcquire() {
		return ErrSyncRunning
	}
	defer release()

	// panic 兜底：同步在后台 goroutine 中执行，任何 panic 默认会终止整个进程。
	// 此处捕获并记录堆栈，保证「同步失败」不会拖垮服务。
	defer func() {
		if r := recover(); r != nil {
			log.Printf("💥 同步任务 panic（已恢复，进程不受影响）: %v\n%s", r, debug.Stack())
			err = fmt.Errorf("同步任务异常终止: %v", r)
		}
	}()

	startTime := time.Now()
	log.Println("🔄 开始从数据源同步用户到 Casdoor...")

	if l.casdoorClient == nil {
		err := fmt.Errorf("Casdoor 客户端未初始化")
		l.finishSyncLog(ctx, 0, "failed", 0, 0, 0, 0, err.Error(), time.Since(startTime), nil)
		return err
	}
	if opts.Concurrency <= 0 {
		opts.Concurrency = 10
	}

	syncLogID, _ := l.createSyncLog(ctx, "datasource", triggeredBy)

	sources, err := l.datasourceLogic.BuildSources(ctx)
	if err != nil {
		l.finishSyncLog(ctx, syncLogID, "failed", 0, 0, 0, 0, err.Error(), time.Since(startTime), nil)
		return err
	}
	if len(sources) == 0 {
		msg := "没有启用的数据源"
		l.finishSyncLog(ctx, syncLogID, "failed", 0, 0, 0, 0, msg, time.Since(startTime), nil)
		return errors.New(msg)
	}

	// 1. 拉取数据源用户并按域账号去重
	seen := make(map[string]struct{})
	syncUsers := make([]casdoor.SyncUser, 0)
	sourceTotal := 0
	for _, source := range sources {
		users, err := source.GetUsers(ctx)
		if err != nil {
			log.Printf("❌ 从数据源 %s 获取用户失败: %v", source.Name(), err)
			continue
		}
		sourceTotal += len(users)
		for _, u := range users {
			if u.DomainAccount == "" {
				continue
			}
			if _, ok := seen[u.DomainAccount]; ok {
				continue
			}
			seen[u.DomainAccount] = struct{}{}
			syncUsers = append(syncUsers, toSyncUser(u))
		}
	}
	log.Printf("📊 数据源共 %d 条，去重后 %d 个用户", sourceTotal, len(syncUsers))

	// 运行中即写入总数，便于页面展示进度
	l.updateSyncProgress(ctx, syncLogID, len(syncUsers), 0)

	// 2. 分页读取 Casdoor 现有用户（避免全量加载导致内存暴涨/OOM）。
	//
	// 说明：只保留「数据源中仍存在」的用户用于增量更新；数据源中已不存在的
	// 用户直接进入待删除列表，不再整体驻留内存。
	existingMap := make(map[string]*casdoorsdk.User)
	toDelete := make([]string, 0)
	err = l.casdoorClient.IterateUsersByOrg(0, func(batch []*casdoorsdk.User) error {
		for _, u := range batch {
			if _, ok := seen[u.Name]; ok {
				existingMap[u.Name] = u
				continue
			}
			if opts.DeleteMissing && !l.casdoorClient.IsProtected(u.Name) {
				toDelete = append(toDelete, u.Name)
			}
		}
		return nil
	})
	if err != nil {
		l.finishSyncLog(ctx, syncLogID, "failed", sourceTotal, 0, 0, 0, "读取 Casdoor 用户失败: "+err.Error(), time.Since(startTime), nil)
		return fmt.Errorf("读取 Casdoor 用户失败: %w", err)
	}

	// 3. 并发新增/更新到 Casdoor（周期性上报进度）
	progress := func(done, total int) {
		l.updateSyncProgress(ctx, syncLogID, total, done)
	}
	result := l.casdoorClient.BatchSync(ctx, syncUsers, existingMap, opts.Concurrency, progress)

	// 4. 删除数据源中不存在的人员（跳过受保护用户）
	deleted := 0
	var deletedNames []string
	if opts.DeleteMissing && len(toDelete) > 0 {
		deleted = l.deleteUsers(ctx, toDelete, opts.Concurrency)
		deletedNames = toDelete
		log.Printf("🗑️  已从 Casdoor 删除 %d 名不存在于数据源的用户", deleted)
	}

	duration := time.Since(startTime)
	log.Printf("✅ 同步完成: 成功 %d/%d（新增 %d，更新 %d），失败 %d，删除 %d，耗时 %v",
		result.Success, result.Total, result.Created, result.Updated, result.Failed, deleted, duration)

	l.finishSyncLog(ctx, syncLogID, "success", result.Total, result.Success, result.Failed, deleted, "", duration, map[string]interface{}{
		"created":      result.Created,
		"updated":      result.Updated,
		"deletedUsers": deletedNames,
		"errorSamples": result.Errors,
	})
	return nil
}

// toSyncUser 数据源用户转 Casdoor 同步结构（人事字段 + 自定义字段写入属性）
func toSyncUser(u datasource.User) casdoor.SyncUser {
	props := map[string]string{
		"empCode":     u.EmployeeCode,
		"compileType": u.CompileType,
		"deptNameLv0": u.DeptNameLv0,
		"deptNameLv1": u.DeptNameLv1,
		"deptNameLv2": u.DeptNameLv2,
		"deptIdLv0":   u.DeptIDLv0,
		"deptIdLv1":   u.DeptIDLv1,
		"deptIdLv2":   u.DeptIDLv2,
		"superior":    u.SuperiorAccount,
		"source":      u.Source,
	}
	// 合并数据源字段映射得到的自定义字段（数据源变化只需改配置）
	for k, v := range u.Extra {
		if k == "" {
			continue
		}
		props[k] = v
	}
	return casdoor.SyncUser{
		DomainAccount: u.DomainAccount,
		Name:          u.Name,
		Email:         u.WorkEmail,
		Phone:         u.Phone,
		Affiliation:   u.DeptNameLv1,
		Properties:    props,
	}
}

// deleteUsers 并发删除用户（跳过受保护用户）
//
// 删除成功后级联清理本地关联（team_members），避免悬挂授权。
func (l *SyncLogic) deleteUsers(ctx context.Context, names []string, concurrency int) int {
	if concurrency <= 0 {
		concurrency = 10
	}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	deleted := 0
	deletedNames := make([]string, 0, len(names))
	for _, name := range names {
		if l.casdoorClient.IsProtected(name) {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(n string) {
			defer wg.Done()
			defer func() { <-sem }()

			// panic 兜底：worker 在独立 goroutine 中，panic 会终止整个进程
			var derr error
			func() {
				defer func() {
					if r := recover(); r != nil {
						derr = fmt.Errorf("panic: %v\n%s", r, debug.Stack())
					}
				}()
				_, derr = l.casdoorClient.DeleteUser(n)
			}()

			if derr == nil {
				mu.Lock()
				deleted++
				deletedNames = append(deletedNames, n)
				mu.Unlock()
			} else {
				log.Printf("⚠️ 删除用户 %s 失败: %v", n, derr)
			}
		}(name)
	}
	wg.Wait()

	// 级联清理本地关联
	if len(deletedNames) > 0 {
		if _, err := l.db.ExecCtx(ctx,
			`DELETE FROM team_members WHERE username = ANY($1)`, pq.Array(deletedNames)); err != nil {
			log.Printf("⚠️ 清理团队成员关联失败: %v", err)
		}
	}
	return deleted
}

// syncLogColumns 显式列出列并对可空列做 COALESCE，避免 NULL 扫描错误
const syncLogColumns = `id, sync_type, COALESCE(data_source,'') AS data_source, status,
	total_count, success_count, failed_count, new_count, updated_count, deleted_count,
	COALESCE(error_message,'') AS error_message, COALESCE(details::text,'') AS details,
	COALESCE(duration_ms,0) AS duration_ms, COALESCE(triggered_by,'') AS triggered_by,
	started_at, completed_at`

// ListSyncLogs 分页查询同步日志
func (l *SyncLogic) ListSyncLogs(ctx context.Context, syncType, status string, offset, limit int) ([]*model.SyncLog, int64, error) {
	where := "WHERE 1=1"
	args := make([]interface{}, 0, 4)
	idx := 1
	if syncType != "" {
		where += fmt.Sprintf(" AND sync_type = $%d", idx)
		args = append(args, syncType)
		idx++
	}
	if status != "" {
		where += fmt.Sprintf(" AND status = $%d", idx)
		args = append(args, status)
		idx++
	}
	var total int64
	if err := l.db.QueryRowCtx(ctx, &total, "SELECT COUNT(*) FROM sync_logs "+where, args...); err != nil {
		return nil, 0, fmt.Errorf("查询同步日志总数失败: %w", err)
	}
	query := "SELECT " + syncLogColumns + " FROM sync_logs " + where + fmt.Sprintf(" ORDER BY started_at DESC LIMIT $%d OFFSET $%d", idx, idx+1)
	args = append(args, limit, offset)
	var logs []*model.SyncLog
	if err := l.db.QueryRowsCtx(ctx, &logs, query, args...); err != nil {
		return nil, 0, fmt.Errorf("查询同步日志失败: %w", err)
	}
	return logs, total, nil
}

// LatestProgress 各同步类型的最新一次执行状态
func (l *SyncLogic) LatestProgress(ctx context.Context) (map[string]*model.SyncLog, error) {
	types := []string{"datasource", "full"}
	result := make(map[string]*model.SyncLog, len(types))
	for _, t := range types {
		var item model.SyncLog
		query := `SELECT ` + syncLogColumns + ` FROM sync_logs WHERE sync_type = $1 ORDER BY started_at DESC LIMIT 1`
		if err := l.db.QueryRowCtx(ctx, &item, query, t); err == nil {
			result[t] = &item
		}
	}
	return result, nil
}

// updateSyncProgress 运行中更新同步进度（总数/已完成数）
func (l *SyncLogic) updateSyncProgress(ctx context.Context, id int64, total, success int) {
	if id == 0 {
		return
	}
	query := `UPDATE sync_logs SET total_count=$1, success_count=$2 WHERE id=$3`
	if _, err := l.db.ExecCtx(ctx, query, total, success, id); err != nil {
		log.Printf("⚠️ 更新同步进度失败: %v", err)
	}
}

// IsRunning 当前是否有同步任务在执行
func IsRunning() bool {
	return atomic.LoadInt32(&runningGuard) == 1
}

// createSyncLog 创建同步日志
func (l *SyncLogic) createSyncLog(ctx context.Context, syncType, triggeredBy string) (int64, error) {
	query := `INSERT INTO sync_logs (sync_type, status, triggered_by, started_at) VALUES ($1, 'running', $2, NOW()) RETURNING id`
	var id int64
	if err := l.db.QueryRowCtx(ctx, &id, query, syncType, triggeredBy); err != nil {
		log.Printf("⚠️ 创建同步日志失败: %v", err)
		return 0, err
	}
	return id, nil
}

// finishSyncLog 完成同步日志
func (l *SyncLogic) finishSyncLog(ctx context.Context, id int64, status string, total, success, failed, deleted int, errMsg string, duration time.Duration, details map[string]interface{}) {
	if id == 0 {
		return
	}
	var detailsVal interface{}
	if details != nil {
		if b, err := json.Marshal(details); err == nil {
			detailsVal = string(b)
		}
	}
	query := `UPDATE sync_logs SET status=$1, total_count=$2, success_count=$3, failed_count=$4,
		deleted_count=$5, error_message=$6, duration_ms=$7, details=$8, completed_at=NOW() WHERE id=$9`
	if _, err := l.db.ExecCtx(ctx, query, status, total, success, failed, deleted, errMsg, duration.Milliseconds(), detailsVal, id); err != nil {
		log.Printf("⚠️ 更新同步日志失败: %v", err)
	}
}
