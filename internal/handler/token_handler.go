package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/iflyelf/flyiam/internal/logic/apitoken"
	"github.com/iflyelf/flyiam/internal/middleware"
	"github.com/iflyelf/flyiam/internal/svc"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

// ListTokensHandler 列出当前用户的 API 令牌
func ListTokensHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := middleware.Username(r.Context())
		list, err := apitoken.NewLogic(svcCtx.DB).List(r.Context(), username)
		if err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok(w, list)
	}
}

// CreateTokenHandler 生成 API 令牌（返回明文，仅此一次）
//
// 入参：{ name: string, expiresInDays: int }（expiresInDays=0 表示永久）
func CreateTokenHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := middleware.Username(r.Context())
		var req struct {
			Name          string `json:"name"`
			ExpiresInDays int    `json:"expiresInDays"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败: "+err.Error())
			return
		}
		if req.ExpiresInDays < 0 {
			req.ExpiresInDays = 0
		}
		token, info, err := apitoken.NewLogic(svcCtx.DB).Create(r.Context(), username, req.Name, req.ExpiresInDays)
		if err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok(w, map[string]interface{}{
			"token": token, // 仅返回一次
			"info":  info,
		})
	}
}

// RevokeTokenHandler 吊销 API 令牌
func RevokeTokenHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := middleware.Username(r.Context())
		id, err := strconv.ParseInt(pathvar.Vars(r)["id"], 10, 64)
		if err != nil {
			fail(w, http.StatusBadRequest, "无效的令牌 ID")
			return
		}
		if err := apitoken.NewLogic(svcCtx.DB).Revoke(r.Context(), id, username); err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok(w, nil)
	}
}
