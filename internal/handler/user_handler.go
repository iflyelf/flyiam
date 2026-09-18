package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	casdoorsdk "github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"github.com/iflyelf/flyiam/internal/model"
	"github.com/iflyelf/flyiam/internal/svc"
	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

// cleanupUserAssociations 级联清理用户在本地的关联数据（团队成员等）。
//
// 用户唯一存储于 Casdoor，但本地 team_members 以用户名为关联键且无外键，
// 删除用户后若不清理会残留悬挂授权；同名用户重建后可能继承旧权限。
func cleanupUserAssociations(ctx context.Context, svcCtx *svc.ServiceContext, names []string) {
	if len(names) == 0 {
		return
	}
	if _, err := svcCtx.DB.ExecCtx(ctx,
		`DELETE FROM team_members WHERE username = ANY($1)`, pq.Array(names)); err != nil {
		log.Printf("⚠️ 清理团队成员关联失败: %v", err)
	}
}

// userPayload 用户新增/更新入参
type userPayload struct {
	DomainAccount   string `json:"domainAccount"`
	EmployeeCode    string `json:"employeeCode"`
	Name            string `json:"name"`
	Avatar          string `json:"avatar"`
	Phone           string `json:"phone"`
	WorkEmail       string `json:"workEmail"`
	CompileType     string `json:"compileType"`
	SuperiorAccount string `json:"superiorAccount"`
	DeptNameLv0     string `json:"deptNameLv0"`
	DeptNameLv1     string `json:"deptNameLv1"`
	DeptNameLv2     string `json:"deptNameLv2"`
	Password        string `json:"password"`
	// Extra 动态字段值（键为 user_field_defs.field_key），
	// 使新增/编辑用户也能写入任意自定义字段，无需改代码。
	Extra map[string]string `json:"extra"`
}

// ListUsersHandler 用户列表（数据来源：Casdoor，服务端分页，避免全量拉取导致卡顿）
func ListUsersHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svcCtx.Casdoor() == nil {
			fail(w, http.StatusServiceUnavailable, "Casdoor 未初始化")
			return
		}
		page := atoiDefault(r.URL.Query().Get("page"), 1)
		if page < 1 {
			page = 1
		}
		pageSize := atoiDefault(r.URL.Query().Get("pageSize"), 20)
		if pageSize < 1 || pageSize > 200 {
			pageSize = 20
		}
		field := normalizeField(r.URL.Query().Get("field"))
		keyword := strings.TrimSpace(r.URL.Query().Get("keyword"))

		users, total, err := svcCtx.Casdoor().GetUsersPage(page, pageSize, field, keyword)
		if err != nil {
			fail(w, http.StatusBadGateway, "查询 Casdoor 用户失败: "+err.Error())
			return
		}
		views := make([]model.UserView, 0, len(users))
		for _, u := range users {
			views = append(views, casdoorToView(svcCtx, u))
		}
		okPage(w, views, int64(total), page, pageSize)
	}
}

// SearchUsersHandler 用户搜索（跨域账号/姓名，供选择器下拉使用）
//
// 与列表接口不同：不分页，按关键字同时匹配域账号与姓名，返回去重后的前 N 条，
// 适合「选择用户」场景（如新增受保护用户）。
func SearchUsersHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svcCtx.Casdoor() == nil {
			fail(w, http.StatusServiceUnavailable, "Casdoor 未初始化")
			return
		}
		keyword := strings.TrimSpace(r.URL.Query().Get("keyword"))
		limit := atoiDefault(r.URL.Query().Get("limit"), 50)
		if limit < 1 || limit > 200 {
			limit = 50
		}
		users, err := svcCtx.Casdoor().SearchUsers(keyword, limit)
		if err != nil {
			fail(w, http.StatusBadGateway, "查询 Casdoor 用户失败: "+err.Error())
			return
		}
		views := make([]model.UserView, 0, len(users))
		for _, u := range users {
			views = append(views, casdoorToView(svcCtx, u))
		}
		ok(w, views)
	}
}

