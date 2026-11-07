package client

import (
	"testing"
	"time"
)

func TestFrameLimitDoesNotDependOnMonitorRefresh(t *testing.T) {
	start := time.Unix(100, 0)
	limiter := frameLimiter{}
	limiter.step(start)
	if delay := limiter.delay(start.Add(time.Second / 144)); delay <= 0 {
		t.Fatal("144 Hz 屏幕上的下一帧必须等待")
	}
	if delay := limiter.delay(start.Add(frameInterval)); delay != 0 {
		t.Fatal("到达计划时刻后可以绘制下一帧")
	}
	if frameInterval*FPSLimit < time.Second {
		t.Fatal("计划帧率不能超过 60 FPS")
	}
}

func TestLateWakeupDoesNotAccumulate(t *testing.T) {
	start := time.Unix(100, 0)
	limiter := frameLimiter{}
	limiter.step(start)
	// 模拟每次 Sleep 都晚醒 0.5 ms，六百帧后也不能积累成 0.3 秒的额外延迟。
	for frame := 1; frame <= 600; frame++ {
		planned := start.Add(time.Duration(frame) * frameInterval)
		limiter.step(planned.Add(500 * time.Microsecond))
		if limiter.next != planned.Add(frameInterval) {
			t.Fatalf("第 %d 帧把晚唤醒误差累加到了下一帧", frame)
		}
	}
}

func TestFrameLimitResynchronizesAfterStall(t *testing.T) {
	start := time.Unix(100, 0)
	limiter := frameLimiter{}
	limiter.step(start)
	resumed := start.Add(time.Second)
	if limiter.delay(resumed) != 0 {
		t.Fatal("卡顿后第一帧不应额外等待")
	}
	limiter.step(resumed)
	if limiter.delay(resumed) != frameInterval {
		t.Fatal("卡顿后不能连续突发补绘旧帧")
	}
}
