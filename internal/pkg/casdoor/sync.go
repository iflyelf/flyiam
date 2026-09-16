package casdoor

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	casdoorsdk "github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

// SyncUser 同步用户到 Casdoor
type SyncUser struct {
	DomainAccount string
	Name          string
	Email         string
	Phone         string
	Password      string
	Affiliation   string            // 一级部门（用于 Casdoor 展示）
	Properties    map[string]string // 人事字段（工号/编制/部门等）
}

// SyncResult 同步结果
type SyncResult struct {
	Total        int
	Success      int
	Failed       int
	Created      int
	Updated      int
	Errors       []string
	SuccessNames map[string]struct{}
}

// ProgressFunc 同步进度回调（done 已完成数，total 总数）
type ProgressFunc func(done, total int)

// BatchSync 并发批量同步用户到 Casdoor
//
// existing 为当前组织已有用户（name -> user），用于增量判断，避免逐个查询。
// progress 可为 nil，用于周期性上报进度（每完成约 200 个回调一次）。
func (c *Client) BatchSync(ctx context.Context, users []SyncUser, existing map[string]*casdoorsdk.User, concurrency int, progress ProgressFunc) *SyncResult {
	if concurrency <= 0 {
		concurrency = 10
	}
	result := &SyncResult{
		Total:        len(users),
		Errors:       make([]string, 0, 16),
		SuccessNames: make(map[string]struct{}, len(users)),
	}

	var mu sync.Mutex
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	const progressStep = 200

	log.Printf("🔄 开始同步 %d 个用户到 Casdoor（并发 %d）", len(users), concurrency)

	for _, u := range users {
		select {
		case <-ctx.Done():
			wg.Wait()
			return result
		default:
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(u SyncUser) {
			defer wg.Done()
			defer func() { <-sem }()

			ex := existing[u.DomainAccount]
			isNew := ex == nil

			var err error
			if isNew {
				err = c.createNewUser(u)
			} else {
				err = c.updateExistingUser(ex, u)
			}

			// 手机号非法或重复时，清空手机号兜底重试，保证用户仍能进入 Casdoor
			if err != nil && strings.Contains(err.Error(), "Phone") && u.Phone != "" {
				u.Phone = ""
				if u.Properties == nil {
					u.Properties = map[string]string{}
				}
				u.Properties["phoneInvalid"] = "true"
				if isNew {
					err = c.createNewUser(u)
				} else {
					err = c.updateExistingUser(ex, u)
				}
			}

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				result.Failed++
				if len(result.Errors) < 50 {
					result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", u.DomainAccount, err))
				}
			} else {
				result.Success++
				if isNew {
					result.Created++
				} else {
					result.Updated++
				}
				result.SuccessNames[u.DomainAccount] = struct{}{}
			}

			done := result.Success + result.Failed
			if progress != nil && (done%progressStep == 0 || done == result.Total) {
				progress(done, result.Total)
			}
		}(u)
	}

	wg.Wait()
	log.Printf("✅ Casdoor 同步完成: 成功 %d/%d（新增 %d，更新 %d），失败 %d",
		result.Success, result.Total, result.Created, result.Updated, result.Failed)
	return result
}

// createNewUser 创建新用户
func (c *Client) createNewUser(user SyncUser) error {
	props := user.Properties
	if props == nil {
		props = map[string]string{}
	}
	password := user.Password
	if password == "" {
		password = c.config.DefaultPassword
	}
	casdoorUser := &casdoorsdk.User{
		Owner:             c.organization,
		Name:              user.DomainAccount,
		Type:              "normal-user",
		Password:          password,
		DisplayName:       user.Name,
		Email:             user.Email,
		Phone:             user.Phone,
		CountryCode:       c.config.CountryCode,
		Affiliation:       user.Affiliation,
		Tag:               "synced",
		Language:          "zh",
		Address:           []string{},
		Properties:        props,
		SignupApplication: c.application,
	}
	_, err := c.AddUser(casdoorUser)
	return err
}

