package handler

import (
	"encoding/json"
	"net/http"

	"github.com/iflyelf/flyiam/internal/logic/schedule"
	"github.com/iflyelf/flyiam/internal/model"
	"github.com/iflyelf/flyiam/internal/svc"
)

// GetScheduleHandler 获取定时任务配置
func GetScheduleHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg, err := schedule.NewLogic(svcCtx.DB).Get(r.Context())
		if err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok(w, cfg)
	}
}

// UpdateScheduleHandler 更新定时任务配置
func UpdateScheduleHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var cfg model.ScheduleConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败: "+err.Error())
			return
		}
		if cfg.Interval == "" {
			cfg.Interval = "6h"
		}
		if cfg.CasdoorBatchSize <= 0 {
			cfg.CasdoorBatchSize = 10
		}
		logic := schedule.NewLogic(svcCtx.DB)
		if err := logic.Update(r.Context(), &cfg); err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		updated, err := logic.Get(r.Context())
		if err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok(w, updated)
	}
}
