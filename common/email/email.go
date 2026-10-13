// Package email 极简 SMTP 发信客户端，用于发送注册邮箱验证码。
// QQ 邮箱要求 SMTPS（465 端口隐式 TLS），账号为完整邮箱地址，密码为 SMTP 授权码（非登录密码）。
package email

import (
	"crypto/tls"
	"fmt"
	"mime"
	"net/smtp"
	"strings"
)

type Sender struct {
	Host     string // smtp.qq.com
	Port     int    // 465
	Account  string // 发件邮箱，如 xxx@qq.com
	Password string // SMTP 授权码
	FromName string // 发件人展示名
}

// Configured 未配置账号时跳过发信（如本地开发环境）
func (s *Sender) Configured() bool {
	return s.Host != "" && s.Account != "" && s.Password != ""
}

// SendHTML 发送 HTML 邮件（to 为单个收件人）
func (s *Sender) SendHTML(to, subject, htmlBody string) error {
	if !s.Configured() {
		return fmt.Errorf("邮件服务未配置（缺少账号或授权码）")
	}
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: s.Host})
	if err != nil {
		return fmt.Errorf("连接 SMTP 服务器失败: %w", err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		return fmt.Errorf("创建 SMTP 客户端失败: %w", err)
	}
	defer c.Close()

	auth := smtp.PlainAuth("", s.Account, s.Password, s.Host)
	if err = c.Auth(auth); err != nil {
		return fmt.Errorf("SMTP 认证失败（检查授权码）: %w", err)
	}
	if err = c.Mail(s.Account); err != nil {
		return err
	}
	if err = c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	fromName := s.FromName
	if fromName == "" {
		fromName = s.Account
	}
	headers := map[string]string{
		"From":         fmt.Sprintf("%s <%s>", mime.QEncoding.Encode("utf-8", fromName), s.Account),
		"To":           to,
		"Subject":      mime.QEncoding.Encode("utf-8", subject),
		"MIME-Version": "1.0",
		"Content-Type": `text/html; charset="UTF-8"`,
	}
	var sb strings.Builder
	for k, v := range headers {
		sb.WriteString(k + ": " + v + "\r\n")
	}
	sb.WriteString("\r\n" + htmlBody)
	if _, err = w.Write([]byte(sb.String())); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	return c.Quit()
}