// updateExistingUser 更新已存在的用户
func (c *Client) updateExistingUser(existing *casdoorsdk.User, newData SyncUser) error {
	existing.DisplayName = newData.Name
	existing.Email = newData.Email
	existing.Phone = newData.Phone
	existing.CountryCode = c.config.CountryCode
	existing.Affiliation = newData.Affiliation
	existing.Tag = "synced"
	if newData.Properties != nil {
		if existing.Properties == nil {
			existing.Properties = map[string]string{}
		}
		for k, v := range newData.Properties {
			existing.Properties[k] = v
		}
	}
	_, err := c.UpdateUser(existing)
	return err
}

// SyncOrganization 同步组织到 Casdoor
type SyncOrganization struct {
	Name        string
	DisplayName string
	ParentName  string
}

// SyncOrganizationsToCAsdoor 批量同步组织到 Casdoor
func (c *Client) SyncOrganizationsToCAsdoor(ctx context.Context, orgs []SyncOrganization) (*SyncResult, error) {
	result := &SyncResult{
		Total:  len(orgs),
		Errors: make([]string, 0),
	}

	log.Printf("🔄 开始同步 %d 个组织到 Casdoor", len(orgs))

	for _, org := range orgs {
		// 检查 context 是否取消
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		// 检查组织是否存在
		existingOrg, err := c.GetOrganization(org.Name)
		if err == nil && existingOrg != nil {
			// 组织已存在，更新
			existingOrg.DisplayName = org.DisplayName
			_, err := c.UpdateOrganization(existingOrg)
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("更新组织 %s 失败: %v", org.Name, err))
				log.Printf("❌ 更新组织失败: %s - %v", org.Name, err)
				continue
			}
			result.Success++
			log.Printf("✅ 组织已更新: %s", org.Name)
		} else {
			// 组织不存在，创建
			casdoorOrg := &casdoorsdk.Organization{
				Owner:         "admin",
				Name:          org.Name,
				CreatedTime:   "",
				DisplayName:   org.DisplayName,
				WebsiteUrl:    "",
				Favicon:       "",
				PasswordType:  "plain",
				PasswordSalt:  "",
				DefaultAvatar: "",
				Tags:          []string{"synced"},
			}

			_, err := c.AddOrganization(casdoorOrg)
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("创建组织 %s 失败: %v", org.Name, err))
				log.Printf("❌ 创建组织失败: %s - %v", org.Name, err)
				continue
			}
			result.Success++
			log.Printf("✅ 组织已创建: %s", org.Name)
		}
	}

	log.Printf("✅ 同步完成: 成功 %d/%d, 失败 %d", result.Success, result.Total, result.Failed)
	return result, nil
}

// ResetUserPassword 重置用户密码
func (c *Client) ResetUserPassword(username string, newPassword string) error {
	user, err := c.GetUser(username)
	if err != nil {
		return fmt.Errorf("获取用户失败: %w", err)
	}

	user.Password = newPassword
	_, err = c.UpdateUser(user)
	if err != nil {
		return fmt.Errorf("重置密码失败: %w", err)
	}

	return nil
}

// DisableUser 禁用用户
func (c *Client) DisableUser(username string) error {
	user, err := c.GetUser(username)
	if err != nil {
		return fmt.Errorf("获取用户失败: %w", err)
	}

	user.IsForbidden = true
	_, err = c.UpdateUser(user)
	if err != nil {
		return fmt.Errorf("禁用用户失败: %w", err)
	}

	return nil
}

// EnableUser 启用用户
func (c *Client) EnableUser(username string) error {
	user, err := c.GetUser(username)
	if err != nil {
		return fmt.Errorf("获取用户失败: %w", err)
	}

	user.IsForbidden = false
	_, err = c.UpdateUser(user)
	if err != nil {
		return fmt.Errorf("启用用户失败: %w", err)
	}

	return nil
}
