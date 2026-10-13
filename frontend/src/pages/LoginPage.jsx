import { useCallback, useEffect, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { EMAIL_RE, getJSON, postJSON, runeLen, sha256Hex } from '../lib/api.js'

export default function LoginPage() {
  const [params] = useSearchParams()
  const authParams = {
    clientId: params.get('client_id') || '',
    redirectUri: params.get('redirect_uri') || '',
    state: params.get('state') || '',
    responseType: params.get('response_type') || 'code'
  }

  const [tab, setTab] = useState('login') // login | register
  const [tip, setTip] = useState({ text: '', invalid: false })
  const [disabled, setDisabled] = useState(false)
  const [loginEmail, setLoginEmail] = useState('')

  const loadClientInfo = useCallback(async () => {
    const clientId = authParams.clientId
    if (!clientId) {
      setTip({ text: '该页面为 SSO 登录页，需由业务系统跳转进入（携带 client_id 参数）', invalid: true })
      setDisabled(true)
      return
    }
    try {
      const j = await getJSON('/oauth/clientinfo?client_id=' + encodeURIComponent(clientId))
      if (j.code === 200 && j.valid) {
        setTip({ text: `「${j.name || clientId}」请求获取你的登录授权`, invalid: false })
      } else {
        setTip({ text: `应用不存在或已被停用（client_id=${clientId}）`, invalid: true })
        setDisabled(true)
      }
    } catch {
      setTip({ text: '授权服务暂不可用，请稍后重试', invalid: true })
      setDisabled(true)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [authParams.clientId])

  useEffect(() => {
    loadClientInfo()
  }, [loadClientInfo])

  return (
    <div className="login-body">
      <div className="panel">
        <div className="logo">S</div>
        <h1>SSO 统一登录</h1>
        {tip.text && <div className={`app-tip${tip.invalid ? ' invalid' : ''}`}>{tip.text}</div>}

        {tab === 'login' ? (
          <LoginForm
            disabled={disabled}
            presetEmail={loginEmail}
            onPresetConsumed={() => setLoginEmail('')}
            authParams={authParams}
            goRegister={() => setTab('register')}
          />
        ) : (
          <RegisterForm
            disabled={disabled}
            onRegistered={(email) => {
              setLoginEmail(email)
              setTab('login')
            }}
            goLogin={() => setTab('login')}
          />
        )}

        <div className="hint">
          登录即代表同意授权该应用获取你的
          <br />
          基本信息（邮箱 / 用户名 / 租户）
        </div>
      </div>
    </div>
  )
}

/* ------------------------------- 登录表单 ------------------------------- */

function LoginForm({ disabled, presetEmail, onPresetConsumed, authParams, goRegister }) {
  const [email, setEmail] = useState('')
  const [pwd, setPwd] = useState('')
  const [captcha, setCaptcha] = useState('')
  const [captchaId, setCaptchaId] = useState('')
  const [captchaImg, setCaptchaImg] = useState('')
  const [msg, setMsg] = useState('')
  const [msgOk, setMsgOk] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const refreshCaptcha = useCallback(async () => {
    const j = await getJSON('/api/v1/util/captcha')
    if (j.code === 200) {
      setCaptchaId(j.data.captchaId)
      setCaptchaImg(j.data.image)
      setCaptcha('')
    }
  }, [])

  useEffect(() => {
    refreshCaptcha()
  }, [refreshCaptcha])

  useEffect(() => {
    if (presetEmail) {
      setEmail(presetEmail)
      onPresetConsumed()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [presetEmail])

  const showMsg = (text, ok = false) => {
    setMsg(text)
    setMsgOk(ok)
  }

  const submit = async () => {
    const mail = email.trim()
    if (!mail || !pwd || !captcha) {
      showMsg('请填写完整信息')
      return
    }
    setSubmitting(true)
    try {
      // 密码推导协议：库中 password = sha256(明文+"_"+salt)，提交 = sha256(库中password+"_"+nonce)
      const saltResp = await getJSON('/api/v1/auth/getSalt/' + encodeURIComponent(mail))
      if (saltResp.code !== 200) {
        showMsg(saltResp.message || '用户不存在')
        await refreshCaptcha()
        return
      }
      const nonceJson = await getJSON('/api/v1/util/nonce')
      const nonce = nonceJson.data
      const stored = await sha256Hex(pwd + '_' + saltResp.data)
      const loginPwd = await sha256Hex(stored + '_' + nonce)
      const j = await postJSON('/api/v1/auth/login', {
        email: mail,
        password: loginPwd,
        captchaId,
        captchaText: captcha,
        nonce
      })
      if (j.code === 200) {
        showMsg('登录成功，正在跳转…', true)
        const back = new URLSearchParams({
          client_id: authParams.clientId,
          redirect_uri: authParams.redirectUri,
          response_type: authParams.responseType,
          state: authParams.state
        })
        setTimeout(() => {
          window.location.href = '/oauth/authorize?' + back.toString()
        }, 400)
        return
      }
      showMsg(j.message || '登录失败')
      await refreshCaptcha()
    } catch {
      showMsg('网络异常，请重试')
      await refreshCaptcha()
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <>
      <div className="tabs">
        <div className="tab active">登录</div>
        <div className="tab" onClick={goRegister}>注册</div>
      </div>
      <div className={`msg${msgOk ? ' ok' : ''}`}>{msg}</div>
      <div className="field">
        <label>邮箱</label>
        <input
          type="email" placeholder="请输入邮箱" autoComplete="username"
          value={email} onChange={(e) => setEmail(e.target.value)}
        />
      </div>
      <div className="field">
        <label>密码</label>
        <input
          type="password" placeholder="请输入密码" autoComplete="current-password"
          value={pwd} onChange={(e) => setPwd(e.target.value)}
        />
      </div>
      <div className="field">
        <label>验证码（点击图片刷新）</label>
        <div className="captcha-row">
          <input
            placeholder="请输入图中字符" maxLength={8}
            value={captcha} onChange={(e) => setCaptcha(e.target.value)}
          />
          <img src={captchaImg} alt="验证码" title="点击刷新" onClick={refreshCaptcha} />
        </div>
      </div>
      <button className="btn" disabled={disabled || submitting} onClick={submit}>
        {submitting ? '登录中…' : '登 录'}
      </button>
    </>
  )
}

/* ------------------------------- 注册表单 ------------------------------- */

function RegisterForm({ disabled, onRegistered, goLogin }) {
  const [email, setEmail] = useState('')
  const [code, setCode] = useState('')
  const [name, setName] = useState('')
  const [pwd, setPwd] = useState('')
  const [confirm, setConfirm] = useState('')

  const [msg, setMsg] = useState('')
  const [msgOk, setMsgOk] = useState(false)
  const [sending, setSending] = useState(false)
  const [countdown, setCountdown] = useState(0)
  // idle | sent | verifying | verified | fail
  const [codeStatus, setCodeStatus] = useState('idle')
  const [codeHint, setCodeHint] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const timerRef = useRef(null)

  useEffect(() => () => clearInterval(timerRef.current), [])

  const emailValid = EMAIL_RE.test(email.trim())
  const nameValid = runeLen(name.trim()) <= 8
  const pwdValid = pwd.length >= 6
  const match = pwd === confirm && confirm !== ''
  const formReady =
    emailValid && codeStatus === 'verified' && nameValid && pwdValid && match

  const showMsg = (text, ok = false) => {
    setMsg(text)
    setMsgOk(ok)
  }

  // 邮箱变动：验证码校验状态作废
  const onEmailChange = (v) => {
    setEmail(v)
    if (codeStatus !== 'idle') {
      setCodeStatus('idle')
      setCodeHint('')
    }
  }

  const startCountdown = () => {
    setCountdown(60)
    timerRef.current = setInterval(() => {
      setCountdown((c) => {
        if (c <= 1) {
          clearInterval(timerRef.current)
          return 0
        }
        return c - 1
      })
    }, 1000)
  }

  const sendCode = async () => {
    if (!emailValid) return
    setSending(true)
    showMsg('')
    try {
      const j = await postJSON('/api/v1/util/emailCode', { email: email.trim() })
      if (j.code === 200) {
        setCodeStatus('sent')
        setCodeHint('验证码已发送至邮箱，5 分钟内有效')
        startCountdown()
      } else {
        setCodeStatus('fail')
        setCodeHint(j.message || '发送失败')
      }
    } catch {
      setCodeStatus('fail')
      setCodeHint('网络异常，请重试')
    } finally {
      setSending(false)
    }
  }

  // 输满 6 位自动校验，校验通过后注册按钮才可点击
  useEffect(() => {
    if (code.length !== 6 || !emailValid) return
    let cancelled = false
    setCodeStatus('verifying')
    setCodeHint('校验中…')
    const t = setTimeout(async () => {
      try {
        const j = await postJSON('/api/v1/util/emailCode/verify', {
          email: email.trim(),
          code
        })
        if (cancelled) return
        if (j.code === 200) {
          setCodeStatus('verified')
          setCodeHint('邮箱校验通过')
        } else {
          setCodeStatus('fail')
          setCodeHint(j.message || '验证码不正确')
        }
      } catch {
        if (!cancelled) {
          setCodeStatus('fail')
          setCodeHint('网络异常，请重试')
        }
      }
    }, 300)
    return () => {
      cancelled = true
      clearTimeout(t)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [code])

  const submit = async () => {
    if (!formReady) return
    setSubmitting(true)
    showMsg('')
    try {
      const j = await postJSON('/api/v1/auth/register', {
        email: email.trim(),
        username: name.trim(),
        password: pwd,
        emailCode: code
      })
      if (j.code === 200) {
        showMsg('注册成功，正在跳转到登录…', true)
        setTimeout(() => onRegistered(email.trim()), 600)
      } else {
        showMsg(j.message || '注册失败')
        // 验证码可能已被消费/失效，允许重新走验证码流程
        setCodeStatus('idle')
        setCode('')
        setCodeHint('')
      }
    } catch {
      showMsg('网络异常，请重试')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <>
      <div className="tabs">
        <div className="tab" onClick={goLogin}>登录</div>
        <div className="tab active">注册</div>
      </div>
      <div className={`msg${msgOk ? ' ok' : ''}`}>{msg}</div>

      <div className="field">
        <label>邮箱</label>
        <input
          type="email" placeholder="请输入邮箱"
          className={email ? (emailValid ? 'ok' : 'bad') : ''}
          value={email} onChange={(e) => onEmailChange(e.target.value)}
        />
        <div className={`field-hint err`}>{email && !emailValid ? '邮箱格式不正确' : ''}</div>
      </div>

      <div className="field">
        <label>邮箱验证码</label>
        <div className="code-row">
          <input
            placeholder="请输入 6 位验证码" maxLength={6} inputMode="numeric"
            className={code.length === 6
              ? (codeStatus === 'verified' ? 'ok' : codeStatus === 'fail' ? 'bad' : '')
              : ''}
            value={code}
            onChange={(e) => {
              setCode(e.target.value.replace(/\D/g, '').slice(0, 6))
              if (codeStatus === 'verified' || codeStatus === 'fail') setCodeStatus('sent')
            }}
          />
          <button
            className="code-btn"
            disabled={disabled || !emailValid || sending || countdown > 0}
            onClick={sendCode}
          >
            {sending ? '发送中…' : countdown > 0 ? `${countdown}s 后重发` : '发送验证码'}
          </button>
        </div>
        <div className={`field-hint${codeStatus === 'verified' ? ' ok' : ' err'}`}>
          {codeHint}
        </div>
      </div>

      <div className="field">
        <label>用户名（可选，最长 8 位）</label>
        <input
          placeholder="请输入用户名"
          className={name && !nameValid ? 'bad' : ''}
          value={name} onChange={(e) => setName(e.target.value)}
        />
        <div className="field-hint err">{name && !nameValid ? '用户名最长 8 位' : ''}</div>
      </div>

      <div className="field">
        <label>密码（至少 6 位）</label>
        <input
          type="password" placeholder="请输入密码"
          className={pwd ? (pwdValid ? 'ok' : 'bad') : ''}
          value={pwd} onChange={(e) => setPwd(e.target.value)}
        />
        <div className="field-hint err">{pwd && !pwdValid ? '密码至少 6 位' : ''}</div>
      </div>

      <div className="field">
        <label>确认密码</label>
        <input
          type="password" placeholder="请再次输入密码"
          className={confirm ? (match ? 'ok' : 'bad') : ''}
          value={confirm} onChange={(e) => setConfirm(e.target.value)}
        />
        <div className="field-hint err">{confirm && !match ? '两次输入的密码不一致' : ''}</div>
      </div>

      <button className="btn" disabled={disabled || submitting || !formReady} onClick={submit}>
        {submitting ? '注册中…' : '注 册'}
      </button>
    </>
  )
}
