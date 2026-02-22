package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// captureStdout はstdoutの出力をキャプチャする
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestCmdListEmpty(t *testing.T) {
	setupTestDir(t)

	output := captureStdout(t, func() {
		err := cmdList([]string{})
		if err != nil {
			t.Fatalf("cmdList error: %v", err)
		}
	})

	if !strings.Contains(output, "タスクはありません") {
		t.Errorf("Expected empty list message, got: %q", output)
	}
}

func TestCmdAddAndList(t *testing.T) {
	setupTestDir(t)

	// タスク追加
	output := captureStdout(t, func() {
		err := cmdAdd([]string{"テストタスク", "--priority", "high"})
		if err != nil {
			t.Fatalf("cmdAdd error: %v", err)
		}
	})
	if !strings.Contains(output, "#1") {
		t.Errorf("Expected task ID in output, got: %q", output)
	}

	// 一覧確認
	output = captureStdout(t, func() {
		err := cmdList([]string{})
		if err != nil {
			t.Fatalf("cmdList error: %v", err)
		}
	})
	if !strings.Contains(output, "テストタスク") {
		t.Errorf("Expected task name in list, got: %q", output)
	}
}

func TestCmdListJSON(t *testing.T) {
	setupTestDir(t)

	now := time.Now()
	SaveWithLock(func(store *TaskStore) {
		store.Tasks = append(store.Tasks, Task{
			ID:        store.NextID,
			Name:      "JSONタスク",
			Status:    StatusWorking,
			Priority:  PriorityHigh,
			CreatedAt: now,
			UpdatedAt: now,
		})
		store.NextID++
	})

	output := captureStdout(t, func() {
		err := cmdList([]string{"--format", "json"})
		if err != nil {
			t.Fatalf("cmdList error: %v", err)
		}
	})

	var tasks []Task
	if err := json.Unmarshal([]byte(output), &tasks); err != nil {
		t.Fatalf("JSON parse error: %v, output: %q", err, output)
	}
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Name != "JSONタスク" {
		t.Errorf("Task name = %q, want %q", tasks[0].Name, "JSONタスク")
	}
}

func TestCmdListFilterByStatus(t *testing.T) {
	setupTestDir(t)

	now := time.Now()
	SaveWithLock(func(store *TaskStore) {
		store.Tasks = append(store.Tasks,
			Task{ID: 1, Name: "作業中タスク", Status: StatusWorking, CreatedAt: now, UpdatedAt: now},
			Task{ID: 2, Name: "未着手タスク", Status: StatusNotStarted, CreatedAt: now, UpdatedAt: now},
		)
		store.NextID = 3
	})

	output := captureStdout(t, func() {
		err := cmdList([]string{"--status", "working"})
		if err != nil {
			t.Fatalf("cmdList error: %v", err)
		}
	})
	if !strings.Contains(output, "作業中タスク") {
		t.Errorf("Expected working task in output, got: %q", output)
	}
	if strings.Contains(output, "未着手タスク") {
		t.Errorf("Unexpected not_started task in filtered output")
	}
}

func TestCmdUpdate(t *testing.T) {
	setupTestDir(t)

	now := time.Now()
	SaveWithLock(func(store *TaskStore) {
		store.Tasks = append(store.Tasks, Task{
			ID:        1,
			Name:      "元のタスク",
			Status:    StatusNotStarted,
			Priority:  PriorityMedium,
			CreatedAt: now,
			UpdatedAt: now,
		})
		store.NextID = 2
	})

	captureStdout(t, func() {
		err := cmdUpdate([]string{"1", "--status", "working", "--name", "更新後"})
		if err != nil {
			t.Fatalf("cmdUpdate error: %v", err)
		}
	})

	store, _ := LoadStore()
	if store.Tasks[0].Status != StatusWorking {
		t.Errorf("Status = %q, want %q", store.Tasks[0].Status, StatusWorking)
	}
	if store.Tasks[0].Name != "更新後" {
		t.Errorf("Name = %q, want %q", store.Tasks[0].Name, "更新後")
	}
}

