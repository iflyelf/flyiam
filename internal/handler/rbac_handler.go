package handler

import (
	"encoding/json"
	"net/http"

	"github.com/iflyelf/flyiam/internal/logic/rbac"
	"github.com/iflyelf/flyiam/internal/middleware"
	"github.com/iflyelf/flyiam/internal/model"
	"github.com/iflyelf/flyiam/internal/svc"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func rbacLogic(svcCtx *svc.ServiceContext) *rbac.Logic {
	return rbac.NewLogic(svcCtx.DB, svcCtx.Config.Permission.AdminUsers)
}

func pathInt64(r *http.Request, key string) (int64, bool) {
	v, err := atoiInt64(pathvar.Vars(r)[key])
	if err != nil {
		return 0, false
	}
	return v, true
}

// ============================ 角色 ============================

// ListRolesHandler 角色列表
func ListRolesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := rbacLogic(svcCtx).ListRoles(r.Context(), r.URL.Query().Get("keyword"))
		if err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok(w, map[string]interface{}{"list": list, "total": len(list), "allPermissions": model.AllPermissions})
	}
}

// CreateRoleHandler 创建角色
func CreateRoleHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name        string   `json:"name"`
			Code        string   `json:"code"`
			Description string   `json:"description"`
			Permissions []string `json:"permissions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败: "+err.Error())
			return
		}
		if req.Name == "" {
			fail(w, http.StatusBadRequest, "角色名称不能为空")
			return
		}
		id, err := rbacLogic(svcCtx).CreateRole(r.Context(), req.Name, req.Code, req.Description, req.Permissions)
		if err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok(w, map[string]int64{"id": id})
	}
}

// UpdateRoleHandler 更新角色
func UpdateRoleHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, valid := pathInt64(r, "id")
		if !valid {
			fail(w, http.StatusBadRequest, "无效的角色 ID")
			return
		}
		var req struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Permissions []string `json:"permissions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败: "+err.Error())
			return
		}
		if err := rbacLogic(svcCtx).UpdateRole(r.Context(), id, req.Name, req.Description, req.Permissions); err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok(w, nil)
	}
}

// DeleteRoleHandler 删除角色
func DeleteRoleHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, valid := pathInt64(r, "id")
		if !valid {
			fail(w, http.StatusBadRequest, "无效的角色 ID")
			return
		}
		if err := rbacLogic(svcCtx).DeleteRole(r.Context(), id); err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok(w, nil)
	}
}

// ============================ 团队 ============================

// ListTeamsHandler 团队列表
func ListTeamsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := rbacLogic(svcCtx).ListTeams(r.Context(), r.URL.Query().Get("keyword"))
		if err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok(w, map[string]interface{}{"list": list, "total": len(list)})
	}
}

// CreateTeamHandler 创建团队
func CreateTeamHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name        string `json:"name"`
			Code        string `json:"code"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败: "+err.Error())
			return
		}
		if req.Name == "" {
			fail(w, http.StatusBadRequest, "团队名称不能为空")
			return
		}
		createdBy := middleware.Username(r.Context())
		id, err := rbacLogic(svcCtx).CreateTeam(r.Context(), req.Name, req.Code, req.Description, createdBy)
		if err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok(w, map[string]int64{"id": id})
	}
}

// UpdateTeamHandler 更新团队
func UpdateTeamHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, valid := pathInt64(r, "id")
		if !valid {
			fail(w, http.StatusBadRequest, "无效的团队 ID")
			return
		}
		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Status      int    `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败: "+err.Error())
			return
		}
		if err := rbacLogic(svcCtx).UpdateTeam(r.Context(), id, req.Name, req.Description, req.Status); err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok(w, nil)
	}
}

// DeleteTeamHandler 删除团队
func DeleteTeamHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, valid := pathInt64(r, "id")
		if !valid {
			fail(w, http.StatusBadRequest, "无效的团队 ID")
			return
		}
		if err := rbacLogic(svcCtx).DeleteTeam(r.Context(), id); err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok(w, nil)
	}
}

// ============================ 团队成员 ============================

// ListTeamMembersHandler 团队成员列表
func ListTeamMembersHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, valid := pathInt64(r, "id")
		if !valid {
			fail(w, http.StatusBadRequest, "无效的团队 ID")
			return
		}
		list, err := rbacLogic(svcCtx).ListTeamMembers(r.Context(), id)
		if err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok(w, list)
	}
}

// AddTeamMemberHandler 添加团队成员
func AddTeamMemberHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, valid := pathInt64(r, "id")
		if !valid {
			fail(w, http.StatusBadRequest, "无效的团队 ID")
			return
		}
		var req struct {
			Username    string `json:"username"`
			DisplayName string `json:"displayName"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败: "+err.Error())
			return
		}
		if req.Username == "" {
			fail(w, http.StatusBadRequest, "用户名不能为空")
			return
		}
		if err := rbacLogic(svcCtx).AddTeamMember(r.Context(), id, req.Username, req.DisplayName); err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok(w, nil)
	}
}

// RemoveTeamMemberHandler 移除团队成员
func RemoveTeamMemberHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, valid := pathInt64(r, "id")
		if !valid {
			fail(w, http.StatusBadRequest, "无效的团队 ID")
			return
		}
		username := pathvar.Vars(r)["username"]
		if err := rbacLogic(svcCtx).RemoveTeamMember(r.Context(), id, username); err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok(w, nil)
	}
}

// ============================ 团队角色 ============================

// ListTeamRolesHandler 团队已授权角色
func ListTeamRolesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, valid := pathInt64(r, "id")
		if !valid {
			fail(w, http.StatusBadRequest, "无效的团队 ID")
			return
		}
		ids, err := rbacLogic(svcCtx).ListTeamRoles(r.Context(), id)
		if err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok(w, ids)
	}
}

// SetTeamRolesHandler 设置团队角色
func SetTeamRolesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, valid := pathInt64(r, "id")
		if !valid {
			fail(w, http.StatusBadRequest, "无效的团队 ID")
			return
		}
		var req struct {
			RoleIDs []int64 `json:"roleIds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败: "+err.Error())
			return
		}
		if err := rbacLogic(svcCtx).SetTeamRoles(r.Context(), id, req.RoleIDs); err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok(w, nil)
	}
}
