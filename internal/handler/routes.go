package handler

import (
	"net/http"

	"github.com/iflyelf/flyiam/internal/middleware"
	"github.com/iflyelf/flyiam/internal/model"
	"github.com/iflyelf/flyiam/internal/svc"
	"github.com/zeromicro/go-zero/rest"
)

// RegisterHandlers 注册所有路由
//
// 权限模型：
//   - 公开：健康检查、登录、回调、登录配置
//   - 需登录：用户信息、修改本人密码
//   - 需权限：其余管理接口，按 <资源>:<动作> 校验
//     超级管理员（配置的管理员名单）拥有全部权限，其余用户权限来自其所在团队关联的角色
func RegisterHandlers(server *rest.Server, svcCtx *svc.ServiceContext) {
	auth := middleware.Auth(svcCtx)
	perm := func(p string) func(http.HandlerFunc) http.HandlerFunc {
		return func(next http.HandlerFunc) http.HandlerFunc {
			return auth(middleware.RequirePermission(svcCtx, p)(next))
		}
	}
	guard := func(next http.HandlerFunc) http.HandlerFunc { return auth(next) }

	server.AddRoutes([]rest.Route{
		// 健康检查
		{Method: http.MethodGet, Path: "/health", Handler: HealthHandler(svcCtx)},

		// 认证（Casdoor OAuth2，公开）
		{Method: http.MethodGet, Path: "/api/auth/login", Handler: LoginHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/auth/callback", Handler: CallbackHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/auth/config", Handler: AuthConfigHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/api/auth/logout", Handler: LogoutHandler(svcCtx)},

		// 当前用户（需登录）
		{Method: http.MethodGet, Path: "/api/auth/userinfo", Handler: guard(UserInfoHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/auth/change-password", Handler: guard(ChangePasswordHandler(svcCtx))},

		// 用户管理（数据来源：Casdoor）
		{Method: http.MethodGet, Path: "/api/users", Handler: perm("user:read")(ListUsersHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/user/detail", Handler: perm("user:read")(GetUserDetailHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/user/stats", Handler: perm("user:read")(GetUserStatsHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/users", Handler: perm("user:write")(CreateUserHandler(svcCtx))},
		{Method: http.MethodPut, Path: "/api/users/:name", Handler: perm("user:write")(UpdateUserHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/users/:name/reset-password", Handler: perm("user:write")(ResetPasswordHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/users/:name/admin", Handler: perm("user:write")(SetUserAdminHandler(svcCtx))},
		{Method: http.MethodDelete, Path: "/api/users/:name", Handler: perm("user:delete")(DeleteUserHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/users/batch-delete", Handler: perm("user:delete")(BatchDeleteUsersHandler(svcCtx))},

		// 用户字段定义（页面可配置，决定用户列表/表单展示哪些字段）
		{Method: http.MethodGet, Path: "/api/user-fields", Handler: perm("userfield:read")(ListUserFieldsHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/user-fields", Handler: perm("userfield:write")(CreateUserFieldHandler(svcCtx))},
		{Method: http.MethodPut, Path: "/api/user-fields/:id", Handler: perm("userfield:write")(UpdateUserFieldHandler(svcCtx))},
		{Method: http.MethodDelete, Path: "/api/user-fields/:id", Handler: perm("userfield:write")(DeleteUserFieldHandler(svcCtx))},

		// 受保护用户（页面可配置）
		{Method: http.MethodGet, Path: "/api/protected-users", Handler: perm("user:read")(ListProtectedUsersHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/protected-users", Handler: perm("user:write")(AddProtectedUserHandler(svcCtx))},
		{Method: http.MethodDelete, Path: "/api/protected-users/:account", Handler: perm("user:write")(DeleteProtectedUserHandler(svcCtx))},

		// 角色管理
		{Method: http.MethodGet, Path: "/api/roles", Handler: perm("role:read")(ListRolesHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/roles", Handler: perm("role:write")(CreateRoleHandler(svcCtx))},
		{Method: http.MethodPut, Path: "/api/roles/:id", Handler: perm("role:write")(UpdateRoleHandler(svcCtx))},
		{Method: http.MethodDelete, Path: "/api/roles/:id", Handler: perm("role:delete")(DeleteRoleHandler(svcCtx))},

		// 团队管理
		{Method: http.MethodGet, Path: "/api/teams", Handler: perm("team:read")(ListTeamsHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/teams", Handler: perm("team:write")(CreateTeamHandler(svcCtx))},
		{Method: http.MethodPut, Path: "/api/teams/:id", Handler: perm("team:write")(UpdateTeamHandler(svcCtx))},
		{Method: http.MethodDelete, Path: "/api/teams/:id", Handler: perm("team:delete")(DeleteTeamHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/teams/:id/members", Handler: perm("team:read")(ListTeamMembersHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/teams/:id/members", Handler: perm("team:write")(AddTeamMemberHandler(svcCtx))},
		{Method: http.MethodDelete, Path: "/api/teams/:id/members/:username", Handler: perm("team:write")(RemoveTeamMemberHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/teams/:id/roles", Handler: perm("team:read")(ListTeamRolesHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/teams/:id/roles", Handler: perm("team:write")(SetTeamRolesHandler(svcCtx))},

		// 数据源配置（页面可管理）
		{Method: http.MethodGet, Path: "/api/datasources", Handler: perm("datasource:read")(ListDataSourcesHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/datasources", Handler: perm("datasource:write")(CreateDataSourceHandler(svcCtx))},
		{Method: http.MethodPut, Path: "/api/datasources/:id", Handler: perm("datasource:write")(UpdateDataSourceHandler(svcCtx))},
		{Method: http.MethodDelete, Path: "/api/datasources/:id", Handler: perm("datasource:delete")(DeleteDataSourceHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/datasources/:id/test", Handler: perm("datasource:read")(TestDataSourceHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/datasources/test", Handler: perm("datasource:read")(TestDataSourceConfigHandler(svcCtx))},

		// 定时任务配置
		{Method: http.MethodGet, Path: "/api/schedule", Handler: perm("schedule:read")(GetScheduleHandler(svcCtx))},
		{Method: http.MethodPut, Path: "/api/schedule", Handler: perm("schedule:write")(UpdateScheduleHandler(svcCtx))},

		// 数据同步
		{Method: http.MethodPost, Path: "/api/sync/datasource", Handler: perm("sync:write")(SyncDataSourceHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/sync/casdoor", Handler: perm("sync:write")(SyncCasdoorHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/sync/full", Handler: perm("sync:write")(SyncFullHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/sync/logs", Handler: perm("sync:read")(GetSyncLogsHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/sync/progress", Handler: perm("sync:read")(GetSyncProgressHandler(svcCtx))},

		// Casdoor 应用管理（含回调地址白名单）
		{Method: http.MethodGet, Path: "/api/casdoor/apps", Handler: perm("casdoor:read")(ListApplicationsHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/casdoor/apps", Handler: perm("casdoor:write")(CreateApplicationHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/casdoor/apps/:name", Handler: perm("casdoor:read")(GetApplicationHandler(svcCtx))},
		{Method: http.MethodPut, Path: "/api/casdoor/apps/:name", Handler: perm("casdoor:write")(UpdateApplicationHandler(svcCtx))},
		{Method: http.MethodDelete, Path: "/api/casdoor/apps/:name", Handler: perm("casdoor:write")(DeleteApplicationHandler(svcCtx))},

		// Casdoor 组织管理
		{Method: http.MethodGet, Path: "/api/casdoor/orgs", Handler: perm("casdoor:read")(ListOrganizationsHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/casdoor/orgs", Handler: perm("casdoor:write")(CreateOrganizationHandler(svcCtx))},
		{Method: http.MethodPut, Path: "/api/casdoor/orgs/:name", Handler: perm("casdoor:write")(UpdateOrganizationHandler(svcCtx))},
		{Method: http.MethodDelete, Path: "/api/casdoor/orgs/:name", Handler: perm("casdoor:write")(DeleteOrganizationHandler(svcCtx))},

		// 权限清单（供角色配置使用）
		{Method: http.MethodGet, Path: "/api/permissions", Handler: perm("role:read")(ListPermissionsHandler(svcCtx))},
	})
}

// HealthHandler 健康检查
func HealthHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok(w, map[string]string{
			"status":  "ok",
			"service": "flyiam",
			"version": "1.0.0",
		})
	}
}

// ListPermissionsHandler 返回系统权限清单
func ListPermissionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok(w, model.AllPermissions)
	}
}
