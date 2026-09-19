package main

import "math"

// forwardPass 正推：最早开始取全部紧前最早完成的最大值（并行两支在汇合点
// 取较大者），最早完成 = 最早开始 + 历时。返回各节点 ES/EF 与项目工期
// （虚拟汇点的最早完成）。
func forwardPass(g *graph, order []string, dur map[string]float64) (es, ef map[string]float64, project float64) {
	es = make(map[string]float64, len(order))
	ef = make(map[string]float64, len(order))
	for _, n := range order {
		start := 0.0
		for _, p := range g.preds[n] {
			if ef[p] > start {
				start = ef[p]
			}
		}
		es[n] = start
		ef[n] = start + dur[n]
	}
	project = ef[sinkID]
	return es, ef, project
}

// backwardPass 反推：最晚完成取全部紧后最晚开始的最小值，
// 最晚开始 = 最晚完成 - 历时。汇点的最晚完成固定为项目工期。
func backwardPass(g *graph, order []string, dur map[string]float64, project float64) (ls, lf map[string]float64) {
	ls = make(map[string]float64, len(order))
	lf = make(map[string]float64, len(order))
	for i := len(order) - 1; i >= 0; i-- {
		n := order[i]
		if n == sinkID {
			lf[n] = project
		} else {
			min := math.Inf(1)
			for _, s := range g.succs[n] {
				if ls[s] < min {
					min = ls[s]
				}
			}
			lf[n] = min
		}
		ls[n] = lf[n] - dur[n]
	}
	return ls, lf
}

// minSuccES 紧后工序最早开始的最小值，用于自由时差。
// 无紧后的工序其紧后是虚拟汇点，ES 即项目工期。
func minSuccES(g *graph, es map[string]float64, id string) float64 {
	min := math.Inf(1)
	for _, s := range g.succs[id] {
		if es[s] < min {
			min = es[s]
		}
	}
	return min
}
