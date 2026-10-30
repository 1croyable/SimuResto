# SimuResto

使用 Go goroutine 模拟 CROUS 大学食堂的多智能体项目，同时提供命令行模拟和基于 Ebitengine 的桌面平面图。项目愿景见 [中文说明](docs/zh/note_d_intention.md) / [法文说明](docs/fr/note_d_intention.md)。

项目使用 Go 1.27，`go.mod` 声明为 `go 1.27.0`。模拟核心使用标准库，图形前端使用 Ebitengine 2.10.5。

## 交付与依赖下载

本项目模块名为 `simuresto`，内部使用 `simuresto/simulation`、`simuresto/frontend` 导入本地代码，不依赖私有 GitHub 仓库。提交或压缩项目时保留 `go.mod`、`go.sum`、`cmd/`、`frontend/` 和 `simulation/`。第三方依赖不随项目提交，由 Go Modules 在首次构建时自动下载并缓存。

老师在 Windows 上安装 Go 1.27 后，在项目根目录执行：

```powershell
go run ./cmd/simuresto -gui
go build -o simuresto.exe ./cmd/simuresto
go test ./...
```

第三方导入保留原始包路径，例如 `github.com/hajimehoshi/ebiten/v2`；`ebiten/v2` 和 `ebiten/v2/vector` 是同一个 Ebitengine 模块中的两个包。Go 根据 `go.mod` 的固定版本自动获取缺少的依赖，并校验 `go.sum`，不需要先手动运行 `go get`。首次构建需要联网；后续构建复用本地缓存，程序运行不需要下载依赖。

Go 1.27 编译器仍需预先安装；其他操作系统可能还需要 Ebitengine 的平台开发组件。项目不使用 `vendor/`，也不设置全局 `GOFLAGS`；主动升级依赖时，应将更新后的 `go.mod` 和 `go.sum` 一起提交。

## 依赖来源

界面控件在 `frontend/` 中实现，没有额外引入 UI 控件库。依赖版本固定在 `go.mod`，校验值记录在 `go.sum`；没有使用自定义 `replace` 或来源不明的二进制包。

