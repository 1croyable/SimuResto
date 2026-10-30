package simulation

import (
	"context"
	"errors"
	"time"
)

var ErrTimeLimit = errors.New("simulation reached its maximum duration")

// Run 在命令行中连续推进 Session；图形前端则按自己的播放速度调用 Step。
// 所有智能体、队列、库存与满意度规则均由同一个会话执行。
func Run(ctx context.Context, config Config, students []Student) (Report, error) {
	session, err := NewSession(config, students)
	if err != nil {
		return Report{}, err
	}
	defer session.Close()
	for {
		done, err := session.Step(ctx)
		if done {
			return session.Report(), err
		}
		if config.TickInterval > 0 && session.engine.now < config.MaxDuration {
			timer := time.NewTimer(config.TickInterval)
			select {
			case <-ctx.Done():
				timer.Stop()
				session.Close()
				return session.Report(), ctx.Err()
			case <-timer.C:
			}
		}
	}
}
