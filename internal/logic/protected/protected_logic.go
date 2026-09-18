package protected

import (
	"context"
	"fmt"
	"strings"

	"github.com/iflyelf/flyiam/internal/model"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Logic 受保护用户业务逻辑
type Logic struct {
	db sqlx.SqlConn
}

// NewLogic 创建受保护用户业务逻辑
func NewLogic(db sqlx.SqlConn) *Logic {
	return &Logic{db: db}
}

// List 获取受保护用户列表
func (l *Logic) List(ctx context.Context) ([]*model.ProtectedUser, error) {
	var list []*model.ProtectedUser
	query := `SELECT id, domain_account, COALESCE(remark,'') AS remark, created_at
		FROM protected_users ORDER BY id ASC`
	if err := l.db.QueryRowsCtx(ctx, &list, query); err != nil {
		return nil, fmt.Errorf("查询受保护用户失败: %w", err)
	}
	return list, nil
}

// Accounts 返回受保护用户域账号集合
func (l *Logic) Accounts(ctx context.Context) ([]string, error) {
	list, err := l.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		out = append(out, item.DomainAccount)
	}
	return out, nil
}

// Add 新增受保护用户（幂等）
func (l *Logic) Add(ctx context.Context, account, remark string) error {
	query := `INSERT INTO protected_users (domain_account, remark)
		VALUES ($1, $2) ON CONFLICT (domain_account) DO UPDATE SET remark = EXCLUDED.remark`
	if _, err := l.db.ExecCtx(ctx, query, account, remark); err != nil {
		return fmt.Errorf("新增受保护用户失败: %w", err)
	}
	return nil
}

// AddBatch 批量新增受保护用户（幂等，忽略空账号并去重）
//
// 返回实际写入的账号数。
func (l *Logic) AddBatch(ctx context.Context, accounts []string, remark string) (int, error) {
	seen := make(map[string]struct{}, len(accounts))
	n := 0
	for _, a := range accounts {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if _, ok := seen[a]; ok {
			continue
		}
		seen[a] = struct{}{}
		if err := l.Add(ctx, a, remark); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// Delete 删除受保护用户
func (l *Logic) Delete(ctx context.Context, account string) error {
	if _, err := l.db.ExecCtx(ctx, `DELETE FROM protected_users WHERE domain_account = $1`, account); err != nil {
		return fmt.Errorf("删除受保护用户失败: %w", err)
	}
	return nil
}

// SeedIfEmpty 当表为空时写入种子数据
func (l *Logic) SeedIfEmpty(ctx context.Context, seeds []string) error {
	var count int64
	if err := l.db.QueryRowCtx(ctx, &count, `SELECT COUNT(*) FROM protected_users`); err != nil {
		return fmt.Errorf("统计受保护用户失败: %w", err)
	}
	if count > 0 {
		return nil
	}
	for _, s := range seeds {
		if s == "" {
			continue
		}
		if err := l.Add(ctx, s, "系统初始化"); err != nil {
			return err
		}
	}
	return nil
}