func TestCmdUpdateNotFound(t *testing.T) {
	setupTestDir(t)

	err := cmdUpdate([]string{"999", "--status", "working"})
	if err == nil {
		t.Fatal("Expected error for nonexistent task")
	}
	if !strings.Contains(err.Error(), "見つかりません") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestCmdGet(t *testing.T) {
	setupTestDir(t)

	now := time.Now()
	SaveWithLock(func(store *TaskStore) {
		store.Tasks = append(store.Tasks, Task{
			ID:          1,
			Name:        "詳細タスク",
			Description: "詳細な説明",
			Status:      StatusWorking,
			Priority:    PriorityHigh,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
		store.NextID = 2
	})

	output := captureStdout(t, func() {
		err := cmdGet([]string{"1"})
		if err != nil {
			t.Fatalf("cmdGet error: %v", err)
		}
	})

	if !strings.Contains(output, "詳細タスク") {
		t.Errorf("Expected task name in output, got: %q", output)
	}
	if !strings.Contains(output, "詳細な説明") {
		t.Errorf("Expected description in output, got: %q", output)
	}
}

func TestCmdGetJSON(t *testing.T) {
	setupTestDir(t)

	now := time.Now()
	SaveWithLock(func(store *TaskStore) {
		store.Tasks = append(store.Tasks, Task{
			ID:     1,
			Name:   "JSONタスク",
			Status: StatusNotStarted,
			CreatedAt: now,
			UpdatedAt: now,
		})
		store.NextID = 2
	})

	output := captureStdout(t, func() {
		err := cmdGet([]string{"1", "--format", "json"})
		if err != nil {
			t.Fatalf("cmdGet error: %v", err)
		}
	})

	var task Task
	if err := json.Unmarshal([]byte(output), &task); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}
	if task.Name != "JSONタスク" {
		t.Errorf("Task name = %q, want %q", task.Name, "JSONタスク")
	}
}

func TestCmdGetNotFound(t *testing.T) {
	setupTestDir(t)

	err := cmdGet([]string{"999"})
	if err == nil {
		t.Fatal("Expected error for nonexistent task")
	}
}

func TestCmdCurrent(t *testing.T) {
	setupTestDir(t)

	now := time.Now()
	SaveWithLock(func(store *TaskStore) {
		store.Tasks = append(store.Tasks, Task{
			ID:     1,
			Name:   "現在のタスク",
			Status: StatusWorking,
			CreatedAt: now,
			UpdatedAt: now,
		})
		store.NextID = 2
	})
	BindSession("test-session-123", 1, StatusWorking)

	output := captureStdout(t, func() {
		err := cmdCurrent([]string{"--session", "test-session-123"})
		if err != nil {
			t.Fatalf("cmdCurrent error: %v", err)
		}
	})

	if !strings.Contains(output, "現在のタスク") {
		t.Errorf("Expected current task in output, got: %q", output)
	}
}

func TestCmdCurrentNoSession(t *testing.T) {
	setupTestDir(t)
	t.Setenv("CLAUDE_SESSION_ID", "")

	err := cmdCurrent([]string{})
	if err == nil {
		t.Fatal("Expected error when no session specified")
	}
}

func TestCmdCurrentEnvVar(t *testing.T) {
	setupTestDir(t)
	t.Setenv("CLAUDE_SESSION_ID", "env-session")

	now := time.Now()
	SaveWithLock(func(store *TaskStore) {
		store.Tasks = append(store.Tasks, Task{
			ID:     1,
			Name:   "環境変数タスク",
			Status: StatusWorking,
			CreatedAt: now,
			UpdatedAt: now,
		})
		store.NextID = 2
	})
	BindSession("env-session", 1, StatusWorking)

	output := captureStdout(t, func() {
		err := cmdCurrent([]string{})
		if err != nil {
			t.Fatalf("cmdCurrent error: %v", err)
		}
	})

	if !strings.Contains(output, "環境変数タスク") {
		t.Errorf("Expected task from env session, got: %q", output)
	}
}

func TestCmdAddNoName(t *testing.T) {
	setupTestDir(t)

	err := cmdAdd([]string{})
	if err == nil {
		t.Fatal("Expected error when no name specified")
	}
}

func TestCmdUpdateNoID(t *testing.T) {
	setupTestDir(t)

	err := cmdUpdate([]string{})
	if err == nil {
		t.Fatal("Expected error when no ID specified")
	}
}

func TestCmdGetNoID(t *testing.T) {
	setupTestDir(t)

	err := cmdGet([]string{})
	if err == nil {
		t.Fatal("Expected error when no ID specified")
	}
}

func TestRunCLIHelp(t *testing.T) {
	setupTestDir(t)

	output := captureStdout(t, func() {
		err := runCLI([]string{"help"})
		if err != nil {
			t.Fatalf("runCLI help error: %v", err)
		}
	})

	if !strings.Contains(output, "task_go") {
		t.Errorf("Expected usage info, got: %q", output)
	}
}

func TestRunCLIUnknownCommand(t *testing.T) {
	setupTestDir(t)

	err := runCLI([]string{"unknown"})
	if err == nil {
		t.Fatal("Expected error for unknown command")
	}
}

func TestParsePriority(t *testing.T) {
	tests := []struct {
		input string
		want  Priority
		err   bool
	}{
		{"high", PriorityHigh, false},
		{"h", PriorityHigh, false},
		{"medium", PriorityMedium, false},
		{"med", PriorityMedium, false},
		{"m", PriorityMedium, false},
		{"low", PriorityLow, false},
		{"l", PriorityLow, false},
		{"invalid", PriorityMedium, true},
	}
	for _, tt := range tests {
		got, err := parsePriority(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("parsePriority(%q) error = %v, want error = %v", tt.input, err, tt.err)
		}
		if !tt.err && got != tt.want {
			t.Errorf("parsePriority(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
