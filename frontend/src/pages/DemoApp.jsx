import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'

const CLIENT_ID = 'demo-app'
const CLIENT_SECRET = 'demo-secret' // demo 专用；生产环境 client_secret 只应保存在业务系统服务端

export default function DemoApp() {
  const [params] = useSearchParams()
  const code = params.get('code')
  const state = params.get('state')

  const [kv, setKv] = useState([])
  const [out, setOut] = useState('')
  const [done, setDone] = useState(false)

  const login = () => {
    const q = new URLSearchParams({
      client_id: CLIENT_ID,
      redirect_uri: window.location.origin + '/demo-app/callback',
      response_type: 'code',
      state: 'st_' + Math.random().toString(36).slice(2, 10)
    })
    window.location.href = '/oauth/authorize?' + q.toString()
  }

  useEffect(() => {
    if (!code) return
    let cancelled = false
    ;(async () => {
      setKv([
        { label: '授权码 code', value: code.slice(0, 16) + '…' },
        { label: 'state', value: state || '-' , hint: '（应与发起时一致，防 CSRF）'},
      ])

      // 用 code + client 凭证换 access_token（真实系统应在业务系统服务端完成）
      const body = new URLSearchParams({
        grant_type: 'authorization_code',
        client_id: CLIENT_ID,
        client_secret: CLIENT_SECRET,
        code,
        redirect_uri: window.location.origin + '/demo-app/callback'
      })
      const tr = await fetch('/oauth/token', { method: 'POST', body })
      const tj = await tr.json()
      if (cancelled) return
      if (!tr.ok || !tj.access_token) {
        setKv((k) => [...k, { kind: 'err', value: '换取 token 失败：' + (tj.error_description || '') }])
        setOut(JSON.stringify(tj, null, 2))
        setDone(true)
        return
      }

      // 用 access_token 获取用户信息
      const ur = await fetch('/oauth/userinfo', {
        headers: { Authorization: 'Bearer ' + tj.access_token }
      })
      const uj = await ur.json()
      if (cancelled) return
      setKv((k) => [...k, { kind: 'ok', value: 'SSO 登录成功，已获得 access_token 并获取用户信息 ✓' }])
      setOut(JSON.stringify({ token: tj, userinfo: uj.data || uj }, null, 2))
      setDone(true)
    })()
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [code])

  return (
    <div className="demo-body">
      <header className="demo-header">
        <span style={{ fontSize: 18, fontWeight: 600 }}>演示业务系统</span>
        <span className="badge">demo-app</span>
        <span style={{ marginLeft: 'auto', fontSize: 12, color: '#8ca0b3' }}>
          模拟第三方系统，验证 OAuth2 授权码模式
        </span>
      </header>

      <main className="demo-main">
        {!code ? (
          <div className="card">
            <h2>接入 SSO 单点登录</h2>
            <p className="desc">
              本页模拟一个独立的业务系统。点击下方按钮将跳转到 SSO 统一登录页
              （<code>/oauth/authorize</code>），登录成功后 SSO 会携带 <code>code</code> 与
              <code>state</code> 302 回跳到本页（<code>/demo-app/callback</code>），
              页面再以 <code>code + client_secret</code> 换取 <code>access_token</code>。
              <br />
              <span style={{ color: '#cf1322' }}>
                注意：真实业务系统换取 token 应在服务端完成，demo 简化在前端发起。
              </span>
            </p>
            <button className="demo-btn" onClick={login}>使用 SSO 登录</button>
          </div>
        ) : (
          <div className="card">
            <h2>SSO 授权结果</h2>
            <div>
              {kv.map((item, i) =>
                item.kind === 'ok' || item.kind === 'err' ? (
                  <div key={i} className={`kv ${item.kind}`}>{item.value}</div>
                ) : (
                  <div key={i} className="kv">
                    {item.label}：<b>{item.value}</b>{item.hint || ''}
                  </div>
                )
              )}
            </div>
            {done && <pre>{out}</pre>}
          </div>
        )}
      </main>
    </div>
  )
}
