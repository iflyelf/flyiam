package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/iflyelf/flyiam/internal/pkg/datasource"
)

// HttpApiSource HTTP API 数据源
type HttpApiSource struct {
	config Config
	client *http.Client
}

// Config HTTP API 数据源配置
type Config struct {
	Enabled      bool
	Name         string
	URL          string
	Method       string
	Timeout      int
	SyncInterval string
	Priority     int
	PageSize     int
	Auth         AuthConfig
}

// AuthConfig 认证配置
type AuthConfig struct {
	Type     string // none, bearer, basic, apikey
	Token    string
	Username string
	Password string
}

// apiResponse 数据源响应结构（count/currentPageNo/data）
type apiResponse struct {
	Count         int       `json:"count"`
	CurrentPageNo int       `json:"currentPageNo"`
	Data          []apiUser `json:"data"`
}

// apiUser 数据源用户结构（真实接口返回大写字段）
type apiUser struct {
	DOMACT            string `json:"DOMACT"`
	EMP_CODE          string `json:"EMP_CODE"`
	NAME              string `json:"NAME"`
	PHONE             string `json:"PHONE"`
	WORK_EMAIL        string `json:"WORK_EMAIL"`
	COMPILE_TYPE_NAME string `json:"COMPILE_TYPE_NAME"`
	SUPERIOR_DOMACT   string `json:"SUPERIOR_DOMACT"`
	DEPT_ID_LV0       string `json:"DEPT_ID_LV0"`
	DEPT_ID_LV1       string `json:"DEPT_ID_LV1"`
	DEPT_ID_LV2       string `json:"DEPT_ID_LV2"`
	DEPT_NAME_LV0     string `json:"DEPT_NAME_LV0"`
	DEPT_NAME_LV1     string `json:"DEPT_NAME_LV1"`
	DEPT_NAME_LV2     string `json:"DEPT_NAME_LV2"`
	STATUS            string `json:"STATUS"`
	SOURCE            string `json:"SOURCE"`
}

// NewHttpApiSource 创建 HTTP API 数据源
func NewHttpApiSource(cfg Config) (*HttpApiSource, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("HTTP API URL is required")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30
	}
	if cfg.Method == "" {
		cfg.Method = http.MethodGet
	}
	if cfg.PageSize <= 0 {
		cfg.PageSize = 1000
	}
	return &HttpApiSource{
		config: cfg,
		client: &http.Client{Timeout: time.Duration(cfg.Timeout) * time.Second},
	}, nil
}

// Name 返回数据源名称
func (h *HttpApiSource) Name() string { return h.config.Name }

// IsEnabled 返回是否启用
func (h *HttpApiSource) IsEnabled() bool { return h.config.Enabled }

// Priority 返回优先级
func (h *HttpApiSource) Priority() int { return h.config.Priority }

// Sync 执行数据同步（拉取校验）
func (h *HttpApiSource) Sync(ctx context.Context) error {
	_, err := h.GetUsers(ctx)
	return err
}

// GetUsers 分页拉取全部用户
func (h *HttpApiSource) GetUsers(ctx context.Context) ([]datasource.User, error) {
	var all []datasource.User
	page := 1
	for {
		batch, total, err := h.fetchPage(ctx, page)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) == 0 || len(all) >= total {
			break
		}
		page++
	}
	return all, nil
}