| 依赖 | 来源与用途 |
|---|---|
| `github.com/hajimehoshi/ebiten/v2` | [Ebitengine 官方项目](https://ebitengine.org/)，提供窗口、绘制、键鼠输入 |
| `golang.org/x/image` | [Go 项目维护的图像扩展库](https://go.googlesource.com/image/)，提供内嵌英文后备字体 |
| `github.com/ebitengine/gomobile`、`hideconsole`、`purego` | [Ebitengine 组织](https://github.com/ebitengine)维护，属于引擎的平台适配依赖 |
| `github.com/go-text/typesetting` | [go-text 原始仓库](https://github.com/go-text/typesetting)，由引擎文字模块间接使用，处理字体与排版 |
| `github.com/srwiley/rasterx` | [作者原始仓库](https://github.com/srwiley/rasterx)，由引擎文字模块间接使用，处理字形栅格化 |
| `golang.org/x/sync`、`x/sys`、`x/text` | Go 项目维护的扩展库，由引擎及文字模块间接使用 |

上述间接依赖均可在 [Ebitengine v2.10.5 的依赖清单](https://github.com/hajimehoshi/ebiten/blob/v2.10.5/go.mod)中追溯。`go mod verify` 校验模块下载缓存；`go mod why -m <模块路径>` 可查看引入链。Windows 中文字体从系统读取，不随项目下载或分发。

## 运行

桌面前端：

```powershell
go run ./cmd/simuresto -gui
go run ./cmd/simuresto -gui -students 120 -cashiers 1 -speed 2
go run ./cmd/simuresto -gui -autostart -paused
```

启动后先进入设置页，点击数值可以直接输入，也可以用 `+ / −` 调整，最后点击“开始模拟”。参数分为“人流与队列”“服务与库存”“行为与时钟”三页，覆盖学生人数、到达间隔、随机种子、各区容量、人员数量、服务耗时、每日库存、等待扣分、配菜上限和播放速度。开始前会检查数值范围及室外扣分规则；设置页不运行智能体。`-autostart` 可以跳过设置页，直接采用命令行参数。

![开始前设置](docs/images/setup-preview.png)

![食堂模拟前端](docs/images/simulation-preview.png)

前端按手绘图布局：右下入口及室外队列 → 室内队列 → 餐具位置 → 上方前菜甜点区 → 主菜区 → 左侧两个收银台 → 绕过饮水调料区前往用餐区。学生用圆点表示，绿色为素食、蓝色为肉食倾向；工作人员忙碌时为橙色，空闲时为灰色。点击学生可查看满意度、等待时间和主菜。

| 操作 | 快捷键 |
|---|---|
| 暂停 / 继续 | 空格 |
| 减速 / 加速 | 左 / 右方向键 |
| 推进一个时间步 | N |
| 使用相同种子重新开始 | R |
| 返回参数设置，结束当前轮次 | P |
| 切换单 / 双收银并重新开始 | C |
| 退出 | Esc |

图形模式默认 100 名学生、两个收银窗口、0.25 秒逻辑步长、4 倍速度。命令行模式保留原有默认值。暂停会停止模拟时间，返回设置、重开和退出会关闭上一轮全部智能体。逻辑更新固定为 60 TPS，绘制端独立限制为最多 60 FPS，高刷新率显示器也不会提高绘制频率；模拟倍速不改变帧率上限。窗口可缩放；画布使用当前屏幕的 DPI 和窗口物理像素，文字按目标字号直接绘制，避免放大低分辨率图像。Windows 自动读取微软雅黑，其他平台尝试系统 CJK 字体，找不到则显示英文；可用 `-font path/to/font.ttf` 指定字体。

餐具、饮水调料、二楼和餐桌当前是平面图标识。圆点的路径是可视化插值，不增加后端行走耗时；付款完成已计入吞吐量，随后向用餐区的移动仅为离场动画，尚未模拟座位、用餐与还盘。前端支持 1–2 个收银位置。

保存实际渲染帧并退出：

```powershell
go run ./cmd/simuresto -gui -screenshot setup.png
go run ./cmd/simuresto -gui -autostart -speed 16 -capture-at 65s -screenshot preview.png
```

命令行模拟：

```powershell
go run ./cmd/simuresto
go run ./cmd/simuresto -students 120 -payment-capacity 2 -cashiers 2
go run ./cmd/simuresto -students 20 -arrival-interval 0s -events
go run ./cmd/simuresto -vegetarian-mains 5 -meat-mains 10 -starters 15 -desserts 15
go run ./cmd/simuresto -json > report.json
go run ./cmd/simuresto -help
```

默认生成 80 名学生，每模拟秒到达一人。`-seed` 同时控制学生属性和选菜，默认 42；相同配置与种子产生相同结果。默认不等待真实时间，`-tick-interval 20ms` 可控制播放速度。Ctrl+C 会停止模拟并输出已完成部分的统计。

## 第一阶段规则

学生按以下路线移动：

```text
室外队列（50）→ 室内队列（20）→ 自助区（6）→ 主菜区（3）→ 付款区（1）→ 完成
```

室内有位置且没有室外等候者时，新到学生直接进入室内。所有区域都是有容量限制的 FIFO；容量包括正在取餐、接受服务以及等待前方腾出位置的人。下一区域满时，学生继续占用当前区域，后面的学生不能越过队首。室内外队列均满时，新到学生被拒绝进入。各区容量通过 `-outdoor-capacity`、`-indoor-capacity`、`-self-capacity`、`-main-capacity`、`-payment-capacity` 调整。

- **学生**：独立 ID、到达时间、素食或肉食倾向、耐心时长、满意度（0–100）、前菜和甜点的期望数量。素食学生不会领取含肉菜品；肉食倾向的学生优先选择肉类主菜，缺货时接受素食替代并降低满意度。
- **自助区**：学生自行领取前菜、甜点，默认每类最多一份、两类合计最多两份。限制可通过 `-max-starters`、`-max-desserts`、`-max-self-items` 调整。这里的“总额”按领取份数和每日库存处理，暂未模拟价格或餐费预算。
- **主菜区**：每名学生最多领取一份主菜。默认两名工作人员，每份主菜服务 8 秒。按每日预制库存接单；接单时预留、交付时扣除预留。取消服务时归还尚未交付的预留库存，不自动补货。`mainChoices` 是按偏好排序的候选列表，实际订单只有一个 `dishID`，餐盘中的 `Meal.Main` 也只有一个 `*Dish`。
- **付款区**：默认一名收银员、一个位置，每次付款 4 秒。两窗口场景设置 `-payment-capacity 2 -cashiers 2`。已付款学生离开服务路线；本阶段尚未模拟座位、用餐和还盘。
- **满意度与放弃**：室内及服务区域每等待一秒扣 0.08，室外扣 0.30；实际自取、打饭和付款的服务时长不计为等待。收到符合倾向的主菜加 3 分，素食替代肉食扣 10 分，没有可接受主菜扣 25 分，缺少一份期望配菜扣 5 分；每领取一个菜品类别加 2 分。数值始终限制在 0–100。
- **耐心**：仅在领取托盘前（室内外排队）因累计等待达到耐心值或满意度降至 0 而离队。进入自助区后继续完成付款；主菜缺货时会带已有配菜继续付款，并记录缺货事件。即使餐品全部缺货，流程也会记录一次付款完成，因而吞吐量表示完成服务的人数，并不等同于售出的主菜份数。

自助取餐默认耗时 4 秒，主菜 8 秒，付款 4 秒，可分别用 `-self-time`、`-main-time`、`-payment-time` 调整。其他行为参数以及任意菜品列表可通过 Go API 的 `simulation.Config` 设置。

## 菜单、满意度与选菜概率

默认菜单定义在 `simulation/menu.go` 的 `DefaultMenu()` 中，模拟使用 `Config.Menu`，可以替换为自定义菜单。每个 `Dish` 的 `SatisfactionGain float32` 表示实际领取一份时增加的满意度，允许范围为 0–100。它与原有的偏好奖励、类别奖励及缺货惩罚叠加；学生最终满意度始终限制在 0–100，例如 96 分领取加 7 分的菜后变为 100 分。

菜名参考 [CROUS Bordeaux-Aquitaine：Le Sirtaki 官方菜单](https://www.crous-bordeaux.fr/restaurant/crous-cafet-le-sirtaki/)，检索日期为 2026-10-30。参考的是 2026-10-19 至 10-23 的午餐菜单，以及同页 Crous Market 的水果泥、布朗尼等供应品类；本模拟把多日选项合并，不代表当天实际同时供应。素食标记按无肉、无鱼的模拟分类设置，未核实每道菜的具体配方。

**下列满意度收益和库存均为人为设定的模拟参数，不是 CROUS 官方评分、调查结果或实际备餐量。** 分数表示领取一份后增加多少满意度，而非菜品的百分制评价。

| 类别 | 法文菜名 | 中文说明 | 满意度收益 | 默认库存 |
|---|---|---|---:|---:|
| 前菜 | Taboulé | 库斯库斯凉拌沙拉 | +4 | 23 |
| 前菜 | Chou rouge | 红甘蓝沙拉 | +2 | 23 |
| 前菜 | Salade de pois chiche | 鹰嘴豆沙拉 | +4 | 22 |
| 前菜 | Salade de riz au thon | 金枪鱼米饭沙拉 | +5 | 22 |
| 素食主菜 | Chili sin carne | 无肉辣豆炖菜 | +7 | 14 |
| 素食主菜 | Omelette | 法式煎蛋 | +5 | 13 |
| 素食主菜 | Crumble courgette et parmesan | 西葫芦帕尔马干酪酥烤 | +7 | 13 |
| 荤食主菜 | Sauté de dinde tandoori | 坦都里风味炒火鸡肉 | +8 | 20 |
| 荤食主菜 | Cuisse de poulet | 鸡腿 | +8 | 20 |
| 荤食主菜 | Poisson pané au fromage | 奶酪裹粉鱼排 | +7 | 20 |
| 甜点 | Fruits de saison | 时令水果 | +3 | 23 |
| 甜点 | Yaourts variés | 什锦口味酸奶 | +3 | 23 |
| 甜点 | Compote | 水果泥 | +4 | 22 |
| 甜点 | Brownie | 巧克力布朗尼 | +6 | 22 |

同一类别及饮食优先级内，抽选权重为 `1 + SatisfactionGain`，概率为该菜权重除以候选权重总和。例如四种甜点的权重依次是 4、4、5、7，均有库存时，首选概率依次为 20%、20%、25%、35%。基础权重 1 使收益为零的菜仍有被选择的机会。

每名学生使用独立随机源，在启动时抽取一次完整候选顺序；不会每秒重新抽选。领取时按顺序跳过缺货菜品。素食学生排除含肉菜品；肉食学生先在肉类主菜中抽选，全部缺货后才尝试素食替代。每人最多领取一份主菜；允许多份配菜时，会继续优先领取同一种有库存的候选菜，不对每份重新抽选。

`-starters`、`-desserts`、`-vegetarian-mains`、`-meat-mains` 表示各组的总库存，平均分配到该组菜品，余数按菜单顺序分配。默认总量仍分别是 90、90、40、60。菜品收益只在实际领取时计入：自助菜品领取时增加，主菜交付时增加；仅预留或取消预留不会增加满意度。

## 时间、并发与统计

每名学生和每名工作人员各自运行在一个 goroutine 中，以 channel 接收观察或任务、返回决策或完成信号。协调器独占队列、库存和结果数据，按固定顺序应用回复，避免数据竞争和库存超发。工作人员在自己的 goroutine 中维护任务剩余时间。

逻辑时钟默认每步 1 秒（`-tick`）。到达及服务完成时间向上对齐到时钟边界；等待学生在每步结束时判断是否离队，再进行下一轮移位。因此需要更精细的时间时可缩小 `-tick`。`-tick-interval` 只控制真实播放速度，不影响模拟结果。默认最大模拟时间为 15 分钟（`-max-duration`）；超过上限返回部分报告及错误，尚未结束或到达的学生记为取消。

结果包含每名学生的路线、各区等待时间、餐盘、最终满意度及状态，另有库存、事件列表、各区人数峰值和工作人员利用率。平均等待与满意度覆盖实际入场的学生（包含中途离队及取消者），各区平均等待以进入该区的人数为分母；放弃率为领取托盘前离队人数 / 实际入场人数。排队满员导致的拒绝单独统计。利用率为实际服务时长 / 模拟总时长。JSON 中 `time.Duration` 采用 Go 默认的纳秒整数。

## 代码结构与验证

```text
cmd/simuresto/       唯一 main 入口；-gui 启动窗口，默认运行命令行
frontend/           平面图、圆点路径、控制按钮、中文字体和高 DPI 绘制
simulation/
  enums.go          饮食倾向、菜品类别、区域、生命周期、信号、岗位枚举
  model.go          学生、工作人员、菜品、库存记录与餐盘
  config.go         默认参数，按时钟、服务、领取限额、满意度分组
  menu.go           默认每日菜单及每种菜品的满意度收益
  choice.go         按收益权重抽取候选顺序，处理饮食限制和独立随机源
  validation.go     配置及输入校验
  population.go     按种子生成学生
  queue.go          有容量限制的 FIFO
  simulation.go     命令行 Run 入口，连续推进会话
  session.go        可逐步推进的会话，供命令行和图形前端共用
  snapshot.go       独立的状态副本，包含区域队列和工作人员任务
  engine.go         协调器状态、初始化、goroutine 启动和退出
  student_agent.go  学生观察、选菜候选、离队决策及 channel 通信
  staff_agent.go    工作人员任务、剩余服务时间及 channel 通信
  movement.go       入场、区域转移、排队回堵
  self_service.go   自助领取与取餐耗时
  staff_service.go  派单、打饭与付款完成处理
  inventory.go      库存领取、预留、交付与归还
  satisfaction.go   等待计时及满意度变化
  events.go         事件结构与记录
  report.go         结果结构及指标汇总
```

这些文件属于同一个 `simulation` 包，按职责划分，业务调用方仍使用 `simulation.Run`。枚举使用 Go 的具名类型和常量，保留字符串值，便于阅读日志和 JSON；`Status` 表示学生生命周期，`Area` 表示位置，`Signal` 表示发生的事件。`ServiceRoute()` 返回路线副本，避免外部修改服务顺序。

学生的满意度、满意度参数、观察消息和个人利用率使用 `float32`；时间使用 `time.Duration`，ID 和份数使用 `int`。汇总均值和比例保留 `float64`，满意度先转为 `float64` 再累加，减少统计时额外的舍入误差。单个学生的浮点结果与旧版可能存在微小差异。

推荐阅读顺序：先看 `model.go` 和 `enums.go` 了解有哪些对象与状态，再看 `simulation.go` 的主循环，然后跟进 `student_agent.go`、`movement.go` 和 `staff_service.go`。最后阅读 `inventory.go` 与 `satisfaction.go`，理解一次服务如何改变库存和学生状态。

以一名学生为例：`admit` 接纳学生 → `moveForward` 将其推进自助区 → 学生 goroutine 返回选菜候选 → `startSelfService` 领取配菜 → 自助耗时结束后进入主菜区 → `assignStaff` 预留一份主菜 → 工作人员 goroutine 完成任务 → `deliverMain` 交付 → 进入付款区 → 收银完成后离开。`started` 表示本区服务已开始，`ready` 表示本区服务已完成；即使 `ready`，也必须等下一区域有位置才能前进。

```powershell
go test ./...
go vet ./...
go test -race ./...
```

Windows 上运行 `-race` 需要启用 CGO 并安装兼容的 C 编译器。

## 前端性能测量

圆角面板和按钮共用按物理像素生成的小纹理：四角保持原尺寸，边和中心伸展，并由 `DrawImage` 合批。这样不必每帧把一个圆角矩形拆成六个形状重复执行矢量抗锯齿。平面图、学生圆点、工作人员标记、忙碌圈和选择圈均使用缓存；窗口尺寸或 DPI 改变时重建，文字仍按原生像素绘制。实现参考 [Ebitengine 官方性能建议](https://ebitengine.org/en/documents/performancetips.html)。

限帧器按绝对计划时刻调度，不将 `Sleep` 每次晚唤醒的误差累加到下一帧；如果错过整帧则重新同步，避免卡顿后突发补绘。正常运行目标为 60 FPS，采样和系统调度会有小幅波动。

性能诊断使用 Go 标准库，没有增加依赖。下面命令预热 1 秒，再测量 6 秒并自动退出；会输出实际画布尺寸、按帧间隔计算的平均 FPS、95% 分位帧间隔、Update / Draw 的 CPU 平均耗时和限帧等待时间。Draw 耗时不包含后续 GPU 执行，不能当作 GPU 帧耗时。

```powershell
go run ./cmd/simuresto -gui -perf-duration 6s
go run ./cmd/simuresto -gui -autostart -speed 4 -perf-duration 6s
go run ./cmd/simuresto -gui -autostart -perf-duration 6s -perf-uncapped
go run ./cmd/simuresto -gui -autostart -perf-duration 6s -cpuprofile cpu.out
go tool pprof -top cpu.out
```

`-perf-uncapped` 只用于短时检查渲染余量，必须同时指定 `-perf-duration`，不会改变普通启动的限帧设置。

2026-10-30，在 i9-13980HX 的 Intel UHD 核显、1920×1200 画布上：圆角纹理优化将设置页从约 16 提升至 58 FPS；再修正限帧调度后，设置页、默认模拟和 2000 人场景均约 60 FPS。短时不限帧测量约为设置页 503 FPS、默认模拟 547 FPS。这是本机当前场景的测量，不是 GPU 通用性能排名；实际结果随窗口尺寸、后台负载和功耗状态变化。
