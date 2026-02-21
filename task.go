package main

import (
	"sort"
	"time"
)

// Status はタスクの状態を表す
type Status string

const (
	StatusWorking    Status = "working"
	StatusWaiting    Status = "waiting"
	StatusNotStarted Status = "not_started"
	StatusCompleted  Status = "completed"
)

// Label はステータスの日本語ラベルを返す
func (s Status) Label() string {
	switch s {
	case StatusWorking:
		return "作業中"
	case StatusWaiting:
		return "確認待"
	case StatusNotStarted:
		return "未着手"
	case StatusCompleted:
		return "完了"
	default:
		return string(s)
	}
}

// Priority はタスクの優先度を表す (数値が小さいほど優先度が高い)
type Priority int

const (
	PriorityHigh   Priority = 0
	PriorityMedium Priority = 1
	PriorityLow    Priority = 2
)

// Label は優先度の日本語ラベルを返す
func (p Priority) Label() string {
	switch p {
	case PriorityHigh:
		return "高"
	case PriorityMedium:
		return "中"
	case PriorityLow:
		return "低"
	default:
		return "?"
	}
}

// Task はタスクを表す
type Task struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      Status    `json:"status"`
	Priority    Priority  `json:"priority"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TaskStore はタスク一覧と自動採番用のNextIDを保持する
type TaskStore struct {
	NextID int    `json:"next_id"`
	Tasks  []Task `json:"tasks"`
}

// SortTasks はタスクを優先度(昇順)→ID(昇順)でソートする
func SortTasks(tasks []Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		if tasks[i].Priority != tasks[j].Priority {
			return tasks[i].Priority < tasks[j].Priority
		}
		return tasks[i].ID < tasks[j].ID
	})
}

// FilterVisible は完了以外のタスクを返す
func FilterVisible(tasks []Task) []Task {
	var result []Task
	for _, t := range tasks {
		if t.Status != StatusCompleted {
			result = append(result, t)
		}
	}
	return result
}

// FilterWorking は作業中のタスクを返す
func FilterWorking(tasks []Task) []Task {
	var result []Task
	for _, t := range tasks {
		if t.Status == StatusWorking {
			result = append(result, t)
		}
	}
	return result
}
