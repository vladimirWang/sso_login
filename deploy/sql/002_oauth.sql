-- OAuth2 授权码模式：第三方业务系统（client）注册表
-- client_secret 存储 sha256(明文secret) 十六进制，与用户密码策略一致
-- redirect_uris 逗号分隔，授权时精确匹配

CREATE TABLE IF NOT EXISTS oauth_clients (
    id            BIGSERIAL    PRIMARY KEY,
    client_id     VARCHAR(64)  NOT NULL UNIQUE,
    client_secret VARCHAR(128) NOT NULL,
    name          VARCHAR(64)  NOT NULL,
    redirect_uris TEXT         NOT NULL,
    -- ACTIVE / DISABLED
    status        VARCHAR(16)  NOT NULL DEFAULT 'ACTIVE',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);
COMMENT ON TABLE oauth_clients IS 'OAuth2 客户端注册表（接入 SSO 的业务系统）';

DO $$ BEGIN
    CREATE TRIGGER trg_oauth_clients_updated BEFORE UPDATE ON oauth_clients
        FOR EACH ROW EXECUTE FUNCTION set_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- 演示业务系统：secret = demo-secret（仅开发用，生产需更换）
-- client_secret = sha256("demo-secret")
INSERT INTO oauth_clients (client_id, client_secret, name, redirect_uris)
VALUES ('demo-app', 'cd577fe2561ebff23505db0bb006300c7cdecbd46bc0e03c449afafaca2c25bf',
        '演示业务系统',
        'http://127.0.0.1:8888/demo-app/callback,http://localhost:8888/demo-app/callback,https://todo.hetou.vip/demo-app/callback,https://www.todo.hetou.vip/demo-app/callback')
ON CONFLICT (client_id) DO NOTHING;
