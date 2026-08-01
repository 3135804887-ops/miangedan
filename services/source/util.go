package source

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"time"
)

// NewSourceID 生成 RFC 4122 v4 UUID 字符串（控制面本地标识；数据平台正式采用 UUIDv7，
// 由存储层生成，此处仅为合成桩/单测提供稳定唯一标识）。
func NewSourceID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand 失败属系统级异常；兜底使用时间种子，保证进程内唯一（测试环境）。
		ts := time.Now().UnixNano()
		for i := 0; i < 8; i++ {
			b[i] = byte(ts >> (8 * i))
		}
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	dst := make([]byte, 36)
	hex.Encode(dst[0:8], b[0:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], b[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], b[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], b[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:36], b[10:16])
	return string(dst)
}

// sortSources 按来源类型优先级稳定排序，同优先级按检索时间倒序（最新在前）。
func sortSources(items []ProcessSource) {
	sort.SliceStable(items, func(i, j int) bool {
		pi := SourceTypeOrder[items[i].SourceType]
		pj := SourceTypeOrder[items[j].SourceType]
		if pi != pj {
			return pi < pj
		}
		return items[i].RetrievedAt.After(items[j].RetrievedAt)
	})
}
