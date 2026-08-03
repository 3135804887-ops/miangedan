// Package adminapi 提供后台服务标识生成（TASK-080）。
package adminapi

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// newID 生成 UUID v4 形态随机标识（开发/测试；生产由数据库 uuid 生成）。
func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	)
}
