# 部署 - Docker

使用 `docker-compose.yml` 一键启动 PostgreSQL、Redis、Casdoor 与 FlyIAM。

## 1. 目录说明

```
docker-compose.yml               # 编排文件（含内置 Casdoor）
Dockerfile                       # 多阶段构建（前端 + 后端）
deploy/docker/Dockerfile         # 同上（备用路径）
deploy/sql/schema.sql            # 数据库 DDL（参考）
```

## 2. 启动

```bash
# 编辑 .env（可选，覆盖默认配置）
cp .env.example .env

docker compose up -d

# 查看日志
docker compose logs -f flyiam
```

## 3. 服务与端口

| 服务 | 端口 | 说明 |
|------|------|------|
| flyiam | 8081 | 应用（含前端） |
| casdoor | 8000 | 认证中心 |
| postgres | 5432 | 数据库 |
| redis | 6379 | 缓存 |

## 4. 首次启动

无需手工执行 SQL：Casdoor 与 FlyIAM 会自动建库表、创建组织/应用/管理员。

- **应用（FlyIAM 控制台）**：http://localhost:8081
- **认证中心（Casdoor）**：http://localhost:8000
- **默认管理员**：`admin` / `ysyh!9Sky`（见 `docker-compose.yml` 中的 `ADMIN_PASSWORD`）

## 5. 环境变量

参见 `docker-compose.yml` 中 `flyiam` 与 `casdoor` 服务的 `environment`，关键项：

```yaml
# FlyIAM
DB_HOST: postgres
DB_PASSWORD: flyiam_password
REDIS_HOST: redis
JWT_SECRET: change-me-at-least-32-chars
ADMIN_PASSWORD: ysyh!9Sky
CASDOOR_ENDPOINT: http://casdoor:8000
CASDOOR_AUTO_SETUP: "true"

# Casdoor
TZ: Asia/Shanghai
RUNNING_IN_DOCKER: "true"
```

## 6. 数据持久化

```yaml
volumes:
  postgres_data:
  redis_data:
  casdoor_data:
```

## 7. 停止与清理

```bash
docker compose down            # 停止（保留数据）
docker compose down -v         # 停止并删除数据卷
```

## 8. 构建自定义镜像

```bash
# 多架构构建（amd64 / arm64）
docker buildx build --platform linux/amd64,linux/arm64 \
  -t iflyelf/flyiam:latest --push .
```
