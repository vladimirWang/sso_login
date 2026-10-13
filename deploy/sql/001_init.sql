-- SSO 登录中心初始化脚本（PostgreSQL）
-- 表结构对齐 repo_backend 的 Tenant / User 模型，仅保留登录所需字段
-- 列名使用 snake_case；表名小写
--
-- 用法（psql 元命令 \gexec / \connect，须用 psql -f 执行）：
--   psql "host=<DB_HOST> port=<DB_PORT> user=postgres password=<DB_PASSWORD> dbname=postgres sslmode=disable" -f 001_init.sql
--   psql "..." -f 002_oauth.sql
--   psql "..." -f 003_repo_cms_client.sql
-- （002/003 需把连接串里的 dbname 改为 sso_login，或执行过 001 后直接 \connect sso_login）

-- 幂等建库：不存在才创建（CREATE DATABASE 不支持 IF NOT EXISTS，用 \gexec 动态执行）
SELECT 'CREATE DATABASE sso_login'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'sso_login')\gexec

\connect sso_login

-- 租户表
CREATE TABLE IF NOT EXISTS tenants (
    id            BIGSERIAL    PRIMARY KEY,
    name          VARCHAR(20)  NOT NULL UNIQUE,
    code          CHAR(6)      NOT NULL UNIQUE,
    logo          TEXT,
    -- ACTIVE / PENDING / SUSPENDED / DISABLED
    status        VARCHAR(16)  NOT NULL DEFAULT 'ACTIVE',
    -- 租户超级管理员（自建租户的注册人）。与 users.id 为循环关系，不建外键约束
    super_user_id BIGINT       UNIQUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);
COMMENT ON TABLE  tenants IS '租户表（对齐 repo_backend Tenant）';
COMMENT ON COLUMN tenants.super_user_id IS '租户超级管理员用户ID，逻辑关联 users.id';

-- 普通用户表
CREATE TABLE IF NOT EXISTS users (
    id         BIGSERIAL    PRIMARY KEY,
    tenant_id  BIGINT       NOT NULL REFERENCES tenants(id),
    email      VARCHAR(255) NOT NULL UNIQUE,
    username   VARCHAR(64),
    -- password = sha256(明文密码 + "_" + salt)，64 位十六进制
    password   VARCHAR(128) NOT NULL,
    salt       VARCHAR(64)  NOT NULL,
    avatar     TEXT,
    github_id  VARCHAR(128) UNIQUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id);
COMMENT ON TABLE users IS '普通用户表（对齐 repo_backend User）';

-- updated_at 自动维护
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$ BEGIN
    CREATE TRIGGER trg_users_updated BEFORE UPDATE ON users
        FOR EACH ROW EXECUTE FUNCTION set_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TRIGGER trg_tenants_updated BEFORE UPDATE ON tenants
        FOR EACH ROW EXECUTE FUNCTION set_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
