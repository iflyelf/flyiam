package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/iflyelf/flyiam/internal/logic/datasource"
	"github.com/iflyelf/flyiam/internal/model"
	"github.com/iflyelf/flyiam/internal/svc"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

// maskedSecret 敏感字段回显占位符（提交该值表示「保持原值不变」）
const maskedSecret = "******"

// maskDataSource 脱敏数据源敏感字段（认证 Token / 密码）后再返回。
//
// 这些字段仅需写入，不应经接口回显；编辑时提交占位符即保持原值。
func maskDataSource(cfg *model.DataSourceConfig) *model.DataSourceConfig {
	if cfg == nil {
		return nil
	}
	masked := *cfg
	if masked.AuthToken != "" {
		masked.AuthToken = maskedSecret
	}
	if masked.AuthPassword != "" {
		masked.AuthPassword = maskedSecret
	}
	return &masked
}

// ListDataSourcesHandler 数据源列表
func ListDataSourcesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logic := datasource.NewLogic(svcCtx.DB)
		list, err := logic.List(r.Context())
		if err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		out := make([]*model.DataSourceConfig, 0, len(list))
		for _, cfg := range list {
			out = append(out, maskDataSource(cfg))
		}
		ok(w, out)
	}
}

// CreateDataSourceHandler 创建数据源
func CreateDataSourceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg, err := decodeDataSource(r)
		if err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败: "+err.Error())
			return
		}
		normalizeDataSource(cfg)

		logic := datasource.NewLogic(svcCtx.DB)
		id, err := logic.Create(r.Context(), cfg)
		if err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		cfg.ID = id
		ok(w, maskDataSource(cfg))
	}
}

// UpdateDataSourceHandler 更新数据源
func UpdateDataSourceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := pathID(r)
		if err != nil {
			fail(w, http.StatusBadRequest, "无效的数据源 ID")
			return
		}
		cfg, err := decodeDataSource(r)
		if err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败: "+err.Error())
			return
		}
		cfg.ID = id
		normalizeDataSource(cfg)

		logic := datasource.NewLogic(svcCtx.DB)
		// 提交的敏感字段为掩码占位符时，保留原值（前端编辑未修改密钥的场景）
		if cfg.AuthToken == maskedSecret || cfg.AuthPassword == maskedSecret {
			if old, gerr := logic.Get(r.Context(), id); gerr == nil && old != nil {
				if cfg.AuthToken == maskedSecret {
					cfg.AuthToken = old.AuthToken
				}
				if cfg.AuthPassword == maskedSecret {
					cfg.AuthPassword = old.AuthPassword
				}
			}
		}
		if err := logic.Update(r.Context(), cfg); err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok(w, maskDataSource(cfg))
	}
}

// DeleteDataSourceHandler 删除数据源
func DeleteDataSourceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := pathID(r)
		if err != nil {
			fail(w, http.StatusBadRequest, "无效的数据源 ID")
			return
		}
		logic := datasource.NewLogic(svcCtx.DB)
		if err := logic.Delete(r.Context(), id); err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok(w, nil)
	}
}

// TestDataSourceHandler 测试数据源连接
func TestDataSourceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := pathID(r)
		if err != nil {
			fail(w, http.StatusBadRequest, "无效的数据源 ID")
			return
		}
		logic := datasource.NewLogic(svcCtx.DB)
		cfg, err := logic.Get(r.Context(), id)
		if err != nil {
			fail(w, http.StatusNotFound, err.Error())
			return
		}
		testDataSource(w, r.Context(), cfg)
	}
}

// TestDataSourceConfigHandler 测试未保存的数据源配置
func TestDataSourceConfigHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg, err := decodeDataSource(r)
		if err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败: "+err.Error())
			return
		}
		normalizeDataSource(cfg)
		testDataSource(w, r.Context(), cfg)
	}
}

// testDataSource 执行连接测试并返回样本数据
func testDataSource(w http.ResponseWriter, ctx context.Context, cfg *model.DataSourceConfig) {
	sources, err := datasource.BuildSourcesFromConfigs(ctx, []*model.DataSourceConfig{cfg})
	if err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	users, err := sources[0].GetUsers(ctx)
	if err != nil {
		fail(w, http.StatusBadGateway, "连接测试失败: "+err.Error())
		return
	}
	sample := users
	if len(sample) > 5 {
		sample = sample[:5]
	}
	ok(w, map[string]interface{}{
		"connected": true,
		"total":     len(users),
		"sample":    sample,
	})
}

// decodeDataSource 解析请求体
func decodeDataSource(r *http.Request) (*model.DataSourceConfig, error) {
	var cfg model.DataSourceConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// normalizeDataSource 规范化默认值
func normalizeDataSource(cfg *model.DataSourceConfig) {
	if cfg.Type == "" {
		cfg.Type = "httpapi"
	}
	if cfg.Method == "" {
		cfg.Method = http.MethodPost
	}
	if cfg.AuthType == "" {
		cfg.AuthType = "none"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30
	}
	if cfg.PageSize <= 0 {
		cfg.PageSize = 1000
	}
	if cfg.SyncInterval == "" {
		cfg.SyncInterval = "6h"
	}
	if cfg.Priority == 0 {
		cfg.Priority = 90
	}
}

// pathID 获取路径参数 id
func pathID(r *http.Request) (int64, error) {
	vars := pathvar.Vars(r)
	return strconv.ParseInt(vars["id"], 10, 64)
}
