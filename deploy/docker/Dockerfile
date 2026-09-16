#############################
#     设置公共的变量         #
#############################
ARG BASE_IMAGE_TAG=resolute
FROM ubuntu:${BASE_IMAGE_TAG}

# 作者描述信息
LABEL org.opencontainers.image.authors="iflyelf" \
      org.opencontainers.image.vendor="iflyelf" \
      org.opencontainers.image.title="FlyIAM" \
      org.opencontainers.image.description="Unified Identity and Access Management System" \
      org.opencontainers.image.source="https://github.com/iflyelf/flyiam" \
      org.opencontainers.image.url="https://github.com/iflyelf/flyiam" \
      org.opencontainers.image.documentation="https://github.com/iflyelf/flyiam/blob/main/README.md" \
      org.opencontainers.image.licenses="MIT"

ARG TARGETARCH
ARG TARGETVARIANT

# 时区设置
ARG TZ=Asia/Shanghai
ENV TZ=$TZ
# 语言设置
ARG LANG=zh_CN.UTF-8
ENV LANG=$LANG

# 镜像变量
ARG DOCKER_IMAGE=iflyelf/flyiam
ENV DOCKER_IMAGE=$DOCKER_IMAGE
ARG DOCKER_IMAGE_OS=ubuntu
ENV DOCKER_IMAGE_OS=$DOCKER_IMAGE_OS
ARG DOCKER_IMAGE_TAG=resolute
ENV DOCKER_IMAGE_TAG=$DOCKER_IMAGE_TAG

# 环境设置
ARG DEBIAN_FRONTEND=noninteractive
ENV DEBIAN_FRONTEND=$DEBIAN_FRONTEND

# GO环境变量
ARG GO_VERSION=1.27.1
ENV GO_VERSION=$GO_VERSION
ARG GOROOT=/opt/go
ENV GOROOT=$GOROOT
ARG GOPATH=/opt/golang
ENV GOPATH=$GOPATH
# Go 模块代理(加速依赖下载, 国内构建必备; 海外可改为 https://proxy.golang.org,direct)
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=$GOPROXY

# 版本号(由 CI 通过 --build-arg VERSION=<git tag> 注入, 缺省为 dev)
ARG VERSION=dev
ENV VERSION=$VERSION

# 构建时间和 Git Commit（由 CI 注入）
ARG BUILD_TIME=unknown
ENV BUILD_TIME=$BUILD_TIME
ARG GIT_COMMIT=unknown
ENV GIT_COMMIT=$GIT_COMMIT

ARG PKG_DEPS="\
    zsh \
    bash \
    bash-doc \
    bash-completion \
    conntrack \
    ipset \
    ipvsadm \
    bind9-dnsutils \
    iproute2 \
    net-tools \
    iptables \
    nftables \
    bridge-utils \
    openvswitch-switch \
    libseccomp2 \
    nfs-common \
    rsync \
    socat \
    psmisc \
    procps \
    sysstat \
    firewalld \
    chrony \
    ntpsec-ntpdate \
    tcpdump \
    telnet \
    lsof \
    iftop \
    htop \
    nmap \
    nmap-common \
    jq \
    curl \
    wget \
    axel \
    git \
    vim \
    tree \
    unzip \
    zip \
    tar \
    subversion \
    lrzsz \
    gcc \
    g++ \
    build-essential \
    binutils \
    autoconf \
    automake \
    libtool \
    gettext \
    autopoint \
    asciidoc \
    gawk \
    patch \
    flex \
    texinfo \
    device-tree-compiler \
    zlib1g-dev \
    libjpeg-dev \
    libelf-dev \
    libssl-dev \
    openssl \
    libffi-dev \
    libglib2.0-dev \
    xmlto \
    libncurses-dev \
    locate \
    lvm2 \
    rsyslog \
    ca-certificates \
    gnupg2 \
    debsums \
    locales \
    tzdata \
    fonts-droid-fallback \
    fonts-wqy-zenhei \
    fonts-wqy-microhei \
    fonts-arphic-ukai \
    fonts-arphic-uming \
    language-pack-zh-hans \
    numactl \
    xz-utils \
    libaio-dev \
    python3 \
    python3-dev \
    python3-pip \
    python3-yaml \
    python3-venv \
    python-is-python3 \
    supervisor \
    tini \
    sshpass \
    iputils-ping \
    ncat \
    upx-ucl \
    libxml2-dev \
    libxslt1-dev \
    cargo \
    rustc \
    sudo \
    npm \
    uglifyjs"
