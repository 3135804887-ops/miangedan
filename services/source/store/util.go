package store

import "time"

// timeNow 为列表失效过滤的时钟（测试可注入）。
var timeNow = time.Now
