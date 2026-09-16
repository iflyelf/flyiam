package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	casdoorsdk "github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"github.com/iflyelf/flyiam/internal/svc"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

// ============================ 受保护用户 ============================

// ListProtectedUsersHandler 受保护用户列表
func ListProtectedUsersHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := svcCtx.ProtectedLogic.List(r.Context())
		if err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok(w, list)
	}
}

// AddProtectedUserHandler 新增受保护用户
func AddProtectedUserHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			DomainAccount string `json:"domainAccount"`
			Remark        string `json:"remark"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败: "+err.Error())
			return
		}
		req.DomainAccount = strings.TrimSpace(req.DomainAccount)
		if req.DomainAccount == "" {
			fail(w, http.StatusBadRequest, "域账号不能为空")
			return
		}
		if err := svcCtx.ProtectedLogic.Add(r.Context(), req.DomainAccount, req.Remark); err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		refreshProtected(svcCtx, r)
		ok(w, nil)
	}
}

// DeleteProtectedUserHandler 删除受保护用户
func DeleteProtectedUserHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		account := pathvar.Vars(r)["account"]
		if account == "" {
			fail(w, http.StatusBadRequest, "缺少域账号")
			return
		}
		if err := svcCtx.ProtectedLogic.Delete(r.Context(), account); err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		refreshProtected(svcCtx, r)
		ok(w, nil)
	}
}

// refreshProtected 重新加载受保护用户到 Casdoor 客户端
func refreshProtected(svcCtx *svc.ServiceContext, r *http.Request) {
	if accounts, err := svcCtx.ProtectedLogic.Accounts(r.Context()); err == nil {
		svcCtx.CasdoorClient.SetProtected(accounts)
	}
}

// ============================ 应用管理 ============================

// applicationPayload 应用入参
type applicationPayload struct {
	Name           string   `json:"name"`
	DisplayName    string   `json:"displayName"`
	Organization   string   `json:"organization"`
	ClientId       string   `json:"clientId"`
	ClientSecret   string   `json:"clientSecret"`
	RedirectUris   []string `json:"redirectUris"`
	EnablePassword bool     `json:"enablePassword"`
	Cert           string   `json:"cert"`
}

// ListApplicationsHandler 应用列表
func ListApplicationsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svcCtx.CasdoorClient == nil {
			fail(w, http.StatusServiceUnavailable, "Casdoor 未初始化")
			return
		}
		apps, err := svcCtx.CasdoorClient.ListApplications()
		if err != nil {
			fail(w, http.StatusBadGateway, err.Error())
			return
		}
		ok(w, apps)
	}
}

// GetApplicationHandler 应用详情
func GetApplicationHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := pathvar.Vars(r)["name"]
		app, err := svcCtx.CasdoorClient.GetApplication(name)
		if err != nil || app == nil {
			fail(w, http.StatusNotFound, "应用不存在")
			return
		}
		ok(w, app)
	}
}

// CreateApplicationHandler 新增应用
func CreateApplicationHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p applicationPayload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败: "+err.Error())
			return
		}
		if p.Name == "" {
			fail(w, http.StatusBadRequest, "应用名称不能为空")
			return
		}
		app := &casdoorsdk.Application{
			Owner:               "admin",
			Name:                p.Name,
			DisplayName:         p.DisplayName,
			Organization:        p.Organization,
			ClientId:            p.ClientId,
			ClientSecret:        p.ClientSecret,
			RedirectUris:        normalizeList(p.RedirectUris),
			EnablePassword:      p.EnablePassword,
			EnableSigninSession: true,
			EnableAutoSignin:    true,
			EnableCodeSignin:    true,
			Cert:                p.Cert,
		}
		if _, err := svcCtx.CasdoorClient.AddApplication(app); err != nil {
			fail(w, http.StatusBadGateway, err.Error())
			return
		}
		ok(w, app)
	}
}

// UpdateApplicationHandler 更新应用（含回调地址白名单）
func UpdateApplicationHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := pathvar.Vars(r)["name"]
		var p applicationPayload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败: "+err.Error())
			return
		}
		app, err := svcCtx.CasdoorClient.GetApplication(name)
		if err != nil || app == nil {
			fail(w, http.StatusNotFound, "应用不存在")
			return
		}
		if p.DisplayName != "" {
			app.DisplayName = p.DisplayName
		}
		if p.Organization != "" {
			app.Organization = p.Organization
		}
		if p.ClientId != "" {
			app.ClientId = p.ClientId
		}
		if p.ClientSecret != "" {
			app.ClientSecret = p.ClientSecret
		}
		if p.Cert != "" {
			app.Cert = p.Cert
		}
		app.EnablePassword = p.EnablePassword
		if p.RedirectUris != nil {
			app.RedirectUris = normalizeList(p.RedirectUris)
		}
		if _, err := svcCtx.CasdoorClient.UpdateApplication(app); err != nil {
			fail(w, http.StatusBadGateway, err.Error())
			return
		}
		ok(w, app)
	}
}

// DeleteApplicationHandler 删除应用
func DeleteApplicationHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := pathvar.Vars(r)["name"]
		if name == "" {
			fail(w, http.StatusBadRequest, "缺少应用名称")
			return
		}
		if _, err := svcCtx.CasdoorClient.DeleteApplication(name); err != nil {
			fail(w, http.StatusBadGateway, err.Error())
			return
		}
		ok(w, nil)
	}
}

// ============================ 组织管理 ============================

// organizationPayload 组织入参
type organizationPayload struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

// ListOrganizationsHandler 组织列表
func ListOrganizationsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svcCtx.CasdoorClient == nil {
			fail(w, http.StatusServiceUnavailable, "Casdoor 未初始化")
			return
		}
		orgs, err := svcCtx.CasdoorClient.ListOrganizations()
		if err != nil {
			fail(w, http.StatusBadGateway, err.Error())
			return
		}
		ok(w, orgs)
	}
}

// CreateOrganizationHandler 新增组织
func CreateOrganizationHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p organizationPayload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败: "+err.Error())
			return
		}
		if p.Name == "" {
			fail(w, http.StatusBadRequest, "组织名称不能为空")
			return
		}
		org := &casdoorsdk.Organization{
			Owner:       "admin",
			Name:        p.Name,
			DisplayName: p.DisplayName,
		}
		if _, err := svcCtx.CasdoorClient.AddOrganization(org); err != nil {
			fail(w, http.StatusBadGateway, err.Error())
			return
		}
		ok(w, org)
	}
}

// UpdateOrganizationHandler 更新组织
func UpdateOrganizationHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := pathvar.Vars(r)["name"]
		var p organizationPayload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			fail(w, http.StatusBadRequest, "参数解析失败: "+err.Error())
			return
		}
		org, err := svcCtx.CasdoorClient.GetOrganization(name)
		if err != nil || org == nil {
			fail(w, http.StatusNotFound, "组织不存在")
			return
		}
		if p.DisplayName != "" {
			org.DisplayName = p.DisplayName
		}
		if _, err := svcCtx.CasdoorClient.UpdateOrganization(org); err != nil {
			fail(w, http.StatusBadGateway, err.Error())
			return
		}
		ok(w, org)
	}
}

// DeleteOrganizationHandler 删除组织
func DeleteOrganizationHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := pathvar.Vars(r)["name"]
		if name == "" {
			fail(w, http.StatusBadRequest, "缺少组织名称")
			return
		}
		if _, err := svcCtx.CasdoorClient.DeleteOrganization(name); err != nil {
			fail(w, http.StatusBadGateway, err.Error())
			return
		}
		ok(w, nil)
	}
}

// normalizeList 去空白、去重、去空项
func normalizeList(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, it := range items {
		v := strings.TrimSpace(it)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
