package simulation

import (
	"context"
	"errors"
	"time"
)

var ErrTimeLimit = errors.New("simulation reached its maximum duration")

// Run 在命令行中连续推进 Game；图形前端则按自己的播放速度调用 Step。
// 所有智能体、队列、库存与满意度规则均由同一个模拟执行。
func Run(ctx context.Context, config Config, students []Student) (Report, error) {
	game, err := NewGame(config, students)
	if err != nil {
		return Report{}, err
	}
	defer game.Close()
	for {
		done, err := game.Step(ctx)
		if done {
			return game.Report(), err
		}
		if config.TickInterval > 0 && game.engine.now < config.MaxDuration {
			timer := time.NewTimer(config.TickInterval)
			select {
			case <-ctx.Done():
				timer.Stop()
				game.Close()
				return game.Report(), ctx.Err()
			case <-timer.C:
			}
		}
	}
}
