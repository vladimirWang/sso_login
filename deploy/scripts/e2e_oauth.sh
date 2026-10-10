#!/usr/bin/env bash
# OAuth2 授权码模式端到端测试
# 前置：auth-rpc(8081) / auth-api(8888) 已启动，oauth_clients 已建表并种子 demo-app
set -euo pipefail

BASE=${BASE:-http://127.0.0.1:8888}
API="$BASE/api/v1"
REDIS="redis-cli -h 127.0.0.1 -p 16379"
JAR=$(mktemp)
trap 'rm -f "$JAR"' EXIT
EMAIL="oauth$(date +%s)@example.com"
PLAIN="secret123"
REDIRECT="http://127.0.0.1:8888/demo-app/callback"

pass=0; fail=0
check() {
  if [[ "$2" == *"$3"* ]]; then echo "  ✅ $1"; pass=$((pass+1));
  else echo "  ❌ $1 => $2"; fail=$((fail+1)); fi
}

echo "1) 注册并登录，验证 sso_token cookie 写入"
R=$(curl -s -XPOST "$API/auth/register" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"username\":\"oa\",\"password\":\"$PLAIN\"}")
check "注册成功" "$R" '"code":200'

SALT=$(curl -s "$API/auth/getSalt/$EMAIL" | jq -r '.data')
NONCE=$(curl -s "$API/util/nonce" | jq -r '.data')
CAP=$(curl -s "$API/util/captcha"); CID=$(echo "$CAP" | jq -r '.data.captchaId')
ANSWER=$($REDIS GET "captcha:login:$CID")
STORED=$(printf '%s_%s' "$PLAIN" "$SALT" | shasum -a 256 | awk '{print $1}')
LOGIN_PWD=$(printf '%s_%s' "$STORED" "$NONCE" | shasum -a 256 | awk '{print $1}')
R=$(curl -s -c "$JAR" -XPOST "$API/auth/login" -H 'Content-Type: application/json' -d "{
  \"email\":\"$EMAIL\",\"password\":\"$LOGIN_PWD\",
  \"captchaId\":\"$CID\",\"captchaText\":\"$ANSWER\",\"nonce\":\"$NONCE\"}")
check "登录成功" "$R" '"code":200'
if grep -q sso_token "$JAR"; then echo "  ✅ sso_token cookie 已写入"; pass=$((pass+1));
else echo "  ❌ sso_token cookie 未写入"; fail=$((fail+1)); fi

echo "2) 未登录访问 authorize → 302 到 SSO 登录页"
LOC=$(curl -s -o /dev/null -w '%{redirect_url}' \
  "$BASE/oauth/authorize?client_id=demo-app&redirect_uri=$REDIRECT&response_type=code&state=abc")
check "302 到 /oauth/login" "$LOC" '/oauth/login?client_id=demo-app'

echo "3) 已登录访问 authorize → 302 回业务系统携带 code+state"
LOC=$(curl -s -o /dev/null -w '%{redirect_url}' -b "$JAR" \
  "$BASE/oauth/authorize?client_id=demo-app&redirect_uri=$REDIRECT&response_type=code&state=abc123")
check "302 回 redirect_uri" "$LOC" "$REDIRECT"
check "携带 code" "$LOC" 'code='
check "携带 state" "$LOC" 'state=abc123'
CODE=$(printf '%s' "$LOC" | sed -n 's/.*[?&]code=\([^&]*\).*/\1/p')

echo "4) 错误 client_secret 换 token 被拒"
R=$(curl -s -XPOST "$BASE/oauth/token" \
  --data-urlencode "grant_type=authorization_code" \
  --data-urlencode "client_id=demo-app" --data-urlencode "client_secret=wrong" \
  --data-urlencode "code=$CODE" --data-urlencode "redirect_uri=$REDIRECT")
check "invalid_grant" "$R" 'invalid_grant'

echo "5) 正确凭证换 access_token"
R=$(curl -s -XPOST "$BASE/oauth/token" \
  --data-urlencode "client_id=demo-app" --data-urlencode "client_secret=demo-secret" \
  --data-urlencode "code=$CODE" --data-urlencode "redirect_uri=$REDIRECT")
check "返回 access_token" "$R" '"access_token":"'
AT=$(echo "$R" | jq -r '.access_token')
EXPIRES=$(echo "$R" | jq -r '.expires_in')
[[ "$EXPIRES" == "86400" ]] && { echo "  ✅ expires_in=86400"; pass=$((pass+1)); } \
  || { echo "  ❌ expires_in=$EXPIRES"; fail=$((fail+1)); }

echo "6) 授权码一次性：重放被拒"
R=$(curl -s -XPOST "$BASE/oauth/token" \
  --data-urlencode "client_id=demo-app" --data-urlencode "client_secret=demo-secret" \
  --data-urlencode "code=$CODE" --data-urlencode "redirect_uri=$REDIRECT")
check "重放 code 失败" "$R" 'invalid_grant'

echo "7) /oauth/userinfo 拉取用户信息"
R=$(curl -s "$BASE/oauth/userinfo" -H "authorization: Bearer $AT")
check "返回用户邮箱" "$R" "$EMAIL"
R=$(curl -s "$BASE/oauth/userinfo")
check "无 token 访问 401" "$R" '401'

echo "8) 安全校验：redirect_uri / client 校验"
HTTP=$(curl -s -o /dev/null -w '%{http_code}' -b "$JAR" \
  "$BASE/oauth/authorize?client_id=demo-app&redirect_uri=http://evil.example/cb&response_type=code")
check "未注册 redirect_uri 被拒(400)" "$HTTP" '400'
HTTP=$(curl -s -o /dev/null -w '%{http_code}' -b "$JAR" \
  "$BASE/oauth/authorize?client_id=nope&redirect_uri=$REDIRECT&response_type=code")
check "不存在 client 被拒(400)" "$HTTP" '400'

echo "9) SSO 登录页 / 演示页 / 应用信息"
HTTP=$(curl -s -o /dev/null -w '%{http_code}' "$BASE/oauth/login?client_id=demo-app&redirect_uri=$REDIRECT")
check "登录页 200" "$HTTP" '200'
HTTP=$(curl -s -o /dev/null -w '%{http_code}' "$BASE/demo-app")
check "演示业务系统页 200" "$HTTP" '200'
R=$(curl -s "$BASE/oauth/clientinfo?client_id=demo-app")
check "clientinfo 返回应用名" "$R" '演示业务系统'

echo
echo "结果：$pass 通过 / $fail 失败"
[[ $fail -eq 0 ]]
