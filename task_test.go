package main

import (
	"testing"
	"time"
)

func TestStatusLabel(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusWorking, "作業中"},
		{StatusWaiting, "確認待"},
		{StatusNotStarted, "未着手"},
		{StatusCompleted, "完了"},
		{Status("unknown"), "unknown"},
	}
	for _, tt := range tests {
		got := tt.status.Label()
		if got != tt.want {
			t.Errorf("Status(%q).Label() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestPriorityLabel(t *testing.T) {
	tests := []struct {
		priority Priority
		want     string
	}{
		{PriorityHigh, "高"},
		{PriorityMedium, "中"},
		{PriorityLow, "低"},
		{Priority(99), "?"},
	}
	for _, tt := range tests {
		got := tt.priority.Label()
		if got != tt.want {
			t.Errorf("Priority(%d).Label() = %q, want %q", tt.priority, got, tt.want)
		}
	}
}

func TestSortTasks(t *testing.T) {
	now := time.Now()
	tasks := []Task{
		{ID: 1, Priority: PriorityLow, CreatedAt: now},
		{ID: 2, Priority: PriorityHigh, CreatedAt: now},
		{ID: 3, Priority: PriorityMedium, CreatedAt: now},
		{ID: 4, Priority: PriorityHigh, CreatedAt: now},
	}

	SortTasks(tasks)

	// 優先度順: High(2,4) → Medium(3) → Low(1)
	expected := []int{2, 4, 3, 1}
	for i, id := range expected {
		if tasks[i].ID != id {
			t.Errorf("SortTasks: index %d = ID %d, want ID %d", i, tasks[i].ID, id)
		}
	}
}

func TestFilterVisible(t *testing.T) {
	tasks := []Task{
		{ID: 1, Status: StatusWorking},
		{ID: 2, Status: StatusCompleted},
		{ID: 3, Status: StatusWaiting},
		{ID: 4, Status: StatusNotStarted},
		{ID: 5, Status: StatusCompleted},
	}

	result := FilterVisible(tasks)

	if len(result) != 3 {
		t.Fatalf("FilterVisible: got %d tasks, want 3", len(result))
	}
	expectedIDs := []int{1, 3, 4}
	for i, id := range expectedIDs {
		if result[i].ID != id {
			t.Errorf("FilterVisible: index %d = ID %d, want ID %d", i, result[i].ID, id)
		}
	}
}

func TestFilterWorking(t *testing.T) {
	tasks := []Task{
		{ID: 1, Status: StatusWorking},
		{ID: 2, Status: StatusCompleted},
		{ID: 3, Status: StatusWorking},
		{ID: 4, Status: StatusNotStarted},
	}

	result := FilterWorking(tasks)

	if len(result) != 2 {
		t.Fatalf("FilterWorking: got %d tasks, want 2", len(result))
	}
	if result[0].ID != 1 || result[1].ID != 3 {
		t.Errorf("FilterWorking: got IDs %d,%d, want 1,3", result[0].ID, result[1].ID)
	}
}

func TestFilterVisibleEmpty(t *testing.T) {
	result := FilterVisible(nil)
	if result != nil {
		t.Errorf("FilterVisible(nil) = %v, want nil", result)
	}
}

func TestFilterWorkingEmpty(t *testing.T) {
	result := FilterWorking(nil)
	if result != nil {
		t.Errorf("FilterWorking(nil) = %v, want nil", result)
	}
}

func TestParseEditorFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantName    string
		wantDesc    string
	}{
		{
			name:     "正常なファイル",
			content:  "Name: テスト\nDescription: 説明文\n---\nこの行は無視\n",
			wantName: "テスト",
			wantDesc: "説明文",
		},
		{
			name:     "空のName",
			content:  "Name: \nDescription: 説明\n---\n",
			wantName: "",
			wantDesc: "説明",
		},
		{
			name:     "区切り線なし",
			content:  "Name: タスク名\nDescription: 詳細\n",
			wantName: "タスク名",
			wantDesc: "詳細",
		},
		{
			name:     "空ファイル",
			content:  "",
			wantName: "",
			wantDesc: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, desc := parseEditorFile(tt.content)
			if name != tt.wantName {
				t.Errorf("parseEditorFile name = %q, want %q", name, tt.wantName)
			}
			if desc != tt.wantDesc {
				t.Errorf("parseEditorFile description = %q, want %q", desc, tt.wantDesc)
			}
		})
	}
}