// friendlyCasdoorErr 将 Casdoor 返回的英文错误转换为友好中文提示
func friendlyCasdoorErr(msg string) string {
	switch {
	case strings.Contains(msg, "Phone already exists"):
		return "手机号已被其他用户占用"
	case strings.Contains(msg, "Email already exists"):
		return "邮箱已被其他用户占用"
	case strings.Contains(msg, "Username already exists"):
		return "域账号已存在"
	case strings.Contains(msg, "Phone number is invalid"):
		return "手机号格式无效"
	case strings.Contains(msg, "Email is invalid"):
		return "邮箱格式无效"
	default:
		return msg
	}
}

// normalizeField 规范化搜索字段
//
// 说明：Casdoor 会对 field 执行 CamelToSnakeCase 后再拼到 SQL，
// 因此这里传入驼峰命名字段（如 displayName → display_name）。
func normalizeField(field string) string {
	switch field {
	case "name", "displayName", "email", "phone":
		return field
	default:
		return "name"
	}
}

// GetUserDetailHandler 用户详情（数据来源：Casdoor）
func GetUserDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svcCtx.Casdoor() == nil {
			fail(w, http.StatusServiceUnavailable, "Casdoor 未初始化")
			return
		}
		account := strings.TrimSpace(r.URL.Query().Get("domainAccount"))
		if account == "" {
			fail(w, http.StatusBadRequest, "domainAccount 不能为空")
			return
		}
		u, err := svcCtx.Casdoor().GetUser(account)
		if err != nil || u == nil {
			fail(w, http.StatusNotFound, "用户不存在")
			return
		}
		ok(w, casdoorToView(svcCtx, u))
	}
}

// GetUserStatsHandler 用户统计（数据来源：Casdoor）
func GetUserStatsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svcCtx.Casdoor() == nil {
			ok(w, map[string]interface{}{"total": 0, "protected": []string{}})
			return
		}
		total, err := svcCtx.Casdoor().GetUserCount()
		if err != nil {
			fail(w, http.StatusBadGateway, "查询 Casdoor 用户数失败: "+err.Error())
			return
		}
		ok(w, map[string]interface{}{
			"total":     total,
			"protected": svcCtx.Casdoor().ProtectedUsers(),
		})
	}
}

// CreateUserHandler 新增用户（写入 Casdoor）
func CreateUserHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svcCtx.Casdoor() == nil {
			fail(w, http.StatusServiceUnavailable, "Casdoor 未初始化")
			return
		}
		var p userPayload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败: "+err.Error())
			return
		}
		p.DomainAccount = strings.TrimSpace(p.DomainAccount)
		if p.DomainAccount == "" || p.Name == "" {
			fail(w, http.StatusBadRequest, "域账号与姓名不能为空")
			return
		}

		user := buildNewUser(svcCtx, &p)
		if okAdd, err := svcCtx.Casdoor().AddUser(user); err != nil {
			fail(w, http.StatusBadGateway, "新增用户失败: "+friendlyCasdoorErr(err.Error()))
			return
		} else if !okAdd {
			fail(w, http.StatusBadGateway, "新增用户失败")
			return
		}
		ok(w, casdoorToView(svcCtx, user))
	}
}

// UpdateUserHandler 更新用户（写入 Casdoor）
func UpdateUserHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svcCtx.Casdoor() == nil {
			fail(w, http.StatusServiceUnavailable, "Casdoor 未初始化")
			return
		}
		name := pathvar.Vars(r)["name"]
		if name == "" {
			fail(w, http.StatusBadRequest, "缺少用户标识")
			return
		}
		var p userPayload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败: "+err.Error())
			return
		}

		existing, err := svcCtx.Casdoor().GetUser(name)
		if err != nil || existing == nil {
			fail(w, http.StatusNotFound, "用户不存在")
			return
		}
		applyUserPayload(svcCtx, existing, &p)
		if _, err := svcCtx.Casdoor().UpdateUser(existing); err != nil {
			fail(w, http.StatusBadGateway, "更新用户失败: "+friendlyCasdoorErr(err.Error()))
			return
		}
		ok(w, casdoorToView(svcCtx, existing))
	}
}

