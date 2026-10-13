// 与后端交互的轻量封装。全部走同源相对路径：
// 开发由 Vite proxy 转发到 auth-api，生产由 nginx 反向代理，任何环境都无 CORS。

async function request(path, options = {}) {
  const r = await fetch(path, options)
  let json
  try {
    json = await r.json()
  } catch {
    throw new Error('服务响应异常')
  }
  return json
}

export function postJSON(path, body) {
  return request(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  })
}

export function getJSON(path) {
  return request(path)
}

export async function sha256Hex(s) {
  const buf = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(s))
  return [...new Uint8Array(buf)].map((b) => b.toString(16).padStart(2, '0')).join('')
}

// 邮箱格式（与服务端 net/mail 宽松风格对齐，前端给即时反馈）
export const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

export function runeLen(s) {
  return [...(s || '')].length
}
