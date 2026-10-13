.PHONY: gen-rpc gen-api build tidy sql test-api test-oauth

# 重新生成 gRPC 骨架（修改 proto/auth/auth.proto 后执行）
gen-rpc:
	goctl rpc protoc proto/auth/auth.proto \
		--go_out=./services/auth/rpc \
		--go-grpc_out=./services/auth/rpc \
		--zrpc_out=./services/auth/rpc

# 重新生成 HTTP 骨架（修改 services/auth/api/auth.api 后执行；
# 注意：已存在的 logic/svc/config 会被跳过，刷新 types/handler 需先删除对应生成文件）
gen-api:
	goctl api go -api services/auth/api/auth.api --dir services/auth/api

tidy:
	go mod tidy

build:
	mkdir -p bin
	go build -o bin/auth-rpc ./services/auth/rpc
	go build -o bin/auth-api  ./services/auth/api
	go build -o bin/audit      ./services/audit

# 初始化数据库（连接参数来自 .env.dev）：001 内含幂等 CREATE DATABASE，故先连默认库 postgres
sql:
	set -a && . ./.env.dev && set +a && \
	psql "host=$$DB_HOST port=$$DB_PORT user=$$DB_USER password=$$DB_PASSWORD dbname=postgres sslmode=$$DB_SSLMODE" \
		-f deploy/sql/001_init.sql && \
	psql "host=$$DB_HOST port=$$DB_PORT user=$$DB_USER password=$$DB_PASSWORD dbname=$$DB_NAME sslmode=$$DB_SSLMODE" \
		-f deploy/sql/002_oauth.sql && \
	psql "host=$$DB_HOST port=$$DB_PORT user=$$DB_USER password=$$DB_PASSWORD dbname=$$DB_NAME sslmode=$$DB_SSLMODE" \
		-f deploy/sql/003_repo_cms_client.sql

# 启动（需在三个终端分别执行，依赖 etcd:2379 / pg:15432 / redis:16379 / rabbitmq:5672）
# auth-rpc 的数据库连接参数从 .env.dev 读取
run-rpc:
	cd services/auth/rpc && set -a && . ../../../.env.dev && set +a && go run . -f etc/auth.yaml

run-api:
	cd services/auth/api && set -a && . ../../../.env.dev && set +a && go run . -f etc/auth-api.yaml

run-audit:
	cd services/audit && go run . -f etc/audit.yaml

# 端到端冒烟测试（登录主链路 / OAuth2 授权码模式）
test-api:
	bash deploy/scripts/e2e.sh

test-oauth:
	bash deploy/scripts/e2e_oauth.sh
