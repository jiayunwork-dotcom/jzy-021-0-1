package main

// demoNetwork 内置的土建示范网络：含两对并行工序（B‖C 在 D 汇合、
// F‖G 在 H 汇合），关键线路 A→B→D→E→G→H，工期 24 天，可手算核对。
func demoNetwork() *NetworkInput {
	d := func(v float64) *float64 { return &v }
	return &NetworkInput{
		Name: "小型土建示范工程",
		Activities: []ActivityInput{
			{ID: "A", Label: "场地平整", Duration: d(3)},
			{ID: "B", Label: "基础开挖", Duration: d(4), Predecessors: []string{"A"}},
			{ID: "C", Label: "材料进场", Duration: d(2), Predecessors: []string{"A"}},
			{ID: "D", Label: "基础浇筑", Duration: d(5), Predecessors: []string{"B", "C"}},
			{ID: "E", Label: "主体结构", Duration: d(6), Predecessors: []string{"D"}},
			{ID: "F", Label: "屋面工程", Duration: d(3), Predecessors: []string{"E"}},
			{ID: "G", Label: "装饰装修", Duration: d(4), Predecessors: []string{"E"}},
			{ID: "H", Label: "竣工验收", Duration: d(2), Predecessors: []string{"F", "G"}},
		},
	}
}
