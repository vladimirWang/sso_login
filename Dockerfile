# ---------- 构建阶段 ----------
FROM golang:1.27-alpine AS builder

WORKDIR /build

# 先拉依赖（利用层缓存）；GOPROXY 走国内代理，避免服务器上访问 proxy.golang.org 卡死
COPY go.mod go.sum ./
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download

# 编译 auth-rpc（gRPC，连 PostgreSQL/Redis/etcd）与 auth-api（HTTP，对外提供 SSO）
# -p 2 限制编译并行度，压低内存峰值，避免小内存服务器上构建时 OOM 夯死
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -p 2 -ldflags="-s -w" -o /out/auth-rpc  ./services/auth/rpc \
    && CGO_ENABLED=0 GOOS=linux go build -p 2 -ldflags="-s -w" -o /out/auth-api ./services/auth/api

# ---------- 运行阶段 ----------
FROM alpine:3.20

# ca-certificates：HTTPS 回调；tzdata：Asia/Shanghai 时区
RUN apk add --no-cache ca-certificates tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

WORKDIR /app
COPY --from=builder /out/auth-rpc  /app/bin/auth-rpc
COPY --from=builder /out/auth-api  /app/bin/auth-api
COPY services/auth/rpc/etc/auth.yaml      /app/etc/auth.yaml
COPY services/auth/api/etc/auth-api.yaml  /app/etc/auth-api.yaml
COPY deploy/docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# 8081 gRPC（auth-rpc），8888 HTTP（auth-api，nginx 反代目标）
EXPOSE 8081 8888

ENTRYPOINT ["/entrypoint.sh"]
