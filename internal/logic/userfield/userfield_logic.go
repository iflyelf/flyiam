// Package userfield 用户字段定义业务逻辑（页面可管理，存储于数据库）
package userfield

import (
	"context"
	"fmt"
	"strings"

	"github.com/iflyelf/flyiam/internal/model"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Logic 用户字段定义业务逻辑
type Logic struct {
	db sqlx.SqlConn
}

// NewLogic 创建用户字段定义业务逻辑
func NewLogic(db sqlx.SqlConn) *Logic {
	return &Logic{db: db}
}

// List 获取全部字段定义（按排序值升序）
func (l *Logic) List(ctx context.Context) ([]*model.UserFieldDef, error) {
	var list []*model.UserFieldDef
	query := `SELECT id, field_key, label, field_type, COALESCE(options,'') AS options,
		show_in_list, show_in_form, editable, builtin, sort_order, created_at, updated_at
		FROM user_field_defs ORDER BY sort_order ASC, id ASC`
	if err := l.db.QueryRowsCtx(ctx, &list, query); err != nil {
		return nil, fmt.Errorf("查询字段定义失败: %w", err)
	}
	return list, nil
}

// Get 获取单个字段定义
func (l *Logic) Get(ctx context.Context, id int64) (*model.UserFieldDef, error) {
	var d model.UserFieldDef
	query := `SELECT id, field_key, label, field_type, COALESCE(options,'') AS options,
		show_in_list, show_in_form, editable, builtin, sort_order, created_at, updated_at
		FROM user_field_defs WHERE id = $1`
	if err := l.db.QueryRowCtx(ctx, &d, query, id); err != nil {
		return nil, fmt.Errorf("查询字段定义失败: %w", err)
	}
	return &d, nil
}

// Current 返回当前生效的字段定义映射（fieldKey -> 定义），供用户列表/表单渲染使用
func (l *Logic) Current(ctx context.Context) (map[string]*model.UserFieldDef, error) {
	list, err := l.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]*model.UserFieldDef, len(list))
	for _, d := range list {
		out[d.FieldKey] = d
	}
	return out, nil
}

// Create 新增字段定义
func (l *Logic) Create(ctx context.Context, d *model.UserFieldDef) (int64, error) {
	if err := normalize(d); err != nil {
		return 0, err
	}
	query := `
		INSERT INTO user_field_defs
			(field_key, label, field_type, options, show_in_list, show_in_form, editable, builtin, sort_order, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,false,$8,NOW(),NOW())
		RETURNING id
	`
	var id int64
	err := l.db.QueryRowCtx(ctx, &id, query,
		d.FieldKey, d.Label, d.FieldType, d.Options, d.ShowInList, d.ShowInForm, d.Editable, d.SortOrder,
	)
	if err != nil {
		return 0, fmt.Errorf("新增字段定义失败: %w", err)
	}
	return id, nil
}

// Update 更新字段定义（内置字段的 FieldKey 与 Builtin 不可改）
func (l *Logic) Update(ctx context.Context, d *model.UserFieldDef) error {
	existing, err := l.Get(ctx, d.ID)
	if err != nil {
		return err
	}
	if existing.Builtin {
		// 内置字段不允许改 key
		d.FieldKey = existing.FieldKey
	}
	if err := normalize(d); err != nil {
		return err
	}
	query := `
		UPDATE user_field_defs SET
			field_key=$1, label=$2, field_type=$3, options=$4,
			show_in_list=$5, show_in_form=$6, editable=$7, sort_order=$8, updated_at=NOW()
		WHERE id=$9
	`
	if _, err := l.db.ExecCtx(ctx, query,
		d.FieldKey, d.Label, d.FieldType, d.Options, d.ShowInList, d.ShowInForm, d.Editable, d.SortOrder, d.ID,
	); err != nil {
		return fmt.Errorf("更新字段定义失败: %w", err)
	}
	return nil
}

// Delete 删除字段定义（内置字段不可删除）
func (l *Logic) Delete(ctx context.Context, id int64) error {
	existing, err := l.Get(ctx, id)
	if err != nil {
		return err
	}
	if existing.Builtin {
		return fmt.Errorf("内置字段 %s 不可删除，可改为不显示", existing.FieldKey)
	}
	if _, err := l.db.ExecCtx(ctx, `DELETE FROM user_field_defs WHERE id = $1`, id); err != nil {
		return fmt.Errorf("删除字段定义失败: %w", err)
	}
	return nil
}

// SeedBuiltin 写入缺失的内置字段（幂等），保证升级后内置字段存在
func (l *Logic) SeedBuiltin(ctx context.Context) error {
	for _, d := range model.BuiltinUserFields() {
		var count int64
		if err := l.db.QueryRowCtx(ctx, &count,
			`SELECT COUNT(*) FROM user_field_defs WHERE field_key = $1`, d.FieldKey); err != nil {
			return fmt.Errorf("检查内置字段失败: %w", err)
		}
		if count > 0 {
			continue
		}
		query := `
			INSERT INTO user_field_defs
				(field_key, label, field_type, options, show_in_list, show_in_form, editable, builtin, sort_order, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,true,$8,NOW(),NOW())
		`
		if _, err := l.db.ExecCtx(ctx, query,
			d.FieldKey, d.Label, d.FieldType, d.Options, d.ShowInList, d.ShowInForm, d.Editable, d.SortOrder,
		); err != nil {
			return fmt.Errorf("写入内置字段失败: %w", err)
		}
	}
	return nil
}

// normalize 规范化字段定义
func normalize(d *model.UserFieldDef) error {
	d.FieldKey = strings.TrimSpace(d.FieldKey)
	d.Label = strings.TrimSpace(d.Label)
	if d.FieldKey == "" {
		return fmt.Errorf("字段键不能为空")
	}
	if d.Label == "" {
		return fmt.Errorf("显示名不能为空")
	}
	if !isValidKey(d.FieldKey) {
		return fmt.Errorf("字段键只能包含字母、数字、下划线，且以字母开头")
	}
	if d.FieldType == "" {
		d.FieldType = model.FieldTypeText
	}
	switch d.FieldType {
	case model.FieldTypeText, model.FieldTypeTextarea, model.FieldTypeNumber, model.FieldTypeSelect, model.FieldTypeDate:
	default:
		return fmt.Errorf("不支持的字段类型: %s", d.FieldType)
	}
	d.Options = strings.TrimSpace(d.Options)
	return nil
}

// isValidKey 字段键必须是合法标识符（作为 Casdoor Properties 键，需与数据源映射一致）
func isValidKey(k string) bool {
	if k == "" {
		return false
	}
	for i, r := range k {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '_':
		case r >= '0' && r <= '9':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}
