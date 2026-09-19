package main

import "math"

// criticalPaths 枚举全部从虚拟源点到虚拟汇点、只经过关键工序的通路；
// 输出时剥掉源汇点，名单里只留调用方提交的标识。
func criticalPaths(g *graph, critical map[string]bool) [][]string {
	var paths [][]string
	var path []string
	var walk func(n string)
	walk = func(n string) {
		path = append(path, n)
		if n == sinkID {
			trimmed := make([]string, 0, len(path)-2)
			trimmed = append(trimmed, path[1:len(path)-1]...)
			paths = append(paths, trimmed)
		} else {
			for _, s := range g.succs[n] {
				if critical[s] {
					walk(s)
				}
			}
		}
		path = path[:len(path)-1]
	}
	walk(sourceID)
	return paths
}

// pertSummary 沿关键通路汇总 PERT：项目工期方差是所选关键通路上各工序
// 方差之和；有几条关键通路时取方差最大的那条，并标明选了哪一条。
func pertSummary(paths [][]string, variance map[string]float64, project float64, target *float64) *PertResult {
	best := -1
	bestVar := math.Inf(-1)
	for i, p := range paths {
		v := 0.0
		for _, id := range p {
			v += variance[id]
		}
		if v > bestVar {
			bestVar = v
			best = i
		}
	}
	std := math.Sqrt(bestVar)
	r := &PertResult{
		ExpectedDuration: round9(project),
		Variance:         round9(bestVar),
		StdDev:           round9(std),
		SelectedPath:     paths[best],
		Formula:          "P(T<=d)=Φ((d-μ)/σ)：μ 为工期期望（汇点最早完成），σ² 为所选关键通路上各工序方差之和，Φ 为标准正态分布函数",
	}
	if target != nil {
		t := *target
		p := completionProbability(project, std, t)
		r.TargetDuration = &t
		r.CompletionProbability = &p
	}
	return r
}

// completionProbability 对目标日期的完工把握，正态近似 Φ((d-μ)/σ)；
// σ 为 0 时退化为阶跃：目标不早于均值则必成，否则必不成。
func completionProbability(mu, sigma, d float64) float64 {
	if sigma <= 0 {
		if d >= mu {
			return 1
		}
		return 0
	}
	return round9(normalCDF((d - mu) / sigma))
}

// normalCDF 标准正态分布函数，用误差函数实现。
func normalCDF(z float64) float64 {
	return 0.5 * (1 + math.Erf(z/math.Sqrt2))
}
