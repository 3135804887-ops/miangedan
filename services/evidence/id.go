package evidence

import (
	"crypto/rand"
	"encoding/hex"
)

// newID 生成 128 位随机十六进制标识（合成开发环境；生产由数据库 uuid 生成）。
func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}
