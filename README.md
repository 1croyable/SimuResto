# SimuResto

使用 Go goroutine 模拟 CROUS 大学食堂的多智能体项目。第一阶段提供无 GUI 的命令行模拟，项目愿景见 [中文说明](docs/zh/note_d_intention.md) / [法文说明](docs/fr/note_d_intention.md)。

需要 Go 1.22 或更新版本，无第三方依赖。

## 运行

```powershell
go run ./cmd/simuresto
go run ./cmd/simuresto -students 120 -payment-capacity 2 -cashiers 2
go run ./cmd/simuresto -students 20 -arrival-interval 0s -events
go run ./cmd/simuresto -vegetarian-mains 5 -meat-mains 10 -starters 15 -desserts 15
go run ./cmd/simuresto -json > report.json
go run ./cmd/simuresto -help
```

默认生成 80 名学生，每模拟秒到达一人。`-seed` 控制学生属性，默认 42；相同配置与种子产生相同结果。默认不等待真实时间，`-tick-interval 20ms` 可控制播放速度。Ctrl+C 会停止模拟并输出已完成部分的统计。

## 第一阶段规则

学生按以下路线移动：

```text
室外队列（50）→ 室内队列（20）→ 自助区（6）→ 主菜区（3）→ 付款区（1）→ 完成
```

室内有位置且没有室外等候者时，新到学生直接进入室内。所有区域都是有容量限制的 FIFO；容量包括正在取餐、接受服务以及等待前方腾出位置的人。下一区域满时，学生继续占用当前区域，后面的学生不能越过队首。室内外队列均满时，新到学生被拒绝进入。各区容量通过 `-outdoor-capacity`、`-indoor-capacity`、`-self-capacity`、`-main-capacity`、`-payment-capacity` 调整。

- **学生**：独立 ID、到达时间、素食或肉食倾向、耐心时长、满意度（0–100）、前菜和甜点的期望数量。素食学生不会领取含肉菜品；肉食倾向的学生优先选择肉类主菜，缺货时接受素食替代并降低满意度。
- **自助区**：学生自行领取前菜、甜点，默认每类最多一份、两类合计最多两份。限制可通过 `-max-starters`、`-max-desserts`、`-max-self-items` 调整。这里的“总额”按领取份数和每日库存处理，暂未模拟价格或餐费预算。
- **主菜区**：默认两名工作人员，每份主菜服务 8 秒。按每日预制库存接单；接单时预留、交付时扣除预留。取消服务时归还尚未交付的预留库存，不自动补货。
- **付款区**：默认一名收银员、一个位置，每次付款 4 秒。两窗口场景设置 `-payment-capacity 2 -cashiers 2`。已付款学生离开服务路线；本阶段尚未模拟座位、用餐和还盘。
- **满意度与放弃**：室内及服务区域每等待一秒扣 0.08，室外扣 0.30；实际自取、打饭和付款的服务时长不计为等待。收到符合倾向的主菜加 3 分，素食替代肉食扣 10 分，没有可接受主菜扣 25 分，缺少一份期望配菜扣 5 分；每领取一个菜品类别加 2 分。数值始终限制在 0–100。
- **耐心**：仅在领取托盘前（室内外排队）因累计等待达到耐心值或满意度降至 0 而离队。进入自助区后继续完成付款；主菜缺货时会带已有配菜继续付款，并记录缺货事件。即使餐品全部缺货，流程也会记录一次付款完成，因而吞吐量表示完成服务的人数，并不等同于售出的主菜份数。

自助取餐默认耗时 4 秒，主菜 8 秒，付款 4 秒，可分别用 `-self-time`、`-main-time`、`-payment-time` 调整。其他行为参数以及任意菜品列表可通过 Go API 的 `simulation.Config` 设置。

## 时间、并发与统计

每名学生和每名工作人员各自运行在一个 goroutine 中，以 channel 接收观察或任务、返回决策或完成信号。协调器独占队列、库存和结果数据，按固定顺序应用回复，避免数据竞争和库存超发。工作人员在自己的 goroutine 中维护任务剩余时间。

逻辑时钟默认每步 1 秒（`-tick`）。到达及服务完成时间向上对齐到时钟边界；等待学生在每步结束时判断是否离队，再进行下一轮移位。因此需要更精细的时间时可缩小 `-tick`。`-tick-interval` 只控制真实播放速度，不影响模拟结果。默认最大模拟时间为 15 分钟（`-max-duration`）；超过上限返回部分报告及错误，尚未结束或到达的学生记为取消。

结果包含每名学生的路线、各区等待时间、餐盘、最终满意度及状态，另有库存、事件列表、各区人数峰值和工作人员利用率。平均等待与满意度覆盖实际入场的学生（包含中途离队及取消者），各区平均等待以进入该区的人数为分母；放弃率为领取托盘前离队人数 / 实际入场人数。排队满员导致的拒绝单独统计。利用率为实际服务时长 / 模拟总时长。JSON 中 `time.Duration` 采用 Go 默认的纳秒整数。

## 代码结构与验证

```text
cmd/simuresto/       命令行入口
simulation/
  model.go          学生、工作人员、菜品、区域、信号与报告类型
  config.go         默认参数、配置校验和学生生成
  queue.go          有容量限制的 FIFO
  agents.go         学生与工作人员的 goroutine
  simulation.go     时钟、移位、库存、服务与取消处理
  report.go         指标汇总
```

```powershell
go test ./...
go vet ./...
go test -race ./...
```

Windows 上运行 `-race` 需要启用 CGO 并安装兼容的 C 编译器。
