import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

// 开发环境：Vite dev server（5173）同源代理后端，避免 CORS。
// 目标地址可用 VITE_API_TARGET 覆盖（默认本地 go run 的 auth-api 8888）。
// 需要代理的路径：
//   /api/**                       REST 接口
//   /oauth/authorize|token|...    OAuth2 协议端点
// /oauth/login、/demo-app 是 SPA 页面路由，由前端 router 承接，不走代理。
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const target = env.VITE_API_TARGET || 'http://127.0.0.1:8888'

  const proxyOpts = { target, changeOrigin: true, secure: false }

  return {
    plugins: [react()],
    server: {
      port: 5173,
      proxy: {
        '/api': proxyOpts,
        '^/oauth/(authorize|token|userinfo|clientinfo|logout)': proxyOpts
      }
    }
  }
})
