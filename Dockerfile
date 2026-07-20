# syntax=docker/dockerfile:1.4
# 构建期可覆盖（国内网络建议保留默认镜像）
ARG GOPROXY=https://goproxy.cn,direct
ARG GOSUMDB=sum.golang.google.cn
ARG NPM_REGISTRY=https://registry.npmmirror.com
ARG APT_MIRROR=https://mirrors.tuna.tsinghua.edu.cn/debian/

# --- Frontend build ---
FROM node:20-alpine AS web-builder
ARG NPM_REGISTRY
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm config set registry "${NPM_REGISTRY}" \
    && npm ci
COPY web/ .
ENV VITE_API_BASE_URL=/api
RUN npm run build

# --- Backend build（CGO + Rust audio-codec：audio codecs for WebRTC/voice）---
FROM golang:1.26 AS go-builder
ARG GOPROXY
ARG GOSUMDB
ARG APT_MIRROR
COPY deploy/apt-packages /tmp/apt-packages
RUN echo "deb ${APT_MIRROR} bookworm main contrib non-free non-free-firmware" > /etc/apt/sources.list \
    && echo "deb ${APT_MIRROR} bookworm-updates main contrib non-free non-free-firmware" >> /etc/apt/sources.list \
    && echo "deb ${APT_MIRROR} bookworm-security main contrib non-free non-free-firmware" >> /etc/apt/sources.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
       $(awk '/^# BUILD/{f=1;next}/^# RUNTIME/{f=0} f && !/^#/ && NF' /tmp/apt-packages) \
       curl \
    && rm -rf /var/lib/apt/lists/*
# Rust toolchain for lingllm/media/encoder/lecodec
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain stable \
    && . "$HOME/.cargo/env" && rustc --version
ENV PATH="/root/.cargo/bin:${PATH}"
WORKDIR /src
ENV GOTOOLCHAIN=auto \
    CGO_ENABLED=1 \
    GOPROXY=${GOPROXY} \
    GOSUMDB=${GOSUMDB}

# 先拷贝 go.mod/go.sum，依赖不变则复用本层 + BuildKit 模块缓存
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# 仅拷贝 Go 源码目录，避免 web/docs 等变更触发全量重编译
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY lingllm/ ./lingllm/
COPY embed.go ./
COPY templates/ ./templates/
COPY data/ ./data/
COPY banner.txt ./

# Build Rust liblecodec (static + shared)
RUN --mount=type=cache,target=/root/.cargo/registry \
    --mount=type=cache,target=/src/lingllm/media/encoder/lecodec/target \
    ./lingllm/media/encoder/lecodec/build.sh

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -tags lingcodec -ldflags="-s -w" -o /out/server ./cmd/server

# --- Runtime ---
FROM debian:bookworm-slim AS runtime
ARG APT_MIRROR
COPY deploy/apt-packages /tmp/apt-packages
RUN echo "deb ${APT_MIRROR} bookworm main contrib non-free non-free-firmware" > /etc/apt/sources.list \
    && echo "deb ${APT_MIRROR} bookworm-updates main contrib non-free non-free-firmware" >> /etc/apt/sources.list \
    && echo "deb ${APT_MIRROR} bookworm-security main contrib non-free non-free-firmware" >> /etc/apt/sources.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
       $(awk '/^# RUNTIME/{f=1;next} f && !/^#/ && NF' /tmp/apt-packages) \
    && rm -rf /var/lib/apt/lists/* /etc/nginx/sites-enabled/default

COPY deploy/nginx.conf /etc/nginx/conf.d/default.conf.template
COPY deploy/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
COPY --from=go-builder /out/server /app/server
COPY --from=go-builder /go/pkg/mod/github.com/xluohome/phonedata@v0.0.0-20231114052328-a9241291f8b1/phone.dat /app/data/phone.dat
COPY --from=web-builder /src/web/dist /usr/share/nginx/html

RUN chmod +x /usr/local/bin/docker-entrypoint.sh \
  && mkdir -p /data/uploads /app/data

WORKDIR /app

ENV MODE=production \
    GIN_MODE=release \
    LINGECHO_DATA_DIR=/data \
    LINGECHO_SEED=auto \
    LINGECHO_STARTUP_TIMEOUT=600 \
    METRICS_ALLOWED_IPS=127.0.0.1,::1

EXPOSE 80

VOLUME ["/data"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=40s --retries=3 \
  CMD wget -qO- http://127.0.0.1/healthz >/dev/null || exit 1

ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["/usr/local/bin/docker-entrypoint.sh"]
