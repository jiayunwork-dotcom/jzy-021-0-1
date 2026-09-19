package main

import "strings"

// validateNetwork 入参检查：标识、历时、三点、紧前关系。
// 有向环的判定不在这里，由 topo.go 的拓扑排序完成。
func validateNetwork(in *NetworkInput) error {
	if in == nil || len(in.Activities) == 0 {
		return newError(ErrTypeMissingField, "activities 不能为空")
	}
	seen := make(map[string]struct{}, len(in.Activities))
	for i := range in.Activities {
		a := &in.Activities[i]
		if a.ID == "" {
			return newError(ErrTypeMissingField, "第 %d 道工序缺少标识", i)
		}
		if strings.ContainsRune(a.ID, 0) {
			return newError(ErrTypeBadRequest, "工序标识 %q 含有非法字符", a.ID)
		}
		if _, dup := seen[a.ID]; dup {
			return newError(ErrTypeDuplicateID, "工序标识 %q 重复", a.ID)
		}
		seen[a.ID] = struct{}{}

		switch {
		case a.Duration == nil && a.ThreePoint == nil:
			return newError(ErrTypeMissingField, "工序 %q 必须给 duration 或 three_point 之一", a.ID)
		case a.Duration != nil && a.ThreePoint != nil:
			return newError(ErrTypeBadRequest, "工序 %q 的 duration 与 three_point 只能给一个", a.ID)
		}
		// 历时必须为正：零历时的「里程碑」不算合法工序。
		if a.Duration != nil && *a.Duration <= 0 {
			return newError(ErrTypeInvalidDuration, "工序 %q 的历时必须为正数，得到 %v", a.ID, *a.Duration)
		}
		if t := a.ThreePoint; t != nil {
			// 三点必须满足 乐观 <= 最可能 <= 悲观。
			if !(t.Optimistic <= t.MostLikely && t.MostLikely <= t.Pessimistic) {
				return newError(ErrTypeInvalidThreePoint,
					"工序 %q 的三点历时必须满足 乐观<=最可能<=悲观，得到 (%v, %v, %v)",
					a.ID, t.Optimistic, t.MostLikely, t.Pessimistic)
			}
			if t.Expected() <= 0 {
				return newError(ErrTypeInvalidDuration, "工序 %q 的期望历时必须为正数", a.ID)
			}
		}
		for _, p := range a.Predecessors {
			if p == a.ID {
				return newError(ErrTypeSelfLoop, "工序 %q 的紧前不能是自己", a.ID)
			}
		}
	}
	for i := range in.Activities {
		a := &in.Activities[i]
		for _, p := range a.Predecessors {
			if _, ok := seen[p]; !ok {
				return newError(ErrTypeUnknownPredecessor, "工序 %q 的紧前 %q 不存在", a.ID, p)
			}
		}
	}
	if in.TargetDuration != nil && *in.TargetDuration < 0 {
		return newError(ErrTypeBadRequest, "target_duration 不能为负")
	}
	return nil
}
