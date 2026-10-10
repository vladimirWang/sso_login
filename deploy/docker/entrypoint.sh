#!/bin/sh
# server 容器入口：同时运行 auth-rpc（:8081）与 auth-api（:8888）
set -e

# 数据库连接参数由 compose env_file（.env.dev / .env.prod）注入，
# yaml 通过 conf.UseEnv() 用 ${DB_*} 占位符读取。

/app/bin/auth-rpc -f /app/etc/auth.yaml &
RPC_PID=$!

/app/bin/auth-api -f /app/etc/auth-api.yaml &
API_PID=$!

# 收到停止信号时优雅关闭两个子进程
trap 'kill -TERM "$RPC_PID" "$API_PID" 2>/dev/null || true' TERM INT

# 任一进程退出即结束容器（restart: unless-stopped 负责整体重启）
wait -n
exit 1
