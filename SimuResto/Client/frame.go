package client

import (
	"time"
)

const FPSLimit = 60

// 向上取整到纳秒，60 个计划帧的时间不短于一秒。
const frameInterval = (time.Second + FPSLimit - 1) / FPSLimit

type frameLimiter struct{ next time.Time }

func (l *frameLimiter) delay(now time.Time) time.Duration {
	if l.next.IsZero() {
		return 0
	}
	return max(0, l.next.Sub(now))
}

// 沿绝对时间表安排下一帧，不把 Sleep 的每次晚唤醒累加到帧周期。
// 若已错过整帧则重新同步，不以连续补帧追赶长时间的卡顿。
func (l *frameLimiter) step(now time.Time) {
	if l.next.IsZero() || now.Sub(l.next) >= frameInterval {
		l.next = now.Add(frameInterval)
		return
	}
	l.next = l.next.Add(frameInterval)
}

// SetTPS 只控制逻辑更新频率，绘制仍需独立调度。
func (l *frameLimiter) wait() {
	if delay := l.delay(time.Now()); delay > 0 {
		time.Sleep(delay)
	}
	l.step(time.Now())
}
