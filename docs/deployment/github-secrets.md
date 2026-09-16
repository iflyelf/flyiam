# 部署 - GitHub Actions 配置

本项目通过 GitHub Actions 自动构建并推送容器镜像、发布多平台二进制。

## 1. 工作流

| 文件 | 说明 |
|------|------|
| `.github/workflows/publish.yml` | 主流程：构建前端 → 交叉编译多平台二进制 → 发布 latest Release → 构建并推送多架构镜像（amd64/arm64） |
| `.github/workflows/helm-lint.yml` | Helm Chart 语法校验：`helmfile lint` + 各环境渲染 + YAML 解析校验 |

## 2. 必需的 Secrets

在仓库 **Settings → Secrets and variables → Actions** 中配置：

| Secret | 说明 |
|--------|------|
| `DOCKER_USERNAME` | Docker Hub 用户名（如 `iflyelf`） |
| `DOCKER_PASSWORD` | Docker Hub 密码或访问令牌 |

`GITHUB_TOKEN` 由 Actions 自动提供，无需配置（用于发布 Release）。

## 3. 触发方式

- **推送**：推送到 `main` 且修改了 `Dockerfile`、`go.mod`、`go.sum`、`cmd/**`、`internal/**`、`web/**`；
- **手动**：Actions 页选择工作流 → `Run workflow`；
- **Star**：点 Star 触发（`watch: started`）。

## 4. 产物

- 容器镜像：`<DOCKER_USERNAME>/flyiam:latest`、`<DOCKER_USERNAME>/flyiam:latest-<短提交>`（amd64/arm64）
- GitHub Release（`latest` 标签）二进制：
  - `flyiam-linux-amd64.tar.gz` / `flyiam-linux-arm64.tar.gz`
  - `flyiam-darwin-amd64.tar.gz` / `flyiam-darwin-arm64.tar.gz`
  - `flyiam-windows-amd64.zip`
  - `checksums.txt`（SHA256 校验）

## 5. 镜像架构

使用 `docker/setup-qemu-action` + `docker/setup-buildx-action` 构建
`linux/amd64` 与 `linux/arm64` 双架构镜像，构建缓存使用 GitHub Actions Cache。

## 6. 关键技术点

- **前端先行**：二进制任务先构建前端（`web/dist`），再 `CGO_ENABLED=0` 交叉编译，
  确保 Go Embed 的前端资源为最新；
- **纯静态二进制**：`CGO_ENABLED=0` + `-trimpath`，跨平台可执行；
- **版本注入**：`-X main.Version` / `-X main.BuildTime` / `-X main.GitCommit`；
- **构建缓存**：Go modules 与 Docker 层均启用缓存，加速构建。

## 7. 本地复现构建

```bash
# 多架构构建（需 buildx）
docker buildx build --platform linux/amd64,linux/arm64 -t iflyelf/flyiam:latest .

# 交叉编译二进制（CGO_ENABLED=0，纯静态）
cd web && npm ci && npm run build && cd ..
CGO_ENABLED=0 GOOS=linux  GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/flyiam-linux-amd64  ./cmd/api
CGO_ENABLED=0 GOOS=linux  GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o dist/flyiam-linux-arm64  ./cmd/api
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/flyiam-darwin-amd64 ./cmd/api
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o dist/flyiam-darwin-arm64 ./cmd/api
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/flyiam-windows-amd64.exe ./cmd/api

# Helm Chart 校验
cd charts/flyiam && helmfile -e default lint
```

## 8. 版本基线

| 组件 | 版本 |
|------|------|
| Go | 1.27.1（Dockerfile）/ `>=1.26`（CI） |
| Node.js | 22（CI）/ 22 LTS（Dockerfile） |
| Helm | 3.16.2 |
| Helmfile | 0.171.0 |
