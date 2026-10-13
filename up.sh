#!/bin/bash
# 服务器上执行：加载 Mac 端预构建的镜像并启动生产服务。
# 本机不构建（服务器仅 1.6G 内存，go build 会 OOM 夯死整机）。
#
# 用法: ./up.sh
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

PROJECT_NAME="sso-login-prod"
COMPOSE_FILE="docker-compose.yml"
IMAGE_TAR="sso-login-server.tar.gz"

if [ ! -f "$COMPOSE_FILE" ]; then
  echo "错误: $COMPOSE_FILE 不存在，请确认部署包完整"; exit 1
fi
if [ ! -f "$IMAGE_TAR" ]; then
  echo "错误: $IMAGE_TAR 不存在，请确认部署包完整"; exit 1
fi

echo "==> [1/3] 加载预构建镜像..."
docker load -i "$IMAGE_TAR"

echo "==> [2/3] 启动服务..."
docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" up -d

echo ""
echo "==> [3/3] 服务状态:"
docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" ps
