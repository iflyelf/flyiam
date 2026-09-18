package handler

import (
	"encoding/json"
	"net/http"

	"github.com/iflyelf/flyiam/internal/setting"
	"github.com/iflyelf/flyiam/internal/svc"
)

// ListSettingsHandler 返回全部可页面配置项（当前值；secret 以占位符返回）
func ListSettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok(w, svcCtx.Settings.View())
	}
}

// UpdateSettingsHandler 更新设置（写 DB 并即时应用到内存，无需重启）
//
// 入参：{ "settings": { "key": "value", ... } }
// 提交 secret 的占位符（******）表示保持原值。
func UpdateSettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Settings map[string]string `json:"settings"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败: "+err.Error())
			return
		}
		if len(req.Settings) == 0 {
			fail(w, http.StatusBadRequest, "没有需要更新的配置")
			return
		}

		// 去掉未修改的 secret 占位符
		toApply := make(map[string]string, len(req.Settings))
		for k, v := range req.Settings {
			if setting.IsMasked(v) {
				continue
			}
			toApply[k] = v
		}
		applied, err := svcCtx.Settings.Apply(r.Context(), toApply)
		if err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}

		// 若修改了 Casdoor 连接配置，热重载客户端（免重启）
		if setting.NeedsCasdoorReload(applied) {
			if err := svcCtx.ReloadCasdoor(r.Context()); err != nil {
				// 保存已成功，但重建失败：提示用户（旧客户端仍可用）
				fail(w, http.StatusBadGateway, "配置已保存，但 Casdoor 客户端重建失败: "+err.Error())
				return
			}
		}
		ok(w, nil)
	}
}
