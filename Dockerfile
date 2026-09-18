#############################
#  FlyIAM 多阶段构建          #
#  builder: iflyelf/ubuntu:latest（含 Go/Node/工具链）#
#  runtime: iflyelf/ubuntu:lite（精简体）             #
#############################

# 构建基础镜像（含 Go / Node / Python / 编译工具链，仅更新依赖即可）
ARG BUILDER_IMAGE=iflyelf/ubuntu:latest
# 运行基础镜像（精简，仅含运行时所需基础包）
ARG RUNTIME_IMAGE=iflyelf/ubuntu:lite

# =============================================================================
# 阶段一：构建（编译前端 + 交叉编译 Go 二进制）
# =============================================================================
FROM ${BUILDER_IMAGE} AS builder

ARG TARGETARCH
ARG TARGETVARIANT

# 版本信息（由 CI 通过 --build-arg 注入）
ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG GIT_COMMIT=unknown

# Go 模块代理（构建基础镜像已内置，这里允许覆盖）
ARG GOPROXY=https://goproxy.cn,direct

# 仅更新依赖包到最新（基础镜像已含全部工具链，无需重装 PKG_DEPS）
RUN set -eux && \
    DEBIAN_FRONTEND=noninteractive apt-get update -qqy && \
    DEBIAN_FRONTEND=noninteractive apt-get upgrade -qqy --option=Dpkg::Options::=--force-confdef && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /src

# 先复制依赖清单，利用层缓存加速 go mod download
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/opt/golang/pkg/mod \
    go mod download

# 复制源码
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY web/ ./web/

# 编译前端（vite outDir = ../internal/pkg/web/dist，供 Go embed 使用）
WORKDIR /src/web
RUN set -eux && \
    npm config set registry https://registry.npmmirror.com && \
    npm ci --production=false && \
    npm run build && \
    rm -rf /tmp/*

# 交叉编译 flyiam（CGO_ENABLED=0 纯静态二进制；注入版本信息）
WORKDIR /src
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/opt/golang/pkg/mod \
    set -eux && \
    CGO_ENABLED=0 go build -trimpath \
        -ldflags "-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}" \
        -o /out/flyiam ./cmd/api && \
    /out/flyiam --version

# =============================================================================
# 阶段二：运行（仅拷贝构建产物到精简镜像）
# =============================================================================
FROM ${RUNTIME_IMAGE} AS runtime

LABEL org.opencontainers.image.authors="iflyelf" \
      org.opencontainers.image.vendor="iflyelf" \
      org.opencontainers.image.title="FlyIAM" \
      org.opencontainers.image.description="Unified Identity and Access Management System" \
      org.opencontainers.image.source="https://github.com/iflyelf/flyiam" \
      org.opencontainers.image.url="https://github.com/iflyelf/flyiam" \
      org.opencontainers.image.documentation="https://github.com/iflyelf/flyiam/blob/main/README.md" \
      org.opencontainers.image.licenses="MIT"

ARG TZ=Asia/Shanghai
ENV TZ=$TZ
ARG LANG=zh_CN.UTF-8
ENV LANG=$LANG

# 复制编译产物
COPY --from=builder /out/flyiam /usr/local/bin/flyiam

# 创建非 root 用户（ubuntu 基础镜像自带 UID/GID 1000 的 ubuntu 用户，先移除以复用 1000）
RUN set -eux && \
    userdel -rf ubuntu 2>/dev/null || true && \
    groupdel ubuntu 2>/dev/null || true && \
    groupadd -g 1000 flyiam && \
    useradd -u 1000 -g flyiam -s /bin/zsh -m flyiam && \
    mkdir -p /app/config /app/logs && \
    chown -R flyiam:flyiam /app

# 内置默认配置（仓库 etc/config.yaml 已脱敏；运行时可用环境变量覆盖或挂载卷替换）
COPY --chown=flyiam:flyiam etc/config.yaml /app/config/config.yaml

WORKDIR /app
USER flyiam

EXPOSE 8081

# 健康检查（runtime 基础镜像已含 wget）
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

# 默认配置文件路径，可通过挂载卷覆盖
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/flyiam"]
CMD ["-c", "/app/config/config.yaml"]
