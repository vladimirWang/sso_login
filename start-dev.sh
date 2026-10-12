#!/bin/bash

docker compose -p sso-login-test -f docker-compose.dev.yml up -d
# 等待数据库初始化完成
sleep 10