// DeleteUserHandler 删除用户（写入 Casdoor，受保护用户不可删除）
func DeleteUserHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svcCtx.Casdoor() == nil {
			fail(w, http.StatusServiceUnavailable, "Casdoor 未初始化")
			return
		}
		name := pathvar.Vars(r)["name"]
		if name == "" {
			fail(w, http.StatusBadRequest, "缺少用户标识")
			return
		}
		if svcCtx.Casdoor().IsProtected(name) {
			fail(w, http.StatusForbidden, "该用户为受保护用户，禁止删除")
			return
		}
		if _, err := svcCtx.Casdoor().DeleteUser(name); err != nil {
			fail(w, http.StatusBadGateway, "删除用户失败: "+err.Error())
			return
		}
		cleanupUserAssociations(r.Context(), svcCtx, []string{name})
		ok(w, nil)
	}
}

// BatchDeleteUsersHandler 批量删除用户（跳过受保护用户）
func BatchDeleteUsersHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svcCtx.Casdoor() == nil {
			fail(w, http.StatusServiceUnavailable, "Casdoor 未初始化")
			return
		}
		var req struct {
			Names []string `json:"names"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败: "+err.Error())
			return
		}
		if len(req.Names) == 0 {
			fail(w, http.StatusBadRequest, "请选择要删除的用户")
			return
		}
		deleted, skipped, failed := svcCtx.Casdoor().BatchDeleteUsers(r.Context(), req.Names)

		// 计算成功删除的用户名，级联清理本地关联
		excluded := make(map[string]struct{}, len(skipped)+len(failed))
		for _, n := range skipped {
			excluded[n] = struct{}{}
		}
		for _, n := range failed {
			excluded[n] = struct{}{}
		}
		deletedNames := make([]string, 0, len(req.Names))
		for _, n := range req.Names {
			if _, skip := excluded[n]; !skip {
				deletedNames = append(deletedNames, n)
			}
		}
		cleanupUserAssociations(r.Context(), svcCtx, deletedNames)

		ok(w, map[string]interface{}{
			"deleted": deleted,
			"skipped": skipped,
			"failed":  failed,
		})
	}
}

// casdoorToView Casdoor 用户转统一视图（人事字段取自属性）
func casdoorToView(svcCtx *svc.ServiceContext, u *casdoorsdk.User) model.UserView {
	props := u.Properties
	if props == nil {
		props = map[string]string{}
	}
	// 复制一份全部属性，供前端按字段定义动态渲染（不改原 map）
	extra := make(map[string]string, len(props))
	for k, v := range props {
		extra[k] = v
	}
	status := "active"
	if u.IsForbidden {
		status = "forbidden"
	}
	view := model.UserView{
		Source:          "casdoor",
		DomainAccount:   u.Name,
		EmployeeCode:    props["empCode"],
		Name:            u.DisplayName,
		Avatar:          u.Avatar,
		Phone:           u.Phone,
		WorkEmail:       u.Email,
		CompileType:     props["compileType"],
		SuperiorAccount: props["superior"],
		DeptNameLv0:     props["deptNameLv0"],
		DeptNameLv1:     props["deptNameLv1"],
		DeptNameLv2:     props["deptNameLv2"],
		Status:          status,
		CasdoorSynced:   true,
		IsAdmin:         u.IsAdmin,
		Extra:           extra,
	}
	if svcCtx.Casdoor() != nil {
		view.IsProtected = svcCtx.Casdoor().IsProtected(u.Name)
	}
	return view
}

// buildNewUser 构建新用户
func buildNewUser(svcCtx *svc.ServiceContext, p *userPayload) *casdoorsdk.User {
	password := p.Password
	if password == "" {
		password = svcCtx.Config.Casdoor.DefaultPassword
	}
	return &casdoorsdk.User{
		Owner:             svcCtx.Config.Casdoor.OrganizationName,
		Name:              p.DomainAccount,
		DisplayName:       p.Name,
		Avatar:            p.Avatar,
		Email:             p.WorkEmail,
		Phone:             p.Phone,
		CountryCode:       svcCtx.Config.Casdoor.CountryCode,
		Password:          password,
		Type:              "normal-user",
		Language:          "zh",
		Tag:               "manual",
		Address:           []string{},
		Properties:        payloadProps(p),
		SignupApplication: svcCtx.Config.Casdoor.ApplicationName,
	}
}

// applyUserPayload 将入参应用到现有用户
func applyUserPayload(svcCtx *svc.ServiceContext, u *casdoorsdk.User, p *userPayload) {
	u.DisplayName = p.Name
	u.Avatar = p.Avatar
	u.Email = p.WorkEmail
	u.Phone = p.Phone
	u.CountryCode = svcCtx.Config.Casdoor.CountryCode
	u.Tag = "manual"
	if u.Properties == nil {
		u.Properties = map[string]string{}
	}
	for k, v := range payloadProps(p) {
		u.Properties[k] = v
	}
}

// payloadProps 提取用户属性（内置人事字段 + 自定义扩展字段）
//
// 自定义字段来自 Extra（键为 user_field_defs.field_key），
// 因此数据源/字段变化无需改代码，只需在页面维护字段定义。
func payloadProps(p *userPayload) map[string]string {
	props := map[string]string{
		"empCode":     p.EmployeeCode,
		"compileType": p.CompileType,
		"deptNameLv0": p.DeptNameLv0,
		"deptNameLv1": p.DeptNameLv1,
		"deptNameLv2": p.DeptNameLv2,
		"superior":    p.SuperiorAccount,
		"source":      "manual",
	}
	for k, v := range p.Extra {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		props[k] = v
	}
	return props
}

// atoiDefault 字符串转 int，失败返回默认值
func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return def
}

// atoiInt64 字符串转 int64
func atoiInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// ResetPasswordHandler 管理员重置用户密码为系统默认密码
func ResetPasswordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svcCtx.Casdoor() == nil {
			fail(w, http.StatusServiceUnavailable, "Casdoor 未初始化")
			return
		}
		name := pathvar.Vars(r)["name"]
		if name == "" {
			fail(w, http.StatusBadRequest, "缺少用户标识")
			return
		}
		var req struct {
			Password string `json:"password"`
		}
		// 允许自定义密码，留空则使用默认密码
		_ = json.NewDecoder(r.Body).Decode(&req)
		newPassword := req.Password
		if newPassword == "" {
			newPassword = svcCtx.Casdoor().DefaultPassword()
		}
		okPwd, err := svcCtx.Casdoor().SetUserPassword(name, "", newPassword)
		if err != nil {
			// 新旧密码相同时 Casdoor 会报错，视为重置成功（幂等）
			if strings.Contains(err.Error(), "must be different") {
				ok(w, map[string]string{"password": newPassword, "message": "密码未变更（与原密码相同）"})
				return
			}
			fail(w, http.StatusBadGateway, "重置密码失败: "+friendlyCasdoorErr(err.Error()))
			return
		}
		if !okPwd {
			fail(w, http.StatusBadGateway, "重置密码失败")
			return
		}
		ok(w, map[string]string{"password": newPassword})
	}
}

// SetUserAdminHandler 设置用户管理员标记
func SetUserAdminHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svcCtx.Casdoor() == nil {
			fail(w, http.StatusServiceUnavailable, "Casdoor 未初始化")
			return
		}
		name := pathvar.Vars(r)["name"]
		var req struct {
			IsAdmin bool `json:"isAdmin"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败")
			return
		}
		if svcCtx.Casdoor().IsProtected(name) && !req.IsAdmin {
			fail(w, http.StatusForbidden, "该用户为受保护用户，禁止取消管理员")
			return
		}
		if err := svcCtx.Casdoor().SetUserAdmin(name, req.IsAdmin); err != nil {
			fail(w, http.StatusBadGateway, err.Error())
			return
		}
		ok(w, nil)
	}
}
