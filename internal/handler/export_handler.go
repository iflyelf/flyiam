package handler

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/iflyelf/flyiam/internal/svc"
)

// ExportUser 服务间用户导出结构。
//
// 只暴露外部系统需要的字段，避免把 Casdoor 内部对象（密码哈希、标签等）带出去。
type ExportUser struct {
	Username    string            `json:"username"`
	DisplayName string            `json:"displayName"`
	Email       string            `json:"email"`
	Phone       string            `json:"phone"`
	Properties  map[string]string `json:"properties"`
}

// ExportUsersHandler 服务间导出用户（分页）。
//
// 路由：GET /api/users/export?page=1&pageSize=500（需 X-Service-Token）
// 响应 data 为数组；响应头 X-Total-Count 为总数，供调用方判断是否继续翻页。
func ExportUsersHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svcCtx.Casdoor() == nil {
			fail(w, http.StatusServiceUnavailable, "Casdoor 未初始化")
			return
		}
		page := atoiDefault(r.URL.Query().Get("page"), 1)
		if page < 1 {
			page = 1
		}
		pageSize := atoiDefault(r.URL.Query().Get("pageSize"), 500)
		if pageSize < 1 {
			pageSize = 500
		}
		if pageSize > 1000 {
			pageSize = 1000
		}
		users, total, err := svcCtx.Casdoor().GetUsersPage(page, pageSize, "", "")
		if err != nil {
			fail(w, http.StatusBadGateway, "查询 Casdoor 用户失败: "+err.Error())
			return
		}
		out := make([]ExportUser, 0, len(users))
		for _, u := range users {
			props := u.Properties
			if props == nil {
				props = map[string]string{}
			}
			out = append(out, ExportUser{
				Username:    u.Name,
				DisplayName: u.DisplayName,
				Email:       u.Email,
				Phone:       u.Phone,
				Properties:  props,
			})
		}
		w.Header().Set("X-Total-Count", strconv.Itoa(total))
		ok(w, out)
	}
}

// ExportDepartment 服务间组织树节点。
type ExportDepartment struct {
	DeptID    string `json:"deptId"`
	DeptName  string `json:"deptName"`
	ParentID  string `json:"parentId"`
	Level     int    `json:"level"`
	FullPath  string `json:"fullPath"`
	UserCount int    `json:"userCount"`
}

// ExportDepartmentsHandler 服务间导出组织树。
//
// 组织数据不单独维护：与数据源同步一致，从用户属性 deptIdLv0/1/2、deptNameLv0/1/2
// 推导（ID 缺失时以「name:父路径/名称」生成稳定键）。同一部门自动去重，
// 用户计入其最深一级部门。
//
// 路由：GET /api/departments/export（需 X-Service-Token）
func ExportDepartmentsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svcCtx.Casdoor() == nil {
			fail(w, http.StatusServiceUnavailable, "Casdoor 未初始化")
			return
		}
		const pageSize = 500
		const maxUsers = 200000
		deptMap := map[string]*ExportDepartment{}
		seen := 0
		for page := 1; page <= maxUsers/pageSize; page++ {
			users, total, err := svcCtx.Casdoor().GetUsersPage(page, pageSize, "", "")
			if err != nil {
				fail(w, http.StatusBadGateway, "查询 Casdoor 用户失败: "+err.Error())
				return
			}
			for _, u := range users {
				applyUserDepartments(deptMap, u.Properties)
			}
			seen += len(users)
			if len(users) < pageSize || seen >= total {
				break
			}
		}
		out := make([]ExportDepartment, 0, len(deptMap))
		for _, d := range deptMap {
			out = append(out, *d)
		}
		sort.Slice(out, func(i, j int) bool {
			if out[i].FullPath == out[j].FullPath {
				return out[i].DeptID < out[j].DeptID
			}
			return out[i].FullPath < out[j].FullPath
		})
		ok(w, out)
	}
}

// applyUserDepartments 把单个用户的部门属性并入部门表。
//
// 逐级处理：某级 ID/名称都为空则跳过该级，父级保持上一级已解析的键。
// 这样即使只有二级部门名，也能生成可读的路径。
func applyUserDepartments(m map[string]*ExportDepartment, props map[string]string) {
	if len(props) == 0 {
		return
	}
	type levelProps struct {
		id, name string
		level    int
	}
	levels := []levelProps{
		{props["deptIdLv0"], props["deptNameLv0"], 0},
		{props["deptIdLv1"], props["deptNameLv1"], 1},
		{props["deptIdLv2"], props["deptNameLv2"], 2},
	}
	parentKey := ""
	leafKey := ""
	for _, l := range levels {
		name := strings.TrimSpace(l.name)
		key := strings.TrimSpace(l.id)
		if key == "" && name != "" {
			// 无部门 ID 的部署：用「父键 + 名称」生成稳定键，避免同名部门被错误合并。
			key = "name:" + parentKey + "/" + name
		}
		if key == "" {
			continue
		}
		if name == "" {
			name = key
		}
		node, okNode := m[key]
		if !okNode {
			fullPath := name
			if p, okParent := m[parentKey]; okParent && parentKey != "" {
				fullPath = p.FullPath + "/" + name
			}
			node = &ExportDepartment{
				DeptID: key, DeptName: name, ParentID: parentKey,
				Level: l.level, FullPath: fullPath,
			}
			m[key] = node
		}
		parentKey = key
		leafKey = key
	}
	// 用户计入最深一级非空部门。
	if leafKey != "" {
		m[leafKey].UserCount++
	}
}
