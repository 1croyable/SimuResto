package engine

import (
	"math/rand/v2"
	"time"
)

// randomServiceTime 在包含上下限的范围内均匀抽取耗时。
// 调用处的范围来自正数常量，完成事件在后续逻辑时钟边界处理。
func randomServiceTime(minimum, maximum time.Duration) time.Duration {
	return minimum + time.Duration(rand.Int64N(int64(maximum-minimum)+1))
}
