#!/bin/bash
# Mac 本地执行：交叉构建 linux/amd64 镜像并打出服务器部署压缩包。
# 核心策略：编译全部在 Mac 完成，服务器只 docker load + up，
# 避免小内存服务器（1.6G）上 go build OOM 夯死整机。
#
# 用法:
#   ./build.sh                      # 默认构建 linux/amd64（阿里云 x86 服务器）
#   TARGET_PLATFORM=linux/arm64 ./build.sh   # ARM 服务器
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

PROJECT_NAME="sso-login"
IMAGE_NAME="sso-login-server"
PLATFORM="${TARGET_PLATFORM:-linux/amd64}"
BUILDER="sso-deploy-builder"
OUTPUT_DIR="$SCRIPT_DIR/dist"

echo "==> [1/4] 准备 buildx builder（$BUILDER, docker-container 驱动，支持跨平台导出）..."
if ! docker buildx inspect "$BUILDER" >/dev/null 2>&1; then
  docker buildx create --name "$BUILDER" --driver docker-container --bootstrap
fi

echo "==> [2/4] 交叉构建 $PLATFORM 镜像（首次较慢：QEMU 模拟编译两个 Go 服务）..."
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"
# type=docker 直接导出为 docker save 格式的 tar，不加载进本地镜像仓库，不污染 dev 镜像
docker buildx build --builder "$BUILDER" \
  --platform "$PLATFORM" \
  -t "$IMAGE_NAME:latest" \
  -o "type=docker,dest=$OUTPUT_DIR/$IMAGE_NAME.tar" \
  .
gzip "$OUTPUT_DIR/$IMAGE_NAME.tar"

echo "==> [3/4] 组装部署文件..."
cp docker-compose.yml "$OUTPUT_DIR/"
cp .env.prod        "$OUTPUT_DIR/"
cp up.sh            "$OUTPUT_DIR/"
# nginx 生产配置与证书（compose 按相对路径 bind mount，需保持目录结构）
mkdir -p "$OUTPUT_DIR/deploy/docker/nginx/ssl"
cp deploy/docker/nginx/prod.conf "$OUTPUT_DIR/deploy/docker/nginx/"
cp deploy/docker/nginx/ssl/*.pem \
   deploy/docker/nginx/ssl/*.key "$OUTPUT_DIR/deploy/docker/nginx/ssl/"
chmod +x "$OUTPUT_DIR/up.sh"

echo "==> [4/4] 打部署压缩包..."
PACKAGE="$SCRIPT_DIR/${PROJECT_NAME}-deploy-$(date +%Y%m%d-%H%M%S).tar.gz"
tar -czf "$PACKAGE" -C "$OUTPUT_DIR" .
rm -rf "$OUTPUT_DIR"

echo ""
echo "==> 完成!"
echo "    部署压缩包: $PACKAGE"
ls -lh "$PACKAGE"
echo ""
echo "下一步 (部署到阿里云):"
echo "  1. scp $(basename "$PACKAGE") <user>@<aliyun-ip>:/path/to/deploy/"
echo "  2. ssh <user>@<aliyun-ip>"
echo "  3. mkdir -p sso-login && tar -xzf $(basename "$PACKAGE") -C sso-login && cd sso-login"
echo "  4. ./up.sh   # 加载镜像 + 启动，不在服务器构建"
