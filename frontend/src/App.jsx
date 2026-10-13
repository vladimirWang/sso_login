import { Navigate, Route, Routes } from 'react-router-dom'
import LoginPage from './pages/LoginPage.jsx'
import DemoApp from './pages/DemoApp.jsx'

export default function App() {
  return (
    <Routes>
      {/* SSO 独立登录页：由业务系统 /oauth/authorize 未登录时 302 跳入，携带授权参数 */}
      <Route path="/oauth/login" element={<LoginPage />} />
      {/* 演示业务系统（含 OAuth2 回跳路径） */}
      <Route path="/demo-app" element={<DemoApp />} />
      <Route path="/demo-app/callback" element={<DemoApp />} />
      <Route path="/" element={<Navigate to="/demo-app" replace />} />
      <Route path="*" element={<div className="fallback-404">404 - 页面不存在</div>} />
    </Routes>
  )
}