// fetchPage 拉取单页
func (h *HttpApiSource) fetchPage(ctx context.Context, page int) ([]datasource.User, int, error) {
	var body io.Reader
	if h.config.Method == http.MethodPost {
		payload, _ := json.Marshal(map[string]interface{}{
			"page":      page,
			"page_size": h.config.PageSize,
		})
		body = bytes.NewReader(payload)
	} else {
		// GET 时通过查询参数分页
		body = nil
	}

	url := h.config.URL
	if h.config.Method != http.MethodPost {
		sep := "?"
		if bytes.ContainsRune([]byte(url), '?') {
			sep = "&"
		}
		url = fmt.Sprintf("%s%spage=%d&page_size=%d", url, sep, page, h.config.PageSize)
	}

	req, err := http.NewRequestWithContext(ctx, h.config.Method, url, body)
	if err != nil {
		return nil, 0, fmt.Errorf("创建请求失败: %w", err)
	}
	if h.config.Method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	h.addAuth(req)

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("请求失败: status=%d body=%s", resp.StatusCode, string(b))
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("读取响应失败: %w", err)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(raw, &apiResp); err != nil {
		return nil, 0, fmt.Errorf("解析响应失败: %w", err)
	}

	users := make([]datasource.User, 0, len(apiResp.Data))
	for _, u := range apiResp.Data {
		users = append(users, datasource.User{
			DomainAccount:   u.DOMACT,
			EmployeeCode:    u.EMP_CODE,
			Name:            u.NAME,
			Phone:           u.PHONE,
			WorkEmail:       u.WORK_EMAIL,
			CompileType:     u.COMPILE_TYPE_NAME,
			SuperiorAccount: u.SUPERIOR_DOMACT,
			DeptIDLv0:       u.DEPT_ID_LV0,
			DeptIDLv1:       u.DEPT_ID_LV1,
			DeptIDLv2:       u.DEPT_ID_LV2,
			DeptNameLv0:     u.DEPT_NAME_LV0,
			DeptNameLv1:     u.DEPT_NAME_LV1,
			DeptNameLv2:     u.DEPT_NAME_LV2,
			Source:          h.Name(),
			Priority:        h.Priority(),
		})
	}
	return users, apiResp.Count, nil
}

// GetDepartments 从用户数据中提取部门
func (h *HttpApiSource) GetDepartments(ctx context.Context) ([]datasource.Department, error) {
	users, err := h.GetUsers(ctx)
	if err != nil {
		return nil, err
	}
	deptMap := make(map[string]datasource.Department)
	for _, u := range users {
		if u.DeptIDLv0 != "" {
			deptMap[u.DeptIDLv0] = datasource.Department{DeptID: u.DeptIDLv0, DeptName: u.DeptNameLv0, Level: 0, FullPath: u.DeptNameLv0, Source: h.Name()}
		}
		if u.DeptIDLv1 != "" {
			deptMap[u.DeptIDLv1] = datasource.Department{DeptID: u.DeptIDLv1, DeptName: u.DeptNameLv1, ParentID: u.DeptIDLv0, Level: 1, FullPath: u.DeptNameLv0 + "/" + u.DeptNameLv1, Source: h.Name()}
		}
		if u.DeptIDLv2 != "" {
			deptMap[u.DeptIDLv2] = datasource.Department{DeptID: u.DeptIDLv2, DeptName: u.DeptNameLv2, ParentID: u.DeptIDLv1, Level: 2, FullPath: u.DeptNameLv0 + "/" + u.DeptNameLv1 + "/" + u.DeptNameLv2, Source: h.Name()}
		}
	}
	depts := make([]datasource.Department, 0, len(deptMap))
	for _, d := range deptMap {
		depts = append(depts, d)
	}
	return depts, nil
}

// GetSyncInterval 获取同步间隔
func (h *HttpApiSource) GetSyncInterval() time.Duration {
	d, err := time.ParseDuration(h.config.SyncInterval)
	if err != nil {
		return 6 * time.Hour
	}
	return d
}

// HealthCheck 健康检查
func (h *HttpApiSource) HealthCheck(ctx context.Context) error {
	_, _, err := h.fetchPage(ctx, 1)
	return err
}

// addAuth 添加认证信息
func (h *HttpApiSource) addAuth(req *http.Request) {
	switch h.config.Auth.Type {
	case "bearer":
		if h.config.Auth.Token != "" {
			req.Header.Set("Authorization", "Bearer "+h.config.Auth.Token)
		}
	case "basic":
		if h.config.Auth.Username != "" {
			req.SetBasicAuth(h.config.Auth.Username, h.config.Auth.Password)
		}
	case "apikey":
		if h.config.Auth.Token != "" {
			req.Header.Set("X-API-Key", h.config.Auth.Token)
		}
	}
}
