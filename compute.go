package main

import (
	"math"
	"time"
)

// eps 浮点判零容差：三点期望历时是分数，正反推会带二进制浮点毛刺。
const eps = 1e-9

// round9 把结果修整到 9 位小数，去掉浮点毛刺；整数历时不受影响。
func round9(x float64) float64 {
	return math.Round(x*1e9) / 1e9
}

// computeJob 对一张网络做全套进度计算，产出计划作业。
// 整张网络有任何非法之处都拒绝计算，返回带类型的错误。
func computeJob(in *NetworkInput) (*Job, error) {
	if err := validateNetwork(in); err != nil {
		return nil, err
	}
	g := buildGraph(in)
	order, err := topoSort(g)
	if err != nil {
		return nil, err
	}

	dur := map[string]float64{sourceID: 0, sinkID: 0}
	variance := map[string]float64{}
	hasPert := false
	for i := range in.Activities {
		a := &in.Activities[i]
		if a.ThreePoint != nil {
			dur[a.ID] = a.ThreePoint.Expected()
			variance[a.ID] = a.ThreePoint.Variance()
			hasPert = true
		} else {
			dur[a.ID] = *a.Duration
		}
	}

	es, ef, project := forwardPass(g, order, dur)
	ls, lf := backwardPass(g, order, dur, project)

	critical := map[string]bool{sourceID: true, sinkID: true}
	activities := make([]ActivityResult, 0, len(in.Activities))
	for i := range in.Activities {
		a := &in.Activities[i]
		// 总时差两式必须同时成立：LS-ES == LF-EF。
		tfStart := ls[a.ID] - es[a.ID]
		tfFinish := lf[a.ID] - ef[a.ID]
		if math.Abs(tfStart-tfFinish) > eps {
			return nil, newError(ErrTypeInternal, "工序 %q 总时差两式不等：%v 与 %v", a.ID, tfStart, tfFinish)
		}
		if tfStart < -eps {
			return nil, newError(ErrTypeInternal, "工序 %q 总时差为负：%v", a.ID, tfStart)
		}
		// 自由时差 = 紧后最早开始的最小值 - 本工序最早完成，不得大于总时差。
		ff := round9(minSuccES(g, es, a.ID) - ef[a.ID])
		if ff < 0 {
			ff = 0
		}
		isCrit := tfStart < eps
		if isCrit {
			critical[a.ID] = true
		}
		r := ActivityResult{
			ID:           a.ID,
			Label:        a.Label,
			Duration:     round9(dur[a.ID]),
			Predecessors: dedupe(a.Predecessors),
			ES:           round9(es[a.ID]),
			EF:           round9(ef[a.ID]),
			LS:           round9(ls[a.ID]),
			LF:           round9(lf[a.ID]),
			TotalFloat:   round9(tfStart),
			FreeFloat:    ff,
			Critical:     isCrit,
		}
		if a.ThreePoint != nil {
			v := round9(variance[a.ID])
			r.Variance = &v
		}
		activities = append(activities, r)
	}

	job := &Job{
		Name:            in.Name,
		CreatedAt:       time.Now().UTC(),
		ProjectDuration: round9(project),
		Activities:      activities,
		CriticalPaths:   criticalPaths(g, critical),
	}
	// 只给确定历时、不给三点时，不编造方差：PERT 汇总整体留空。
	if hasPert {
		job.Pert = pertSummary(job.CriticalPaths, variance, project, in.TargetDuration)
	}
	return job, nil
}

// dedupe 紧前列表去重并保持提交顺序，返回值保证非 nil。
func dedupe(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]bool, len(in))
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
