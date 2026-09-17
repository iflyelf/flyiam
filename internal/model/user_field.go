package model

import "time"

// 字段类型常量
const (
	FieldTypeText     = "text"
	FieldTypeTextarea = "textarea"
	FieldTypeNumber   = "number"
	FieldTypeSelect   = "select"
	FieldTypeDate     = "date"
)

// UserFieldDef 用户字段定义（页面可管理，存储于数据库，零硬编码）。
//
// 用途：
//  1. 定义用户的扩展字段（值存储于 Casdoor User.Properties，键为 FieldKey）；
//  2. 定义内置人事字段的显示名 / 是否在列表展示 / 是否可编辑；
//
// 数据源字段映射也以 FieldKey 为目标键，从而实现「数据源变化不改代码」。
type UserFieldDef struct {
	ID         int64     `db:"id" json:"id"`
	FieldKey   string    `db:"field_key" json:"fieldKey"`   // Properties 键名（唯一）
	Label      string    `db:"label" json:"label"`          // 显示名
	FieldType  string    `db:"field_type" json:"fieldType"` // text/textarea/number/select/date
	Options    string    `db:"options" json:"options"`      // select 选项（JSON 数组字符串）
	ShowInList bool      `db:"show_in_list" json:"showInList"`
	ShowInForm bool      `db:"show_in_form" json:"showInForm"`
	Editable   bool      `db:"editable" json:"editable"`
	Builtin    bool      `db:"builtin" json:"builtin"` // 内置字段不可删除、FieldKey 不可改
	SortOrder  int       `db:"sort_order" json:"sortOrder"`
	CreatedAt  time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt  time.Time `db:"updated_at" json:"updatedAt"`
}

// BuiltinUserFields 返回内置人事字段定义。
//
// 这些键与历史版本硬编码的 Properties 键保持一致（empCode/compileType/deptNameLv0...），
// 因此对既有数据向后兼容。首次启动写库，之后管理员可改显示名与可见性。
func BuiltinUserFields() []UserFieldDef {
	return []UserFieldDef{
		{FieldKey: "empCode", Label: "工号", FieldType: FieldTypeText, ShowInList: true, ShowInForm: true, Editable: true, Builtin: true, SortOrder: 10},
		{FieldKey: "deptNameLv0", Label: "零级部门", FieldType: FieldTypeText, ShowInList: false, ShowInForm: true, Editable: true, Builtin: true, SortOrder: 20},
		{FieldKey: "deptNameLv1", Label: "一级部门", FieldType: FieldTypeText, ShowInList: true, ShowInForm: true, Editable: true, Builtin: true, SortOrder: 30},
		{FieldKey: "deptNameLv2", Label: "二级部门", FieldType: FieldTypeText, ShowInList: true, ShowInForm: true, Editable: true, Builtin: true, SortOrder: 40},
		{FieldKey: "compileType", Label: "编制类型", FieldType: FieldTypeSelect, Options: `["正编","外包","实习"]`, ShowInList: false, ShowInForm: true, Editable: true, Builtin: true, SortOrder: 50},
		{FieldKey: "superior", Label: "上级账号", FieldType: FieldTypeText, ShowInList: false, ShowInForm: true, Editable: true, Builtin: true, SortOrder: 60},
		{FieldKey: "deptIdLv0", Label: "零级部门ID", FieldType: FieldTypeText, ShowInList: false, ShowInForm: false, Editable: true, Builtin: true, SortOrder: 70},
		{FieldKey: "deptIdLv1", Label: "一级部门ID", FieldType: FieldTypeText, ShowInList: false, ShowInForm: false, Editable: true, Builtin: true, SortOrder: 80},
		{FieldKey: "deptIdLv2", Label: "二级部门ID", FieldType: FieldTypeText, ShowInList: false, ShowInForm: false, Editable: true, Builtin: true, SortOrder: 90},
	}
}
