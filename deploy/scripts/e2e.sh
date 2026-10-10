#!/usr/bin/env bash
# SSO 登录中心端到端冒烟测试
# 前置：etcd:2379 / PostgreSQL:15432 / Redis:16379 / RabbitMQ:5672 可用，
#       auth-rpc(8081) 与 auth-api(8888) 已启动
set -euo pipefail

BASE=${BASE:-http://127.0.0.1:8888/api/v1}
REDIS="redis-cli -h 127.0.0.1 -p 16379"
EMAIL="user$(date +%s)@example.com"
PLAIN="secret123"

pass=0; fail=0
check() { # check <描述> <实际> <期望(子串)>
  if [[ "$2" == *"$3"* ]]; then echo "  ✅ $1"; pass=$((pass+1));
  else echo "  ❌ $1 => $2"; fail=$((fail+1)); fi
}

echo "1) 注册新用户 $EMAIL"
R=$(curl -s -XPOST "$BASE/auth/register" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"username\":\"测试用户\",\"password\":\"$PLAIN\"}")
check "注册成功(code=200)" "$R" '"code":200'

echo "2) 重复注册被拒绝"
R=$(curl -s -XPOST "$BASE/auth/register" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PLAIN\"}")
check "邮箱已存在(code=10001)" "$R" '10001'

echo "3) 获取 salt / nonce / 验证码"
S=$(curl -s "$BASE/auth/getSalt/$EMAIL")
check "取 salt 成功" "$S" '"code":200'
SALT=$(echo "$S" | jq -r '.data')
[[ ${#SALT} -eq 32 ]] && { echo "  ✅ salt 为 32 位十六进制"; pass=$((pass+1)); } || { echo "  ❌ salt 长度异常: $SALT"; fail=$((fail+1)); }
NONCE=$(curl -s "$BASE/util/nonce" | jq -r '.data')
CAP=$(curl -s "$BASE/util/captcha")
CID=$(echo "$CAP" | jq -r '.data.captchaId')
ANSWER=$($REDIS GET "captcha:login:$CID")
[[ -n "$ANSWER" ]] && { echo "  ✅ 验证码答案已写 Redis"; pass=$((pass+1)); } || { echo "  ❌ 验证码答案为空"; fail=$((fail+1)); }

# 客户端密码推导（与 repo_backend 前端一致）：
#   库中 password = sha256(明文 + "_" + salt)，登录提交 sha256(库password + "_" + nonce)
STORED=$(printf '%s_%s' "$PLAIN" "$SALT" | shasum -a 256 | awk '{print $1}')
LOGIN_PWD=$(printf '%s_%s' "$STORED" "$NONCE" | shasum -a 256 | awk '{print $1}')

echo "4) 正常登录"
R=$(curl -s -XPOST "$BASE/auth/login" -H 'Content-Type: application/json' -d "{
  \"email\":\"$EMAIL\",\"password\":\"$LOGIN_PWD\",
  \"captchaId\":\"$CID\",\"captchaText\":\"$ANSWER\",\"nonce\":\"$NONCE\"}")
check "登录成功(code=200)" "$R" '"code":200'
TOKEN=$(echo "$R" | jq -r '.data')
check "返回 token" "$TOKEN" 'eyJ'
# JWT payload 为 base64url，需补齐 padding 解码后再断言
B64=$(printf '%s' "$TOKEN" | cut -d. -f2 | tr '_-' '/+')
case $((${#B64} % 4)) in 2) B64="${B64}=" ;; 3) B64="${B64}==" ;; esac
CLAIMS=$(printf '%s' "$B64" | base64 -d 2>/dev/null)
[[ -z "$CLAIMS" ]] && CLAIMS=$(printf '%s' "$B64" | base64 -D 2>/dev/null)
check "JWT payload role=merchant" "$CLAIMS" '"role":"merchant"'

echo "5) 验证码一次性 / nonce 防重放"
R=$(curl -s -XPOST "$BASE/auth/login" -H 'Content-Type: application/json' -d "{
  \"email\":\"$EMAIL\",\"password\":\"$LOGIN_PWD\",
  \"captchaId\":\"$CID\",\"captchaText\":\"$ANSWER\",\"nonce\":\"$NONCE\"}")
check "nonce 已消费防重放(code=10011)" "$R" '10011'

echo "6) 受保护接口"
R=$(curl -s "$BASE/auth/current" -H "authorization: $TOKEN")
check "current 返回用户信息" "$R" "$EMAIL"
CODE=$(curl -s -o /dev/null -w '%{http_code}' "$BASE/auth/current")
check "无 token 访问 401" "$CODE" '401'

echo "7) 登出并立即失效"
R=$(curl -s -XPOST "$BASE/auth/logout" -H "authorization: $TOKEN")
check "登出成功" "$R" '"code":200'
CODE=$(curl -s -o /dev/null -w '%{http_code}' "$BASE/auth/current" -H "authorization: $TOKEN")
check "登出后再访问 401" "$CODE" '401'

echo "8) 密码错误计数（重新取三件套，连续输错）"
for i in 1 2; do
  NONCE=$(curl -s "$BASE/util/nonce" | jq -r '.data')
  CAP=$(curl -s "$BASE/util/captcha"); CID=$(echo "$CAP" | jq -r '.data.captchaId')
  ANSWER=$($REDIS GET "captcha:login:$CID")
  R=$(curl -s -XPOST "$BASE/auth/login" -H 'Content-Type: application/json' -d "{
    \"email\":\"$EMAIL\",\"password\":\"wronghash\",\"captchaId\":\"$CID\",
    \"captchaText\":\"$ANSWER\",\"nonce\":\"$NONCE\"}")
  check "第 $i 次密码错误(10008)" "$R" '10008'
done

echo
echo "结果：$pass 通过 / $fail 失败"
[[ $fail -eq 0 ]]
