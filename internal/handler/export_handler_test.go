package handler

import "testing"

// TestApplyUserDepartments 从用户属性推导组织树：层级、父级、路径与人数。
func TestApplyUserDepartments(t *testing.T) {
	m := map[string]*ExportDepartment{}
	props := map[string]string{
		"deptIdLv0": "g", "deptNameLv0": "集团",
		"deptIdLv1": "rd", "deptNameLv1": "研发中心",
		"deptIdLv2": "plat", "deptNameLv2": "平台组",
	}
	applyUserDepartments(m, props)
	applyUserDepartments(m, props) // 第二个用户，同一部门

	// 只有一级部门（计入 rd）
	applyUserDepartments(m, map[string]string{
		"deptIdLv0": "g", "deptNameLv0": "集团",
		"deptIdLv1": "rd", "deptNameLv1": "研发中心",
	})

	if len(m) != 3 {
		t.Fatalf("应有 3 个部门, got %d: %+v", len(m), m)
	}
	if d := m["g"]; d == nil || d.ParentID != "" || d.Level != 0 || d.FullPath != "集团" || d.UserCount != 0 {
		t.Fatalf("零级部门不符: %+v", d)
	}
	if d := m["rd"]; d == nil || d.ParentID != "g" || d.Level != 1 ||
		d.FullPath != "集团/研发中心" || d.UserCount != 1 {
		t.Fatalf("一级部门不符: %+v", d)
	}
	if d := m["plat"]; d == nil || d.ParentID != "rd" || d.Level != 2 ||
		d.FullPath != "集团/研发中心/平台组" || d.UserCount != 2 {
		t.Fatalf("二级部门不符: %+v", d)
	}
}

// TestApplyUserDepartmentsNameKey 无部门 ID 时用父路径+名称生成稳定键。
func TestApplyUserDepartmentsNameKey(t *testing.T) {
	m := map[string]*ExportDepartment{}
	applyUserDepartments(m, map[string]string{
		"deptNameLv0": "总部",
		"deptNameLv1": "财务部",
	})
	applyUserDepartments(m, map[string]string{
		"deptNameLv0": "总部",
		"deptNameLv1": "财务部",
	})
	if len(m) != 2 {
		t.Fatalf("同名部门应合并, got %d: %+v", len(m), m)
	}
	root := m["name:/总部"]
	if root == nil || root.UserCount != 0 || root.FullPath != "总部" {
		t.Fatalf("根部门不符: %+v", root)
	}
	child := m["name:name:/总部/财务部"]
	if child == nil || child.ParentID != "name:/总部" || child.UserCount != 2 {
		t.Fatalf("子部门不符: %+v", child)
	}
}
