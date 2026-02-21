package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// cmdMCP はMCPサーバーを起動する
func cmdMCP(args []string) error {
	log.SetOutput(os.Stderr)

	s := server.NewMCPServer(
		"task_go",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// ツール登録
	s.AddTool(mcpListTasksTool(), mcpListTasksHandler)
	s.AddTool(mcpGetTaskTool(), mcpGetTaskHandler)
	s.AddTool(mcpCreateTaskTool(), mcpCreateTaskHandler)
	s.AddTool(mcpUpdateTaskTool(), mcpUpdateTaskHandler)
	s.AddTool(mcpStartTaskTool(), mcpStartTaskHandler)
	s.AddTool(mcpCompleteTaskTool(), mcpCompleteTaskHandler)
	s.AddTool(mcpCurrentTaskTool(), mcpCurrentTaskHandler)

	return server.ServeStdio(s)
}

// --- list_tasks ---

func mcpListTasksTool() mcp.Tool {
	return mcp.NewTool("list_tasks",
		mcp.WithDescription("タスク一覧を取得する。ステータスでフィルタ可能。"),
		mcp.WithString("status",
			mcp.Description("フィルタするステータス (working, waiting, not_started, completed)。省略時は全件。"),
		),
	)
}

func mcpListTasksHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store, err := LoadStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("ストアの読み込みに失敗: %v", err)), nil
	}

	tasks := store.Tasks
	status := request.GetString("status", "")
	if status != "" {
		tasks = filterByStatus(tasks, Status(status))
	}
	SortTasks(tasks)

	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("JSONエンコードに失敗: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// --- get_task ---

func mcpGetTaskTool() mcp.Tool {
	return mcp.NewTool("get_task",
		mcp.WithDescription("指定IDのタスク詳細を取得する。"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("タスクID"),
		),
	)
}

func mcpGetTaskHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := request.GetInt("id", 0)
	if id == 0 {
		return mcp.NewToolResultError("タスクIDを指定してください"), nil
	}

	store, err := LoadStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("ストアの読み込みに失敗: %v", err)), nil
	}

	for _, t := range store.Tasks {
		if t.ID == id {
			data, err := json.MarshalIndent(t, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("JSONエンコードに失敗: %v", err)), nil
			}
			return mcp.NewToolResultText(string(data)), nil
		}
	}

	return mcp.NewToolResultError(fmt.Sprintf("タスク #%d が見つかりません", id)), nil
}

// --- create_task ---

func mcpCreateTaskTool() mcp.Tool {
	return mcp.NewTool("create_task",
		mcp.WithDescription("新しいタスクを作成する。"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("タスク名"),
		),
		mcp.WithString("description",
			mcp.Description("タスクの説明"),
		),
		mcp.WithString("priority",
			mcp.Description("優先度 (high, medium, low)。デフォルト: medium"),
		),
	)
}

func mcpCreateTaskHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := request.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("タスク名を指定してください"), nil
	}

	description := request.GetString("description", "")
	priorityStr := request.GetString("priority", "medium")
	priority, err := parsePriority(priorityStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("無効な優先度: %s", priorityStr)), nil
	}

	now := time.Now()
	var newTask Task

	err = SaveWithLock(func(store *TaskStore) {
		newTask = Task{
			ID:          store.NextID,
			Name:        name,
			Description: description,
			Status:      StatusNotStarted,
			Priority:    priority,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		store.Tasks = append(store.Tasks, newTask)
		store.NextID++
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("タスクの作成に失敗: %v", err)), nil
	}

	data, _ := json.MarshalIndent(newTask, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// --- update_task ---

func mcpUpdateTaskTool() mcp.Tool {
	return mcp.NewTool("update_task",
		mcp.WithDescription("既存タスクを部分更新する。"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("タスクID"),
		),
		mcp.WithString("status",
			mcp.Description("ステータス (working, waiting, not_started, completed)"),
		),
		mcp.WithString("priority",
			mcp.Description("優先度 (high, medium, low)"),
		),
		mcp.WithString("name",
			mcp.Description("タスク名"),
		),
		mcp.WithString("description",
			mcp.Description("タスクの説明"),
		),
	)
}

func mcpUpdateTaskHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := request.GetInt("id", 0)
	if id == 0 {
		return mcp.NewToolResultError("タスクIDを指定してください"), nil
	}

	statusStr := request.GetString("status", "")
	priorityStr := request.GetString("priority", "")
	name := request.GetString("name", "")
	description := request.GetString("description", "")

	found := false
	var updatedTask Task

	err := SaveWithLock(func(store *TaskStore) {
		for i := range store.Tasks {
			if store.Tasks[i].ID == id {
				found = true
				if statusStr != "" {
					store.Tasks[i].Status = Status(statusStr)
				}
				if priorityStr != "" {
					p, err := parsePriority(priorityStr)
					if err == nil {
						store.Tasks[i].Priority = p
					}
				}
				if name != "" {
					store.Tasks[i].Name = name
				}
				if description != "" {
					store.Tasks[i].Description = description
				}
				store.Tasks[i].UpdatedAt = time.Now()
				updatedTask = store.Tasks[i]
				return
			}
		}
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("タスクの更新に失敗: %v", err)), nil
	}
	if !found {
		return mcp.NewToolResultError(fmt.Sprintf("タスク #%d が見つかりません", id)), nil
	}

	data, _ := json.MarshalIndent(updatedTask, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// --- start_task ---

func mcpStartTaskTool() mcp.Tool {
	return mcp.NewTool("start_task",
		mcp.WithDescription("タスクを作業中にし、セッションに紐付ける。TUIに通知を送信する。"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("タスクID"),
		),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("Claude Codeのセッション ID"),
		),
	)
}

func mcpStartTaskHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := request.GetInt("id", 0)
	if id == 0 {
		return mcp.NewToolResultError("タスクIDを指定してください"), nil
	}
	sessionID := request.GetString("session_id", "")
	if sessionID == "" {
		return mcp.NewToolResultError("セッションIDを指定してください"), nil
	}

	found := false
	var updatedTask Task

	err := SaveWithLock(func(store *TaskStore) {
		for i := range store.Tasks {
			if store.Tasks[i].ID == id {
				found = true
				store.Tasks[i].Status = StatusWorking
				store.Tasks[i].UpdatedAt = time.Now()
				updatedTask = store.Tasks[i]
				return
			}
		}
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("タスクの更新に失敗: %v", err)), nil
	}
	if !found {
		return mcp.NewToolResultError(fmt.Sprintf("タスク #%d が見つかりません", id)), nil
	}

	// セッション紐付け
	if err := BindSession(sessionID, id, StatusWorking); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("セッション紐付けに失敗: %v", err)), nil
	}

	// 通知送信
	AddNotification(id, NotifyTaskStarted, fmt.Sprintf("タスク #%d「%s」の作業を開始しました", id, updatedTask.Name))

	data, _ := json.MarshalIndent(updatedTask, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// --- complete_task ---

func mcpCompleteTaskTool() mcp.Tool {
	return mcp.NewTool("complete_task",
		mcp.WithDescription("タスクを完了にし、セッション紐付けを解除する。TUIに通知を送信する。"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("タスクID"),
		),
	)
}

func mcpCompleteTaskHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := request.GetInt("id", 0)
	if id == 0 {
		return mcp.NewToolResultError("タスクIDを指定してください"), nil
	}

	found := false
	var updatedTask Task

	err := SaveWithLock(func(store *TaskStore) {
		for i := range store.Tasks {
			if store.Tasks[i].ID == id {
				found = true
				store.Tasks[i].Status = StatusCompleted
				store.Tasks[i].UpdatedAt = time.Now()
				updatedTask = store.Tasks[i]
				return
			}
		}
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("タスクの更新に失敗: %v", err)), nil
	}
	if !found {
		return mcp.NewToolResultError(fmt.Sprintf("タスク #%d が見つかりません", id)), nil
	}

	// セッション紐付け解除（タスクIDから逆引き）
	if sessionID, ok := GetSessionByTask(id); ok {
		UnbindSession(sessionID)
	}

	// 通知送信
	AddNotification(id, NotifyTaskCompleted, fmt.Sprintf("タスク #%d「%s」が完了しました", id, updatedTask.Name))

	data, _ := json.MarshalIndent(updatedTask, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// --- current_task ---

func mcpCurrentTaskTool() mcp.Tool {
	return mcp.NewTool("current_task",
		mcp.WithDescription("セッションに紐付いた現在のタスクを取得する。"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("Claude CodeのセッションID"),
		),
	)
}

func mcpCurrentTaskHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := request.GetString("session_id", "")
	if sessionID == "" {
		return mcp.NewToolResultError("セッションIDを指定してください"), nil
	}

	taskID, ok := GetTaskBySession(sessionID)
	if !ok {
		return mcp.NewToolResultText(`{"message": "セッションに紐付いたタスクはありません"}`), nil
	}

	store, err := LoadStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("ストアの読み込みに失敗: %v", err)), nil
	}

	for _, t := range store.Tasks {
		if t.ID == taskID {
			data, err := json.MarshalIndent(t, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("JSONエンコードに失敗: %v", err)), nil
			}
			return mcp.NewToolResultText(string(data)), nil
		}
	}

	return mcp.NewToolResultError(fmt.Sprintf("タスク #%d が見つかりません", taskID)), nil
}
