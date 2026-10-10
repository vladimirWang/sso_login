// Package errcode 业务错误码，数值与 repo_backend errorCode 保持一致，便于前端/客户端复用
package errcode

const (
	Success           = 200
	EmailExisted      = 10001 // 邮箱已存在
	ValidationError   = 10002 // 校验失败
	UserNotFound      = 10003 // 用户不存在
	CaptchaExpired    = 10006 // 验证码已过期
	CaptchaIncorrect  = 10007 // 验证码不正确
	PasswordIncorrect = 10008 // 密码不正确
	EmailVerifyFail   = 10009 // 邮箱验证失败
	AccountLocked     = 10010 // 账号锁定
	NonceInvalid      = 10011 // nonce 无效
	SystemError       = 10015 // 系统错误
	Forbidden         = 10018 // 无权操作
	SessionInvalid    = 401   // 会话无效/已过期（供 ValidateSession 与网关使用）
)
