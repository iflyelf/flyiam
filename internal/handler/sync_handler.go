package handler

import (
	"context"
	"log"
	"net/http"

	dslogic "github.com/iflyelf/flyiam/internal/logic/datasource"
	"github.com/iflyelf/flyiam/internal/logic/schedule"
	synclogic "github.com/iflyelf/flyiam/internal/logic/sync"
	"github.com/iflyelf/flyiam/internal/svc"
)

// syncOptions 解析同步选项。
//
// 并发度优先取 URL 显式参数 ?batchSize（API 声明的入参），未提供时回退到
// 「定时任务」页面配置的 casdoor_batch_size。两者都无效时用默认 10。
func syncOptions(svcCtx *svc.ServiceContext, r *http.Request) synclogic.SyncOptions {
	opts := synclogic.SyncOptions{DeleteMissing: true, Concurrency: 10}
	if cfg, err := schedule.NewLogic(svcCtx.DB).Get(r.Context()); err == nil {
		opts.DeleteMissing = cfg.DeleteMissing
		if cfg.CasdoorBatchSize > 0 {
			opts.Concurrency = cfg.CasdoorBatchSize
		}
	}
	// URL 显式指定时以其为准（如 ?batchSize=50）
	if bs := atoiDefault(r.URL.Query().Get("batchSize"), 0); bs > 0 {
		opts.Concurrency = bs
	}
	return opts
}

// SyncDataSourceHandler 从数据源同步到 Casdoor
func SyncDataSourceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 并发拒绝返回 200 + 业务错误码，避免浏览器控制台出现红色网络错误
		if synclogic.IsRunning() {
			writeJSON(w, http.StatusOK, respBody{Code: 409, Message: "已有同步任务正在执行，请稍后再试"})
			return
		}
		opts := syncOptions(svcCtx, r)
		logic := synclogic.NewSyncLogic(svcCtx.DB, svcCtx.Casdoor(), dslogic.NewLogic(svcCtx.DB))
		go func() {
			if err := logic.SyncFromDataSources(context.Background(), opts, "manual"); err != nil {
				log.Printf("⚠️ 数据源同步失败: %v", err)
			}
		}()
		ok(w, map[string]string{"message": "数据源同步任务已启动"})
	}
}

// SyncCasdoorHandler 兼容端点：等同数据源同步
func SyncCasdoorHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return SyncDataSourceHandler(svcCtx)
}

// SyncFullHandler 完整同步
func SyncFullHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return SyncDataSourceHandler(svcCtx)
}

// GetSyncLogsHandler 同步日志
func GetSyncLogsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := atoiDefault(r.URL.Query().Get("page"), 1)
		if page < 1 {
			page = 1
		}
		pageSize := atoiDefault(r.URL.Query().Get("pageSize"), 20)
		if pageSize < 1 || pageSize > 200 {
			pageSize = 20
		}
		syncType := r.URL.Query().Get("syncType")
		status := r.URL.Query().Get("status")

		logic := synclogic.NewSyncLogic(svcCtx.DB, svcCtx.Casdoor(), dslogic.NewLogic(svcCtx.DB))
		logs, total, err := logic.ListSyncLogs(r.Context(), syncType, status, (page-1)*pageSize, pageSize)
		if err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		okPage(w, logs, total, page, pageSize)
	}
}

// GetSyncProgressHandler 同步进度
func GetSyncProgressHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logic := synclogic.NewSyncLogic(svcCtx.DB, svcCtx.Casdoor(), dslogic.NewLogic(svcCtx.DB))
		progress, err := logic.LatestProgress(r.Context())
		if err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok(w, progress)
	}
}
