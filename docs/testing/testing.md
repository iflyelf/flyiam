# 测试文档

## 1. 后端测试

```bash
# 运行全部测试
make test

# 生成覆盖率报告
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## 2. 前端测试

```bash
cd web

# 单元测试（Vitest）
npm run test

# 覆盖率
npm run test:coverage
```

## 3. 接口冒烟测试

```bash
# 健康检查
curl -s http://localhost:8081/health

# 登录跳转（应 302 到 Casdoor）
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8081/api/auth/login

# 未登录访问受保护接口（应 401）
curl -s http://localhost:8081/api/users

# 带 Token 访问
TOKEN="<本地 JWT>"
curl -s -H "Authorization: Bearer $TOKEN" "http://localhost:8081/api/user/stats"
```

## 4. 端到端验证（浏览器）

覆盖关键路径：

1. Casdoor OAuth2 登录 → 跳转回控制台；
2. 仪表盘统计正确（用户数、受保护数、数据源数、团队数）；
3. 用户管理：列表加载、搜索、新增、编辑、重置密码、删除；
4. 组织架构：角色新建/授权、团队建/加成员/授权；
5. 数据同步：触发同步、进度实时刷新、日志记录；
6. Casdoor 管理：应用/组织/受保护用户配置。

## 5. 自动初始化验证

清空数据库后重启，验证：

```bash
# 1. 清空（谨慎，会丢失数据）
psql ... -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"

# 2. 重启 Casdoor（自动建 casdoor_* 表）
docker restart casdoor

# 3. 重启 FlyIAM（自动建业务表 + 初始化 Casdoor）
./build/flyiam -c etc/config-local.yaml

# 4. 预期日志
# ✅ 数据库初始化完成
# ✅ Casdoor 组织已创建: flyiam
# ✅ Casdoor 应用已创建: flyiam (clientId=...)
# ✅ Casdoor 管理员账号已创建: flyiam/admin
```

## 6. Helm Chart 校验

```bash
cd charts/flyiam
helmfile -e default lint       # 语法检查
helmfile -e default template   # 渲染清单（确认含 casdoor 相关资源）
```

## 7. 验收清单

- [ ] 空库启动自动建表成功
- [ ] Casdoor 组织/应用/管理员自动创建
- [ ] 登录、登出正常
- [ ] 用户 CRUD 与密码重置正常
- [ ] 受保护用户不可删除
- [ ] 数据源同步（含手机号更新、离职删除）正常
- [ ] 定时任务按间隔自动执行
- [ ] 权限校验生效（无权限返回 403）
- [ ] Helm Chart 渲染与部署正常
