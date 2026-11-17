package agents

import (
	"testing"
	"time"
)

func TestNormalArrivalsFollowGameTicks(t *testing.T) {
	generator := StudentGenerator{
		Tick:            250 * time.Millisecond,
		ArrivalDuration: 150 * time.Minute,
		ArrivalPeak:     45 * time.Minute,
		ArrivalSigma:    15 * time.Minute,
	}
	for _, count := range []int{0, 1, 12, 400, 2000} {
		students, err := generator.Generate(count)
		if err != nil {
			t.Fatal(err)
		}
		if len(students) != count {
			t.Fatalf("应生成 %d 人，实际 %d 人", count, len(students))
		}
		previous := time.Duration(0)
		for _, student := range students {
			arrival := student.ArrivalTime
			if arrival <= 0 || arrival > generator.ArrivalDuration || arrival%generator.Tick != 0 || arrival < previous {
				t.Fatalf("到达时间应按游戏刻度递增且在营业时间内：%s", arrival)
			}
			previous = arrival
		}
		if count != 400 {
			continue
		}
		batches := make(map[time.Duration]int)
		peakPeriod := 0
		for _, student := range students {
			batches[student.ArrivalTime]++
			// 统计 12:00–12:30 这段时间的客流。
			if student.ArrivalTime > 30*time.Minute && student.ArrivalTime <= 60*time.Minute {
				peakPeriod++
			}
		}
		// 正态分布在均值左右一个标准差内约有 68.3% 的人数。
		if peakPeriod < 270 || peakPeriod > 276 {
			t.Fatalf("400 人中高峰半小时应约到达 273 人，实际 %d", peakPeriod)
		}
		for _, arrivals := range batches {
			if arrivals > 1 {
				t.Fatal("400 人、0.25 秒刻度下，不应再出现整分钟集中到达")
			}
		}
	}
}

func TestNormalArrivalGeneratorRejectsInvalidParameters(t *testing.T) {
	for _, generator := range []StudentGenerator{
		{},
		{Tick: -time.Second, ArrivalDuration: time.Hour, ArrivalSigma: time.Minute},
		{Tick: time.Second, ArrivalDuration: time.Hour, ArrivalSigma: -time.Minute},
		{Tick: time.Second, ArrivalDuration: time.Hour, ArrivalSigma: time.Minute, ArrivalPeak: 2 * time.Hour},
		{Tick: time.Second, ArrivalDuration: time.Hour, ArrivalSigma: time.Minute, OpeningRatio: -0.1},
		{Tick: time.Second, ArrivalDuration: time.Hour, ArrivalSigma: time.Minute, OpeningRatio: 1.1},
		{Tick: time.Second, ArrivalDuration: time.Hour, ArrivalSigma: time.Minute, UniformRatio: -0.1},
		{Tick: time.Second, ArrivalDuration: time.Hour, ArrivalSigma: time.Minute, UniformRatio: 1.1},
	} {
		if _, err := generator.Generate(10); err == nil {
			t.Fatal("无效的客流参数应返回错误")
		}
	}
}

