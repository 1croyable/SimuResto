package engine

import (
	"context"
	"errors"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

var ErrTimeLimit = errors.New("simulation reached its maximum duration")

// Run 在命令行中连续推进 Game；图形前端则按自己的播放速度调用 Step。
// 所有智能体、队列、库存与满意度规则均由同一个模拟执行。
func Run(ctx context.Context, config environment.Config, students []agents.Student) (Report, error) {
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
	}
}
