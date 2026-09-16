.PHONY: help build build-backend-only run test clean install web-build docker-build all version

.DEFAULT_GOAL := help

# 变量定义
APP_NAME := flyiam
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0-dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

GO := go
BUILD_DIR := ./build
WEB_DIR := ./web

## help: 显示帮助信息
help:
	@echo "FlyIAM Makefile 命令列表:"
	@echo ""
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## install: 安装所有依赖
install:
	@echo "📦 安装 Go 依赖..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "📦 安装前端依赖..."
	cd $(WEB_DIR) && npm install
	@echo "✅ 依赖安装完成"

## web-build: 构建前端（嵌入到 Go）
web-build:
	@echo "🔨 构建前端..."
	cd $(WEB_DIR) && npm run build
	@echo "✅ 前端构建完成: internal/pkg/web/dist"

## build: 编译后端（包含前端）
build: web-build
	@echo "🔨 编译 $(APP_NAME) (包含前端)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) ./cmd/api
	@echo "✅ 编译完成: $(BUILD_DIR)/$(APP_NAME)"
	@echo "   包含前端静态文件"

## build-backend-only: 仅编译后端（不构建前端）
build-backend-only:
	@echo "🔨 编译 $(APP_NAME) (仅后端)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) ./cmd/api
	@echo "✅ 编译完成: $(BUILD_DIR)/$(APP_NAME)"

## run: 运行程序
run: build
	@echo "🚀 运行 $(APP_NAME)..."
	$(BUILD_DIR)/$(APP_NAME) -c etc/config.yaml

## dev: 开发模式（前端开发服务器）
dev:
	@echo "🔧 启动前端开发服务器..."
	cd $(WEB_DIR) && npm run dev

## test: 运行测试
test:
	@echo "🧪 运行测试..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	@echo "✅ 测试完成"

## clean: 清理构建产物
clean:
	@echo "🧹 清理构建产物..."
	rm -rf $(BUILD_DIR)
	rm -rf $(WEB_DIR)/dist
	rm -rf internal/pkg/web/dist
	rm -f coverage.out coverage.html
	@echo "✅ 清理完成"

## docker-build: 构建 Docker 镜像
docker-build:
	@echo "🐳 构建 Docker 镜像..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t $(APP_NAME):$(VERSION) \
		-t $(APP_NAME):latest \
		.
	@echo "✅ Docker 镜像构建完成"

## all: 完整构建（前端+后端）
all: clean install web-build build
	@echo "✅ 完整构建完成"

## version: 显示版本信息
version:
	@echo "版本: $(VERSION)"
	@echo "构建时间: $(BUILD_TIME)"
	@echo "Git 提交: $(GIT_COMMIT)"