func TestMixedArrivalsKeepLateTrafficAndExactTotal(t *testing.T) {
	generator := StudentGenerator{
		Tick: 250 * time.Millisecond, ArrivalDuration: 135 * time.Minute,
		ArrivalPeak: 50 * time.Minute, ArrivalSigma: 20 * time.Minute,
		OpeningRatio: 0.02, UniformRatio: 0.20,
	}
	students, err := generator.Generate(1200)
	if err != nil {
		t.Fatal(err)
	}
	if len(students) != 1200 {
		t.Fatal("混合客流不能改变总人数")
	}
	opening, late, peak := 0, 0, 0
	for i, student := range students {
		arrival := student.ArrivalTime
		if arrival < 0 || arrival > generator.ArrivalDuration || arrival%generator.Tick != 0 {
			t.Fatal("所有学生必须在营业时间内按游戏刻度到达")
		}
		if i > 0 && arrival < students[i-1].ArrivalTime {
			t.Fatal("到达计划必须按时间排序")
		}
		if arrival == 0 {
			opening++
		}
		if arrival > 125*time.Minute {
			late++
		}
		if arrival > generator.ArrivalPeak-30*time.Second && arrival <= generator.ArrivalPeak+30*time.Second {
			peak++
		}
	}
	if opening != 24 {
		t.Fatalf("开门应到 24 人，实际 %d 人", opening)
	}
	if late < 17 || late > 19 {
		t.Fatalf("最后十分钟仍应有约 17–18 人的基础客流，实际 %d 人", late)
	}
	if peak < 20 || peak > 22 {
		t.Fatalf("12:20 附近一分钟应约到达 21 人，实际 %d 人", peak)
	}

	// 均匀占比为 100% 时，后续客流在营业前后半段各占一半。
	generator.UniformRatio = 1
	students, err = generator.Generate(1200)
	if err != nil {
		t.Fatal(err)
	}
	firstHalf := 0
	for _, student := range students {
		if student.ArrivalTime > 0 && student.ArrivalTime <= generator.ArrivalDuration/2 {
			firstHalf++
		}
	}
	if firstHalf != (1200-24)/2 {
		t.Fatalf("纯均匀客流的前半段应到 588 人，实际 %d 人", firstHalf)
	}
}

func TestOpeningStudentsAreIncludedInTotal(t *testing.T) {
	generator := StudentGenerator{
		Tick: 250 * time.Millisecond, ArrivalDuration: 150 * time.Minute,
		ArrivalPeak: 50 * time.Minute, ArrivalSigma: 20 * time.Minute, OpeningRatio: 0.02,
	}
	for _, sample := range []struct{ total, opening int }{{0, 0}, {1, 1}, {12, 1}, {400, 8}, {2000, 40}} {
		students, err := generator.Generate(sample.total)
		if err != nil {
			t.Fatal(err)
		}
		if len(students) != sample.total {
			t.Fatal("开门客流不能额外增加学生总数")
		}
		opening, peakPeriod := 0, 0
		for i, student := range students {
			if student.ArrivalTime == 0 {
				opening++
			}
			if student.ArrivalTime < 0 || student.ArrivalTime > generator.ArrivalDuration || student.ArrivalTime%generator.Tick != 0 {
				t.Fatal("开门客流和后续客流都应在营业时间内、对齐游戏刻度")
			}
			if i >= sample.opening && student.ArrivalTime <= students[i-1].ArrivalTime {
				t.Fatal("开门之后的学生应按新曲线陆续到达")
			}
			if student.ArrivalTime > 30*time.Minute && student.ArrivalTime <= 60*time.Minute {
				peakPeriod++
			}
		}
		if opening != sample.opening {
			t.Fatalf("总人数 %d，开门应到 %d 人，实际 %d 人", sample.total, sample.opening, opening)
		}
		if sample.total == 400 && (peakPeriod < 208 || peakPeriod > 212) {
			t.Fatalf("峰值 12:20、标准差 20 分钟时，12:00–12:30 应约有 210 人，实际 %d", peakPeriod)
		}
	}
}

func TestArrivalScheduleUsesConfiguredTick(t *testing.T) {
	for _, tick := range []time.Duration{50 * time.Millisecond, 700 * time.Millisecond, 2 * time.Second} {
		generator := StudentGenerator{
			Tick: tick, ArrivalDuration: 150 * time.Minute,
			ArrivalPeak: 45 * time.Minute, ArrivalSigma: 15 * time.Minute,
		}
		students, err := generator.Generate(400)
		if err != nil {
			t.Fatal(err)
		}
		for i, student := range students {
			if student.ArrivalTime <= 0 || student.ArrivalTime > generator.ArrivalDuration || student.ArrivalTime%tick != 0 {
				t.Fatalf("步长 %s：到达时间 %s 未对齐刻度", tick, student.ArrivalTime)
			}
			if i > 0 && student.ArrivalTime <= students[i-1].ArrivalTime {
				t.Fatal("本场景应逐个到达，且不能遗漏学生")
			}
		}
	}
}
