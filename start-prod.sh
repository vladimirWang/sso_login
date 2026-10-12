#!/bin/bash

docker compose -p sso-login-prod up -d
# 等待数据库初始化完成
sleep 10