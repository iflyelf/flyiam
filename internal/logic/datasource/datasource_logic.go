package datasource

import (
	"context"
	"fmt"

	"github.com/iflyelf/flyiam/internal/model"
	pkgds "github.com/iflyelf/flyiam/internal/pkg/datasource"
	"github.com/iflyelf/flyiam/internal/pkg/datasource/httpapi"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Logic 数据源配置业务逻辑
type Logic struct {
	db sqlx.SqlConn
}

// NewLogic 创建数据源配置业务逻辑
func NewLogic(db sqlx.SqlConn) *Logic {
	return &Logic{db: db}
}

// List 获取全部数据源配置
func (l *Logic) List(ctx context.Context) ([]*model.DataSourceConfig, error) {
	var list []*model.DataSourceConfig
	query := `SELECT * FROM datasource_configs ORDER BY priority DESC, id ASC`
	if err := l.db.QueryRowsCtx(ctx, &list, query); err != nil {
		return nil, fmt.Errorf("查询数据源列表失败: %w", err)
	}
	return list, nil
}

// Get 获取单个数据源配置
func (l *Logic) Get(ctx context.Context, id int64) (*model.DataSourceConfig, error) {
	var cfg model.DataSourceConfig
	query := `SELECT * FROM datasource_configs WHERE id = $1`
	if err := l.db.QueryRowCtx(ctx, &cfg, query, id); err != nil {
		return nil, fmt.Errorf("查询数据源失败: %w", err)
	}
	return &cfg, nil
}

// Create 创建数据源配置
func (l *Logic) Create(ctx context.Context, cfg *model.DataSourceConfig) (int64, error) {
	query := `
		INSERT INTO datasource_configs (
			name, type, enabled, url, method, auth_type, auth_token, auth_username, auth_password,
			timeout, sync_interval, auto_sync, priority, page_size, remark, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,NOW(),NOW())
		RETURNING id
	`
	var id int64
	err := l.db.QueryRowCtx(ctx, &id, query,
		cfg.Name, cfg.Type, cfg.Enabled, cfg.URL, cfg.Method, cfg.AuthType, cfg.AuthToken,
		cfg.AuthUsername, cfg.AuthPassword, cfg.Timeout, cfg.SyncInterval, cfg.AutoSync,
		cfg.Priority, cfg.PageSize, cfg.Remark,
	)
	if err != nil {
		return 0, fmt.Errorf("创建数据源失败: %w", err)
	}
	return id, nil
}

// Update 更新数据源配置
func (l *Logic) Update(ctx context.Context, cfg *model.DataSourceConfig) error {
	query := `
		UPDATE datasource_configs SET
			name=$1, type=$2, enabled=$3, url=$4, method=$5, auth_type=$6, auth_token=$7,
			auth_username=$8, auth_password=$9, timeout=$10, sync_interval=$11, auto_sync=$12,
			priority=$13, page_size=$14, remark=$15, updated_at=NOW()
		WHERE id=$16
	`
	_, err := l.db.ExecCtx(ctx, query,
		cfg.Name, cfg.Type, cfg.Enabled, cfg.URL, cfg.Method, cfg.AuthType, cfg.AuthToken,
		cfg.AuthUsername, cfg.AuthPassword, cfg.Timeout, cfg.SyncInterval, cfg.AutoSync,
		cfg.Priority, cfg.PageSize, cfg.Remark, cfg.ID,
	)
	if err != nil {
		return fmt.Errorf("更新数据源失败: %w", err)
	}
	return nil
}

// Delete 删除数据源配置
func (l *Logic) Delete(ctx context.Context, id int64) error {
	if _, err := l.db.ExecCtx(ctx, `DELETE FROM datasource_configs WHERE id = $1`, id); err != nil {
		return fmt.Errorf("删除数据源失败: %w", err)
	}
	return nil
}

// BuildSources 从启用的数据库配置构建数据源实例
func (l *Logic) BuildSources(ctx context.Context) ([]pkgds.DataSource, error) {
	configs, err := l.listEnabled(ctx)
	if err != nil {
		return nil, err
	}
	return BuildSourcesFromConfigs(ctx, configs)
}

// BuildSourcesFromConfigs 根据给定配置构建数据源实例（用于连接测试等）
func BuildSourcesFromConfigs(_ context.Context, configs []*model.DataSourceConfig) ([]pkgds.DataSource, error) {
	sources := make([]pkgds.DataSource, 0, len(configs))
	for _, c := range configs {
		src, err := buildSource(*c)
		if err != nil {
			return nil, fmt.Errorf("构建数据源 %s 失败: %w", c.Name, err)
		}
		sources = append(sources, src)
	}
	return sources, nil
}

// listEnabled 获取启用的数据源
func (l *Logic) listEnabled(ctx context.Context) ([]*model.DataSourceConfig, error) {
	var list []*model.DataSourceConfig
	query := `SELECT * FROM datasource_configs WHERE enabled = TRUE ORDER BY priority DESC, id ASC`
	if err := l.db.QueryRowsCtx(ctx, &list, query); err != nil {
		return nil, fmt.Errorf("查询启用数据源失败: %w", err)
	}
	return list, nil
}

// buildSource 根据配置构建数据源
func buildSource(c model.DataSourceConfig) (pkgds.DataSource, error) {
	switch c.Type {
	case "httpapi", "":
		return httpapi.NewHttpApiSource(httpapi.Config{
			Enabled:      c.Enabled,
			Name:         c.Name,
			URL:          c.URL,
			Method:       c.Method,
			Timeout:      c.Timeout,
			SyncInterval: c.SyncInterval,
			Priority:     c.Priority,
			PageSize:     c.PageSize,
			Auth: httpapi.AuthConfig{
				Type:     c.AuthType,
				Token:    c.AuthToken,
				Username: c.AuthUsername,
				Password: c.AuthPassword,
			},
		})
	default:
		return nil, fmt.Errorf("不支持的数据源类型: %s", c.Type)
	}
}
