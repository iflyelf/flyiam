package handler

import (
	"encoding/json"
	"net/http"

	"github.com/iflyelf/flyiam/internal/logic/userfield"
	"github.com/iflyelf/flyiam/internal/model"
	"github.com/iflyelf/flyiam/internal/svc"
)

// ListUserFieldsHandler 字段定义列表
func ListUserFieldsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logic := userfield.NewLogic(svcCtx.DB)
		list, err := logic.List(r.Context())
		if err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok(w, list)
	}
}

// CreateUserFieldHandler 新增字段定义
func CreateUserFieldHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var d model.UserFieldDef
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败: "+err.Error())
			return
		}
		logic := userfield.NewLogic(svcCtx.DB)
		id, err := logic.Create(r.Context(), &d)
		if err != nil {
			fail(w, http.StatusBadRequest, err.Error())
			return
		}
		d.ID = id
		ok(w, d)
	}
}

// UpdateUserFieldHandler 更新字段定义
func UpdateUserFieldHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := pathID(r)
		if err != nil {
			fail(w, http.StatusBadRequest, "无效的字段 ID")
			return
		}
		var d model.UserFieldDef
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败: "+err.Error())
			return
		}
		d.ID = id
		logic := userfield.NewLogic(svcCtx.DB)
		if err := logic.Update(r.Context(), &d); err != nil {
			fail(w, http.StatusBadRequest, err.Error())
			return
		}
		ok(w, d)
	}
}

// DeleteUserFieldHandler 删除字段定义
func DeleteUserFieldHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := pathID(r)
		if err != nil {
			fail(w, http.StatusBadRequest, "无效的字段 ID")
			return
		}
		logic := userfield.NewLogic(svcCtx.DB)
		if err := logic.Delete(r.Context(), id); err != nil {
			fail(w, http.StatusBadRequest, err.Error())
			return
		}
		ok(w, nil)
	}
}