ENV PKG_DEPS=$PKG_DEPS

# ***** 安装依赖 *****
RUN set -eux && \
   # 更新源地址
   sed -i 's@URIs: http://[a-z.]*\.ubuntu\.com/ubuntu/@URIs: https://mirrors.aliyun.com/ubuntu/@g' /etc/apt/sources.list.d/ubuntu.sources && \
   sed -i 's@^Types: deb$@Types: deb deb-src@' /etc/apt/sources.list.d/ubuntu.sources && \
   # 解决证书认证失败问题
   touch /etc/apt/apt.conf.d/99verify-peer.conf && echo >>/etc/apt/apt.conf.d/99verify-peer.conf "Acquire { https::Verify-Peer false }" && \
   # 更新系统软件
   DEBIAN_FRONTEND=noninteractive apt update -qqy && apt upgrade -qqy && \
   # 安装依赖包
   DEBIAN_FRONTEND=noninteractive apt install -qqy $PKG_DEPS --option=Dpkg::Options::=--force-confdef && \
   # multilib/i386 交叉编译包仅 amd64 架构提供, 其他架构跳过
   if [ "${TARGETARCH}" = "amd64" ]; then \
       DEBIAN_FRONTEND=noninteractive apt install -qqy \
           gcc-multilib g++-multilib libc6-dev-i386 --option=Dpkg::Options::=--force-confdef ; \
   fi && \
   # 验证依赖包是否真正安装成功(逐个检查 dpkg 状态, 缺失则构建失败)
   for pkg in $PKG_DEPS; do \
       if ! dpkg-query -W -f='${Status}' "$pkg" 2>/dev/null | grep -q "install ok installed"; then \
           echo "ERROR: 依赖包未成功安装: $pkg" >&2 && exit 1; \
       fi; \
   done && \
   echo "所有依赖包验证通过" && \
   DEBIAN_FRONTEND=noninteractive apt -qqy autoremove --purge && \
   DEBIAN_FRONTEND=noninteractive apt -qqy autoclean && \
   rm -rf /var/lib/apt/lists/* && \
   # 更新时区
   ln -sf /usr/share/zoneinfo/${TZ} /etc/localtime && \
   # 更新时间
   echo ${TZ} > /etc/timezone && \
   # 更改为zsh
   sh -c "$(curl -fsSL https://raw.github.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" || true && \
   sed -i -e "s/bin\/ash/bin\/zsh/" /etc/passwd && \
   # vim 默认配置文件存在时才关闭 mouse(不同版本路径不同, 用 find 定位)
   find /usr/share/vim -name defaults.vim -exec sed -i -e 's/mouse=/mouse-=/g' {} + && \
   locale-gen zh_CN.UTF-8 && localedef -f UTF-8 -i zh_CN zh_CN.UTF-8 && locale-gen

# ***** 安装 Node.js 最新 LTS（每次构建时安装当前最新版本）*****
# 使用 n 在构建时获取最新 LTS；若需最新 Current 可改为 n latest
RUN set -eux && \
curl -fsSL https://deb.nodesource.com/setup_22.x | bash - && \
   DEBIAN_FRONTEND=noninteractive apt update -qqy && \
   DEBIAN_FRONTEND=noninteractive apt install -qqy nodejs && \
   npm config set registry https://registry.npmmirror.com && \
   npm install -g n && \
   n lts && \
   npm install -g wrangler && \
   rm -rf /var/lib/apt/lists/* /tmp/*

# ***** 安装 python3 版本 *****
RUN set -eux && \
    python3 -m pip config set global.break-system-packages true && \
    pip3 config set global.index-url http://mirrors.aliyun.com/pypi/simple/ && \
    pip3 config set install.trusted-host mirrors.aliyun.com && \
    python3 -m pip install --no-cache-dir --ignore-installed setuptools wheel cython && \
    python3 -m pip install --no-cache-dir pycryptodome lxml cython beautifulsoup4 requests && \
    rm -rf /tmp/* /var/lib/apt/lists/*

# ***** 安装golang *****
RUN set -eux && \
    # 映射 buildx TARGETARCH 到 Go 官方包名 (arm -> armv6l, 其他直接用)
    case "${TARGETARCH}" in \
        amd64)   GO_ARCH=amd64   ;; \
        arm64)   GO_ARCH=arm64   ;; \
        arm)     GO_ARCH=armv6l  ;; \
        386)     GO_ARCH=386     ;; \
        *)       echo "不支持的架构: ${TARGETARCH}" && exit 1 ;; \
    esac && \
    echo "目标架构: ${TARGETARCH} => Go 包: linux-${GO_ARCH}" && \
    wget --no-check-certificate https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz \
         -O /tmp/go-${GO_ARCH}.tar.gz && \
    tar xzf /tmp/go-${GO_ARCH}.tar.gz -C /opt && \
    mkdir -pv ${GOPATH}/bin && \
    # 仅删除 Go 压缩包, 不清空整个 /tmp (避免误删 DOWNLOAD_SRC=/tmp/src)
    rm -f /tmp/go-${GO_ARCH}.tar.gz && \
    # 软链 go 到 /usr/bin, 后续 RUN 层无需配 PATH
    ln -sf /opt/go/bin/* /usr/bin/ && \
    # 加载环境变量
    export GOROOT=/opt/go && \
    export GOPATH=/opt/golang && \
    export PATH=$PATH:$GOROOT/bin:$GOPATH/bin && \
    # 创建目录并清理文件
    mkdir -pv $GOPATH/bin && rm -rf /tmp/* /var/lib/apt/lists/* && \
    # 验证版本
    go version

# ***** 编译前端 *****
WORKDIR /src/web
COPY web/package*.json ./
RUN npm config set registry https://registry.npmmirror.com && \
    npm ci --production=false && \
    rm -rf /tmp/*

COPY web/ ./
RUN npm run build && \
    rm -rf /tmp/*

# ***** 编译 flyiam *****
# CGO_ENABLED=0 生成纯静态二进制; VERSION/BUILD_TIME/GIT_COMMIT 注入版本信息
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/opt/golang/pkg/mod \
    go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY --from=0 /src/web/dist ./web/dist

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/opt/golang/pkg/mod \
    set -eux && \
    CGO_ENABLED=0 go build -trimpath \
        -ldflags "-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT}" \
        -o /usr/local/bin/flyiam ./cmd/api && \
    /usr/local/bin/flyiam --version

# ***** 创建非 root 用户 *****
RUN groupadd -g 1000 flyiam && \
    useradd -u 1000 -g flyiam -s /bin/zsh -m flyiam && \
    mkdir -p /app/config /app/logs /app/web/dist && \
    cp -r /src/web/dist/* /app/web/dist/ && \
    chown -R flyiam:flyiam /app

# ***** 运行配置 *****
WORKDIR /app
USER flyiam

# 暴露端口
EXPOSE 8081

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

# 默认配置文件路径, 可通过挂载卷覆盖
ENTRYPOINT ["/usr/local/bin/flyiam"]
CMD ["-c", "/app/config/config.yaml"]
