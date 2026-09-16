package rbac

import (
	"context"
	"fmt"

	"github.com/iflyelf/flyiam/internal/model"
	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Logic 权限（角色 / 团队）业务逻辑
type Logic struct {
	db         sqlx.SqlConn
	adminUsers []string
}

// NewLogic 创建权限业务逻辑
func NewLogic(db sqlx.SqlConn, adminUsers []string) *Logic {
	return &Logic{db: db, adminUsers: adminUsers}
}

// IsSuperAdmin 判断是否超级管理员（配置的管理员名单）
func (l *Logic) IsSuperAdmin(username string) bool {
	for _, u := range l.adminUsers {
		if u == username {
			return true
		}
	}
	return false
}

// UserPermissions 计算用户的有效权限
//
// 规则：超级管理员拥有全部权限；否则聚合其所在启用团队关联角色的权限并集。
func (l *Logic) UserPermissions(ctx context.Context, username string) ([]string, error) {
	if l.IsSuperAdmin(username) {
		return append([]string{}, model.AllPermissions...), nil
	}

	var perms model.StringArray
	query := `
		SELECT DISTINCT unnest(r.permissions) AS perm
		FROM team_members tm
		JOIN teams t ON t.id = tm.team_id AND t.status = 1
		JOIN team_roles tr ON tr.team_id = t.id
		JOIN roles r ON r.id = tr.role_id
		WHERE tm.username = $1
	`
	if err := l.db.QueryRowsCtx(ctx, &perms, query, username); err != nil {
		return nil, fmt.Errorf("查询用户权限失败: %w", err)
	}
	return []string(perms), nil
}

// HasPermission 判断用户是否拥有指定权限
func (l *Logic) HasPermission(ctx context.Context, username, permission string) (bool, error) {
	if l.IsSuperAdmin(username) {
		return true, nil
	}
	perms, err := l.UserPermissions(ctx, username)
	if err != nil {
		return false, err
	}
	for _, p := range perms {
		if p == permission || p == "*" {
			return true, nil
		}
	}
	return false, nil
}

// ============================ 角色 ============================

// ListRoles 角色列表
func (l *Logic) ListRoles(ctx context.Context, keyword string) ([]*model.Role, error) {
	var list []*model.Role
	query := `SELECT id, name, code, COALESCE(description,'') AS description,
		COALESCE(permissions,'{}') AS permissions, created_at, updated_at
		FROM roles`
	args := []interface{}{}
	if keyword != "" {
		query += ` WHERE name ILIKE $1 OR code ILIKE $1`
		args = append(args, "%"+keyword+"%")
	}
	query += ` ORDER BY id ASC`
	if err := l.db.QueryRowsCtx(ctx, &list, query, args...); err != nil {
		return nil, fmt.Errorf("查询角色列表失败: %w", err)
	}
	return list, nil
}

// GetRole 角色详情
func (l *Logic) GetRole(ctx context.Context, id int64) (*model.Role, error) {
	var role model.Role
	query := `SELECT id, name, code, COALESCE(description,'') AS description,
		COALESCE(permissions,'{}') AS permissions, created_at, updated_at
		FROM roles WHERE id = $1`
	if err := l.db.QueryRowCtx(ctx, &role, query, id); err != nil {
		return nil, fmt.Errorf("查询角色失败: %w", err)
	}
	return &role, nil
}

// CreateRole 创建角色
func (l *Logic) CreateRole(ctx context.Context, name, code, description string, permissions []string) (int64, error) {
	if code == "" {
		code = name
	}
	query := `INSERT INTO roles (name, code, description, permissions)
		VALUES ($1, $2, $3, $4) RETURNING id`
	var id int64
	if err := l.db.QueryRowCtx(ctx, &id, query, name, code, description, pq.Array(permissions)); err != nil {
		return 0, fmt.Errorf("创建角色失败: %w", err)
	}
	return id, nil
}

// UpdateRole 更新角色
func (l *Logic) UpdateRole(ctx context.Context, id int64, name, description string, permissions []string) error {
	query := `UPDATE roles SET name=$1, description=$2, permissions=$3, updated_at=NOW() WHERE id=$4`
	if _, err := l.db.ExecCtx(ctx, query, name, description, pq.Array(permissions), id); err != nil {
		return fmt.Errorf("更新角色失败: %w", err)
	}
	return nil
}

// DeleteRole 删除角色
func (l *Logic) DeleteRole(ctx context.Context, id int64) error {
	if _, err := l.db.ExecCtx(ctx, `DELETE FROM roles WHERE id=$1`, id); err != nil {
		return fmt.Errorf("删除角色失败: %w", err)
	}
	return nil
}

// ============================ 团队 ============================

// ListTeams 团队列表（含成员数与角色数）
func (l *Logic) ListTeams(ctx context.Context, keyword string) ([]*model.Team, error) {
	var list []*model.Team
	query := `
		SELECT t.id, t.name, t.code, COALESCE(t.description,'') AS description, t.status,
			COALESCE(t.created_by,'') AS created_by,
			(SELECT COUNT(*) FROM team_members tm WHERE tm.team_id = t.id) AS member_count,
			(SELECT COUNT(*) FROM team_roles tr WHERE tr.team_id = t.id) AS role_count,
			t.created_at, t.updated_at
		FROM teams t`
	args := []interface{}{}
	if keyword != "" {
		query += ` WHERE t.name ILIKE $1 OR t.code ILIKE $1`
		args = append(args, "%"+keyword+"%")
	}
	query += ` ORDER BY t.id ASC`
	if err := l.db.QueryRowsCtx(ctx, &list, query, args...); err != nil {
		return nil, fmt.Errorf("查询团队列表失败: %w", err)
	}
	return list, nil
}

// CreateTeam 创建团队
func (l *Logic) CreateTeam(ctx context.Context, name, code, description, createdBy string) (int64, error) {
	if code == "" {
		code = name
	}
	query := `INSERT INTO teams (name, code, description, created_by)
		VALUES ($1, $2, $3, $4) RETURNING id`
	var id int64
	if err := l.db.QueryRowCtx(ctx, &id, query, name, code, description, createdBy); err != nil {
		return 0, fmt.Errorf("创建团队失败: %w", err)
	}
	return id, nil
}

// UpdateTeam 更新团队
func (l *Logic) UpdateTeam(ctx context.Context, id int64, name, description string, status int) error {
	query := `UPDATE teams SET name=$1, description=$2, status=$3, updated_at=NOW() WHERE id=$4`
	if _, err := l.db.ExecCtx(ctx, query, name, description, status, id); err != nil {
		return fmt.Errorf("更新团队失败: %w", err)
	}
	return nil
}

// DeleteTeam 删除团队
func (l *Logic) DeleteTeam(ctx context.Context, id int64) error {
	if _, err := l.db.ExecCtx(ctx, `DELETE FROM teams WHERE id=$1`, id); err != nil {
		return fmt.Errorf("删除团队失败: %w", err)
	}
	return nil
}

// ============================ 团队成员 ============================

// ListTeamMembers 团队成员列表
func (l *Logic) ListTeamMembers(ctx context.Context, teamID int64) ([]*model.TeamMember, error) {
	var list []*model.TeamMember
	query := `SELECT id, team_id, username, COALESCE(display_name,'') AS display_name, created_at
		FROM team_members WHERE team_id = $1 ORDER BY id ASC`
	if err := l.db.QueryRowsCtx(ctx, &list, query, teamID); err != nil {
		return nil, fmt.Errorf("查询团队成员失败: %w", err)
	}
	return list, nil
}

// AddTeamMember 添加团队成员
func (l *Logic) AddTeamMember(ctx context.Context, teamID int64, username, displayName string) error {
	query := `INSERT INTO team_members (team_id, username, display_name)
		VALUES ($1, $2, $3) ON CONFLICT (team_id, username) DO UPDATE SET display_name = EXCLUDED.display_name`
	if _, err := l.db.ExecCtx(ctx, query, teamID, username, displayName); err != nil {
		return fmt.Errorf("添加团队成员失败: %w", err)
	}
	return nil
}

// RemoveTeamMember 移除团队成员
func (l *Logic) RemoveTeamMember(ctx context.Context, teamID int64, username string) error {
	if _, err := l.db.ExecCtx(ctx, `DELETE FROM team_members WHERE team_id=$1 AND username=$2`, teamID, username); err != nil {
		return fmt.Errorf("移除团队成员失败: %w", err)
	}
	return nil
}

// ============================ 团队角色 ============================

// ListTeamRoles 团队已授权角色 ID
func (l *Logic) ListTeamRoles(ctx context.Context, teamID int64) ([]int64, error) {
	var ids []int64
	query := `SELECT role_id FROM team_roles WHERE team_id = $1 ORDER BY role_id ASC`
	if err := l.db.QueryRowsCtx(ctx, &ids, query, teamID); err != nil {
		return nil, fmt.Errorf("查询团队角色失败: %w", err)
	}
	return ids, nil
}

// SetTeamRoles 覆盖式设置团队角色
func (l *Logic) SetTeamRoles(ctx context.Context, teamID int64, roleIDs []int64) error {
	if _, err := l.db.ExecCtx(ctx, `DELETE FROM team_roles WHERE team_id=$1`, teamID); err != nil {
		return fmt.Errorf("清理团队角色失败: %w", err)
	}
	for _, rid := range roleIDs {
		if _, err := l.db.ExecCtx(ctx,
			`INSERT INTO team_roles (team_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			teamID, rid); err != nil {
			return fmt.Errorf("设置团队角色失败: %w", err)
		}
	}
	return nil
}
