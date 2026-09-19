package main

import "time"

// ThreePoint 三点历时估计：乐观、最可能、悲观。
type ThreePoint struct {
	Optimistic  float64 `json:"optimistic"`
	MostLikely  float64 `json:"most_likely"`
	Pessimistic float64 `json:"pessimistic"`
}

// Expected 三点期望历时：(乐观 + 4×最可能 + 悲观) / 6。
func (t ThreePoint) Expected() float64 {
	return (t.Optimistic + 4*t.MostLikely + t.Pessimistic) / 6
}

// Variance 历时方差：((悲观 - 乐观) / 6)²。
func (t ThreePoint) Variance() float64 {
	s := (t.Pessimistic - t.Optimistic) / 6
	return s * s
}

// ActivityInput 调用方提交的一道工序：标识、历时（确定或三点）、紧前列表。
type ActivityInput struct {
	ID           string      `json:"id"`
	Label        string      `json:"label,omitempty"`
	Duration     *float64    `json:"duration,omitempty"`
	ThreePoint   *ThreePoint `json:"three_point,omitempty"`
	Predecessors []string    `json:"predecessors"`
}

// NetworkInput 提交计算的整张工序网络。
type NetworkInput struct {
	Name           string          `json:"name,omitempty"`
	TargetDuration *float64        `json:"target_duration,omitempty"`
	Activities     []ActivityInput `json:"activities"`
}

// ActivityResult 一道工序的进度计算结果：四个时间参数与总时差、自由时差。
type ActivityResult struct {
	ID           string   `json:"id"`
	Label        string   `json:"label,omitempty"`
	Duration     float64  `json:"duration"` // 参与计算的历时（三点时取期望值）
	Variance     *float64 `json:"variance,omitempty"`
	Predecessors []string `json:"predecessors"`
	ES           float64  `json:"es"`
	EF           float64  `json:"ef"`
	LS           float64  `json:"ls"`
	LF           float64  `json:"lf"`
	TotalFloat   float64  `json:"total_float"`
	FreeFloat    float64  `json:"free_float"`
	Critical     bool     `json:"critical"`
}

// PertResult PERT 汇总，仅当网络含三点历时的时候存在。
type PertResult struct {
	ExpectedDuration      float64  `json:"expected_duration"`
	Variance              float64  `json:"variance"`
	StdDev                float64  `json:"std_dev"`
	SelectedPath          []string `json:"selected_path"`
	Formula               string   `json:"formula"`
	TargetDuration        *float64 `json:"target_duration,omitempty"`
	CompletionProbability *float64 `json:"completion_probability,omitempty"`
}

// Job 一份落库的计划作业。
type Job struct {
	ID              string           `json:"id"`
	Name            string           `json:"name,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	ProjectDuration float64          `json:"project_duration"`
	Activities      []ActivityResult `json:"activities"`
	CriticalPaths   [][]string       `json:"critical_paths"`
	Pert            *PertResult      `json:"pert,omitempty"`
}
