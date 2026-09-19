package main

// 虚拟源点与汇点的内部标识：只用于让网络闭合，绝不出现在对外的名单里。
// 以 NUL 开头，validate.go 已拒绝含 NUL 的调用方标识，不会撞名。
const (
	sourceID = "\x00source"
	sinkID   = "\x00sink"
)

// graph 工序网络的邻接表，含虚拟源点与汇点。
type graph struct {
	order []string            // 真实工序，按提交顺序
	preds map[string][]string // 紧前（已去重），源点是无紧前工序的紧前
	succs map[string][]string // 紧后（已去重），汇点是无紧后工序的紧后
}

// buildGraph 把提交的网络建成图：无紧前的工序从虚拟源点出发，
// 无紧后的工序进入虚拟汇点。
func buildGraph(in *NetworkInput) *graph {
	g := &graph{
		preds: map[string][]string{},
		succs: map[string][]string{},
	}
	addEdge := func(from, to string) {
		g.preds[to] = append(g.preds[to], from)
		g.succs[from] = append(g.succs[from], to)
	}
	for i := range in.Activities {
		a := &in.Activities[i]
		g.order = append(g.order, a.ID)
		seen := map[string]bool{}
		for _, p := range a.Predecessors {
			if !seen[p] {
				seen[p] = true
				addEdge(p, a.ID)
			}
		}
	}
	for _, id := range g.order {
		if len(g.preds[id]) == 0 {
			addEdge(sourceID, id)
		}
		if len(g.succs[id]) == 0 {
			addEdge(id, sinkID)
		}
	}
	return g
}

// topoSort Kahn 拓扑排序，含源点与汇点；排不完说明网络里有有向环。
// 队列按提交顺序入队，保证同一网络每次算出的顺序一致。
func topoSort(g *graph) ([]string, error) {
	nodes := make([]string, 0, len(g.order)+2)
	nodes = append(nodes, sourceID)
	nodes = append(nodes, g.order...)
	nodes = append(nodes, sinkID)

	indeg := make(map[string]int, len(nodes))
	for _, n := range nodes {
		indeg[n] = len(g.preds[n])
	}
	var queue []string
	for _, n := range nodes {
		if indeg[n] == 0 {
			queue = append(queue, n)
		}
	}
	var order []string
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		order = append(order, n)
		for _, s := range g.succs[n] {
			indeg[s]--
			if indeg[s] == 0 {
				queue = append(queue, s)
			}
		}
	}
	if len(order) != len(nodes) {
		return nil, newError(ErrTypeCycle, "工序网络存在有向环，无法拓扑排序")
	}
	return order, nil
}
