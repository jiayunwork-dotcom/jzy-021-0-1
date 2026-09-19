package main

import (
	"errors"
	"math"
	"testing"
)

// ---------- 测试辅助 ----------

func d(v float64) *float64 { return &v }

func tp(o, m, p float64) *ThreePoint {
	return &ThreePoint{Optimistic: o, MostLikely: m, Pessimistic: p}
}

func act(id string, dur float64, preds ...string) ActivityInput {
	return ActivityInput{ID: id, Duration: d(dur), Predecessors: preds}
}

func actTP(id string, o, m, p float64, preds ...string) ActivityInput {
	return ActivityInput{ID: id, ThreePoint: tp(o, m, p), Predecessors: preds}
}

func mustCompute(t *testing.T, in *NetworkInput) *Job {
	t.Helper()
	job, err := computeJob(in)
	if err != nil {
		t.Fatalf("computeJob 失败: %v", err)
	}
	return job
}

func mustErr(t *testing.T, in *NetworkInput, typ ErrorType) {
	t.Helper()
	_, err := computeJob(in)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Type != typ {
		t.Fatalf("期望错误类型 %s，得到 %v", typ, err)
	}
}

func findAct(t *testing.T, job *Job, id string) ActivityResult {
	t.Helper()
	for _, a := range job.Activities {
		if a.ID == id {
			return a
		}
	}
	t.Fatalf("作业里找不到工序 %s", id)
	return ActivityResult{}
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// pertNet 三点历时网络：C 是与 B 并行的非关键支。
// 期望历时 A2 B4 C1 D5 E3，工期 14；方差 A1/9 B4/9 C1/36 D4/9 E1/9，
// 项目方差 = 10/9，所选关键通路 [A B D E]。
func pertNet() *NetworkInput {
	return &NetworkInput{
		Activities: []ActivityInput{
			actTP("A", 1, 2, 3),
			actTP("B", 2, 4, 6, "A"),
			actTP("C", 0.5, 1, 1.5, "A"),
			actTP("D", 3, 5, 7, "B", "C"),
			actTP("E", 2, 3, 4, "D"),
		},
	}
}

// multiCriticalNet 两条等长关键通路：A-B-D-F 方差 1/9，A-C-E-F 方差 1。
func multiCriticalNet() *NetworkInput {
	return &NetworkInput{
		Activities: []ActivityInput{
			actTP("A", 2, 2, 2),
			actTP("B", 2, 3, 4, "A"),
			actTP("C", 3, 3, 3, "A"),
			actTP("D", 4, 4, 4, "B"),
			actTP("E", 1, 4, 7, "C"),
			actTP("F", 1, 1, 1, "D", "E"),
		},
	}
}

// ---------- 示范网络手算核对 ----------

func TestDemoNetworkHandComputed(t *testing.T) {
	job := mustCompute(t, demoNetwork())
	if job.ProjectDuration != 24 {
		t.Fatalf("示范网络工期应为 24，得到 %v", job.ProjectDuration)
	}
	// es, ef, ls, lf, tf, ff
	want := map[string][6]float64{
		"A": {0, 3, 0, 3, 0, 0},
		"B": {3, 7, 3, 7, 0, 0},
		"C": {3, 5, 5, 7, 2, 2},
		"D": {7, 12, 7, 12, 0, 0},
		"E": {12, 18, 12, 18, 0, 0},
		"F": {18, 21, 19, 22, 1, 1},
		"G": {18, 22, 18, 22, 0, 0},
		"H": {22, 24, 22, 24, 0, 0},
	}
	for id, w := range want {
		a := findAct(t, job, id)
		got := [6]float64{a.ES, a.EF, a.LS, a.LF, a.TotalFloat, a.FreeFloat}
		if got != w {
			t.Fatalf("工序 %s 时间参数 = %v，期望 %v", id, got, w)
		}
		if a.Critical != (a.TotalFloat == 0) {
			t.Fatalf("工序 %s 关键标记与总时差矛盾", id)
		}
	}
	if len(job.CriticalPaths) != 1 || !equalStrings(job.CriticalPaths[0], []string{"A", "B", "D", "E", "G", "H"}) {
		t.Fatalf("示范网络关键线路应为 [A B D E G H]，得到 %v", job.CriticalPaths)
	}
	// 并行两支在汇合点取较大者：D 的 ES = max(EF[B]=7, EF[C]=5) = 7，
	// H 的 ES = max(EF[F]=21, EF[G]=22) = 22；较晚才完工的支 F 总时差为正，不关键。
	if findAct(t, job, "F").Critical || findAct(t, job, "C").Critical {
		t.Fatal("并行支中较晚完工仍有时差的工序不应关键")
	}
	if job.Pert != nil {
		t.Fatal("纯确定历时不应有 PERT 汇总，方差与完工概率须留空")
	}
}

// ---------- 时差两式相等 / 自由时差不大于总时差 ----------

func TestTotalFloatIdentityAndFreeFloat(t *testing.T) {
	nets := map[string]*NetworkInput{
		"demo":  demoNetwork(),
		"pert":  pertNet(),
		"multi": multiCriticalNet(),
	}
	for name, n := range nets {
		job := mustCompute(t, n)
		for _, a := range job.Activities {
			tfStart := a.LS - a.ES
			tfFinish := a.LF - a.EF
			if math.Abs(tfStart-tfFinish) > 1e-9 {
				t.Fatalf("%s/%s: 总时差两式不等：%v 与 %v", name, a.ID, tfStart, tfFinish)
			}
			if a.TotalFloat < 0 {
				t.Fatalf("%s/%s: 总时差为负 %v", name, a.ID, a.TotalFloat)
			}
			if a.FreeFloat > a.TotalFloat+1e-9 {
				t.Fatalf("%s/%s: 自由时差 %v 大于总时差 %v", name, a.ID, a.FreeFloat, a.TotalFloat)
			}
		}
	}
}

// 整数历时则总时差恰好是 0 或正整数。
func TestIntegerDurationsIntegerFloat(t *testing.T) {
	job := mustCompute(t, demoNetwork())
	for _, a := range job.Activities {
		if a.TotalFloat != math.Trunc(a.TotalFloat) {
			t.Fatalf("工序 %s 总时差 %v 不是整数", a.ID, a.TotalFloat)
		}
	}
}

// ---------- 关键通路贯穿源到汇 ----------

func TestCriticalPathsSpanSourceToSink(t *testing.T) {
	nets := map[string]*NetworkInput{
		"demo":  demoNetwork(),
		"pert":  pertNet(),
		"multi": multiCriticalNet(),
	}
	for name, n := range nets {
		job := mustCompute(t, n)
		if len(job.CriticalPaths) == 0 {
			t.Fatalf("%s: 至少应有一条关键通路", name)
		}
		preds := map[string][]string{}
		succs := map[string][]string{}
		durs := map[string]float64{}
		crit := map[string]bool{}
		for _, a := range job.Activities {
			preds[a.ID] = a.Predecessors
			durs[a.ID] = a.Duration
			crit[a.ID] = a.Critical
			for _, p := range a.Predecessors {
				succs[p] = append(succs[p], a.ID)
			}
		}
		for _, path := range job.CriticalPaths {
			if len(path) == 0 {
				t.Fatalf("%s: 空关键通路", name)
			}
			for _, id := range path {
				if _, ok := durs[id]; !ok {
					t.Fatalf("%s: 关键通路名单出现非提交标识 %q", name, id)
				}
				if !crit[id] {
					t.Fatalf("%s: 非关键工序 %q 上了关键通路", name, id)
				}
			}
			// 起点必须无紧前（从虚拟源点出发），终点必须无紧后（进入虚拟汇点）。
			if len(preds[path[0]]) != 0 {
				t.Fatalf("%s: 关键通路起点 %q 还有紧前 %v", name, path[0], preds[path[0]])
			}
			if len(succs[path[len(path)-1]]) != 0 {
				t.Fatalf("%s: 关键通路终点 %q 还有紧后 %v", name, path[len(path)-1], succs[path[len(path)-1]])
			}
			// 相邻工序必须有真实紧前关系，历时之和等于项目工期。
			sum := 0.0
			for i, id := range path {
				sum += durs[id]
				if i+1 < len(path) && !contains(preds[path[i+1]], id) {
					t.Fatalf("%s: 关键通路 %v 中 %q 不是 %q 的紧前", name, path, id, path[i+1])
				}
			}
			if math.Abs(sum-job.ProjectDuration) > 1e-9 {
				t.Fatalf("%s: 关键通路历时之和 %v ≠ 项目工期 %v", name, sum, job.ProjectDuration)
			}
		}
	}
}

// 存在若干条都关键时，全部列出来；PERT 取方差最大的那条并标明。
func TestMultipleCriticalPathsAllListed(t *testing.T) {
	job := mustCompute(t, multiCriticalNet())
	if job.ProjectDuration != 10 {
		t.Fatalf("工期应为 10，得到 %v", job.ProjectDuration)
	}
	if len(job.CriticalPaths) != 2 {
		t.Fatalf("应列出 2 条关键通路，得到 %v", job.CriticalPaths)
	}
	p1 := equalStrings(job.CriticalPaths[0], []string{"A", "B", "D", "F"})
	p2 := equalStrings(job.CriticalPaths[1], []string{"A", "C", "E", "F"})
	if !p1 || !p2 {
		t.Fatalf("关键通路应为 [A B D F] 与 [A C E F]，得到 %v", job.CriticalPaths)
	}
	if job.Pert == nil {
		t.Fatal("三点网络应有 PERT 汇总")
	}
	if job.Pert.Variance != 1 {
		t.Fatalf("项目方差应取较大者 1，得到 %v", job.Pert.Variance)
	}
	if !equalStrings(job.Pert.SelectedPath, []string{"A", "C", "E", "F"}) {
		t.Fatalf("所选关键通路应为 [A C E F]，得到 %v", job.Pert.SelectedPath)
	}
}

// ---------- 非法网络整张拒绝 ----------

func TestCycleRejected(t *testing.T) {
	mustErr(t, &NetworkInput{Activities: []ActivityInput{
		act("A", 1, "C"), act("B", 1, "A"), act("C", 1, "B"),
	}}, ErrTypeCycle)
}

func TestSelfLoopRejected(t *testing.T) {
	mustErr(t, &NetworkInput{Activities: []ActivityInput{act("A", 1, "A")}}, ErrTypeSelfLoop)
}

func TestUnknownPredecessorRejected(t *testing.T) {
	mustErr(t, &NetworkInput{Activities: []ActivityInput{act("A", 1, "ghost")}}, ErrTypeUnknownPredecessor)
}

func TestDuplicateIDRejected(t *testing.T) {
	mustErr(t, &NetworkInput{Activities: []ActivityInput{act("A", 1), act("A", 2)}}, ErrTypeDuplicateID)
}

// 零历时「里程碑」与负历时都不是合法工序。
func TestNonPositiveDurationRejected(t *testing.T) {
	mustErr(t, &NetworkInput{Activities: []ActivityInput{act("A", 0)}}, ErrTypeInvalidDuration)
	mustErr(t, &NetworkInput{Activities: []ActivityInput{act("A", -3)}}, ErrTypeInvalidDuration)
}

// 三点顺序颠倒被拒。
func TestThreePointOrderRejected(t *testing.T) {
	mustErr(t, &NetworkInput{Activities: []ActivityInput{actTP("A", 5, 2, 8)}}, ErrTypeInvalidThreePoint)
	mustErr(t, &NetworkInput{Activities: []ActivityInput{actTP("A", 1, 5, 3)}}, ErrTypeInvalidThreePoint)
	mustErr(t, &NetworkInput{Activities: []ActivityInput{actTP("A", 9, 2, 3)}}, ErrTypeInvalidThreePoint)
}

func TestMissingFieldRejected(t *testing.T) {
	mustErr(t, &NetworkInput{}, ErrTypeMissingField)
	mustErr(t, &NetworkInput{Activities: []ActivityInput{{ID: "A"}}}, ErrTypeMissingField)
	mustErr(t, &NetworkInput{Activities: []ActivityInput{{Duration: d(1)}}}, ErrTypeMissingField)
}

// ---------- PERT 方差敏感性 ----------

// 改一条非关键工序的三点宽度，项目方差不得变。
func TestNonCriticalWidthChangeKeepsVariance(t *testing.T) {
	base := mustCompute(t, pertNet())
	if base.Pert.Variance != round9(10.0/9.0) {
		t.Fatalf("基准项目方差应为 10/9，得到 %v", base.Pert.Variance)
	}
	mod := pertNet()
	mod.Activities[2].ThreePoint = tp(0, 1, 2) // C：宽度变宽，期望不变
	job2 := mustCompute(t, mod)
	if findAct(t, job2, "C").Critical {
		t.Fatal("C 加宽后期望不变，仍应非关键")
	}
	if job2.Pert.Variance != base.Pert.Variance {
		t.Fatalf("非关键工序宽度改变，项目方差不得变：%v → %v", base.Pert.Variance, job2.Pert.Variance)
	}
	if !equalStrings(job2.Pert.SelectedPath, base.Pert.SelectedPath) {
		t.Fatalf("所选关键通路不应变：%v → %v", base.Pert.SelectedPath, job2.Pert.SelectedPath)
	}
}

// 改关键通路上的三点宽度，项目方差必须变。
func TestCriticalWidthChangeAltersVariance(t *testing.T) {
	base := mustCompute(t, pertNet())
	mod := pertNet()
	mod.Activities[1].ThreePoint = tp(2, 4, 12) // B：悲观拉长，在关键通路上
	job2 := mustCompute(t, mod)
	if !findAct(t, job2, "B").Critical {
		t.Fatal("B 加宽后仍应关键")
	}
	if job2.Pert.Variance == base.Pert.Variance {
		t.Fatalf("关键工序宽度改变，项目方差必须变，仍得到 %v", job2.Pert.Variance)
	}
}

// ---------- 完工概率 ----------

func TestCompletionProbability(t *testing.T) {
	in := pertNet()
	in.TargetDuration = d(12) // 早于均值 14
	job := mustCompute(t, in)
	if job.Pert.CompletionProbability == nil {
		t.Fatal("带三点与目标日期应保存完工概率")
	}
	if p := *job.Pert.CompletionProbability; p >= 0.5 {
		t.Fatalf("目标早于均值，概率须小于一半，得到 %v", p)
	}
	if job.Pert.Formula == "" {
		t.Fatal("近似公式必须写进结果")
	}
	if !equalStrings(job.Pert.SelectedPath, []string{"A", "B", "D", "E"}) {
		t.Fatalf("所选关键通路应为 [A B D E]，得到 %v", job.Pert.SelectedPath)
	}

	in2 := pertNet()
	in2.TargetDuration = d(14) // 恰为均值
	job2 := mustCompute(t, in2)
	if math.Abs(*job2.Pert.CompletionProbability-0.5) > 1e-9 {
		t.Fatalf("目标等于均值，概率应为 0.5，得到 %v", *job2.Pert.CompletionProbability)
	}

	in3 := pertNet()
	in3.TargetDuration = d(16) // 晚于均值
	job3 := mustCompute(t, in3)
	if p := *job3.Pert.CompletionProbability; p <= 0.5 {
		t.Fatalf("目标晚于均值，概率须大于一半，得到 %v", p)
	}

	// 只给确定历时、不给三点：方差与完工概率留空，不编造。
	in4 := demoNetwork()
	in4.TargetDuration = d(30)
	job4 := mustCompute(t, in4)
	if job4.Pert != nil {
		t.Fatal("纯确定历时不应编造方差与完工概率")
	}
	for _, a := range job4.Activities {
		if a.Variance != nil {
			t.Fatalf("确定历时工序 %s 不应带方差", a.ID)
		}
	}
}
