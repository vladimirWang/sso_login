-- 仓库管理系统（repo_cms）接入 SSO 的 OAuth2 客户端注册
-- redirect_uri 使用 hash 路由形态（#/...），SSO 追加 ?code= 后实际落地为
--   http://localhost:5174/?code=xxx&state=yyy#/landing/sso/callback
-- secret 明文由 repo_cms 侧保管（repo_backend .env 的 SSO_CLIENT_SECRET），此处仅存 sha256
-- secret = repo-cms-ca4fcb283b57ab87e7c5b596（开发环境）
-- 本地开发支持两个前端端口：5174（vite 默认）/ 5175（vite --host 或 5174 被占用时）

INSERT INTO oauth_clients (client_id, client_secret, name, redirect_uris)
VALUES (
    'repo-cms',
    '97d32c2004fc638533730bcd0f37e77478bf64f34f2d733d394743b4265b741a',
    '仓库管理系统',
    'http://localhost:5174/#/landing/sso/callback,http://127.0.0.1:5174/#/landing/sso/callback,http://localhost:5175/#/landing/sso/callback,http://127.0.0.1:5175/#/landing/sso/callback'
)
ON CONFLICT (client_id) DO UPDATE
SET redirect_uris = EXCLUDED.redirect_uris,
    updated_at = now();
