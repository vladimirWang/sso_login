// Package crypto 密码学与随机数工具，逻辑与 repo_backend src/utils/algo.ts 完全对齐
package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// GenerateSalt 生成固定盐：16 字节随机数的十六进制（对应 randomBytes(16).toString("hex")）
func GenerateSalt() string {
	return randomHex(16)
}

// GenerateNonce 生成一次性 nonce：32 位十六进制随机串 + "_" + 毫秒时间戳
func GenerateNonce() string {
	return fmt.Sprintf("%s_%d", randomHex(16), time.Now().UnixMilli())
}

// Sha256 计算 sha256(s) 的十六进制小写字符串（对应 createHash("sha256").update(s).digest("hex")）
func Sha256(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// ParseNonce 解析 nonce，返回毫秒时间戳；格式非法返回 error
func ParseNonce(nonce string) (int64, error) {
	parts := strings.SplitN(nonce, "_", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return 0, fmt.Errorf("malformed nonce")
	}
	ts, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("malformed nonce timestamp")
	}
	return ts, nil
}

// IsNonceExpired nonce 是否超过 5 分钟有效期（300000 毫秒）
func IsNonceExpired(nonceMilli int64) bool {
	return time.Now().UnixMilli()-nonceMilli > 5*60*1000
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
