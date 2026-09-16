# API 设计

FlyIAM 提供 REST API，统一返回 JSON：

```json
{ "code": 0, "message": "success", "data": {} }
```

- `code=0` 表示成功；非 0 表示业务错误；
- 分页接口的 `data` 包含 `list` / `total` / `page` / `pageSize`；
- 需要认证的接口通过请求头 `Authorization: Bearer <token>` 传递本地 JWT；
- 权限校验格式为 `<资源>:<动作>`（如 `user:read`），超级管理员拥有全部权限。

## 1. 认证（Casdoor OAuth2）

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | `/api/auth/config` | 前端登录配置 | 公开 |
| GET | `/api/auth/login` | 跳转 Casdoor 登录（302） | 公开 |
| GET | `/api/auth/callback` | OAuth2 回调，签发本地 JWT 并跳转前端 | 公开 |
| GET | `/api/auth/userinfo` | 当前用户信息（含权限列表） | 登录 |
| POST | `/api/auth/change-password` | 修改本人密码 | 登录 |

## 2. 健康检查

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 健康检查 |

## 3. 用户管理（数据来源：Casdoor）

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | `/api/users` | 用户列表（服务端分页 + 搜索） | `user:read` |
| GET | `/api/user/detail` | 用户详情 | `user:read` |
| GET | `/api/user/stats` | 用户统计（总数/受保护列表） | `user:read` |
| POST | `/api/users` | 新增用户 | `user:write` |
| PUT | `/api/users/:name` | 更新用户 | `user:write` |
| POST | `/api/users/:name/reset-password` | 重置密码（留空用默认密码） | `user:write` |
| POST | `/api/users/:name/admin` | 设置管理员标记 | `user:write` |
| DELETE | `/api/users/:name` | 删除用户 | `user:delete` |
| POST | `/api/users/batch-delete` | 批量删除（跳过受保护用户） | `user:delete` |

**列表查询参数**

| 参数 | 说明 | 默认 |
|------|------|------|
| `field` | 搜索字段：`name` / `displayName` / `email` / `phone` | `name` |
| `keyword` | 关键词（模糊匹配） | - |
| `page` | 页码 | `1` |
| `pageSize` | 每页条数（1-200） | `20` |

## 4. 受保护用户（不可删除）

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | `/api/protected-users` | 列表 | `user:read` |
| POST | `/api/protected-users` | 新增 | `user:write` |
| DELETE | `/api/protected-users/:account` | 删除 | `user:write` |

## 5. 角色管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | `/api/roles` | 角色列表（含全部权限清单） | `role:read` |
| POST | `/api/roles` | 新建角色 | `role:write` |
| PUT | `/api/roles/:id` | 更新角色 | `role:write` |
| DELETE | `/api/roles/:id` | 删除角色 | `role:delete` |
| GET | `/api/permissions` | 系统权限清单 | `role:read` |

## 6. 团队管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | `/api/teams` | 团队列表（含成员数/角色数） | `team:read` |
| POST | `/api/teams` | 新建团队 | `team:write` |
| PUT | `/api/teams/:id` | 更新团队 | `team:write` |
| DELETE | `/api/teams/:id` | 删除团队 | `team:delete` |
| GET | `/api/teams/:id/members` | 成员列表 | `team:read` |
| POST | `/api/teams/:id/members` | 添加成员 | `team:write` |
| DELETE | `/api/teams/:id/members/:username` | 移除成员 | `team:write` |
| GET | `/api/teams/:id/roles` | 已授权角色 ID | `team:read` |
| POST | `/api/teams/:id/roles` | 设置团队角色 | `team:write` |

## 7. 数据源配置（页面可维护）

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | `/api/datasources` | 数据源列表 | `datasource:read` |
| POST | `/api/datasources` | 新增数据源 | `datasource:write` |
| PUT | `/api/datasources/:id` | 更新数据源 | `datasource:write` |
| DELETE | `/api/datasources/:id` | 删除数据源 | `datasource:delete` |
| POST | `/api/datasources/:id/test` | 测试已保存数据源 | `datasource:read` |
| POST | `/api/datasources/test` | 测试未保存配置 | `datasource:read` |

## 8. 定时任务

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | `/api/schedule` | 获取定时配置 | `schedule:read` |
| PUT | `/api/schedule` | 更新定时配置 | `schedule:write` |

## 9. 数据同步

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | `/api/sync/datasource` | 从数据源同步到 Casdoor | `sync:write` |
| POST | `/api/sync/full` | 完整同步（等同数据源同步） | `sync:write` |
| POST | `/api/sync/casdoor` | 兼容端点 | `sync:write` |
| GET | `/api/sync/logs` | 同步日志（分页 + 过滤） | `sync:read` |
| GET | `/api/sync/progress` | 各类型最近一次执行状态 | `sync:read` |

> 同步为异步执行：接口立即返回「任务已启动」，进度写入 `sync_logs`，
> 页面通过 `/api/sync/progress` 轮询展示。同一时刻仅允许一个同步任务（并发触发返回 `code=409`）。

## 10. Casdoor 应用 / 组织管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | `/api/casdoor/apps` | 应用列表 | `casdoor:read` |
| GET | `/api/casdoor/apps/:name` | 应用详情 | `casdoor:read` |
| POST | `/api/casdoor/apps` | 新建应用 | `casdoor:write` |
| PUT | `/api/casdoor/apps/:name` | 更新应用（含回调地址） | `casdoor:write` |
| DELETE | `/api/casdoor/apps/:name` | 删除应用 | `casdoor:write` |
| GET | `/api/casdoor/orgs` | 组织列表 | `casdoor:read` |
| POST | `/api/casdoor/orgs` | 新建组织 | `casdoor:write` |
| PUT | `/api/casdoor/orgs/:name` | 更新组织 | `casdoor:write` |
| DELETE | `/api/casdoor/orgs/:name` | 删除组织 | `casdoor:write` |

## 11. 对外集成 API（供其他系统调用）

FlyIAM 作为用户中心的对外能力，通过用户查询接口提供：

```bash
# 单个用户
GET /api/users?field=name&keyword=<域账号>

# 用户详情
GET /api/user/detail?domainAccount=<域账号>

# 用户统计
GET /api/user/stats
```

> 说明：外部系统（如 Consul Manager）可复用上述只读接口获取用户与组织信息；
> 生产环境建议为外部系统单独分配只读账号并按最小权限授权。

## 12. 错误码

| code | 含义 |
|------|------|
| 0 | 成功 |
| 401 | 未登录 / 登录已过期 |
| 403 | 无权限 |
| 409 | 冲突（如同一时刻重复触发同步） |
| 500 | 服务端错误 |
