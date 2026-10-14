#!/bin/bash
# Mac 本地执行：原生交叉编译 linux/amd64 二进制 → 打轻量镜像 → 生成服务器部署包。
#
# 核心策略（避免在任何小内存机器上编译）：
#   1. 用 Mac 原生 Go 工具链直接交叉编译（GOOS=linux GOARCH=amd64，CGO_ENABLED=0 纯静态），
#      不经过 QEMU，速度快、内存占用低；
#   2. 镜像只做 alpine + apk add(ca 证书/时区) + COPY 二进制，
#      QEMU 仅执行这条很轻的 apk 命令，不再编译庞大的 Go 依赖树；
#   3. 服务器只 docker load + up。
#
# 用法:
#   ./build.sh                          # 默认 amd64（阿里云 x86 服务器）
#   TARGET_GOARCH=arm64 ./build.sh      # ARM 服务器
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

PROJECT_NAME="sso-login"
IMAGE_NAME="sso-login-server"
GOARCH="${TARGET_GOARCH:-amd64}"
BUILD_DIR="$SCRIPT_DIR/build-out"
STAGE_DIR="$SCRIPT_DIR/.deploy-stage"
OUTPUT_DIR="$SCRIPT_DIR/dist"

echo "==> [1/6] 构建前端 SPA（frontend/，Vite + React）..."
( cd frontend && pnpm install --frozen-lockfile=false && pnpm build )

echo "==> [2/6] Mac 原生交叉编译 linux/$GOARCH（不经过 QEMU）..."
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"
CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" \
  go build -ldflags="-s -w" -o "$BUILD_DIR/auth-rpc"  ./services/auth/rpc
CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" \
  go build -ldflags="-s -w" -o "$BUILD_DIR/auth-api"  ./services/auth/api
echo "    产物架构确认:"
file "$BUILD_DIR/auth-rpc" "$BUILD_DIR/auth-api" | sed 's/^/      /'

echo "==> [3/6] 组装镜像构建上下文（$STAGE_DIR）..."
rm -rf "$STAGE_DIR"
mkdir -p "$STAGE_DIR"
cp Dockerfile.deploy                          "$STAGE_DIR/Dockerfile"
cp "$BUILD_DIR/auth-rpc" "$BUILD_DIR/auth-api" "$STAGE_DIR/"
cp services/auth/rpc/etc/auth.yaml            "$STAGE_DIR/"
cp services/auth/api/etc/auth-api.yaml        "$STAGE_DIR/"
cp deploy/docker/entrypoint.sh                "$STAGE_DIR/"

echo "==> [4/6] 构建 linux/$GOARCH 镜像（QEMU 仅执行轻量 apk add）..."
docker build --platform "linux/$GOARCH" -t "$IMAGE_NAME:latest" "$STAGE_DIR"
echo "    镜像架构确认: $(docker image inspect "$IMAGE_NAME:latest" --format '{{.Architecture}}')"

echo "==> [5/6] 导出镜像并组装部署文件..."
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"
docker save "$IMAGE_NAME:latest" | gzip > "$OUTPUT_DIR/$IMAGE_NAME.tar.gz"
cp docker-compose.yml "$OUTPUT_DIR/"
cp .env.prod          "$OUTPUT_DIR/"
cp up.sh              "$OUTPUT_DIR/"
# nginx 生产配置与证书（compose 按相对路径 bind mount，需保持目录结构）
mkdir -p "$OUTPUT_DIR/deploy/docker/nginx/ssl"
cp deploy/docker/nginx/prod.conf "$OUTPUT_DIR/deploy/docker/nginx/"
cp deploy/docker/nginx/ssl/*.pem \
   deploy/docker/nginx/ssl/*.key "$OUTPUT_DIR/deploy/docker/nginx/ssl/"
# 前端 SPA 构建产物（compose 挂载 frontend/dist 到 nginx html 目录）
mkdir -p "$OUTPUT_DIR/frontend"
cp -R frontend/dist "$OUTPUT_DIR/frontend/dist"
# 数据库初始化脚本（服务器首次部署执行，幂等）
mkdir -p "$OUTPUT_DIR/deploy/sql"
cp deploy/sql/*.sql "$OUTPUT_DIR/deploy/sql/"
chmod +x "$OUTPUT_DIR/up.sh"

echo "==> [6/6] 打部署压缩包..."
PACKAGE="$SCRIPT_DIR/${PROJECT_NAME}-deploy-$(date +%Y%m%d-%H%M%S).tar.gz"
# COPYFILE_DISABLE=1：阻止 macOS bsdtar 把扩展属性打成 ._* AppleDouble 垃圾文件带进部署包
COPYFILE_DISABLE=1 tar -czf "$PACKAGE" -C "$OUTPUT_DIR" .
rm -rf "$OUTPUT_DIR" "$STAGE_DIR"

ip="139.224.68.145"
echo ""
echo "==> 完成!"
echo "    部署压缩包: $PACKAGE"
ls -lh "$PACKAGE"
echo ""
echo "下一步 (逐行复制粘贴执行)："
echo "  scp $(basename "$PACKAGE") root@$ip:~"
echo "  ssh root@$ip"
echo "  mkdir -p sso-login && tar -xzf $(basename "$PACKAGE") -C sso-login && cd sso-login && ./up.sh"
