// Package frontend 使用 Ebitengine 展示食堂平面图和模拟快照。
package client

import (
	"math"

	agents "simuresto/SimuResto/Agents"
)

const Width, Height = 1440, 900

type point struct{ x, y float64 }
type rectangle struct{ x, y, w, h float32 }

func (r rectangle) contains(x, y int) bool {
	return float32(x) >= r.x && float32(x) <= r.x+r.w && float32(y) >= r.y && float32(y) <= r.y+r.h
}

var (
	pauseButton   = rectangle{755, 24, 110, 42}
	speedButton   = rectangle{877, 24, 100, 42}
	restartButton = rectangle{989, 24, 120, 42}
	cashierButton = rectangle{1121, 24, 180, 42}
	stepButton    = rectangle{1313, 24, 106, 42}
)

// 平面图坐标集中在此处：右下入口，上方自助与主菜，左侧收银，下方用餐。
var entryPath = []point{{970, 655}, {970, 515}, {820, 515}, {820, 350}}
var indoorLine = []point{{820, 400}, {820, 515}, {970, 515}, {970, 635}}

func alongLine(points []point, distance float64) point {
	for i := 1; i < len(points); i++ {
		from, to := points[i-1], points[i]
		length := math.Hypot(to.x-from.x, to.y-from.y)
		if distance <= length {
			ratio := distance / length
			return point{from.x + (to.x-from.x)*ratio, from.y + (to.y-from.y)*ratio}
		}
		distance -= length
	}
	return points[len(points)-1]
}

func slot(area agents.Area, index, capacity int) point {
	switch area {
	case agents.Outside:
		columns := 10
		rows := max(1, (capacity+columns-1)/columns)
		column := index % columns
		if index/columns%2 == 1 {
			column = columns - 1 - column
		}
		return point{848 + float64(column)*18, 721 + float64(index/columns)*min(19, 95/float64(rows))}
	case agents.Inside:
		return alongLine(indoorLine, float64(index)*385/float64(max(1, capacity-1)))
	case agents.SelfService:
		return point{550 + float64(index)*240/float64(max(1, capacity-1)), 350}
	case agents.MainService:
		return point{290 + float64(index)*150/float64(max(1, capacity-1)), 350}
	case agents.Payment:
		return point{140, 345 + float64(index)*45}
	}
	return point{970, 830}
}

func areaRank(area agents.Area) int {
	for i, candidate := range agents.ServiceRoute() {
		if candidate == area {
			return i
		}
	}
	return -1
}

// 仅做视觉插值，不额外改变后端服务耗时。路径拐点绕过柜台和饮水区。
func transitionPath(from, to agents.Area, target point) []point {
	var path []point
	if from == agents.NoArea || from == agents.Outside {
		if to == agents.Outside {
			return []point{target}
		}
		path = append(path, entryPath[0])
		if to == agents.Inside {
			// 从入口沿室内队列反向走向对应位置。
			for i := len(indoorLine) - 1; i >= 0; i-- {
				p := indoorLine[i]
				if p.x == target.x && target.y >= p.y {
					break
				}
				path = append(path, p)
			}
			return append(path, target)
		}
		path = append(path, entryPath[1:]...)
	} else if from == agents.Inside && to != agents.Inside {
		// 剩余的室内路径会保留在圆点的待走路径中。
		path = append(path, point{820, 400}, point{820, 350})
	}
	if areaRank(to) >= areaRank(agents.MainService) && areaRank(from) < areaRank(agents.MainService) {
		path = append(path, point{510, 350})
	}
	if to == agents.Payment {
		path = append(path, point{230, 350}, point{230, target.y})
	}
	return append(path, target)
}

func exitPath(id int, current point) []point {
	return []point{{50, current.y}, {50, 610}, {660, 610}, {660, 680 + float64(id%5)*21}}
}
