# OpenEAAP Dockerfile - 多阶段构建
# Stage 1: 构建阶段
FROM golang:1.24-alpine AS builder

# 设置工作目录
WORKDIR /build

# 安装构建依赖
RUN apk add --no-cache git make gcc musl-dev

# 复制 Go 模块文件
COPY go.mod go.sum ./

# 下载依赖（利用 Docker 缓存层）
RUN go mod download && go mod verify

# 复制源代码
COPY . .

# 编译应用
# CGO_ENABLED=0: 静态编译，不依赖 C 库
# GOOS=linux: 目标操作系统
# -ldflags: 链接参数，去除调试信息，减小二进制大小
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
   -ldflags="-w -s -X main.Version=$(git describe --tags --always) -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
   -o /build/openeeap-server \
   ./cmd/server/main.go

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
   -ldflags="-w -s" \
   -o /build/openeeap-worker \
   ./cmd/worker/main.go

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
   -ldflags="-w -s" \
   -o /build/openeeap-cli \
   ./cmd/cli/main.go

# Stage 2: 运行时镜像
FROM alpine:3.19

# 安装运行时依赖
RUN apk --no-cache add \
   ca-certificates \
   tzdata \
   curl \
   && rm -rf /var/cache/apk/*

# 设置时区为 UTC
ENV TZ=UTC

# 创建非 root 用户
RUN addgroup -g 1000 openeeap && \
   adduser -D -u 1000 -G openeeap openeeap

# 创建必要的目录
RUN mkdir -p /app/configs /app/logs /app/data && \
   chown -R openeeap:openeeap /app

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/openeeap-server /app/
COPY --from=builder /build/openeeap-worker /app/
COPY --from=builder /build/openeeap-cli /app/

# 复制配置文件
COPY --chown=openeeap:openeeap configs/ /app/configs/

# 复制数据库迁移文件
COPY --chown=openeeap:openeeap migrations/ /app/migrations/

# 切换到非 root 用户
USER openeeap

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
   CMD curl -f http://localhost:8080/health || exit 1

# 暴露端口
# 8080: HTTP API
# 9090: gRPC API
# 9091: Prometheus Metrics
EXPOSE 8080 9090 9091

# 定义启动命令（默认启动 server）
# 可通过 docker run 覆盖启动其他服务
CMD ["/app/openeeap-server", "--config", "/app/configs/production.yaml"]

# 元数据标签
LABEL maintainer="OpenEAAP Team <team@openeeap.io>" \
     description="OpenEAAP - Enterprise AI Agent Platform" \
     version="1.0.0" \
     org.opencontainers.image.source="https://github.com/openeeap/openeeap" \
     org.opencontainers.image.documentation="https://docs.openeeap.io"

# Personal.AI order the ending
